package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	// httpStatusErrorBodyCap bounds how much response detail we surface in
	// errors, so a misbehaving server cannot flood Terraform diagnostics or CI
	// logs.
	httpStatusErrorBodyCap = 1024

	// httpStatusErrorParseCap bounds how much body data we inspect before
	// formatting a capped error. It is intentionally larger than the displayed
	// snippet so structured OAuth errors are parsed before display truncation.
	httpStatusErrorParseCap = 64 * 1024
)

// HTTPStatusError is returned when the API responds with a non-success status.
// It preserves the status code so resource lifecycle code can handle expected
// statuses such as 404 Not Found without parsing error strings.
type HTTPStatusError struct {
	Action     string
	StatusCode int
	Snippet    string
}

func (e *HTTPStatusError) Error() string {
	if e.Snippet == "" {
		return fmt.Sprintf("%s failed (status %d)", e.Action, e.StatusCode)
	}
	return fmt.Sprintf("%s failed (status %d): %s", e.Action, e.StatusCode, e.Snippet)
}

// IsHTTPStatus reports whether err is an HTTPStatusError with the given status.
func IsHTTPStatus(err error, statusCode int) bool {
	var statusErr *HTTPStatusError
	return errors.As(err, &statusErr) && statusErr.StatusCode == statusCode
}

// IsNotFound reports whether err is an HTTP 404 response error.
func IsNotFound(err error) bool {
	return IsHTTPStatus(err, http.StatusNotFound)
}

// httpStatusError builds a redacted error for a non-OK HTTP response. It
// prefers the RFC 6749 OAuth error envelope ({"error", "error_description"})
// when present and otherwise emits a trimmed snippet, capped at
// httpStatusErrorBodyCap bytes.
func httpStatusError(resp *http.Response, action string) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, httpStatusErrorParseCap+1))
	bodyTruncated := len(body) > httpStatusErrorParseCap
	if bodyTruncated {
		body = body[:httpStatusErrorParseCap]
	}

	snippet := oauthErrorSnippet(body)
	if snippet == "" {
		snippet = strings.TrimSpace(string(body))
	}
	snippet, snippetTruncated := truncateHTTPErrorSnippet(snippet)
	if snippetTruncated || bodyTruncated {
		snippet += "...(truncated)"
	}

	return &HTTPStatusError{
		Action:     action,
		StatusCode: resp.StatusCode,
		Snippet:    snippet,
	}
}

func oauthErrorSnippet(body []byte) string {
	var env struct {
		Err  string `json:"error"`
		Desc string `json:"error_description"`
	}
	if json.Unmarshal(body, &env) == nil && env.Err != "" {
		if env.Desc != "" {
			return strings.TrimSpace(env.Err + ": " + env.Desc)
		}
		return strings.TrimSpace(env.Err)
	}
	return ""
}

func truncateHTTPErrorSnippet(snippet string) (string, bool) {
	if len(snippet) <= httpStatusErrorBodyCap {
		return snippet, false
	}
	return snippet[:httpStatusErrorBodyCap], true
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
