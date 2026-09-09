package client

import (
	"context"
	"fmt"
	"net/http"
)

type L4ProxyRequest struct {
	Data L4ProxyRequestData `json:"data"`
}

type L4ProxyRequestData struct {
	Type          string                        `json:"type"`
	Attributes    L4ProxyAttributes             `json:"attributes"`
	Relationships TrafficConfigRequestRelations `json:"relationships"`
}

type L4ProxyAttributes struct {
	Name          string                `json:"name"`
	Version       string                `json:"version"`
	Frontend      *L4ProxyFrontend      `json:"frontend,omitempty"`
	Backend       *L4ProxyBackend       `json:"backend,omitempty"`
	Deployment    Deployment            `json:"deployment"`
	Protocols     []string              `json:"protocols,omitempty"`
	ProxyProtocol string                `json:"proxyProtocol,omitempty"`
	IPBasedAccess *L4ProxyIPBasedAccess `json:"ipBasedAccessControl,omitempty"`
	GatewayPath   string                `json:"gatewayPath,omitempty"`
}

type L4ProxyFrontend struct {
	IPv4 string `json:"ipv4,omitempty"`
	IPv6 string `json:"ipv6,omitempty"`
	IP   string `json:"ip,omitempty"`
	Port int64  `json:"port,omitempty"`
}

type L4ProxyBackend struct {
	Hosts          []L4ProxyHost `json:"hosts"`
	DeliveryMethod string        `json:"deliveryMethod"`
	ServerName     string        `json:"serverName,omitempty"`
}

type L4ProxyHost struct {
	Address string `json:"address"`
	Port    int64  `json:"port"`
}

type L4ProxyIPBasedAccess struct {
	DefaultPolicy string                    `json:"defaultPolicy"`
	Rules         L4ProxyIPBasedAccessRules `json:"rules"`
}

type L4ProxyIPBasedAccessRules struct {
	IPRanges     []L4ProxyIPBasedAccessIPRangeRule      `json:"ipRanges"`
	IPLists      []L4ProxyIPBasedAccessIPListRule       `json:"ipLists"`
	GeoLocations []L4ProxyIPBasedAccessGeoLocationRule  `json:"geoLocations"`
	ASNs         []L4ProxyIPBasedAccessAutonomousSystem `json:"asns"`
}

type L4ProxyIPBasedAccessIPRangeRule struct {
	Policy  string `json:"policy"`
	Address string `json:"address"`
	Note    string `json:"note,omitempty"`
}

type L4ProxyIPBasedAccessIPListRule struct {
	Type   string `json:"type,omitempty"`
	ID     string `json:"id"`
	Policy string `json:"policy"`
}

type L4ProxyIPBasedAccessGeoLocationRule struct {
	Policy string `json:"policy"`
	Region string `json:"region"`
	Note   string `json:"note,omitempty"`
}

type L4ProxyIPBasedAccessAutonomousSystem struct {
	Policy string `json:"policy"`
	ASN    int64  `json:"asn"`
	Note   string `json:"note,omitempty"`
}

type L4ProxyResponse = TrafficConfigResponse

func NewL4ProxyRequest(attributes L4ProxyAttributes) L4ProxyRequest {
	return L4ProxyRequest{
		Data: L4ProxyRequestData{
			Type:       "l4Proxy",
			Attributes: attributes,
		},
	}
}

func (c *Client) CreateL4Proxy(ctx context.Context, reqData L4ProxyRequest) (*L4ProxyResponse, error) {
	return c.mutateL4Proxy(ctx, http.MethodPost, "/api/v2/traffic-mgmt/traffic-configs", "create L4 Proxy traffic config", reqData)
}

func (c *Client) UpdateL4Proxy(ctx context.Context, id string, reqData L4ProxyRequest) (*L4ProxyResponse, error) {
	return c.mutateL4Proxy(ctx, http.MethodPut, "/api/v2/traffic-mgmt/traffic-configs/"+id, "update L4 Proxy traffic config", reqData)
}

func (c *Client) GetL4Proxy(ctx context.Context, id string) (*L4ProxyResponse, error) {
	return c.GetTrafficConfig(ctx, id)
}

func (c *Client) mutateL4Proxy(ctx context.Context, method, path, operation string, reqData L4ProxyRequest) (*L4ProxyResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to %s", operation)
	}

	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.TenantID

	var proxyResp L4ProxyResponse
	if err := c.mutateTrafficConfig(ctx, method, path, operation, reqData, &proxyResp, http.StatusOK, http.StatusCreated, http.StatusAccepted); err != nil {
		return nil, err
	}
	return &proxyResp, nil
}
