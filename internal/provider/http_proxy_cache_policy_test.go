package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwrschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	cacheDisabled = "DISABLED"
	cacheBackend  = "BACKEND_RESPONSE_HEADERS"
)

func TestHTTPProxyCachePolicySchema(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	NewHTTPProxyResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	attr, ok := resp.Schema.Attributes["cache_policy"].(fwrschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("cache_policy should be a single nested attribute, got %T", resp.Schema.Attributes["cache_policy"])
	}
	if !attr.Optional || !attr.Computed || attr.Required {
		t.Fatalf("cache_policy should be optional+computed, got optional=%v computed=%v required=%v", attr.Optional, attr.Computed, attr.Required)
	}
	strategy, ok := attr.Attributes["strategy"].(fwrschema.StringAttribute)
	if !ok || !strategy.Optional || !strategy.Computed {
		t.Fatalf("cache_policy.strategy should be an optional+computed string, got %#v", attr.Attributes["strategy"])
	}
}

func TestHTTPProxyCachePolicyRejectsUnknownStrategy(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	// ENABLED is a backend alias, not a documented value; keep the plan aligned with what the API returns.
	for _, value := range []string{"ENABLED", "RESPONSE_HEADERS", "backend_response_headers"} {
		tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
			Config:      testProviderConfig(server.URL) + httpProxyCachePolicyResource("test-http", value),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`(?s)cache_policy.strategy.*DISABLED.*BACKEND_RESPONSE_HEADERS`),
		}}})
	}
}

func TestHTTPProxyCachePolicyPlanMerge(t *testing.T) {
	backend := &HTTPProxyCachePolicyModel{Strategy: types.StringValue(cacheBackend)}
	disabled := &HTTPProxyCachePolicyModel{Strategy: types.StringValue(cacheDisabled)}
	for _, tc := range []struct {
		name     string
		config   *HTTPProxyCachePolicyModel
		prior    *HTTPProxyCachePolicyModel
		creating bool
		want     *HTTPProxyCachePolicyModel
	}{
		{name: "create without block defaults to disabled", creating: true, want: disabled},
		{name: "create with explicit value", config: backend, creating: true, want: backend},
		{name: "create with empty block defaults strategy", config: &HTTPProxyCachePolicyModel{Strategy: types.StringNull()}, creating: true, want: disabled},
		{name: "update without block keeps remote value", prior: backend, want: backend},
		{name: "update with explicit value wins", config: disabled, prior: backend, want: disabled},
		{name: "update without block or prior stays unresolved", want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			planned := mergeHTTPProxyPlan(HTTPProxyResourceModel{CachePolicy: tc.config}, HTTPProxyResourceModel{CachePolicy: tc.prior}, tc.creating)
			if tc.want == nil {
				if planned.CachePolicy != nil {
					t.Fatalf("expected unresolved cache policy, got %#v", planned.CachePolicy)
				}
				return
			}
			if planned.CachePolicy == nil || !planned.CachePolicy.Strategy.Equal(tc.want.Strategy) {
				t.Fatalf("cache policy = %#v, want %#v", planned.CachePolicy, tc.want)
			}
		})
	}
}

func TestHTTPProxyCachePolicyRequestMapping(t *testing.T) {
	for _, mapper := range []func(HTTPProxyResourceModel) client.HTTPProxyRequest{mapHTTPProxyModelToCreateRequest, mapHTTPProxyModelToUpdateRequest} {
		payload, err := json.Marshal(mapper(HTTPProxyResourceModel{Name: typesString("p"), CachePolicy: &HTTPProxyCachePolicyModel{Strategy: typesString(cacheBackend)}}))
		if err != nil {
			t.Fatal(err)
		}
		if want := `"cachePolicy":{"strategy":"BACKEND_RESPONSE_HEADERS"}`; !strings.Contains(string(payload), want) {
			t.Errorf("payload missing %s: %s", want, payload)
		}
	}
}

func TestHTTPProxyCachePolicyResponseMapping(t *testing.T) {
	response := &client.HTTPProxyResponse{}
	response.Data.ID = "traffic-config-id"
	response.Data.Attributes.CachePolicy = &client.HTTPProxyCachePolicy{Strategy: cacheBackend}
	mapped := mapHTTPProxyResponseToModel(response, HTTPProxyResourceModel{})
	if mapped.CachePolicy == nil || mapped.CachePolicy.Strategy.ValueString() != cacheBackend {
		t.Fatalf("cachePolicy not mapped: %#v", mapped.CachePolicy)
	}

	// A response without cachePolicy (older backend) leaves the prior value unchanged.
	response.Data.Attributes.CachePolicy = nil
	prior := HTTPProxyResourceModel{CachePolicy: &HTTPProxyCachePolicyModel{Strategy: typesString(cacheDisabled)}}
	mapped = mapHTTPProxyResponseToModel(response, prior)
	if mapped.CachePolicy == nil || mapped.CachePolicy.Strategy.ValueString() != cacheDisabled {
		t.Fatalf("prior cache policy not preserved: %#v", mapped.CachePolicy)
	}
}

