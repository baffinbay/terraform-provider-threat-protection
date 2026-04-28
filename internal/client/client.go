package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// httpStatusErrorBodyCap bounds how much of a non-OK response body we
// surface in errors. Bigger bodies are truncated so a misbehaving server
// cannot flood Terraform diagnostics or CI logs.
const httpStatusErrorBodyCap = 1024

// httpStatusError builds a redacted error for a non-OK HTTP response. It
// prefers the RFC 6749 OAuth error envelope ({"error", "error_description"})
// when present and otherwise emits a trimmed snippet, capped at
// httpStatusErrorBodyCap bytes.
func httpStatusError(resp *http.Response, action string) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, httpStatusErrorBodyCap+1))
	truncated := len(body) > httpStatusErrorBodyCap
	if truncated {
		body = body[:httpStatusErrorBodyCap]
	}
	var env struct {
		Err  string `json:"error"`
		Desc string `json:"error_description"`
	}
	if json.Unmarshal(body, &env) == nil && env.Err != "" {
		if env.Desc != "" {
			return fmt.Errorf("%s failed (status %d): %s: %s", action, resp.StatusCode, env.Err, env.Desc)
		}
		return fmt.Errorf("%s failed (status %d): %s", action, resp.StatusCode, env.Err)
	}
	snippet := strings.TrimSpace(string(body))
	if snippet == "" {
		return fmt.Errorf("%s failed (status %d)", action, resp.StatusCode)
	}
	if truncated {
		snippet += "…(truncated)"
	}
	return fmt.Errorf("%s failed (status %d): %s", action, resp.StatusCode, snippet)
}

type Client struct {
	HostURL     string
	HTTPClient  *http.Client
	TokenSource oauth2.TokenSource
	AccountID   string
}

func NewClient(host string) *Client {
	return &Client{
		HostURL: host,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.HostURL+path, body)
	if err != nil {
		return nil, err
	}

	if c.TokenSource != nil {
		tok, err := c.TokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("acquire bearer token: %w", err)
		}
		req.Header.Set("Authorization", tok.Type()+" "+tok.AccessToken)
	}

	return req, nil
}

func (c *Client) Ping(ctx context.Context) error {
	// Use TPC IP Sources as a lightweight ping endpoint
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/tpc-ip-sources", nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("ping failed: unauthorized")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed: %d", resp.StatusCode)
	}

	return nil
}

type IPSourcesResponse struct {
	Data []struct {
		ID string `json:"id"`

		Type string `json:"type"`

		Attributes struct {
			CIDR string `json:"cidr"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetIpSources(ctx context.Context) (*IPSourcesResponse, error) {

	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/tpc-ip-sources", nil)

	if err != nil {

		return nil, err

	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get IP sources")
	}

	var ipResp IPSourcesResponse

	if err := json.NewDecoder(resp.Body).Decode(&ipResp); err != nil {

		return nil, err

	}

	return &ipResp, nil

}
