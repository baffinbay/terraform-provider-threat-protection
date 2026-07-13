package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwrschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	testCertificateID       = "11111111-1111-4111-8111-111111111111"
	testCACertificateID     = "22222222-2222-4222-8222-222222222222"
	testClientCertificateID = "33333333-3333-4333-8333-333333333333"
	testCustomPageID        = "44444444-4444-4444-8444-444444444444"
	testIPListID            = "55555555-5555-4555-8555-555555555555"
	testWAFRuleID           = "66666666-6666-4666-8666-666666666666"
)

func TestProviderResourcesIncludeHTTPProxyAndExistingTrafficConfigResources(t *testing.T) {
	p := &BaffinBayProvider{}
	resourceTypes := map[string]bool{}
	for _, newResource := range p.Resources(context.Background()) {
		r := newResource()
		resp := &fwresource.MetadataResponse{}
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		resourceTypes[resp.TypeName] = true
	}
	for _, resourceType := range []string{"baffinbay_traffic_config", "baffinbay_http_proxy", "baffinbay_l4_proxy", "baffinbay_routed_dsr"} {
		if !resourceTypes[resourceType] {
			t.Fatalf("expected %s to be registered", resourceType)
		}
	}
}

func TestHTTPProxyResourceSchema(t *testing.T) {
	r := NewHTTPProxyResource()
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema generation failed: %s", resp.Diagnostics)
	}
	for _, attrName := range []string{"type", "protocols", "proxy_protocol", "prefix", "announced", "geo_fencing", "allowed_sources", "gateway_path"} {
		if _, ok := resp.Schema.Attributes[attrName]; ok {
			t.Fatalf("HTTP Proxy schema unexpectedly included %q", attrName)
		}
	}
	for _, attrName := range []string{"frontend", "backend", "protocol_settings"} {
		attr, ok := resp.Schema.Attributes[attrName].(fwrschema.SingleNestedAttribute)
		if !ok || !attr.Required {
			t.Fatalf("%s should be a required single nested attribute, got %T", attrName, resp.Schema.Attributes[attrName])
		}
	}
	frontend := resp.Schema.Attributes["frontend"].(fwrschema.SingleNestedAttribute)
	clientVerification := frontend.Attributes["client_certificate_verification"].(fwrschema.SingleNestedAttribute)
	if _, ok := clientVerification.Attributes["verify_crl"]; ok {
		t.Fatal("frontend client certificate verification unexpectedly includes verify_crl")
	}
	backend := resp.Schema.Attributes["backend"].(fwrschema.SingleNestedAttribute)
	serverName := backend.Attributes["server_name"].(fwrschema.StringAttribute)
	if !serverName.Optional || serverName.Required {
		t.Fatal("backend.server_name should be optional")
	}
	verifyCertificate := backend.Attributes["tls_settings"].(fwrschema.SingleNestedAttribute).Attributes["verify_certificate"].(fwrschema.SingleNestedAttribute)
	if _, ok := verifyCertificate.Attributes["verify_crl"]; ok {
		t.Fatal("backend certificate verification unexpectedly includes verify_crl")
	}
	protocol := resp.Schema.Attributes["protocol_settings"].(fwrschema.SingleNestedAttribute)
	if _, ok := protocol.Attributes["multiplexing"]; ok {
		t.Fatal("protocol_settings unexpectedly includes multiplexing")
	}
	for _, attrName := range []string{"connection_reuse_enabled", "bot_protection", "data_protection", "custom_pages", "waf", "rate_limiting", "ip_based_access_control", "traffic_rules"} {
		if _, ok := resp.Schema.Attributes[attrName]; !ok {
			t.Fatalf("HTTP Proxy schema is missing %q", attrName)
		}
	}
}

func TestHTTPProxyResourceFrontendRequiresIPv4OrIPv6(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config:      testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", false),
		PlanOnly:    true,
		ExpectError: regexp.MustCompile(`Missing Frontend IP|At least one of|frontend\.ipv4`),
	}}})
}

func TestHTTPProxyResourceRejectsPlaintextHTTP2AndSecureOnlyFields(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	config := testProviderConfig(server.URL) + `
resource "baffinbay_http_proxy" "test" {
  name = "test-http"
  frontend = {
    connection_type = "PLAINTEXT"
    ipv4 = "192.0.2.10"
    port = 80
    redirect_http = false
    hosts = [{ host = "example.com" }]
  }
  backend = { hosts = [{ address = "origin.example.com", port = 80 }] }
  protocol_settings = { version = "HTTP2.0" }
}
`
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(`HTTP/2 requires|redirect_http is only valid`)}}})
}

