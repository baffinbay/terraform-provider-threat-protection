package provider

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwrschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestHTTPProxyRateLimitingSchema(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	NewHTTPProxyResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema generation failed: %s", resp.Diagnostics)
	}
	rateLimiting := resp.Schema.Attributes["rate_limiting"].(fwrschema.SingleNestedAttribute)
	for _, name := range []string{"strategy", "overall", "per_path", "ip_rate_limit_multiplier", "exclusions"} {
		attr, ok := rateLimiting.Attributes[name]
		if !ok {
			t.Fatalf("rate_limiting is missing %q", name)
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Fatalf("rate_limiting.%s should be optional and computed", name)
		}
		if attr.GetDeprecationMessage() != "" {
			t.Fatalf("rate_limiting.%s should not be deprecated", name)
		}
	}
	for _, name := range []string{"by_source_ip", "by_source_ip_and_url"} {
		attr, ok := rateLimiting.Attributes[name]
		if !ok {
			t.Fatalf("rate_limiting is missing deprecated alias %q", name)
		}
		if attr.GetDeprecationMessage() == "" {
			t.Fatalf("rate_limiting.%s should be deprecated", name)
		}
	}

	// Session cookie rate limiting is global only; traffic rules keep the IP-based shape.
	actions := resp.Schema.Attributes["traffic_rules"].(fwrschema.ListNestedAttribute).NestedObject.Attributes["actions"].(fwrschema.SingleNestedAttribute)
	ruleRateLimit := actions.Attributes["rate_limit"].(fwrschema.SingleNestedAttribute)
	if len(ruleRateLimit.Attributes) != 2 || ruleRateLimit.Attributes["by_source_ip"] == nil || ruleRateLimit.Attributes["by_source_ip_and_url"] == nil {
		t.Fatalf("traffic rule rate_limit should only have by_source_ip and by_source_ip_and_url, got %v", ruleRateLimit.Attributes)
	}
	if ruleRateLimit.Attributes["by_source_ip"].GetDeprecationMessage() != "" {
		t.Fatal("traffic rule rate_limit.by_source_ip should not be deprecated")
	}
}

func testRateLimitRule(enforcement string, value int64, unit string, burst int64) *HTTPProxyRateLimitRuleModel {
	return &HTTPProxyRateLimitRuleModel{Enforcement: types.StringValue(enforcement), Rate: &HTTPProxyRateLimitRateModel{Value: types.Int64Value(value), Unit: types.StringValue(unit)}, Burst: types.Int64Value(burst)}
}

func assertRateLimitRule(t *testing.T, label string, got *HTTPProxyRateLimitRuleModel, want *HTTPProxyRateLimitRuleModel) {
	t.Helper()
	if got == nil || got.Rate == nil {
		t.Fatalf("%s: got nil rule", label)
	}
	if !got.Enforcement.Equal(want.Enforcement) || !got.Rate.Value.Equal(want.Rate.Value) || !got.Rate.Unit.Equal(want.Rate.Unit) || !got.Burst.Equal(want.Burst) {
		t.Fatalf("%s: got %s %s %s burst %s, want %s %s %s burst %s", label, got.Enforcement, got.Rate.Value, got.Rate.Unit, got.Burst, want.Enforcement, want.Rate.Value, want.Rate.Unit, want.Burst)
	}
}

