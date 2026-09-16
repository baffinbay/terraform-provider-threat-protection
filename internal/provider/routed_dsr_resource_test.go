package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwrschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestProviderResourcesIncludeRoutedDsr(t *testing.T) {
	p := &BaffinBayProvider{}
	resourceTypes := map[string]bool{}

	for _, newResource := range p.Resources(context.Background()) {
		r := newResource()
		resp := &fwresource.MetadataResponse{}
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		resourceTypes[resp.TypeName] = true
	}

	if !resourceTypes["baffinbay_routed_dsr"] {
		t.Fatal("expected baffinbay_routed_dsr to be registered")
	}
}

func TestRoutedDsrResourceSchema(t *testing.T) {
	r := NewRoutedDsrResource()
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema generation failed: %s", resp.Diagnostics)
	}

	unsupportedAttributes := []string{
		"type",
		"frontend",
		"backend",
		"waf",
		"protocol_settings",
		"rate_limiting",
	}
	for _, attrName := range unsupportedAttributes {
		if _, ok := resp.Schema.Attributes[attrName]; ok {
			t.Fatalf("Routed DSR schema unexpectedly included %q", attrName)
		}
	}

	assertRoutedDsrStringAttribute(t, resp, "id", false, true)
	assertRoutedDsrStringAttribute(t, resp, "name", true, false)
	assertRoutedDsrStringAttribute(t, resp, "prefix", true, false)
	assertRoutedDsrStringAttribute(t, resp, "deployment_state", false, true)

	announcedAttr, ok := resp.Schema.Attributes["announced"].(fwrschema.BoolAttribute)
	if !ok {
		t.Fatalf("announced attribute has type %T, want schema.BoolAttribute", resp.Schema.Attributes["announced"])
	}
	if !announcedAttr.Optional || !announcedAttr.Computed {
		t.Fatalf("announced should be optional and computed for a default, got optional=%t computed=%t", announcedAttr.Optional, announcedAttr.Computed)
	}
	if announcedAttr.Default == nil {
		t.Fatal("announced should default to false")
	}
}

func TestRoutedDsrResource_LifecycleUpdateImportAndDelete(t *testing.T) {
	setTestTokenCache(t)

	var currentName atomic.Value
	currentName.Store("test-dsr")
	var currentPrefix atomic.Value
	currentPrefix.Store("192.0.2.0/24")
	var currentDeploymentState atomic.Value
	currentDeploymentState.Store("UNDEPLOYED")
	var currentAnnounced atomic.Bool
	currentAnnounced.Store(false)
	var putCount atomic.Int64
	var deleteCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			payload := readRoutedDsrRequestAndAssertType(t, r)
			currentName.Store(payload.Name)
			currentPrefix.Store(payload.Prefix)
			currentAnnounced.Store(payload.Announced)
			currentDeploymentState.Store(payload.DeploymentState)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, routedDsrResponse(currentName.Load().(string), currentPrefix.Load().(string), currentAnnounced.Load(), currentDeploymentState.Load().(string)))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			writeJSONAPI(w, routedDsrResponse(currentName.Load().(string), currentPrefix.Load().(string), currentAnnounced.Load(), currentDeploymentState.Load().(string)))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			putCount.Add(1)
			payload := readRoutedDsrRequestAndAssertType(t, r)
			currentName.Store(payload.Name)
			currentPrefix.Store(payload.Prefix)
			currentAnnounced.Store(payload.Announced)
			currentDeploymentState.Store(payload.DeploymentState)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, routedDsrResponse(currentName.Load().(string), currentPrefix.Load().(string), currentAnnounced.Load(), currentDeploymentState.Load().(string)))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			deleteCount.Add(1)
			if currentDeploymentState.Load().(string) == "DEPLOYED" {
				http.Error(w, "deployed traffic config cannot be deleted", http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: routedDsrConfig(server.URL, "test-dsr", "192.0.2.0/24", nil, ""),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "id", "traffic-config-id"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "name", "test-dsr"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "prefix", "192.0.2.0/24"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "announced", "false"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "deployment_state", "UNDEPLOYED"),
				),
			},
			{
				ResourceName:      "baffinbay_routed_dsr.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: routedDsrConfig(server.URL, "test-dsr-updated", "198.51.100.0/24", boolPtr(true), "DEPLOYED"),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_routed_dsr.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "name", "test-dsr-updated"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "prefix", "198.51.100.0/24"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "announced", "true"),
					tfresource.TestCheckResourceAttr("baffinbay_routed_dsr.test", "deployment_state", "DEPLOYED"),
				),
			},
		},
	})

	if putCount.Load() != 2 {
		t.Fatalf("expected Routed DSR update and destroy undeploy to call PUT, got %d", putCount.Load())
	}
	if deleteCount.Load() == 0 {
		t.Fatalf("expected Routed DSR destroy to call DELETE")
	}
	if got := currentDeploymentState.Load().(string); got != "UNDEPLOYED" {
		t.Fatalf("expected Routed DSR destroy to undeploy before delete, got %q", got)
	}
}

