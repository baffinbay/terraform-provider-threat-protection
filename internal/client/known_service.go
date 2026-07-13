package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

const KnownServiceType = "knownService"

type KnownServiceAttributes struct {
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
	Provider string   `json:"provider"`
}

type KnownServiceData struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Attributes KnownServiceAttributes `json:"attributes"`
}

type KnownServiceResponse struct {
	Data KnownServiceData `json:"data"`
}

type KnownServicesResponse struct {
	Data []KnownServiceData `json:"data"`
}

func (c *Client) GetKnownService(ctx context.Context, id string) (*KnownServiceResponse, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/v2/traffic-mgmt/known-services/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get known service")
	}

	var result KnownServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetKnownServices(ctx context.Context) (*KnownServicesResponse, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/v2/traffic-mgmt/known-services", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get known services")
	}

	var result KnownServicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
