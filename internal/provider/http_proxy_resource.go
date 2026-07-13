package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/baffinbay/terraform-provider-baffinbay/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	httpProxyTrafficConfigType    = "httpProxy"
	httpProxyTrafficConfigVersion = "0.1.0"
	httpProxyDefaultCRSVersion    = "4.*.*"
	httpProxyDefaultHSTSMaxAge    = int64(14_515_200)
)

var httpProxyCRSVersionPattern = regexp.MustCompile(`^[0-9]+\.(?:[0-9]+\.(?:[0-9]+|\*)|\*\.\*)$`)

var _ resource.Resource = &HTTPProxyResource{}
var _ resource.ResourceWithImportState = &HTTPProxyResource{}
var _ resource.ResourceWithModifyPlan = &HTTPProxyResource{}
var _ resource.ResourceWithValidateConfig = &HTTPProxyResource{}

func NewHTTPProxyResource() resource.Resource {
	return &HTTPProxyResource{}
}

type HTTPProxyResource struct {
	client *client.Client
}

type HTTPProxyResourceModel struct {
	ID                   types.String                    `tfsdk:"id"`
	Name                 types.String                    `tfsdk:"name"`
	DeploymentState      types.String                    `tfsdk:"deployment_state"`
	ConnectionReuse      types.Bool                      `tfsdk:"connection_reuse_enabled"`
	Frontend             *HTTPProxyFrontendModel         `tfsdk:"frontend"`
	Backend              *HTTPProxyBackendModel          `tfsdk:"backend"`
	ProtocolSettings     *HTTPProxyProtocolSettingsModel `tfsdk:"protocol_settings"`
	BotProtection        *HTTPProxyBotProtectionModel    `tfsdk:"bot_protection"`
	DataProtection       *HTTPProxyDataProtectionModel   `tfsdk:"data_protection"`
	CustomPages          []HTTPProxyCustomPageModel      `tfsdk:"custom_pages"`
	WAF                  *HTTPProxyWAFModel              `tfsdk:"waf"`
	RateLimiting         *HTTPProxyRateLimitingModel     `tfsdk:"rate_limiting"`
	IPBasedAccessControl *HTTPProxyIPAccessControlModel  `tfsdk:"ip_based_access_control"`
	TrafficRules         []HTTPProxyTrafficRuleModel     `tfsdk:"traffic_rules"`
}

type HTTPProxyFrontendModel struct {
	ConnectionType                types.String                                 `tfsdk:"connection_type"`
	IPv4                          types.String                                 `tfsdk:"ipv4"`
	IPv6                          types.String                                 `tfsdk:"ipv6"`
	Port                          types.Int64                                  `tfsdk:"port"`
	RedirectHTTP                  types.Bool                                   `tfsdk:"redirect_http"`
	Hosts                         []HTTPProxyFrontendHostModel                 `tfsdk:"hosts"`
	HSTS                          *HTTPProxyHSTSModel                          `tfsdk:"hsts"`
	ClientCertificateVerification *HTTPProxyClientCertificateVerificationModel `tfsdk:"client_certificate_verification"`
}

type HTTPProxyFrontendHostModel struct {
	Host          types.String `tfsdk:"host"`
	CertificateID types.String `tfsdk:"certificate_id"`
	TLSConfig     types.String `tfsdk:"tls_config"`
}

type HTTPProxyHSTSModel struct {
	Enabled           types.Bool  `tfsdk:"enabled"`
	MaxAge            types.Int64 `tfsdk:"max_age"`
	IncludeSubdomains types.Bool  `tfsdk:"include_subdomains"`
	Preload           types.Bool  `tfsdk:"preload"`
}

type HTTPProxyClientCertificateVerificationModel struct {
	Mode             types.String   `tfsdk:"mode"`
	CACertificateIDs []types.String `tfsdk:"ca_certificate_ids"`
}

type HTTPProxyBackendModel struct {
	Hosts          []HTTPProxyBackendHostModel `tfsdk:"hosts"`
	DeliveryMethod types.String                `tfsdk:"delivery_method"`
	ServerName     types.String                `tfsdk:"server_name"`
	TLSSettings    *HTTPProxyTLSSettingsModel  `tfsdk:"tls_settings"`
}

type HTTPProxyBackendHostModel struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
}

type HTTPProxyTLSSettingsModel struct {
	ClientCertificateID types.String                     `tfsdk:"client_certificate_id"`
	VerifyCertificate   *HTTPProxyVerifyCertificateModel `tfsdk:"verify_certificate"`
}

type HTTPProxyVerifyCertificateModel struct {
	Mode             types.String   `tfsdk:"mode"`
	CACertificateIDs []types.String `tfsdk:"ca_certificate_ids"`
}

type HTTPProxyProtocolSettingsModel struct {
	Version          types.String `tfsdk:"version"`
	EnableWebsockets types.Bool   `tfsdk:"enable_websockets"`
}

type HTTPProxyBotProtectionModel struct {
	Strategy      types.String `tfsdk:"strategy"`
	ChallengeType types.String `tfsdk:"challenge_type"`
}

type HTTPProxyDataProtectionModel struct {
	LogRedaction *HTTPProxyLogRedactionModel `tfsdk:"log_redaction"`
}

type HTTPProxyLogRedactionModel struct {
	Headers []types.String `tfsdk:"headers"`
	Cookies []types.String `tfsdk:"cookies"`
}

type HTTPProxyCustomPageModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

type HTTPProxyRateLimitingModel struct {
	BySourceIP       *HTTPProxyRateLimitRuleModel `tfsdk:"by_source_ip"`
	BySourceIPAndURL *HTTPProxyRateLimitRuleModel `tfsdk:"by_source_ip_and_url"`
	Exclusions       []types.String               `tfsdk:"exclusions"`
}