func TestHTTPProxyResourceCreateDefaultsMatchPortal(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", true),
		Check: tfresource.ComposeTestCheckFunc(
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "connection_reuse_enabled", "true"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "backend.delivery_method", "LEAST_CONNECTIONS"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "bot_protection.strategy", "AUTO"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "bot_protection.challenge_type", "JS"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "waf.enforcement", "LOG"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "waf.core_rule_set_version", "4.*.*"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "rate_limiting.by_source_ip.rate.value", "10"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "rate_limiting.by_source_ip.burst", "100"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "ip_based_access_control.default_policy", "ALLOW"),
		),
	}}})

	payload := server.lastMutation(t)
	attrs := payload.Data.Attributes
	if attrs.Version != httpProxyTrafficConfigVersion || attrs.ConnectionReuseEnabled == nil || !*attrs.ConnectionReuseEnabled {
		t.Fatalf("unexpected create defaults: version=%q connectionReuse=%v", attrs.Version, attrs.ConnectionReuseEnabled)
	}
	if attrs.WAF == nil || attrs.WAF.Enforcement != "LOG" || attrs.WAF.CoreRuleSet.Version != "4.*.*" {
		t.Fatalf("unexpected WAF defaults: %#v", attrs.WAF)
	}
	if attrs.RateLimiting == nil || attrs.RateLimiting.BySourceIP.Rate.Value != 10 || attrs.RateLimiting.BySourceIP.Burst != 100 {
		t.Fatalf("unexpected rate limiting defaults: %#v", attrs.RateLimiting)
	}
}

func TestHTTPProxyResourceRejectsInvalidCRSVersionSelectors(t *testing.T) {
	for _, testCase := range []struct {
		name string
		from string
		to   string
	}{
		{name: "missing component", from: `core_rule_set_version = "4.*.*"`, to: `core_rule_set_version = "4.*"`},
		{name: "major wildcard", from: `core_rule_set_version = "5.*.*"`, to: `core_rule_set_version = "*.5.0"`},
		{name: "non-trailing wildcard", from: `core_rule_set_version = "4.*.*"`, to: `core_rule_set_version = "4.*.1"`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newHTTPProxyStateServer(t)
			defer server.Close()

			config := strings.Replace(httpProxyFullResource("test-http"), testCase.from, testCase.to, 1)
			if config == httpProxyFullResource("test-http") {
				t.Fatalf("test fixture does not contain %q", testCase.from)
			}

			tfresource.UnitTest(t, tfresource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []tfresource.TestStep{{
					Config:      testProviderConfig(server.URL) + config,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`three-part CRS version selector|core_rule_set_version`),
				}},
			})
		})
	}
}

func TestHTTPProxyResourceLifecycleFullPortalShape(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testProviderConfig(server.URL) + httpProxyFullResource("test-http"),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "id", "traffic-config-id"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "backend.delivery_method", "IP_HASH"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "custom_pages.0.type", "WAF_BLOCK"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "ip_based_access_control.rules.known_services.0.bypass_bot_protection", "true"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "waf.staged_waf.mode", "CRS_VERSION"),
					tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "traffic_rules.0.actions.redirect.status_code", "302"),
				),
			},
			{ResourceName: "baffinbay_http_proxy.test", ImportState: true, ImportStateVerify: true},
			{
				Config:           testProviderConfig(server.URL) + httpProxyFullResource("test-http-updated"),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("baffinbay_http_proxy.test", plancheck.ResourceActionUpdate)}},
				Check:            tfresource.TestCheckResourceAttr("baffinbay_http_proxy.test", "name", "test-http-updated"),
			},
		},
	})

	if server.putCount.Load() == 0 || server.deleteCount.Load() == 0 {
		t.Fatalf("expected update and delete calls, put=%d delete=%d", server.putCount.Load(), server.deleteCount.Load())
	}
	payload := server.lastMutation(t)
	assertFullHTTPProxyPayload(t, payload)
}

