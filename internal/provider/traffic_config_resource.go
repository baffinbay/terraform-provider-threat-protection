package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &TrafficConfigResource{}
var _ resource.ResourceWithImportState = &TrafficConfigResource{}

func NewTrafficConfigResource() resource.Resource {
	return &TrafficConfigResource{}
}

type TrafficConfigResource struct {
	client *client.Client
}

type TrafficConfigResourceModel struct {
	ID              types.String  `tfsdk:"id"`
	Type            types.String  `tfsdk:"type"`
	Name            types.String  `tfsdk:"name"`
	DeploymentState types.String  `tfsdk:"deployment_state"`
	FrontendPort    types.Int64   `tfsdk:"frontend_port"`
	FrontendIPv4    types.String  `tfsdk:"frontend_ipv4"`
	FrontendIPv6    types.String  `tfsdk:"frontend_ipv6"`
	Prefix          types.String  `tfsdk:"prefix"`
	Announced       types.Bool    `tfsdk:"announced"`
	Backend         *BackendModel `tfsdk:"backend"`
}

type BackendModel struct {
	Hosts          []BackendHostModel `tfsdk:"hosts"`
	DeliveryMethod types.String       `tfsdk:"delivery_method"`
	ServerName     types.String       `tfsdk:"server_name"`
}

type BackendHostModel struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
}

func (r *TrafficConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_config"
}

func (r *TrafficConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay Traffic Configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the traffic configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of traffic configuration (l4Proxy, routedDsr, httpProxy).",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the traffic configuration.",
			},
			"deployment_state": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The deployment state (DEPLOYED, UNDEPLOYED).",
			},
			"frontend_port": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The frontend port. (L4/HTTP Proxy)",
			},
			"frontend_ipv4": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The frontend IPv4 address. (L4/HTTP Proxy)",
			},
			"frontend_ipv6": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The frontend IPv6 address. (L4/HTTP Proxy)",
			},
			"prefix": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The CIDR prefix. (Routed DSR)",
			},
			"announced": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Controls if the Routed Dsr is to be announced on Juniper. (Routed DSR)",
			},
			"backend": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"hosts": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"address": schema.StringAttribute{Required: true},
								"port":    schema.Int64Attribute{Required: true},
							},
						},
					},
					"delivery_method": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "ROUND_ROBIN or LEAST_CONNECTIONS",
					},
					"server_name": schema.StringAttribute{
						Required: true,
					},
				},
			},
		},
	}
}

func (r *TrafficConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *TrafficConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TrafficConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqData := client.TrafficConfigRequest{}
	reqData.Data.Type = data.Type.ValueString()
	reqData.Data.Attributes.Name = data.Name.ValueString()
	reqData.Data.Attributes.Version = "0.0.1"

	if !data.DeploymentState.IsNull() {
		reqData.Data.Attributes.Deployment.State = data.DeploymentState.ValueString()
	} else {
		reqData.Data.Attributes.Deployment.State = "UNDEPLOYED" // Default
	}

	// Frontend
	if !data.FrontendPort.IsNull() {
		reqData.Data.Attributes.Frontend = &client.Frontend{}
		reqData.Data.Attributes.Frontend.Port = data.FrontendPort.ValueInt64()
		if !data.FrontendIPv4.IsNull() {
			reqData.Data.Attributes.Frontend.IPv4 = data.FrontendIPv4.ValueString()
		}
		if !data.FrontendIPv6.IsNull() {
			reqData.Data.Attributes.Frontend.IPv6 = data.FrontendIPv6.ValueString()
		}
	}

	// Backend
	if data.Backend != nil {
		reqData.Data.Attributes.Backend = &client.Backend{}
		reqData.Data.Attributes.Backend.DeliveryMethod = data.Backend.DeliveryMethod.ValueString()
		reqData.Data.Attributes.Backend.ServerName = data.Backend.ServerName.ValueString()
		for _, host := range data.Backend.Hosts {
			reqData.Data.Attributes.Backend.Hosts = append(reqData.Data.Attributes.Backend.Hosts, client.Host{
				Address: host.Address.ValueString(),
				Port:    host.Port.ValueInt64(),
			})
		}
	}

	// DSR specific
	if !data.Prefix.IsNull() {
		reqData.Data.Attributes.Prefix = data.Prefix.ValueString()
	}
	if !data.Announced.IsNull() {
		reqData.Data.Attributes.Announced = data.Announced.ValueBool()
	}

	// L4 specific (Protocols) - defaulting to TCP if not specified or implicitly handled by API/client
	if data.Type.ValueString() == "l4Proxy" {
		reqData.Data.Attributes.Protocols = []string{"TCP"}
	}

	tc, err := r.client.CreateTrafficConfig(reqData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create traffic config, got error: %s", err))
		return
	}

	data.ID = types.StringValue(tc.Data.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TrafficConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TrafficConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *TrafficConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *TrafficConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TrafficConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTrafficConfig(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete traffic config, got error: %s", err))
		return
	}
}

func (r *TrafficConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