type HTTPProxyRateLimitRuleModel struct {
	Enforcement types.String                 `tfsdk:"enforcement"`
	Rate        *HTTPProxyRateLimitRateModel `tfsdk:"rate"`
	Burst       types.Int64                  `tfsdk:"burst"`
}

type HTTPProxyRateLimitRateModel struct {
	Value types.Int64  `tfsdk:"value"`
	Unit  types.String `tfsdk:"unit"`
}

type HTTPProxyIPAccessControlModel struct {
	DefaultPolicy types.String                        `tfsdk:"default_policy"`
	Rules         *HTTPProxyIPAccessControlRulesModel `tfsdk:"rules"`
}

type HTTPProxyIPAccessControlRulesModel struct {
	IPRanges      []HTTPProxyIPRangeRuleModel      `tfsdk:"ip_ranges"`
	IPLists       []HTTPProxyIPListRuleModel       `tfsdk:"ip_lists"`
	KnownServices []HTTPProxyKnownServiceRuleModel `tfsdk:"known_services"`
	GeoLocations  []HTTPProxyGeoLocationRuleModel  `tfsdk:"geo_locations"`
	ASNs          []HTTPProxyASNRuleModel          `tfsdk:"asns"`
}

type HTTPProxyIPRangeRuleModel struct {
	Policy              types.String `tfsdk:"policy"`
	Address             types.String `tfsdk:"address"`
	Note                types.String `tfsdk:"note"`
	BypassBotProtection types.Bool   `tfsdk:"bypass_bot_protection"`
}

type HTTPProxyIPListRuleModel struct {
	Policy types.String `tfsdk:"policy"`
	ID     types.String `tfsdk:"id"`
}

type HTTPProxyKnownServiceRuleModel struct {
	Policy              types.String `tfsdk:"policy"`
	ID                  types.String `tfsdk:"id"`
	Note                types.String `tfsdk:"note"`
	BypassBotProtection types.Bool   `tfsdk:"bypass_bot_protection"`
}

type HTTPProxyGeoLocationRuleModel struct {
	Policy types.String `tfsdk:"policy"`
	Region types.String `tfsdk:"region"`
	Note   types.String `tfsdk:"note"`
}

type HTTPProxyASNRuleModel struct {
	Policy types.String `tfsdk:"policy"`
	ASN    types.Int64  `tfsdk:"asn"`
	Note   types.String `tfsdk:"note"`
}

type HTTPProxyWAFModel struct {
	Enforcement        types.String                       `tfsdk:"enforcement"`
	ParanoiaLevel      types.Int64                        `tfsdk:"paranoia_level"`
	CoreRuleSetVersion types.String                       `tfsdk:"core_rule_set_version"`
	MatchedDataEnabled types.Bool                         `tfsdk:"matched_data_enabled"`
	SourceExclusions   *HTTPProxySourceExclusionsModel    `tfsdk:"source_exclusions"`
	HTTPCompliance     *HTTPProxyHTTPComplianceModel      `tfsdk:"http_compliance"`
	PathExclusions     []HTTPProxyWAFPathExclusionModel   `tfsdk:"path_exclusions"`
	CookieExclusions   []HTTPProxyWAFCookieExclusionModel `tfsdk:"cookie_exclusions"`
	StagedWAF          *HTTPProxyStagedWAFModel           `tfsdk:"staged_waf"`
}

type HTTPProxySourceExclusionsModel struct {
	Enabled types.Bool     `tfsdk:"enabled"`
	Sources []types.String `tfsdk:"sources"`
}

type HTTPProxyHTTPComplianceModel struct {
	ParameterLimit  *HTTPProxyParameterLimitModel                `tfsdk:"parameter_limit"`
	AllowedMethods  []types.String                               `tfsdk:"allowed_methods"`
	AllowedVersions []types.String                               `tfsdk:"allowed_versions"`
	ResourceConfigs []HTTPProxyHTTPComplianceResourceConfigModel `tfsdk:"resource_configs"`
}

type HTTPProxyParameterLimitModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Limit   types.Int64 `tfsdk:"limit"`
}

type HTTPProxyHTTPComplianceResourceConfigModel struct {
	Matches                      []types.String `tfsdk:"matches"`
	ParseJSONEnabled             types.Bool     `tfsdk:"parse_json_enabled"`
	ParseXMLEnabled              types.Bool     `tfsdk:"parse_xml_enabled"`
	ParseMultipartRequestEnabled types.Bool     `tfsdk:"parse_multipart_request_enabled"`
}

type HTTPProxyWAFPathExclusionModel struct {
	Match      types.String   `tfsdk:"match"`
	DisableAll types.Bool     `tfsdk:"disable_all"`
	RuleIDs    []types.String `tfsdk:"rule_ids"`
}

type HTTPProxyWAFCookieExclusionModel struct {
	CookieName      types.String   `tfsdk:"cookie_name"`
	ExcludeAllRules types.Bool     `tfsdk:"exclude_all_rules"`
	RuleIDs         []types.String `tfsdk:"rule_ids"`
}

type HTTPProxyStagedWAFModel struct {
	State              types.String                       `tfsdk:"state"`
	Mode               types.String                       `tfsdk:"mode"`
	ParanoiaLevel      types.Int64                        `tfsdk:"paranoia_level"`
	CoreRuleSetVersion types.String                       `tfsdk:"core_rule_set_version"`
	PathExclusions     []HTTPProxyWAFPathExclusionModel   `tfsdk:"path_exclusions"`
	CookieExclusions   []HTTPProxyWAFCookieExclusionModel `tfsdk:"cookie_exclusions"`
}

