package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProvider_LivePing(t *testing.T) {
	// Only run if specifically requested to prevent unwanted API calls
	if os.Getenv("BAFFINBAY_LIVE_TEST") == "" {
		t.Skip("Skipping live API ping test. Set BAFFINBAY_LIVE_TEST=1 to run.")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "baffinbay" {}
                
                data "baffinbay_ip_sources" "live_test" {}
                `,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.baffinbay_ip_sources.live_test", "id"),
				),
			},
		},
	})
}
