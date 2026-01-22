package provider

import (
	"context"
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
		if r.URL.Path == "/ping" {
			if r.Header.Get("Authorization") != "Bearer test" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
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
                    api_key = "test"
                    api_url = "` + server.URL + `"
                }
                
                data "baffinbay_ping" "test" {}
                `,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.baffinbay_ping.test", "ok", "true"),
				),
			},
		},
	})
}
