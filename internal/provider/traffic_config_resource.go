package provider

import (
	"context"
	"fmt"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	ID               types.String           `tfsdk:"id"`
	Type             types.String           `tfsdk:"type"`
	Name             types.String           `tfsdk:"name"`
	DeploymentState  types.String           `tfsdk:"deployment_state"`
	Frontend         *FrontendModel         `tfsdk:"frontend"`
	ProtocolSettings *ProtocolSettingsModel `tfsdk:"protocol_settings"`
	WAF              *WafModel              `tfsdk:"waf"`
	RateLimiting     *RateLimitingModel     `tfsdk:"rate_limiting"`
	Prefix           types.String           `tfsdk:"prefix"`
	Announced        types.Bool             `tfsdk:"announced"`
	Backend          *BackendModel          `tfsdk:"backend"`
}

type FrontendModel struct {
	ConnectionType                types.String                   `tfsdk:"connection_type"`
	Port                          types.Int64                    `tfsdk:"port"`
	IPv4                          types.String                   `tfsdk:"ipv4"`
	IPv6                          types.String                   `tfsdk:"ipv6"`
	RedirectHttp                  types.Bool                     `tfsdk:"redirect_http"`
	Hosts                         []FrontendHostModel            `tfsdk:"hosts"`
	HSTS                          *HstsModel                     `tfsdk:"hsts"`
	ClientCertificateVerification *ClientCertificateVerification `tfsdk:"client_certificate_verification"`
}

type FrontendHostModel struct {
	Host          types.String `tfsdk:"host"`
	CertificateID types.String `tfsdk:"certificate_id"`
	TLSConfig     types.String `tfsdk:"tls_config"`
}

type HstsModel struct {
	Enabled           types.Bool  `tfsdk:"enabled"`
	MaxAge            types.Int64 `tfsdk:"max_age"`
	IncludeSubdomains types.Bool  `tfsdk:"include_subdomains"`
	Preload           types.Bool  `tfsdk:"preload"`
}

type ClientCertificateVerification struct {
	Mode             types.String   `tfsdk:"mode"`
	VerifyCrl        types.Bool     `tfsdk:"verify_crl"`
	CaCertificateIds []types.String `tfsdk:"ca_certificate_ids"`
}

type ProtocolSettingsModel struct {
	Version          types.String `tfsdk:"version"`
	EnableWebsockets types.Bool   `tfsdk:"enable_websockets"`
	Multiplexing     types.Bool   `tfsdk:"multiplexing"`
}

type BackendModel struct {
	Hosts          []BackendHostModel `tfsdk:"hosts"`
	DeliveryMethod types.String       `tfsdk:"delivery_method"`
	ServerName     types.String       `tfsdk:"server_name"`
	TLSSettings    *TlsSettingsModel  `tfsdk:"tls_settings"`
}

type TlsSettingsModel struct {
	ClientCertificateID types.String               `tfsdk:"client_certificate_id"`
	VerifyCertificate   *VerifyCertificateSettings `tfsdk:"verify_certificate"`
}

type VerifyCertificateSettings struct {
	Mode             types.String   `tfsdk:"mode"`
	CaCertificateIds []types.String `tfsdk:"ca_certificate_ids"`
	VerifyCrl        types.Bool     `tfsdk:"verify_crl"`
}

type BackendHostModel struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
}

type WafModel struct {
	Enforcement      types.String         `tfsdk:"enforcement"`
	ParanoidLevel    types.Int64          `tfsdk:"paranoid_level"`
	CoreRuleSetID    types.String         `tfsdk:"core_rule_set_id"`
	SourceExclusions []types.String       `tfsdk:"source_exclusions"`
	HttpCompliance   *HttpComplianceModel `tfsdk:"http_compliance"`
	Exclusions       []WafExclusionModel  `tfsdk:"exclusions"`
}

type HttpComplianceModel struct {
	AllowedMethods  []types.String `tfsdk:"allowed_methods"`
	AllowedVersions []types.String `tfsdk:"allowed_versions"`
	ParameterLimit  types.Int64    `tfsdk:"parameter_limit"`
}

