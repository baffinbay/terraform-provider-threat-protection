package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	lookupIPListID        = "11111111-1111-4111-8111-111111111111"
	lookupCertificateID   = "22222222-2222-4222-8222-222222222222"
	lookupCACertificateID = "33333333-3333-4333-8333-333333333333"
	lookupCustomPageID    = "44444444-4444-4444-8444-444444444444"
	lookupIPSourceID      = "55555555-5555-4555-8555-555555555555"
	lookupTrafficConfigID = "66666666-6666-4666-8666-666666666666"
	lookupHTTPProxyID     = "77777777-7777-4777-8777-777777777777"
	lookupL4ProxyID       = "88888888-8888-4888-8888-888888888888"
	lookupRoutedDsrID     = "99999999-9999-4999-8999-999999999999"
	lookupWrongTypeID     = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
)

func TestProviderIncludesSingularDataSources(t *testing.T) {
	p := &BaffinBayProvider{}
	types := map[string]bool{}
	for _, factory := range p.DataSources(context.Background()) {
		resp := &datasource.MetadataResponse{}
		factory().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		types[resp.TypeName] = true
	}

	for _, typeName := range []string{
		"baffinbay_ip_list",
		"baffinbay_certificate",
		"baffinbay_ca_certificate",
		"baffinbay_custom_page",
		"baffinbay_known_service",
		"baffinbay_ip_source",
		"baffinbay_traffic_config",
		"baffinbay_http_proxy",
		"baffinbay_l4_proxy",
		"baffinbay_routed_dsr",
	} {
		if !types[typeName] {
			t.Errorf("expected singular data source %q to be registered", typeName)
		}
	}
}

func TestSingularDataSourceSchemasRequireOnlyID(t *testing.T) {
	for name, factory := range map[string]func() datasource.DataSource{
		"ip_list":        NewIPListDataSource,
		"ca_certificate": NewCACertificateDataSource,
		"custom_page":    NewCustomPageDataSource,
		"ip_source":      NewIPSourceDataSource,
		"traffic_config": NewTrafficConfigDataSource,
		"http_proxy":     NewHTTPProxyDataSource,
		"l4_proxy":       NewL4ProxyDataSource,
		"routed_dsr":     NewRoutedDsrDataSource,
	} {
		t.Run(name, func(t *testing.T) {
			resp := &datasource.SchemaResponse{}
			factory().Schema(context.Background(), datasource.SchemaRequest{}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("schema errors: %s", resp.Diagnostics)
			}
			id, ok := resp.Schema.Attributes["id"].(schema.StringAttribute)
			if !ok || !id.Required || id.Computed || id.Optional {
				t.Fatalf("id must be a required string attribute, got %#v", resp.Schema.Attributes["id"])
			}
			for attributeName, attribute := range resp.Schema.Attributes {
				if attributeName != "id" && !attribute.IsComputed() {
					t.Errorf("attribute %q should be computed", attributeName)
				}
			}
		})
	}

	for name, factory := range map[string]func() datasource.DataSource{
		"http_proxy": NewHTTPProxyDataSource,
		"l4_proxy":   NewL4ProxyDataSource,
		"routed_dsr": NewRoutedDsrDataSource,
	} {
		resp := &datasource.SchemaResponse{}
		factory().Schema(context.Background(), datasource.SchemaRequest{}, resp)
		if _, exists := resp.Schema.Attributes["type"]; exists {
			t.Errorf("typed %s data source should not expose type", name)
		}
	}
}

