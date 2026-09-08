package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-threat-protection/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	l4ProxyTrafficConfigType    = "l4Proxy"
	l4ProxyTrafficConfigVersion = "0.0.1"
)

var _ resource.Resource = &L4ProxyResource{}
var _ resource.ResourceWithImportState = &L4ProxyResource{}

func NewL4ProxyResource() resource.Resource {
	return &L4ProxyResource{}
}

type L4ProxyResource struct {
	configuredResource
}

type L4ProxyResourceModel struct {
	ID                   types.String                 `tfsdk:"id"`
	Name                 types.String                 `tfsdk:"name"`
	DeploymentState      types.String                 `tfsdk:"deployment_state"`
	Protocols            []types.String               `tfsdk:"protocols"`
	ProxyProtocol        types.String                 `tfsdk:"proxy_protocol"`
	Frontend             *L4FrontendModel             `tfsdk:"frontend"`
	Backend              *L4BackendModel              `tfsdk:"backend"`
	IPBasedAccessControl *L4IPBasedAccessControlModel `tfsdk:"ip_based_access_control"`
}

type L4FrontendModel struct {
	IPv4 types.String `tfsdk:"ipv4"`
	IPv6 types.String `tfsdk:"ipv6"`
	Port types.Int64  `tfsdk:"port"`
}

type L4BackendModel struct {
	Hosts          []L4BackendHostModel `tfsdk:"hosts"`
	DeliveryMethod types.String         `tfsdk:"delivery_method"`
	ServerName     types.String         `tfsdk:"server_name"`
}

type L4BackendHostModel struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
}

type L4IPBasedAccessControlModel struct {
	DefaultPolicy types.String                      `tfsdk:"default_policy"`
	Rules         *L4IPBasedAccessControlRulesModel `tfsdk:"rules"`
}

type L4IPBasedAccessControlRulesModel struct {
	IPRanges     []L4IPRangeRuleModel      `tfsdk:"ip_ranges"`
	IPLists      []L4IPListRuleModel       `tfsdk:"ip_lists"`
	GeoLocations []L4GeoLocationRuleModel  `tfsdk:"geo_locations"`
	ASNs         []L4AutonomousSystemModel `tfsdk:"asns"`
}

type L4IPRangeRuleModel struct {
	Policy  types.String `tfsdk:"policy"`
	Address types.String `tfsdk:"address"`
	Note    types.String `tfsdk:"note"`
}

type L4IPListRuleModel struct {
	ID     types.String `tfsdk:"id"`
	Policy types.String `tfsdk:"policy"`
}

type L4GeoLocationRuleModel struct {
	Policy types.String `tfsdk:"policy"`
	Region types.String `tfsdk:"region"`
	Note   types.String `tfsdk:"note"`
}

type L4AutonomousSystemModel struct {
	Policy types.String `tfsdk:"policy"`
	ASN    types.Int64  `tfsdk:"asn"`
	Note   types.String `tfsdk:"note"`
}

func (r *L4ProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_l4_proxy"
}

func (r *L4ProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay L4 Proxy traffic configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the L4 Proxy traffic configuration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the L4 Proxy traffic configuration.",
			},
			"deployment_state": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The deployment state (DEPLOYED, UNDEPLOYED).",
				Default:             stringdefault.StaticString("UNDEPLOYED"),
				Validators: []validator.String{
					stringvalidator.OneOf("DEPLOYED", "UNDEPLOYED"),
				},
			},
			"protocols": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The L4 protocols to proxy (TCP, UDP).",
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("TCP"),
				})),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("TCP", "UDP")),
				},
			},
			"proxy_protocol": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Controls the L4 proxy protocol setting (ENABLED, DISABLED).",
				Default:             stringdefault.StaticString("DISABLED"),
				Validators: []validator.String{
					stringvalidator.OneOf("ENABLED", "DISABLED"),
				},
			},
			"ip_based_access_control": l4IPBasedAccessControlAttribute(),
			"frontend": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The L4 frontend configuration.",
				Attributes: map[string]schema.Attribute{
					"ipv4": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The frontend IPv4 address.",
						Validators: []validator.String{
							stringvalidator.AtLeastOneOf(path.Expressions{
								path.MatchRelative().AtParent().AtName("ipv6"),
							}...),
						},
					},
					"ipv6": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The frontend IPv6 address.",
					},
					"port": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "The frontend port.",
					},
				},
			},
			"backend": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The L4 backend configuration.",
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
						Validators: []validator.String{
							stringvalidator.OneOf("ROUND_ROBIN", "LEAST_CONNECTIONS"),
						},
					},
					"server_name": schema.StringAttribute{
						Required: true,
					},
				},
			},
		},
	}
}