func TestHTTPProxyRateLimitingPlanMerge(t *testing.T) {
	portalDefault := testRateLimitRule("BLOCK", 10, "r/s", 100)
	legacyDisabled := testRateLimitRule("DISABLED", 10, "r/s", 10)

	t.Run("create without block uses BY_IP portal defaults", func(t *testing.T) {
		planned := mergeHTTPProxyPlan(HTTPProxyResourceModel{}, HTTPProxyResourceModel{}, true)
		rl := planned.RateLimiting
		if rl.Strategy.ValueString() != "BY_IP" || !rl.IPRateLimitMultiplier.IsNull() {
			t.Fatalf("unexpected strategy/multiplier: %s %s", rl.Strategy, rl.IPRateLimitMultiplier)
		}
		assertRateLimitRule(t, "overall", rl.Overall, portalDefault)
		assertRateLimitRule(t, "per_path", rl.PerPath, portalDefault)
		assertRateLimitRule(t, "by_source_ip", rl.BySourceIP, portalDefault)
		assertRateLimitRule(t, "by_source_ip_and_url", rl.BySourceIPAndURL, portalDefault)
	})

	t.Run("session cookie strategy defaults multiplier and disables legacy fields", func(t *testing.T) {
		config := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{Strategy: types.StringValue("BY_SESSION_COOKIE")}}
		rl := mergeHTTPProxyPlan(config, HTTPProxyResourceModel{}, true).RateLimiting
		if rl.IPRateLimitMultiplier.ValueInt64() != 100 {
			t.Fatalf("expected default multiplier 100, got %s", rl.IPRateLimitMultiplier)
		}
		assertRateLimitRule(t, "overall", rl.Overall, portalDefault)
		assertRateLimitRule(t, "per_path", rl.PerPath, portalDefault)
		assertRateLimitRule(t, "by_source_ip", rl.BySourceIP, legacyDisabled)
		assertRateLimitRule(t, "by_source_ip_and_url", rl.BySourceIPAndURL, legacyDisabled)
		if len(rl.Exclusions) != 0 || rl.Exclusions == nil {
			t.Fatalf("expected empty exclusions, got %v", rl.Exclusions)
		}
	})

	t.Run("explicit values win", func(t *testing.T) {
		config := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{
			Strategy:              types.StringValue("BY_SESSION_COOKIE"),
			Overall:               testRateLimitRule("BLOCK", 50, "r/m", 5),
			PerPath:               &HTTPProxyRateLimitRuleModel{Enforcement: types.StringValue("DISABLED")},
			IPRateLimitMultiplier: types.Int64Value(250),
		}}
		rl := mergeHTTPProxyPlan(config, HTTPProxyResourceModel{}, true).RateLimiting
		if rl.IPRateLimitMultiplier.ValueInt64() != 250 {
			t.Fatalf("expected multiplier 250, got %s", rl.IPRateLimitMultiplier)
		}
		assertRateLimitRule(t, "overall", rl.Overall, testRateLimitRule("BLOCK", 50, "r/m", 5))
		assertRateLimitRule(t, "per_path", rl.PerPath, testRateLimitRule("DISABLED", 10, "r/s", 100))
	})

	t.Run("BY_IP mirrors overall and per_path into the deprecated fields", func(t *testing.T) {
		config := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{
			Strategy: types.StringValue("BY_IP"),
			Overall:  testRateLimitRule("BLOCK", 20, "r/s", 200),
			PerPath:  testRateLimitRule("DISABLED", 30, "r/m", 300),
		}}
		rl := mergeHTTPProxyPlan(config, HTTPProxyResourceModel{}, true).RateLimiting
		if !rl.IPRateLimitMultiplier.IsNull() {
			t.Fatalf("expected null multiplier for BY_IP, got %s", rl.IPRateLimitMultiplier)
		}
		assertRateLimitRule(t, "by_source_ip", rl.BySourceIP, testRateLimitRule("BLOCK", 20, "r/s", 200))
		assertRateLimitRule(t, "by_source_ip_and_url", rl.BySourceIPAndURL, testRateLimitRule("DISABLED", 30, "r/m", 300))
		if rl.BySourceIP == rl.Overall || rl.BySourceIP.Rate == rl.Overall.Rate {
			t.Fatal("deprecated fields must be copies, not aliases of overall")
		}
	})

	t.Run("legacy-only config is treated as BY_IP", func(t *testing.T) {
		config := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{
			BySourceIP:       testRateLimitRule("BLOCK", 20, "r/s", 200),
			BySourceIPAndURL: &HTTPProxyRateLimitRuleModel{Enforcement: types.StringValue("DISABLED")},
		}}
		prior := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{Strategy: types.StringValue("BY_SESSION_COOKIE"), IPRateLimitMultiplier: types.Int64Value(100)}}
		rl := mergeHTTPProxyPlan(config, prior, false).RateLimiting
		if rl.Strategy.ValueString() != "BY_IP" || !rl.IPRateLimitMultiplier.IsNull() {
			t.Fatalf("unexpected strategy/multiplier: %s %s", rl.Strategy, rl.IPRateLimitMultiplier)
		}
		assertRateLimitRule(t, "overall", rl.Overall, testRateLimitRule("BLOCK", 20, "r/s", 200))
		assertRateLimitRule(t, "per_path", rl.PerPath, testRateLimitRule("DISABLED", 10, "r/s", 100))
		assertRateLimitRule(t, "by_source_ip_and_url", rl.BySourceIPAndURL, testRateLimitRule("DISABLED", 10, "r/s", 100))
	})

	t.Run("update without block keeps prior session cookie settings", func(t *testing.T) {
		prior := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{
			Strategy:              types.StringValue("BY_SESSION_COOKIE"),
			Overall:               testRateLimitRule("BLOCK", 5, "r/s", 50),
			PerPath:               testRateLimitRule("BLOCK", 2, "r/s", 20),
			IPRateLimitMultiplier: types.Int64Value(42),
			BySourceIP:            testRateLimitRule("DISABLED", 10, "r/s", 10),
			BySourceIPAndURL:      testRateLimitRule("DISABLED", 10, "r/s", 10),
			Exclusions:            []types.String{types.StringValue("192.0.2.0/24")},
		}}
		rl := mergeHTTPProxyPlan(HTTPProxyResourceModel{}, prior, false).RateLimiting
		if rl.Strategy.ValueString() != "BY_SESSION_COOKIE" || rl.IPRateLimitMultiplier.ValueInt64() != 42 || len(rl.Exclusions) != 1 {
			t.Fatalf("prior settings not kept: %#v", rl)
		}
		assertRateLimitRule(t, "overall", rl.Overall, testRateLimitRule("BLOCK", 5, "r/s", 50))
		assertRateLimitRule(t, "by_source_ip", rl.BySourceIP, legacyDisabled)
	})

	t.Run("prior state from before strategy existed is read as BY_IP", func(t *testing.T) {
		prior := HTTPProxyResourceModel{RateLimiting: &HTTPProxyRateLimitingModel{
			BySourceIP:       testRateLimitRule("BLOCK", 20, "r/s", 200),
			BySourceIPAndURL: testRateLimitRule("BLOCK", 30, "r/s", 300),
			Exclusions:       []types.String{},
		}}
		rl := mergeHTTPProxyPlan(HTTPProxyResourceModel{}, prior, false).RateLimiting
		if rl.Strategy.ValueString() != "BY_IP" {
			t.Fatalf("expected BY_IP, got %s", rl.Strategy)
		}
		assertRateLimitRule(t, "overall", rl.Overall, testRateLimitRule("BLOCK", 20, "r/s", 200))
		assertRateLimitRule(t, "per_path", rl.PerPath, testRateLimitRule("BLOCK", 30, "r/s", 300))
	})
}

