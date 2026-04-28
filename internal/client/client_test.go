package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type stubTokenSource struct {
	tok *oauth2.Token
	err error
}

func (s *stubTokenSource) Token() (*oauth2.Token, error) {
	return s.tok, s.err
}

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	if c == nil {
		t.Fatal("Expected client to be non-nil")
	}
	if c.HostURL != "https://api.baffinbay.com" {
		t.Errorf("Expected HostURL to be 'https://api.baffinbay.com', got '%s'", c.HostURL)
	}
}

func TestNewRequest_WithTokenSource(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	c.TokenSource = &stubTokenSource{tok: &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}}

	req, err := c.NewRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("Expected Authorization 'Bearer test-token', got '%s'", got)
	}
}

func TestNewRequest_NoTokenSource_NoAuthHeader(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	req, err := c.NewRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Expected no Authorization header, got '%s'", got)
	}
}

func responseWith(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestHttpStatusError_OAuthEnvelope(t *testing.T) {
	err := httpStatusError(responseWith(401, `{"error":"invalid_client","error_description":"bad creds"}`), "oidc token request")
	got := err.Error()
	for _, want := range []string{"oidc token request", "401", "invalid_client", "bad creds"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected error to contain %q, got %q", want, got)
		}
	}
}

func TestHttpStatusError_OAuthEnvelopeNoDescription(t *testing.T) {
	err := httpStatusError(responseWith(401, `{"error":"invalid_client"}`), "oidc token request")
	got := err.Error()
	if !strings.Contains(got, "invalid_client") {
		t.Errorf("expected error to contain invalid_client, got %q", got)
	}
	if strings.Contains(got, "error_description") {
		t.Errorf("expected error not to mention error_description key, got %q", got)
	}
}

func TestHttpStatusError_TruncatesLargeBody(t *testing.T) {
	big := strings.Repeat("A", httpStatusErrorBodyCap*4)
	err := httpStatusError(responseWith(500, big), "get IP sources")
	got := err.Error()
	if !strings.Contains(got, "(truncated)") {
		t.Errorf("expected truncation marker, got %q", got)
	}
	if len(got) > httpStatusErrorBodyCap+128 {
		t.Errorf("expected error length capped near %d, got %d", httpStatusErrorBodyCap, len(got))
	}
}

func TestHttpStatusError_EmptyBody(t *testing.T) {
	err := httpStatusError(responseWith(503, ""), "get IP sources")
	got := err.Error()
	if got != "get IP sources failed (status 503)" {
		t.Errorf("unexpected error string: %q", got)
	}
}

func TestHttpStatusError_NonJSONSnippet(t *testing.T) {
	err := httpStatusError(responseWith(502, "<html>bad gateway</html>"), "get IP sources")
	got := err.Error()
	if !strings.Contains(got, "bad gateway") {
		t.Errorf("expected snippet to include body, got %q", got)
	}
	if strings.Contains(got, "(truncated)") {
		t.Errorf("did not expect truncation for short body, got %q", got)
	}
}
