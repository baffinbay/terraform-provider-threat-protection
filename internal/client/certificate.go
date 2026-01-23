package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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
