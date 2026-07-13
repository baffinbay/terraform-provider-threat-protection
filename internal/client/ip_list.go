package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const IPListType = "ipList"

type IPListEntry struct {
	Value string  `json:"value"`
	Note  *string `json:"note"`
}

type IPListAttributes struct {
	Name    string        `json:"name"`
	Entries []IPListEntry `json:"entries"`
}

type IPListReference struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type IPListRelationships struct {
	BelongsTo struct {
		Data IPListReference `json:"data"`
	} `json:"belongsTo"`
	UsedBy struct {
		Data []IPListReference `json:"data"`
	} `json:"usedBy"`
	CreatedBy struct {
		Data IPListReference `json:"data"`
	} `json:"createdBy"`
	LastUpdatedBy struct {
		Data IPListReference `json:"data"`
	} `json:"lastUpdatedBy"`
}

type IPListMeta struct {
	CreatedAt     string `json:"createdAt"`
	LastUpdatedAt string `json:"lastUpdatedAt"`
}

type IPListData struct {
	ID            string              `json:"id"`
	Type          string              `json:"type"`
	Attributes    IPListAttributes    `json:"attributes"`
	Relationships IPListRelationships `json:"relationships"`
	Meta          IPListMeta          `json:"meta"`
}

type IPListResponse struct {
	Data IPListData `json:"data"`
}

type IPListsResponse struct {
	Data []IPListData `json:"data"`
}

type IPListRequest struct {
	Data struct {
		ID            string           `json:"id,omitempty"`
		Type          string           `json:"type"`
		Attributes    IPListAttributes `json:"attributes"`
		Relationships struct {
			BelongsTo struct {
				Data IPListReference `json:"data"`
			} `json:"belongsTo"`
		} `json:"relationships"`
	} `json:"data"`
}

func (c *Client) CreateIPList(ctx context.Context, name string, entries []IPListEntry) (*IPListResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to create an IP list")
	}

	reqData := c.newIPListRequest("", name, entries)
	return c.sendIPListRequest(ctx, http.MethodPost, "/api/v2/traffic-mgmt/ip-lists", reqData, http.StatusCreated, http.StatusOK)
}

func (c *Client) GetIPList(ctx context.Context, id string) (*IPListResponse, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/v2/traffic-mgmt/ip-lists/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get IP list")
	}

	var ipListResp IPListResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipListResp); err != nil {
		return nil, err
	}

	return &ipListResp, nil
}

func (c *Client) GetIPLists(ctx context.Context) (*IPListsResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to list IP lists")
	}

	path := "/api/v2/traffic-mgmt/ip-lists?filter[tenant-id]=" + url.QueryEscape(c.TenantID)
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get IP lists")
	}

	var ipListsResp IPListsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipListsResp); err != nil {
		return nil, err
	}

	return &ipListsResp, nil
}

func (c *Client) UpdateIPList(ctx context.Context, id, name string, entries []IPListEntry) (*IPListResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to update an IP list")
	}

	reqData := c.newIPListRequest(id, name, entries)
	return c.sendIPListRequest(ctx, http.MethodPut, "/api/v2/traffic-mgmt/ip-lists/"+url.PathEscape(id), reqData, http.StatusAccepted, http.StatusCreated, http.StatusOK)
}

func (c *Client) DeleteIPList(ctx context.Context, id string) error {
	req, err := c.NewRequest(ctx, http.MethodDelete, "/api/v2/traffic-mgmt/ip-lists/"+url.PathEscape(id), nil)
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
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return httpStatusError(resp, "delete IP list")
	}

	return nil
}

func (c *Client) newIPListRequest(id, name string, entries []IPListEntry) IPListRequest {
	reqData := IPListRequest{}
	reqData.Data.ID = id
	reqData.Data.Type = IPListType
	reqData.Data.Attributes = IPListAttributes{Name: name, Entries: entries}
	reqData.Data.Relationships.BelongsTo.Data = IPListReference{Type: "tenant", ID: c.TenantID}
	return reqData
}

func (c *Client) sendIPListRequest(ctx context.Context, method, path string, reqData IPListRequest, expectedStatuses ...int) (*IPListResponse, error) {
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

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

	validStatus := false
	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			validStatus = true
			break
		}
	}
	if !validStatus {
		action := "create IP list"
		if method == http.MethodPut {
			action = "update IP list"
		}
		return nil, httpStatusError(resp, action)
	}

	var ipListResp IPListResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipListResp); err != nil {
		return nil, err
	}

	return &ipListResp, nil
}