func TestRoutedDsrResource_ReadWrongTypeFails(t *testing.T) {
	setTestTokenCache(t)

	var remoteType atomic.Value
	remoteType.Store(routedDsrTrafficConfigType)

	server := routedDsrWrongTypeServer(t, &remoteType)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: routedDsrConfig(server.URL, "test-dsr", "192.0.2.0/24", nil, ""),
			},
			{
				PreConfig: func() {
					remoteType.Store("l4Proxy")
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Traffic Config Type Mismatch|has type "l4Proxy", expected "routedDsr"`),
			},
		},
	})
}

func TestRoutedDsrResource_ImportWrongTypeFails(t *testing.T) {
	setTestTokenCache(t)

	var remoteType atomic.Value
	remoteType.Store(routedDsrTrafficConfigType)

	server := routedDsrWrongTypeServer(t, &remoteType)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: routedDsrConfig(server.URL, "test-dsr", "192.0.2.0/24", nil, ""),
			},
			{
				PreConfig: func() {
					remoteType.Store("httpProxy")
				},
				ResourceName:  "baffinbay_routed_dsr.test",
				ImportState:   true,
				ImportStateId: "traffic-config-id",
				ExpectError:   regexp.MustCompile(`Traffic Config Type Mismatch|has type "httpProxy", expected "routedDsr"`),
			},
		},
	})
}

type routedDsrRequestPayload struct {
	Name            string
	Prefix          string
	Announced       bool
	DeploymentState string
}

func assertRoutedDsrStringAttribute(t *testing.T, resp *fwresource.SchemaResponse, name string, required, computed bool) {
	t.Helper()

	attr, ok := resp.Schema.Attributes[name].(fwrschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute has type %T, want schema.StringAttribute", name, resp.Schema.Attributes[name])
	}
	if attr.Required != required {
		t.Fatalf("%s required: got %t, want %t", name, attr.Required, required)
	}
	if attr.Computed != computed {
		t.Fatalf("%s computed: got %t, want %t", name, attr.Computed, computed)
	}
	if name == "deployment_state" && attr.Default == nil {
		t.Fatal("deployment_state should default to UNDEPLOYED")
	}
}

func readRoutedDsrRequestAndAssertType(t *testing.T, r *http.Request) routedDsrRequestPayload {
	t.Helper()

	var req struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				Name             string           `json:"name"`
				Prefix           string           `json:"prefix"`
				Announced        *bool            `json:"announced"`
				Deployment       deploymentState  `json:"deployment"`
				Frontend         *json.RawMessage `json:"frontend"`
				Backend          *json.RawMessage `json:"backend"`
				WAF              *json.RawMessage `json:"waf"`
				ProtocolSettings *json.RawMessage `json:"protocolSettings"`
				RateLimiting     *json.RawMessage `json:"rateLimiting"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("failed to decode Routed DSR request: %v", err)
		return routedDsrRequestPayload{}
	}

	if req.Data.Type != routedDsrTrafficConfigType {
		t.Errorf("expected Routed DSR request type %q, got %q", routedDsrTrafficConfigType, req.Data.Type)
	}
	if req.Data.Attributes.Announced == nil {
		t.Errorf("expected Routed DSR request to include announced")
	}
	assertRoutedDsrRequestOmitsAttribute(t, "frontend", req.Data.Attributes.Frontend)
	assertRoutedDsrRequestOmitsAttribute(t, "backend", req.Data.Attributes.Backend)
	assertRoutedDsrRequestOmitsAttribute(t, "waf", req.Data.Attributes.WAF)
	assertRoutedDsrRequestOmitsAttribute(t, "protocolSettings", req.Data.Attributes.ProtocolSettings)
	assertRoutedDsrRequestOmitsAttribute(t, "rateLimiting", req.Data.Attributes.RateLimiting)

	announced := false
	if req.Data.Attributes.Announced != nil {
		announced = *req.Data.Attributes.Announced
	}

	return routedDsrRequestPayload{
		Name:            req.Data.Attributes.Name,
		Prefix:          req.Data.Attributes.Prefix,
		Announced:       announced,
		DeploymentState: req.Data.Attributes.Deployment.State,
	}
}

type deploymentState struct {
	State string `json:"state"`
}

func assertRoutedDsrRequestOmitsAttribute(t *testing.T, name string, value *json.RawMessage) {
	t.Helper()

	if value != nil {
		t.Errorf("Routed DSR request unexpectedly included %s", name)
	}
}

func routedDsrWrongTypeServer(t *testing.T, remoteType *atomic.Value) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			writeJSONAPI(w, routedDsrResponse("test-dsr", "192.0.2.0/24", false, "UNDEPLOYED"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			switch remoteType.Load().(string) {
			case routedDsrTrafficConfigType:
				writeJSONAPI(w, routedDsrResponse("test-dsr", "192.0.2.0/24", false, "UNDEPLOYED"))
			case "httpProxy":
				writeJSONAPI(w, trafficConfigHTTPProxyResponse("test-http"))
			default:
				writeJSONAPI(w, trafficConfigL4Response("test-l4"))
			}
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func routedDsrConfig(serverURL, name, prefix string, announced *bool, deploymentState string) string {
	announcedConfig := ""
	if announced != nil {
		announcedConfig = fmt.Sprintf("\n  announced = %t", *announced)
	}

	deploymentStateConfig := ""
	if deploymentState != "" {
		deploymentStateConfig = fmt.Sprintf("\n  deployment_state = %q", deploymentState)
	}

	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_routed_dsr" "test" {
  name   = %q
  prefix = %q%s%s
}
`, name, prefix, announcedConfig, deploymentStateConfig)
}

func routedDsrResponse(name, prefix string, announced bool, deploymentState string) string {
	return trafficConfigJSON(routedDsrTrafficConfigType, fmt.Sprintf(`"name":%q,"prefix":%q,"announced":%t,"deployment":{"state":%q}`, name, prefix, announced, deploymentState), "")
}

func boolPtr(value bool) *bool {
	return &value
}
