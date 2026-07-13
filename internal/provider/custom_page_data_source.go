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

var _ datasource.DataSource = &CustomPageDataSource{}
var _ datasource.DataSourceWithConfigValidators = &CustomPageDataSource{}

func NewCustomPageDataSource() datasource.DataSource {
	return &CustomPageDataSource{}
}

type CustomPageDataSource struct {
	client *client.Client
}

type CustomPageDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Type      types.String `tfsdk:"type"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
	Content   types.String `tfsdk:"content"`
}

func (d *CustomPageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_page"
}

func (d *CustomPageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay Custom Page by UUID or exact name.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The custom page UUID. Exactly one of `id` or `name` must be configured.",
			Validators: []validator.String{
				stringvalidator.RegexMatches(singularDataSourceUUIDPattern, "must be a valid UUID"),
			},
		},
		"type": schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "The exact, case-sensitive custom page name. Exactly one of `id` or `name` must be configured.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"created_at": schema.StringAttribute{Computed: true},
		"content":    schema.StringAttribute{Computed: true, MarkdownDescription: "The HTML content downloaded from the custom page file endpoint."},
	}}
}

func (d *CustomPageDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *CustomPageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CustomPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CustomPageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var page client.CustomPageData
	if !config.ID.IsNull() && !config.ID.IsUnknown() {
		response, err := d.client.FindCustomPageByID(ctx, config.ID.ValueString())
		if err != nil {
			addSingularDataSourceReadError(&resp.Diagnostics, "Custom Page", config.ID.ValueString(), err)
			return
		}
		page = *response
	} else {
		response, err := d.client.GetCustomPages(ctx, d.client.TenantID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Custom Pages while looking up name %q, got error: %s", config.Name.ValueString(), err))
			return
		}
		matches := make([]client.CustomPageData, 0, 1)
		for _, candidate := range response.Data {
			if candidate.Attributes.Name == config.Name.ValueString() {
				matches = append(matches, candidate)
			}
		}
		switch len(matches) {
		case 0:
			resp.Diagnostics.AddError("Custom Page Not Found", fmt.Sprintf("Unable to find a Custom Page with exact name %q.", config.Name.ValueString()))
			return
		case 1:
			page = matches[0]
		default:
			resp.Diagnostics.AddError("Ambiguous Custom Page Name", fmt.Sprintf("Found %d Custom Pages with exact name %q. Configure the data source with an ID instead.", len(matches), config.Name.ValueString()))
			return
		}
	}

	content, err := d.client.DownloadCustomPage(ctx, page.ID)
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "Custom Page", page.ID, err)
		return
	}
	state := CustomPageDataSourceModel{
		ID:        types.StringValue(page.ID),
		Type:      types.StringValue(page.Type),
		Name:      nullableStringValue(page.Attributes.Name),
		CreatedAt: nullableStringValue(page.Attributes.CreatedAt),
		Content:   types.StringValue(content),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
