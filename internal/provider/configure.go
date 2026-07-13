package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type errorDiagnostics interface {
	AddError(summary, detail string)
}

type configuredDataSource struct {
	client *client.Client
}

func (d *configuredDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, "Data Source", &resp.Diagnostics)
}

type configuredResource struct {
	client *client.Client
}

func (r *configuredResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, "Resource", &resp.Diagnostics)
}

func clientFromProviderData(providerData any, target string, diags errorDiagnostics) *client.Client {
	if providerData == nil {
		return nil
	}
	configuredClient, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected "+target+" Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}
	return configuredClient
}
