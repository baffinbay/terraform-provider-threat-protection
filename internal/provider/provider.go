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
	APIURL       types.String `tfsdk:"api_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	OIDCURL      types.String `tfsdk:"oidc_url"`
	TenantID     types.String `tfsdk:"tenant_id"`
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
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The API URL for Baffin Bay Threat Protection API. Defaults to production URL.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The OIDC Client ID for authentication. May also be set via the `BAFFINBAY_CLIENT_ID` environment variable.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The OIDC Client Secret for authentication. May also be set via the `BAFFINBAY_CLIENT_SECRET` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"oidc_url": schema.StringAttribute{
				MarkdownDescription: "The OIDC token endpoint URL. Defaults to production URL.",
				Optional:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID for Baffin Bay Threat Protection. May also be set via the `BAFFINBAY_TENANT_ID` environment variable.",
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

	clientID := data.ClientID.ValueString()
	if data.ClientID.IsNull() {
		clientID = os.Getenv("BAFFINBAY_CLIENT_ID")
	}

	clientSecret := data.ClientSecret.ValueString()
	if data.ClientSecret.IsNull() {
		clientSecret = os.Getenv("BAFFINBAY_CLIENT_SECRET")
	}

	apiURL := "https://portal.baffinbay.com"
	if !data.APIURL.IsNull() {
		apiURL = data.APIURL.ValueString()
	}

	oidcURL := "https://m2m-auth.baffinbay.com/oauth/token"
	if !data.OIDCURL.IsNull() {
		oidcURL = data.OIDCURL.ValueString()
	}

	tenantID := data.TenantID.ValueString()
	if data.TenantID.IsNull() {
		tenantID = os.Getenv("BAFFINBAY_TENANT_ID")
	}

	if clientID == "" || clientSecret == "" {
		resp.Diagnostics.AddError(
			"Missing OIDC Credentials",
			"Both client_id and client_secret must be set (via provider config or BAFFINBAY_CLIENT_ID / BAFFINBAY_CLIENT_SECRET environment variables).",
		)
		return
	}

	if tenantID == "" {
		resp.Diagnostics.AddError(
			"Missing Tenant ID",
			"tenant_id must be set via provider config or the BAFFINBAY_TENANT_ID environment variable.",
		)
		return
	}

	ts, err := client.BuildTokenSource(ctx, client.TokenSourceConfig{
		OIDCURL:      oidcURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		resp.Diagnostics.AddError("Token Source Initialization Failed", err.Error())
		return
	}

	c := client.NewClient(apiURL)
	c.TenantID = tenantID
	c.TokenSource = ts

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *BaffinBayProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPingDataSource,
		NewTrafficConfigsDataSource,
		NewCertificatesDataSource,
		NewCaCertificatesDataSource,
		NewCustomPagesDataSource,
		NewIpSourcesDataSource,
		NewIPListsDataSource,
	}
}

func (p *BaffinBayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCertificateResource,
		NewCaCertificateResource,
		NewCustomPageResource,
		NewIPListResource,
		NewTrafficConfigResource,
		NewHTTPProxyResource,
		NewL4ProxyResource,
		NewRoutedDsrResource,
	}
}
