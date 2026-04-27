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

		if r.URL.Path == "/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/certificates/pem" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "cert-id", "type": "importPemCertificate", "attributes": {"fqdn": "example.com"}}}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/certificates/cert-id" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "cert-id", "type": "importPemCertificate", "attributes": {"fqdn": "example.com"}}}`))
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "baffinbay" {
					client_id     = "test-id"
					client_secret = "test-secret"
					api_key       = "mock-token"
					api_url       = "` + server.URL + `"
					oidc_url      = "` + server.URL + `/oauth/token"
					account_id    = "test-account-id"
				}

				resource "baffinbay_certificate" "pem" {
					type        = "pem"
					certificate = "cert-content"
					key         = "key-content"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_certificate.pem", "id", "cert-id"),
					resource.TestCheckResourceAttr("baffinbay_certificate.pem", "type", "pem"),
				),
			},
		},
	})
}
