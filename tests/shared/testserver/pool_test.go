//go:build fast
// +build fast

package testserver

import (
	"net/http"
	"testing"
	"time"
)

func TestServerPool_GetServer(t *testing.T) {
	t.Parallel()
	pool := NewServerPool()
	defer pool.Shutdown()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	config := ServerConfig{
		Handler:    handler,
		ConfigName: "test-server",
		Timeout:    5 * time.Second,
	}

	server, err := pool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get server: %v", err)
	}

	if server.URL == "" {
		t.Fatal("Server URL is empty")
	}

	if server.Port == 0 {
		t.Fatal("Server port is 0")
	}

	// Test server is working
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestServerPool_ReuseServer(t *testing.T) {
	t.Parallel()
	pool := NewServerPool()
	defer pool.Shutdown()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := ServerConfig{
		Handler:    handler,
		ConfigName: "reuse-test",
	}

	// Get server first time
	server1, err := pool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get server first time: %v", err)
	}

	// Release server
	pool.ReleaseServer(server1)

	// Get server second time - should reuse
	server2, err := pool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get server second time: %v", err)
	}

	if server1.URL != server2.URL {
		t.Fatalf("Expected same server URL, got %s and %s", server1.URL, server2.URL)
	}
}

func TestServerPool_TLSServer(t *testing.T) {
	t.Parallel()
	pool := NewServerPool()
	defer pool.Shutdown()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := ServerConfig{
		Handler:    handler,
		TLS:        true,
		ConfigName: "tls-test",
	}

	server, err := pool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get TLS server: %v", err)
	}

	if server.URL == "" {
		t.Fatal("TLS server URL is empty")
	}

	// TLS server URL should start with https
	if len(server.URL) < 8 || server.URL[:8] != "https://" {
		t.Fatalf("Expected HTTPS URL, got %s", server.URL)
	}
}

func TestServerPool_Shutdown(t *testing.T) {
	t.Parallel()
	pool := NewServerPool()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := ServerConfig{
		Handler:    handler,
		ConfigName: "shutdown-test",
	}

	server, err := pool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get server: %v", err)
	}

	// Verify server is working
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request before shutdown: %v", err)
	}
	resp.Body.Close()

	// Shutdown pool
	pool.Shutdown()

	// Server should no longer be accessible
	_, err = http.Get(server.URL)
	if err == nil {
		t.Fatal("Expected error after shutdown, but request succeeded")
	}
}
