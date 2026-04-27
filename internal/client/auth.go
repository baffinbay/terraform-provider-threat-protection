package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTokenTimeFile = ".token_time"
	defaultEnvFile       = ".env"
	refreshLimit         = 12 * time.Hour
)

func findProjectRoot() (string, error) {
	curr, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(curr, ".git")); err == nil {
			return curr, nil
		}
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", fmt.Errorf("could not find project root (.git or go.mod)")
		}
		curr = parent
	}
}

func (c *Client) getTokenTimePath() string {
	if p := os.Getenv("BAFFINBAY_TOKEN_TIME_PATH"); p != "" {
		return p
	}
	if c.BaseDir != "" {
		return filepath.Join(c.BaseDir, defaultTokenTimeFile)
	}
	root, err := findProjectRoot()
	if err != nil {
		return defaultTokenTimeFile
	}
	return filepath.Join(root, defaultTokenTimeFile)
}

func (c *Client) getEnvPath() string {
	if p := os.Getenv("BAFFINBAY_ENV_PATH"); p != "" {
		return p
	}
	if c.BaseDir != "" {
		return filepath.Join(c.BaseDir, defaultEnvFile)
	}
	root, err := findProjectRoot()
	if err != nil {
		return defaultEnvFile
	}
	return filepath.Join(root, defaultEnvFile)
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

func (c *Client) SetAuth(token string) {
	c.Token = token
}

// Authenticate implements a persistent API key cache.
func (c *Client) Authenticate(oidcURL, clientID, clientSecret string, force bool) error {
	// 0. Skip persistency and refresh limits if force is set (useful for tests)
	if force {
		return c.performOIDC(oidcURL, clientID, clientSecret)
	}

	// 1. Check if we already have a token and it works
	if c.Token != "" {
		if err := c.Ping(); err == nil {
			return nil
		}
	}

	// 2. Check for token in .env if not already set or failed Ping
	envToken := os.Getenv("BAFFINBAY_API_KEY")
	if envToken != "" && envToken != c.Token {
		c.Token = envToken
		if err := c.Ping(); err == nil {
			return nil
		}
	}

	// 3. Safety check: avoid frequent refreshes (12h limit)
	if !c.canRefresh() {
		return fmt.Errorf("token expired or invalid, and refresh limit (12h) not yet reached. Please check your credentials or wait")
	}

	return c.performOIDC(oidcURL, clientID, clientSecret)
}

func (c *Client) performOIDC(oidcURL, clientID, clientSecret string) error {
	// 4. Perform OIDC exchange
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var authResp OIDCAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	c.Token = authResp.AccessToken

	// 5. Update .env and .token_time
	// Only persist if we aren't in a test environment (BaseDir would be set in tests)
	// Also skip if TF_ACC=1 to avoid overwriting real .env with mock-token
	if c.BaseDir == "" && os.Getenv("TF_ACC") != "1" {
		if err := c.updateEnvFile(c.Token); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update .env file: %v\n", err)
		}
		if err := c.updateTokenTime(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update .token_time file: %v\n", err)
		}
	}

	return nil
}

func (c *Client) canRefresh() bool {
	data, err := os.ReadFile(c.getTokenTimePath())
	if err != nil {
		return true // File missing, assume we can refresh
	}

	lastRefreshUnix, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true // Corrupt file, allow refresh
	}

	lastRefresh := time.Unix(lastRefreshUnix, 0)
	return time.Since(lastRefresh) > refreshLimit
}

func (c *Client) updateTokenTime() error {
	now := time.Now().Unix()
	return os.WriteFile(c.getTokenTimePath(), []byte(strconv.FormatInt(now, 10)), 0644)
}

func (c *Client) updateEnvFile(newToken string) error {
	path := c.getEnvPath()
	file, err := os.Open(path)
	if err != nil {
		// If .env doesn't exist, we create a minimal one
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte("BAFFINBAY_API_KEY="+newToken+"\n"), 0644)
		}
		return err
	}
	defer func() { _ = file.Close() }()

	var lines []string
	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "BAFFINBAY_API_KEY=") {
			lines = append(lines, "BAFFINBAY_API_KEY="+newToken)
			found = true
		} else {
			lines = append(lines, line)
		}
	}

	if !found {
		lines = append(lines, "BAFFINBAY_API_KEY="+newToken)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
