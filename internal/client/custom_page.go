package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type CustomPageData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name      string `json:"name"`
		CreatedAt string `json:"createdAt"`
	} `json:"attributes"`
}

type CustomPageResponse struct {
	Data CustomPageData `json:"data"`
}

type CustomPageRequest struct {
	Data struct {
		Type          string   `json:"type"`
		Attributes    struct{} `json:"attributes"`
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

func (c *Client) CreateCustomPage(ctx context.Context, name, content string) (*CustomPageResponse, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to create a custom page")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file part
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}
	_, _ = part.Write([]byte(content))

	// Add data part (JSON)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="%s"`, "data")}
	h["Content-Type"] = []string{"application/vnd.api+json"}
	dataPart, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}

	reqData := CustomPageRequest{}
	reqData.Data.Type = "custom-page"
	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.TenantID

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	_, _ = dataPart.Write(jsonData)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(ctx, "POST", "/api/v2/traffic-mgmt/custom-pages", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, httpStatusError(resp, "create custom page")
	}

	var cpResp CustomPageResponse
	if err := json.NewDecoder(resp.Body).Decode(&cpResp); err != nil {
		return nil, err
	}

	return &cpResp, nil
}

func (c *Client) DeleteCustomPage(ctx context.Context, id string) error {
	req, err := c.NewRequest(ctx, "DELETE", "/api/v2/traffic-mgmt/custom-pages/"+id, nil)
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
		return httpStatusError(resp, "delete custom page")
	}

	return nil
}

type CustomPagesResponse struct {
	Data []CustomPageData `json:"data"`
}

func (c *Client) GetCustomPages(ctx context.Context, tenantID string) (*CustomPagesResponse, error) {
	path := "/api/v2/traffic-mgmt/custom-pages"
	if tenantID != "" {
		path += "?filter[tenantId]=" + url.QueryEscape(tenantID)
	}
	req, err := c.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get custom pages")
	}

	var cpResp CustomPagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&cpResp); err != nil {
		return nil, err
	}

	return &cpResp, nil
}

func (c *Client) FindCustomPageByID(ctx context.Context, id string) (*CustomPageData, error) {
	if c.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required to read a custom page")
	}

	cpResp, err := c.GetCustomPages(ctx, c.TenantID)
	if err != nil {
		return nil, err
	}

	for i := range cpResp.Data {
		if cpResp.Data[i].ID == id {
			return &cpResp.Data[i], nil
		}
	}

	return nil, &HTTPStatusError{Action: "find custom page", StatusCode: http.StatusNotFound}
}

func (c *Client) DownloadCustomPage(ctx context.Context, id string) (string, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/custom-pages/"+id+"/file", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", httpStatusError(resp, "download custom page")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
