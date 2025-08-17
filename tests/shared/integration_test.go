//go:build integration
// +build integration

package shared

import (
	"net/http"
	"os"
	"testing"

	"vn/tests/shared/fixtures"
	"vn/tests/shared/testserver"
)

func TestSharedInfrastructureIntegration(t *testing.T) {
	infra := SetupTest(t)

	// Test BuildManager
	binaryPath, err := infra.BuildManager.GetBinary("vn")
	if err != nil {
		t.Fatalf("Failed to get binary: %v", err)
	}

	if binaryPath == "" {
		t.Fatal("Binary path is empty")
	}

	// Test ServerPool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("integration test"))
	})

	timeouts := GetOptimizedTimeouts()
	config := testserver.ServerConfig{
		Handler:    handler,
		ConfigName: "integration-test",
		Timeout:    timeouts.ServerStart,
	}

	server, err := infra.ServerPool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get server: %v", err)
	}
	defer infra.ServerPool.ReleaseServer(server)

	// Test server is working
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test MockProvider
	responses := map[string]fixtures.MockResponse{
		"/test": {
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"message": "mock test"}`,
		},
	}

	mockServer := infra.MockProvider.CreateMockServer(responses)
	defer mockServer.Close()

	resp, err = http.Get(mockServer.URL + "/test")
	if err != nil {
		t.Fatalf("Failed to make request to mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected mock status 200, got %d", resp.StatusCode)
	}

	// Test HTTPBin mock
	httpbinMock := infra.MockProvider.CreateHTTPBinMock()
	defer httpbinMock.Close()

	resp, err = http.Get(httpbinMock.URL + "/get")
	if err != nil {
		t.Fatalf("Failed to make request to httpbin mock: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected httpbin mock status 200, got %d", resp.StatusCode)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	CleanupGlobalResources()
	os.Exit(code)
}
