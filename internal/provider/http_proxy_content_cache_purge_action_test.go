package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const purgeTestProxyID = "88888888-8888-4888-8888-888888888888"

func TestProviderActionsIncludeContentCachePurge(t *testing.T) {
	p := &BaffinBayProvider{}
	var names []string
	for _, newAction := range p.Actions(context.Background()) {
		resp := &action.MetadataResponse{}
		newAction().Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "baffinbay"}, resp)
		names = append(names, resp.TypeName)
	}
	if len(names) != 1 || names[0] != "baffinbay_http_proxy_content_cache_purge" {
		t.Fatalf("unexpected actions: %v", names)
	}
}

func TestContentCachePurgeActionSchema(t *testing.T) {
	resp := &action.SchemaResponse{}
	NewHTTPProxyContentCachePurgeAction().Schema(context.Background(), action.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %s", resp.Diagnostics)
	}
	schema := resp.Schema
	if len(schema.Attributes) != 1 {
		t.Fatalf("expected only traffic_config_id, got %v", schema.Attributes)
	}
	id, ok := schema.Attributes["traffic_config_id"].(actionschema.StringAttribute)
	if !ok || !id.Required || len(id.Validators) == 0 {
		t.Fatalf("traffic_config_id should be a required, validated string: %#v", schema.Attributes["traffic_config_id"])
	}
}

func TestContentCachePurgeActionRejectsInvalidID(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{{
		Config: testProviderConfig(server.URL) + `
action "baffinbay_http_proxy_content_cache_purge" "purge" {
  config { traffic_config_id = "not-a-uuid" }
}
`,
		PlanOnly:    true,
		ExpectError: regexp.MustCompile(`must be a valid UUID`),
	}}})
}

// purgeServer is a small API mock for calling Invoke directly.
type purgeServer struct {
	*httptest.Server
	purgeStatus   int
	purgeBody     string
	rolloutID     string
	rolloutStates []string
	purges        atomic.Int64
	rolloutReads  atomic.Int64
}

func newPurgeServer(t *testing.T, configure func(*purgeServer)) *purgeServer {
	t.Helper()
	t.Cleanup(client.SetTrafficConfigChangePollIntervalForTesting(time.Millisecond))
	s := &purgeServer{purgeStatus: http.StatusAccepted, rolloutID: "purge-rollout", rolloutStates: []string{"IN_PROGRESS", "COMPLETED"}}
	if configure != nil {
		configure(s)
	}
	purgePath := "/api/v2/traffic-mgmt/traffic-configs/" + purgeTestProxyID + "/content-cache/purge"
	rolloutPath := "/api/v2/traffic-mgmt/traffic-configs/" + purgeTestProxyID + "/rollouts/" + s.rolloutID
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == purgePath:
			s.purges.Add(1)
			if s.purgeStatus != http.StatusAccepted {
				w.WriteHeader(s.purgeStatus)
				_, _ = w.Write([]byte(s.purgeBody))
				return
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusAccepted)
			writeJSONAPI(w, fmt.Sprintf(`{"data":{"id":%q,"type":"httpProxy","attributes":{"name":"p"},"relationships":{"activeRollout":{"data":{"type":"rollout","id":%q}}}}}`, purgeTestProxyID, s.rolloutID))
		case r.Method == http.MethodGet && r.URL.Path == rolloutPath:
			i := int(s.rolloutReads.Add(1)) - 1
			if i >= len(s.rolloutStates) {
				i = len(s.rolloutStates) - 1
			}
			writeTrafficConfigRollout(w, s.rolloutID, s.rolloutStates[i])
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func invokePurge(t *testing.T, serverURL string) (*action.InvokeResponse, []string) {
	t.Helper()
	ctx := context.Background()
	a := NewHTTPProxyContentCachePurgeAction().(action.ActionWithConfigure)
	configureResp := &action.ConfigureResponse{}
	a.Configure(ctx, action.ConfigureRequest{ProviderData: client.NewClient(serverURL)}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("configure: %s", configureResp.Diagnostics)
	}
	schemaResp := &action.SchemaResponse{}
	a.Schema(ctx, action.SchemaRequest{}, schemaResp)
	schema := schemaResp.Schema
	config := tfsdk.Config{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), map[string]tftypes.Value{
		"traffic_config_id": tftypes.NewValue(tftypes.String, purgeTestProxyID),
	})}
	var mu sync.Mutex
	var progress []string
	resp := &action.InvokeResponse{SendProgress: func(event action.InvokeProgressEvent) {
		mu.Lock()
		defer mu.Unlock()
		progress = append(progress, event.Message)
	}}
	a.Invoke(ctx, action.InvokeRequest{Config: config}, resp)
	return resp, progress
}

func TestContentCachePurgeInvokeWaitsForRollout(t *testing.T) {
	server := newPurgeServer(t, nil)
	resp, progress := invokePurge(t, server.URL)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics)
	}
	if server.purges.Load() != 1 || server.rolloutReads.Load() != 2 {
		t.Fatalf("purges=%d rolloutReads=%d, want 1 and 2", server.purges.Load(), server.rolloutReads.Load())
	}
	joined := strings.Join(progress, "\n")
	for _, want := range []string{"purge requested", "purge-rollout", "completed"} {
		if !strings.Contains(joined, want) {
			t.Errorf("progress %q missing %q", progress, want)
		}
	}
}

