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

func TestProviderResourcesIncludeL4ProxyAndExistingTrafficConfigResources(t *testing.T) {
	p := &BaffinBayProvider{}
	resourceTypes := map[string]bool{}

	for _, newResource := range p.Resources(context.Background()) {
		r := newResource()
		resp := &fwresource.MetadataResponse{}
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		resourceTypes[resp.TypeName] = true
	}

	if !resourceTypes["baffinbay_traffic_config"] {
		t.Fatal("expected baffinbay_traffic_config to remain registered")
	}
	if !resourceTypes["baffinbay_routed_dsr"] {
		t.Fatal("expected baffinbay_routed_dsr to remain registered")
	}
	if !resourceTypes["baffinbay_l4_proxy"] {
		t.Fatal("expected baffinbay_l4_proxy to be registered")
	}
}

func TestL4ProxyResourceSchema(t *testing.T) {
	r := NewL4ProxyResource()
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema generation failed: %s", resp.Diagnostics)
	}

	unsupportedAttributes := []string{
		"type",
		"geo_fencing",
		"allowed_sources",
		"waf",
		"protocol_settings",
		"rate_limiting",
		"prefix",
		"announced",
	}
	for _, attrName := range unsupportedAttributes {
		if _, ok := resp.Schema.Attributes[attrName]; ok {
			t.Fatalf("L4 Proxy schema unexpectedly included %q", attrName)
		}
	}

	assertL4ProxyStringAttribute(t, resp, "id", false, true)
	assertL4ProxyStringAttribute(t, resp, "name", true, false)
	assertL4ProxyStringAttribute(t, resp, "deployment_state", false, true)
	assertL4ProxyStringAttribute(t, resp, "proxy_protocol", false, true)
	assertL4ProxyListAttribute(t, resp, "protocols", false, true)

	ipBasedAccessAttr, ok := resp.Schema.Attributes["ip_based_access_control"].(fwrschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("ip_based_access_control attribute has type %T, want schema.SingleNestedAttribute", resp.Schema.Attributes["ip_based_access_control"])
	}
	if !ipBasedAccessAttr.Optional || !ipBasedAccessAttr.Computed {
		t.Fatal("ip_based_access_control should be optional and computed")
	}
	assertNestedStringAttribute(t, ipBasedAccessAttr.Attributes, "default_policy", false, true)
	rulesAttr, ok := ipBasedAccessAttr.Attributes["rules"].(fwrschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("ip_based_access_control.rules attribute has type %T, want schema.SingleNestedAttribute", ipBasedAccessAttr.Attributes["rules"])
	}
	if !rulesAttr.Optional || !rulesAttr.Computed {
		t.Fatal("ip_based_access_control.rules should be optional and computed")
	}
	ipRangesAttr, ok := rulesAttr.Attributes["ip_ranges"].(fwrschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("ip_based_access_control.rules.ip_ranges attribute has type %T, want schema.ListNestedAttribute", rulesAttr.Attributes["ip_ranges"])
	}
	if !ipRangesAttr.Optional || !ipRangesAttr.Computed {
		t.Fatal("ip_based_access_control.rules.ip_ranges should be optional and computed")
	}
	assertNestedStringAttribute(t, ipRangesAttr.NestedObject.Attributes, "policy", true, false)
	assertNestedStringAttribute(t, ipRangesAttr.NestedObject.Attributes, "address", true, false)

	geoLocationsAttr, ok := rulesAttr.Attributes["geo_locations"].(fwrschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("ip_based_access_control.rules.geo_locations attribute has type %T, want schema.ListNestedAttribute", rulesAttr.Attributes["geo_locations"])
	}
	if !geoLocationsAttr.Optional || !geoLocationsAttr.Computed {
		t.Fatal("ip_based_access_control.rules.geo_locations should be optional and computed")
	}
	assertNestedStringAttribute(t, geoLocationsAttr.NestedObject.Attributes, "policy", true, false)
	assertNestedStringAttribute(t, geoLocationsAttr.NestedObject.Attributes, "region", true, false)

	for _, attrName := range []string{"ip_lists", "asns"} {
		attr, ok := rulesAttr.Attributes[attrName].(fwrschema.ListNestedAttribute)
		if !ok {
			t.Fatalf("ip_based_access_control.rules.%s attribute has type %T, want schema.ListNestedAttribute", attrName, rulesAttr.Attributes[attrName])
		}
		if !attr.Optional || !attr.Computed {
			t.Fatalf("ip_based_access_control.rules.%s should be optional and computed", attrName)
		}
		assertNestedStringAttribute(t, attr.NestedObject.Attributes, "policy", true, false)
	}

	frontendAttr, ok := resp.Schema.Attributes["frontend"].(fwrschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("frontend attribute has type %T, want schema.SingleNestedAttribute", resp.Schema.Attributes["frontend"])
	}
	if !frontendAttr.Required {
		t.Fatal("frontend should be required")
	}
	assertNestedStringAttribute(t, frontendAttr.Attributes, "ipv4", false, false)
	assertNestedStringAttribute(t, frontendAttr.Attributes, "ipv6", false, false)
	assertNestedInt64Attribute(t, frontendAttr.Attributes, "port", true)

	unsupportedFrontendAttributes := []string{
		"connection_type",
		"redirect_http",
		"hosts",
		"hsts",
		"client_certificate_verification",
	}
	for _, attrName := range unsupportedFrontendAttributes {
		if _, ok := frontendAttr.Attributes[attrName]; ok {
			t.Fatalf("L4 Proxy frontend schema unexpectedly included %q", attrName)
		}
	}

	backendAttr, ok := resp.Schema.Attributes["backend"].(fwrschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("backend attribute has type %T, want schema.SingleNestedAttribute", resp.Schema.Attributes["backend"])
	}
	if !backendAttr.Required {
		t.Fatal("backend should be required")
	}
	if _, ok := backendAttr.Attributes["tls_settings"]; ok {
		t.Fatal("L4 Proxy backend schema unexpectedly included tls_settings")
	}

	hostsAttr, ok := backendAttr.Attributes["hosts"].(fwrschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("backend.hosts attribute has type %T, want schema.ListNestedAttribute", backendAttr.Attributes["hosts"])
	}
	if !hostsAttr.Required {
		t.Fatal("backend.hosts should be required")
	}
	assertNestedStringAttribute(t, hostsAttr.NestedObject.Attributes, "address", true, false)
	assertNestedInt64Attribute(t, hostsAttr.NestedObject.Attributes, "port", true)
	assertNestedStringAttribute(t, backendAttr.Attributes, "delivery_method", true, false)
	assertNestedStringAttribute(t, backendAttr.Attributes, "server_name", true, false)
}

