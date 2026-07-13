package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const lookupKnownServiceID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

func TestKnownServiceDataSourceReadsByIDAndName(t *testing.T) {
	setTestTokenCache(t)
	server := newKnownServiceDataSourceServer(t, "single")
	defer server.Close()

	config := testProviderConfig(server.URL) + fmt.Sprintf(`
data "baffinbay_known_service" "by_id" {
  id = %q
}

data "baffinbay_known_service" "by_name" {
  name = "gh-pages"
}
`, lookupKnownServiceID)
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: config,
		Check: tfresource.ComposeAggregateTestCheckFunc(
			tfresource.TestCheckResourceAttr("data.baffinbay_known_service.by_id", "name", "gh-pages"),
			tfresource.TestCheckResourceAttr("data.baffinbay_known_service.by_id", "upstream_provider", "github"),
			tfresource.TestCheckResourceAttr("data.baffinbay_known_service.by_name", "id", lookupKnownServiceID),
			tfresource.TestCheckResourceAttr("data.baffinbay_known_service.by_name", "tags.#", "2"),
			tfresource.TestCheckResourceAttr("data.baffinbay_known_service.by_name", "tags.0", "cdn"),
		),
	}}})
}

func TestKnownServiceDataSourceRequiresExactlyOneSelector(t *testing.T) {
	for _, testCase := range []struct {
		name, body string
	}{
		{name: "neither", body: ""},
		{name: "both", body: fmt.Sprintf("id = %q\nname = \"gh-pages\"", lookupKnownServiceID)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newKnownServiceDataSourceServer(t, "single")
			defer server.Close()
			config := testProviderConfig(server.URL) + fmt.Sprintf("data \"baffinbay_known_service\" \"test\" {\n%s\n}", testCase.body)
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)exactly one|id.*name|name.*id`),
			}}})
		})
	}
}

func TestKnownServiceDataSourceNameLookupErrors(t *testing.T) {
	for _, testCase := range []struct {
		name, mode, expected string
	}{
		{name: "missing", mode: "missing", expected: `Known Service Not Found|exact name`},
		{name: "duplicate", mode: "duplicate", expected: `Ambiguous Known Service Name|Found 2`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newKnownServiceDataSourceServer(t, testCase.mode)
			defer server.Close()
			config := testProviderConfig(server.URL) + `data "baffinbay_known_service" "test" { name = "gh-pages" }`
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				ExpectError: regexp.MustCompile(testCase.expected),
			}}})
		})
	}
}

func newKnownServiceDataSourceServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	service := fmt.Sprintf(`{"id":%q,"type":"knownService","attributes":{"name":"gh-pages","tags":["cdn","github"],"provider":"github"}}`, lookupKnownServiceID)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			writeTokenResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/known-services/"+lookupKnownServiceID:
			writeJSONAPI(w, `{"data":`+service+`}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/known-services":
			switch mode {
			case "missing":
				writeJSONAPI(w, `{"data":[]}`)
			case "duplicate":
				writeJSONAPI(w, `{"data":[`+service+`,`+service+`]}`)
			default:
				writeJSONAPI(w, `{"data":[`+service+`]}`)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
