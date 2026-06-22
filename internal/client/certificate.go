package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type CertificateAttributes struct {
	CommonName string `json:"commonName"`
	FQDN       string `json:"fqdn"`
}

type CertificateData struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Attributes CertificateAttributes `json:"attributes"`
}

type CertificateResponse struct {
	Data CertificateData `json:"data"`
}

type CertificatesResponse struct {
	Data []CertificateData `json:"data"`
}

func (c *Client) CreateCertificate(ctx context.Context, certType, cert, intermediate, key, fqdn string) (*CertificateResponse, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to create a certificate")
	}

	reqData := CertificateRequest{}
	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID

	var path string
	switch certType {
	case "pem":
		reqData.Data.Type = "importPemCertificate"
		reqData.Data.Attributes.Certificate = cert
		reqData.Data.Attributes.Intermediate = intermediate
		reqData.Data.Attributes.Key = key
		path = "/api/v2/traffic-mgmt/certificates/pem"
	case "lets_encrypt":
		reqData.Data.Type = "letsEncrypt"
		reqData.Data.Attributes.FQDN = fqdn
		path = "/api/v2/traffic-mgmt/certificates/lets-encrypt"
	default:
		return nil, fmt.Errorf("invalid certificate type: %s", certType)
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(ctx, "POST", path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, httpStatusError(resp, "create certificate")
	}

	var certResp CertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&certResp); err != nil {
		return nil, err
	}

	return &certResp, nil
}

func (c *Client) DeleteCertificate(ctx context.Context, id string) error {
	req, err := c.NewRequest(ctx, "DELETE", "/api/v2/traffic-mgmt/certificates/"+id, nil)
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
		return httpStatusError(resp, "delete certificate")
	}

	return nil
}

type CaCertificateRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Certificates []CaCertificateRequestCertificate `json:"certificates"`
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

type CaCertificateRequestCertificate struct {
	Certificate string `json:"certificate"`
	CRLURL      string `json:"crlUrl,omitempty"`
}

type CaCertificateAttributes struct {
	Name         string                         `json:"name"`
	Certificates []CaCertificateDataCertificate `json:"certificates"`
}

type CaCertificateDataCertificate struct {
	Certificate string `json:"certificate"`
	CRLURL      string `json:"crlUrl"`
}

type CaCertificateData struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Attributes CaCertificateAttributes `json:"attributes"`
}

type CaCertificateResponse struct {
	Data CaCertificateData `json:"data"`
}

type CaCertificatesResponse struct {
	Data []CaCertificateData `json:"data"`
}

func (c *Client) CreateCaCertificate(ctx context.Context, _ string, certificate string) (*CaCertificateResponse, error) {
	if c.AccountID == "" {
		return nil, fmt.Errorf("account_id is required to create a CA certificate")
	}

	reqData := CaCertificateRequest{}
	reqData.Data.Type = "caCertificate"
	reqData.Data.Attributes.Certificates = []CaCertificateRequestCertificate{{Certificate: certificate}}
	reqData.Data.Relationships.BelongsTo.Data.Type = "account"
	reqData.Data.Relationships.BelongsTo.Data.ID = c.AccountID

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(ctx, "POST", "/api/v2/traffic-mgmt/ca-certificates", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, httpStatusError(resp, "create CA certificate")
	}

	var caResp CaCertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&caResp); err != nil {
		return nil, err
	}

	return &caResp, nil
}

func (c *Client) GetCaCertificate(ctx context.Context, id string) (*CaCertificateResponse, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/ca-certificates/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get CA certificate")
	}

	var caResp CaCertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&caResp); err != nil {
		return nil, err
	}

	return &caResp, nil
}

func (c *Client) DeleteCaCertificate(ctx context.Context, id string) error {
	req, err := c.NewRequest(ctx, "DELETE", "/api/v2/traffic-mgmt/ca-certificates/"+id, nil)
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
		return httpStatusError(resp, "delete CA certificate")
	}

	return nil
}

func (c *Client) GetCertificates(ctx context.Context) (*CertificatesResponse, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/certificates", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get certificates")
	}

	var certsResp CertificatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&certsResp); err != nil {
		return nil, err
	}

	return &certsResp, nil
}

func (c *Client) FindCertificateByID(ctx context.Context, id string) (*CertificateData, error) {
	certsResp, err := c.GetCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for i := range certsResp.Data {
		if certsResp.Data[i].ID == id {
			return &certsResp.Data[i], nil
		}
	}

	return nil, &HTTPStatusError{Action: "find certificate", StatusCode: http.StatusNotFound}
}

func (c *Client) GetCaCertificates(ctx context.Context) (*CaCertificatesResponse, error) {
	req, err := c.NewRequest(ctx, "GET", "/api/v2/traffic-mgmt/ca-certificates", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError(resp, "get CA certificates")
	}

	var caResp CaCertificatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&caResp); err != nil {
		return nil, err
	}

	return &caResp, nil
}
