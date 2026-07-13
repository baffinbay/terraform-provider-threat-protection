package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCACertificateDataSourceReadsByIDAndName(t *testing.T) {
	setTestTokenCache(t)
	server := newCACertificateDataSourceServer(t, "single")
	defer server.Close()

	config := testProviderConfig(server.URL) + fmt.Sprintf(`
data "baffinbay_ca_certificate" "by_id" {
  id = %q
}

data "baffinbay_ca_certificate" "by_name" {
  name = "Example Root CA"
}
`, lookupCACertificateID)
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: config,
		Check: tfresource.ComposeAggregateTestCheckFunc(
			tfresource.TestCheckResourceAttr("data.baffinbay_ca_certificate.by_id", "name", "Example Root CA"),
			tfresource.TestCheckResourceAttr("data.baffinbay_ca_certificate.by_id", "type", "caCertificate"),
			tfresource.TestCheckResourceAttr("data.baffinbay_ca_certificate.by_name", "id", lookupCACertificateID),
		),
	}}})
}

func TestCACertificateDataSourceRequiresExactlyOneSelector(t *testing.T) {
	for _, testCase := range []struct {
		name, body string
	}{
		{name: "neither", body: ""},
		{name: "both", body: fmt.Sprintf("id = %q\nname = \"Example Root CA\"", lookupCACertificateID)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newCACertificateDataSourceServer(t, "single")
			defer server.Close()
			config := testProviderConfig(server.URL) + fmt.Sprintf("data \"baffinbay_ca_certificate\" \"test\" {\n%s\n}", testCase.body)
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)exactly one|id.*name|name.*id`),
			}}})
		})
	}
}

func TestCACertificateDataSourceNameLookupErrors(t *testing.T) {
	for _, testCase := range []struct {
		name, mode, expected string
	}{
		{name: "missing", mode: "missing", expected: `CA Certificate Not Found|exact name`},
		{name: "duplicate", mode: "duplicate", expected: `Ambiguous CA Certificate Name|Found 2`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newCACertificateDataSourceServer(t, testCase.mode)
			defer server.Close()
			config := testProviderConfig(server.URL) + `data "baffinbay_ca_certificate" "test" { name = "Example Root CA" }`
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				ExpectError: regexp.MustCompile(testCase.expected),
			}}})
		})
	}
}

func newCACertificateDataSourceServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	certificate := fmt.Sprintf(`{"id":%q,"type":"caCertificate","attributes":{"name":"Example Root CA","certificates":[{"certificate":"-----BEGIN CERTIFICATE-----\\nTEST\\n-----END CERTIFICATE-----","crlUrl":""}]}}`, lookupCACertificateID)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			writeTokenResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates/"+lookupCACertificateID:
			writeJSONAPI(w, `{"data":`+certificate+`}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/ca-certificates":
			switch mode {
			case "missing":
				writeJSONAPI(w, `{"data":[]}`)
			case "duplicate":
				writeJSONAPI(w, `{"data":[`+certificate+`,`+certificate+`]}`)
			default:
				writeJSONAPI(w, `{"data":[`+certificate+`]}`)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
