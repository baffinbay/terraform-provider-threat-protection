package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	trafficConfigChangePollTimeout = 2 * time.Minute
)

var trafficConfigChangePollInterval = 2 * time.Second

func SetTrafficConfigChangePollIntervalForTesting(interval time.Duration) func() {
	previous := trafficConfigChangePollInterval
	trafficConfigChangePollInterval = interval
	return func() {
		trafficConfigChangePollInterval = previous
	}
}

type TrafficConfigRequest struct {
	Data struct {
		Type          string                  `json:"type"`
		Attributes    TrafficConfigAttributes `json:"attributes"`
		Relationships struct {
			BelongsTo struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"belongsTo"`
		} `json:"relationships"`
	} `json:"data"`
}

type TrafficConfigAttributes struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Frontend         *Frontend         `json:"frontend,omitempty"`
	Backend          *Backend          `json:"backend,omitempty"`
	Deployment       Deployment        `json:"deployment"`
	Protocols        []string          `json:"protocols,omitempty"`
	ProxyProtocol    string            `json:"proxyProtocol,omitempty"`
	Prefix           string            `json:"prefix,omitempty"`
	Announced        *bool             `json:"announced,omitempty"`
	WAF              *WAF              `json:"waf,omitempty"`
	GeoFencing       *GeoFencing       `json:"geoFencing,omitempty"`
	AllowedSources   *AllowedSources   `json:"allowedSources,omitempty"`
	IPBasedAccess    *IPBasedAccess    `json:"ipBasedAccessControl,omitempty"`
	ConnectionReuse  *bool             `json:"connectionReuseEnabled,omitempty"`
	BotProtection    *BotProtection    `json:"botProtection,omitempty"`
	CustomPages      *[]any            `json:"customPages,omitempty"`
	DataProtection   *DataProtection   `json:"dataProtection,omitempty"`
	GatewayPath      string            `json:"gatewayPath,omitempty"`
	RateLimiting     *RateLimit        `json:"rateLimiting,omitempty"`
	TrafficRules     *[]any            `json:"trafficRules,omitempty"`
	ProtocolSettings *ProtocolSettings `json:"protocolSettings,omitempty"`
}

type Deployment struct {
	State string `json:"state"`
}

type WAF struct {
	Enforcement      string `json:"enforcement"`
	ParanoidLevel    int64  `json:"paranoidLevel"`
	CoreRuleSetID    string `json:"coreRuleSetId"`
	SourceExclusions struct {
		Enabled bool     `json:"enabled"`
		Sources []string `json:"sources"`
	} `json:"sourceExclusions"`
	HTTPCompliance struct {
		GlobalConfig struct {
			ParameterLimit struct {
				Enabled bool `json:"enabled"`
				Limit   int  `json:"limit"`
			} `json:"parameterLimit"`
			AllowedHttpMethods  []string `json:"allowedHttpMethods"`
			AllowedHttpVersions []string `json:"allowedHttpVersions"`
		} `json:"globalConfig"`
		ResourceConfigs []any `json:"resourceConfigs"`
	} `json:"httpCompliance"`
	PathExclusions []any `json:"pathExclusions"`
}

type RateLimit struct {
	BySrcIP       RateLimitRule `json:"bySrcIp"`
	BySrcIPAndURL RateLimitRule `json:"bySrcIpAndUrl"`
}

type RateLimitRule struct {
	Enforcement string        `json:"enforcement"`
	Rate        RateLimitRate `json:"rate"`
	Burst       int           `json:"burst"`
}

type RateLimitRate struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type GeoFencing struct {
	Type    string   `json:"type"`
	Regions []string `json:"regions"`
}

type AllowedSources struct {
	Enforcement string   `json:"enforcement"`
	Sources     []string `json:"sources"`
}

type IPBasedAccess struct {
	DefaultPolicy string             `json:"defaultPolicy"`
	Rules         IPBasedAccessRules `json:"rules"`
}

type IPBasedAccessRules struct {
	IPRanges      []IPBasedAccessIPRangeRule      `json:"ipRanges"`
	KnownServices *[]any                          `json:"knownServices,omitempty"`
	IPLists       []IPBasedAccessIPListRule       `json:"ipLists"`
	GeoLocations  []IPBasedAccessGeoLocationRule  `json:"geoLocations"`
	ASNs          []IPBasedAccessAutonomousSystem `json:"asns"`
}

type IPBasedAccessIPRangeRule struct {
	Policy  string `json:"policy"`
	Address string `json:"address"`
	Note    string `json:"note,omitempty"`
}

type IPBasedAccessIPListRule struct {
	Type   string `json:"type,omitempty"`
	ID     string `json:"id"`
	Policy string `json:"policy"`
}

type IPBasedAccessGeoLocationRule struct {
	Policy string `json:"policy"`
	Region string `json:"region"`
	Note   string `json:"note,omitempty"`
}

type IPBasedAccessAutonomousSystem struct {
	Policy string `json:"policy"`
	ASN    int64  `json:"asn"`
	Note   string `json:"note,omitempty"`
}

type BotProtection struct {
	Strategy      string `json:"strategy"`
	ChallengeType string `json:"challengeType"`
}

type DataProtection struct {
	LogRedaction LogRedaction `json:"logRedaction"`
}

type LogRedaction struct {
	Headers []string `json:"headers"`
	Cookies []string `json:"cookies"`
}

type Frontend struct {
	IPv4                          string                         `json:"ipv4,omitempty"`
	IPv6                          string                         `json:"ipv6,omitempty"`
	IP                            string                         `json:"ip,omitempty"`
	Port                          int64                          `json:"port,omitempty"`
	ConnectionType                string                         `json:"connectionType,omitempty"`
	RedirectHttp                  *bool                          `json:"redirectHttp,omitempty"`
	Hosts                         []FrontendHost                 `json:"hosts,omitempty"`
	HTTPStrictTransportSecurity   *HSTS                          `json:"httpStrictTransportSecurity,omitempty"`
	ClientCertificateVerification *ClientCertificateVerification `json:"clientCertificateVerification,omitempty"`
}

type FrontendHost struct {
	Host          string `json:"host"`
	CertificateID string `json:"certificateId,omitempty"`
	TLSConfig     string `json:"tlsConfig,omitempty"`
}

type HSTS struct {
	Enabled           bool  `json:"enabled"`
	MaxAge            int64 `json:"maxAge"`
	IncludeSubdomains bool  `json:"includeSubdomains"`
	Preload           bool  `json:"preload"`
}

type ClientCertificateVerification struct {
	Mode             string   `json:"mode"`
	VerifyCrl        *bool    `json:"verifyCrl,omitempty"`
	CaCertificateIds []string `json:"caCertificateIds,omitempty"`
}

type Backend struct {
	Hosts          []Host       `json:"hosts"`
	DeliveryMethod string       `json:"deliveryMethod"`
	ServerName     string       `json:"serverName"`
	TLSSettings    *TLSSettings `json:"tlsSettings,omitempty"`
}

type TLSSettings struct {
	ClientCertificateID string             `json:"clientCertificateId,omitempty"`
	VerifyCertificate   *VerifyCertificate `json:"verifyCertificate,omitempty"`
}

type VerifyCertificate struct {
	Mode             string   `json:"mode"`
	CaCertificateIds []string `json:"caCertificateIds,omitempty"`
	VerifyCrl        *bool    `json:"verifyCrl,omitempty"`
}

type Host struct {
	Address string `json:"address"`
	Port    int64  `json:"port"`
}

type ProtocolSettings struct {
	Version          string `json:"httpVersion"`
	EnableWebsockets *bool  `json:"enableWebsockets,omitempty"`
	Multiplexing     *bool  `json:"multiplexing,omitempty"`
}

type TrafficConfigData struct {
	ID            string                          `json:"id"`
	Type          string                          `json:"type"`
	Attributes    TrafficConfigResponseAttributes `json:"attributes"`
	Relationships struct {
		ActiveChange struct {
			Data struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"data"`
		} `json:"activeChange"`
	} `json:"relationships"`
}