type HTTPProxyTrafficRuleModel struct {
	Name               types.String                             `tfsdk:"name"`
	MatchingConditions []HTTPProxyTrafficMatchingConditionModel `tfsdk:"matching_conditions"`
	Actions            *HTTPProxyTrafficRuleActionsModel        `tfsdk:"actions"`
}

type HTTPProxyTrafficMatchingConditionModel struct {
	Paths []types.String                      `tfsdk:"paths"`
	Hosts *HTTPProxyTrafficRuleHostMatchModel `tfsdk:"hosts"`
}

type HTTPProxyTrafficRuleHostMatchModel struct {
	Type   types.String   `tfsdk:"type"`
	Values []types.String `tfsdk:"values"`
}

type HTTPProxyTrafficRuleActionsModel struct {
	Backends    []HTTPProxyBackendHostModel     `tfsdk:"backends"`
	Headers     []HTTPProxyHeaderModel          `tfsdk:"headers"`
	HostHeader  types.String                    `tfsdk:"host_header"`
	Redirect    *HTTPProxyRedirectModel         `tfsdk:"redirect"`
	RateLimit   *HTTPProxyTrafficRateLimitModel `tfsdk:"rate_limit"`
	MaxBodySize *HTTPProxyMaximumBodySizeModel  `tfsdk:"max_body_size"`
}

type HTTPProxyHeaderModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type HTTPProxyRedirectModel struct {
	URL                types.String `tfsdk:"url"`
	StatusCode         types.Int64  `tfsdk:"status_code"`
	AppendOriginalPath types.Bool   `tfsdk:"append_original_path"`
}

type HTTPProxyTrafficRateLimitModel struct {
	BySourceIP       *HTTPProxyRateLimitRuleModel `tfsdk:"by_source_ip"`
	BySourceIPAndURL *HTTPProxyRateLimitRuleModel `tfsdk:"by_source_ip_and_url"`
}

type HTTPProxyMaximumBodySizeModel struct {
	Enforcement types.String `tfsdk:"enforcement"`
	ValueBytes  types.Int64  `tfsdk:"value_bytes"`
}

func (r *HTTPProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_proxy"
}

func (r *HTTPProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Baffin Bay HTTP Proxy traffic configuration.",
		Attributes: map[string]schema.Attribute{
			"id":                       schema.StringAttribute{Computed: true, MarkdownDescription: "The HTTP Proxy traffic configuration ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":                     schema.StringAttribute{Required: true, MarkdownDescription: "The HTTP Proxy traffic configuration name."},
			"deployment_state":         schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("UNDEPLOYED"), Validators: []validator.String{stringvalidator.OneOf("DEPLOYED", "UNDEPLOYED")}},
			"connection_reuse_enabled": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"frontend":                 httpProxyFrontendAttribute(),
			"backend":                  httpProxyBackendAttribute(),
			"protocol_settings":        httpProxyProtocolSettingsAttribute(),
			"bot_protection":           httpProxyBotProtectionAttribute(),
			"data_protection":          httpProxyDataProtectionAttribute(),
			"custom_pages":             httpProxyCustomPagesAttribute(),
			"waf":                      httpProxyWAFAttribute(),
			"rate_limiting":            httpProxyRateLimitingAttribute(),
			"ip_based_access_control":  httpProxyIPAccessControlAttribute(),
			"traffic_rules":            httpProxyTrafficRulesAttribute(),
		},
	}
}

func httpProxyFrontendAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Required: true,
		Attributes: map[string]schema.Attribute{
			"connection_type": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("SECURE", "PLAINTEXT")}},
			"ipv4":            schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.AtLeastOneOf(path.Expressions{path.MatchRelative().AtParent().AtName("ipv6")}...)}},
			"ipv6":            schema.StringAttribute{Optional: true},
			"port":            schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 65535)}},
			"redirect_http":   schema.BoolAttribute{Optional: true, Computed: true},
			"hosts": schema.ListNestedAttribute{Required: true, Validators: []validator.List{listvalidator.SizeAtLeast(1)}, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"host":           schema.StringAttribute{Required: true},
				"certificate_id": schema.StringAttribute{Optional: true},
				"tls_config":     schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.OneOf("ADVANCED", "INTERMEDIATE")}},
			}}},
			"hsts": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
				"enabled":            schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
				"max_age":            schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(httpProxyDefaultHSTSMaxAge), Validators: []validator.Int64{int64validator.Between(1, 2_147_483_647)}},
				"include_subdomains": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
				"preload":            schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			}},
			"client_certificate_verification": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
				"mode":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("DISABLED"), Validators: []validator.String{stringvalidator.OneOf("DISABLED", "VERIFY_AND_REJECT")}},
				"ca_certificate_ids": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
			}},
		},
	}
}

func httpProxyBackendAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Required: true,
		Attributes: map[string]schema.Attribute{
			"hosts": schema.ListNestedAttribute{Required: true, Validators: []validator.List{listvalidator.SizeAtLeast(1)}, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"address": schema.StringAttribute{Required: true},
				"port":    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 65535)}},
			}}},
			"delivery_method": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("LEAST_CONNECTIONS"), Validators: []validator.String{stringvalidator.OneOf("ROUND_ROBIN", "LEAST_CONNECTIONS", "IP_HASH")}},
			"server_name":     schema.StringAttribute{Optional: true},
			"tls_settings": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
				"client_certificate_id": schema.StringAttribute{Optional: true},
				"verify_certificate": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
					"mode":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("SYSTEM_TRUSTSTORE"), Validators: []validator.String{stringvalidator.OneOf("DISABLED", "SYSTEM_TRUSTSTORE", "CUSTOM_TRUSTSTORE")}},
					"ca_certificate_ids": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
				}},
			}},
		},
	}
}

func httpProxyProtocolSettingsAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Required: true, Attributes: map[string]schema.Attribute{
		"version":           schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("HTTP1.1", "HTTP2.0")}},
		"enable_websockets": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
	}}
}

func httpProxyBotProtectionAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
		"strategy":       schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("AUTO"), Validators: []validator.String{stringvalidator.OneOf("ALWAYS_ON", "AUTO", "DISABLED")}},
		"challenge_type": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("JS"), Validators: []validator.String{stringvalidator.OneOf("HTTP", "JS")}},
	}}
}

func httpProxyDataProtectionAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
		"log_redaction": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
			"headers": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
			"cookies": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
		}},
	}}
}

func httpProxyCustomPagesAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"id":   schema.StringAttribute{Required: true},
		"type": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("WAF_BLOCK", "429_RATE_LIMIT_BLOCK", "500_INTERNAL_ERROR", "502_NETWORK_ERROR", "503_UPSTREAM_SERVICE_UNAVAILABLE", "504_UPSTREAM_TIMEOUT")}},
	}}}
}

func httpProxyRateLimitingAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
		"by_source_ip":         httpProxyRateLimitRuleAttribute(true),
		"by_source_ip_and_url": httpProxyRateLimitRuleAttribute(true),
		"exclusions":           schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
	}}
}

func httpProxyRateLimitRuleAttribute(computed bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Optional: true, Computed: computed, Attributes: map[string]schema.Attribute{
		"enforcement": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("BLOCK"), Validators: []validator.String{stringvalidator.OneOf("BLOCK", "DISABLED")}},
		"rate": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
			"value": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(10), Validators: []validator.Int64{int64validator.Between(1, 10000)}},
			"unit":  schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("r/s"), Validators: []validator.String{stringvalidator.OneOf("r/s", "r/m")}},
		}},
		"burst": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(100), Validators: []validator.Int64{int64validator.Between(1, 10000)}},
	}}
}

func httpProxyIPAccessControlAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
		"default_policy": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("ALLOW"), Validators: []validator.String{stringvalidator.OneOf("ALLOW", "BLOCK")}},
		"rules": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
			"ip_ranges": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: httpProxyAccessRuleAttributes("address", true)}},
			"ip_lists": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"policy": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("ALLOW", "BLOCK")}},
				"id":     schema.StringAttribute{Required: true},
			}}},
			"known_services": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: httpProxyAccessRuleAttributes("id", true)}},
			"geo_locations":  schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: httpProxyAccessRuleAttributes("region", false)}},
			"asns": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"policy": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("ALLOW", "BLOCK")}},
				"asn":    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(0, 4_294_967_295)}},
				"note":   schema.StringAttribute{Optional: true},
			}}},
		}},
	}}
}

func httpProxyAccessRuleAttributes(valueName string, bypass bool) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"policy":  schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("ALLOW", "BLOCK")}},
		valueName: schema.StringAttribute{Required: true},
		"note":    schema.StringAttribute{Optional: true},
	}
	if bypass {
		attrs["bypass_bot_protection"] = schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)}
	}
	return attrs
}

func httpProxyWAFAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
		"enforcement":           schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("LOG"), Validators: []validator.String{stringvalidator.OneOf("LOG", "BLOCK", "DISABLED")}},
		"paranoia_level":        schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1), Validators: []validator.Int64{int64validator.Between(1, 4)}},
		"core_rule_set_version": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(httpProxyDefaultCRSVersion), Validators: []validator.String{stringvalidator.RegexMatches(httpProxyCRSVersionPattern, "must be a three-part CRS version selector such as 4.22.0, 4.22.*, or 4.*.*")}},
		"matched_data_enabled":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"source_exclusions": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"sources": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
		}},
		"http_compliance": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
			"parameter_limit": schema.SingleNestedAttribute{Optional: true, Computed: true, Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
				"limit":   schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(500), Validators: []validator.Int64{int64validator.Between(1, 10000)}},
			}},
			"allowed_methods":  schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
			"allowed_versions": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.SizeAtLeast(1), listvalidator.UniqueValues(), listvalidator.ValueStringsAre(stringvalidator.OneOf("HTTP/1.0", "HTTP/1.1", "HTTP/2.0", "HTTP/2"))}},
			"resource_configs": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"matches":                         schema.ListAttribute{Required: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.SizeAtLeast(1), listvalidator.UniqueValues()}},
				"parse_json_enabled":              schema.BoolAttribute{Required: true},
				"parse_xml_enabled":               schema.BoolAttribute{Required: true},
				"parse_multipart_request_enabled": schema.BoolAttribute{Required: true},
			}}},
		}},
		"path_exclusions":   httpProxyWAFPathExclusionsAttribute(true),
		"cookie_exclusions": httpProxyWAFCookieExclusionsAttribute(true),
		"staged_waf": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
			"state":                 schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("ENABLED", "DISABLED")}},
			"mode":                  schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("PARANOIA_LEVEL", "CRS_VERSION")}},
			"paranoia_level":        schema.Int64Attribute{Optional: true, Validators: []validator.Int64{int64validator.Between(1, 4)}},
			"core_rule_set_version": schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.RegexMatches(httpProxyCRSVersionPattern, "must be a three-part CRS version selector such as 4.22.0, 4.22.*, or 4.*.*")}},
			"path_exclusions":       httpProxyWAFPathExclusionsAttribute(false),
			"cookie_exclusions":     httpProxyWAFCookieExclusionsAttribute(false),
		}},
	}}
}

func httpProxyWAFPathExclusionsAttribute(computed bool) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{Optional: true, Computed: computed, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"match":       schema.StringAttribute{Required: true},
		"disable_all": schema.BoolAttribute{Required: true},
		"rule_ids":    schema.ListAttribute{Required: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
	}}}
}

