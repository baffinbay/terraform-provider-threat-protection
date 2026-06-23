package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TrafficConfigsDataSource{}

func NewTrafficConfigsDataSource() datasource.DataSource {
	return &TrafficConfigsDataSource{}
}

type TrafficConfigsDataSource struct {
	client *client.Client
}

type TrafficConfigsDataSourceModel struct {
	Configs []TrafficConfigModel `tfsdk:"configs"`
}

type TrafficConfigModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

func (d *TrafficConfigsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_configs"
}

func (d *TrafficConfigsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all Baffin Bay Traffic Configurations.",
		Attributes: map[string]schema.Attribute{
			"configs": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"type": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *TrafficConfigsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TrafficConfigsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TrafficConfigsDataSourceModel

	respConfigs, err := d.client.GetTrafficConfigs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read traffic configs, got error: %s", err))
		return
	}

	for _, config := range respConfigs.Data {
		name := types.StringNull()
		if config.Attributes.Name != nil {
			name = types.StringValue(*config.Attributes.Name)
		}

		state.Configs = append(state.Configs, TrafficConfigModel{
			ID:   types.StringValue(config.ID),
			Type: types.StringValue(config.Type),
			Name: name,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