func TestL4ProxyResource_FrontendRequiresIPv4OrIPv6(t *testing.T) {
	setTestTokenCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	payload := defaultL4ProxyPayload("test-l4")
	payload.Frontend.IPv4 = ""
	payload.Frontend.IPv6 = ""

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config:      l4ProxyConfig(server.URL, payload),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`At least one of|frontend\.ipv4|frontend\.ipv6`),
			},
		},
	})
}

func TestL4ProxyResource_ProtocolsValidation(t *testing.T) {
	setTestTokenCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	payload := defaultL4ProxyPayload("test-l4")
	payload.Protocols = []string{"ICMP"}

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config:      l4ProxyConfig(server.URL, payload),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`expected value to be one of|TCP|UDP|ICMP`),
			},
		},
	})
}

func TestL4ProxyResource_LifecycleUpdateImportAndDelete(t *testing.T) {
	setTestTokenCache(t)

	var current atomic.Value
	current.Store(l4ProxyRequestPayload{
		Name:            "test-l4",
		DeploymentState: "UNDEPLOYED",
		Protocols:       []string{"TCP"},
		ProxyProtocol:   "DISABLED",
		IPBasedAccess: l4ProxyIPBasedAccessPayload{
			DefaultPolicy: "ALLOW",
			Rules: l4ProxyIPBasedAccessRulesPayload{
				IPRanges:     []l4ProxyIPRangeRulePayload{},
				IPLists:      []l4ProxyIPListRulePayload{},
				GeoLocations: []l4ProxyGeoLocationRulePayload{},
				ASNs:         []l4ProxyAutonomousSystemRulePayload{},
			},
		},
		Frontend: l4ProxyFrontendPayload{
			IPv4: "192.168.1.1",
			Port: 80,
		},
		Backend: l4ProxyBackendPayload{
			Hosts: []l4ProxyBackendHostPayload{
				{
					Address: "example.com",
					Port:    8080,
				},
			},
			DeliveryMethod: "ROUND_ROBIN",
			ServerName:     "example.com",
		},
	})
	var putCount atomic.Int64
	var deleteCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			payload := readL4ProxyRequestAndAssertType(t, r)
			current.Store(payload)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, l4ProxyResponse(payload))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			payload := current.Load().(l4ProxyRequestPayload)
			writeJSONAPI(w, l4ProxyResponse(payload))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			putCount.Add(1)
			payload := readL4ProxyRequestAndAssertType(t, r)
			current.Store(payload)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, l4ProxyResponse(payload))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			deleteCount.Add(1)
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
				Config: l4ProxyConfig(server.URL, l4ProxyRequestPayload{
					Name:            "test-l4",
					DeploymentState: "",
					Frontend: l4ProxyFrontendPayload{
						IPv4: "192.168.1.1",
						Port: 80,
					},
					Backend: l4ProxyBackendPayload{
						Hosts: []l4ProxyBackendHostPayload{
							{
								Address: "example.com",
								Port:    8080,
							},
						},
						DeliveryMethod: "ROUND_ROBIN",
						ServerName:     "example.com",
					},
				}),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "id", "traffic-config-id"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "name", "test-l4"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "deployment_state", "UNDEPLOYED"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "protocols.#", "1"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "protocols.0", "TCP"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "proxy_protocol", "DISABLED"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.default_policy", "ALLOW"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_ranges.#", "0"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_lists.#", "0"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.#", "0"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.asns.#", "0"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "frontend.ipv4", "192.168.1.1"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "frontend.port", "80"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.hosts.0.address", "example.com"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.hosts.0.port", "8080"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.delivery_method", "ROUND_ROBIN"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.server_name", "example.com"),
				),
			},
			{
				ResourceName:      "baffinbay_l4_proxy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: l4ProxyConfig(server.URL, l4ProxyRequestPayload{
					Name:            "test-l4-updated",
					DeploymentState: "DEPLOYED",
					Protocols:       []string{"TCP", "UDP"},
					ProxyProtocol:   "ENABLED",
					IPBasedAccess: l4ProxyIPBasedAccessPayload{
						DefaultPolicy: "BLOCK",
						Rules: l4ProxyIPBasedAccessRulesPayload{
							IPRanges: []l4ProxyIPRangeRulePayload{
								{
									Policy:  "ALLOW",
									Address: "198.51.100.0/24",
									Note:    "office",
								},
							},
							IPLists: []l4ProxyIPListRulePayload{
								{
									ID:     "11111111-1111-1111-1111-111111111111",
									Policy: "BLOCK",
								},
							},
							GeoLocations: []l4ProxyGeoLocationRulePayload{
								{
									Policy: "ALLOW",
									Region: "SE",
									Note:   "sweden",
								},
								{
									Policy: "ALLOW",
									Region: "NO",
									Note:   "norway",
								},
							},
							ASNs: []l4ProxyAutonomousSystemRulePayload{
								{
									Policy: "BLOCK",
									ASN:    64512,
									Note:   "test-asn",
								},
							},
						},
					},
					Frontend: l4ProxyFrontendPayload{
						IPv4: "192.168.1.2",
						IPv6: "2001:db8::8a2e:370:7334",
						Port: 443,
					},
					Backend: l4ProxyBackendPayload{
						Hosts: []l4ProxyBackendHostPayload{
							{
								Address: "origin.example.com",
								Port:    8443,
							},
						},
						DeliveryMethod: "LEAST_CONNECTIONS",
						ServerName:     "origin.example.com",
					},
				}),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_l4_proxy.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "name", "test-l4-updated"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "deployment_state", "DEPLOYED"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "protocols.#", "2"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "protocols.0", "TCP"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "protocols.1", "UDP"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "proxy_protocol", "ENABLED"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.default_policy", "BLOCK"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_ranges.#", "1"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_ranges.0.policy", "ALLOW"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_ranges.0.address", "198.51.100.0/24"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_ranges.0.note", "office"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_lists.#", "1"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_lists.0.id", "11111111-1111-1111-1111-111111111111"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.ip_lists.0.policy", "BLOCK"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.#", "2"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.0.policy", "ALLOW"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.0.region", "SE"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.0.note", "sweden"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.1.policy", "ALLOW"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.1.region", "NO"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.geo_locations.1.note", "norway"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.asns.#", "1"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.asns.0.policy", "BLOCK"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.asns.0.asn", "64512"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "ip_based_access_control.rules.asns.0.note", "test-asn"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "frontend.ipv4", "192.168.1.2"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "frontend.ipv6", "2001:db8::8a2e:370:7334"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "frontend.port", "443"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.hosts.0.address", "origin.example.com"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.hosts.0.port", "8443"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.delivery_method", "LEAST_CONNECTIONS"),
					tfresource.TestCheckResourceAttr("baffinbay_l4_proxy.test", "backend.server_name", "origin.example.com"),
				),
			},
		},
	})

	if putCount.Load() == 0 {
		t.Fatalf("expected L4 Proxy update to call PUT")
	}
	if deleteCount.Load() == 0 {
		t.Fatalf("expected L4 Proxy destroy to call DELETE")
	}
}

