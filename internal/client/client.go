package client

import (
	"io"
	"net/http"
	"time"
)

type Client struct {

	HostURL    string

	HTTPClient *http.Client

	Token      string

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
