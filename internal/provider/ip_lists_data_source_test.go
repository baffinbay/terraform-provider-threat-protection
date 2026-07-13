package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIPListsDataSource(t *testing.T) {
	setTestTokenCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}
		if req.Method == http.MethodGet && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists" {
			if got := req.URL.Query().Get("filter[tenant-id]"); got != "test-tenant-id" {
				t.Errorf("filter[tenant-id] = %q, want test-tenant-id", got)
			}
			writeJSONAPI(w, `{
				"data": [{
					"id": "11111111-1111-1111-1111-111111111111",
					"type": "ipList",
					"attributes": {
						"name": "employees",
						"entries": [
							{"value": "198.51.100.0/24", "note": "office"},
							{"value": "2001:db8::/32"}
						]
					},
					"relationships": {
						"usedBy": {
							"data": [{"id": "22222222-2222-2222-2222-222222222222", "type": "httpProxy"}]
						}
					},
					"meta": {
						"createdAt": "2026-07-01T10:00:00Z",
						"lastUpdatedAt": "2026-07-02T11:00:00Z"
					}
				}]
			}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := `
		provider "baffinbay" {
			client_id     = "test-id"
			client_secret = "test-secret"
			api_url       = "` + server.URL + `"
			oidc_url      = "` + server.URL + `/oauth/token"
			tenant_id     = "test-tenant-id"
		}

		data "baffinbay_ip_lists" "test" {}
	`

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: config,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.#", "1"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.id", "11111111-1111-1111-1111-111111111111"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.type", "ipList"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.name", "employees"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.entries.#", "2"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.entries.0.value", "198.51.100.0/24"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.entries.0.note", "office"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.used_by.#", "1"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.used_by.0.id", "22222222-2222-2222-2222-222222222222"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.used_by.0.type", "httpProxy"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.created_at", "2026-07-01T10:00:00Z"),
					tfresource.TestCheckResourceAttr("data.baffinbay_ip_lists.test", "ip_lists.0.last_updated_at", "2026-07-02T11:00:00Z"),
				),
			},
		},
	})
}
