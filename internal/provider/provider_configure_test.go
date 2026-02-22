package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderConfigure_OIDC(t *testing.T) {
	// Mock OIDC Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "mock-token", "token_type": "Bearer", "expires_in": 3600}`))
	}))
	defer server.Close()

	p := &BaffinBayProvider{}
	ctx := context.Background()
	t.Setenv("BAFFINBAY_TOKEN_TIME_PATH", t.TempDir()+"/.token_time")
	t.Setenv("BAFFINBAY_ENV_PATH", t.TempDir()+"/.env")

	// Prepare configuration
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"api_key":       tftypes.String,
			"api_url":       tftypes.String,
			"client_id":     tftypes.String,
			"client_secret": tftypes.String,
			"oidc_url":      tftypes.String,
			"account_id":    tftypes.String,
		},
	}
	configBody := tftypes.NewValue(objType, map[string]tftypes.Value{
		"api_key":       tftypes.NewValue(tftypes.String, nil),
		"api_url":       tftypes.NewValue(tftypes.String, "https://api.example.com"),
		"client_id":     tftypes.NewValue(tftypes.String, "test-client-id"),
		"client_secret": tftypes.NewValue(tftypes.String, "test-client-secret"),
		"oidc_url":      tftypes.NewValue(tftypes.String, server.URL),
		"account_id":    tftypes.NewValue(tftypes.String, "test-account-id"),
	})

	req := provider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw: configBody,
			Schema: schema.Schema{
				Attributes: map[string]schema.Attribute{
					"api_key":       schema.StringAttribute{Optional: true},
					"api_url":       schema.StringAttribute{Optional: true},
					"client_id":     schema.StringAttribute{Optional: true},
					"client_secret": schema.StringAttribute{Optional: true},
					"oidc_url":      schema.StringAttribute{Optional: true},
					"account_id":    schema.StringAttribute{Optional: true},
				},
			},
		},
	}
	resp := &provider.ConfigureResponse{}

	p.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %s", resp.Diagnostics)
	}
}