func TestHTTPProxyCachePolicyResolvedFromRemoteWhenStateHasNone(t *testing.T) {
	var gets int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gets++
		writeJSONAPI(w, `{"data":{"id":"traffic-config-id","type":"httpProxy","attributes":{"name":"p","cachePolicy":{"strategy":"BACKEND_RESPONSE_HEADERS"}}}}`)
	}))
	defer server.Close()

	r := &HTTPProxyResource{}
	r.client = client.NewClient(server.URL)

	planned := HTTPProxyResourceModel{ID: typesString("traffic-config-id")}
	if err := r.resolveHTTPProxyCachePolicy(context.Background(), &planned); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planned.CachePolicy == nil || planned.CachePolicy.Strategy.ValueString() != cacheBackend {
		t.Fatalf("cache policy not resolved from remote: %#v", planned.CachePolicy)
	}

	// Already resolved: no extra API call.
	if err := r.resolveHTTPProxyCachePolicy(context.Background(), &planned); err != nil {
		t.Fatal(err)
	}
	if gets != 1 {
		t.Fatalf("expected exactly one GET, got %d", gets)
	}
}

func TestHTTPProxyCachePolicyResolvesToDisabledWhenRemoteOmitsIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONAPI(w, `{"data":{"id":"traffic-config-id","type":"httpProxy","attributes":{"name":"p"}}}`)
	}))
	defer server.Close()
	r := &HTTPProxyResource{}
	r.client = client.NewClient(server.URL)
	planned := HTTPProxyResourceModel{ID: typesString("traffic-config-id")}
	if err := r.resolveHTTPProxyCachePolicy(context.Background(), &planned); err != nil {
		t.Fatal(err)
	}
	if planned.CachePolicy == nil || planned.CachePolicy.Strategy.ValueString() != cacheDisabled {
		t.Fatalf("expected DISABLED fallback, got %#v", planned.CachePolicy)
	}
}

func TestHTTPProxyCachePolicyLifecycle(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	const addr = "baffinbay_http_proxy.test"

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{
			// Create without a block: DISABLED is sent explicitly and stored.
			Config: testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", true),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", cacheDisabled),
			PostApplyFunc: func() {
				assertCachePolicySent(t, server.lastMutation(t), cacheDisabled)
			},
		},
		{
			// Enable caching.
			Config: testProviderConfig(server.URL) + httpProxyCachePolicyResource("test-http", cacheBackend),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", cacheBackend),
			PostApplyFunc: func() {
				assertCachePolicySent(t, server.lastMutation(t), cacheBackend)
			},
		},
		{ResourceName: addr, ImportState: true, ImportStateVerify: true},
		{
			// Someone disables caching in the portal: an explicit policy shows drift.
			PreConfig:          func() { server.setRemoteCachePolicy(cacheDisabled) },
			Config:             testProviderConfig(server.URL) + httpProxyCachePolicyResource("test-http", cacheBackend),
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		},
		{
			// Apply restores the configured policy.
			Config: testProviderConfig(server.URL) + httpProxyCachePolicyResource("test-http", cacheBackend),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", cacheBackend),
		},
		{
			// Turn it off again.
			Config: testProviderConfig(server.URL) + httpProxyCachePolicyResource("test-http", cacheDisabled),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", cacheDisabled),
			PostApplyFunc: func() {
				assertCachePolicySent(t, server.lastMutation(t), cacheDisabled)
			},
		},
	}})
}

// Regression: PUT treats a missing cachePolicy as DISABLED, so an apply that
// doesn't manage cache_policy must not silently turn off portal-enabled caching.
func TestHTTPProxyCachePolicyPortalValueSurvivesUnmanagedApply(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	const addr = "baffinbay_http_proxy.test"

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{Config: testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", true)},
		{
			// Caching enabled in the portal; config doesn't manage it, so no diff.
			PreConfig: func() { server.setRemoteCachePolicy(cacheBackend) },
			Config:    testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", true),
			ConfigPlanChecks: tfresource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			Check: tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", cacheBackend),
		},
		{
			// An unrelated change sends the portal value back, not DISABLED.
			Config: testProviderConfig(server.URL) + httpProxyMinimalResource("test-http-renamed", true),
			Check:  tfresource.TestCheckResourceAttr(addr, "cache_policy.strategy", cacheBackend),
			PostApplyFunc: func() {
				assertCachePolicySent(t, server.lastMutation(t), cacheBackend)
			},
		},
	}})
}

func assertCachePolicySent(t *testing.T, payload client.HTTPProxyRequest, want string) {
	t.Helper()
	got := payload.Data.Attributes.CachePolicy
	if got == nil || got.Strategy != want {
		t.Fatalf("cachePolicy sent = %#v, want strategy %s", got, want)
	}
}

func httpProxyCachePolicyResource(name, strategy string) string {
	return strings.Replace(httpProxyMinimalResource(name, true),
		`protocol_settings = { version = "HTTP1.1" }`,
		`protocol_settings = { version = "HTTP1.1" }
  cache_policy = { strategy = "`+strategy+`" }`, 1)
}
