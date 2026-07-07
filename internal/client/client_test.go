package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AliseMarfina/swordfish-verifier/internal/client"
	"github.com/AliseMarfina/swordfish-verifier/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		EmulatorURL: "http://localhost:5000",
		Timeout:     30,
	}

	c := client.NewClient(cfg)
	if c == nil {
		t.Errorf("Expected client to be created, got nil")
	}
}

func TestGetResource_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/redfish/v1/Storage/1/Volumes/1" {
			t.Errorf("Expected path /redfish/v1/Storage/1/Volumes/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Id":   "volume-1",
			"Name": "test-volume",
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
	}

	c := client.NewClient(cfg)
	body, err := c.GetResource("/redfish/v1/Storage/1/Volumes/1")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(body) == 0 {
		t.Errorf("Expected response body, got empty")
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	if data["Id"] != "volume-1" {
		t.Errorf("Expected Id 'volume-1', got %v", data["Id"])
	}
}

func TestGetResource_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Resource not found"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
	}

	c := client.NewClient(cfg)
	_, err := c.GetResource("/redfish/v1/nonexistent")

	if err == nil {
		t.Errorf("Expected error for 404 status, got nil")
	}
}

func TestGetResource_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
	}

	c := client.NewClient(cfg)
	_, err := c.GetResource("/redfish/v1/Storage/1/Volumes/1")

	if err == nil {
		t.Errorf("Expected error for 500 status, got nil")
	}
}

func TestGetResource_WithBasicAuth(t *testing.T) {
	var authHeaderReceived string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderReceived = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Id": "volume-1",
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
		Auth: config.Auth{
			Type:     "basic",
			Username: "admin",
			Password: "admin",
		},
	}

	c := client.NewClient(cfg)
	_, err := c.GetResource("/redfish/v1/Storage/1/Volumes/1")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if authHeaderReceived == "" {
		t.Errorf("Expected Authorization header to be set")
	}

	if authHeaderReceived[:6] != "Basic " {
		t.Errorf("Expected Basic auth header, got %s", authHeaderReceived)
	}
}

func TestListEndpoints_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/redfish/v1" {
			t.Errorf("Expected path /redfish/v1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"@odata.id": "/redfish/v1",
			"Links": map[string]interface{}{
				"Systems": map[string]interface{}{
					"@odata.id": "/redfish/v1/Systems",
				},
			},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
	}

	c := client.NewClient(cfg)
	body, err := c.ListEndpoints()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(body) == 0 {
		t.Errorf("Expected response body, got empty")
	}
}

func TestPing_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"@odata.id": "/redfish/v1",
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
	}

	c := client.NewClient(cfg)
	err := c.Ping()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestPing_FailedConnection(t *testing.T) {
	cfg := &config.Config{
		EmulatorURL: "http://invalid-server-that-does-not-exist:5000",
		Timeout:     1,
	}

	c := client.NewClient(cfg)
	err := c.Ping()

	if err == nil {
		t.Errorf("Expected error for failed connection, got nil")
	}
}

func TestGetResource_AcceptsJSON(t *testing.T) {
	var acceptHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Id": "volume-1",
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		EmulatorURL: server.URL,
		Timeout:     5,
	}

	c := client.NewClient(cfg)
	_, err := c.GetResource("/redfish/v1/Storage/1/Volumes/1")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if acceptHeader != "application/json" {
		t.Errorf("Expected Accept header to be 'application/json', got %s", acceptHeader)
	}
}
