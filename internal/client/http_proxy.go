package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPProxyRequest struct {
	Data HTTPProxyRequestData `json:"data"`
}

type HTTPProxyRequestData struct {
	Type          string                       `json:"type"`
	Attributes    HTTPProxyAttributes          `json:"attributes"`
	Relationships HTTPProxyRequestRelationship `json:"relationships"`
}

type HTTPProxyRequestRelationship struct {
	BelongsTo HTTPProxyBelongsTo `json:"belongsTo"`
}

type HTTPProxyBelongsTo struct {
	Data HTTPProxyRelationshipData `json:"data"`
}

type HTTPProxyRelationshipData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type HTTPProxyAttributes struct {
	Name                   string                         `json:"name,omitempty"`
	Version                string                         `json:"version,omitempty"`
	Frontend               *HTTPProxyFrontend             `json:"frontend,omitempty"`
	Backend                *HTTPProxyBackend              `json:"backend,omitempty"`
	ProtocolSettings       *HTTPProxyProtocolSettings     `json:"protocolSettings,omitempty"`
	Deployment             *HTTPProxyDeployment           `json:"deployment,omitempty"`
	WAF                    *HTTPProxyWAF                  `json:"waf,omitempty"`
	IPBasedAccessControl   *HTTPProxyIPBasedAccessControl `json:"ipBasedAccessControl,omitempty"`
	RateLimiting           *HTTPProxyRateLimiting         `json:"rateLimiting,omitempty"`
	TrafficRules           *[]HTTPProxyTrafficRule        `json:"trafficRules,omitempty"`
	DataProtection         *HTTPProxyDataProtection       `json:"dataProtection,omitempty"`
	CustomPages            *[]HTTPProxyCustomPage         `json:"customPages,omitempty"`
	BotProtection          *HTTPProxyBotProtection        `json:"botProtection,omitempty"`
	ConnectionReuseEnabled *bool                          `json:"connectionReuseEnabled,omitempty"`
}

type HTTPProxyDeployment struct {
	State string `json:"state"`
}

type HTTPProxyFrontend struct {
	ConnectionType                string                                  `json:"connectionType"`
	IPv4                          string                                  `json:"ipv4,omitempty"`
	IPv6                          string                                  `json:"ipv6,omitempty"`
	Port                          int64                                   `json:"port"`
	RedirectHTTP                  *bool                                   `json:"redirectHttp,omitempty"`
	Hosts                         []HTTPProxyFrontendHost                 `json:"hosts"`
	HTTPStrictTransportSecurity   *HTTPProxyHSTS                          `json:"httpStrictTransportSecurity,omitempty"`
	ClientCertificateVerification *HTTPProxyClientCertificateVerification `json:"clientCertificateVerification,omitempty"`
}

type HTTPProxyFrontendHost struct {
	Host          string `json:"host"`
	CertificateID string `json:"certificateId,omitempty"`
	TLSConfig     string `json:"tlsConfig,omitempty"`
}

type HTTPProxyHSTS struct {
	Enabled           bool  `json:"enabled"`
	MaxAge            int64 `json:"maxAge"`
	IncludeSubdomains bool  `json:"includeSubdomains"`
	Preload           bool  `json:"preload"`
}

type HTTPProxyClientCertificateVerification struct {
	Mode             string   `json:"mode"`
	CACertificateIDs []string `json:"caCertificateIds,omitempty"`
}

type HTTPProxyBackend struct {
	Hosts          []HTTPProxyBackendHost `json:"hosts"`
	DeliveryMethod string                 `json:"deliveryMethod"`
	ServerName     string                 `json:"serverName,omitempty"`
	TLSSettings    *HTTPProxyTLSSettings  `json:"tlsSettings,omitempty"`
}

type HTTPProxyBackendHost struct {
	Address string `json:"address"`
	Port    int64  `json:"port"`
}

type HTTPProxyTLSSettings struct {
	ClientCertificateID string                             `json:"clientCertificateId,omitempty"`
	VerifyCertificate   *HTTPProxyBackendVerifyCertificate `json:"verifyCertificate,omitempty"`
}

type HTTPProxyBackendVerifyCertificate struct {
	Mode             string   `json:"mode"`
	CACertificateIDs []string `json:"caCertificateIds,omitempty"`
}

type HTTPProxyProtocolSettings struct {
	HTTPVersion      string `json:"httpVersion"`
	EnableWebsockets *bool  `json:"enableWebsockets,omitempty"`
}

type HTTPProxyBotProtection struct {
	Strategy      string `json:"strategy"`
	ChallengeType string `json:"challengeType"`
}

type HTTPProxyDataProtection struct {
	LogRedaction HTTPProxyLogRedaction `json:"logRedaction"`
}

type HTTPProxyLogRedaction struct {
	Headers []string `json:"headers"`
	Cookies []string `json:"cookies"`
}

