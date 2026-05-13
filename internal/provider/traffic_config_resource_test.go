package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTrafficConfigResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")

		if r.URL.Path == "/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "traffic-config-id", "type": "l4Proxy", "attributes": {"name": "test-l4"}}}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "traffic-config-id", "type": "l4Proxy", "attributes": {"name": "test-l4"}}}`))
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
					account_id    = "test-account-id"
				}

				resource "baffinbay_traffic_config" "test" {
					type          = "l4Proxy"
					name          = "test-l4"
					frontend = {
						connection_type = "PLAINTEXT"
						port            = 80
						ipv4            = "192.168.1.1"
						hosts = [
							{
								host = "example.com"
							}
						]
					}
					backend = {
						hosts = [
							{
								address = "example.com"
								port    = 8080
							}
						]
						delivery_method = "ROUND_ROBIN"
						server_name     = "example.com"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "id", "traffic-config-id"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "name", "test-l4"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "frontend.connection_type", "PLAINTEXT"),
				),
			},
		},
	})
}
