package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AliseMarfina/swordfish-verifier/internal/config"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	authType   string
	username   string
	password   string
	token      string
	retryCount int
	endpoints  []string
}

func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.EmulatorURL == "" {
		return nil, fmt.Errorf("emulator_url is required")
	}
	client := &Client{
		baseURL:    strings.TrimRight(cfg.EmulatorURL, "/"),
		httpClient: &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second},
		authType:   cfg.Auth.Type,
		username:   cfg.Auth.Username,
		password:   cfg.Auth.Password,
		retryCount: cfg.RetryCount,
		endpoints:  cfg.EndpointsFilter,
	}
	return client, nil
}

// authenticate выполняет сессионную аутентификацию и сохраняет токен.
func (c *Client) authenticate() error {
	body := map[string]string{
		"UserName": c.username,
		"Password": c.password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := c.httpClient.Post(c.baseURL+"/redfish/v1/SessionService/Sessions", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("auth failed: status %s", resp.Status)
	}

	// Пробуем взять токен из заголовка X-Auth-Token (эмулятор отдаёт его именно сюда)
	token := resp.Header.Get("X-Auth-Token")
	if token == "" {
		// Если нет в заголовке, пробуем из тела (поле token)
		var data struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return fmt.Errorf("failed to decode auth response: %w", err)
		}
		token = data.Token
	}
	if token == "" {
		return fmt.Errorf("no token found in response")
	}
	c.token = token
	return nil
}

func (c *Client) ensureToken() error {
	if c.authType == "session" && c.token == "" {
		return c.authenticate()
	}
	return nil
}

func (c *Client) Get(endpoint string) ([]byte, int, error) {
	if err := c.ensureToken(); err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, 0, err
	}

	switch c.authType {
	case "basic":
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	case "session":
		if c.token == "" {
			return nil, 0, fmt.Errorf("no session token available")
		}
		// ИЗМЕНЕНИЕ: используем Authorization: Bearer вместо X-Auth-Token
		req.Header.Set("Authorization", "Bearer "+c.token)
	default:
		return nil, 0, fmt.Errorf("unsupported auth type: %s", c.authType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	// Переавторизация при 401 (если есть попытки)
	if resp.StatusCode == http.StatusUnauthorized && c.authType == "session" && c.retryCount > 0 {
		if err := c.authenticate(); err != nil {
			return nil, resp.StatusCode, err
		}
		c.retryCount--
		return c.Get(endpoint)
	}

	return body, resp.StatusCode, nil
}

func (c *Client) Post(endpoint string, body []byte) ([]byte, int, error) {
	if err := c.ensureToken(); err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	switch c.authType {
	case "basic":
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	case "session":
		if c.token == "" {
			return nil, 0, fmt.Errorf("no session token available")
		}
		// ИЗМЕНЕНИЕ: используем Authorization: Bearer вместо X-Auth-Token
		req.Header.Set("Authorization", "Bearer "+c.token)
	default:
		return nil, 0, fmt.Errorf("unsupported auth type: %s", c.authType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode == http.StatusUnauthorized && c.authType == "session" && c.retryCount > 0 {
		if err := c.authenticate(); err != nil {
			return nil, resp.StatusCode, err
		}
		c.retryCount--
		return c.Post(endpoint, body)
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) GetEndpoints(filter []string) ([]string, error) {
	all, err := c.discoverEndpoints()
	if err != nil {
		return nil, err
	}

	return filterEndpoints(all, c, filter), nil
}

func (c *Client) discoverEndpoints() ([]string, error) {
	var endpoints []string
	visited := make(map[string]bool)
	queue := []string{"/redfish/v1"}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		if visited[path] {
			continue
		}
		visited[path] = true

		body, status, err := c.Get(path)
		if err != nil {
			continue
		}
		endpoints = append(endpoints, path)
		if status != http.StatusOK {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			continue
		}
		extractLinks(data, &queue)
	}
	return endpoints, nil
}

func extractLinks(data interface{}, queue *[]string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if key == "@odata.id" {
				if id, ok := val.(string); ok {
					*queue = append(*queue, id)
				}
			}
			if str, ok := val.(string); ok && strings.HasPrefix(str, "/redfish/") {
				*queue = append(*queue, str)
			}
			extractLinks(val, queue)
		}
	case []interface{}:
		for _, item := range v {
			extractLinks(item, queue)
		}
	}
}

func shouldInclude(path string, body []byte, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return false
	}
	odataType, ok := data["@odata.type"].(string)
	if !ok {
		return true
	}
	for _, f := range filter {
		if strings.Contains(strings.ToLower(odataType), strings.ToLower(f)) {
			return true
		}
	}
	return false
}

func filterEndpoints(endpoints []string, client *Client, filter []string) []string {
	if len(filter) == 0 {
		return endpoints
	}
	var result []string
	for _, ep := range endpoints {
		body, status, err := client.Get(ep)
		if err != nil || status != http.StatusOK {
			continue
		}
		if shouldInclude(ep, body, filter) {
			result = append(result, ep)
		}
	}
	return result
}
