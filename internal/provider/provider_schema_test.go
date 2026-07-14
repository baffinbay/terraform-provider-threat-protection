package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderMetadata(t *testing.T) {
	p := New("1.2.3")().(*BaffinBayProvider)
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "baffinbay" {
		t.Fatalf("expected provider type baffinbay, got %q", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("expected provider version 1.2.3, got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	p := &BaffinBayProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema generation failed: %s", resp.Diagnostics)
	}

	if _, ok := resp.Schema.Attributes["api_url"]; !ok {
		t.Error("Missing api_url attribute")
	}
	if _, ok := resp.Schema.Attributes["client_id"]; !ok {
		t.Error("Missing client_id attribute")
	}
	if _, ok := resp.Schema.Attributes["client_secret"]; !ok {
		t.Error("Missing client_secret attribute")
	}
	if _, ok := resp.Schema.Attributes["oidc_url"]; !ok {
		t.Error("Missing oidc_url attribute")
	}
	if _, ok := resp.Schema.Attributes["tenant_id"]; !ok {
		t.Error("Missing tenant_id attribute")
	}
	if _, ok := resp.Schema.Attributes["api_key"]; ok {
		t.Error("Unexpected api_key attribute — should have been removed")
	}
}