type TrafficConfigResponseAttributes struct {
	Name             *string                                `json:"name,omitempty"`
	Version          *string                                `json:"version,omitempty"`
	Frontend         *TrafficConfigResponseFrontend         `json:"frontend,omitempty"`
	Backend          *TrafficConfigResponseBackend          `json:"backend,omitempty"`
	Deployment       *TrafficConfigResponseDeployment       `json:"deployment,omitempty"`
	Protocols        []string                               `json:"protocols,omitempty"`
	ProxyProtocol    *string                                `json:"proxyProtocol,omitempty"`
	Prefix           *string                                `json:"prefix,omitempty"`
	Announced        *bool                                  `json:"announced,omitempty"`
	WAF              *TrafficConfigResponseWAF              `json:"waf,omitempty"`
	GeoFencing       *TrafficConfigResponseGeoFencing       `json:"geoFencing,omitempty"`
	AllowedSources   *TrafficConfigResponseAllowedSources   `json:"allowedSources,omitempty"`
	IPBasedAccess    *TrafficConfigResponseIPBasedAccess    `json:"ipBasedAccessControl,omitempty"`
	RateLimiting     *TrafficConfigResponseRateLimit        `json:"rateLimiting,omitempty"`
	ProtocolSettings *TrafficConfigResponseProtocolSettings `json:"protocolSettings,omitempty"`
}

