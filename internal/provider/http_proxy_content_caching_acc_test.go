package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccHTTPProxyContentCachingLive is the live check for content caching. It needs the
// same environment as TestAccHTTPProxyResourceFullLive. The purge step runs only when
// BAFFINBAY_ACC_DEPLOYMENT_STATE=DEPLOYED, because the API rejects purges for undeployed proxies.
func TestAccHTTPProxyContentCachingLive(t *testing.T) {
	cfg := newHTTPProxyLiveAcceptanceConfig(t)
	t.Setenv("BAFFINBAY_TOKEN_CACHE", t.TempDir()+"/token-cache.json")
	const addr = "baffinbay_http_proxy.cache"
	var proxyID string

	steps := []tfresource.TestStep{
		{
			// Create with caching on.
			Config: httpProxyLiveCacheHCL(cfg, cfg.Name, `cache_policy = { strategy = "BACKEND_RESPONSE_HEADERS" }`, false),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", "BACKEND_RESPONSE_HEADERS"),
				captureResourceID(addr, &proxyID),
			),
		},
		{
			// Updating without the block keeps the current policy.
			Config: httpProxyLiveCacheHCL(cfg, cfg.Name+"-unmanaged", "", false),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", "BACKEND_RESPONSE_HEADERS"),
		},
		{
			// A change outside Terraform shows up as drift against an explicit policy.
			PreConfig:          func() { setLiveHTTPProxyCachePolicy(t, proxyID, "DISABLED") },
			Config:             httpProxyLiveCacheHCL(cfg, cfg.Name+"-unmanaged", `cache_policy = { strategy = "BACKEND_RESPONSE_HEADERS" }`, false),
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		},
		{
			// Applying restores it.
			Config: httpProxyLiveCacheHCL(cfg, cfg.Name+"-managed", `cache_policy = { strategy = "BACKEND_RESPONSE_HEADERS" }`, false),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", "BACKEND_RESPONSE_HEADERS"),
		},
	}
	if cfg.DeploymentState == "DEPLOYED" {
		// The update fires the purge action through action_trigger; it fails the step if the purge rollout fails.
		steps = append(steps, tfresource.TestStep{
			Config: httpProxyLiveCacheHCL(cfg, cfg.Name+"-purged", `cache_policy = { strategy = "BACKEND_RESPONSE_HEADERS" }`, true),
			Check:  tfresource.TestCheckResourceAttr(addr, "name", cfg.Name+"-purged"),
		})
	} else {
		t.Log("Skipping the purge step: set BAFFINBAY_ACC_DEPLOYMENT_STATE=DEPLOYED to run it.")
	}

	tfresource.Test(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
}

func httpProxyLiveCacheHCL(cfg httpProxyLiveAcceptanceConfig, name, cachePolicy string, purgeOnUpdate bool) string {
	action, lifecycle := "", ""
	if purgeOnUpdate {
		action = `
action "baffinbay_http_proxy_content_cache_purge" "purge" {
  config { traffic_config_id = baffinbay_http_proxy.cache.id }
}
`
		lifecycle = `
  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.baffinbay_http_proxy_content_cache_purge.purge]
    }
  }`
	}
	return fmt.Sprintf(`
provider "baffinbay" {}
%s
resource "baffinbay_http_proxy" "cache" {
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
  %s
}
`, action, name, cfg.DeploymentState, cfg.FrontendIPv4, cfg.Host, cfg.BackendAddress, cachePolicy, lifecycle)
}

func captureResourceID(addr string, id *string) tfresource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("%s not found in state", addr)
		}
		*id = rs.Primary.ID
		return nil
	}
}

// setLiveHTTPProxyCachePolicy changes only the cache policy through the API, like a portal user would.
func setLiveHTTPProxyCachePolicy(t *testing.T, id, strategy string) {
	t.Helper()
	ctx := context.Background()
	c := newLiveAcceptanceClient(t)
	current, err := c.GetHTTPProxy(ctx, id)
	if err != nil {
		t.Fatalf("read HTTP Proxy %s: %v", id, err)
	}
	model := mapHTTPProxyResponseToModel(current, HTTPProxyResourceModel{})
	model.CachePolicy = &HTTPProxyCachePolicyModel{Strategy: typesString(strategy)}
	updated, err := c.UpdateHTTPProxy(ctx, id, mapHTTPProxyModelToUpdateRequest(model))
	if err != nil {
		t.Fatalf("update cache policy of HTTP Proxy %s: %v", id, err)
	}
	if err := c.WaitForTrafficConfigRollout(ctx, id, updated.Data.ActiveRolloutID()); err != nil {
		t.Fatalf("wait for rollout: %v", err)
	}
}

func newLiveAcceptanceClient(t *testing.T) *client.Client {
	t.Helper()
	ts, err := client.BuildTokenSource(context.Background(), client.TokenSourceConfig{
		OIDCURL:      envOrDefault("BAFFINBAY_OIDC_URL", "https://m2m-auth.baffinbay.com/oauth/token"),
		ClientID:     os.Getenv("BAFFINBAY_CLIENT_ID"),
		ClientSecret: os.Getenv("BAFFINBAY_CLIENT_SECRET"),
	})
	if err != nil {
		t.Fatalf("build token source: %v", err)
	}
	c := client.NewClient(envOrDefault("BAFFINBAY_API_URL", "https://portal.baffinbay.com"))
	c.TenantID = os.Getenv("BAFFINBAY_TENANT_ID")
	c.TokenSource = ts
	return c
}