type HTTPProxyCustomPage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type HTTPProxyRateLimiting struct {
	BySourceIP       HTTPProxyRateLimitRule `json:"bySrcIp"`
	BySourceIPAndURL HTTPProxyRateLimitRule `json:"bySrcIpAndUrl"`
	Exclusions       []string               `json:"exclusions"`
}

type HTTPProxyRateLimitRule struct {
	Enforcement string                 `json:"enforcement"`
	Rate        HTTPProxyRateLimitRate `json:"rate"`
	Burst       int64                  `json:"burst"`
}

type HTTPProxyRateLimitRate struct {
	Value int64  `json:"value"`
	Unit  string `json:"unit"`
}

type HTTPProxyIPBasedAccessControl struct {
	DefaultPolicy string                          `json:"defaultPolicy"`
	Rules         HTTPProxyIPBasedAccessRuleGroup `json:"rules"`
}

type HTTPProxyIPBasedAccessRuleGroup struct {
	IPRanges      []HTTPProxyIPRangeRule      `json:"ipRanges"`
	IPLists       []HTTPProxyIPListRule       `json:"ipLists"`
	KnownServices []HTTPProxyKnownServiceRule `json:"knownServices"`
	GeoLocations  []HTTPProxyGeoLocationRule  `json:"geoLocations"`
	ASNs          []HTTPProxyASNRule          `json:"asns"`
}

type HTTPProxyBypassProtection struct {
	BotProtection bool `json:"botProtection"`
}

type HTTPProxyIPRangeRule struct {
	Policy           string                    `json:"policy"`
	Address          string                    `json:"address"`
	Note             string                    `json:"note,omitempty"`
	BypassProtection HTTPProxyBypassProtection `json:"bypassProtection"`
}

type HTTPProxyIPListRule struct {
	Policy string `json:"policy"`
	ID     string `json:"id"`
}

type HTTPProxyKnownServiceRule struct {
	Policy           string                    `json:"policy"`
	ID               string                    `json:"id"`
	Note             string                    `json:"note,omitempty"`
	BypassProtection HTTPProxyBypassProtection `json:"bypassProtection"`
}

type HTTPProxyGeoLocationRule struct {
	Policy string `json:"policy"`
	Region string `json:"region"`
	Note   string `json:"note,omitempty"`
}

type HTTPProxyASNRule struct {
	Policy string `json:"policy"`
	ASN    int64  `json:"asn"`
	Note   string `json:"note,omitempty"`
}

type HTTPProxyWAF struct {
	Enforcement        string                     `json:"enforcement"`
	ParanoiaLevel      int64                      `json:"paranoidLevel"`
	CoreRuleSet        HTTPProxyCoreRuleSet       `json:"coreRuleSet"`
	CoreRuleSetID      string                     `json:"coreRuleSetId,omitempty"`
	SourceExclusions   HTTPProxySourceExclusions  `json:"sourceExclusions"`
	HTTPCompliance     HTTPProxyHTTPCompliance    `json:"httpCompliance"`
	PathExclusions     []HTTPProxyPathExclusion   `json:"pathExclusions"`
	CookieExclusions   []HTTPProxyCookieExclusion `json:"cookieExclusions"`
	MatchedDataEnabled bool                       `json:"matchedDataEnabled"`
	StagedWAF          *HTTPProxyStagedWAF        `json:"stagedWaf"`
}

type HTTPProxyCoreRuleSet struct {
	Version string `json:"version"`
}

type HTTPProxySourceExclusions struct {
	Enabled bool     `json:"enabled"`
	Sources []string `json:"sources"`
}

type HTTPProxyHTTPCompliance struct {
	GlobalConfig    HTTPProxyHTTPComplianceGlobalConfig     `json:"globalConfig"`
	ResourceConfigs []HTTPProxyHTTPComplianceResourceConfig `json:"resourceConfigs"`
}

type HTTPProxyHTTPComplianceGlobalConfig struct {
	ParameterLimit      HTTPProxyParameterLimit `json:"parameterLimit"`
	AllowedHTTPMethods  []string                `json:"allowedHttpMethods"`
	AllowedHTTPVersions []string                `json:"allowedHttpVersions"`
}

type HTTPProxyParameterLimit struct {
	Enabled bool  `json:"enabled"`
	Limit   int64 `json:"limit"`
}

type HTTPProxyHTTPComplianceResourceConfig struct {
	Matches []HTTPProxyResourceMatch `json:"matches"`
	Config  HTTPProxyResourceConfig  `json:"config"`
}

type HTTPProxyResourceMatch struct {
	Path string `json:"path"`
}

type HTTPProxyResourceConfig struct {
	ParseJSONEnabled             bool `json:"parseJsonEnabled"`
	ParseXMLEnabled              bool `json:"parseXmlEnabled"`
	ParseMultipartRequestEnabled bool `json:"parseMultipartRequestEnabled"`
}

