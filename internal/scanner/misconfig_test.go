package scanner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupTestServer is a helper function to create a test server for misconfiguration scans.
func setupTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// newTestScanner creates a new MisconfigScanner with a default configuration for testing.
func newTestScanner(url string) *MisconfigScanner {
	config := MisconfigConfig{
		URL:     url,
		Method:  "GET",
		Timeout: 5 * time.Second,
		Threads: 1,
	}
	return NewMisconfigScanner(config)
}

func TestMisconfigScanner_SensitiveFiles(t *testing.T) {
	t.Parallel()
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret"))
		case "/backup.sql":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("CREATE TABLE..."))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer server.Close()

	scanner := newTestScanner(server.URL)
	results := scanner.TestSensitiveFiles()

	if len(results) < 2 {
		t.Errorf("expected at least 2 sensitive files, got %d", len(results))
	}

	foundEnv := false
	foundBackup := false
	for _, res := range results {
		if strings.Contains(res.URL, "/.env") {
			foundEnv = true
		}
		if strings.Contains(res.URL, "/backup.sql") {
			foundBackup = true
		}
	}

	if !foundEnv {
		t.Error("expected to find .env file")
	}
	if !foundBackup {
		t.Error("expected to find backup.sql file")
	}
}

func TestMisconfigScanner_SecurityHeaders(t *testing.T) {
	t.Parallel()
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		// No security headers set
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	scanner := newTestScanner(server.URL)
	results := scanner.TestSecurityHeaders()

	if len(results) < 3 {
		t.Errorf("expected at least 3 missing header findings, got %d", len(results))
	}

	expectedHeaders := []string{"X-Frame-Options", "X-Content-Type-Options", "Strict-Transport-Security"}
	for _, expected := range expectedHeaders {
		found := false
		for _, res := range results {
			if strings.Contains(res.Finding, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find missing header: %s", expected)
		}
	}
}

func TestMisconfigScanner_DefaultCredentials(t *testing.T) {
	t.Parallel()
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			user := r.Form.Get("username")
			pass := r.Form.Get("password")
			if (user == "admin" && pass == "admin") || (user == "root" && pass == "root") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Welcome to Dashboard"))
				return
			}
		}
		// Serve login form for GET requests
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<form method="post"><input name="username"><input type="password" name="password"></form>`))
	})
	defer server.Close()

	scanner := newTestScanner(server.URL)
	results := scanner.TestDefaultCredentials()

	if len(results) == 0 {
		t.Error("expected to find default credentials vulnerability")
	}

	found := false
	for _, res := range results {
		if res.Finding == "Default credentials accepted" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Default credentials accepted' finding")
	}
}

func TestMisconfigScanner_ErrorHandling(t *testing.T) {
	t.Parallel()
	// Server that immediately closes to simulate a connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	scanner := newTestScanner(server.URL)
	scanner.Scan() // Run a full scan

	errors := scanner.GetErrors()
	if len(errors) == 0 {
		t.Fatal("expected connection errors, but got none")
	}

	for _, err := range errors {
		if !strings.Contains(err.Error(), "connection refused") {
			t.Errorf("expected error to contain 'connection refused', got %v", err)
		}
	}
}

func TestMisconfigScanner_FullScan(t *testing.T) {
	t.Parallel()
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		// A handler for a comprehensive test
		if r.URL.Path == "/.env" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("SECRET_KEY=12345"))
			return
		}
		// Missing security headers is the default
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	defer server.Close()

	scanner := newTestScanner(server.URL)
	// Explicitly set the tests to run
	scanner.config.Tests = []string{"files", "headers"}
	results := scanner.Scan()

	if len(results) < 4 { // 1 sensitive file + 3 missing headers
		t.Errorf("expected at least 4 findings, got %d", len(results))
	}

	categories := make(map[string]int)
	for _, res := range results {
		categories[res.Category]++
	}

	if categories["sensitive-files"] == 0 {
		t.Error("expected findings in category 'sensitive-files'")
	}
	if categories["headers"] == 0 {
		t.Error("expected findings in category 'headers'")
	}
}