func TestL4ProxyResource_ReadWrongTypeFails(t *testing.T) {
	setTestTokenCache(t)

	var remoteType atomic.Value
	remoteType.Store(l4ProxyTrafficConfigType)

	server := l4ProxyWrongTypeServer(t, &remoteType)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: l4ProxyConfig(server.URL, defaultL4ProxyPayload("test-l4")),
			},
			{
				PreConfig: func() {
					remoteType.Store(routedDsrTrafficConfigType)
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Traffic Config Type Mismatch|has type "routedDsr", expected "l4Proxy"`),
			},
		},
	})
}

func TestL4ProxyResource_ImportWrongTypeFails(t *testing.T) {
	setTestTokenCache(t)

	var remoteType atomic.Value
	remoteType.Store(l4ProxyTrafficConfigType)

	server := l4ProxyWrongTypeServer(t, &remoteType)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: l4ProxyConfig(server.URL, defaultL4ProxyPayload("test-l4")),
			},
			{
				PreConfig: func() {
					remoteType.Store("httpProxy")
				},
				ResourceName:  "baffinbay_l4_proxy.test",
				ImportState:   true,
				ImportStateId: "traffic-config-id",
				ExpectError:   regexp.MustCompile(`Traffic Config Type Mismatch|has type "httpProxy", expected "l4Proxy"`),
			},
		},
	})
}

type l4ProxyRequestPayload struct {
	Name            string
	DeploymentState string
	Protocols       []string
	ProxyProtocol   string
	IPBasedAccess   l4ProxyIPBasedAccessPayload
	Frontend        l4ProxyFrontendPayload
	Backend         l4ProxyBackendPayload
}

type l4ProxyIPBasedAccessPayload struct {
	DefaultPolicy string                           `json:"defaultPolicy"`
	Rules         l4ProxyIPBasedAccessRulesPayload `json:"rules"`
}

type l4ProxyIPBasedAccessRulesPayload struct {
	IPRanges      []l4ProxyIPRangeRulePayload          `json:"ipRanges"`
	KnownServices *json.RawMessage                     `json:"knownServices,omitempty"`
	IPLists       []l4ProxyIPListRulePayload           `json:"ipLists"`
	GeoLocations  []l4ProxyGeoLocationRulePayload      `json:"geoLocations"`
	ASNs          []l4ProxyAutonomousSystemRulePayload `json:"asns"`
}

type l4ProxyIPRangeRulePayload struct {
	Policy  string `json:"policy"`
	Address string `json:"address"`
	Note    string `json:"note,omitempty"`
}

type l4ProxyIPListRulePayload struct {
	Type   string `json:"type,omitempty"`
	ID     string `json:"id"`
	Policy string `json:"policy"`
}

type l4ProxyGeoLocationRulePayload struct {
	Policy string `json:"policy"`
	Region string `json:"region"`
	Note   string `json:"note,omitempty"`
}

type l4ProxyAutonomousSystemRulePayload struct {
	Policy string `json:"policy"`
	ASN    int64  `json:"asn"`
	Note   string `json:"note,omitempty"`
}

type l4ProxyFrontendPayload struct {
	IPv4 string
	IPv6 string
	IP   string
	Port int64
}

type l4ProxyBackendPayload struct {
	Hosts          []l4ProxyBackendHostPayload
	DeliveryMethod string
	ServerName     string
}

type l4ProxyBackendHostPayload struct {
	Address string
	Port    int64
}

func assertL4ProxyStringAttribute(t *testing.T, resp *fwresource.SchemaResponse, name string, required, computed bool) {
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
	if name == "proxy_protocol" && attr.Default == nil {
		t.Fatal("proxy_protocol should default to DISABLED")
	}
}

func assertL4ProxyListAttribute(t *testing.T, resp *fwresource.SchemaResponse, name string, required, computed bool) {
	t.Helper()

	attr, ok := resp.Schema.Attributes[name].(fwrschema.ListAttribute)
	if !ok {
		t.Fatalf("%s attribute has type %T, want schema.ListAttribute", name, resp.Schema.Attributes[name])
	}
	if attr.Required != required {
		t.Fatalf("%s required: got %t, want %t", name, attr.Required, required)
	}
	if attr.Computed != computed {
		t.Fatalf("%s computed: got %t, want %t", name, attr.Computed, computed)
	}
	if name == "protocols" && attr.Default == nil {
		t.Fatal("protocols should default to [TCP]")
	}
}

func assertNestedStringAttribute(t *testing.T, attrs map[string]fwrschema.Attribute, name string, required, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(fwrschema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute has type %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required {
		t.Fatalf("%s required: got %t, want %t", name, attr.Required, required)
	}
	if attr.Computed != computed {
		t.Fatalf("%s computed: got %t, want %t", name, attr.Computed, computed)
	}
}

func assertNestedListAttribute(t *testing.T, attrs map[string]fwrschema.Attribute, name string, required, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(fwrschema.ListAttribute)
	if !ok {
		t.Fatalf("%s attribute has type %T, want schema.ListAttribute", name, attrs[name])
	}
	if attr.Required != required {
		t.Fatalf("%s required: got %t, want %t", name, attr.Required, required)
	}
	if attr.Computed != computed {
		t.Fatalf("%s computed: got %t, want %t", name, attr.Computed, computed)
	}
}

func assertNestedInt64Attribute(t *testing.T, attrs map[string]fwrschema.Attribute, name string, required bool) {
	t.Helper()

	attr, ok := attrs[name].(fwrschema.Int64Attribute)
	if !ok {
		t.Fatalf("%s attribute has type %T, want schema.Int64Attribute", name, attrs[name])
	}
	if attr.Required != required {
		t.Fatalf("%s required: got %t, want %t", name, attr.Required, required)
	}
}

func readL4ProxyRequestAndAssertType(t *testing.T, r *http.Request) l4ProxyRequestPayload {
	t.Helper()

	var req struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				Version  string `json:"version"`
				Name     string `json:"name"`
				Frontend *struct {
					IPv4                          string           `json:"ipv4"`
					IPv6                          string           `json:"ipv6"`
					IP                            string           `json:"ip"`
					Port                          int64            `json:"port"`
					ConnectionType                *json.RawMessage `json:"connectionType"`
					RedirectHttp                  *json.RawMessage `json:"redirectHttp"`
					Hosts                         *json.RawMessage `json:"hosts"`
					HTTPStrictTransportSecurity   *json.RawMessage `json:"httpStrictTransportSecurity"`
					ClientCertificateVerification *json.RawMessage `json:"clientCertificateVerification"`
				} `json:"frontend"`
				Backend *struct {
					Hosts []struct {
						Address string `json:"address"`
						Port    int64  `json:"port"`
					} `json:"hosts"`
					DeliveryMethod string           `json:"deliveryMethod"`
					ServerName     string           `json:"serverName"`
					TLSSettings    *json.RawMessage `json:"tlsSettings"`
				} `json:"backend"`
				Deployment       deploymentState              `json:"deployment"`
				Protocols        []string                     `json:"protocols"`
				ProxyProtocol    string                       `json:"proxyProtocol"`
				GeoFencing       *json.RawMessage             `json:"geoFencing"`
				AllowedSources   *json.RawMessage             `json:"allowedSources"`
				IPBasedAccess    *l4ProxyIPBasedAccessPayload `json:"ipBasedAccessControl"`
				Prefix           *json.RawMessage             `json:"prefix"`
				Announced        *json.RawMessage             `json:"announced"`
				WAF              *json.RawMessage             `json:"waf"`
				ProtocolSettings *json.RawMessage             `json:"protocolSettings"`
				RateLimiting     *json.RawMessage             `json:"rateLimiting"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("failed to decode L4 Proxy request: %v", err)
		return l4ProxyRequestPayload{}
	}

	if req.Data.Type != l4ProxyTrafficConfigType {
		t.Errorf("expected L4 Proxy request type %q, got %q", l4ProxyTrafficConfigType, req.Data.Type)
	}
	if req.Data.Attributes.Version != l4ProxyTrafficConfigVersion {
		t.Errorf("expected L4 Proxy request version %q, got %q", l4ProxyTrafficConfigVersion, req.Data.Attributes.Version)
	}
	if len(req.Data.Attributes.Protocols) == 0 {
		t.Errorf("expected L4 Proxy request to include at least one protocol")
	}
	if req.Data.Attributes.ProxyProtocol == "" {
		t.Errorf("expected L4 Proxy request to include proxyProtocol")
	}
	assertL4ProxyRequestOmitsAttribute(t, "prefix", req.Data.Attributes.Prefix)
	assertL4ProxyRequestOmitsAttribute(t, "announced", req.Data.Attributes.Announced)
	assertL4ProxyRequestOmitsAttribute(t, "waf", req.Data.Attributes.WAF)
	assertL4ProxyRequestOmitsAttribute(t, "protocolSettings", req.Data.Attributes.ProtocolSettings)
	assertL4ProxyRequestOmitsAttribute(t, "rateLimiting", req.Data.Attributes.RateLimiting)
	assertL4ProxyRequestOmitsAttribute(t, "geoFencing", req.Data.Attributes.GeoFencing)
	assertL4ProxyRequestOmitsAttribute(t, "allowedSources", req.Data.Attributes.AllowedSources)
	if req.Data.Attributes.IPBasedAccess == nil {
		t.Errorf("expected L4 Proxy request to include ipBasedAccessControl")
	}
	assertL4ProxyIPListRuleTypes(t, req.Data.Attributes.IPBasedAccess)
	assertL4ProxyRequestOmitsKnownServices(t, req.Data.Attributes.IPBasedAccess)

	if req.Data.Attributes.Frontend == nil {
		t.Errorf("expected L4 Proxy request to include frontend")
		return l4ProxyRequestPayload{}
	}
	assertL4ProxyRequestOmitsAttribute(t, "frontend.connectionType", req.Data.Attributes.Frontend.ConnectionType)
	assertL4ProxyRequestOmitsAttribute(t, "frontend.redirectHttp", req.Data.Attributes.Frontend.RedirectHttp)
	assertL4ProxyRequestOmitsAttribute(t, "frontend.hosts", req.Data.Attributes.Frontend.Hosts)
	assertL4ProxyRequestOmitsAttribute(t, "frontend.httpStrictTransportSecurity", req.Data.Attributes.Frontend.HTTPStrictTransportSecurity)
	assertL4ProxyRequestOmitsAttribute(t, "frontend.clientCertificateVerification", req.Data.Attributes.Frontend.ClientCertificateVerification)

	if req.Data.Attributes.Frontend.IPv4 != "" && req.Data.Attributes.Frontend.IP != req.Data.Attributes.Frontend.IPv4 {
		t.Errorf("expected L4 Proxy request legacy ip to mirror ipv4, got %q", req.Data.Attributes.Frontend.IP)
	}
	if req.Data.Attributes.Frontend.IPv4 == "" && req.Data.Attributes.Frontend.IPv6 != "" && req.Data.Attributes.Frontend.IP != req.Data.Attributes.Frontend.IPv6 {
		t.Errorf("expected L4 Proxy request legacy ip to mirror ipv6 when ipv4 is unset, got %q", req.Data.Attributes.Frontend.IP)
	}

	if req.Data.Attributes.Backend == nil {
		t.Errorf("expected L4 Proxy request to include backend")
		return l4ProxyRequestPayload{}
	}
	assertL4ProxyRequestOmitsAttribute(t, "backend.tlsSettings", req.Data.Attributes.Backend.TLSSettings)

	backendHosts := make([]l4ProxyBackendHostPayload, 0, len(req.Data.Attributes.Backend.Hosts))
	for _, host := range req.Data.Attributes.Backend.Hosts {
		backendHosts = append(backendHosts, l4ProxyBackendHostPayload{
			Address: host.Address,
			Port:    host.Port,
		})
	}

	return l4ProxyRequestPayload{
		Name:            req.Data.Attributes.Name,
		DeploymentState: req.Data.Attributes.Deployment.State,
		Protocols:       req.Data.Attributes.Protocols,
		ProxyProtocol:   req.Data.Attributes.ProxyProtocol,
		IPBasedAccess:   l4ProxyIPBasedAccessOrDefault(derefL4ProxyIPBasedAccess(req.Data.Attributes.IPBasedAccess)),
		Frontend: l4ProxyFrontendPayload{
			IPv4: req.Data.Attributes.Frontend.IPv4,
			IPv6: req.Data.Attributes.Frontend.IPv6,
			IP:   req.Data.Attributes.Frontend.IP,
			Port: req.Data.Attributes.Frontend.Port,
		},
		Backend: l4ProxyBackendPayload{
			Hosts:          backendHosts,
			DeliveryMethod: req.Data.Attributes.Backend.DeliveryMethod,
			ServerName:     req.Data.Attributes.Backend.ServerName,
		},
	}
}

