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
	APIKey       types.String `tfsdk:"api_key"`
	APIURL       types.String `tfsdk:"api_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	OIDCURL      types.String `tfsdk:"oidc_url"`
	AccountID    types.String `tfsdk:"account_id"`
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
				MarkdownDescription: "The API Key for Baffin Bay Threat Protection API. Deprecated: Use OIDC authentication instead.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The API URL for Baffin Bay Threat Protection API. Defaults to production URL.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The OIDC Client ID for authentication.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The OIDC Client Secret for authentication.",
				Optional:            true,
				Sensitive:           true,
			},
			"oidc_url": schema.StringAttribute{
				MarkdownDescription: "The OIDC token endpoint URL. Defaults to production URL.",
				Optional:            true,
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "The Account ID for Baffin Bay Threat Protection. Required for some resources.",
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

	// Environment variable fallbacks
	apiKey := data.APIKey.ValueString()
	if data.APIKey.IsNull() {
		apiKey = os.Getenv("BAFFINBAY_API_KEY")
	}

	clientID := data.ClientID.ValueString()
	if data.ClientID.IsNull() {
		clientID = os.Getenv("BAFFINBAY_CLIENT_ID")
	}

	clientSecret := data.ClientSecret.ValueString()
	if data.ClientSecret.IsNull() {
		clientSecret = os.Getenv("BAFFINBAY_CLIENT_SECRET")
	}

	apiUrl := "https://portal.baffinbay.com"
	if !data.APIURL.IsNull() {
		apiUrl = data.APIURL.ValueString()
	}

	oidcUrl := "https://m2m-auth.baffinbay.com/oauth/token"
	if !data.OIDCURL.IsNull() {
		oidcUrl = data.OIDCURL.ValueString()
	}

	accountID := data.AccountID.ValueString()
	if data.AccountID.IsNull() {
		accountID = os.Getenv("BAFFINBAY_ACCOUNT_ID")
	}

	// Determine if we should force a refresh (e.g., in acceptance tests)
	// We force it if TF_ACC is set, UNLESS we already have an API key.
	// Note: We use the local token from .env/cache first if available.
	force := os.Getenv("TF_ACC") == "1" && (apiKey == "" || apiKey == "mock-token")

	c := client.NewClient(apiUrl)
	c.AccountID = accountID

	// If we are in acceptance test mode and have a cached token, let's just use it 
	// without calling Authenticate if it looks like a real token.
	if os.Getenv("TF_ACC") == "1" && apiKey != "" && apiKey != "mock-token" && len(apiKey) > 20 {
		c.SetAuth(apiKey)
		// Try a quick ping to see if it's still good
		if err := c.Ping(); err == nil {
			resp.DataSourceData = c
			resp.ResourceData = c
			return
		}
	}

	// Always try to authenticate if we have OIDC credentials,
	// as Authenticate now handles the caching and key reuse internally.
	// Always try to authenticate if we have OIDC credentials,
	// as Authenticate now handles the caching and key reuse internally.
	if clientID != "" && clientSecret != "" {
		// Pass the existing apiKey (if any) to the client so it can try to use it first
		if apiKey != "" {
			c.SetAuth(apiKey)
		}
		err := c.Authenticate(oidcUrl, clientID, clientSecret, force)
		if err != nil {
			resp.Diagnostics.AddError("Authentication Failed", err.Error())
			return
		}
	} else if apiKey != "" {
		c.SetAuth(apiKey)
	} else {
		resp.Diagnostics.AddError("Missing Credentials", "Either OIDC credentials (client_id and client_secret) or an API Key must be configured.")
		return
	}

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
	}
}

func (p *BaffinBayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCertificateResource,
		NewCaCertificateResource,
		NewCustomPageResource,
		NewTrafficConfigResource,
	}
}
