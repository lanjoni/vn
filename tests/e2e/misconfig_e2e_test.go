//go:build e2e
// +build e2e

package e2e

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vn/internal/scanner"
)

// createE2ETestServer sets up a simple server for E2E testing.
func createE2ETestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.env" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret123"))
			return
		}
		if r.Method == "TRACE" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Default response for header and other checks
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
}

func TestMisconfig_E2E_FullScan(t *testing.T) {
	t.Parallel()
	server := createE2ETestServer()
	defer server.Close()

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 2,
		Tests:   []string{"files", "headers", "server"}, // Run a subset of tests for simplicity
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	if len(results) == 0 {
		t.Fatal("E2E scan returned no results")
	}

	// High-level checks for key vulnerability types
	foundFindings := make(map[string]bool)
	for _, res := range results {
		if strings.Contains(res.Finding, "Sensitive file exposed") {
			foundFindings["sensitive-file"] = true
		}
		if strings.Contains(res.Finding, "Missing security header") {
			foundFindings["missing-header"] = true
		}
		if strings.Contains(res.Finding, "Dangerous HTTP method enabled") {
			foundFindings["dangerous-method"] = true
		}
	}

	if !foundFindings["sensitive-file"] {
		t.Error("E2E scan did not detect the sensitive .env file")
	}
	if !foundFindings["missing-header"] {
		t.Error("E2E scan did not detect missing security headers")
	}
	if !foundFindings["dangerous-method"] {
		t.Error("E2E scan did not detect dangerous HTTP methods")
	}

	errors := misconfigScanner.GetErrors()
	if len(errors) > 0 {
		t.Errorf("E2E scan produced unexpected errors: %v", errors)
	}
}