func assertL4ProxyRequestOmitsKnownServices(t *testing.T, ipBasedAccess *l4ProxyIPBasedAccessPayload) {
	t.Helper()

	if ipBasedAccess == nil {
		return
	}
	if ipBasedAccess.Rules.KnownServices != nil {
		t.Errorf("L4 Proxy request unexpectedly included ipBasedAccessControl.rules.knownServices")
	}
}

func assertL4ProxyIPListRuleTypes(t *testing.T, ipBasedAccess *l4ProxyIPBasedAccessPayload) {
	t.Helper()

	if ipBasedAccess == nil {
		return
	}
	for i, rule := range ipBasedAccess.Rules.IPLists {
		if rule.Type != "ipList" {
			t.Errorf("expected ipBasedAccessControl.rules.ipLists[%d].type to be ipList, got %q", i, rule.Type)
		}
	}
}

func derefL4ProxyIPBasedAccess(value *l4ProxyIPBasedAccessPayload) l4ProxyIPBasedAccessPayload {
	if value == nil {
		return l4ProxyIPBasedAccessPayload{}
	}

	return *value
}

func assertL4ProxyRequestOmitsAttribute(t *testing.T, name string, value *json.RawMessage) {
	t.Helper()

	if value != nil {
		t.Errorf("L4 Proxy request unexpectedly included %s", name)
	}
}

