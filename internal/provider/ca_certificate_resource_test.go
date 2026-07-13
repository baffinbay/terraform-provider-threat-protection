package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCaCertificateResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")

		if r.URL.Path == "/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "ca-cert-id", "type": "caCertificate", "attributes": {"name": "test-ca"}}}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/ca-cert-id" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "ca-cert-id", "type": "caCertificate", "attributes": {"name": "test-ca"}}}`))
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Setenv("BAFFINBAY_TOKEN_CACHE", t.TempDir()+"/token-cache.json")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "baffinbay" {
					client_id     = "test-id"
					client_secret = "test-secret"
					api_url       = "` + server.URL + `"
					oidc_url      = "` + server.URL + `/oauth/token"
					tenant_id    = "test-tenant-id"
				}

				resource "baffinbay_ca_certificate" "test" {
					name        = "test-ca"
					certificate = "ca-content"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_ca_certificate.test", "id", "ca-cert-id"),
					resource.TestCheckResourceAttr("baffinbay_ca_certificate.test", "name", "test-ca"),
				),
			},
		},
	})
}