func httpProxyWAFCookieExclusionsAttribute(computed bool) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{Optional: true, Computed: computed, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"cookie_name":       schema.StringAttribute{Required: true},
		"exclude_all_rules": schema.BoolAttribute{Required: true},
		"rule_ids":          schema.ListAttribute{Required: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
	}}}
}

func httpProxyTrafficRulesAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true},
		"matching_conditions": schema.ListNestedAttribute{Required: true, Validators: []validator.List{listvalidator.SizeAtLeast(1)}, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"paths": schema.ListAttribute{Required: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.SizeAtLeast(1), listvalidator.UniqueValues()}},
			"hosts": schema.SingleNestedAttribute{Required: true, Attributes: map[string]schema.Attribute{
				"type":   schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("ALL", "SELECT")}},
				"values": schema.ListAttribute{Optional: true, ElementType: types.StringType, Validators: []validator.List{listvalidator.UniqueValues()}},
			}},
		}}},
		"actions": schema.SingleNestedAttribute{Required: true, Attributes: map[string]schema.Attribute{
			"backends": schema.ListNestedAttribute{Optional: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"address": schema.StringAttribute{Required: true},
				"port":    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 65535)}},
			}}},
			"headers": schema.ListNestedAttribute{Optional: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"key":   schema.StringAttribute{Required: true},
				"value": schema.StringAttribute{Required: true},
			}}},
			"host_header": schema.StringAttribute{Optional: true},
			"redirect": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
				"url":                  schema.StringAttribute{Required: true},
				"status_code":          schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.OneOf(301, 302)}},
				"append_original_path": schema.BoolAttribute{Required: true},
			}},
			"rate_limit": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
				"by_source_ip":         httpProxyRateLimitRuleAttribute(false),
				"by_source_ip_and_url": httpProxyRateLimitRuleAttribute(false),
			}},
			"max_body_size": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
				"enforcement": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("ENABLED", "DISABLED")}},
				"value_bytes": schema.Int64Attribute{Optional: true, Validators: []validator.Int64{int64validator.Between(1, 1_099_511_627_776)}},
			}},
		}},
	}}}
}

func (r *HTTPProxyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = apiClient
}

func (r *HTTPProxyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config HTTPProxyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prior HTTPProxyResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	planned := mergeHTTPProxyPlan(config, prior, req.State.Raw.IsNull())
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &planned)...)
}

func mergeHTTPProxyPlan(config, prior HTTPProxyResourceModel, creating bool) HTTPProxyResourceModel {
	planned := config
	if creating {
		planned.ID = types.StringUnknown()
	} else {
		planned.ID = prior.ID
	}

	if planned.DeploymentState.IsNull() || planned.DeploymentState.IsUnknown() {
		planned.DeploymentState = prior.DeploymentState
		if creating || planned.DeploymentState.IsNull() || planned.DeploymentState.IsUnknown() {
			planned.DeploymentState = types.StringValue("UNDEPLOYED")
		}
	}
	if planned.ConnectionReuse.IsNull() || planned.ConnectionReuse.IsUnknown() {
		planned.ConnectionReuse = prior.ConnectionReuse
		if creating || planned.ConnectionReuse.IsNull() || planned.ConnectionReuse.IsUnknown() {
			planned.ConnectionReuse = types.BoolValue(true)
		}
	}

	defaultHTTPProxyCore(&planned)
	mergeHTTPProxyOptionalState(&planned, prior, creating)
	return planned
}

func defaultHTTPProxyCore(data *HTTPProxyResourceModel) {
	if data.Frontend != nil && data.Frontend.ConnectionType.ValueString() == "SECURE" {
		if data.Frontend.RedirectHTTP.IsNull() || data.Frontend.RedirectHTTP.IsUnknown() {
			data.Frontend.RedirectHTTP = types.BoolValue(true)
		}
		if data.Frontend.HSTS == nil {
			data.Frontend.HSTS = defaultHTTPProxyHSTS()
		} else {
			defaultHTTPProxyHSTSFields(data.Frontend.HSTS)
		}
		if data.Frontend.ClientCertificateVerification == nil {
			data.Frontend.ClientCertificateVerification = &HTTPProxyClientCertificateVerificationModel{Mode: types.StringValue("DISABLED"), CACertificateIDs: []types.String{}}
		} else {
			if data.Frontend.ClientCertificateVerification.Mode.IsNull() || data.Frontend.ClientCertificateVerification.Mode.IsUnknown() {
				data.Frontend.ClientCertificateVerification.Mode = types.StringValue("DISABLED")
			}
			if data.Frontend.ClientCertificateVerification.CACertificateIDs == nil {
				data.Frontend.ClientCertificateVerification.CACertificateIDs = []types.String{}
			}
		}
	}
	if data.Backend != nil {
		if data.Backend.DeliveryMethod.IsNull() || data.Backend.DeliveryMethod.IsUnknown() {
			data.Backend.DeliveryMethod = types.StringValue("LEAST_CONNECTIONS")
		}
		if data.Frontend != nil && data.Frontend.ConnectionType.ValueString() == "SECURE" && data.Backend.TLSSettings == nil {
			data.Backend.TLSSettings = defaultHTTPProxyTLSSettings()
		} else if data.Backend.TLSSettings != nil {
			if data.Backend.TLSSettings.VerifyCertificate == nil {
				data.Backend.TLSSettings.VerifyCertificate = &HTTPProxyVerifyCertificateModel{Mode: types.StringValue("SYSTEM_TRUSTSTORE"), CACertificateIDs: []types.String{}}
			} else {
				if data.Backend.TLSSettings.VerifyCertificate.Mode.IsNull() || data.Backend.TLSSettings.VerifyCertificate.Mode.IsUnknown() {
					data.Backend.TLSSettings.VerifyCertificate.Mode = types.StringValue("SYSTEM_TRUSTSTORE")
				}
				if data.Backend.TLSSettings.VerifyCertificate.CACertificateIDs == nil {
					data.Backend.TLSSettings.VerifyCertificate.CACertificateIDs = []types.String{}
				}
			}
		}
	}
	if data.ProtocolSettings != nil && (data.ProtocolSettings.EnableWebsockets.IsNull() || data.ProtocolSettings.EnableWebsockets.IsUnknown()) {
		data.ProtocolSettings.EnableWebsockets = types.BoolValue(false)
	}
}

