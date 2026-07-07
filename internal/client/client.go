package client

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AliseMarfina/swordfish-verifier/internal/config"
)

// Client handles HTTP communication with Swordfish-like servers
type Client struct {
	httpClient *http.Client
	baseURL    string
	auth       *config.Auth
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		baseURL: cfg.EmulatorURL,
		auth:    &cfg.Auth,
	}
}

// GetResource fetches a resource from the server at the given endpoint
func (c *Client) GetResource(endpoint string) ([]byte, error) {
	url := strings.TrimSuffix(c.baseURL, "/") + endpoint
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if configured
	if c.auth != nil && c.auth.Type == "basic" && c.auth.Username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.auth.Username + ":" + c.auth.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resource from %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP error statuses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d for %s: %s", resp.StatusCode, url, string(body))
	}

	return body, nil
}

// ListEndpoints returns a list of available endpoints from the service root
func (c *Client) ListEndpoints() ([]byte, error) {
	return c.GetResource("/redfish/v1")
}

// Ping checks if the server is reachable
func (c *Client) Ping() error {
	_, err := c.ListEndpoints()
	return err
}