type HTTPProxyExclusionRule struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type HTTPProxyPathExclusion struct {
	Match      string                   `json:"match"`
	DisableAll bool                     `json:"disableAll"`
	Rules      []HTTPProxyExclusionRule `json:"rules"`
}

type HTTPProxyCookieExclusion struct {
	CookieName      string                   `json:"cookieName"`
	ExcludeAllRules bool                     `json:"excludeAllRules"`
	ExcludedRules   []HTTPProxyExclusionRule `json:"excludedRules"`
}

type HTTPProxyStagedWAF struct {
	State            string                     `json:"state"`
	Mode             string                     `json:"mode"`
	ParanoiaLevel    *int64                     `json:"paranoiaLevel,omitempty"`
	CoreRuleSet      *HTTPProxyCoreRuleSet      `json:"coreRuleSet,omitempty"`
	CookieExclusions []HTTPProxyCookieExclusion `json:"cookieExclusions"`
	PathExclusions   []HTTPProxyPathExclusion   `json:"pathExclusions"`
}

type HTTPProxyTrafficRule struct {
	Name               string                              `json:"name"`
	MatchingConditions []HTTPProxyTrafficMatchingCondition `json:"matchingConditions"`
	Actions            HTTPProxyTrafficRuleActions         `json:"actions"`
}

type HTTPProxyTrafficMatchingCondition struct {
	Paths []string                      `json:"paths"`
	Hosts HTTPProxyTrafficRuleHostMatch `json:"hosts"`
}

type HTTPProxyTrafficRuleHostMatch struct {
	Type   string   `json:"type"`
	Values []string `json:"values,omitempty"`
}

type HTTPProxyTrafficRuleActions struct {
	SetBackends    []HTTPProxyBackendHost       `json:"setBackends"`
	SetHeaders     []HTTPProxyHeader            `json:"setHeaders"`
	SetHostHeader  *string                      `json:"setHostHeader"`
	SetRedirect    *HTTPProxyRedirect           `json:"setRedirect"`
	SetRateLimit   *HTTPProxyTrafficRateLimit   `json:"setRateLimit"`
	SetMaxBodySize *HTTPProxySetMaximumBodySize `json:"setMaxBodySize"`
}

type HTTPProxyHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type HTTPProxyRedirect struct {
	URL                string `json:"url"`
	StatusCode         int64  `json:"statusCode"`
	AppendOriginalPath bool   `json:"appendOriginalPath"`
}

type HTTPProxyTrafficRateLimit struct {
	BySourceIP       HTTPProxyRateLimitRule `json:"bySrcIp"`
	BySourceIPAndURL HTTPProxyRateLimitRule `json:"bySrcIpAndUrl"`
}

type HTTPProxySetMaximumBodySize struct {
	Enforcement string `json:"enforcement"`
	Value       *int64 `json:"value,omitempty"`
}

type HTTPProxyData struct {
	ID            string              `json:"id"`
	Type          string              `json:"type"`
	Attributes    HTTPProxyAttributes `json:"attributes"`
	Relationships struct {
		ActiveChange struct {
			Data HTTPProxyRelationshipData `json:"data"`
		} `json:"activeChange"`
	} `json:"relationships"`
}

func (d HTTPProxyData) ActiveChangeID() string {
	return d.Relationships.ActiveChange.Data.ID
}

type HTTPProxyResponse struct {
	Data HTTPProxyData `json:"data"`
}

func (c *Client) CreateHTTPProxy(ctx context.Context, reqData HTTPProxyRequest) (*HTTPProxyResponse, error) {
	return c.mutateHTTPProxy(ctx, http.MethodPost, "/api/v2/traffic-mgmt/traffic-configs", "create HTTP Proxy traffic config", reqData)
}

func (c *Client) UpdateHTTPProxy(ctx context.Context, id string, reqData HTTPProxyRequest) (*HTTPProxyResponse, error) {
	return c.mutateHTTPProxy(ctx, http.MethodPut, "/api/v2/traffic-mgmt/traffic-configs/"+id, "update HTTP Proxy traffic config", reqData)
}

func (c *Client) mutateHTTPProxy(ctx context.Context, method, path, operation string, reqData HTTPProxyRequest) (*HTTPProxyResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to %s", operation)
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.TenantID

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	logTrafficConfigRequest(ctx, method, path, jsonData)
	req, err := c.NewRequest(ctx, method, path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, httpStatusError(resp, operation)
	}

	var proxyResp HTTPProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		return nil, err
	}
	return &proxyResp, nil
}

func (c *Client) GetHTTPProxy(ctx context.Context, id string) (*HTTPProxyResponse, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/v2/traffic-mgmt/traffic-configs/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get HTTP Proxy traffic config")
	}

	var proxyResp HTTPProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		return nil, err
	}
	return &proxyResp, nil
}
