package client

import (
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

	req, err := c.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("Expected Authorization 'Bearer test-token', got '%s'", got)
	}
}

func TestNewRequest_NoTokenSource_NoAuthHeader(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	req, err := c.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Expected no Authorization header, got '%s'", got)
	}
}