func TestHTTPProxyRateLimitingRequestMapping(t *testing.T) {
	model := &HTTPProxyRateLimitingModel{
		Strategy:              types.StringValue("BY_SESSION_COOKIE"),
		Overall:               testRateLimitRule("BLOCK", 5, "r/s", 50),
		PerPath:               testRateLimitRule("DISABLED", 2, "r/m", 20),
		IPRateLimitMultiplier: types.Int64Value(42),
		BySourceIP:            testRateLimitRule("DISABLED", 10, "r/s", 10),
		BySourceIPAndURL:      testRateLimitRule("DISABLED", 10, "r/s", 10),
		Exclusions:            []types.String{types.StringValue("192.0.2.0/24")},
	}
	body, err := json.Marshal(mapHTTPProxyRateLimitingToAPI(model))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"strategy":"BY_SESSION_COOKIE","overall":{"enforcement":"BLOCK","rate":{"value":5,"unit":"r/s"},"burst":50},"perPath":{"enforcement":"DISABLED","rate":{"value":2,"unit":"r/m"},"burst":20},"ipRateLimitMultiplier":42,"exclusions":["192.0.2.0/24"]}`
	if string(body) != want {
		t.Fatalf("unexpected session cookie request:\n got %s\nwant %s", body, want)
	}

	model.Strategy = types.StringValue("BY_IP")
	model.IPRateLimitMultiplier = types.Int64Null()
	body, err = json.Marshal(mapHTTPProxyRateLimitingToAPI(model))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "ipRateLimitMultiplier") || strings.Contains(string(body), "bySrcIp") || !strings.Contains(string(body), `"strategy":"BY_IP"`) {
		t.Fatalf("unexpected BY_IP request: %s", body)
	}
}

func TestHTTPProxyRateLimitingResponseMapping(t *testing.T) {
	var sessionResponse client.HTTPProxyRateLimiting
	if err := json.Unmarshal([]byte(`{
		"strategy": "BY_SESSION_COOKIE",
		"overall": {"enforcement": "BLOCK", "rate": {"value": 5, "unit": "r/s"}, "burst": 50},
		"perPath": {"enforcement": "BLOCK", "rate": {"value": 2, "unit": "r/s"}, "burst": 20},
		"ipRateLimitMultiplier": 42,
		"bySrcIp": {"enforcement": "DISABLED", "rate": {"value": 10, "unit": "r/s"}, "burst": 10},
		"bySrcIpAndUrl": {"enforcement": "DISABLED", "rate": {"value": 10, "unit": "r/s"}, "burst": 10},
		"exclusions": ["192.0.2.0/24"]
	}`), &sessionResponse); err != nil {
		t.Fatal(err)
	}
	rl := mapHTTPProxyRateLimitingFromAPI(&sessionResponse)
	if rl.Strategy.ValueString() != "BY_SESSION_COOKIE" || rl.IPRateLimitMultiplier.ValueInt64() != 42 || len(rl.Exclusions) != 1 {
		t.Fatalf("unexpected session cookie mapping: %#v", rl)
	}
	assertRateLimitRule(t, "overall", rl.Overall, testRateLimitRule("BLOCK", 5, "r/s", 50))
	assertRateLimitRule(t, "per_path", rl.PerPath, testRateLimitRule("BLOCK", 2, "r/s", 20))
	assertRateLimitRule(t, "by_source_ip", rl.BySourceIP, testRateLimitRule("DISABLED", 10, "r/s", 10))

	// A response without strategy (legacy shape only) is read as BY_IP.
	var legacyResponse client.HTTPProxyRateLimiting
	if err := json.Unmarshal([]byte(`{
		"bySrcIp": {"enforcement": "BLOCK", "rate": {"value": 20, "unit": "r/s"}, "burst": 200},
		"bySrcIpAndUrl": {"enforcement": "DISABLED", "rate": {"value": 30, "unit": "r/m"}, "burst": 300},
		"exclusions": []
	}`), &legacyResponse); err != nil {
		t.Fatal(err)
	}
	rl = mapHTTPProxyRateLimitingFromAPI(&legacyResponse)
	if rl.Strategy.ValueString() != "BY_IP" || !rl.IPRateLimitMultiplier.IsNull() {
		t.Fatalf("unexpected legacy mapping: %s %s", rl.Strategy, rl.IPRateLimitMultiplier)
	}
	assertRateLimitRule(t, "overall", rl.Overall, testRateLimitRule("BLOCK", 20, "r/s", 200))
	assertRateLimitRule(t, "per_path", rl.PerPath, testRateLimitRule("DISABLED", 30, "r/m", 300))
	assertRateLimitRule(t, "by_source_ip_and_url", rl.BySourceIPAndURL, testRateLimitRule("DISABLED", 30, "r/m", 300))
}

func httpProxyRateLimitingResource(rateLimiting string) string {
	return strings.Replace(httpProxyMinimalResource("test-http", true), `protocol_settings = { version = "HTTP1.1" }`, `protocol_settings = { version = "HTTP1.1" }
  rate_limiting = `+rateLimiting, 1)
}

func TestHTTPProxyRateLimitingValidation(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		rateLimiting string
		expectError  string
	}{
		{name: "unknown strategy", rateLimiting: `{ strategy = "IP_BASED" }`, expectError: `value must be one of`},
		{name: "multiplier with BY_IP", rateLimiting: `{ strategy = "BY_IP", ip_rate_limit_multiplier = 10 }`, expectError: `only valid when strategy is`},
		{name: "multiplier below range", rateLimiting: `{ strategy = "BY_SESSION_COOKIE", ip_rate_limit_multiplier = 0 }`, expectError: `between 1 and\s+100000`},
		{name: "multiplier above range", rateLimiting: `{ strategy = "BY_SESSION_COOKIE", ip_rate_limit_multiplier = 100001 }`, expectError: `between 1 and\s+100000`},
		{name: "legacy mixed with strategy", rateLimiting: `{ strategy = "BY_SESSION_COOKIE", by_source_ip = { enforcement = "BLOCK" } }`, expectError: `Conflicting Rate Limiting Attributes`},
		{name: "legacy mixed with overall", rateLimiting: `{ overall = { enforcement = "BLOCK" }, by_source_ip_and_url = { enforcement = "BLOCK" } }`, expectError: `Conflicting Rate Limiting Attributes`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newHTTPProxyStateServer(t)
			defer server.Close()
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      testProviderConfig(server.URL) + httpProxyRateLimitingResource(testCase.rateLimiting),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(testCase.expectError),
			}}})
		})
	}
}

func TestHTTPProxyRateLimitingLifecycle(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	const name = "baffinbay_http_proxy.test"

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{
			Config: testProviderConfig(server.URL) + httpProxyRateLimitingResource(`{
    strategy = "BY_SESSION_COOKIE"
    overall = { enforcement = "BLOCK", rate = { value = 5, unit = "r/s" }, burst = 50 }
    per_path = { enforcement = "DISABLED" }
  }`),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(name, "rate_limiting.strategy", "BY_SESSION_COOKIE"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.ip_rate_limit_multiplier", "100"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.overall.rate.value", "5"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.per_path.enforcement", "DISABLED"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.by_source_ip.enforcement", "DISABLED"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.by_source_ip.burst", "10"),
				func(_ *terraform.State) error {
					attrs := server.lastMutation(t).Data.Attributes.RateLimiting
					if attrs == nil || attrs.Strategy != "BY_SESSION_COOKIE" || attrs.IPRateLimitMultiplier == nil || *attrs.IPRateLimitMultiplier != 100 || attrs.Overall == nil || attrs.Overall.Burst != 50 || attrs.BySourceIP != nil {
						t.Fatalf("unexpected session cookie request: %s", mustJSON(t, attrs))
					}
					return nil
				},
			),
		},
		{
			// Switching back to IP-based limiting with the deprecated attributes still works.
			Config: testProviderConfig(server.URL) + httpProxyRateLimitingResource(`{
    by_source_ip = { enforcement = "BLOCK", rate = { value = 20, unit = "r/s" }, burst = 200 }
    by_source_ip_and_url = { enforcement = "BLOCK", rate = { value = 30, unit = "r/s" }, burst = 300 }
  }`),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(name, "rate_limiting.strategy", "BY_IP"),
				tfresource.TestCheckNoResourceAttr(name, "rate_limiting.ip_rate_limit_multiplier"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.overall.rate.value", "20"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.per_path.burst", "300"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.by_source_ip.burst", "200"),
				func(_ *terraform.State) error {
					attrs := server.lastMutation(t).Data.Attributes.RateLimiting
					if attrs == nil || attrs.Strategy != "BY_IP" || attrs.IPRateLimitMultiplier != nil || attrs.PerPath == nil || attrs.PerPath.Burst != 300 {
						t.Fatalf("unexpected BY_IP request: %s", mustJSON(t, attrs))
					}
					return nil
				},
			),
		},
		{
			// Removing the block keeps the current settings.
			Config: testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", true),
			Check: tfresource.ComposeTestCheckFunc(
				tfresource.TestCheckResourceAttr(name, "rate_limiting.strategy", "BY_IP"),
				tfresource.TestCheckResourceAttr(name, "rate_limiting.overall.rate.value", "20"),
			),
		},
	}})
}
