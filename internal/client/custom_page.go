package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type CustomPageResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name      string `json:"name"`
			CreatedAt string `json:"createdAt"`
		} `json:"attributes"`
	} `json:"data"`
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

func (c *Client) CreateCustomPage(name, content string) (*CustomPageResponse, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to create a custom page")
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
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	_, _ = dataPart.Write(jsonData)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.HostURL+"/api/v2/traffic-mgmt/custom-pages", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create custom page (status %d): %s", resp.StatusCode, string(respBody))
	}

	var cpResp CustomPageResponse
	if err := json.NewDecoder(resp.Body).Decode(&cpResp); err != nil {
		return nil, err
	}

	return &cpResp, nil
}

func (c *Client) DeleteCustomPage(id string) error {
	req, err := c.NewRequest("DELETE", "/api/v2/traffic-mgmt/custom-pages/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete custom page (status %d)", resp.StatusCode)
	}

	return nil
}

type CustomPagesResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetCustomPages(tenantID string) (*CustomPagesResponse, error) {
	path := "/api/v2/traffic-mgmt/custom-pages"
	if tenantID != "" {
		path += "?filter[tenantId]=" + tenantID
	}
	req, err := c.NewRequest("GET", path, nil)
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
		return nil, fmt.Errorf("failed to get custom pages (status %d): %s", resp.StatusCode, string(respBody))
	}

	var cpResp CustomPagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&cpResp); err != nil {
		return nil, err
	}

	return &cpResp, nil
}