func l4ProxyWrongTypeServer(t *testing.T, remoteType *atomic.Value) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			writeJSONAPI(w, l4ProxyResponse(defaultL4ProxyPayload("test-l4")))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			switch remoteType.Load().(string) {
			case l4ProxyTrafficConfigType:
				writeJSONAPI(w, l4ProxyResponse(defaultL4ProxyPayload("test-l4")))
			case "httpProxy":
				writeJSONAPI(w, trafficConfigHTTPProxyResponse("test-http"))
			default:
				writeJSONAPI(w, routedDsrResponse("test-dsr", "192.0.2.0/24", false, "UNDEPLOYED"))
			}
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func defaultL4ProxyPayload(name string) l4ProxyRequestPayload {
	return l4ProxyRequestPayload{
		Name:            name,
		DeploymentState: "UNDEPLOYED",
		Protocols:       []string{"TCP"},
		ProxyProtocol:   "DISABLED",
		Frontend: l4ProxyFrontendPayload{
			IPv4: "192.168.1.1",
			Port: 80,
		},
		Backend: l4ProxyBackendPayload{
			Hosts: []l4ProxyBackendHostPayload{
				{
					Address: "example.com",
					Port:    8080,
				},
			},
			DeliveryMethod: "ROUND_ROBIN",
			ServerName:     "example.com",
		},
	}
}