type TrafficConfigResponseDeployment struct {
	State *string `json:"state,omitempty"`
}

type TrafficConfigResponseFrontend struct {
	IPv4                          *string                                             `json:"ipv4,omitempty"`
	IPv6                          *string                                             `json:"ipv6,omitempty"`
	IP                            *string                                             `json:"ip,omitempty"`
	Port                          *int64                                              `json:"port,omitempty"`
	ConnectionType                *string                                             `json:"connectionType,omitempty"`
	RedirectHttp                  *bool                                               `json:"redirectHttp,omitempty"`
	Hosts                         []TrafficConfigResponseFrontendHost                 `json:"hosts,omitempty"`
	HTTPStrictTransportSecurity   *TrafficConfigResponseHSTS                          `json:"httpStrictTransportSecurity,omitempty"`
	ClientCertificateVerification *TrafficConfigResponseClientCertificateVerification `json:"clientCertificateVerification,omitempty"`
}

type TrafficConfigResponseFrontendHost struct {
	Host          *string `json:"host,omitempty"`
	CertificateID *string `json:"certificateId,omitempty"`
	TLSConfig     *string `json:"tlsConfig,omitempty"`
}

type TrafficConfigResponseHSTS struct {
	Enabled           *bool  `json:"enabled,omitempty"`
	MaxAge            *int64 `json:"maxAge,omitempty"`
	IncludeSubdomains *bool  `json:"includeSubdomains,omitempty"`
	Preload           *bool  `json:"preload,omitempty"`
}

type TrafficConfigResponseClientCertificateVerification struct {
	Mode             *string  `json:"mode,omitempty"`
	VerifyCrl        *bool    `json:"verifyCrl,omitempty"`
	CaCertificateIds []string `json:"caCertificateIds,omitempty"`
}

type TrafficConfigResponseBackend struct {
	Hosts          []TrafficConfigResponseHost       `json:"hosts,omitempty"`
	DeliveryMethod *string                           `json:"deliveryMethod,omitempty"`
	ServerName     *string                           `json:"serverName,omitempty"`
	TLSSettings    *TrafficConfigResponseTLSSettings `json:"tlsSettings,omitempty"`
}

type TrafficConfigResponseTLSSettings struct {
	ClientCertificateID *string                                 `json:"clientCertificateId,omitempty"`
	VerifyCertificate   *TrafficConfigResponseVerifyCertificate `json:"verifyCertificate,omitempty"`
}

type TrafficConfigResponseVerifyCertificate struct {
	Mode             *string  `json:"mode,omitempty"`
	CaCertificateIds []string `json:"caCertificateIds,omitempty"`
	VerifyCrl        *bool    `json:"verifyCrl,omitempty"`
}

type TrafficConfigResponseHost struct {
	Address *string `json:"address,omitempty"`
	Port    *int64  `json:"port,omitempty"`
}

type TrafficConfigResponseProtocolSettings struct {
	Version          *string `json:"httpVersion,omitempty"`
	EnableWebsockets *bool   `json:"enableWebsockets,omitempty"`
	Multiplexing     *bool   `json:"multiplexing,omitempty"`
}

