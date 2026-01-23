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
			Name       string `json:"name"`
			Version    string `json:"version"`
			Frontend   *Frontend `json:"frontend,omitempty"`
			Backend    *Backend  `json:"backend,omitempty"`
			Deployment struct {
				State string `json:"state"`
			} `json:"deployment"`
			Protocols []string `json:"protocols,omitempty"`
			Prefix    string   `json:"prefix,omitempty"`
			Announced bool     `json:"announced,omitempty"`
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

type Frontend struct {
	IPv4 string `json:"ipv4,omitempty"`
	IPv6 string `json:"ipv6,omitempty"`
	Port int64  `json:"port,omitempty"`
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
	reqData.Data.Attributes.Version = "0.0.1" // Default version

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create traffic config (status %d): %s", resp.StatusCode, string(respBody))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete traffic config (status %d)", resp.StatusCode)
	}

	return nil
}