func TestHTTPProxyResourceTrafficRuleOptionalListsRemainNull(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	config := testProviderConfig(server.URL) + `
resource "baffinbay_http_proxy" "traffic_rules" {
  name = "traffic-rules-http"
  frontend = {
    connection_type = "PLAINTEXT"
    ipv4 = "192.0.2.11"
    port = 80
    hosts = [{ host = "rules.example.com" }]
  }
  backend = { hosts = [{ address = "origin.example.com", port = 80 }] }
  protocol_settings = { version = "HTTP1.1" }
  traffic_rules = [
    {
      name = "redirect"
      matching_conditions = [{ paths = ["/old/{*}"], hosts = { type = "ALL" } }]
      actions = {
        redirect = { url = "https://rules.example.com/new", status_code = 301, append_original_path = true }
      }
    },
    {
      name = "backend"
      matching_conditions = [{ paths = ["/api/{*}"], hosts = { type = "ALL" } }]
      actions = {
        backends = [{ address = "api-origin.example.com", port = 80 }]
      }
    },
    {
      name = "header"
      matching_conditions = [{ paths = ["/internal/{*}"], hosts = { type = "SELECT", values = ["rules.example.com"] } }]
      actions = {
        headers = [{ key = "X-Environment", value = "terraform" }]
      }
    }
  ]
}
`
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: config,
		Check: tfresource.ComposeAggregateTestCheckFunc(
			tfresource.TestCheckNoResourceAttr("baffinbay_http_proxy.traffic_rules", "traffic_rules.0.actions.backends"),
			tfresource.TestCheckNoResourceAttr("baffinbay_http_proxy.traffic_rules", "traffic_rules.0.actions.headers"),
			tfresource.TestCheckNoResourceAttr("baffinbay_http_proxy.traffic_rules", "traffic_rules.0.matching_conditions.0.hosts.values"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.traffic_rules", "traffic_rules.1.actions.backends.#", "1"),
			tfresource.TestCheckResourceAttr("baffinbay_http_proxy.traffic_rules", "traffic_rules.2.actions.headers.#", "1"),
		),
	}}})
}

func TestHTTPProxyResourceImportThenUpdatePreservesRemoteFields(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{Config: testProviderConfig(server.URL) + httpProxyFullResource("test-http")},
		{ResourceName: "baffinbay_http_proxy.test", ImportState: true, ImportStateVerify: true},
		{Config: testProviderConfig(server.URL) + httpProxySecureCoreResource("test-http-renamed")},
	}})

	payload := server.lastMutation(t)
	assertFullHTTPProxyPayload(t, payload)
}

func TestHTTPProxyResourceWrongTypeReadAndImportFail(t *testing.T) {
	for _, testCase := range []struct {
		name, wrongType string
		importState     bool
	}{{"read", l4ProxyTrafficConfigType, false}, {"import", routedDsrTrafficConfigType, true}} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newHTTPProxyStateServer(t)
			defer server.Close()
			steps := []tfresource.TestStep{{Config: testProviderConfig(server.URL) + httpProxyMinimalResource("test-http", true)}}
			if testCase.importState {
				steps = append(steps, tfresource.TestStep{PreConfig: func() { server.remoteType.Store(testCase.wrongType) }, ResourceName: "baffinbay_http_proxy.test", ImportState: true, ImportStateId: "traffic-config-id", ExpectError: regexp.MustCompile(`Traffic Config Type Mismatch|expected "httpProxy"`)})
			} else {
				steps = append(steps, tfresource.TestStep{PreConfig: func() { server.remoteType.Store(testCase.wrongType) }, RefreshState: true, ExpectError: regexp.MustCompile(`Traffic Config Type Mismatch|expected "httpProxy"`)})
			}
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: steps})
		})
	}
}