type WafExclusionModel struct {
	Type        types.String `tfsdk:"type"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
}

type RateLimitingModel struct {
	Enforcement types.String `tfsdk:"enforcement"`
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
				Validators: []validator.String{
					stringvalidator.OneOf("l4Proxy", "routedDsr", "httpProxy"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the traffic configuration.",
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
			"frontend": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"connection_type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The connection type (SECURE, PLAINTEXT).",
					},
					"port": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "The frontend port.",
					},
					"ipv4": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The frontend IPv4 address.",
					},
					"ipv6": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "The frontend IPv6 address.",
					},
					"redirect_http": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Whether to redirect HTTP to HTTPS (SECURE only).",
					},
					"hosts": schema.ListNestedAttribute{
						Optional: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"host": schema.StringAttribute{
									Required: true,
								},
								"certificate_id": schema.StringAttribute{
									Optional: true,
								},
								"tls_config": schema.StringAttribute{
									Optional:            true,
									MarkdownDescription: "TLS configuration level (ADVANCED, INTERMEDIATE).",
								},
							},
						},
					},
					"hsts": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"enabled":            schema.BoolAttribute{Optional: true},
							"max_age":            schema.Int64Attribute{Optional: true},
							"include_subdomains": schema.BoolAttribute{Optional: true},
							"preload":            schema.BoolAttribute{Optional: true},
						},
					},
					"client_certificate_verification": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"mode":       schema.StringAttribute{Required: true, MarkdownDescription: "DISABLED, VERIFY_AND_REJECT"},
							"verify_crl": schema.BoolAttribute{Optional: true},
							"ca_certificate_ids": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
			"protocol_settings": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "HTTP protocol version (HTTP1.1, HTTP2.0).",
					},
					"enable_websockets": schema.BoolAttribute{
						Optional: true,
					},
					"multiplexing": schema.BoolAttribute{
						Optional: true,
					},
				},
			},
			"waf": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"enforcement": schema.StringAttribute{
						Required: true,
					},
					"paranoid_level": schema.Int64Attribute{
						Required: true,
					},
					"core_rule_set_id": schema.StringAttribute{
						Required: true,
					},
					"source_exclusions": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"http_compliance": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"allowed_methods": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"allowed_versions": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
							"parameter_limit": schema.Int64Attribute{
								Optional: true,
							},
						},
					},
					"exclusions": schema.ListNestedAttribute{
						Optional: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type":        schema.StringAttribute{Required: true},
								"value":       schema.StringAttribute{Required: true},
								"description": schema.StringAttribute{Optional: true},
							},
						},
					},
				},
			},
			"rate_limiting": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"enforcement": schema.StringAttribute{
						Required: true,
					},
				},
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
						Validators: []validator.String{
							stringvalidator.OneOf("ROUND_ROBIN", "LEAST_CONNECTIONS"),
						},
					},
					"server_name": schema.StringAttribute{
						Required: true,
					},
					"tls_settings": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"client_certificate_id": schema.StringAttribute{Optional: true},
							"verify_certificate": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"mode": schema.StringAttribute{Required: true, MarkdownDescription: "DISABLED, SYSTEM_TRUSTSTORE, CUSTOM_TRUSTSTORE"},
									"ca_certificate_ids": schema.ListAttribute{
										Optional:    true,
										ElementType: types.StringType,
									},
									"verify_crl": schema.BoolAttribute{Optional: true},
								},
							},
						},
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

	reqData := mapModelToRequest(data)

	tc, err := r.client.CreateTrafficConfig(ctx, reqData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create traffic config, got error: %s", err))
		return
	}

	data.ID = types.StringValue(tc.Data.ID)

	if err := r.client.WaitForTrafficConfigChange(ctx, tc.Data.ID, tc.Data.ActiveChangeID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for traffic config create, got error: %s", err))
		return
	}

	data, err = r.readTrafficConfig(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read traffic config after create, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TrafficConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TrafficConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data, err := r.readTrafficConfig(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read traffic config, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TrafficConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TrafficConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqData := mapModelToRequest(data)

	tc, err := r.client.UpdateTrafficConfig(ctx, data.ID.ValueString(), reqData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update traffic config, got error: %s", err))
		return
	}

	if err := r.client.WaitForTrafficConfigChange(ctx, data.ID.ValueString(), tc.Data.ActiveChangeID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for traffic config update, got error: %s", err))
		return
	}

	data, err = r.readTrafficConfig(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read traffic config after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapModelToRequest(data TrafficConfigResourceModel) client.TrafficConfigRequest {
	reqData := client.TrafficConfigRequest{}
	reqData.Data.Type = data.Type.ValueString()
	reqData.Data.Attributes.Name = data.Name.ValueString()
	reqData.Data.Attributes.Version = "0.1.0"

	if !data.DeploymentState.IsNull() {
		reqData.Data.Attributes.Deployment.State = data.DeploymentState.ValueString()
	} else {
		reqData.Data.Attributes.Deployment.State = "UNDEPLOYED"
	}

	if data.Frontend != nil {
		reqData.Data.Attributes.Frontend = &client.Frontend{
			Port:           data.Frontend.Port.ValueInt64(),
			ConnectionType: data.Frontend.ConnectionType.ValueString(),
			RedirectHttp:   data.Frontend.RedirectHttp.ValueBool(),
		}
		if !data.Frontend.IPv4.IsNull() {
			reqData.Data.Attributes.Frontend.IPv4 = data.Frontend.IPv4.ValueString()
			if data.Type.ValueString() != "httpProxy" {
				reqData.Data.Attributes.Frontend.IP = data.Frontend.IPv4.ValueString()
			}
		} else if data.Type.ValueString() == "httpProxy" {
			reqData.Data.Attributes.Frontend.IPv4 = ""
		}
		if !data.Frontend.IPv6.IsNull() {
			reqData.Data.Attributes.Frontend.IPv6 = data.Frontend.IPv6.ValueString()
			if data.Type.ValueString() != "httpProxy" && reqData.Data.Attributes.Frontend.IP == "" {
				reqData.Data.Attributes.Frontend.IP = data.Frontend.IPv6.ValueString()
			}
		}

		for _, host := range data.Frontend.Hosts {
			reqData.Data.Attributes.Frontend.Hosts = append(reqData.Data.Attributes.Frontend.Hosts, client.FrontendHost{
				Host:          host.Host.ValueString(),
				CertificateID: host.CertificateID.ValueString(),
				TLSConfig:     host.TLSConfig.ValueString(),
			})
		}

		if data.Frontend.HSTS != nil {
			reqData.Data.Attributes.Frontend.HTTPStrictTransportSecurity = &client.HSTS{
				Enabled:           data.Frontend.HSTS.Enabled.ValueBool(),
				MaxAge:            data.Frontend.HSTS.MaxAge.ValueInt64(),
				IncludeSubdomains: data.Frontend.HSTS.IncludeSubdomains.ValueBool(),
				Preload:           data.Frontend.HSTS.Preload.ValueBool(),
			}
		}

		if data.Frontend.ClientCertificateVerification != nil {
			reqData.Data.Attributes.Frontend.ClientCertificateVerification = &client.ClientCertificateVerification{
				Mode:      data.Frontend.ClientCertificateVerification.Mode.ValueString(),
				VerifyCrl: data.Frontend.ClientCertificateVerification.VerifyCrl.ValueBool(),
			}
			for _, id := range data.Frontend.ClientCertificateVerification.CaCertificateIds {
				reqData.Data.Attributes.Frontend.ClientCertificateVerification.CaCertificateIds = append(reqData.Data.Attributes.Frontend.ClientCertificateVerification.CaCertificateIds, id.ValueString())
			}
		}
	}

	if data.Backend != nil {
		reqData.Data.Attributes.Backend = &client.Backend{
			DeliveryMethod: data.Backend.DeliveryMethod.ValueString(),
			ServerName:     data.Backend.ServerName.ValueString(),
		}
		for _, host := range data.Backend.Hosts {
			reqData.Data.Attributes.Backend.Hosts = append(reqData.Data.Attributes.Backend.Hosts, client.Host{
				Address: host.Address.ValueString(),
				Port:    host.Port.ValueInt64(),
			})
		}
		if data.Backend.TLSSettings != nil {
			reqData.Data.Attributes.Backend.TLSSettings = &client.TLSSettings{
				ClientCertificateID: data.Backend.TLSSettings.ClientCertificateID.ValueString(),
			}
			if data.Backend.TLSSettings.VerifyCertificate != nil {
				reqData.Data.Attributes.Backend.TLSSettings.VerifyCertificate = &client.VerifyCertificate{
					Mode:      data.Backend.TLSSettings.VerifyCertificate.Mode.ValueString(),
					VerifyCrl: data.Backend.TLSSettings.VerifyCertificate.VerifyCrl.ValueBool(),
				}
				for _, id := range data.Backend.TLSSettings.VerifyCertificate.CaCertificateIds {
					reqData.Data.Attributes.Backend.TLSSettings.VerifyCertificate.CaCertificateIds = append(reqData.Data.Attributes.Backend.TLSSettings.VerifyCertificate.CaCertificateIds, id.ValueString())
				}
			}
		}
	}

	if !data.Prefix.IsNull() {
		reqData.Data.Attributes.Prefix = data.Prefix.ValueString()
	}
	if !data.Announced.IsNull() {
		announced := data.Announced.ValueBool()
		reqData.Data.Attributes.Announced = &announced
	}

	if data.Type.ValueString() == "l4Proxy" {
		reqData.Data.Attributes.Protocols = []string{"TCP"}
	}

	if data.Type.ValueString() == "httpProxy" {
		trafficRules := []any{}
		customPages := []any{}
		connectionReuseEnabled := true
		reqData.Data.Attributes.TrafficRules = &trafficRules
		reqData.Data.Attributes.CustomPages = &customPages
		reqData.Data.Attributes.ConnectionReuse = &connectionReuseEnabled
		reqData.Data.Attributes.GeoFencing = defaultHTTPProxyGeoFencingRequest()
		reqData.Data.Attributes.AllowedSources = defaultHTTPProxyAllowedSourcesRequest()
		reqData.Data.Attributes.IPBasedAccess = defaultHTTPProxyIPBasedAccessRequest()
		reqData.Data.Attributes.BotProtection = defaultHTTPProxyBotProtectionRequest()
		reqData.Data.Attributes.DataProtection = defaultHTTPProxyDataProtectionRequest()
		reqData.Data.Attributes.GatewayPath = "EXTERNAL"

		if data.ProtocolSettings != nil {
			reqData.Data.Attributes.ProtocolSettings = &client.ProtocolSettings{
				Version:          data.ProtocolSettings.Version.ValueString(),
				EnableWebsockets: data.ProtocolSettings.EnableWebsockets.ValueBool(),
				Multiplexing:     data.ProtocolSettings.Multiplexing.ValueBool(),
			}
		}

		if data.WAF != nil {
			reqData.Data.Attributes.WAF = mapWafModelToRequest(data.WAF)
		} else {
			reqData.Data.Attributes.WAF = defaultHTTPProxyWAFRequest()
		}

		if data.RateLimiting != nil {
			reqData.Data.Attributes.RateLimiting = rateLimitRequest(data.RateLimiting.Enforcement.ValueString())
		} else {
			reqData.Data.Attributes.RateLimiting = defaultHTTPProxyRateLimitRequest()
		}
	}

	return reqData
}

func mapWafModelToRequest(waf *WafModel) *client.WAF {
	reqWAF := &client.WAF{
		Enforcement:   waf.Enforcement.ValueString(),
		ParanoidLevel: waf.ParanoidLevel.ValueInt64(),
		CoreRuleSetID: waf.CoreRuleSetID.ValueString(),
	}
	reqWAF.SourceExclusions.Sources = []string{}
	reqWAF.HTTPCompliance.ResourceConfigs = []any{}
	reqWAF.PathExclusions = []any{}

	if len(waf.SourceExclusions) > 0 {
		reqWAF.SourceExclusions.Enabled = true
		for _, src := range waf.SourceExclusions {
			reqWAF.SourceExclusions.Sources = append(reqWAF.SourceExclusions.Sources, src.ValueString())
		}
	}

	if waf.HttpCompliance != nil {
		for _, m := range waf.HttpCompliance.AllowedMethods {
			reqWAF.HTTPCompliance.GlobalConfig.AllowedHttpMethods = append(reqWAF.HTTPCompliance.GlobalConfig.AllowedHttpMethods, m.ValueString())
		}
		for _, v := range waf.HttpCompliance.AllowedVersions {
			reqWAF.HTTPCompliance.GlobalConfig.AllowedHttpVersions = append(reqWAF.HTTPCompliance.GlobalConfig.AllowedHttpVersions, v.ValueString())
		}
		if !waf.HttpCompliance.ParameterLimit.IsNull() {
			reqWAF.HTTPCompliance.GlobalConfig.ParameterLimit.Enabled = true
			reqWAF.HTTPCompliance.GlobalConfig.ParameterLimit.Limit = int(waf.HttpCompliance.ParameterLimit.ValueInt64())
		}
	}

	for _, excl := range waf.Exclusions {
		reqWAF.PathExclusions = append(reqWAF.PathExclusions, map[string]interface{}{
			"type":        excl.Type.ValueString(),
			"value":       excl.Value.ValueString(),
			"description": excl.Description.ValueString(),
		})
	}

	return reqWAF
}

func defaultHTTPProxyWAFRequest() *client.WAF {
	waf := &client.WAF{
		Enforcement:    "DISABLED",
		ParanoidLevel:  1,
		CoreRuleSetID:  "4.27.0",
		PathExclusions: []any{},
	}
	waf.SourceExclusions.Sources = []string{}
	waf.HTTPCompliance.GlobalConfig.ParameterLimit.Limit = 500
	waf.HTTPCompliance.GlobalConfig.AllowedHttpMethods = []string{"GET", "POST", "DELETE", "PATCH", "PUT"}
	waf.HTTPCompliance.GlobalConfig.AllowedHttpVersions = []string{"HTTP/1.0", "HTTP/1.1", "HTTP/2.0", "HTTP/2"}
	waf.HTTPCompliance.ResourceConfigs = []any{}

	return waf
}

func defaultHTTPProxyRateLimitRequest() *client.RateLimit {
	return rateLimitRequest("BLOCK")
}

func defaultHTTPProxyGeoFencingRequest() *client.GeoFencing {
	return &client.GeoFencing{
		Type:    "BLOCK",
		Regions: []string{},
	}
}

func defaultHTTPProxyAllowedSourcesRequest() *client.AllowedSources {
	return &client.AllowedSources{
		Enforcement: "DISABLED",
		Sources:     []string{},
	}
}

func defaultHTTPProxyIPBasedAccessRequest() *client.IPBasedAccess {
	return &client.IPBasedAccess{
		DefaultPolicy: "ALLOW",
		Rules: client.IPBasedAccessRules{
			IPRanges:      []any{},
			KnownServices: []any{},
			IPLists:       []any{},
			GeoLocations:  []any{},
			ASNs:          []any{},
		},
	}
}

func defaultHTTPProxyBotProtectionRequest() *client.BotProtection {
	return &client.BotProtection{
		Strategy:      "AUTO",
		ChallengeType: "HTTP",
	}
}

func defaultHTTPProxyDataProtectionRequest() *client.DataProtection {
	return &client.DataProtection{
		LogRedaction: client.LogRedaction{
			Headers: []string{},
			Cookies: []string{},
		},
	}
}

func rateLimitRequest(enforcement string) *client.RateLimit {
	return &client.RateLimit{
		BySrcIP:       rateLimitRuleRequest(enforcement),
		BySrcIPAndURL: rateLimitRuleRequest(enforcement),
	}
}

func rateLimitRuleRequest(enforcement string) client.RateLimitRule {
	return client.RateLimitRule{
		Enforcement: enforcement,
		Rate: client.RateLimitRate{
			Value: 100,
			Unit:  "r/s",
		},
		Burst: 10,
	}
}

func (r *TrafficConfigResource) readTrafficConfig(ctx context.Context, prior TrafficConfigResourceModel) (TrafficConfigResourceModel, error) {
	tc, err := r.client.GetTrafficConfig(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}

	return mapTrafficConfigResponseToModel(tc, prior), nil
}

func mapTrafficConfigResponseToModel(tc *client.TrafficConfigResponse, prior TrafficConfigResourceModel) TrafficConfigResourceModel {
	data := prior
	hydrateUnconfigured := prior.Type.IsNull() || prior.Type.IsUnknown()

	if tc.Data.ID != "" {
		data.ID = types.StringValue(tc.Data.ID)
	}
	if tc.Data.Type != "" {
		data.Type = types.StringValue(tc.Data.Type)
	}

	attrs := tc.Data.Attributes
	if attrs.Name != nil {
		data.Name = types.StringValue(*attrs.Name)
	}
	if attrs.Deployment != nil && attrs.Deployment.State != nil {
		data.DeploymentState = types.StringValue(*attrs.Deployment.State)
	}

	data.Frontend = mapFrontendResponseToModel(attrs.Frontend, prior.Frontend, hydrateUnconfigured)
	data.Backend = mapBackendResponseToModel(attrs.Backend, prior.Backend, hydrateUnconfigured)
	data.ProtocolSettings = mapProtocolSettingsResponseToModel(attrs.ProtocolSettings, prior.ProtocolSettings, hydrateUnconfigured)
	data.WAF = mapWafResponseToModel(attrs.WAF, prior.WAF, hydrateUnconfigured)
	data.RateLimiting = mapRateLimitingResponseToModel(attrs.RateLimiting, prior.RateLimiting, hydrateUnconfigured)

	if attrs.Prefix != nil {
		data.Prefix = types.StringValue(*attrs.Prefix)
	}
	if attrs.Announced != nil {
		data.Announced = types.BoolValue(*attrs.Announced)
	}

	return data
}

func mapFrontendResponseToModel(frontend *client.TrafficConfigResponseFrontend, prior *FrontendModel, hydrateUnconfigured bool) *FrontendModel {
	if frontend == nil {
		return prior
	}
	if prior == nil && !hydrateUnconfigured {
		return nil
	}

	data := &FrontendModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}

	if frontend.ConnectionType != nil {
		data.ConnectionType = types.StringValue(*frontend.ConnectionType)
	}
	if frontend.Port != nil {
		data.Port = types.Int64Value(*frontend.Port)
	}
	if frontend.IPv4 != nil && shouldSetString(priorString(prior, func(p *FrontendModel) types.String { return p.IPv4 }), hydrateUnconfigured) {
		data.IPv4 = types.StringValue(*frontend.IPv4)
	} else if frontend.IP != nil && shouldSetString(priorString(prior, func(p *FrontendModel) types.String { return p.IPv4 }), hydrateUnconfigured) {
		data.IPv4 = types.StringValue(*frontend.IP)
	}
	if frontend.IPv6 != nil && shouldSetString(priorString(prior, func(p *FrontendModel) types.String { return p.IPv6 }), hydrateUnconfigured) {
		data.IPv6 = types.StringValue(*frontend.IPv6)
	}
	if frontend.RedirectHttp != nil && shouldSetBool(priorBool(prior, func(p *FrontendModel) types.Bool { return p.RedirectHttp }), hydrateUnconfigured) {
		data.RedirectHttp = types.BoolValue(*frontend.RedirectHttp)
	}

	if frontend.Hosts != nil && (hydrateUnconfigured || prior != nil && prior.Hosts != nil) {
		data.Hosts = make([]FrontendHostModel, 0, len(frontend.Hosts))
		for _, host := range frontend.Hosts {
			data.Hosts = append(data.Hosts, FrontendHostModel{
				Host:          optionalStringPtrValue(host.Host),
				CertificateID: optionalStringPtrValue(host.CertificateID),
				TLSConfig:     optionalStringPtrValue(host.TLSConfig),
			})
		}
	}

	if frontend.HTTPStrictTransportSecurity != nil && (hydrateUnconfigured || prior != nil && prior.HSTS != nil) {
		hsts := &HstsModel{}
		if prior != nil && prior.HSTS != nil {
			priorHSTS := *prior.HSTS
			hsts = &priorHSTS
		}
		if frontend.HTTPStrictTransportSecurity.Enabled != nil && shouldSetBool(priorBool(hsts, func(p *HstsModel) types.Bool { return p.Enabled }), hydrateUnconfigured) {
			hsts.Enabled = types.BoolValue(*frontend.HTTPStrictTransportSecurity.Enabled)
		}
		if frontend.HTTPStrictTransportSecurity.MaxAge != nil && shouldSetInt64(priorInt64(hsts, func(p *HstsModel) types.Int64 { return p.MaxAge }), hydrateUnconfigured) {
			hsts.MaxAge = types.Int64Value(*frontend.HTTPStrictTransportSecurity.MaxAge)
		}
		if frontend.HTTPStrictTransportSecurity.IncludeSubdomains != nil && shouldSetBool(priorBool(hsts, func(p *HstsModel) types.Bool { return p.IncludeSubdomains }), hydrateUnconfigured) {
			hsts.IncludeSubdomains = types.BoolValue(*frontend.HTTPStrictTransportSecurity.IncludeSubdomains)
		}
		if frontend.HTTPStrictTransportSecurity.Preload != nil && shouldSetBool(priorBool(hsts, func(p *HstsModel) types.Bool { return p.Preload }), hydrateUnconfigured) {
			hsts.Preload = types.BoolValue(*frontend.HTTPStrictTransportSecurity.Preload)
		}
		data.HSTS = hsts
	}

	if frontend.ClientCertificateVerification != nil && (hydrateUnconfigured || prior != nil && prior.ClientCertificateVerification != nil) {
		clientCertificateVerification := &ClientCertificateVerification{}
		if prior != nil && prior.ClientCertificateVerification != nil {
			priorClientCertificateVerification := *prior.ClientCertificateVerification
			clientCertificateVerification = &priorClientCertificateVerification
		}
		if frontend.ClientCertificateVerification.Mode != nil {
			clientCertificateVerification.Mode = types.StringValue(*frontend.ClientCertificateVerification.Mode)
		}
		if frontend.ClientCertificateVerification.VerifyCrl != nil && shouldSetBool(priorBool(clientCertificateVerification, func(p *ClientCertificateVerification) types.Bool {
			return p.VerifyCrl
		}), hydrateUnconfigured) {
			clientCertificateVerification.VerifyCrl = types.BoolValue(*frontend.ClientCertificateVerification.VerifyCrl)
		}
		if frontend.ClientCertificateVerification.CaCertificateIds != nil && (hydrateUnconfigured || clientCertificateVerification.CaCertificateIds != nil) {
			clientCertificateVerification.CaCertificateIds = stringSliceToTypeValues(frontend.ClientCertificateVerification.CaCertificateIds)
		}
		data.ClientCertificateVerification = clientCertificateVerification
	}

	return data
}

func mapBackendResponseToModel(backend *client.TrafficConfigResponseBackend, prior *BackendModel, hydrateUnconfigured bool) *BackendModel {
	if backend == nil {
		return prior
	}
	if prior == nil && !hydrateUnconfigured {
		return nil
	}

	data := &BackendModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}

	if backend.Hosts != nil {
		data.Hosts = make([]BackendHostModel, 0, len(backend.Hosts))
		for _, host := range backend.Hosts {
			data.Hosts = append(data.Hosts, BackendHostModel{
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

	if backend.TLSSettings != nil && (hydrateUnconfigured || prior != nil && prior.TLSSettings != nil) {
		tlsSettings := &TlsSettingsModel{}
		if prior != nil && prior.TLSSettings != nil {
			priorTLSSettings := *prior.TLSSettings
			tlsSettings = &priorTLSSettings
		}
		if backend.TLSSettings.ClientCertificateID != nil && shouldSetString(priorString(tlsSettings, func(p *TlsSettingsModel) types.String {
			return p.ClientCertificateID
		}), hydrateUnconfigured) {
			tlsSettings.ClientCertificateID = optionalStringPtrValue(backend.TLSSettings.ClientCertificateID)
		}
		if backend.TLSSettings.VerifyCertificate != nil && (hydrateUnconfigured || prior != nil && prior.TLSSettings != nil && prior.TLSSettings.VerifyCertificate != nil) {
			verifyCertificate := &VerifyCertificateSettings{}
			if prior != nil && prior.TLSSettings != nil && prior.TLSSettings.VerifyCertificate != nil {
				priorVerifyCertificate := *prior.TLSSettings.VerifyCertificate
				verifyCertificate = &priorVerifyCertificate
			}
			if backend.TLSSettings.VerifyCertificate.Mode != nil {
				verifyCertificate.Mode = types.StringValue(*backend.TLSSettings.VerifyCertificate.Mode)
			}
			if backend.TLSSettings.VerifyCertificate.CaCertificateIds != nil && (hydrateUnconfigured || verifyCertificate.CaCertificateIds != nil) {
				verifyCertificate.CaCertificateIds = stringSliceToTypeValues(backend.TLSSettings.VerifyCertificate.CaCertificateIds)
			}
			if backend.TLSSettings.VerifyCertificate.VerifyCrl != nil && shouldSetBool(priorBool(verifyCertificate, func(p *VerifyCertificateSettings) types.Bool {
				return p.VerifyCrl
			}), hydrateUnconfigured) {
				verifyCertificate.VerifyCrl = types.BoolValue(*backend.TLSSettings.VerifyCertificate.VerifyCrl)
			}
			tlsSettings.VerifyCertificate = verifyCertificate
		}
		data.TLSSettings = tlsSettings
	}

	return data
}

func mapProtocolSettingsResponseToModel(settings *client.TrafficConfigResponseProtocolSettings, prior *ProtocolSettingsModel, hydrateUnconfigured bool) *ProtocolSettingsModel {
	if settings == nil {
		return prior
	}
	if prior == nil && !hydrateUnconfigured {
		return nil
	}

	data := &ProtocolSettingsModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}

	if settings.Version != nil {
		data.Version = types.StringValue(*settings.Version)
	}
	if settings.EnableWebsockets != nil && shouldSetBool(priorBool(prior, func(p *ProtocolSettingsModel) types.Bool {
		return p.EnableWebsockets
	}), hydrateUnconfigured) {
		data.EnableWebsockets = types.BoolValue(*settings.EnableWebsockets)
	}
	if settings.Multiplexing != nil && shouldSetBool(priorBool(prior, func(p *ProtocolSettingsModel) types.Bool {
		return p.Multiplexing
	}), hydrateUnconfigured) {
		data.Multiplexing = types.BoolValue(*settings.Multiplexing)
	}

	return data
}

func mapWafResponseToModel(waf *client.TrafficConfigResponseWAF, prior *WafModel, hydrateUnconfigured bool) *WafModel {
	if waf == nil {
		return prior
	}
	if prior == nil && !hydrateUnconfigured {
		return nil
	}

	data := &WafModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}

	if waf.Enforcement != nil {
		data.Enforcement = types.StringValue(*waf.Enforcement)
	}
	if waf.ParanoidLevel != nil {
		data.ParanoidLevel = types.Int64Value(*waf.ParanoidLevel)
	}
	if waf.CoreRuleSetID != nil {
		data.CoreRuleSetID = types.StringValue(*waf.CoreRuleSetID)
	} else if waf.CoreRuleSet != nil && waf.CoreRuleSet.Version != nil {
		data.CoreRuleSetID = types.StringValue(*waf.CoreRuleSet.Version)
	}
	if waf.SourceExclusions != nil && waf.SourceExclusions.Sources != nil && (hydrateUnconfigured || prior != nil && prior.SourceExclusions != nil) {
		data.SourceExclusions = stringSliceToTypeValues(waf.SourceExclusions.Sources)
	}

	if waf.HTTPCompliance != nil && (hydrateUnconfigured || prior != nil && prior.HttpCompliance != nil) {
		httpCompliance := &HttpComplianceModel{}
		if prior != nil && prior.HttpCompliance != nil {
			priorHTTPCompliance := *prior.HttpCompliance
			httpCompliance = &priorHTTPCompliance
		}
		if waf.HTTPCompliance.GlobalConfig != nil {
			globalConfig := waf.HTTPCompliance.GlobalConfig
			if globalConfig.AllowedHTTPMethods != nil && (hydrateUnconfigured || httpCompliance.AllowedMethods != nil) {
				httpCompliance.AllowedMethods = stringSliceToTypeValues(globalConfig.AllowedHTTPMethods)
			}
			if globalConfig.AllowedHTTPVersions != nil && (hydrateUnconfigured || httpCompliance.AllowedVersions != nil) {
				httpCompliance.AllowedVersions = stringSliceToTypeValues(globalConfig.AllowedHTTPVersions)
			}
			if globalConfig.ParameterLimit != nil && shouldSetInt64(httpCompliance.ParameterLimit, hydrateUnconfigured) {
				httpCompliance.ParameterLimit = types.Int64Null()
				if globalConfig.ParameterLimit.Enabled != nil && *globalConfig.ParameterLimit.Enabled {
					if globalConfig.ParameterLimit.Limit != nil {
						httpCompliance.ParameterLimit = types.Int64Value(int64(*globalConfig.ParameterLimit.Limit))
					} else {
						httpCompliance.ParameterLimit = types.Int64Value(0)
					}
				}
			}
		}
		data.HttpCompliance = httpCompliance
	}

	if waf.PathExclusions != nil && (hydrateUnconfigured || prior != nil && prior.Exclusions != nil) {
		data.Exclusions = make([]WafExclusionModel, 0, len(waf.PathExclusions))
		for _, exclusion := range waf.PathExclusions {
			mapped := WafExclusionModel{
				Type:        optionalStringPtrValue(exclusion.Type),
				Value:       optionalStringPtrValue(exclusion.Value),
				Description: optionalStringPtrValue(exclusion.Description),
			}
			if mapped.Type.IsNull() && exclusion.Match != nil {
				mapped.Type = types.StringValue("path")
			}
			if mapped.Value.IsNull() && exclusion.Match != nil {
				mapped.Value = types.StringValue(*exclusion.Match)
			}
			data.Exclusions = append(data.Exclusions, mapped)
		}
	}

	return data
}

func mapRateLimitingResponseToModel(rateLimiting *client.TrafficConfigResponseRateLimit, prior *RateLimitingModel, hydrateUnconfigured bool) *RateLimitingModel {
	if rateLimiting == nil {
		return prior
	}
	if prior == nil && !hydrateUnconfigured {
		return nil
	}

	data := &RateLimitingModel{}
	if prior != nil {
		priorCopy := *prior
		data = &priorCopy
	}
	if rateLimiting.Enforcement != nil {
		data.Enforcement = types.StringValue(*rateLimiting.Enforcement)
	} else if rateLimiting.BySrcIP != nil && rateLimiting.BySrcIP.Enforcement != nil {
		data.Enforcement = types.StringValue(*rateLimiting.BySrcIP.Enforcement)
	} else if rateLimiting.BySrcIPAndURL != nil && rateLimiting.BySrcIPAndURL.Enforcement != nil {
		data.Enforcement = types.StringValue(*rateLimiting.BySrcIPAndURL.Enforcement)
	}

	return data
}

func priorString[T any](prior *T, get func(*T) types.String) types.String {
	if prior == nil {
		return types.StringNull()
	}

	return get(prior)
}

func priorBool[T any](prior *T, get func(*T) types.Bool) types.Bool {
	if prior == nil {
		return types.BoolNull()
	}

	return get(prior)
}

func priorInt64[T any](prior *T, get func(*T) types.Int64) types.Int64 {
	if prior == nil {
		return types.Int64Null()
	}

	return get(prior)
}

func shouldSetString(value types.String, hydrateUnconfigured bool) bool {
	return hydrateUnconfigured || (!value.IsNull() && !value.IsUnknown())
}

func shouldSetBool(value types.Bool, hydrateUnconfigured bool) bool {
	return hydrateUnconfigured || (!value.IsNull() && !value.IsUnknown())
}

func shouldSetInt64(value types.Int64, hydrateUnconfigured bool) bool {
	return hydrateUnconfigured || (!value.IsNull() && !value.IsUnknown())
}

func optionalStringPtrValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func optionalInt64PtrValue(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func stringSliceToTypeValues(values []string) []types.String {
	result := make([]types.String, 0, len(values))
	for _, value := range values {
		result = append(result, types.StringValue(value))
	}

	return result
}

func (r *TrafficConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TrafficConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTrafficConfig(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete traffic config, got error: %s", err))
		return
	}
}

func (r *TrafficConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
