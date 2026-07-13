package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &IPListDataSource{}

func NewIPListDataSource() datasource.DataSource {
	return &IPListDataSource{}
}

type IPListDataSource struct {
	client *client.Client
}

func (d *IPListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_list"
}

func (d *IPListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Baffin Bay IP List by UUID.",
		Attributes:          ipListDataSourceAttributes(true),
	}
}

func (d *IPListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IPListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config IPListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := d.client.GetIPList(ctx, config.ID.ValueString())
	if err != nil {
		addSingularDataSourceReadError(&resp.Diagnostics, "IP List", config.ID.ValueString(), err)
		return
	}
	if response.Data.Type != client.IPListType {
		resp.Diagnostics.AddError("Unexpected IP List Type", fmt.Sprintf("IP List %q has API type %q, expected %q.", config.ID.ValueString(), response.Data.Type, client.IPListType))
		return
	}

	state := mapIPListDataSourceModel(response.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