func TestHTTPProxyRequestUsesCurrentWireContract(t *testing.T) {
	data := HTTPProxyResourceModel{
		Name: typesString("test-http"), DeploymentState: typesString("UNDEPLOYED"), ConnectionReuse: typesBool(false),
		Frontend:         &HTTPProxyFrontendModel{ConnectionType: typesString("SECURE"), IPv4: typesString("192.0.2.10"), Port: typesInt64(443), RedirectHTTP: typesBool(false), Hosts: []HTTPProxyFrontendHostModel{{Host: typesString("example.com"), CertificateID: typesString(testCertificateID), TLSConfig: typesString("ADVANCED")}}, HSTS: defaultHTTPProxyHSTS(), ClientCertificateVerification: &HTTPProxyClientCertificateVerificationModel{Mode: typesString("VERIFY_AND_REJECT"), CACertificateIDs: []types.String{typesString(testCACertificateID)}}},
		Backend:          &HTTPProxyBackendModel{Hosts: []HTTPProxyBackendHostModel{{Address: typesString("origin.example.com"), Port: typesInt64(443)}}, DeliveryMethod: typesString("ROUND_ROBIN"), TLSSettings: &HTTPProxyTLSSettingsModel{VerifyCertificate: &HTTPProxyVerifyCertificateModel{Mode: typesString("CUSTOM_TRUSTSTORE"), CACertificateIDs: []types.String{typesString(testCACertificateID)}}}},
		ProtocolSettings: &HTTPProxyProtocolSettingsModel{Version: typesString("HTTP1.1"), EnableWebsockets: typesBool(false)},
		BotProtection:    defaultHTTPProxyBotProtection(), DataProtection: defaultHTTPProxyDataProtection(), CustomPages: []HTTPProxyCustomPageModel{}, WAF: defaultHTTPProxyWAF(), RateLimiting: defaultHTTPProxyRateLimiting(), IPBasedAccessControl: defaultHTTPProxyIPAccessControl(), TrafficRules: []HTTPProxyTrafficRuleModel{},
	}
	payload, err := json.Marshal(mapHTTPProxyModelToCreateRequest(data))
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, required := range []string{`"type":"httpProxy"`, `"version":"0.1.0"`, `"caCertificateIds"`, `"connectionReuseEnabled":false`, `"enableWebsockets":false`} {
		if !strings.Contains(text, required) {
			t.Errorf("payload missing %s: %s", required, text)
		}
	}
	for _, forbidden := range []string{"caBundleIds", "verifyCrl", "multiplexing", "geoFencing", "allowedSources", "protocols", "proxyProtocol"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("payload unexpectedly contains %s: %s", forbidden, text)
		}
	}
}

func TestHTTPProxyUpdateMapperOmitsNilManagedSections(t *testing.T) {
	data := HTTPProxyResourceModel{
		Name:            typesString("test-http"),
		DeploymentState: typesString("UNDEPLOYED"),
		Frontend: &HTTPProxyFrontendModel{
			ConnectionType: typesString("PLAINTEXT"),
			IPv4:           typesString("192.0.2.10"),
			Port:           typesInt64(80),
			Hosts:          []HTTPProxyFrontendHostModel{{Host: typesString("example.com")}},
		},
		Backend: &HTTPProxyBackendModel{
			Hosts:          []HTTPProxyBackendHostModel{{Address: typesString("origin.example.com"), Port: typesInt64(80)}},
			DeliveryMethod: typesString("ROUND_ROBIN"),
		},
		ProtocolSettings: &HTTPProxyProtocolSettingsModel{Version: typesString("HTTP1.1"), EnableWebsockets: typesBool(false)},
	}
	payload, err := json.Marshal(mapHTTPProxyModelToUpdateRequest(data))
	if err != nil {
		t.Fatal(err)
	}
	for _, omitted := range []string{"waf", "rateLimiting", "ipBasedAccessControl", "trafficRules", "customPages", "dataProtection", "botProtection", "gatewayPath"} {
		if strings.Contains(string(payload), `"`+omitted+`"`) {
			t.Errorf("update payload unexpectedly includes %s: %s", omitted, payload)
		}
	}
}

