package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Client struct {

	HostURL    string

	HTTPClient *http.Client

	Token      string

	AccountID  string
}

type OIDCAuthRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	Audience     string `json:"audience"`
}

type OIDCAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

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

func NewClient(host string) *Client {
	return &Client{
		HostURL: host,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SetAuth(token string) {
	c.Token = token
}

func (c *Client) Authenticate(oidcURL, clientID, clientSecret string) error {
	reqBody := OIDCAuthRequest{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		GrantType:    "client_credentials",
		Audience:     "https://portal.baffinbay.com",
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, oidcURL, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var authResp OIDCAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	c.Token = authResp.AccessToken
	return nil
}

func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.HostURL+path, body)
	if err != nil {
		return nil, err
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return req, nil
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
	part.Write([]byte(content))

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
	dataPart.Write(jsonData)

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete custom page (status %d)", resp.StatusCode)
	}

	return nil
}

type CertificateRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Certificate  string `json:"certificate,omitempty"`
			Intermediate string `json:"intermediate,omitempty"`
			Key          string `json:"key,omitempty"`
			FQDN         string `json:"fqdn,omitempty"`
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

type CertificateResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *Client) CreateCertificate(certType, cert, intermediate, key, fqdn string) (*CertificateResponse, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to create a certificate")
	}

	reqData := CertificateRequest{}
	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID

	var path string
	if certType == "pem" {
		reqData.Data.Type = "importPemCertificate"
		reqData.Data.Attributes.Certificate = cert
		reqData.Data.Attributes.Intermediate = intermediate
		reqData.Data.Attributes.Key = key
		path = "/api/v2/traffic-mgmt/certificates/pem"
	} else if certType == "lets_encrypt" {
		reqData.Data.Type = "letsEncrypt"
		reqData.Data.Attributes.FQDN = fqdn
		path = "/api/v2/traffic-mgmt/certificates/lets-encrypt"
	} else {
		return nil, fmt.Errorf("invalid certificate type: %s", certType)
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.HostURL+path, bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("failed to create certificate (status %d): %s", resp.StatusCode, string(respBody))
	}

	var certResp CertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&certResp); err != nil {
		return nil, err
	}

	return &certResp, nil
}

func (c *Client) DeleteCertificate(id string) error {
	req, err := c.NewRequest("DELETE", "/api/v2/traffic-mgmt/certificates/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete certificate (status %d)", resp.StatusCode)
	}

	return nil
}

type CaBundleRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Name        string `json:"name"`
			Certificate string `json:"certificate"`
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

type CaBundleResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *Client) CreateCaBundle(name, certificate string) (*CaBundleResponse, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to create a CA bundle")
	}

	reqData := CaBundleRequest{}
	reqData.Data.Type = "caBundle"
	reqData.Data.Attributes.Name = name
	reqData.Data.Attributes.Certificate = certificate
	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.HostURL+"/api/v2/traffic-mgmt/ca-bundles", bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("failed to create CA bundle (status %d): %s", resp.StatusCode, string(respBody))
	}

	var caResp CaBundleResponse
	if err := json.NewDecoder(resp.Body).Decode(&caResp); err != nil {
		return nil, err
	}

	return &caResp, nil
}

func (c *Client) DeleteCaBundle(id string) error {
	req, err := c.NewRequest("DELETE", "/api/v2/traffic-mgmt/ca-bundles/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete CA bundle (status %d)", resp.StatusCode)
	}

	return nil
}

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

func (c *Client) Ping() error {
	req, err := c.NewRequest("GET", "/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ping failed: %d", resp.StatusCode)
	}
	return nil
}

