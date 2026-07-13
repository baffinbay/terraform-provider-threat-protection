package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CACertificateDataSource{}

func NewCACertificateDataSource() datasource.DataSource {
	return &CACertificateDataSource{}
}

type CACertificateDataSource struct {
	client *client.Client
}

type CACertificateDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	Name        types.String `tfsdk:"name"`
	Certificate types.String `tfsdk:"certificate"`
}

func (d *CACertificateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ca_certificate"
}

func (d *CACertificateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay CA Certificate by UUID.", Attributes: map[string]schema.Attribute{
		"id":          singularDataSourceIDAttribute("The UUID of the CA certificate to retrieve."),
		"type":        schema.StringAttribute{Computed: true},
		"name":        schema.StringAttribute{Computed: true},
		"certificate": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "The PEM-encoded CA certificate chain returned by the API."},
	}}
}

func (d *CACertificateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	d.client = configuredClient
}

func (d *CACertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CACertificateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetCaCertificate(ctx, state.ID.ValueString())
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "CA Certificate", state.ID.ValueString(), err)
		return
	}
	state.ID = types.StringValue(response.Data.ID)
	state.Type = types.StringValue(response.Data.Type)
	state.Name = nullableStringValue(response.Data.Attributes.Name)
	state.Certificate = nullableStringValue(caCertificatePEM(response.Data.Attributes.Certificates))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
