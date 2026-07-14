package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &KnownServiceDataSource{}
var _ datasource.DataSourceWithConfigValidators = &KnownServiceDataSource{}

func NewKnownServiceDataSource() datasource.DataSource {
	return &KnownServiceDataSource{}
}

type KnownServiceDataSource struct {
	configuredDataSource
}

type KnownServiceDataSourceModel struct {
	ID               types.String   `tfsdk:"id"`
	Name             types.String   `tfsdk:"name"`
	Tags             []types.String `tfsdk:"tags"`
	UpstreamProvider types.String   `tfsdk:"upstream_provider"`
}

func (d *KnownServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_known_service"
}

func (d *KnownServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Baffin Bay-managed Known Service by UUID or exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The Known Service UUID. Exactly one of `id` or `name` must be configured.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(singularDataSourceUUIDPattern, "must be a valid UUID"),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact, case-sensitive Known Service name. Exactly one of `id` or `name` must be configured.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"tags":              schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"upstream_provider": schema.StringAttribute{Computed: true, MarkdownDescription: "The upstream provider or owner of the service."},
		},
	}
}

func (d *KnownServiceDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *KnownServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KnownServiceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var knownService client.KnownServiceData
	if !config.ID.IsNull() && !config.ID.IsUnknown() {
		response, err := d.client.GetKnownService(ctx, config.ID.ValueString())
		if err != nil {
			addSingularDataSourceReadError(&resp.Diagnostics, "Known Service", config.ID.ValueString(), err)
			return
		}
		knownService = response.Data
	} else {
		response, err := d.client.GetKnownServices(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Known Services while looking up name %q, got error: %s", config.Name.ValueString(), err))
			return
		}
		match, ok := findDataSourceByExactName(&resp.Diagnostics, "Known Service", "Known Services", config.Name.ValueString(), response.Data, func(candidate client.KnownServiceData) string {
			return candidate.Attributes.Name
		})
		if !ok {
			return
		}
		knownService = match
	}

	if knownService.Type != client.KnownServiceType {
		resp.Diagnostics.AddError("Unexpected Known Service Type", fmt.Sprintf("Known Service %q has API type %q, expected %q.", knownService.ID, knownService.Type, client.KnownServiceType))
		return
	}
	state := KnownServiceDataSourceModel{
		ID:               types.StringValue(knownService.ID),
		Name:             types.StringValue(knownService.Attributes.Name),
		Tags:             stringSliceToTypeValues(knownService.Attributes.Tags),
		UpstreamProvider: nullableStringValue(knownService.Attributes.Provider),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
