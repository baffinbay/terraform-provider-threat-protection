package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"baffinbay": providerserver.NewProtocol6WithError(New()),
}

func TestAccProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}
		if r.URL.Path == "/api/v2/traffic-mgmt/tpc-ip-sources" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test" && auth != "Bearer mock-token" {
				t.Logf("IP Sources Unauthorized: Got %s", auth)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write([]byte(`{"data": []}`))
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
                    api_key = "test"
                    api_url = "` + server.URL + `"
                }
                
                data "baffinbay_ip_sources" "api_key_test" {}
                `,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.baffinbay_ip_sources.api_key_test", "id"),
				),
			},
			{
				Config: `provider "baffinbay" {
                    client_id     = "test-id"
                    client_secret = "test-secret"
                    api_key       = ""
                    api_url       = "` + server.URL + `"
                    oidc_url      = "` + server.URL + `/oauth/token"
                }
                
                data "baffinbay_ip_sources" "oidc_test" {}
                `,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.baffinbay_ip_sources.oidc_test", "id"),
				),
			},
		},
	})
}
