package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	trafficConfigChangePollInterval = 2 * time.Second
	trafficConfigChangePollTimeout  = 2 * time.Minute
)

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
	Prefix           string            `json:"prefix,omitempty"`
	Announced        bool              `json:"announced,omitempty"`
	WAF              *WAF              `json:"waf,omitempty"`
	RateLimiting     *RateLimit        `json:"rateLimiting,omitempty"`
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
	} `json:"httpCompliance"`
	PathExclusions []any `json:"pathExclusions"`
}

type RateLimit struct {
	Enforcement string `json:"enforcement"`
}

type Frontend struct {
	IPv4                          string                         `json:"ipv4,omitempty"`
	IPv6                          string                         `json:"ipv6,omitempty"`
	IP                            string                         `json:"ip,omitempty"`
	Port                          int64                          `json:"port,omitempty"`
	ConnectionType                string                         `json:"connectionType,omitempty"`
	RedirectHttp                  bool                           `json:"redirectHttp,omitempty"`
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
	VerifyCrl        bool     `json:"verifyCrl,omitempty"`
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
	VerifyCrl        bool     `json:"verifyCrl,omitempty"`
}

type Host struct {
	Address string `json:"address"`
	Port    int64  `json:"port"`
}

type ProtocolSettings struct {
	Version          string `json:"httpVersion"`
	EnableWebsockets bool   `json:"enableWebsockets"`
	Multiplexing     bool   `json:"multiplexing"`
}

type TrafficConfigData struct {
	ID            string                  `json:"id"`
	Type          string                  `json:"type"`
	Attributes    TrafficConfigAttributes `json:"attributes"`
	Relationships struct {
		ActiveChange struct {
			Data struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"data"`
		} `json:"activeChange"`
	} `json:"relationships"`
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
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to create a traffic config")
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID
	reqData.Data.Attributes.Version = "0.1.0" // Default version

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(ctx, "POST", "/api/v2/traffic-mgmt/traffic-configs", bytes.NewBuffer(jsonData))
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
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to update a traffic config")
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID
	reqData.Data.Attributes.Version = "0.1.0"

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(ctx, "PUT", "/api/v2/traffic-mgmt/traffic-configs/"+id, bytes.NewBuffer(jsonData))
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
