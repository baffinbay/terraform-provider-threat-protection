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

var _ datasource.DataSource = &CertificateDataSource{}
var _ datasource.DataSourceWithConfigValidators = &CertificateDataSource{}

func NewCertificateDataSource() datasource.DataSource {
	return &CertificateDataSource{}
}

type CertificateDataSource struct {
	configuredDataSource
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
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay Certificate by UUID or exact common name.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The certificate UUID. Exactly one of `id` or `common_name` must be configured.",
			Validators: []validator.String{
				stringvalidator.RegexMatches(singularDataSourceUUIDPattern, "must be a valid UUID"),
			},
		},
		"type": schema.StringAttribute{Computed: true, MarkdownDescription: "The API certificate type."},
		"common_name": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The exact, case-sensitive certificate common name. Exactly one of `id` or `common_name` must be configured.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"fqdn": schema.StringAttribute{Computed: true},
	}}
}

func (d *CertificateDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("common_name")),
	}
}

func (d *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CertificateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificates, err := d.client.GetCertificates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Certificates, got error: %s", err))
		return
	}

	matches := make([]client.CertificateData, 0, 1)
	for _, certificate := range certificates.Data {
		if (!config.ID.IsNull() && !config.ID.IsUnknown() && certificate.ID == config.ID.ValueString()) ||
			(!config.CommonName.IsNull() && !config.CommonName.IsUnknown() && certificate.Attributes.CommonName == config.CommonName.ValueString()) {
			matches = append(matches, certificate)
		}
	}

	selector := config.ID.ValueString()
	selectorLabel := "ID"
	if config.ID.IsNull() || config.ID.IsUnknown() {
		selector = config.CommonName.ValueString()
		selectorLabel = "exact common name"
	}
	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Certificate Not Found", fmt.Sprintf("Unable to find a Certificate with %s %q.", selectorLabel, selector))
		return
	case 1:
		// Continue below.
	default:
		resp.Diagnostics.AddError("Ambiguous Certificate Common Name", fmt.Sprintf("Found %d Certificates with exact common name %q. Configure the data source with an ID instead.", len(matches), selector))
		return
	}

	certificate := matches[0]
	state := CertificateDataSourceModel{
		ID:         types.StringValue(certificate.ID),
		Type:       types.StringValue(certificate.Type),
		CommonName: nullableStringValue(certificate.Attributes.CommonName),
		FQDN:       nullableStringValue(certificate.Attributes.FQDN),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
