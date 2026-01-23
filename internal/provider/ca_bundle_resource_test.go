package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCaBundleResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		
		if r.URL.Path == "/oauth/token" {
			w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/ca-bundles" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"id": "ca-bundle-id", "type": "caBundle", "attributes": {"name": "test-bundle"}}}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-bundles/ca-bundle-id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"id": "ca-bundle-id", "type": "caBundle", "attributes": {"name": "test-bundle"}}}`))
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
					api_url       = "` + server.URL + `"
					oidc_url      = "` + server.URL + `/oauth/token"
				}

				resource "baffinbay_ca_bundle" "test" {
					name        = "test-bundle"
					certificate = "---\nBEGIN CERTIFICATE---\n...\n---END CERTIFICATE---"
				}
				`,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("baffinbay_ca_bundle.test", "id", "ca-bundle-id"),
						resource.TestCheckResourceAttr("baffinbay_ca_bundle.test", "name", "test-bundle"),
					),
				},
			},
		})
}
