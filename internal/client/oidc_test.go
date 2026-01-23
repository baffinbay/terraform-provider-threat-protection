package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		var req OIDCAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req.ClientID != "test-client-id" {
			t.Errorf("Expected client_id test-client-id, got %s", req.ClientID)
		}
		if req.ClientSecret != "test-client-secret" {
			t.Errorf("Expected client_secret test-client-secret, got %s", req.ClientSecret)
		}
		if req.GrantType != "client_credentials" {
			t.Errorf("Expected grant_type client_credentials, got %s", req.GrantType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OIDCAuthResponse{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
		})
	}))
	defer server.Close()

	c := NewClient("https://api.baffinbay.com")
	err := c.Authenticate(server.URL, "test-client-id", "test-client-secret")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}

	if c.Token != "test-access-token" {
		t.Errorf("Expected token test-access-token, got %s", c.Token)
	}
}