func l4ProxyConfig(serverURL string, payload l4ProxyRequestPayload) string {
	deploymentStateConfig := ""
	if payload.DeploymentState != "" {
		deploymentStateConfig = fmt.Sprintf("\n  deployment_state = %q", payload.DeploymentState)
	}

	protocolsConfig := ""
	if len(payload.Protocols) > 0 {
		protocolsConfig = "\n  protocols = ["
		for i, protocol := range payload.Protocols {
			if i > 0 {
				protocolsConfig += ", "
			}
			protocolsConfig += fmt.Sprintf("%q", protocol)
		}
		protocolsConfig += "]"
	}

	proxyProtocolConfig := ""
	if payload.ProxyProtocol != "" {
		proxyProtocolConfig = fmt.Sprintf("\n  proxy_protocol = %q", payload.ProxyProtocol)
	}

	accessConfig := l4ProxyAccessConfig(payload)

	ipv4Config := ""
	if payload.Frontend.IPv4 != "" {
		ipv4Config = fmt.Sprintf("\n    ipv4 = %q", payload.Frontend.IPv4)
	}

	ipv6Config := ""
	if payload.Frontend.IPv6 != "" {
		ipv6Config = fmt.Sprintf("\n    ipv6 = %q", payload.Frontend.IPv6)
	}

	hostConfig := ""
	for _, host := range payload.Backend.Hosts {
		hostConfig += fmt.Sprintf(`
      {
        address = %q
        port    = %d
      }`, host.Address, host.Port)
	}

	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_l4_proxy" "test" {
  name = %q%s%s%s%s

  frontend = {%s%s
    port = %d
  }

  backend = {
    hosts = [%s
    ]
    delivery_method = %q
    server_name     = %q
  }
}
`, payload.Name, deploymentStateConfig, protocolsConfig, proxyProtocolConfig, accessConfig, ipv4Config, ipv6Config, payload.Frontend.Port, hostConfig, payload.Backend.DeliveryMethod, payload.Backend.ServerName)
}

func l4ProxyAccessConfig(payload l4ProxyRequestPayload) string {
	config := ""
	if payload.IPBasedAccess.DefaultPolicy != "" || !payload.IPBasedAccess.Rules.empty() {
		defaultPolicy := payload.IPBasedAccess.DefaultPolicy
		if defaultPolicy == "" {
			defaultPolicy = "ALLOW"
		}
		config += fmt.Sprintf(`

  ip_based_access_control = {
    default_policy = %q
    rules = {
      ip_ranges = [%s
      ]
      ip_lists = [%s
      ]
      geo_locations = [%s
      ]
      asns = [%s
      ]
    }
  }`,
			defaultPolicy,
			hclIPRangeRules(payload.IPBasedAccess.Rules.IPRanges),
			hclIPListRules(payload.IPBasedAccess.Rules.IPLists),
			hclGeoLocationRules(payload.IPBasedAccess.Rules.GeoLocations),
			hclAutonomousSystemRules(payload.IPBasedAccess.Rules.ASNs),
		)
	}

	return config
}

func (rules l4ProxyIPBasedAccessRulesPayload) empty() bool {
	return len(rules.IPRanges) == 0 &&
		len(rules.IPLists) == 0 &&
		len(rules.GeoLocations) == 0 &&
		len(rules.ASNs) == 0
}

func hclIPRangeRules(rules []l4ProxyIPRangeRulePayload) string {
	result := ""
	for i, rule := range rules {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`
        {
          policy  = %q
          address = %q%s
        }`, rule.Policy, rule.Address, hclOptionalNote(rule.Note))
	}

	return result
}

func hclIPListRules(rules []l4ProxyIPListRulePayload) string {
	result := ""
	for i, rule := range rules {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`
        {
          id     = %q
          policy = %q
        }`, rule.ID, rule.Policy)
	}

	return result
}

func hclGeoLocationRules(rules []l4ProxyGeoLocationRulePayload) string {
	result := ""
	for i, rule := range rules {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`
        {
          policy = %q
          region = %q%s
        }`, rule.Policy, rule.Region, hclOptionalNote(rule.Note))
	}

	return result
}

func hclAutonomousSystemRules(rules []l4ProxyAutonomousSystemRulePayload) string {
	result := ""
	for i, rule := range rules {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`
        {
          policy = %q
          asn    = %d%s
        }`, rule.Policy, rule.ASN, hclOptionalNote(rule.Note))
	}

	return result
}

func hclOptionalNote(note string) string {
	if note == "" {
		return ""
	}

	return fmt.Sprintf("\n          note    = %q", note)
}

func l4ProxyResponse(payload l4ProxyRequestPayload) string {
	frontend := fmt.Sprintf(`"port":%d`, payload.Frontend.Port)
	if payload.Frontend.IPv4 != "" {
		frontend = fmt.Sprintf(`"ipv4":%q,%s`, payload.Frontend.IPv4, frontend)
	}
	if payload.Frontend.IPv6 != "" {
		frontend = fmt.Sprintf(`"ipv6":%q,%s`, payload.Frontend.IPv6, frontend)
	}

	hosts := ""
	for i, host := range payload.Backend.Hosts {
		if i > 0 {
			hosts += ","
		}
		hosts += fmt.Sprintf(`{"address":%q,"port":%d}`, host.Address, host.Port)
	}

	protocols := payload.Protocols
	if len(protocols) == 0 {
		protocols = []string{"TCP"}
	}
	protocolJSON, err := json.Marshal(protocols)
	if err != nil {
		panic(err)
	}

	proxyProtocol := payload.ProxyProtocol
	if proxyProtocol == "" {
		proxyProtocol = "DISABLED"
	}

	ipBasedAccessJSON := mustMarshalJSON(l4ProxyIPBasedAccessOrDefault(payload.IPBasedAccess))

	attrs := fmt.Sprintf(`"name":%q,"version":%q,"frontend":{%s},"backend":{"hosts":[%s],"deliveryMethod":%q,"serverName":%q},"deployment":{"state":%q},"protocols":%s,"proxyProtocol":%q,"ipBasedAccessControl":%s`,
		payload.Name,
		l4ProxyTrafficConfigVersion,
		frontend,
		hosts,
		payload.Backend.DeliveryMethod,
		payload.Backend.ServerName,
		payload.DeploymentState,
		string(protocolJSON),
		proxyProtocol,
		string(ipBasedAccessJSON),
	)

	return trafficConfigJSON(l4ProxyTrafficConfigType, attrs, "")
}

func l4ProxyIPBasedAccessOrDefault(ipBasedAccess l4ProxyIPBasedAccessPayload) l4ProxyIPBasedAccessPayload {
	if ipBasedAccess.DefaultPolicy == "" {
		ipBasedAccess.DefaultPolicy = "ALLOW"
	}
	if ipBasedAccess.Rules.IPRanges == nil {
		ipBasedAccess.Rules.IPRanges = []l4ProxyIPRangeRulePayload{}
	}
	if ipBasedAccess.Rules.IPLists == nil {
		ipBasedAccess.Rules.IPLists = []l4ProxyIPListRulePayload{}
	}
	if ipBasedAccess.Rules.GeoLocations == nil {
		ipBasedAccess.Rules.GeoLocations = []l4ProxyGeoLocationRulePayload{}
	}
	if ipBasedAccess.Rules.ASNs == nil {
		ipBasedAccess.Rules.ASNs = []l4ProxyAutonomousSystemRulePayload{}
	}
	for i := range ipBasedAccess.Rules.IPRanges {
		if ipBasedAccess.Rules.IPRanges[i].Policy == "" {
			ipBasedAccess.Rules.IPRanges[i].Policy = "BLOCK"
		}
	}
	for i := range ipBasedAccess.Rules.GeoLocations {
		if ipBasedAccess.Rules.GeoLocations[i].Policy == "" {
			ipBasedAccess.Rules.GeoLocations[i].Policy = "BLOCK"
		}
	}

	return ipBasedAccess
}

func mustMarshalJSON(value any) []byte {
	jsonData, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return jsonData
}
