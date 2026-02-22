package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IpSourcesDataSource{}

func NewIpSourcesDataSource() datasource.DataSource {
	return &IpSourcesDataSource{}
}

type IpSourcesDataSource struct {
	client *client.Client
}

type IpSourcesDataSourceModel struct {
	ID        types.String    `tfsdk:"id"`
	IpSources []IpSourceModel `tfsdk:"ip_sources"`
}

type IpSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	CIDR types.String `tfsdk:"cidr"`
}

func (d *IpSourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_sources"
}

func (d *IpSourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all Baffin Bay TPC IP Sources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"ip_sources": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"cidr": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *IpSourcesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *IpSourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state IpSourcesDataSourceModel

	respIp, err := d.client.GetIpSources()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IP sources, got error: %s", err))
		return
	}

	state.ID = types.StringValue("ip_sources")

	for _, ip := range respIp.Data {
		state.IpSources = append(state.IpSources, IpSourceModel{
			ID:   types.StringValue(ip.ID),
			Type: types.StringValue(ip.Type),
			CIDR: types.StringValue(ip.Attributes.CIDR),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
