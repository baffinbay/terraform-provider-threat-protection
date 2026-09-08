package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const testTenantID = "test-tenant-id"

func testProviderConfig(serverURL string) string {
	return fmt.Sprintf(`provider "baffinbay" {
  client_id     = "test-id"
  client_secret = "test-secret"
  api_url       = %[1]q
  oidc_url      = "%[1]s/oauth/token"
  tenant_id    = %[2]q
}
`, serverURL, testTenantID)
}

func setTestTokenCache(t *testing.T) {
	t.Helper()
	t.Setenv("BAFFINBAY_TOKEN_CACHE", t.TempDir()+"/token-cache.json")
}

func writeTokenResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
}

func writeJSONAPI(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	_, _ = w.Write([]byte(body))
}

func TestCertificateResource_ImportSensitiveAndReplacement(t *testing.T) {
	setTestTokenCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/certificates/pem":
			writeJSONAPI(w, `{"data":{"id":"cert-id","type":"importedCertificate","attributes":{"commonName":"example.com"}}}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/certificates":
			writeJSONAPI(w, `{"data":[{"id":"cert-id","type":"importedCertificate","attributes":{"commonName":"example.com"}}]}`)
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/certificates/cert-id":
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: certificateConfig(server.URL, "cert-content", "key-content"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectSensitiveValue("baffinbay_certificate.pem", tfjsonpath.New("certificate")),
						plancheck.ExpectSensitiveValue("baffinbay_certificate.pem", tfjsonpath.New("key")),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_certificate.pem", "id", "cert-id"),
					resource.TestCheckResourceAttr("baffinbay_certificate.pem", "type", "pem"),
				),
			},
			{
				ResourceName:            "baffinbay_certificate.pem",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"certificate", "intermediate", "key"},
			},
			{
				Config: certificateConfig(server.URL, "cert-content-updated", "key-content"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_certificate.pem", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestCaCertificateResource_PreservesConfiguredCertificateWhitespace(t *testing.T) {
	setTestTokenCache(t)

	const configuredCertificate = "ca-content\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates":
			if got := readCaCertificateCreatePayload(t, r); got != configuredCertificate && got != "ca-content-updated" {
				t.Errorf("unexpected CA certificate create payload %q", got)
			}
			writeJSONAPI(w, caCertificateResponse("ca-content"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/ca-cert-id":
			writeJSONAPI(w, caCertificateResponse("ca-content"))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/ca-cert-id":
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: caCertificateConfig(server.URL, configuredCertificate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_ca_certificate.test", "id", "ca-cert-id"),
					resource.TestCheckResourceAttr("baffinbay_ca_certificate.test", "name", "Example CA"),
					resource.TestCheckResourceAttr("baffinbay_ca_certificate.test", "certificate", configuredCertificate),
				),
			},
			{
				Config:   caCertificateConfig(server.URL, configuredCertificate),
				PlanOnly: true,
			},
			{
				Config: caCertificateConfig(server.URL, "ca-content-updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_ca_certificate.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestCaCertificateResource_ImportHydratesCertificate(t *testing.T) {
	setTestTokenCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates":
			writeJSONAPI(w, caCertificateResponse("imported-ca-content"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/ca-cert-id":
			writeJSONAPI(w, caCertificateResponse("imported-ca-content"))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/ca-cert-id":
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: caCertificateConfig(server.URL, "imported-ca-content"),
			},
			{
				ResourceName:      "baffinbay_ca_certificate.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestCustomPageResource_ReadImportReplacementAndDelete404(t *testing.T) {
	setTestTokenCache(t)

	var currentContent atomic.Value
	currentContent.Store("<html>test</html>")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages":
			currentContent.Store(readCustomPageUploadContent(t, r))
			writeJSONAPI(w, `{"data":{"id":"custom-page-id","type":"custom-page","attributes":{"name":"test-page.html"}}}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages":
			if got := r.URL.Query().Get("filter[tenantId]"); got != testTenantID {
				t.Errorf("expected custom page tenant filter %q, got %q", testTenantID, got)
			}
			writeJSONAPI(w, `{"data":[{"id":"custom-page-id","type":"custom-page","attributes":{"name":"test-page.html"}}]}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/custom-page-id/file":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(currentContent.Load().(string)))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/custom-page-id":
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: customPageConfig(server.URL, "<html>test</html>"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_custom_page.test", "id", "custom-page-id"),
					resource.TestCheckResourceAttr("baffinbay_custom_page.test", "name", "test-page.html"),
					resource.TestCheckResourceAttr("baffinbay_custom_page.test", "content", "<html>test</html>"),
				),
			},
			{
				ResourceName:      "baffinbay_custom_page.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: customPageConfig(server.URL, "<html>changed</html>"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_custom_page.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestCustomPageResource_Remote404RemovesState(t *testing.T) {
	setTestTokenCache(t)

	var exists atomic.Bool
	exists.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages":
			exists.Store(true)
			writeJSONAPI(w, `{"data":{"id":"custom-page-id","type":"custom-page","attributes":{"name":"test-page.html"}}}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages":
			if !exists.Load() {
				writeJSONAPI(w, `{"data":[]}`)
				return
			}
			writeJSONAPI(w, `{"data":[{"id":"custom-page-id","type":"custom-page","attributes":{"name":"test-page.html"}}]}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/custom-page-id/file":
			if !exists.Load() {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>test</html>"))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/custom-page-id":
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: customPageConfig(server.URL, "<html>test</html>"),
			},
			{
				PreConfig: func() {
					exists.Store(false)
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func trafficConfigL4Response(name string) string {
	return trafficConfigL4ResponseWithDeployment(name, "UNDEPLOYED")
}

func trafficConfigL4ResponseWithDeployment(name, deploymentState string) string {
	return trafficConfigL4ResponseWithDeploymentAndActiveRollout(name, deploymentState, "")
}

func trafficConfigL4ResponseWithDeploymentAndActiveRollout(name, deploymentState, activeRolloutID string) string {
	if deploymentState == "" {
		deploymentState = "UNDEPLOYED"
	}
	return trafficConfigJSONWithRelationships("l4Proxy", fmt.Sprintf(`"name":%q,"version":%q,"frontend":{"ipv4":"192.168.1.1","port":80},"backend":{"hosts":[{"address":"example.com","port":8080}],"deliveryMethod":"ROUND_ROBIN","serverName":"example.com"},"deployment":{"state":%q},"protocols":["TCP"],"proxyProtocol":"DISABLED"`, name, l4ProxyTrafficConfigVersion, deploymentState), "", activeRolloutID)
}

func trafficConfigHTTPProxyResponse(name string) string {
	return trafficConfigJSON("httpProxy", fmt.Sprintf(`"name":%q,"version":"0.1.0","frontend":{"connectionType":"SECURE","ipv4":"192.168.1.1","port":443,"redirectHttp":false,"hosts":[{"host":"example.com","certificateId":"cert-id","tlsConfig":"ADVANCED"}],"httpStrictTransportSecurity":{"enabled":false,"maxAge":0,"includeSubdomains":false,"preload":false},"clientCertificateVerification":{"mode":"VERIFY_AND_REJECT","verifyCrl":false,"caCertificateIds":["ca-cert-id"]}},"backend":{"hosts":[{"address":"origin.example.com","port":8443}],"deliveryMethod":"ROUND_ROBIN","serverName":"origin.example.com","tlsSettings":{"clientCertificateId":"client-cert-id","verifyCertificate":{"mode":"CUSTOM_TRUSTSTORE","caCertificateIds":["ca-cert-id"],"verifyCrl":false}}},"protocolSettings":{"httpVersion":"HTTP1.1","enableWebsockets":false,"multiplexing":false},"deployment":{"state":"UNDEPLOYED"},"waf":{"enforcement":"LOG","paranoidLevel":1,"coreRuleSetId":"crs-v4.22.0","sourceExclusions":{"enabled":true,"sources":["192.0.2.1/32"]},"httpCompliance":{"globalConfig":{"parameterLimit":{"enabled":true,"limit":0},"allowedHttpMethods":["GET","POST"],"allowedHttpVersions":["HTTP/1.1"]}},"pathExclusions":[{"type":"path","value":"/health","description":"Health check"}]},"rateLimiting":{"enforcement":"BLOCK"}`, name), "")
}

func trafficConfigJSON(trafficConfigType, attributes, activeChangeID string) string {
	return trafficConfigJSONWithRelationships(trafficConfigType, attributes, activeChangeID, "")
}

func trafficConfigJSONWithRelationships(trafficConfigType, attributes, activeChangeID, activeRolloutID string) string {
	relationships := ""
	relationshipEntries := make([]string, 0, 2)
	if activeChangeID != "" {
		relationshipEntries = append(relationshipEntries, fmt.Sprintf(`"activeChange":{"data":{"type":"trafficConfigChange","id":%q}}`, activeChangeID))
	}
	if activeRolloutID != "" {
		relationshipEntries = append(relationshipEntries, fmt.Sprintf(`"activeRollout":{"data":{"type":"rollout","id":%q}}`, activeRolloutID))
	}
	if len(relationshipEntries) > 0 {
		relationships = fmt.Sprintf(`,"relationships":{%s}`, strings.Join(relationshipEntries, ","))
	}

	return fmt.Sprintf(`{"data":{"id":"traffic-config-id","type":%q,"attributes":{%s}%s}}`, trafficConfigType, attributes, relationships)
}

func writeTrafficConfigRollout(w http.ResponseWriter, id, state string) {
	writeJSONAPI(w, fmt.Sprintf(`{"data":{"type":"rollout","id":%q,"attributes":{"state":%q}}}`, id, state))
}

func certificateConfig(serverURL, certificate, key string) string {
	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_certificate" "pem" {
  type        = "pem"
  certificate = %q
  key         = %q
}
`, certificate, key)
}

func caCertificateConfig(serverURL, certificate string) string {
	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_ca_certificate" "test" {
  certificate = %q
}
`, certificate)
}

func caCertificateResponse(certificate string) string {
	return fmt.Sprintf(`{"data":{"id":"ca-cert-id","type":"caCertificate","attributes":{"name":"Example CA","certificates":[{"certificate":%q}]}}}`, certificate)
}

func readCaCertificateCreatePayload(t *testing.T, r *http.Request) string {
	t.Helper()

	var req struct {
		Data struct {
			Attributes struct {
				Name         string `json:"name"`
				Certificates []struct {
					Certificate string `json:"certificate"`
				} `json:"certificates"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("failed to decode CA certificate request: %v", err)
		return ""
	}
	if req.Data.Attributes.Name != "" {
		t.Errorf("CA certificate create payload included unsupported name attribute %q", req.Data.Attributes.Name)
	}
	if len(req.Data.Attributes.Certificates) != 1 || req.Data.Attributes.Certificates[0].Certificate == "" {
		t.Errorf("unexpected CA certificate create certificates payload: %#v", req.Data.Attributes.Certificates)
		return ""
	}

	return req.Data.Attributes.Certificates[0].Certificate
}

func readCustomPageUploadContent(t *testing.T, r *http.Request) string {
	t.Helper()

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Errorf("failed to parse custom page multipart request: %v", err)
		return ""
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		t.Errorf("failed to read custom page file part: %v", err)
		return ""
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("failed to read custom page upload content: %v", err)
		return ""
	}

	return string(content)
}

func customPageConfig(serverURL, content string) string {
	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_custom_page" "test" {
  name    = "test-page.html"
  content = %q
}
`, content)
}
