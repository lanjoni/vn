//go:build fast
// +build fast

package fixtures

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestMockProvider_CreateMockServer(t *testing.T) {
	t.Parallel()
	provider := NewMockProvider()
	
	responses := map[string]MockResponse{
		"/test": {
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "test"}`,
		},
		"/error": {
			StatusCode: 500,
			Body:       "Internal Server Error",
		},
	}
	
	server := provider.CreateMockServer(responses)
	defer server.Close()
	
	// Test successful response
	resp, err := http.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}
	
	// Test error response
	resp, err = http.Get(server.URL + "/error")
	if err != nil {
		t.Fatalf("Failed to make error request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 500 {
		t.Fatalf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestMockProvider_NetworkDelay(t *testing.T) {
	t.Parallel()
	provider := NewMockProvider()
	provider.SimulateNetworkDelay(100 * time.Millisecond)
	
	responses := map[string]MockResponse{
		"/delayed": {
			StatusCode: 200,
			Body:       "delayed response",
		},
	}
	
	server := provider.CreateMockServer(responses)
	defer server.Close()
	
	start := time.Now()
	resp, err := http.Get(server.URL + "/delayed")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to make delayed request: %v", err)
	}
	defer resp.Body.Close()
	
	if duration < 100*time.Millisecond {
		t.Fatalf("Expected delay of at least 100ms, got %v", duration)
	}
}

func TestMockProvider_HTTPBinMock(t *testing.T) {
	t.Parallel()
	provider := NewMockProvider()
	server := provider.CreateHTTPBinMock()
	defer server.Close()
	
	// Test GET endpoint
	resp, err := http.Get(server.URL + "/get?param=value")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var getResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&getResponse); err != nil {
		t.Fatalf("Failed to decode GET response: %v", err)
	}
	
	args, ok := getResponse["args"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected args in response")
	}
	
	paramValues, ok := args["param"].([]interface{})
	if !ok || len(paramValues) == 0 {
		t.Fatal("Expected param in args")
	}
	
	if paramValues[0] != "value" {
		t.Fatalf("Expected param value 'value', got %v", paramValues[0])
	}
}

func TestMockProvider_HTTPBinStatus(t *testing.T) {
	t.Parallel()
	provider := NewMockProvider()
	server := provider.CreateHTTPBinMock()
	defer server.Close()
	
	// Test status endpoint
	resp, err := http.Get(server.URL + "/status/404")
	if err != nil {
		t.Fatalf("Failed to make status request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 404 {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestMockProvider_NetworkError(t *testing.T) {
	t.Parallel()
	provider := NewMockProvider()
	provider.SimulateNetworkError("internal_error")
	
	responses := map[string]MockResponse{
		"/error": {
			StatusCode: 200,
			Body:       "should not reach here",
		},
	}
	
	server := provider.CreateMockServer(responses)
	defer server.Close()
	
	resp, err := http.Get(server.URL + "/error")
	if err != nil {
		t.Fatalf("Failed to make error request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 500 {
		t.Fatalf("Expected status 500 due to error simulation, got %d", resp.StatusCode)
	}
}