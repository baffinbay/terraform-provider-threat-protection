package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newStubOIDC(t *testing.T, callCount *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(callCount, 1)

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		var req oidcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.ClientID == "" || req.ClientSecret == "" {
			t.Errorf("empty credentials in request: %+v", req)
		}
		if req.GrantType != "client_credentials" {
			t.Errorf("expected grant_type=client_credentials, got %q", req.GrantType)
		}
		if req.Audience != defaultAudience {
			t.Errorf("expected default audience %q, got %q", defaultAudience, req.Audience)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oidcResponse{
			AccessToken: "minted-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
}

func TestBuildTokenSource_MintsOnce(t *testing.T) {
	var calls int32
	srv := newStubOIDC(t, &calls)
	defer srv.Close()

	cachePath := filepath.Join(t.TempDir(), "cache.json")
	ts, err := BuildTokenSource(context.Background(), TokenSourceConfig{
		OIDCURL:      srv.URL,
		ClientID:     "cid",
		ClientSecret: "csecret",
		CachePath:    cachePath,
	})
	if err != nil {
		t.Fatalf("BuildTokenSource: %v", err)
	}

	for i := 0; i < 3; i++ {
		tok, err := ts.Token()
		if err != nil {
			t.Fatalf("Token(): %v", err)
		}
		if tok.AccessToken != "minted-token" {
			t.Errorf("unexpected access token: %q", tok.AccessToken)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 OIDC call, got %d", got)
	}
}

func TestBuildTokenSource_SkewSubtracted(t *testing.T) {
	var calls int32
	srv := newStubOIDC(t, &calls)
	defer srv.Close()

	cachePath := filepath.Join(t.TempDir(), "cache.json")
	before := time.Now()
	ts, err := BuildTokenSource(context.Background(), TokenSourceConfig{
		OIDCURL:      srv.URL,
		ClientID:     "cid",
		ClientSecret: "csecret",
		CachePath:    cachePath,
	})
	if err != nil {
		t.Fatalf("BuildTokenSource: %v", err)
	}
	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("Token(): %v", err)
	}
	after := time.Now()

	// ExpiresIn was 3600, skewWindow is 60s, so expiry should fall into
	// [before + (3600 - 60)s, after + (3600 - 60)s].
	minExpiry := before.Add(3540 * time.Second)
	maxExpiry := after.Add(3540 * time.Second)
	if tok.Expiry.Before(minExpiry) || tok.Expiry.After(maxExpiry) {
		t.Errorf("expiry %v not within [%v, %v]", tok.Expiry, minExpiry, maxExpiry)
	}
}

func TestBuildTokenSource_UsesCacheOnSecondBuild(t *testing.T) {
	var calls int32
	srv := newStubOIDC(t, &calls)
	defer srv.Close()

	cachePath := filepath.Join(t.TempDir(), "cache.json")
	cfg := TokenSourceConfig{
		OIDCURL:      srv.URL,
		ClientID:     "cid",
		ClientSecret: "csecret",
		CachePath:    cachePath,
	}

	ts1, err := BuildTokenSource(context.Background(), cfg)
	if err != nil {
		t.Fatalf("build 1: %v", err)
	}
	if _, err := ts1.Token(); err != nil {
		t.Fatalf("token 1: %v", err)
	}

	ts2, err := BuildTokenSource(context.Background(), cfg)
	if err != nil {
		t.Fatalf("build 2: %v", err)
	}
	if _, err := ts2.Token(); err != nil {
		t.Fatalf("token 2: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 OIDC call across two BuildTokenSource invocations, got %d", got)
	}
}

func TestBuildTokenSource_MissingCreds(t *testing.T) {
	_, err := BuildTokenSource(context.Background(), TokenSourceConfig{
		OIDCURL: "https://example.com",
	})
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}