func l4IPBasedAccessControlAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "Controls L4 source IP based access control.",
		Default:             objectdefault.StaticValue(defaultL4IPBasedAccessControlObjectValue()),
		Attributes: map[string]schema.Attribute{
			"default_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Default IP based access policy (ALLOW, BLOCK).",
				Default:             stringdefault.StaticString("ALLOW"),
				Validators: []validator.String{
					stringvalidator.OneOf("ALLOW", "BLOCK"),
				},
			},
			"rules": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "IP based access rules.",
				Default:             objectdefault.StaticValue(defaultL4IPBasedAccessRulesObjectValue()),
				Attributes: map[string]schema.Attribute{
					"ip_ranges": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "CIDR based access rules.",
						Default:             listdefault.StaticValue(emptyObjectListValue(l4IPRangeRuleAttrTypes())),
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"policy": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Policy for this CIDR based access rule (ALLOW, BLOCK).",
									Validators: []validator.String{
										stringvalidator.OneOf("ALLOW", "BLOCK"),
									},
								},
								"address": schema.StringAttribute{Required: true},
								"note":    schema.StringAttribute{Optional: true},
							},
						},
					},
					"ip_lists": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "IP list access rules.",
						Default:             listdefault.StaticValue(emptyObjectListValue(l4IPListRuleAttrTypes())),
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{Required: true},
								"policy": schema.StringAttribute{
									Required: true,
									Validators: []validator.String{
										stringvalidator.OneOf("ALLOW", "BLOCK"),
									},
								},
							},
						},
					},
					"geo_locations": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Geo location access rules.",
						Default:             listdefault.StaticValue(emptyObjectListValue(l4GeoLocationRuleAttrTypes())),
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"policy": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Policy for this geo location access rule (ALLOW, BLOCK).",
									Validators: []validator.String{
										stringvalidator.OneOf("ALLOW", "BLOCK"),
									},
								},
								"region": schema.StringAttribute{Required: true},
								"note":   schema.StringAttribute{Optional: true},
							},
						},
					},
					"asns": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Autonomous system access rules.",
						Default:             listdefault.StaticValue(emptyObjectListValue(l4AutonomousSystemRuleAttrTypes())),
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"policy": schema.StringAttribute{
									Required: true,
									Validators: []validator.String{
										stringvalidator.OneOf("ALLOW", "BLOCK"),
									},
								},
								"asn":  schema.Int64Attribute{Required: true},
								"note": schema.StringAttribute{Optional: true},
							},
						},
					},
				},
			},
		},
	}
}

func defaultL4IPBasedAccessControlObjectValue() types.Object {
	return types.ObjectValueMust(l4IPBasedAccessControlAttrTypes(), map[string]attr.Value{
		"default_policy": types.StringValue("ALLOW"),
		"rules":          defaultL4IPBasedAccessRulesObjectValue(),
	})
}

func defaultL4IPBasedAccessRulesObjectValue() types.Object {
	return types.ObjectValueMust(l4IPBasedAccessRulesAttrTypes(), map[string]attr.Value{
		"ip_ranges":     emptyObjectListValue(l4IPRangeRuleAttrTypes()),
		"ip_lists":      emptyObjectListValue(l4IPListRuleAttrTypes()),
		"geo_locations": emptyObjectListValue(l4GeoLocationRuleAttrTypes()),
		"asns":          emptyObjectListValue(l4AutonomousSystemRuleAttrTypes()),
	})
}