func TestContentCachePurgeInvokeWithoutRollout(t *testing.T) {
	server := newPurgeServer(t, func(s *purgeServer) { s.rolloutID = "" })
	resp, _ := invokePurge(t, server.URL)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics)
	}
	if server.rolloutReads.Load() != 0 {
		t.Fatalf("expected no rollout polling, got %d", server.rolloutReads.Load())
	}
}

func TestContentCachePurgeInvokeErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*purgeServer)
		want      string
	}{
		{
			name:      "rollout failed",
			configure: func(s *purgeServer) { s.rolloutStates = []string{"FAILED"} },
			want:      `rollout purge-rollout failed`,
		},
		{
			// The backend returns 400 both when the proxy isn't deployed and when the config isn't an HTTP proxy.
			name: "not deployed or wrong type",
			configure: func(s *purgeServer) {
				s.purgeStatus, s.purgeBody = http.StatusBadRequest, `{"errors":[{"detail":"Traffic config is not deployed"}]}`
			},
			want: `(?s)deployed HTTP proxy that is ready for updates.*Traffic config is not deployed`,
		},
		{
			name:      "not found",
			configure: func(s *purgeServer) { s.purgeStatus = http.StatusNotFound },
			want:      `HTTP Proxy "` + purgeTestProxyID + `" was not found`,
		},
		{
			name:      "server error",
			configure: func(s *purgeServer) { s.purgeStatus, s.purgeBody = http.StatusInternalServerError, "boom" },
			want:      `status 500.*boom`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newPurgeServer(t, tc.configure)
			resp, _ := invokePurge(t, server.URL)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected an error diagnostic")
			}
			var details []string
			for _, d := range resp.Diagnostics.Errors() {
				details = append(details, d.Summary()+": "+d.Detail())
			}
			if !regexp.MustCompile(tc.want).MatchString(strings.Join(details, "\n")) {
				t.Fatalf("diagnostics %q do not match %q", details, tc.want)
			}
		})
	}
}

func TestContentCachePurgeInvokeWithoutConfiguredClient(t *testing.T) {
	ctx := context.Background()
	a := NewHTTPProxyContentCachePurgeAction()
	schemaResp := &action.SchemaResponse{}
	a.Schema(ctx, action.SchemaRequest{}, schemaResp)
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
		"traffic_config_id": tftypes.NewValue(tftypes.String, purgeTestProxyID),
	})}
	resp := &action.InvokeResponse{SendProgress: func(action.InvokeProgressEvent) {}}
	a.Invoke(ctx, action.InvokeRequest{Config: config}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the provider is not configured")
	}
}

func TestContentCachePurgeConfigureRejectsWrongProviderData(t *testing.T) {
	a := NewHTTPProxyContentCachePurgeAction().(action.ActionWithConfigure)
	resp := &action.ConfigureResponse{}
	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: "nope"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for unexpected provider data")
	}
}

// End to end through Terraform: an action_trigger on after_update purges once per update.
func TestContentCachePurgeActionTriggerLifecycle(t *testing.T) {
	setTestTokenCache(t)
	t.Cleanup(client.SetTrafficConfigChangePollIntervalForTesting(time.Millisecond))
	server := newHTTPProxyStateServer(t)
	defer server.Close()

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{
			Config: testProviderConfig(server.URL) + httpProxyWithPurgeTrigger("test-http"),
			PostApplyFunc: func() {
				if got := server.purgeCount.Load(); got != 0 {
					t.Errorf("expected no purge on create, got %d", got)
				}
			},
		},
		{
			Config: testProviderConfig(server.URL) + httpProxyWithPurgeTrigger("test-http-renamed"),
			PostApplyFunc: func() {
				if got := server.purgeCount.Load(); got != 1 {
					t.Errorf("expected exactly one purge on update, got %d", got)
				}
				if got := server.purgeRolloutReads.Load(); got == 0 {
					t.Error("expected the purge rollout to be polled")
				}
			},
		},
	}})
}

func TestContentCachePurgeActionTriggerError(t *testing.T) {
	setTestTokenCache(t)
	server := newHTTPProxyStateServer(t)
	defer server.Close()
	server.purgeStatus.Store(http.StatusBadRequest)

	tfresource.UnitTest(t, tfresource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []tfresource.TestStep{
		{Config: testProviderConfig(server.URL) + httpProxyWithPurgeTrigger("test-http")},
		{
			Config:      testProviderConfig(server.URL) + httpProxyWithPurgeTrigger("test-http-renamed"),
			ExpectError: regexp.MustCompile(`deployed HTTP proxy that is ready for updates`),
		},
	}})
}

func httpProxyWithPurgeTrigger(name string) string {
	return `
action "baffinbay_http_proxy_content_cache_purge" "purge" {
  config { traffic_config_id = baffinbay_http_proxy.test.id }
}
` + strings.Replace(httpProxyMinimalResource(name, true),
		`protocol_settings = { version = "HTTP1.1" }`,
		`protocol_settings = { version = "HTTP1.1" }
  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.baffinbay_http_proxy_content_cache_purge.purge]
    }
  }`, 1)
}
