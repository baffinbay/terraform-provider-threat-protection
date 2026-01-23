package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	HostURL    string
	HTTPClient *http.Client
	Token      string
	AccountID  string
}

func NewClient(host string) *Client {
	return &Client{
		HostURL: host,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
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



type IPSourcesResponse struct {

	Data []struct {

		ID         string `json:"id"`

		Type       string `json:"type"`

		Attributes struct {

			CIDR string `json:"cidr"`

		} `json:"attributes"`

	} `json:"data"`

}



func (c *Client) GetIpSources() (*IPSourcesResponse, error) {

	req, err := c.NewRequest("GET", "/api/v2/traffic-mgmt/tpc-ip-sources", nil)

	if err != nil {

		return nil, err

	}



	resp, err := c.HTTPClient.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()



	if resp.StatusCode != http.StatusOK {

		respBody, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to get IP sources (status %d): %s", resp.StatusCode, string(respBody))

	}



	var ipResp IPSourcesResponse

	if err := json.NewDecoder(resp.Body).Decode(&ipResp); err != nil {

		return nil, err

	}



	return &ipResp, nil

}
