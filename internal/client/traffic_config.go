package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TrafficConfigRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Name       string    `json:"name"`
			Version    string    `json:"version"`
			Frontend   *Frontend `json:"frontend,omitempty"`
			Backend    *Backend  `json:"backend,omitempty"`
			Deployment struct {
				State string `json:"state"`
			} `json:"deployment"`
			Protocols    []string   `json:"protocols,omitempty"`
			Prefix       string     `json:"prefix,omitempty"`
			Announced    bool       `json:"announced,omitempty"`
			WAF          *WAF       `json:"waf,omitempty"`
			RateLimiting *RateLimit `json:"rateLimiting,omitempty"`
			ProtocolSettings *ProtocolSettings `json:"protocolSettings,omitempty"`
		} `json:"attributes"`
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
	PathExclusions []interface{} `json:"pathExclusions"`
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
	Mode        string   `json:"mode"`
	VerifyCrl   bool     `json:"verifyCrl,omitempty"`
	CaBundleIds []string `json:"caBundleIds,omitempty"`
}

type Backend struct {
	Hosts          []Host `json:"hosts"`
	DeliveryMethod string `json:"deliveryMethod"`
	ServerName     string `json:"serverName"`
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

type TrafficConfigResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *Client) CreateTrafficConfig(reqData TrafficConfigRequest) (*TrafficConfigResponse, error) {
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

	req, err := http.NewRequest("POST", c.HostURL+"/api/v2/traffic-mgmt/traffic-configs", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create traffic config (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tcResp TrafficConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}

func (c *Client) UpdateTrafficConfig(id string, reqData TrafficConfigRequest) (*TrafficConfigResponse, error) {
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

	req, err := http.NewRequest("PATCH", c.HostURL+"/api/v2/traffic-mgmt/traffic-configs/"+id, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update traffic config (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tcResp TrafficConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}

func (c *Client) DeleteTrafficConfig(id string) error {
	req, err := c.NewRequest("DELETE", "/api/v2/traffic-mgmt/traffic-configs/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete traffic config (status %d)", resp.StatusCode)
	}

	return nil
}

type TrafficConfigsResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetTrafficConfigs() (*TrafficConfigsResponse, error) {
	req, err := c.NewRequest("GET", "/api/v2/traffic-mgmt/traffic-configs", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get traffic configs (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tcResp TrafficConfigsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tcResp); err != nil {
		return nil, err
	}

	return &tcResp, nil
}
