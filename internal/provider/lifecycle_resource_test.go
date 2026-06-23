package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const testAccountID = "test-account-id"

func testProviderConfig(serverURL string) string {
	return fmt.Sprintf(`provider "baffinbay" {
  client_id     = "test-id"
  client_secret = "test-secret"
  api_url       = %[1]q
  oidc_url      = "%[1]s/oauth/token"
  account_id    = %[2]q
}
`, serverURL, testAccountID)
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

func TestTrafficConfigResource_LifecycleUpdateReplaceAndDelete404(t *testing.T) {
	setTestTokenCache(t)

	var currentName atomic.Value
	currentName.Store("test-l4")
	var currentType atomic.Value
	currentType.Store("l4Proxy")
	var putCount atomic.Int64
	var deleteCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			trafficConfigType, trafficConfigName := readTrafficConfigTypeAndNameFromRequest(t, r)
			currentType.Store(trafficConfigType)
			currentName.Store(trafficConfigName)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, trafficConfigResponse(currentType.Load().(string), currentName.Load().(string)))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			writeJSONAPI(w, trafficConfigResponse(currentType.Load().(string), currentName.Load().(string)))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			putCount.Add(1)
			trafficConfigType, trafficConfigName := readTrafficConfigTypeAndNameFromRequest(t, r)
			currentType.Store(trafficConfigType)
			currentName.Store(trafficConfigName)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, trafficConfigResponse(currentType.Load().(string), currentName.Load().(string)))
			return
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			t.Errorf("traffic config update used PATCH; expected PUT")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			deleteCount.Add(1)
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
				Config: trafficConfigL4Config(server.URL, "test-l4"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "id", "traffic-config-id"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "name", "test-l4"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "deployment_state", "UNDEPLOYED"),
				),
			},
			{
				Config: trafficConfigL4Config(server.URL, "test-l4-updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_traffic_config.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "name", "test-l4-updated"),
				),
			},
			{
				Config: trafficConfigRoutedDsrConfig(server.URL),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_traffic_config.test", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})

	if putCount.Load() == 0 {
		t.Fatalf("expected traffic config update to call PUT")
	}
	if deleteCount.Load() == 0 {
		t.Fatalf("expected destroy to call DELETE")
	}
}

