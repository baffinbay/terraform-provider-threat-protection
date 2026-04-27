package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTrafficConfig_Live(t *testing.T) {
	// Only run if specifically requested to avoid accidental live hits
	if os.Getenv("RUN_LIVE_TESTS") == "" {
		t.Skip("Skipping live test. Set RUN_LIVE_TESTS=1 to run.")
	}

	_ = os.Getenv("BAFFINBAY_ACCOUNT_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "baffinbay_traffic_config" "live_test" {
					type          = "httpProxy"
					name          = "hermes-live-smoke-test"
					deployment_state = "UNDEPLOYED"
					
					frontend = {
						connection_type = "PLAINTEXT"
						port            = 80
						ipv4            = "203.0.113.3"
						hosts = [
							{
								host = "smoke-test.example.com"
							}
						]
					}
					
					backend = {
						hosts = [
							{
								address = "origin.example.com"
								port    = 80
							}
						]
						delivery_method = "ROUND_ROBIN"
						server_name     = "origin.example.com"
					}

					protocol_settings = {
						version = "HTTP1.1"
					}
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_traffic_config.live_test", "name", "hermes-live-smoke-test"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.live_test", "frontend.ipv4", "203.0.113.3"),
				),
			},
		},
	})
}