func emptyObjectListValue(attrTypes map[string]attr.Type) types.List {
	return types.ListValueMust(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{})
}

func l4IPBasedAccessControlAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"default_policy": types.StringType,
		"rules":          types.ObjectType{AttrTypes: l4IPBasedAccessRulesAttrTypes()},
	}
}

func l4IPBasedAccessRulesAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip_ranges":     types.ListType{ElemType: types.ObjectType{AttrTypes: l4IPRangeRuleAttrTypes()}},
		"ip_lists":      types.ListType{ElemType: types.ObjectType{AttrTypes: l4IPListRuleAttrTypes()}},
		"geo_locations": types.ListType{ElemType: types.ObjectType{AttrTypes: l4GeoLocationRuleAttrTypes()}},
		"asns":          types.ListType{ElemType: types.ObjectType{AttrTypes: l4AutonomousSystemRuleAttrTypes()}},
	}
}

func l4IPRangeRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"policy":  types.StringType,
		"address": types.StringType,
		"note":    types.StringType,
	}
}

func l4IPListRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":     types.StringType,
		"policy": types.StringType,
	}
}

func l4GeoLocationRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"policy": types.StringType,
		"region": types.StringType,
		"note":   types.StringType,
	}
}

func l4AutonomousSystemRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"policy": types.StringType,
		"asn":    types.Int64Type,
		"note":   types.StringType,
	}
}

