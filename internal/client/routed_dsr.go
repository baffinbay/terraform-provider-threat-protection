package client

import (
	"context"
	"fmt"
	"net/http"
)

type RoutedDsrRequest struct {
	Data RoutedDsrRequestData `json:"data"`
}

type RoutedDsrRequestData struct {
	Type          string                        `json:"type"`
	Attributes    RoutedDsrAttributes           `json:"attributes"`
	Relationships TrafficConfigRequestRelations `json:"relationships"`
}

type RoutedDsrAttributes struct {
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	Prefix     string     `json:"prefix"`
	Announced  *bool      `json:"announced,omitempty"`
	Deployment Deployment `json:"deployment"`
}

type RoutedDsrResponse = TrafficConfigResponse

func NewRoutedDsrRequest(attributes RoutedDsrAttributes) RoutedDsrRequest {
	return RoutedDsrRequest{
		Data: RoutedDsrRequestData{
			Type:       "routedDsr",
			Attributes: attributes,
		},
	}
}

func (c *Client) CreateRoutedDsr(ctx context.Context, reqData RoutedDsrRequest) (*RoutedDsrResponse, error) {
	return c.mutateRoutedDsr(ctx, http.MethodPost, "/api/v2/traffic-mgmt/traffic-configs", "create Routed DSR traffic config", reqData)
}

func (c *Client) UpdateRoutedDsr(ctx context.Context, id string, reqData RoutedDsrRequest) (*RoutedDsrResponse, error) {
	return c.mutateRoutedDsr(ctx, http.MethodPut, "/api/v2/traffic-mgmt/traffic-configs/"+id, "update Routed DSR traffic config", reqData)
}

func (c *Client) GetRoutedDsr(ctx context.Context, id string) (*RoutedDsrResponse, error) {
	return c.GetTrafficConfig(ctx, id)
}

func (c *Client) mutateRoutedDsr(ctx context.Context, method, path, operation string, reqData RoutedDsrRequest) (*RoutedDsrResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to %s", operation)
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.TenantID

	var dsrResp RoutedDsrResponse
	if err := c.mutateTrafficConfig(ctx, method, path, operation, reqData, &dsrResp, http.StatusOK, http.StatusCreated, http.StatusAccepted); err != nil {
		return nil, err
	}
	return &dsrResp, nil
}
