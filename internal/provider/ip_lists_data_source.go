package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IPListsDataSource{}

func NewIPListsDataSource() datasource.DataSource {
	return &IPListsDataSource{}
}

type IPListsDataSource struct {
	client *client.Client
}

type IPListsDataSourceModel struct {
	IPLists []IPListDataSourceModel `tfsdk:"ip_lists"`
}

type IPListDataSourceModel struct {
	ID            types.String        `tfsdk:"id"`
	Type          types.String        `tfsdk:"type"`
	Name          types.String        `tfsdk:"name"`
	Entries       []IPListEntryModel  `tfsdk:"entries"`
	UsedBy        []IPListUsedByModel `tfsdk:"used_by"`
	CreatedAt     types.String        `tfsdk:"created_at"`
	LastUpdatedAt types.String        `tfsdk:"last_updated_at"`
}

type IPListUsedByModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

func (d *IPListsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_lists"
}

func (d *IPListsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all Baffin Bay IP Lists for the configured tenant.",
		Attributes: map[string]schema.Attribute{
			"ip_lists": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: ipListDataSourceAttributes(false),
				},
			},
		},
	}
}

func (d *IPListsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	configuredClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = configuredClient
}

func (d *IPListsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	response, err := d.client.GetIPLists(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IP lists, got error: %s", err))
		return
	}

	state := IPListsDataSourceModel{IPLists: make([]IPListDataSourceModel, 0, len(response.Data))}
	for _, ipList := range response.Data {
		if ipList.Type != client.IPListType {
			resp.Diagnostics.AddError("Unexpected IP List Type", fmt.Sprintf("IP list collection contained object %q with type %q, expected %q.", ipList.ID, ipList.Type, client.IPListType))
			return
		}

		state.IPLists = append(state.IPLists, mapIPListDataSourceModel(ipList))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func ipListDataSourceAttributes(idRequired bool) map[string]schema.Attribute {
	id := schema.Attribute(schema.StringAttribute{Computed: true})
	if idRequired {
		id = singularDataSourceIDAttribute("The UUID of the IP list to retrieve.")
	}
	return map[string]schema.Attribute{
		"id":   id,
		"type": schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{Computed: true},
		"entries": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"value": schema.StringAttribute{Computed: true},
			"note":  schema.StringAttribute{Computed: true},
		}}},
		"used_by": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true},
			"type": schema.StringAttribute{Computed: true},
		}}},
		"created_at":      schema.StringAttribute{Computed: true},
		"last_updated_at": schema.StringAttribute{Computed: true},
	}
}

func mapIPListDataSourceModel(ipList client.IPListData) IPListDataSourceModel {
	usedBy := make([]IPListUsedByModel, 0, len(ipList.Relationships.UsedBy.Data))
	for _, reference := range ipList.Relationships.UsedBy.Data {
		usedBy = append(usedBy, IPListUsedByModel{ID: types.StringValue(reference.ID), Type: types.StringValue(reference.Type)})
	}
	return IPListDataSourceModel{
		ID:            types.StringValue(ipList.ID),
		Type:          types.StringValue(ipList.Type),
		Name:          types.StringValue(ipList.Attributes.Name),
		Entries:       ipListEntriesToModel(ipList.Attributes.Entries),
		UsedBy:        usedBy,
		CreatedAt:     nullableStringValue(ipList.Meta.CreatedAt),
		LastUpdatedAt: nullableStringValue(ipList.Meta.LastUpdatedAt),
	}
}