func (r *L4ProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data L4ProxyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tc, err := r.client.CreateL4Proxy(ctx, mapL4ProxyModelToRequest(data))
	if err != nil {
		if addFrontendBindingConflictDiagnostic(&resp.Diagnostics, err, "create", "L4 Proxy", "baffinbay_l4_proxy", data.Name.ValueString(), l4ProxyFrontendBindingDescription(data)) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create L4 Proxy traffic config, got error: %s", err))
		return
	}

	data.ID = types.StringValue(tc.Data.ID)

	if err := r.client.WaitForTrafficConfigRollout(ctx, tc.Data.ID, tc.Data.ActiveRolloutID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for L4 Proxy traffic config create, got error: %s", err))
		return
	}

	data, err = r.readL4Proxy(ctx, data)
	if err != nil {
		addL4ProxyReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "after create")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *L4ProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data L4ProxyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.readL4Proxy(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		addL4ProxyReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *L4ProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data L4ProxyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tc, err := r.client.UpdateL4Proxy(ctx, data.ID.ValueString(), mapL4ProxyModelToRequest(data))
	if err != nil {
		if addFrontendBindingConflictDiagnostic(&resp.Diagnostics, err, "update", "L4 Proxy", "baffinbay_l4_proxy", data.Name.ValueString(), l4ProxyFrontendBindingDescription(data)) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update L4 Proxy traffic config, got error: %s", err))
		return
	}
	if err := r.client.WaitForTrafficConfigRollout(ctx, data.ID.ValueString(), tc.Data.ActiveRolloutID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for L4 Proxy traffic config update, got error: %s", err))
		return
	}

	data, err = r.readL4Proxy(ctx, data)
	if err != nil {
		addL4ProxyReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "after update")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *L4ProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data L4ProxyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	undeploy := func(ctx context.Context) (string, error) {
		current, err := r.readL4Proxy(ctx, data)
		if err != nil {
			return "", err
		}
		current.DeploymentState = types.StringValue("UNDEPLOYED")
		tc, err := r.client.UpdateL4Proxy(ctx, current.ID.ValueString(), mapL4ProxyModelToRequest(current))
		if err != nil {
			return "", err
		}
		return tc.Data.ActiveRolloutID(), nil
	}

	deleteTrafficConfigResource(
		ctx,
		resp,
		r.client,
		"L4 Proxy",
		data.ID.ValueString(),
		func(tc *client.TrafficConfigResponse) bool { return tc.Data.Type == l4ProxyTrafficConfigType },
		undeploy,
		addL4ProxyReadDiagnostic,
	)
}

func (r *L4ProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapL4ProxyModelToRequest(data L4ProxyResourceModel) client.L4ProxyRequest {
	attrs := client.L4ProxyAttributes{
		Name:          data.Name.ValueString(),
		Version:       l4ProxyTrafficConfigVersion,
		Protocols:     l4ProxyProtocolsOrDefault(data.Protocols),
		ProxyProtocol: l4ProxyProxyProtocolOrDefault(data.ProxyProtocol),
		IPBasedAccess: mapL4IPBasedAccessControlToRequest(data.IPBasedAccessControl),
	}

	if !data.DeploymentState.IsNull() {
		attrs.Deployment.State = data.DeploymentState.ValueString()
	} else {
		attrs.Deployment.State = "UNDEPLOYED"
	}

	if data.Frontend != nil {
		attrs.Frontend = &client.L4ProxyFrontend{
			Port: data.Frontend.Port.ValueInt64(),
		}
		if !data.Frontend.IPv4.IsNull() {
			attrs.Frontend.IPv4 = data.Frontend.IPv4.ValueString()
			attrs.Frontend.IP = data.Frontend.IPv4.ValueString()
		}
		if !data.Frontend.IPv6.IsNull() {
			attrs.Frontend.IPv6 = data.Frontend.IPv6.ValueString()
			if attrs.Frontend.IP == "" {
				attrs.Frontend.IP = data.Frontend.IPv6.ValueString()
			}
		}
	}

	if data.Backend != nil {
		attrs.Backend = &client.L4ProxyBackend{
			DeliveryMethod: data.Backend.DeliveryMethod.ValueString(),
			ServerName:     data.Backend.ServerName.ValueString(),
		}
		for _, host := range data.Backend.Hosts {
			attrs.Backend.Hosts = append(attrs.Backend.Hosts, client.L4ProxyHost{
				Address: host.Address.ValueString(),
				Port:    host.Port.ValueInt64(),
			})
		}
	}

	return client.NewL4ProxyRequest(attrs)
}

func mapL4IPBasedAccessControlToRequest(data *L4IPBasedAccessControlModel) *client.L4ProxyIPBasedAccess {
	req := defaultL4IPBasedAccessControlRequest()
	if data == nil {
		return req
	}

	if !data.DefaultPolicy.IsNull() && !data.DefaultPolicy.IsUnknown() {
		req.DefaultPolicy = data.DefaultPolicy.ValueString()
	}
	if data.Rules != nil {
		req.Rules = mapL4IPBasedAccessControlRulesToRequest(data.Rules)
	}

	return req
}

func defaultL4IPBasedAccessControlRequest() *client.L4ProxyIPBasedAccess {
	return &client.L4ProxyIPBasedAccess{
		DefaultPolicy: "ALLOW",
		Rules: client.L4ProxyIPBasedAccessRules{
			IPRanges:     []client.L4ProxyIPBasedAccessIPRangeRule{},
			IPLists:      []client.L4ProxyIPBasedAccessIPListRule{},
			GeoLocations: []client.L4ProxyIPBasedAccessGeoLocationRule{},
			ASNs:         []client.L4ProxyIPBasedAccessAutonomousSystem{},
		},
	}
}

func mapL4IPBasedAccessControlRulesToRequest(data *L4IPBasedAccessControlRulesModel) client.L4ProxyIPBasedAccessRules {
	req := client.L4ProxyIPBasedAccessRules{
		IPRanges:     []client.L4ProxyIPBasedAccessIPRangeRule{},
		IPLists:      []client.L4ProxyIPBasedAccessIPListRule{},
		GeoLocations: []client.L4ProxyIPBasedAccessGeoLocationRule{},
		ASNs:         []client.L4ProxyIPBasedAccessAutonomousSystem{},
	}

	for _, rule := range data.IPRanges {
		req.IPRanges = append(req.IPRanges, client.L4ProxyIPBasedAccessIPRangeRule{
			Policy:  rule.Policy.ValueString(),
			Address: rule.Address.ValueString(),
			Note:    optionalStringValue(rule.Note),
		})
	}
	for _, rule := range data.IPLists {
		req.IPLists = append(req.IPLists, client.L4ProxyIPBasedAccessIPListRule{
			Type:   "ipList",
			ID:     rule.ID.ValueString(),
			Policy: rule.Policy.ValueString(),
		})
	}
	for _, rule := range data.GeoLocations {
		req.GeoLocations = append(req.GeoLocations, client.L4ProxyIPBasedAccessGeoLocationRule{
			Policy: rule.Policy.ValueString(),
			Region: rule.Region.ValueString(),
			Note:   optionalStringValue(rule.Note),
		})
	}
	for _, rule := range data.ASNs {
		req.ASNs = append(req.ASNs, client.L4ProxyIPBasedAccessAutonomousSystem{
			Policy: rule.Policy.ValueString(),
			ASN:    rule.ASN.ValueInt64(),
			Note:   optionalStringValue(rule.Note),
		})
	}

	return req
}

func l4ProxyProxyProtocolOrDefault(proxyProtocol types.String) string {
	if proxyProtocol.IsNull() || proxyProtocol.IsUnknown() {
		return "DISABLED"
	}

	return proxyProtocol.ValueString()
}

func l4ProxyProtocolsOrDefault(protocols []types.String) []string {
	if len(protocols) == 0 {
		return []string{"TCP"}
	}

	result := make([]string, 0, len(protocols))
	for _, protocol := range protocols {
		if !protocol.IsNull() && !protocol.IsUnknown() {
			result = append(result, protocol.ValueString())
		}
	}
	if len(result) == 0 {
		return []string{"TCP"}
	}

	return result
}

func optionalStringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}

	return value.ValueString()
}

