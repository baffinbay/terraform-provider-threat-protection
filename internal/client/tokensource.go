package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// skewWindow is subtracted from OIDC ExpiresIn when persisting a token to
// cover NTP drift and API round-trip. Not configurable.
const skewWindow = 60 * time.Second

// defaultAudience is the OAuth2 audience claim Baffin Bay's OIDC server expects.
const defaultAudience = "https://portal.baffinbay.com"

// TokenSourceConfig describes a Baffin Bay OIDC client_credentials grant.
type TokenSourceConfig struct {
	OIDCURL      string
	ClientID     string
	ClientSecret string
	Audience     string
	CachePath    string
	HTTPClient   *http.Client
}

type oidcRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	Audience     string `json:"audience"`
}

type oidcResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// BuildTokenSource returns an oauth2.TokenSource that:
//   - serves from a local on-disk cache when a valid token is present,
//   - mints via the OIDC client_credentials grant (JSON body) on cache miss,
//   - persists refreshed tokens atomically at 0600,
//   - coordinates concurrent processes via flock so matrix CI jobs share one mint.
func BuildTokenSource(ctx context.Context, cfg TokenSourceConfig) (oauth2.TokenSource, error) {
	if cfg.OIDCURL == "" {
		return nil, fmt.Errorf("oidc_url is required")
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client_id and client_secret are required")
	}
	if cfg.Audience == "" {
		cfg.Audience = defaultAudience
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.CachePath == "" {
		p, err := CachePath(cfg.OIDCURL, cfg.ClientID)
		if err != nil {
			return nil, err
		}
		cfg.CachePath = p
	}

	cached, err := loadCache(ctx, cfg.CachePath)
	if err != nil {
		return nil, fmt.Errorf("read token cache: %w", err)
	}
	// If cached token is already expired, discard so ReuseTokenSource refreshes.
	if cached != nil && !cached.Valid() {
		cached = nil
	}
	inner := &savingTokenSource{
		ctx:       ctx,
		mint:      &oidcTokenSource{ctx: ctx, cfg: cfg},
		cachePath: cfg.CachePath,
	}
	return oauth2.ReuseTokenSource(cached, inner), nil
}

// oidcTokenSource performs the actual OIDC client_credentials exchange.
// Posts JSON (Baffin Bay's endpoint does not accept form-encoded).
type oidcTokenSource struct {
	ctx context.Context
	cfg TokenSourceConfig
}

func (s *oidcTokenSource) Token() (*oauth2.Token, error) {
	body, err := json.Marshal(oidcRequest{
		ClientID:     s.cfg.ClientID,
		ClientSecret: s.cfg.ClientSecret,
		GrantType:    "client_credentials",
		Audience:     s.cfg.Audience,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(s.ctx, http.MethodPost, s.cfg.OIDCURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "oidc token request")
	}

	var out oidcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("oidc response missing access_token")
	}
	if out.TokenType == "" {
		out.TokenType = "Bearer"
	}
	var expiry time.Time
	if out.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(out.ExpiresIn)*time.Second - skewWindow)
	}
	return &oauth2.Token{
		AccessToken: out.AccessToken,
		TokenType:   out.TokenType,
		Expiry:      expiry,
	}, nil
}

// savingTokenSource wraps a minting TokenSource with double-checked locking
// against an on-disk cache, so concurrent processes mint exactly once.
type savingTokenSource struct {
	ctx       context.Context
	mint      oauth2.TokenSource
	cachePath string
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	var tok *oauth2.Token
	err := withLock(s.ctx, s.cachePath, func() error {
		if t, err := loadCache(s.ctx, s.cachePath); err == nil && t != nil && t.Valid() {
			tok = t
			return nil
		}
		t, err := s.mint.Token()
		if err != nil {
			return err
		}
		if err := saveCache(s.cachePath, t); err != nil {
			return fmt.Errorf("write token cache: %w", err)
		}
		tok = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tok, nil
}
