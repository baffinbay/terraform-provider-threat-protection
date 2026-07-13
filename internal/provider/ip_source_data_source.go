package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IPSourceDataSource{}

func NewIPSourceDataSource() datasource.DataSource {
	return &IPSourceDataSource{}
}

type IPSourceDataSource struct {
	client *client.Client
}

func (d *IPSourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_source"
}

func (d *IPSourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Baffin Bay TPC IP Source by UUID.", Attributes: map[string]schema.Attribute{
		"id":   singularDataSourceIDAttribute("The UUID of the IP source to retrieve."),
		"type": schema.StringAttribute{Computed: true},
		"cidr": schema.StringAttribute{Computed: true},
	}}
}

func (d *IPSourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IPSourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IpSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetIpSources(ctx)
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "IP Source", state.ID.ValueString(), err)
		return
	}
	for _, source := range response.Data {
		if source.ID == state.ID.ValueString() {
			state.ID = types.StringValue(source.ID)
			state.Type = types.StringValue(source.Type)
			state.CIDR = types.StringValue(source.Attributes.CIDR)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddError("IP Source Not Found", fmt.Sprintf("Unable to find IP Source with ID %q.", state.ID.ValueString()))
}
