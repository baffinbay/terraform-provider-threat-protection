package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCustomPageDataSourceReadsByIDAndName(t *testing.T) {
	setTestTokenCache(t)
	server := newCustomPageDataSourceServer(t, "single")
	defer server.Close()

	config := testProviderConfig(server.URL) + fmt.Sprintf(`
data "baffinbay_custom_page" "by_id" {
  id = %q
}

data "baffinbay_custom_page" "by_name" {
  name = "blocked.html"
}
`, lookupCustomPageID)
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: config,
		Check: tfresource.ComposeAggregateTestCheckFunc(
			tfresource.TestCheckResourceAttr("data.baffinbay_custom_page.by_id", "name", "blocked.html"),
			tfresource.TestCheckResourceAttr("data.baffinbay_custom_page.by_id", "content", "<html>blocked</html>"),
			tfresource.TestCheckResourceAttr("data.baffinbay_custom_page.by_name", "id", lookupCustomPageID),
			tfresource.TestCheckResourceAttr("data.baffinbay_custom_page.by_name", "created_at", "2026-07-01T10:00:00Z"),
		),
	}}})
}

func TestCustomPageDataSourceRequiresExactlyOneSelector(t *testing.T) {
	for _, testCase := range []struct {
		name, body string
	}{
		{name: "neither", body: ""},
		{name: "both", body: fmt.Sprintf("id = %q\nname = \"blocked.html\"", lookupCustomPageID)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newCustomPageDataSourceServer(t, "single")
			defer server.Close()
			config := testProviderConfig(server.URL) + fmt.Sprintf("data \"baffinbay_custom_page\" \"test\" {\n%s\n}", testCase.body)
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)exactly one|id.*name|name.*id`),
			}}})
		})
	}
}

func TestCustomPageDataSourceNameLookupErrors(t *testing.T) {
	for _, testCase := range []struct {
		name, mode, expected string
	}{
		{name: "missing", mode: "missing", expected: `Custom Page Not Found|exact name`},
		{name: "duplicate", mode: "duplicate", expected: `Ambiguous Custom Page Name|Found 2`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setTestTokenCache(t)
			server := newCustomPageDataSourceServer(t, testCase.mode)
			defer server.Close()
			config := testProviderConfig(server.URL) + `data "baffinbay_custom_page" "test" { name = "blocked.html" }`
			tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
				Config:      config,
				ExpectError: regexp.MustCompile(testCase.expected),
			}}})
		})
	}
}

func newCustomPageDataSourceServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	page := fmt.Sprintf(`{"id":%q,"type":"custom-page","attributes":{"name":"blocked.html","createdAt":"2026-07-01T10:00:00Z"}}`, lookupCustomPageID)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			writeTokenResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages":
			if !strings.Contains(r.URL.RawQuery, "filter%5BtenantId%5D=") && !strings.Contains(r.URL.RawQuery, "filter[tenantId]=") {
				t.Errorf("custom page lookup did not include tenant filter: %s", r.URL.String())
			}
			switch mode {
			case "missing":
				writeJSONAPI(w, `{"data":[]}`)
			case "duplicate":
				writeJSONAPI(w, `{"data":[`+page+`,`+page+`]}`)
			default:
				writeJSONAPI(w, `{"data":[`+page+`]}`)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/traffic-mgmt/custom-pages/"+lookupCustomPageID+"/file":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>blocked</html>"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