type TrafficConfigResponseWAF struct {
	Enforcement      *string                                 `json:"enforcement,omitempty"`
	ParanoidLevel    *int64                                  `json:"paranoidLevel,omitempty"`
	CoreRuleSetID    *string                                 `json:"coreRuleSetId,omitempty"`
	CoreRuleSet      *TrafficConfigResponseCoreRuleSet       `json:"coreRuleSet,omitempty"`
	SourceExclusions *TrafficConfigResponseSourceExclusions  `json:"sourceExclusions,omitempty"`
	HTTPCompliance   *TrafficConfigResponseHTTPCompliance    `json:"httpCompliance,omitempty"`
	PathExclusions   []TrafficConfigResponseWAFPathExclusion `json:"pathExclusions,omitempty"`
}

type TrafficConfigResponseCoreRuleSet struct {
	Version *string `json:"version,omitempty"`
}

type TrafficConfigResponseSourceExclusions struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Sources []string `json:"sources,omitempty"`
}

type TrafficConfigResponseHTTPCompliance struct {
	GlobalConfig *TrafficConfigResponseHTTPComplianceGlobalConfig `json:"globalConfig,omitempty"`
}

type TrafficConfigResponseHTTPComplianceGlobalConfig struct {
	ParameterLimit      *TrafficConfigResponseParameterLimit `json:"parameterLimit,omitempty"`
	AllowedHTTPMethods  []string                             `json:"allowedHttpMethods,omitempty"`
	AllowedHTTPVersions []string                             `json:"allowedHttpVersions,omitempty"`
}

type TrafficConfigResponseParameterLimit struct {
	Enabled *bool `json:"enabled,omitempty"`
	Limit   *int  `json:"limit,omitempty"`
}

type TrafficConfigResponseWAFPathExclusion struct {
	Type        *string                                 `json:"type,omitempty"`
	Value       *string                                 `json:"value,omitempty"`
	Description *string                                 `json:"description,omitempty"`
	Match       *string                                 `json:"match,omitempty"`
	DisableAll  *bool                                   `json:"disableAll,omitempty"`
	Rules       []TrafficConfigResponseWAFExclusionRule `json:"rules,omitempty"`
}

type TrafficConfigResponseWAFExclusionRule struct {
	Type *string `json:"type,omitempty"`
	ID   *string `json:"id,omitempty"`
}

type TrafficConfigResponseRateLimit struct {
	Enforcement   *string                             `json:"enforcement,omitempty"`
	BySrcIP       *TrafficConfigResponseRateLimitRule `json:"bySrcIp,omitempty"`
	BySrcIPAndURL *TrafficConfigResponseRateLimitRule `json:"bySrcIpAndUrl,omitempty"`
}

type TrafficConfigResponseRateLimitRule struct {
	Enforcement *string `json:"enforcement,omitempty"`
}

type TrafficConfigResponseGeoFencing struct {
	Type    *string  `json:"type,omitempty"`
	Regions []string `json:"regions,omitempty"`
}

type TrafficConfigResponseAllowedSources struct {
	Enforcement *string  `json:"enforcement,omitempty"`
	Sources     []string `json:"sources,omitempty"`
}

type TrafficConfigResponseIPBasedAccess struct {
	DefaultPolicy *string                                  `json:"defaultPolicy,omitempty"`
	Rules         *TrafficConfigResponseIPBasedAccessRules `json:"rules,omitempty"`
}

type TrafficConfigResponseIPBasedAccessRules struct {
	IPRanges      []TrafficConfigResponseIPRangeRule      `json:"ipRanges,omitempty"`
	KnownServices []any                                   `json:"knownServices,omitempty"`
	IPLists       []TrafficConfigResponseIPListRule       `json:"ipLists,omitempty"`
	GeoLocations  []TrafficConfigResponseGeoLocationRule  `json:"geoLocations,omitempty"`
	ASNs          []TrafficConfigResponseAutonomousSystem `json:"asns,omitempty"`
}

type TrafficConfigResponseIPRangeRule struct {
	Policy  *string `json:"policy,omitempty"`
	Address *string `json:"address,omitempty"`
	Note    *string `json:"note,omitempty"`
}