func (r *L4ProxyResource) readL4Proxy(ctx context.Context, prior L4ProxyResourceModel) (L4ProxyResourceModel, error) {
	tc, err := r.client.GetL4Proxy(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}

	if tc.Data.Type != l4ProxyTrafficConfigType {
		return prior, trafficConfigTypeMismatchError{
			ID:       prior.ID.ValueString(),
			GotType:  tc.Data.Type,
			WantType: l4ProxyTrafficConfigType,
		}
	}

	return mapL4ProxyResponseToModel(tc, prior), nil
}

func mapL4ProxyResponseToModel(tc *client.TrafficConfigResponse, prior L4ProxyResourceModel) L4ProxyResourceModel {
	data := prior

	if tc.Data.ID != "" {
		data.ID = types.StringValue(tc.Data.ID)
	}

	attrs := tc.Data.Attributes
	if attrs.Name != nil {
		data.Name = types.StringValue(*attrs.Name)
	}
	if attrs.Deployment != nil && attrs.Deployment.State != nil {
		data.DeploymentState = types.StringValue(*attrs.Deployment.State)
	} else if data.DeploymentState.IsNull() || data.DeploymentState.IsUnknown() {
		data.DeploymentState = types.StringValue("UNDEPLOYED")
	}
	if attrs.Protocols != nil {
		data.Protocols = stringSliceToTypeValues(attrs.Protocols)
	} else if len(data.Protocols) == 0 {
		data.Protocols = stringSliceToTypeValues([]string{"TCP"})
	}
	if attrs.ProxyProtocol != nil {
		data.ProxyProtocol = types.StringValue(*attrs.ProxyProtocol)
	} else if data.ProxyProtocol.IsNull() || data.ProxyProtocol.IsUnknown() {
		data.ProxyProtocol = types.StringValue("DISABLED")
	}
	data.IPBasedAccessControl = mapL4IPBasedAccessControlResponseToModel(attrs.IPBasedAccess, data.IPBasedAccessControl)
	if attrs.Frontend != nil {
		data.Frontend = mapL4FrontendResponseToModel(attrs.Frontend, data.Frontend)
	}
	if attrs.Backend != nil {
		data.Backend = mapL4BackendResponseToModel(attrs.Backend, data.Backend)
	}

	return data
}

