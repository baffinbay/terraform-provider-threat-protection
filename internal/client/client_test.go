package client

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	if c == nil {
		t.Fatal("Expected client to be non-nil")
	}
	if c.HostURL != "https://api.baffinbay.com" {
		t.Errorf("Expected HostURL to be 'https://api.baffinbay.com', got '%s'", c.HostURL)
	}
}

func TestSetAuth(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	c.SetAuth("test-token")

	if c.Token != "test-token" {
		t.Errorf("Expected Token to be 'test-token', got '%s'", c.Token)
	}
}

func TestNewRequest(t *testing.T) {
	c := NewClient("https://api.baffinbay.com")
	c.SetAuth("test-token")

	req, err := c.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth != "Bearer test-token" {
		t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", auth)
	}
}

