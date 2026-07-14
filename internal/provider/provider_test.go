package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"baffinbay": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccProviderRequiresTenantID(t *testing.T) {
	t.Setenv("BAFFINBAY_TOKEN_CACHE", t.TempDir()+"/token-cache.json")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "baffinbay" {
                    client_id     = "test-id"
                    client_secret = "test-secret"
                    api_url       = "https://example.com"
                    oidc_url      = "https://example.com/oauth/token"
                }

                data "baffinbay_ip_sources" "oidc_test" {}
                `,
				ExpectError: regexp.MustCompile(`Missing Tenant ID`),
			},
		},
	})
}

func TestAccProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}
		if r.URL.Path == "/api/v2/traffic-mgmt/tpc-ip-sources" {
			if r.Header.Get("Authorization") != "Bearer mock-token" {
				t.Logf("IP Sources Unauthorized: Got %s", r.Header.Get("Authorization"))
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
                    tenant_id     = "test-tenant-id"
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
