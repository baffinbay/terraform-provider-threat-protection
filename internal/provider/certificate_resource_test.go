package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCertificateResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		
		// Mock OIDC token endpoint
		if r.URL.Path == "/oauth/token" {
			w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		// Mock Certificate endpoints
		if r.Method == http.MethodPost {
			if r.URL.Path == "/api/v2/traffic-mgmt/certificates/pem" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data": {"id": "pem-cert-id", "type": "importedCertificate", "attributes": {"fqdn": "example.com"}}}`))
				return
			}
			if r.URL.Path == "/api/v2/traffic-mgmt/certificates/lets-encrypt" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data": {"id": "le-cert-id", "type": "letsEncrypt", "attributes": {"fqdn": "le.example.com"}}}`))
				return
			}
		}

		if r.Method == http.MethodGet {
			if r.URL.Path == "/api/v2/traffic-mgmt/certificates/pem-cert-id" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data": {"id": "pem-cert-id", "type": "importedCertificate", "attributes": {"fqdn": "example.com"}}}`))
				return
			}
			if r.URL.Path == "/api/v2/traffic-mgmt/certificates/le-cert-id" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data": {"id": "le-cert-id", "type": "letsEncrypt", "attributes": {"fqdn": "le.example.com"}}}`))
				return
			}
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// PEM Import
			{
				Config: `provider "baffinbay" {
					client_id     = "test-id"
					client_secret = "test-secret"
					api_url       = "` + server.URL + `"
					oidc_url      = "` + server.URL + `/oauth/token"
				}

				resource "baffinbay_certificate" "pem" {
					type         = "pem"
					certificate  = "---BEGIN CERTIFICATE---\n...\n---END CERTIFICATE---"
					intermediate = "---BEGIN CERTIFICATE---\n...\n---END CERTIFICATE---"
					key          = "---BEGIN PRIVATE KEY---\n...\n---END PRIVATE KEY---"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_certificate.pem", "id", "pem-cert-id"),
				),
			},
			// Let's Encrypt
			{
				Config: `provider "baffinbay" {
					client_id     = "test-id"
					client_secret = "test-secret"
					api_url       = "` + server.URL + `"
					oidc_url      = "` + server.URL + `/oauth/token"
				}

				resource "baffinbay_certificate" "le" {
					type = "lets_encrypt"
					fqdn = "le.example.com"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_certificate.le", "id", "le-cert-id"),
				),
			},
		},
	})
}
