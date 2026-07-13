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

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwrschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const ipListTestID = "11111111-1111-1111-1111-111111111111"

func TestProviderIncludesIPListResourceAndDataSource(t *testing.T) {
	provider := &BaffinBayProvider{}
	resourceTypes := map[string]bool{}
	for _, newResource := range provider.Resources(context.Background()) {
		resource := newResource()
		resp := &fwresource.MetadataResponse{}
		resource.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		resourceTypes[resp.TypeName] = true
	}
	if !resourceTypes["baffinbay_ip_list"] {
		t.Fatal("expected baffinbay_ip_list to be registered")
	}

	dataSourceTypes := map[string]bool{}
	for _, newDataSource := range provider.DataSources(context.Background()) {
		dataSource := newDataSource()
		resp := &datasource.MetadataResponse{}
		dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		dataSourceTypes[resp.TypeName] = true
	}
	if !dataSourceTypes["baffinbay_ip_lists"] {
		t.Fatal("expected baffinbay_ip_lists to be registered")
	}
}

func TestIPListResourceSchema(t *testing.T) {
	resource := NewIPListResource()
	resp := &fwresource.SchemaResponse{}
	resource.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema generation failed: %s", resp.Diagnostics)
	}

	for _, unsupported := range []string{"type", "tenant_id", "used_by", "created_at", "last_updated_at"} {
		if _, exists := resp.Schema.Attributes[unsupported]; exists {
			t.Fatalf("IP list resource schema unexpectedly included %q", unsupported)
		}
	}

	id, ok := resp.Schema.Attributes["id"].(fwrschema.StringAttribute)
	if !ok || !id.Computed {
		t.Fatalf("id attribute has type %T and computed=%t, want computed string", resp.Schema.Attributes["id"], ok && id.Computed)
	}
	name, ok := resp.Schema.Attributes["name"].(fwrschema.StringAttribute)
	if !ok || !name.Required {
		t.Fatalf("name attribute has type %T and required=%t, want required string", resp.Schema.Attributes["name"], ok && name.Required)
	}
	entries, ok := resp.Schema.Attributes["entries"].(fwrschema.ListNestedAttribute)
	if !ok || !entries.Required {
		t.Fatalf("entries attribute has type %T and required=%t, want required nested list", resp.Schema.Attributes["entries"], ok && entries.Required)
	}
	value, ok := entries.NestedObject.Attributes["value"].(fwrschema.StringAttribute)
	if !ok || !value.Required {
		t.Fatal("entries.value should be required")
	}
	note, ok := entries.NestedObject.Attributes["note"].(fwrschema.StringAttribute)
	if !ok || !note.Optional {
		t.Fatal("entries.note should be optional")
	}
}