func mapL4IPBasedAccessControlResponseToModel(ipBasedAccess *client.TrafficConfigResponseIPBasedAccess, prior *L4IPBasedAccessControlModel) *L4IPBasedAccessControlModel {
	data := defaultL4IPBasedAccessControlModel()
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}
	if ipBasedAccess == nil {
		return data
	}
	if l4IPBasedAccessResponseIsDefault(ipBasedAccess) && !l4IPBasedAccessControlModelIsDefault(prior) {
		return data
	}

	if ipBasedAccess.DefaultPolicy != nil {
		data.DefaultPolicy = types.StringValue(*ipBasedAccess.DefaultPolicy)
	}
	data.Rules = mapL4IPBasedAccessControlRulesResponseToModel(ipBasedAccess.Rules, data.Rules)

	return data
}

func defaultL4IPBasedAccessControlModel() *L4IPBasedAccessControlModel {
	return &L4IPBasedAccessControlModel{
		DefaultPolicy: types.StringValue("ALLOW"),
		Rules:         defaultL4IPBasedAccessControlRulesModel(),
	}
}

func l4IPBasedAccessResponseIsDefault(ipBasedAccess *client.TrafficConfigResponseIPBasedAccess) bool {
	if ipBasedAccess == nil {
		return true
	}
	if ipBasedAccess.DefaultPolicy != nil && *ipBasedAccess.DefaultPolicy != "ALLOW" {
		return false
	}

	return l4IPBasedAccessRulesResponseIsDefault(ipBasedAccess.Rules)
}

func l4IPBasedAccessRulesResponseIsDefault(rules *client.TrafficConfigResponseIPBasedAccessRules) bool {
	if rules == nil {
		return true
	}

	return len(rules.IPRanges) == 0 &&
		len(rules.IPLists) == 0 &&
		len(rules.GeoLocations) == 0 &&
		len(rules.ASNs) == 0
}

func l4IPBasedAccessControlModelIsDefault(data *L4IPBasedAccessControlModel) bool {
	if data == nil {
		return true
	}
	if !data.DefaultPolicy.IsNull() && !data.DefaultPolicy.IsUnknown() && data.DefaultPolicy.ValueString() != "ALLOW" {
		return false
	}

	return l4IPBasedAccessRulesModelIsDefault(data.Rules)
}

func l4IPBasedAccessRulesModelIsDefault(rules *L4IPBasedAccessControlRulesModel) bool {
	if rules == nil {
		return true
	}

	return len(rules.IPRanges) == 0 &&
		len(rules.IPLists) == 0 &&
		len(rules.GeoLocations) == 0 &&
		len(rules.ASNs) == 0
}

func mapL4IPBasedAccessControlRulesResponseToModel(rules *client.TrafficConfigResponseIPBasedAccessRules, prior *L4IPBasedAccessControlRulesModel) *L4IPBasedAccessControlRulesModel {
	data := defaultL4IPBasedAccessControlRulesModel()
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}
	if rules == nil {
		return data
	}

	data.IPRanges = mapL4IPRangeRulesResponseToModel(rules.IPRanges, data.IPRanges)
	data.IPLists = make([]L4IPListRuleModel, 0, len(rules.IPLists))
	for _, rule := range rules.IPLists {
		data.IPLists = append(data.IPLists, L4IPListRuleModel{
			ID:     optionalStringPtrValue(rule.ID),
			Policy: optionalStringPtrValue(rule.Policy),
		})
	}
	data.GeoLocations = mapL4GeoLocationRulesResponseToModel(rules.GeoLocations, data.GeoLocations)
	data.ASNs = make([]L4AutonomousSystemModel, 0, len(rules.ASNs))
	for _, rule := range rules.ASNs {
		data.ASNs = append(data.ASNs, L4AutonomousSystemModel{
			Policy: optionalStringPtrValue(rule.Policy),
			ASN:    optionalInt64PtrValue(rule.ASN),
			Note:   optionalStringPtrValue(rule.Note),
		})
	}

	return data
}