type TrafficConfigResponseIPListRule struct {
	Type   *string `json:"type,omitempty"`
	ID     *string `json:"id,omitempty"`
	Policy *string `json:"policy,omitempty"`
}

type TrafficConfigResponseGeoLocationRule struct {
	Policy *string `json:"policy,omitempty"`
	Region *string `json:"region,omitempty"`
	Note   *string `json:"note,omitempty"`
}

type TrafficConfigResponseAutonomousSystem struct {
	Policy *string `json:"policy,omitempty"`
	ASN    *int64  `json:"asn,omitempty"`
	Note   *string `json:"note,omitempty"`
}

func (d TrafficConfigData) ActiveChangeID() string {
	return d.Relationships.ActiveChange.Data.ID
}

type TrafficConfigResponse struct {
	Data TrafficConfigData `json:"data"`
}

type TrafficConfigsResponse struct {
	Data []TrafficConfigData `json:"data"`
}

type TrafficConfigChange struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func (c *Client) CreateTrafficConfig(ctx context.Context, reqData TrafficConfigRequest) (*TrafficConfigResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to create a traffic config")
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.TenantID
	if reqData.Data.Attributes.Version == "" {
		reqData.Data.Attributes.Version = "0.1.0" // Default version
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	path := "/api/v2/traffic-mgmt/traffic-configs"
	logTrafficConfigRequest(ctx, "POST", path, jsonData)

	req, err := c.NewRequest(ctx, "POST", path, bytes.NewBuffer(jsonData))
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
		return nil, httpStatusError(resp, "create traffic config")
	}

	var tcResp TrafficConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}

func (c *Client) GetTrafficConfig(ctx context.Context, id string) (*TrafficConfigResponse, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/traffic-configs/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get traffic config")
	}

	var tcResp TrafficConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}

func (c *Client) UpdateTrafficConfig(ctx context.Context, id string, reqData TrafficConfigRequest) (*TrafficConfigResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to update a traffic config")
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.TenantID
	if reqData.Data.Attributes.Version == "" {
		reqData.Data.Attributes.Version = "0.1.0"
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	path := "/api/v2/traffic-mgmt/traffic-configs/" + id
	logTrafficConfigRequest(ctx, "PUT", path, jsonData)

	req, err := c.NewRequest(ctx, "PUT", path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, httpStatusError(resp, "update traffic config")
	}

	var tcResp TrafficConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}

func logTrafficConfigRequest(ctx context.Context, method, path string, payload []byte) {
	tflog.Debug(ctx, "baffinbay traffic config request payload",
		map[string]interface{}{
			"method":  method,
			"path":    path,
			"payload": string(payload),
		},
	)
}

func (c *Client) DeleteTrafficConfig(ctx context.Context, id string) error {
	req, err := c.NewRequest(ctx, "DELETE", "/api/v2/traffic-mgmt/traffic-configs/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return httpStatusError(resp, "delete traffic config")
	}

	return nil
}

func (c *Client) GetTrafficConfigs(ctx context.Context) (*TrafficConfigsResponse, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/traffic-configs", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get traffic configs")
	}

	var tcResp TrafficConfigsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}

func (c *Client) GetTrafficConfigChange(ctx context.Context, trafficConfigID, changeID string) (*TrafficConfigChange, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/traffic-configs/"+trafficConfigID+"/changelog/"+changeID, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get traffic config change")
	}

	var change TrafficConfigChange
	if err := json.NewDecoder(resp.Body).Decode(&change); err != nil {
		return nil, err
	}

	return &change, nil
}

func (c *Client) WaitForTrafficConfigChange(ctx context.Context, trafficConfigID, changeID string) error {
	if changeID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, trafficConfigChangePollTimeout)
	defer cancel()

	for {
		change, err := c.GetTrafficConfigChange(ctx, trafficConfigID, changeID)
		if err != nil {
			return err
		}

		switch change.State {
		case "APPLIED":
			return nil
		case "FAILED":
			return fmt.Errorf("traffic config change %s failed", changeID)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for traffic config change %s: %w", changeID, ctx.Err())
		case <-time.After(trafficConfigChangePollInterval):
		}
	}
}