func TestHTTPProxyHTTP2PayloadOmitsHTTP1OnlyFields(t *testing.T) {
	data := HTTPProxyResourceModel{
		Name:            typesString("test-http2"),
		DeploymentState: typesString("UNDEPLOYED"),
		Frontend: &HTTPProxyFrontendModel{
			ConnectionType: typesString("SECURE"), IPv4: typesString("192.0.2.10"), Port: typesInt64(443), RedirectHTTP: typesBool(true),
			Hosts: []HTTPProxyFrontendHostModel{{Host: typesString("example.com"), CertificateID: typesString(testCertificateID), TLSConfig: typesString("INTERMEDIATE")}},
			HSTS:  defaultHTTPProxyHSTS(), ClientCertificateVerification: &HTTPProxyClientCertificateVerificationModel{Mode: typesString("DISABLED"), CACertificateIDs: []types.String{}},
		},
		Backend:          &HTTPProxyBackendModel{Hosts: []HTTPProxyBackendHostModel{{Address: typesString("origin.example.com"), Port: typesInt64(443)}}, DeliveryMethod: typesString("LEAST_CONNECTIONS"), TLSSettings: defaultHTTPProxyTLSSettings()},
		ProtocolSettings: &HTTPProxyProtocolSettingsModel{Version: typesString("HTTP2.0"), EnableWebsockets: typesBool(false)},
	}
	request := mapHTTPProxyModelToCreateRequest(data)
	if request.Data.Attributes.ProtocolSettings.EnableWebsockets != nil {
		t.Fatal("HTTP/2 payload unexpectedly includes enableWebsockets")
	}
	if request.Data.Attributes.Backend.TLSSettings.VerifyCertificate.Mode != "SYSTEM_TRUSTSTORE" {
		t.Fatalf("expected default SYSTEM_TRUSTSTORE, got %#v", request.Data.Attributes.Backend.TLSSettings)
	}
}

func TestHTTPProxyResponseFixtureMapsCurrentBackendShape(t *testing.T) {
	fixture := fmt.Sprintf(`{
  "data": {
    "id": "traffic-config-id",
    "type": "httpProxy",
    "attributes": {
      "name": "fixture-http",
      "version": "0.1.0",
      "deployment": {"state": "UNDEPLOYED"},
      "connectionReuseEnabled": false,
      "frontend": {
        "connectionType": "SECURE",
        "ipv4": "192.0.2.10",
        "port": 443,
        "redirectHttp": false,
        "hosts": [{"host": "example.com", "certificateId": %q, "tlsConfig": "ADVANCED"}],
        "httpStrictTransportSecurity": {"enabled": false, "maxAge": 14515200, "includeSubdomains": false, "preload": false},
        "clientCertificateVerification": {"mode": "VERIFY_AND_REJECT", "caCertificateIds": [%q]}
      },
      "backend": {
        "hosts": [{"address": "origin.example.com", "port": 443}],
        "deliveryMethod": "LEAST_CONNECTIONS",
        "tlsSettings": {"verifyCertificate": {"mode": "SYSTEM_TRUSTSTORE"}}
      },
      "protocolSettings": {"httpVersion": "HTTP2.0"},
      "botProtection": {"strategy": "AUTO", "challengeType": "JS"},
      "dataProtection": {"logRedaction": {"headers": [], "cookies": []}},
      "customPages": [],
      "trafficRules": []
    }
  }
}`, testCertificateID, testCACertificateID)
	var response client.HTTPProxyResponse
	if err := json.Unmarshal([]byte(fixture), &response); err != nil {
		t.Fatal(err)
	}
	mapped := mapHTTPProxyResponseToModel(&response, HTTPProxyResourceModel{})
	if mapped.Name.ValueString() != "fixture-http" || mapped.ProtocolSettings.Version.ValueString() != "HTTP2.0" {
		t.Fatalf("unexpected response mapping: %#v", mapped)
	}
	if mapped.Frontend.ClientCertificateVerification.CACertificateIDs[0].ValueString() != testCACertificateID {
		t.Fatalf("CA certificate IDs were not mapped: %#v", mapped.Frontend.ClientCertificateVerification)
	}
	if mapped.ConnectionReuse.IsNull() || mapped.ConnectionReuse.ValueBool() {
		t.Fatal("explicit connectionReuseEnabled=false was not preserved")
	}
}