func mapL4GeoLocationRulesResponseToModel(rules []client.TrafficConfigResponseGeoLocationRule, prior []L4GeoLocationRuleModel) []L4GeoLocationRuleModel {
	data := make([]L4GeoLocationRuleModel, 0, len(rules))
	for _, rule := range rules {
		data = append(data, L4GeoLocationRuleModel{
			Policy: optionalStringPtrValue(rule.Policy),
			Region: optionalStringPtrValue(rule.Region),
			Note:   optionalStringPtrValue(rule.Note),
		})
	}

	return data
}

func mapL4IPRangeRulesResponseToModel(rules []client.TrafficConfigResponseIPRangeRule, prior []L4IPRangeRuleModel) []L4IPRangeRuleModel {
	data := make([]L4IPRangeRuleModel, 0, len(rules))
	for _, rule := range rules {
		data = append(data, L4IPRangeRuleModel{
			Policy:  optionalStringPtrValue(rule.Policy),
			Address: optionalStringPtrValue(rule.Address),
			Note:    optionalStringPtrValue(rule.Note),
		})
	}

	return data
}

func defaultL4IPBasedAccessControlRulesModel() *L4IPBasedAccessControlRulesModel {
	return &L4IPBasedAccessControlRulesModel{
		IPRanges:     []L4IPRangeRuleModel{},
		IPLists:      []L4IPListRuleModel{},
		GeoLocations: []L4GeoLocationRuleModel{},
		ASNs:         []L4AutonomousSystemModel{},
	}
}

func mapL4FrontendResponseToModel(frontend *client.TrafficConfigResponseFrontend, prior *L4FrontendModel) *L4FrontendModel {
	data := &L4FrontendModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}

	if frontend.IPv4 != nil {
		data.IPv4 = types.StringValue(*frontend.IPv4)
	} else if frontend.IP != nil {
		data.IPv4 = types.StringValue(*frontend.IP)
	}
	if frontend.IPv6 != nil {
		data.IPv6 = types.StringValue(*frontend.IPv6)
	}
	if frontend.Port != nil {
		data.Port = types.Int64Value(*frontend.Port)
	}

	return data
}

func mapL4BackendResponseToModel(backend *client.TrafficConfigResponseBackend, prior *L4BackendModel) *L4BackendModel {
	data := &L4BackendModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}

	if backend.Hosts != nil {
		data.Hosts = make([]L4BackendHostModel, 0, len(backend.Hosts))
		for _, host := range backend.Hosts {
			data.Hosts = append(data.Hosts, L4BackendHostModel{
				Address: optionalStringPtrValue(host.Address),
				Port:    optionalInt64PtrValue(host.Port),
			})
		}
	}
	if backend.DeliveryMethod != nil {
		data.DeliveryMethod = types.StringValue(*backend.DeliveryMethod)
	}
	if backend.ServerName != nil {
		data.ServerName = types.StringValue(*backend.ServerName)
	}

	return data
}

func addL4ProxyReadDiagnostic(diags errorDiagnostics, id string, err error, context string) {
	if _, ok := err.(trafficConfigTypeMismatchError); ok {
		diags.AddError(
			"Traffic Config Type Mismatch",
			fmt.Sprintf("Unable to read L4 Proxy traffic config %q%s: %s. Import or reference a traffic config with API type %q.", id, readContextSuffix(context), err, l4ProxyTrafficConfigType),
		)
		return
	}

	diags.AddError(
		"Client Error",
		fmt.Sprintf("Unable to read L4 Proxy traffic config %q%s, got error: %s", id, readContextSuffix(context), err),
	)
}
