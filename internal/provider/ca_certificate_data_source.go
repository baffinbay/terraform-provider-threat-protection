package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CACertificateDataSource{}
var _ datasource.DataSourceWithConfigValidators = &CACertificateDataSource{}

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
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay CA Certificate by UUID or exact name.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The CA certificate UUID. Exactly one of `id` or `name` must be configured.",
			Validators: []validator.String{
				stringvalidator.RegexMatches(singularDataSourceUUIDPattern, "must be a valid UUID"),
			},
		},
		"type": schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The exact, case-sensitive CA certificate name. Exactly one of `id` or `name` must be configured.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"certificate": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "The PEM-encoded CA certificate chain returned by the API."},
	}}
}

func (d *CACertificateDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
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
	var config CACertificateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var certificate client.CaCertificateData
	if !config.ID.IsNull() && !config.ID.IsUnknown() {
		response, err := d.client.GetCaCertificate(ctx, config.ID.ValueString())
		if err != nil {
			addSingularDataSourceReadError(&resp.Diagnostics, "CA Certificate", config.ID.ValueString(), err)
			return
		}
		certificate = response.Data
	} else {
		response, err := d.client.GetCaCertificates(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list CA Certificates while looking up name %q, got error: %s", config.Name.ValueString(), err))
			return
		}
		matches := make([]client.CaCertificateData, 0, 1)
		for _, candidate := range response.Data {
			if candidate.Attributes.Name == config.Name.ValueString() {
				matches = append(matches, candidate)
			}
		}
		switch len(matches) {
		case 0:
			resp.Diagnostics.AddError("CA Certificate Not Found", fmt.Sprintf("Unable to find a CA Certificate with exact name %q.", config.Name.ValueString()))
			return
		case 1:
			certificate = matches[0]
		default:
			resp.Diagnostics.AddError("Ambiguous CA Certificate Name", fmt.Sprintf("Found %d CA Certificates with exact name %q. Configure the data source with an ID instead.", len(matches), config.Name.ValueString()))
			return
		}
	}

	state := CACertificateDataSourceModel{
		ID:          types.StringValue(certificate.ID),
		Type:        types.StringValue(certificate.Type),
		Name:        nullableStringValue(certificate.Attributes.Name),
		Certificate: nullableStringValue(caCertificatePEM(certificate.Attributes.Certificates)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
