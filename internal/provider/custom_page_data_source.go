package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CustomPageDataSource{}

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
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay Custom Page by UUID.", Attributes: map[string]schema.Attribute{
		"id":         singularDataSourceIDAttribute("The UUID of the custom page to retrieve."),
		"type":       schema.StringAttribute{Computed: true},
		"name":       schema.StringAttribute{Computed: true},
		"created_at": schema.StringAttribute{Computed: true},
		"content":    schema.StringAttribute{Computed: true, MarkdownDescription: "The HTML content downloaded from the custom page file endpoint."},
	}}
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
	var state CustomPageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	page, err := d.client.FindCustomPageByID(ctx, state.ID.ValueString())
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "Custom Page", state.ID.ValueString(), err)
		return
	}
	content, err := d.client.DownloadCustomPage(ctx, state.ID.ValueString())
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "Custom Page", state.ID.ValueString(), err)
		return
	}
	state.ID = types.StringValue(page.ID)
	state.Type = types.StringValue(page.Type)
	state.Name = nullableStringValue(page.Attributes.Name)
	state.CreatedAt = nullableStringValue(page.Attributes.CreatedAt)
	state.Content = types.StringValue(content)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
