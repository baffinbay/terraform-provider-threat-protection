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
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the traffic configuration.",
			},
			"deployment_state": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The deployment state (DEPLOYED, UNDEPLOYED).",
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
	var data TrafficConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqData := mapModelToRequest(data)

	_, err := r.client.UpdateTrafficConfig(ctx, data.ID.ValueString(), reqData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update traffic config, got error: %s", err))
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
		reqData.Data.Attributes.Announced = data.Announced.ValueBool()
	}

	if data.Type.ValueString() == "l4Proxy" {
		reqData.Data.Attributes.Protocols = []string{"TCP"}
	}

	if data.Type.ValueString() == "httpProxy" {
		if data.ProtocolSettings != nil {
			reqData.Data.Attributes.ProtocolSettings = &client.ProtocolSettings{
				Version:          data.ProtocolSettings.Version.ValueString(),
				EnableWebsockets: data.ProtocolSettings.EnableWebsockets.ValueBool(),
				Multiplexing:     data.ProtocolSettings.Multiplexing.ValueBool(),
			}
		}

		if data.WAF != nil {
			reqData.Data.Attributes.WAF = &client.WAF{
				Enforcement:   data.WAF.Enforcement.ValueString(),
				ParanoidLevel: data.WAF.ParanoidLevel.ValueInt64(),
				CoreRuleSetID: data.WAF.CoreRuleSetID.ValueString(),
			}

			if len(data.WAF.SourceExclusions) > 0 {
				reqData.Data.Attributes.WAF.SourceExclusions.Enabled = true
				for _, src := range data.WAF.SourceExclusions {
					reqData.Data.Attributes.WAF.SourceExclusions.Sources = append(reqData.Data.Attributes.WAF.SourceExclusions.Sources, src.ValueString())
				}
			}

			if data.WAF.HttpCompliance != nil {
				for _, m := range data.WAF.HttpCompliance.AllowedMethods {
					reqData.Data.Attributes.WAF.HTTPCompliance.GlobalConfig.AllowedHttpMethods = append(reqData.Data.Attributes.WAF.HTTPCompliance.GlobalConfig.AllowedHttpMethods, m.ValueString())
				}
				for _, v := range data.WAF.HttpCompliance.AllowedVersions {
					reqData.Data.Attributes.WAF.HTTPCompliance.GlobalConfig.AllowedHttpVersions = append(reqData.Data.Attributes.WAF.HTTPCompliance.GlobalConfig.AllowedHttpVersions, v.ValueString())
				}
				if !data.WAF.HttpCompliance.ParameterLimit.IsNull() {
					reqData.Data.Attributes.WAF.HTTPCompliance.GlobalConfig.ParameterLimit.Enabled = true
					reqData.Data.Attributes.WAF.HTTPCompliance.GlobalConfig.ParameterLimit.Limit = int(data.WAF.HttpCompliance.ParameterLimit.ValueInt64())
				}
			}

			for _, excl := range data.WAF.Exclusions {
				reqData.Data.Attributes.WAF.PathExclusions = append(reqData.Data.Attributes.WAF.PathExclusions, map[string]interface{}{
					"type":        excl.Type.ValueString(),
					"value":       excl.Value.ValueString(),
					"description": excl.Description.ValueString(),
				})
			}
		}

		if data.RateLimiting != nil {
			reqData.Data.Attributes.RateLimiting = &client.RateLimit{
				Enforcement: data.RateLimiting.Enforcement.ValueString(),
			}
		}
	}

	return reqData
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