func mergeHTTPProxyOptionalState(data *HTTPProxyResourceModel, prior HTTPProxyResourceModel, creating bool) {
	if data.BotProtection == nil {
		data.BotProtection = prior.BotProtection
		if creating || data.BotProtection == nil {
			data.BotProtection = defaultHTTPProxyBotProtection()
		}
	} else {
		defaultHTTPProxyBotProtectionFields(data.BotProtection)
	}
	if data.DataProtection == nil {
		data.DataProtection = prior.DataProtection
		if creating || data.DataProtection == nil {
			data.DataProtection = defaultHTTPProxyDataProtection()
		}
	} else {
		defaultHTTPProxyDataProtectionFields(data.DataProtection)
	}
	if data.CustomPages == nil {
		data.CustomPages = prior.CustomPages
		if creating || data.CustomPages == nil {
			data.CustomPages = []HTTPProxyCustomPageModel{}
		}
	}
	if data.WAF == nil {
		data.WAF = prior.WAF
		if creating || data.WAF == nil {
			data.WAF = defaultHTTPProxyWAF()
		}
	} else {
		defaultHTTPProxyWAFFields(data.WAF)
	}
	if data.RateLimiting == nil {
		data.RateLimiting = prior.RateLimiting
		if creating || data.RateLimiting == nil {
			data.RateLimiting = defaultHTTPProxyRateLimiting()
		}
	} else {
		defaultHTTPProxyRateLimitingFields(data.RateLimiting)
	}
	if data.IPBasedAccessControl == nil {
		data.IPBasedAccessControl = prior.IPBasedAccessControl
		if creating || data.IPBasedAccessControl == nil {
			data.IPBasedAccessControl = defaultHTTPProxyIPAccessControl()
		}
	} else {
		defaultHTTPProxyIPAccessControlFields(data.IPBasedAccessControl)
	}
	if data.TrafficRules == nil {
		data.TrafficRules = prior.TrafficRules
		if creating || data.TrafficRules == nil {
			data.TrafficRules = []HTTPProxyTrafficRuleModel{}
		}
	}
	for i := range data.TrafficRules {
		actions := data.TrafficRules[i].Actions
		if actions == nil || actions.RateLimit == nil {
			continue
		}
		if actions.RateLimit.BySourceIP == nil {
			actions.RateLimit.BySourceIP = defaultHTTPProxyRateLimitRule()
		} else {
			defaultHTTPProxyRateLimitRuleFields(actions.RateLimit.BySourceIP)
		}
		if actions.RateLimit.BySourceIPAndURL == nil {
			actions.RateLimit.BySourceIPAndURL = defaultHTTPProxyRateLimitRule()
		} else {
			defaultHTTPProxyRateLimitRuleFields(actions.RateLimit.BySourceIPAndURL)
		}
	}
}

func defaultHTTPProxyHSTS() *HTTPProxyHSTSModel {
	return &HTTPProxyHSTSModel{Enabled: types.BoolValue(false), MaxAge: types.Int64Value(httpProxyDefaultHSTSMaxAge), IncludeSubdomains: types.BoolValue(false), Preload: types.BoolValue(false)}
}

func defaultHTTPProxyHSTSFields(v *HTTPProxyHSTSModel) {
	if v.Enabled.IsNull() || v.Enabled.IsUnknown() {
		v.Enabled = types.BoolValue(false)
	}
	if v.MaxAge.IsNull() || v.MaxAge.IsUnknown() {
		v.MaxAge = types.Int64Value(httpProxyDefaultHSTSMaxAge)
	}
	if v.IncludeSubdomains.IsNull() || v.IncludeSubdomains.IsUnknown() {
		v.IncludeSubdomains = types.BoolValue(false)
	}
	if v.Preload.IsNull() || v.Preload.IsUnknown() {
		v.Preload = types.BoolValue(false)
	}
}

func defaultHTTPProxyTLSSettings() *HTTPProxyTLSSettingsModel {
	return &HTTPProxyTLSSettingsModel{VerifyCertificate: &HTTPProxyVerifyCertificateModel{Mode: types.StringValue("SYSTEM_TRUSTSTORE"), CACertificateIDs: []types.String{}}}
}

func defaultHTTPProxyBotProtection() *HTTPProxyBotProtectionModel {
	return &HTTPProxyBotProtectionModel{Strategy: types.StringValue("AUTO"), ChallengeType: types.StringValue("JS")}
}

func defaultHTTPProxyBotProtectionFields(v *HTTPProxyBotProtectionModel) {
	if v.Strategy.IsNull() || v.Strategy.IsUnknown() {
		v.Strategy = types.StringValue("AUTO")
	}
	if v.ChallengeType.IsNull() || v.ChallengeType.IsUnknown() {
		v.ChallengeType = types.StringValue("JS")
	}
}

func defaultHTTPProxyDataProtection() *HTTPProxyDataProtectionModel {
	return &HTTPProxyDataProtectionModel{LogRedaction: &HTTPProxyLogRedactionModel{Headers: []types.String{}, Cookies: []types.String{}}}
}

