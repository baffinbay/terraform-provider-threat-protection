package provider

import (
	"context"
	"os"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &BaffinBayProvider{}

type BaffinBayProvider struct{}

type BaffinBayProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
	APIURL types.String `tfsdk:"api_url"`
}

func New() provider.Provider {
	return &BaffinBayProvider{}
}

func (p *BaffinBayProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "baffinbay"
}

func (p *BaffinBayProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The API Key for Baffin Bay Threat Protection API.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The API URL for Baffin Bay Threat Protection API. Defaults to production URL.",
				Optional:            true,
			},
		},
	}
}

func (p *BaffinBayProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data BaffinBayProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := data.APIKey.ValueString()
	if data.APIKey.IsNull() {
		apiKey = os.Getenv("BAFFINBAY_API_KEY")
	}

	if apiKey == "" {
		resp.Diagnostics.AddError("Missing API Key", "API Key must be configured or set via BAFFINBAY_API_KEY env var.")
		return
	}

	apiUrl := "https://api.baffinbay.com"
	if !data.APIURL.IsNull() {
		apiUrl = data.APIURL.ValueString()
	}

	c := client.NewClient(apiUrl)
	c.SetAuth(apiKey)

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *BaffinBayProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPingDataSource,
	}
}

func (p *BaffinBayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}
