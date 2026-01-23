package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderSchema(t *testing.T) {
	p := &BaffinBayProvider{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema generation failed: %s", resp.Diagnostics)
	}

	// Check for attributes
	if _, ok := resp.Schema.Attributes["api_key"]; !ok {
		t.Error("Missing api_key attribute")
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
}
