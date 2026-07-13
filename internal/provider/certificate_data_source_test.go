package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCertificateDataSourceReadsByIDAndCommonName(t *testing.T) {
	setTestTokenCache(t)
	server := newCertificateDataSourceServer(t, "single")
	defer server.Close()

	config := testProviderConfig(server.URL) + fmt.Sprintf(`
data "baffinbay_certificate" "by_id" {
  id = %q
}

data "baffinbay_certificate" "by_common_name" {
  common_name = "proxy.example.com"
}
`, lookupCertificateID)
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: config,
		Check: tfresource.ComposeAggregateTestCheckFunc(
			tfresource.TestCheckResourceAttr("data.baffinbay_certificate.by_id", "common_name", "proxy.example.com"),
			tfresource.TestCheckResourceAttr("data.baffinbay_certificate.by_id", "type", "importedCertificate"),
			tfresource.TestCheckResourceAttr("data.baffinbay_certificate.by_common_name", "id", lookupCertificateID),
		),
	}}})
}

func TestCertificateDataSourceRequiresExactlyOneSelector(t *testing.T) {
	for _, testCase := range []struct {
		name, body string
	}{
		{name: "neither", body: ""},
		{name: "both", body: fmt.Sprintf("id = %q\ncommon_name = \"proxy.example.com\"", lookupCertificateID)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newCertificateDataSourceServer(t, "single")
			defer server.Close()
			config := testProviderConfig(server.URL) + fmt.Sprintf("data \"baffinbay_certificate\" \"test\" {\n%s\n}", testCase.body)
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)exactly one|id.*common_name|common_name.*id`),
			}}})
		})
	}
}

func TestCertificateDataSourceCommonNameLookupErrors(t *testing.T) {
	for _, testCase := range []struct {
		name, mode, expected string
	}{
		{name: "missing", mode: "missing", expected: `Certificate Not Found|exact common name`},
		{name: "duplicate", mode: "duplicate", expected: `Ambiguous Certificate Common Name|Found 2`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newCertificateDataSourceServer(t, testCase.mode)
			defer server.Close()
			config := testProviderConfig(server.URL) + `data "baffinbay_certificate" "test" { common_name = "proxy.example.com" }`
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				ExpectError: regexp.MustCompile(testCase.expected),
			}}})
		})
	}
}

func newCertificateDataSourceServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	certificate := fmt.Sprintf(`{"id":%q,"type":"importedCertificate","attributes":{"commonName":"proxy.example.com"}}`, lookupCertificateID)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			writeTokenResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/certificates":
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
