package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CertificateDataSource{}

func NewCertificateDataSource() datasource.DataSource {
	return &CertificateDataSource{}
}

type CertificateDataSource struct {
	client *client.Client
}

type CertificateDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Type       types.String `tfsdk:"type"`
	CommonName types.String `tfsdk:"common_name"`
	FQDN       types.String `tfsdk:"fqdn"`
}

func (d *CertificateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (d *CertificateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay Certificate by UUID.", Attributes: map[string]schema.Attribute{
		"id":          singularDataSourceIDAttribute("The UUID of the certificate to retrieve."),
		"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "The API certificate type."},
		"common_name": schema.StringAttribute{Computed: true},
		"fqdn":        schema.StringAttribute{Computed: true},
	}}
}

func (d *CertificateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state CertificateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificate, err := d.client.FindCertificateByID(ctx, state.ID.ValueString())
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "Certificate", state.ID.ValueString(), err)
		return
	}
	state.ID = types.StringValue(certificate.ID)
	state.Type = types.StringValue(certificate.Type)
	state.CommonName = nullableStringValue(certificate.Attributes.CommonName)
	state.FQDN = nullableStringValue(certificate.Attributes.FQDN)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
