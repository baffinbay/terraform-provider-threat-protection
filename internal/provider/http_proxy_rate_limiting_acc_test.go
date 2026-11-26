package provider

import (
	"fmt"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccHTTPProxyRateLimitingLive is the live check for session cookie rate limiting
// (PDEV-4575). It needs the same environment as TestAccHTTPProxyResourceFullLive.
// Every step also checks that a plan right after apply is empty, so the planned
// deprecated fields and multiplier must match what the API returns.
func TestAccHTTPProxyRateLimitingLive(t *testing.T) {
	cfg := newHTTPProxyLiveAcceptanceConfig(t)
	t.Setenv("BAFFINBAY_TOKEN_CACHE", t.TempDir()+"/token-cache.json")
	const addr = "baffinbay_http_proxy.rate_limiting"

	tfresource.Test(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{
			// Existing configurations that use the deprecated attributes keep working as BY_IP.
			Config: httpProxyLiveRateLimitingHCL(cfg, cfg.Name, `rate_limiting = {
    by_source_ip         = { enforcement = "BLOCK", rate = { value = 20, unit = "r/s" }, burst = 200 }
    by_source_ip_and_url = { enforcement = "DISABLED", rate = { value = 30, unit = "r/m" }, burst = 300 }
  }`),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.strategy", "BY_IP"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.overall.burst", "200"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.per_path.rate.unit", "r/m"),
				tfresource.TestCheckNoResourceAttr(addr, "rate_limiting.ip_rate_limit_multiplier"),
			),
		},
		{
			// Switch to session cookie rate limiting.
			Config: httpProxyLiveRateLimitingHCL(cfg, cfg.Name, `rate_limiting = {
    strategy                 = "BY_SESSION_COOKIE"
    overall                  = { enforcement = "BLOCK", rate = { value = 5, unit = "r/s" }, burst = 50 }
    per_path                 = { enforcement = "BLOCK", rate = { value = 2, unit = "r/s" }, burst = 20 }
    ip_rate_limit_multiplier = 250
    exclusions               = ["192.0.2.0/24"]
  }`),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.strategy", "BY_SESSION_COOKIE"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.ip_rate_limit_multiplier", "250"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.overall.burst", "50"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.by_source_ip.enforcement", "DISABLED"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.exclusions.#", "1"),
			),
		},
		{
			// Updating without the block keeps the current settings.
			Config: httpProxyLiveRateLimitingHCL(cfg, cfg.Name+"-unmanaged", ""),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.strategy", "BY_SESSION_COOKIE"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.ip_rate_limit_multiplier", "250"),
			),
		},
		{
			// The default multiplier is used when it is omitted.
			Config: httpProxyLiveRateLimitingHCL(cfg, cfg.Name, `rate_limiting = { strategy = "BY_SESSION_COOKIE" }`),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.ip_rate_limit_multiplier", "100"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.overall.rate.value", "10"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.overall.burst", "100"),
			),
		},
		{
			// Back to IP-based limiting with the current attributes.
			Config: httpProxyLiveRateLimitingHCL(cfg, cfg.Name, `rate_limiting = {
    strategy = "BY_IP"
    overall  = { enforcement = "BLOCK", rate = { value = 40, unit = "r/s" }, burst = 400 }
    per_path = { enforcement = "BLOCK", rate = { value = 4, unit = "r/s" }, burst = 40 }
  }`),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.strategy", "BY_IP"),
				tfresource.TestCheckNoResourceAttr(addr, "rate_limiting.ip_rate_limit_multiplier"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.by_source_ip.burst", "400"),
				tfresource.TestCheckResourceAttr(addr, "rate_limiting.by_source_ip_and_url.burst", "40"),
			),
		},
	}})
}

func httpProxyLiveRateLimitingHCL(cfg httpProxyLiveAcceptanceConfig, name, rateLimiting string) string {
	return fmt.Sprintf(`
provider "baffinbay" {}

resource "baffinbay_http_proxy" "rate_limiting" {
  name             = %q
  deployment_state = %q
  frontend = {
    connection_type = "PLAINTEXT"
    ipv4            = %q
    port            = 80
    hosts           = [{ host = %q }]
  }
  backend = { hosts = [{ address = %q, port = 80 }] }
  protocol_settings = { version = "HTTP1.1" }
  %s
}
`, name, cfg.DeploymentState, cfg.FrontendIPv4, cfg.Host, cfg.BackendAddress, rateLimiting)
}