func TestIPListResourceLifecycleUpdateImportAndDelete(t *testing.T) {
	setTestTokenCache(t)

	initialNote := "office"
	var current atomic.Value
	current.Store(client.IPListAttributes{
		Name: "employees",
		Entries: []client.IPListEntry{
			{Value: "198.51.100.0/24", Note: &initialNote},
			{Value: "2001:db8::/32"},
		},
	})
	var putCount atomic.Int64
	var deleteCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}

		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists":
			request := readIPListRequest(t, req, "")
			current.Store(request.Data.Attributes)
			writeIPListResponse(w, http.StatusCreated, client.IPListType, request.Data.Attributes)
		case req.Method == http.MethodGet && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists/"+ipListTestID:
			writeIPListResponse(w, http.StatusOK, client.IPListType, current.Load().(client.IPListAttributes))
		case req.Method == http.MethodPut && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists/"+ipListTestID:
			putCount.Add(1)
			request := readIPListRequest(t, req, ipListTestID)
			current.Store(request.Data.Attributes)
			writeIPListResponse(w, http.StatusCreated, client.IPListType, request.Data.Attributes)
		case req.Method == http.MethodDelete && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists/"+ipListTestID:
			deleteCount.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: ipListResourceConfig(server.URL, "employees", `
					{ value = "198.51.100.0/24", note = "office" },
					{ value = "2001:db8::/32" },
				`),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "id", ipListTestID),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "name", "employees"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.#", "2"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.0.value", "198.51.100.0/24"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.0.note", "office"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.1.value", "2001:db8::/32"),
				),
			},
			{
				ResourceName:      "baffinbay_ip_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: ipListResourceConfig(server.URL, "employees-and-contractors", `
					{ value = "203.0.113.10/32", note = "vpn" },
				`),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "name", "employees-and-contractors"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.#", "1"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.0.value", "203.0.113.10/32"),
					tfresource.TestCheckResourceAttr("baffinbay_ip_list.test", "entries.0.note", "vpn"),
				),
			},
		},
	})

	if putCount.Load() == 0 {
		t.Fatal("expected IP list update to call PUT")
	}
	if deleteCount.Load() == 0 {
		t.Fatal("expected IP list destroy to call DELETE")
	}
}

func TestIPListResourceRejectsInvalidEntries(t *testing.T) {
	setTestTokenCache(t)

	tests := []struct {
		name        string
		entries     string
		expectError string
	}{
		{
			name:        "bare IP",
			entries:     `{ value = "198.51.100.1" }`,
			expectError: `Invalid IP List Entry|CIDR notation`,
		},
		{
			name:        "non-canonical CIDR",
			entries:     `{ value = "198.51.100.1/24" }`,
			expectError: `Non-canonical IP List Entry|198.51.100.0/24`,
		},
		{
			name:        "duplicate CIDR",
			entries:     `{ value = "198.51.100.0/24" }, { value = "198.51.100.0/24", note = "duplicate" }`,
			expectError: `Duplicate IP List Entry|each prefix must be unique`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tfresource.UnitTest(t, tfresource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []tfresource.TestStep{
					{
						Config:      ipListResourceConfig("https://example.com", "test-list", test.entries),
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(test.expectError),
					},
				},
			})
		})
	}
}

func TestIPListResourceReadWrongTypeFails(t *testing.T) {
	setTestTokenCache(t)

	var remoteType atomic.Value
	remoteType.Store(client.IPListType)
	attributes := client.IPListAttributes{Name: "employees", Entries: []client.IPListEntry{}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth/token" {
			writeTokenResponse(w)
			return
		}
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists":
			writeIPListResponse(w, http.StatusCreated, client.IPListType, attributes)
		case req.Method == http.MethodGet && req.URL.Path == "/api/v2/traffic-mgmt/ip-lists/"+ipListTestID:
			writeIPListResponse(w, http.StatusOK, remoteType.Load().(string), attributes)
		case req.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: ipListResourceConfig(server.URL, "employees", ""),
			},
			{
				PreConfig: func() {
					remoteType.Store("httpProxy")
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`has type "httpProxy"`),
			},
		},
	})
}

func readIPListRequest(t *testing.T, req *http.Request, expectedID string) client.IPListRequest {
	t.Helper()

	if got := req.Header.Get("Content-Type"); got != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got)
	}

	var request client.IPListRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		t.Fatalf("decode IP list request: %v", err)
	}
	if request.Data.ID != expectedID {
		t.Errorf("data.id = %q, want %q", request.Data.ID, expectedID)
	}
	if request.Data.Type != client.IPListType {
		t.Errorf("data.type = %q, want %q", request.Data.Type, client.IPListType)
	}
	if request.Data.Relationships.BelongsTo.Data.Type != "tenant" {
		t.Errorf("belongsTo type = %q, want tenant", request.Data.Relationships.BelongsTo.Data.Type)
	}
	if request.Data.Relationships.BelongsTo.Data.ID != "test-tenant-id" {
		t.Errorf("belongsTo id = %q, want test-tenant-id", request.Data.Relationships.BelongsTo.Data.ID)
	}

	return request
}

func writeIPListResponse(w http.ResponseWriter, status int, resourceType string, attributes client.IPListAttributes) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(client.IPListResponse{Data: client.IPListData{
		ID:         ipListTestID,
		Type:       resourceType,
		Attributes: attributes,
	}})
}

func ipListResourceConfig(apiURL, name, entries string) string {
	return fmt.Sprintf(`
		provider "baffinbay" {
			client_id     = "test-id"
			client_secret = "test-secret"
			api_url       = %q
			oidc_url      = %q
			tenant_id     = "test-tenant-id"
		}

		resource "baffinbay_ip_list" "test" {
			name = %q
			entries = [%s]
		}
	`, apiURL, apiURL+"/oauth/token", name, entries)
}