func defaultHTTPProxyDataProtectionFields(v *HTTPProxyDataProtectionModel) {
	if v.LogRedaction == nil {
		v.LogRedaction = &HTTPProxyLogRedactionModel{}
	}
	if v.LogRedaction.Headers == nil {
		v.LogRedaction.Headers = []types.String{}
	}
	if v.LogRedaction.Cookies == nil {
		v.LogRedaction.Cookies = []types.String{}
	}
}

func defaultHTTPProxyRateLimitRule() *HTTPProxyRateLimitRuleModel {
	return &HTTPProxyRateLimitRuleModel{Enforcement: types.StringValue("BLOCK"), Rate: &HTTPProxyRateLimitRateModel{Value: types.Int64Value(10), Unit: types.StringValue("r/s")}, Burst: types.Int64Value(100)}
}

func defaultHTTPProxyRateLimitRuleFields(v *HTTPProxyRateLimitRuleModel) {
	if v.Enforcement.IsNull() || v.Enforcement.IsUnknown() {
		v.Enforcement = types.StringValue("BLOCK")
	}
	if v.Rate == nil {
		v.Rate = &HTTPProxyRateLimitRateModel{}
	}
	if v.Rate.Value.IsNull() || v.Rate.Value.IsUnknown() {
		v.Rate.Value = types.Int64Value(10)
	}
	if v.Rate.Unit.IsNull() || v.Rate.Unit.IsUnknown() {
		v.Rate.Unit = types.StringValue("r/s")
	}
	if v.Burst.IsNull() || v.Burst.IsUnknown() {
		v.Burst = types.Int64Value(100)
	}
}

func defaultHTTPProxyRateLimiting() *HTTPProxyRateLimitingModel {
	return &HTTPProxyRateLimitingModel{BySourceIP: defaultHTTPProxyRateLimitRule(), BySourceIPAndURL: defaultHTTPProxyRateLimitRule(), Exclusions: []types.String{}}
}

func defaultHTTPProxyRateLimitingFields(v *HTTPProxyRateLimitingModel) {
	if v.BySourceIP == nil {
		v.BySourceIP = defaultHTTPProxyRateLimitRule()
	} else {
		defaultHTTPProxyRateLimitRuleFields(v.BySourceIP)
	}
	if v.BySourceIPAndURL == nil {
		v.BySourceIPAndURL = defaultHTTPProxyRateLimitRule()
	} else {
		defaultHTTPProxyRateLimitRuleFields(v.BySourceIPAndURL)
	}
	if v.Exclusions == nil {
		v.Exclusions = []types.String{}
	}
}

func defaultHTTPProxyIPAccessControl() *HTTPProxyIPAccessControlModel {
	return &HTTPProxyIPAccessControlModel{DefaultPolicy: types.StringValue("ALLOW"), Rules: &HTTPProxyIPAccessControlRulesModel{IPRanges: []HTTPProxyIPRangeRuleModel{}, IPLists: []HTTPProxyIPListRuleModel{}, KnownServices: []HTTPProxyKnownServiceRuleModel{}, GeoLocations: []HTTPProxyGeoLocationRuleModel{}, ASNs: []HTTPProxyASNRuleModel{}}}
}

func defaultHTTPProxyIPAccessControlFields(v *HTTPProxyIPAccessControlModel) {
	if v.DefaultPolicy.IsNull() || v.DefaultPolicy.IsUnknown() {
		v.DefaultPolicy = types.StringValue("ALLOW")
	}
	if v.Rules == nil {
		v.Rules = &HTTPProxyIPAccessControlRulesModel{}
	}
	if v.Rules.IPRanges == nil {
		v.Rules.IPRanges = []HTTPProxyIPRangeRuleModel{}
	}
	if v.Rules.IPLists == nil {
		v.Rules.IPLists = []HTTPProxyIPListRuleModel{}
	}
	if v.Rules.KnownServices == nil {
		v.Rules.KnownServices = []HTTPProxyKnownServiceRuleModel{}
	}
	if v.Rules.GeoLocations == nil {
		v.Rules.GeoLocations = []HTTPProxyGeoLocationRuleModel{}
	}
	if v.Rules.ASNs == nil {
		v.Rules.ASNs = []HTTPProxyASNRuleModel{}
	}
	for i := range v.Rules.IPRanges {
		if v.Rules.IPRanges[i].BypassBotProtection.IsNull() {
			v.Rules.IPRanges[i].BypassBotProtection = types.BoolValue(false)
		}
	}
	for i := range v.Rules.KnownServices {
		if v.Rules.KnownServices[i].BypassBotProtection.IsNull() {
			v.Rules.KnownServices[i].BypassBotProtection = types.BoolValue(false)
		}
	}
}

func defaultHTTPProxyWAF() *HTTPProxyWAFModel {
	return &HTTPProxyWAFModel{
		Enforcement: types.StringValue("LOG"), ParanoiaLevel: types.Int64Value(1), CoreRuleSetVersion: types.StringValue(httpProxyDefaultCRSVersion), MatchedDataEnabled: types.BoolValue(false),
		SourceExclusions: &HTTPProxySourceExclusionsModel{Enabled: types.BoolValue(false), Sources: []types.String{}},
		HTTPCompliance:   &HTTPProxyHTTPComplianceModel{ParameterLimit: &HTTPProxyParameterLimitModel{Enabled: types.BoolValue(false), Limit: types.Int64Value(500)}, AllowedMethods: stringSliceToTypeValues([]string{"GET", "POST", "DELETE", "PATCH", "PUT"}), AllowedVersions: stringSliceToTypeValues([]string{"HTTP/1.0", "HTTP/1.1", "HTTP/2.0", "HTTP/2"}), ResourceConfigs: []HTTPProxyHTTPComplianceResourceConfigModel{}},
		PathExclusions:   []HTTPProxyWAFPathExclusionModel{}, CookieExclusions: []HTTPProxyWAFCookieExclusionModel{},
	}
}