func TestHTTPProxyTrafficRuleResponsePreservesOptionalListShape(t *testing.T) {
	response := []client.HTTPProxyTrafficRule{{
		Name: "redirect",
		MatchingConditions: []client.HTTPProxyTrafficMatchingCondition{{
			Paths: []string{"/old/{*}"},
			Hosts: client.HTTPProxyTrafficRuleHostMatch{Type: "ALL", Values: []string{}},
		}},
		Actions: client.HTTPProxyTrafficRuleActions{SetBackends: []client.HTTPProxyBackendHost{}, SetHeaders: []client.HTTPProxyHeader{}},
	}}

	omitted := mapHTTPProxyTrafficRulesFromAPI(response, []HTTPProxyTrafficRuleModel{{
		MatchingConditions: []HTTPProxyTrafficMatchingConditionModel{{Hosts: &HTTPProxyTrafficRuleHostMatchModel{}}},
		Actions:            &HTTPProxyTrafficRuleActionsModel{},
	}})
	if omitted[0].Actions.Backends != nil || omitted[0].Actions.Headers != nil || omitted[0].MatchingConditions[0].Hosts.Values != nil {
		t.Fatalf("omitted optional lists should remain nil: %#v", omitted[0])
	}

	explicitEmpty := mapHTTPProxyTrafficRulesFromAPI(response, []HTTPProxyTrafficRuleModel{{
		MatchingConditions: []HTTPProxyTrafficMatchingConditionModel{{Hosts: &HTTPProxyTrafficRuleHostMatchModel{Values: []types.String{}}}},
		Actions:            &HTTPProxyTrafficRuleActionsModel{Backends: []HTTPProxyBackendHostModel{}, Headers: []HTTPProxyHeaderModel{}},
	}})
	if explicitEmpty[0].Actions.Backends == nil || explicitEmpty[0].Actions.Headers == nil || explicitEmpty[0].MatchingConditions[0].Hosts.Values == nil {
		t.Fatalf("explicit empty optional lists should remain non-nil: %#v", explicitEmpty[0])
	}
}

func TestHTTPProxyCrossFieldValidators(t *testing.T) {
	t.Run("block default needs allow rule", func(t *testing.T) {
		resp := &fwresource.ValidateConfigResponse{}
		validateHTTPProxyAccessControl(&HTTPProxyIPAccessControlModel{DefaultPolicy: typesString("BLOCK"), Rules: &HTTPProxyIPAccessControlRulesModel{}}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected access-control validation error")
		}
	})
	t.Run("WAF methods need GET and POST", func(t *testing.T) {
		resp := &fwresource.ValidateConfigResponse{}
		validateHTTPProxyWAF(&HTTPProxyWAFModel{HTTPCompliance: &HTTPProxyHTTPComplianceModel{AllowedMethods: []types.String{typesString("GET")}}}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected WAF HTTP method validation error")
		}
	})
	t.Run("traffic rule needs action", func(t *testing.T) {
		resp := &fwresource.ValidateConfigResponse{}
		validateHTTPProxyTrafficRules([]HTTPProxyTrafficRuleModel{{Name: typesString("rule"), MatchingConditions: []HTTPProxyTrafficMatchingConditionModel{{Paths: []types.String{typesString("/")}, Hosts: &HTTPProxyTrafficRuleHostMatchModel{Type: typesString("ALL")}}}, Actions: &HTTPProxyTrafficRuleActionsModel{}}}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected traffic-rule action validation error")
		}
	})
}

type httpProxyStateServer struct {
	*httptest.Server
	t           *testing.T
	mu          sync.Mutex
	attributes  client.HTTPProxyAttributes
	mutations   []client.HTTPProxyRequest
	remoteType  atomic.Value
	putCount    atomic.Int64
	deleteCount atomic.Int64
}