func TestSingularDataSourcesReadByID(t *testing.T) {
	setTestTokenCache(t)
	server := newSingularDataSourceServer(t)
	defer server.Close()

	config := testProviderConfig(server.URL) + fmt.Sprintf(`
data "baffinbay_ip_list" "test" { id = %q }
data "baffinbay_certificate" "test" { id = %q }
data "baffinbay_ca_certificate" "test" { id = %q }
data "baffinbay_custom_page" "test" { id = %q }
data "baffinbay_ip_source" "test" { id = %q }
data "baffinbay_traffic_config" "test" { id = %q }
data "baffinbay_http_proxy" "test" { id = %q }
data "baffinbay_l4_proxy" "test" { id = %q }
data "baffinbay_routed_dsr" "test" { id = %q }
`, lookupIPListID, lookupCertificateID, lookupCACertificateID, lookupCustomPageID, lookupIPSourceID, lookupTrafficConfigID, lookupHTTPProxyID, lookupL4ProxyID, lookupRoutedDsrID)

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: config,
		Check: tfresource.ComposeAggregateTestCheckFunc(
			tfresource.TestCheckResourceAttr("data.baffinbay_ip_list.test", "name", "office-networks"),
			tfresource.TestCheckResourceAttr("data.baffinbay_ip_list.test", "entries.0.value", "198.51.100.0/24"),
			tfresource.TestCheckResourceAttr("data.baffinbay_certificate.test", "type", "importedCertificate"),
			tfresource.TestCheckResourceAttr("data.baffinbay_certificate.test", "common_name", "proxy.example.com"),
			tfresource.TestCheckResourceAttr("data.baffinbay_ca_certificate.test", "name", "Example Root CA"),
			tfresource.TestCheckResourceAttr("data.baffinbay_custom_page.test", "content", "<html>blocked</html>"),
			tfresource.TestCheckResourceAttr("data.baffinbay_ip_source.test", "cidr", "185.195.95.0/24"),
			tfresource.TestCheckResourceAttr("data.baffinbay_traffic_config.test", "type", "routedDsr"),
			tfresource.TestCheckResourceAttr("data.baffinbay_traffic_config.test", "prefix", "203.0.113.0/24"),
			tfresource.TestCheckResourceAttr("data.baffinbay_http_proxy.test", "name", "lookup-http"),
			tfresource.TestCheckResourceAttr("data.baffinbay_http_proxy.test", "frontend.ipv4", "192.0.2.10"),
			tfresource.TestCheckResourceAttr("data.baffinbay_l4_proxy.test", "protocols.0", "TCP"),
			tfresource.TestCheckResourceAttr("data.baffinbay_l4_proxy.test", "backend.delivery_method", "ROUND_ROBIN"),
			tfresource.TestCheckResourceAttr("data.baffinbay_routed_dsr.test", "prefix", "2001:db8::/48"),
		),
	}}})
}

func TestSingularTypedTrafficDataSourceRejectsWrongType(t *testing.T) {
	setTestTokenCache(t)
	server := newSingularDataSourceServer(t)
	defer server.Close()

	config := testProviderConfig(server.URL) + fmt.Sprintf(`data "baffinbay_l4_proxy" "wrong" { id = %q }`, lookupWrongTypeID)
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config:      config,
		ExpectError: regexp.MustCompile(`Traffic Config Type Mismatch|expected "l4Proxy"`),
	}}})
}

func TestSingularDataSourceReportsNotFound(t *testing.T) {
	setTestTokenCache(t)
	server := newSingularDataSourceServer(t)
	defer server.Close()

	config := testProviderConfig(server.URL) + `data "baffinbay_ip_list" "missing" { id = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" }`
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config:      config,
		ExpectError: regexp.MustCompile(`IP List Not Found|Unable to find IP List`),
	}}})
}

func newSingularDataSourceServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			writeTokenResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ip-lists/"+lookupIPListID:
			writeJSONAPI(w, fmt.Sprintf(`{"data":{"id":%q,"type":"ipList","attributes":{"name":"office-networks","entries":[{"value":"198.51.100.0/24","note":"office"}]},"relationships":{"usedBy":{"data":[{"id":%q,"type":"httpProxy"}]}},"meta":{"createdAt":"2026-07-01T10:00:00Z","lastUpdatedAt":"2026-07-02T10:00:00Z"}}}`, lookupIPListID, lookupHTTPProxyID))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/certificates":
			writeJSONAPI(w, fmt.Sprintf(`{"data":[{"id":%q,"type":"importedCertificate","attributes":{"commonName":"proxy.example.com"}}]}`, lookupCertificateID))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/"+lookupCACertificateID:
			writeJSONAPI(w, fmt.Sprintf(`{"data":{"id":%q,"type":"caCertificate","attributes":{"name":"Example Root CA","certificates":[{"certificate":"-----BEGIN CERTIFICATE-----\\nTEST\\n-----END CERTIFICATE-----","crlUrl":""}]}}}`, lookupCACertificateID))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages":
			if !strings.Contains(r.URL.RawQuery, "filter%5BtenantId%5D=") && !strings.Contains(r.URL.RawQuery, "filter[tenantId]=") {
				t.Errorf("custom page lookup did not include tenant filter: %s", r.URL.String())
			}
			writeJSONAPI(w, fmt.Sprintf(`{"data":[{"id":%q,"type":"custom-page","attributes":{"name":"blocked.html","createdAt":"2026-07-01T10:00:00Z"}}]}`, lookupCustomPageID))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/"+lookupCustomPageID+"/file":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>blocked</html>"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/tpc-ip-sources":
			writeJSONAPI(w, fmt.Sprintf(`{"data":[{"id":%q,"type":"ipv4Source","attributes":{"cidr":"185.195.95.0/24"}}]}`, lookupIPSourceID))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/traffic-mgmt/traffic-configs/"):
			id := strings.TrimPrefix(r.URL.Path, "/api/v2/traffic-mgmt/traffic-configs/")
			response, ok := singularTrafficConfigResponse(id)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSONAPI(w, response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func singularTrafficConfigResponse(id string) (string, bool) {
	switch id {
	case lookupTrafficConfigID:
		return fmt.Sprintf(`{"data":{"id":%q,"type":"routedDsr","attributes":{"name":"lookup-generic","prefix":"203.0.113.0/24","announced":false,"deployment":{"state":"UNDEPLOYED"}}}}`, id), true
	case lookupHTTPProxyID:
		return fmt.Sprintf(`{"data":{"id":%q,"type":"httpProxy","attributes":{"name":"lookup-http","version":"0.1.0","deployment":{"state":"UNDEPLOYED"},"connectionReuseEnabled":true,"frontend":{"connectionType":"PLAINTEXT","ipv4":"192.0.2.10","port":80,"hosts":[{"host":"proxy.example.com"}]},"backend":{"hosts":[{"address":"origin.example.com","port":80}],"deliveryMethod":"ROUND_ROBIN","serverName":"origin.example.com"},"protocolSettings":{"httpVersion":"HTTP1.1","enableWebsockets":false},"trafficRules":[]}}}`, id), true
	case lookupL4ProxyID:
		return fmt.Sprintf(`{"data":{"id":%q,"type":"l4Proxy","attributes":{"name":"lookup-l4","version":"0.0.1","deployment":{"state":"UNDEPLOYED"},"frontend":{"ipv4":"192.0.2.20","port":9000},"backend":{"hosts":[{"address":"origin.example.com","port":9000}],"deliveryMethod":"ROUND_ROBIN","serverName":"origin.example.com"},"protocols":["TCP","UDP"],"proxyProtocol":"DISABLED"}}}`, id), true
	case lookupRoutedDsrID:
		return fmt.Sprintf(`{"data":{"id":%q,"type":"routedDsr","attributes":{"name":"lookup-routed","prefix":"2001:db8::/48","announced":true,"deployment":{"state":"DEPLOYED"}}}}`, id), true
	case lookupWrongTypeID:
		return fmt.Sprintf(`{"data":{"id":%q,"type":"routedDsr","attributes":{"name":"wrong-type","prefix":"203.0.113.0/24","announced":false,"deployment":{"state":"UNDEPLOYED"}}}}`, id), true
	default:
		return "", false
	}
}