func defaultHTTPProxyWAFFields(v *HTTPProxyWAFModel) {
	if v.Enforcement.IsNull() || v.Enforcement.IsUnknown() {
		v.Enforcement = types.StringValue("LOG")
	}
	if v.ParanoiaLevel.IsNull() || v.ParanoiaLevel.IsUnknown() {
		v.ParanoiaLevel = types.Int64Value(1)
	}
	if v.CoreRuleSetVersion.IsNull() || v.CoreRuleSetVersion.IsUnknown() {
		v.CoreRuleSetVersion = types.StringValue(httpProxyDefaultCRSVersion)
	}
	if v.MatchedDataEnabled.IsNull() || v.MatchedDataEnabled.IsUnknown() {
		v.MatchedDataEnabled = types.BoolValue(false)
	}
	if v.SourceExclusions == nil {
		v.SourceExclusions = &HTTPProxySourceExclusionsModel{}
	}
	if v.SourceExclusions.Enabled.IsNull() || v.SourceExclusions.Enabled.IsUnknown() {
		v.SourceExclusions.Enabled = types.BoolValue(false)
	}
	if v.SourceExclusions.Sources == nil {
		v.SourceExclusions.Sources = []types.String{}
	}
	if v.HTTPCompliance == nil {
		v.HTTPCompliance = &HTTPProxyHTTPComplianceModel{}
	}
	if v.HTTPCompliance.ParameterLimit == nil {
		v.HTTPCompliance.ParameterLimit = &HTTPProxyParameterLimitModel{}
	}
	if v.HTTPCompliance.ParameterLimit.Enabled.IsNull() || v.HTTPCompliance.ParameterLimit.Enabled.IsUnknown() {
		v.HTTPCompliance.ParameterLimit.Enabled = types.BoolValue(false)
	}
	if v.HTTPCompliance.ParameterLimit.Limit.IsNull() || v.HTTPCompliance.ParameterLimit.Limit.IsUnknown() {
		v.HTTPCompliance.ParameterLimit.Limit = types.Int64Value(500)
	}
	if v.HTTPCompliance.AllowedMethods == nil {
		v.HTTPCompliance.AllowedMethods = stringSliceToTypeValues([]string{"GET", "POST", "DELETE", "PATCH", "PUT"})
	}
	if v.HTTPCompliance.AllowedVersions == nil {
		v.HTTPCompliance.AllowedVersions = stringSliceToTypeValues([]string{"HTTP/1.0", "HTTP/1.1", "HTTP/2.0", "HTTP/2"})
	}
	if v.HTTPCompliance.ResourceConfigs == nil {
		v.HTTPCompliance.ResourceConfigs = []HTTPProxyHTTPComplianceResourceConfigModel{}
	}
	if v.PathExclusions == nil {
		v.PathExclusions = []HTTPProxyWAFPathExclusionModel{}
	}
	if v.CookieExclusions == nil {
		v.CookieExclusions = []HTTPProxyWAFCookieExclusionModel{}
	}
}

func (r *HTTPProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HTTPProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tc, err := r.client.CreateHTTPProxy(ctx, mapHTTPProxyModelToCreateRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create HTTP Proxy traffic config, got error: %s", err))
		return
	}
	data.ID = types.StringValue(tc.Data.ID)
	if err := r.client.WaitForTrafficConfigChange(ctx, tc.Data.ID, tc.Data.ActiveChangeID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for HTTP Proxy traffic config create, got error: %s", err))
		return
	}
	data, err = r.readHTTPProxy(ctx, data)
	if err != nil {
		addHTTPProxyReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "after create")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HTTPProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HTTPProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data, err := r.readHTTPProxy(ctx, data)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		addHTTPProxyReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HTTPProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HTTPProxyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tc, err := r.client.UpdateHTTPProxy(ctx, data.ID.ValueString(), mapHTTPProxyModelToUpdateRequest(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update HTTP Proxy traffic config, got error: %s", err))
		return
	}
	if err := r.client.WaitForTrafficConfigChange(ctx, data.ID.ValueString(), tc.Data.ActiveChangeID()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for HTTP Proxy traffic config update, got error: %s", err))
		return
	}
	data, err = r.readHTTPProxy(ctx, data)
	if err != nil {
		addHTTPProxyReadDiagnostic(&resp.Diagnostics, data.ID.ValueString(), err, "after update")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HTTPProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HTTPProxyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTrafficConfig(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete HTTP Proxy traffic config, got error: %s", err))
	}
}

func (r *HTTPProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *HTTPProxyResource) readHTTPProxy(ctx context.Context, prior HTTPProxyResourceModel) (HTTPProxyResourceModel, error) {
	tc, err := r.client.GetHTTPProxy(ctx, prior.ID.ValueString())
	if err != nil {
		return prior, err
	}
	if tc.Data.Type != httpProxyTrafficConfigType {
		return prior, trafficConfigTypeMismatchError{ID: prior.ID.ValueString(), GotType: tc.Data.Type, WantType: httpProxyTrafficConfigType}
	}
	return mapHTTPProxyResponseToModel(tc, prior), nil
}

func addHTTPProxyReadDiagnostic(diags interface{ AddError(summary, detail string) }, id string, err error, context string) {
	if _, ok := err.(trafficConfigTypeMismatchError); ok {
		diags.AddError("Traffic Config Type Mismatch", fmt.Sprintf("Unable to read HTTP Proxy traffic config %q%s: %s. Import or reference a traffic config with API type %q.", id, readContextSuffix(context), err, httpProxyTrafficConfigType))
		return
	}
	diags.AddError("Client Error", fmt.Sprintf("Unable to read HTTP Proxy traffic config %q%s, got error: %s", id, readContextSuffix(context), err))
}