func newHTTPProxyStateServer(t *testing.T) *httpProxyStateServer {
	t.Helper()
	s := &httpProxyStateServer{t: t}
	s.remoteType.Store(httpProxyTrafficConfigType)
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *httpProxyStateServer) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/oauth/token" {
		writeTokenResponse(w)
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
		s.readMutation(r)
		w.WriteHeader(http.StatusAccepted)
		s.writeResponse(w)
		return
	case r.Method == http.MethodPut && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
		s.putCount.Add(1)
		s.readMutation(r)
		w.WriteHeader(http.StatusAccepted)
		s.writeResponse(w)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
		s.writeResponse(w)
		return
	case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
		s.deleteCount.Add(1)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *httpProxyStateServer) readMutation(r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.t.Errorf("read request: %v", err)
		return
	}
	var request client.HTTPProxyRequest
	if err := json.Unmarshal(body, &request); err != nil {
		s.t.Errorf("decode request %s: %v", body, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes = request.Data.Attributes
	s.mutations = append(s.mutations, request)
}

func (s *httpProxyStateServer) writeResponse(w http.ResponseWriter) {
	s.mu.Lock()
	attributes := s.attributes
	s.mu.Unlock()
	response := client.HTTPProxyResponse{}
	response.Data.ID = "traffic-config-id"
	response.Data.Type = s.remoteType.Load().(string)
	response.Data.Attributes = attributes
	writeJSONAPI(w, mustJSON(s.t, response))
}

func (s *httpProxyStateServer) lastMutation(t *testing.T) client.HTTPProxyRequest {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.mutations) == 0 {
		t.Fatal("expected at least one HTTP Proxy mutation")
	}
	return s.mutations[len(s.mutations)-1]
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertFullHTTPProxyPayload(t *testing.T, payload client.HTTPProxyRequest) {
	t.Helper()
	attrs := payload.Data.Attributes
	if payload.Data.Type != httpProxyTrafficConfigType || attrs.Version != httpProxyTrafficConfigVersion {
		t.Fatalf("wrong type/version: %#v", payload.Data)
	}
	if attrs.Frontend == nil || attrs.Frontend.ClientCertificateVerification == nil || len(attrs.Frontend.ClientCertificateVerification.CACertificateIDs) != 1 {
		t.Fatalf("frontend certificate mapping missing: %#v", attrs.Frontend)
	}
	if attrs.Backend == nil || attrs.Backend.DeliveryMethod != "IP_HASH" || attrs.Backend.TLSSettings == nil {
		t.Fatalf("backend mapping missing: %#v", attrs.Backend)
	}
	if attrs.WAF == nil || attrs.WAF.StagedWAF == nil || len(attrs.WAF.CookieExclusions) != 1 {
		t.Fatalf("WAF mapping missing: %#v", attrs.WAF)
	}
	if attrs.IPBasedAccessControl == nil || len(attrs.IPBasedAccessControl.Rules.KnownServices) != 1 {
		t.Fatalf("access control mapping missing: %#v", attrs.IPBasedAccessControl)
	}
	if attrs.RateLimiting == nil || len(attrs.RateLimiting.Exclusions) != 1 {
		t.Fatalf("rate limiting mapping missing: %#v", attrs.RateLimiting)
	}
	if attrs.TrafficRules == nil || len(*attrs.TrafficRules) != 1 {
		t.Fatalf("traffic rules mapping missing: %#v", attrs.TrafficRules)
	}
	if attrs.DataProtection == nil || len(attrs.DataProtection.LogRedaction.Headers) != 1 {
		t.Fatalf("data protection mapping missing: %#v", attrs.DataProtection)
	}
}

func httpProxyMinimalResource(name string, includeIP bool) string {
	ip := ""
	if includeIP {
		ip = `ipv4 = "192.0.2.10"`
	}
	return fmt.Sprintf(`
resource "baffinbay_http_proxy" "test" {
  name = %q
  frontend = {
    connection_type = "PLAINTEXT"
    %s
    port = 80
    hosts = [{ host = "example.com" }]
  }
  backend = { hosts = [{ address = "origin.example.com", port = 80 }] }
  protocol_settings = { version = "HTTP1.1" }
}
`, name, ip)
}

func httpProxyFullResource(name string) string {
	return fmt.Sprintf(`
resource "baffinbay_http_proxy" "test" {
  name = %q
  deployment_state = "UNDEPLOYED"
  connection_reuse_enabled = false
  frontend = {
    connection_type = "SECURE"
    ipv4 = "192.0.2.10"
    ipv6 = "2001:db8::10"
    port = 443
    redirect_http = false
    hosts = [{ host = "example.com", certificate_id = %q, tls_config = "ADVANCED" }]
    hsts = { enabled = true, max_age = 31536000, include_subdomains = true, preload = true }
    client_certificate_verification = { mode = "VERIFY_AND_REJECT", ca_certificate_ids = [%q] }
  }
  backend = {
    hosts = [{ address = "origin.example.com", port = 8443 }]
    delivery_method = "IP_HASH"
    server_name = "origin.example.com"
    tls_settings = {
      client_certificate_id = %q
      verify_certificate = { mode = "CUSTOM_TRUSTSTORE", ca_certificate_ids = [%q] }
    }
  }
  protocol_settings = { version = "HTTP1.1", enable_websockets = true }
  bot_protection = { strategy = "ALWAYS_ON", challenge_type = "JS" }
  data_protection = { log_redaction = { headers = ["Authorization"], cookies = ["session"] } }
  custom_pages = [{ id = %q, type = "WAF_BLOCK" }]
  rate_limiting = {
    by_source_ip = { enforcement = "BLOCK", rate = { value = 20, unit = "r/s" }, burst = 200 }
    by_source_ip_and_url = { enforcement = "DISABLED", rate = { value = 30, unit = "r/m" }, burst = 300 }
    exclusions = ["192.0.2.0/24"]
  }
  ip_based_access_control = {
    default_policy = "BLOCK"
    rules = {
      ip_ranges = [{ policy = "ALLOW", address = "198.51.100.0/24", note = "office", bypass_bot_protection = true }]
      ip_lists = [{ policy = "BLOCK", id = %q }]
      known_services = [{ policy = "BLOCK", id = "service-feed", note = "feed", bypass_bot_protection = true }]
      geo_locations = [{ policy = "BLOCK", region = "SE", note = "country" }]
      asns = [{ policy = "BLOCK", asn = 64500, note = "network" }]
    }
  }
  waf = {
    enforcement = "BLOCK"
    paranoia_level = 2
    core_rule_set_version = "4.*.*"
    matched_data_enabled = true
    source_exclusions = { enabled = true, sources = ["203.0.113.0/24"] }
    http_compliance = {
      parameter_limit = { enabled = true, limit = 750 }
      allowed_methods = ["GET", "POST", "PUT"]
      allowed_versions = ["HTTP/1.1", "HTTP/2"]
      resource_configs = [{ matches = ["/api/{*}"], parse_json_enabled = true, parse_xml_enabled = false, parse_multipart_request_enabled = true }]
    }
    path_exclusions = [{ match = "/health", disable_all = false, rule_ids = [%q] }]
    cookie_exclusions = [{ cookie_name = "session", exclude_all_rules = false, rule_ids = [%q] }]
    staged_waf = {
      state = "ENABLED"
      mode = "CRS_VERSION"
      core_rule_set_version = "5.*.*"
      path_exclusions = [{ match = "/preview", disable_all = true, rule_ids = [] }]
      cookie_exclusions = []
    }
  }
  traffic_rules = [{
    name = "api-rule"
    matching_conditions = [{ paths = ["/api/{*}"], hosts = { type = "SELECT", values = ["example.com"] } }]
    actions = {
      backends = [{ address = "special-origin.example.com", port = 9443 }]
      headers = [{ key = "X-Proxy", value = "baffinbay" }]
      host_header = "special-origin.example.com"
      redirect = { url = "https://example.com/maintenance", status_code = 302, append_original_path = false }
      rate_limit = {
        by_source_ip = { enforcement = "BLOCK", rate = { value = 10, unit = "r/s" }, burst = 100 }
        by_source_ip_and_url = { enforcement = "BLOCK", rate = { value = 10, unit = "r/s" }, burst = 100 }
      }
      max_body_size = { enforcement = "ENABLED", value_bytes = 1048576 }
    }
  }]
}
`, name, testCertificateID, testCACertificateID, testClientCertificateID, testCACertificateID, testCustomPageID, testIPListID, testWAFRuleID, testWAFRuleID)
}

func httpProxySecureCoreResource(name string) string {
	return fmt.Sprintf(`
resource "baffinbay_http_proxy" "test" {
  name = %q
  frontend = {
    connection_type = "SECURE"
    ipv4 = "192.0.2.10"
    ipv6 = "2001:db8::10"
    port = 443
    redirect_http = false
    hosts = [{ host = "example.com", certificate_id = %q, tls_config = "ADVANCED" }]
    hsts = { enabled = true, max_age = 31536000, include_subdomains = true, preload = true }
    client_certificate_verification = { mode = "VERIFY_AND_REJECT", ca_certificate_ids = [%q] }
  }
  backend = {
    hosts = [{ address = "origin.example.com", port = 8443 }]
    delivery_method = "IP_HASH"
    server_name = "origin.example.com"
    tls_settings = {
      client_certificate_id = %q
      verify_certificate = { mode = "CUSTOM_TRUSTSTORE", ca_certificate_ids = [%q] }
    }
  }
  protocol_settings = { version = "HTTP1.1", enable_websockets = true }
}
`, name, testCertificateID, testCACertificateID, testClientCertificateID, testCACertificateID)
}

func typesString(value string) types.String { return types.StringValue(value) }
func typesBool(value bool) types.Bool       { return types.BoolValue(value) }
func typesInt64(value int64) types.Int64    { return types.Int64Value(value) }
