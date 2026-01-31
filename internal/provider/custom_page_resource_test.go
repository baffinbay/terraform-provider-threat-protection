package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomPageResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")

		if r.URL.Path == "/oauth/token" {
			w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"id": "custom-page-id", "type": "customPage", "attributes": {"name": "test-page"}}}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/custom-page-id" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"id": "custom-page-id", "type": "customPage", "attributes": {"name": "test-page"}}}`))
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
					account_id    = "test-account-id"
				}

				resource "baffinbay_custom_page" "test" {
					name    = "test-page"
					content = "<html>test</html>"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_custom_page.test", "id", "custom-page-id"),
					resource.TestCheckResourceAttr("baffinbay_custom_page.test", "name", "test-page"),
				),
			},
		},
	})
}