func TestTrafficConfigResource_Remote404RemovesState(t *testing.T) {
	setTestTokenCache(t)

	var exists atomic.Bool
	exists.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			exists.Store(true)
			writeJSONAPI(w, trafficConfigL4Response("test-l4"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			if !exists.Load() {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSONAPI(w, trafficConfigL4Response("test-l4"))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
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
				Config: trafficConfigL4Config(server.URL, "test-l4"),
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

func TestTrafficConfigResource_ImportHydratesHTTPProxy(t *testing.T) {
	setTestTokenCache(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			writeJSONAPI(w, trafficConfigHTTPProxyResponse("test-http"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			writeJSONAPI(w, trafficConfigHTTPProxyResponse("test-http"))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
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
				Config: trafficConfigHTTPProxyConfig(server.URL, "test-http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "id", "traffic-config-id"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "type", "httpProxy"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "frontend.redirect_http", "false"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "frontend.hsts.enabled", "false"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "protocol_settings.enable_websockets", "false"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "protocol_settings.multiplexing", "false"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "waf.http_compliance.parameter_limit", "0"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "waf.exclusions.0.type", "path"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "waf.exclusions.0.value", "/health"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "rate_limiting.enforcement", "BLOCK"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "backend.tls_settings.client_certificate_id", "client-cert-id"),
					resource.TestCheckResourceAttr("baffinbay_traffic_config.test", "backend.tls_settings.verify_certificate.verify_crl", "false"),
				),
			},
			{
				ResourceName:      "baffinbay_traffic_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestTrafficConfigResource_ActiveChangePollingCreateUpdateApplied(t *testing.T) {
	setTestTokenCache(t)
	t.Cleanup(client.SetTrafficConfigChangePollIntervalForTesting(time.Millisecond))

	var currentName atomic.Value
	currentName.Store("test-l4")
	var createChangeCalls atomic.Int64
	var updateChangeCalls atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			_, trafficConfigName := readTrafficConfigTypeAndNameFromRequest(t, r)
			currentName.Store(trafficConfigName)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, trafficConfigL4ResponseWithActiveChange(currentName.Load().(string), "create-change"))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			_, trafficConfigName := readTrafficConfigTypeAndNameFromRequest(t, r)
			currentName.Store(trafficConfigName)
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, trafficConfigL4ResponseWithActiveChange(currentName.Load().(string), "update-change"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id/changelog/create-change":
			writeTrafficConfigChange(w, "create-change", deployingThenApplied(&createChangeCalls))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id/changelog/update-change":
			writeTrafficConfigChange(w, "update-change", deployingThenApplied(&updateChangeCalls))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
			writeJSONAPI(w, trafficConfigL4Response(currentName.Load().(string)))
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id":
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
				Config: trafficConfigL4Config(server.URL, "test-l4"),
			},
			{
				Config: trafficConfigL4Config(server.URL, "test-l4-updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("baffinbay_traffic_config.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})

	if createChangeCalls.Load() < 2 {
		t.Fatalf("expected create change polling to continue until APPLIED")
	}
	if updateChangeCalls.Load() < 2 {
		t.Fatalf("expected update change polling to continue until APPLIED")
	}
}

func TestTrafficConfigResource_ActiveChangePollingFailed(t *testing.T) {
	setTestTokenCache(t)
	t.Cleanup(client.SetTrafficConfigChangePollIntervalForTesting(time.Millisecond))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs":
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, trafficConfigL4ResponseWithActiveChange("test-l4", "failed-change"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/traffic-configs/traffic-config-id/changelog/failed-change":
			writeTrafficConfigChange(w, "failed-change", "FAILED")
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      trafficConfigL4Config(server.URL, "test-l4"),
				ExpectError: regexp.MustCompile("failed-change failed"),
			},
		},
	})
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
			if got := r.URL.Query().Get("filter[tenantId]"); got != testAccountID {
				t.Errorf("expected custom page tenant filter %q, got %q", testAccountID, got)
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

func readTrafficConfigTypeAndNameFromRequest(t *testing.T, r *http.Request) (string, string) {
	t.Helper()

	var req struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("failed to decode traffic config request: %v", err)
		return "", ""
	}

	return req.Data.Type, req.Data.Attributes.Name
}

func trafficConfigResponse(trafficConfigType, name string) string {
	switch trafficConfigType {
	case "routedDsr":
		return trafficConfigRoutedDsrResponse(name)
	case "httpProxy":
		return trafficConfigHTTPProxyResponse(name)
	default:
		return trafficConfigL4Response(name)
	}
}

func trafficConfigL4Response(name string) string {
	return trafficConfigL4ResponseWithActiveChange(name, "")
}

func trafficConfigL4ResponseWithActiveChange(name, activeChangeID string) string {
	return trafficConfigJSON("l4Proxy", fmt.Sprintf(`"name":%q,"version":"0.1.0","frontend":{"ipv4":"192.168.1.1","port":80},"backend":{"hosts":[{"address":"example.com","port":8080}],"deliveryMethod":"ROUND_ROBIN","serverName":"example.com"},"deployment":{"state":"UNDEPLOYED"},"protocols":["TCP"]`, name), activeChangeID)
}

func trafficConfigRoutedDsrResponse(name string) string {
	return trafficConfigJSON("routedDsr", fmt.Sprintf(`"name":%q,"prefix":"192.0.2.0/24","announced":false,"deployment":{"state":"UNDEPLOYED"}`, name), "")
}

func trafficConfigHTTPProxyResponse(name string) string {
	return trafficConfigJSON("httpProxy", fmt.Sprintf(`"name":%q,"version":"0.1.0","frontend":{"connectionType":"SECURE","ipv4":"192.168.1.1","port":443,"redirectHttp":false,"hosts":[{"host":"example.com","certificateId":"cert-id","tlsConfig":"ADVANCED"}],"httpStrictTransportSecurity":{"enabled":false,"maxAge":0,"includeSubdomains":false,"preload":false},"clientCertificateVerification":{"mode":"VERIFY_AND_REJECT","verifyCrl":false,"caCertificateIds":["ca-cert-id"]}},"backend":{"hosts":[{"address":"origin.example.com","port":8443}],"deliveryMethod":"ROUND_ROBIN","serverName":"origin.example.com","tlsSettings":{"clientCertificateId":"client-cert-id","verifyCertificate":{"mode":"CUSTOM_TRUSTSTORE","caCertificateIds":["ca-cert-id"],"verifyCrl":false}}},"protocolSettings":{"httpVersion":"HTTP1.1","enableWebsockets":false,"multiplexing":false},"deployment":{"state":"UNDEPLOYED"},"waf":{"enforcement":"LOG","paranoidLevel":1,"coreRuleSetId":"crs-v4.22.0","sourceExclusions":{"enabled":true,"sources":["192.0.2.1/32"]},"httpCompliance":{"globalConfig":{"parameterLimit":{"enabled":true,"limit":0},"allowedHttpMethods":["GET","POST"],"allowedHttpVersions":["HTTP/1.1"]}},"pathExclusions":[{"type":"path","value":"/health","description":"Health check"}]},"rateLimiting":{"enforcement":"BLOCK"}`, name), "")
}

func trafficConfigJSON(trafficConfigType, attributes, activeChangeID string) string {
	relationships := ""
	if activeChangeID != "" {
		relationships = fmt.Sprintf(`,"relationships":{"activeChange":{"data":{"type":"trafficConfigChange","id":%q}}}`, activeChangeID)
	}

	return fmt.Sprintf(`{"data":{"id":"traffic-config-id","type":%q,"attributes":{%s}%s}}`, trafficConfigType, attributes, relationships)
}

func writeTrafficConfigChange(w http.ResponseWriter, id, state string) {
	writeJSONAPI(w, fmt.Sprintf(`{"id":%q,"state":%q}`, id, state))
}

func deployingThenApplied(calls *atomic.Int64) string {
	if calls.Add(1) == 1 {
		return "DEPLOYING"
	}

	return "APPLIED"
}

func trafficConfigL4Config(serverURL, name string) string {
	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_traffic_config" "test" {
  type = "l4Proxy"
  name = %q

  frontend = {
    connection_type = "PLAINTEXT"
    port            = 80
    ipv4            = "192.168.1.1"
    hosts = [
      {
        host = "example.com"
      }
    ]
  }

  backend = {
    hosts = [
      {
        address = "example.com"
        port    = 8080
      }
    ]
    delivery_method = "ROUND_ROBIN"
    server_name     = "example.com"
  }
}
`, name)
}

func trafficConfigRoutedDsrConfig(serverURL string) string {
	return testProviderConfig(serverURL) + `
resource "baffinbay_traffic_config" "test" {
  type      = "routedDsr"
  name      = "test-dsr"
  prefix    = "192.0.2.0/24"
  announced = false
}
`
}

func trafficConfigHTTPProxyConfig(serverURL, name string) string {
	return testProviderConfig(serverURL) + fmt.Sprintf(`
resource "baffinbay_traffic_config" "test" {
  type = "httpProxy"
  name = %q

  frontend = {
    connection_type = "SECURE"
    port            = 443
    ipv4            = "192.168.1.1"
    redirect_http   = false
    hosts = [
      {
        host           = "example.com"
        certificate_id = "cert-id"
        tls_config     = "ADVANCED"
      }
    ]
    hsts = {
      enabled            = false
      max_age            = 0
      include_subdomains = false
      preload            = false
    }
    client_certificate_verification = {
      mode               = "VERIFY_AND_REJECT"
      verify_crl         = false
      ca_certificate_ids = ["ca-cert-id"]
    }
  }

  protocol_settings = {
    version           = "HTTP1.1"
    enable_websockets = false
    multiplexing      = false
  }

  waf = {
    enforcement      = "LOG"
    paranoid_level   = 1
    core_rule_set_id = "crs-v4.22.0"
    source_exclusions = [
      "192.0.2.1/32"
    ]
    http_compliance = {
      allowed_methods  = ["GET", "POST"]
      allowed_versions = ["HTTP/1.1"]
      parameter_limit  = 0
    }
    exclusions = [
      {
        type        = "path"
        value       = "/health"
        description = "Health check"
      }
    ]
  }

  rate_limiting = {
    enforcement = "BLOCK"
  }

  backend = {
    hosts = [
      {
        address = "origin.example.com"
        port    = 8443
      }
    ]
    delivery_method = "ROUND_ROBIN"
    server_name     = "origin.example.com"
    tls_settings = {
      client_certificate_id = "client-cert-id"
      verify_certificate = {
        mode               = "CUSTOM_TRUSTSTORE"
        ca_certificate_ids = ["ca-cert-id"]
        verify_crl         = false
      }
    }
  }
}
`, name)
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
