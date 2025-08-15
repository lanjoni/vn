//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"vn/internal/scanner"
	"vn/tests/shared"
	"vn/tests/shared/testserver"
)

const (
	riskHigh               = "High"
	categorySensitiveFiles = "sensitive-files"
)

var sharedServerPool testserver.ServerPool

func getE2EServerPool() testserver.ServerPool {
	if sharedServerPool == nil {
		sharedServerPool = testserver.NewServerPool()
	}
	return sharedServerPool
}

func createE2ETestHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "ok", "message": "Test server is running"}`)
	})

	mux.HandleFunc("/.env", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("DB_PASSWORD=secret123\nAPI_KEY=abc123"))
	})

	mux.HandleFunc("/config.php", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<?php $db_pass = 'secret'; ?>"))
	})

	mux.HandleFunc("/backup.sql", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("CREATE TABLE users (id INT, password VARCHAR(255));"))
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("User-agent: *\nDisallow: /admin"))
	})

	mux.HandleFunc("/config.php.bak", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("database_password=secret123"))
	})

	mux.HandleFunc("/index.html.old", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><configuration><connectionStrings><add name="DefaultConnection" connectionString="Server=localhost;Database=prod;User=admin;Password=admin123;" /></connectionStrings></configuration>`))
	})

	mux.HandleFunc("/database.sql.backup", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("CREATE TABLE users (id INT PRIMARY KEY, username VARCHAR(50), password VARCHAR(255)); INSERT INTO users VALUES (1, 'admin', 'admin123');"))
	})

	mux.HandleFunc("/config.bak", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("database_password=secret123"))
	})

	mux.HandleFunc("/app.config.old", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><configuration><connectionStrings><add name="DefaultConnection" connectionString="Server=localhost;Database=prod;User=admin;Password=admin123;" /></connectionStrings></configuration>`))
	})

	mux.HandleFunc("/uploads/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		htmlContent := `<html><head><title>Index of /uploads</title></head><body>` +
			`<h1>Index of /uploads</h1><pre><a href="../">Parent Directory</a></pre></body></html>`
		w.Write([]byte(htmlContent))
	})

	isValidCredential := func(username, password string) bool {
		validCreds := map[string]string{
			"admin":         "admin",
			"root":          "root",
			"administrator": "administrator",
			"guest":         "guest",
			"test":          "test",
			"user":          "user",
		}

		if validPass, exists := validCreds[username]; exists && validPass == password {
			return true
		}

		if username == "admin" && (password == "password" || password == "") {
			return true
		}

		return false
	}

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			username := r.FormValue("username")
			password := r.FormValue("password")
			if isValidCredential(username, password) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<h1>Welcome to Dashboard</h1><a href="/logout">Logout</a>`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<h1>Login Failed</h1><p>Invalid credentials</p>`))
			}
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<form method="post"><input name="username"><input type="password" name="password"></form>`))
		}
	})

	mux.HandleFunc("/admin/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			username := r.FormValue("username")
			password := r.FormValue("password")
			if isValidCredential(username, password) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<h1>Admin Panel</h1><p>Login successful</p>`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<h1>Login Failed</h1><p>Invalid credentials</p>`))
			}
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<form method="post"><input name="user"><input type="password" name="pass"></form>`))
		}
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			user := r.FormValue("user")
			pass := r.FormValue("pass")
			if isValidCredential(user, pass) {
				w.Header().Set("Location", "/dashboard")
				w.WriteHeader(http.StatusFound)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<p>Login failed</p>`))
			}
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<form action="/login" method="post"><input name="user"><input type="password" name="pass"></form>`))
		}
	})

	mux.HandleFunc("/methods-test", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("PUT method is enabled"))
		case "DELETE":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DELETE method is enabled"))
		case "TRACE":
			w.Header().Set("Content-Type", "message/http")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("TRACE /methods-test HTTP/1.1\r\n"))
		case "OPTIONS":
			w.Header().Set("Allow", "GET, POST, PUT, DELETE, TRACE, OPTIONS")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Methods allowed"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})

	mux.HandleFunc("/insecure-headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Apache/2.4.41")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Insecure Page</h1></body></html>"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Apache/2.4.41")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}

func TestMisconfigScanner_E2E_TestServer(t *testing.T) {
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-test-server",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	req, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}
	req.Body.Close()

	testCases := []struct {
		name             string
		tests            []string
		expectedCount    int
		expectedFindings []string
	}{
		{
			name:          "Sensitive files detection",
			tests:         []string{"files"},
			expectedCount: 5,
			expectedFindings: []string{
				"Sensitive file exposed: Environment configuration file",
				"Sensitive file exposed: PHP configuration file",
				"Backup file exposed",
			},
		},
		{
			name:          "Security headers analysis",
			tests:         []string{"headers"},
			expectedCount: 3,
			expectedFindings: []string{
				"Missing security header: X-Frame-Options",
				"Missing security header: X-Content-Type-Options",
				"Missing security header: Strict-Transport-Security",
			},
		},
		{
			name:          "Default credentials testing",
			tests:         []string{"defaults"},
			expectedCount: 8,
			expectedFindings: []string{
				"Default credentials accepted",
			},
		},
		{
			name:          "Server configuration testing",
			tests:         []string{"server"},
			expectedCount: 4,
			expectedFindings: []string{
				"Dangerous HTTP method enabled: PUT",
				"Dangerous HTTP method enabled: DELETE",
				"Dangerous HTTP method enabled: TRACE",
			},
		},
		{
			name:          "Full comprehensive scan",
			tests:         []string{"files", "headers", "defaults", "server"},
			expectedCount: 15,
			expectedFindings: []string{
				"Sensitive file exposed",
				"Missing security header",
				"Default credentials accepted",
				"Dangerous HTTP method enabled",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := scanner.MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: timeouts.HTTPRequest,
				Threads: 5,
				Tests:   tc.tests,
			}

			misconfigScanner := scanner.NewMisconfigScanner(config)
			results := misconfigScanner.Scan()

			if len(results) < tc.expectedCount {
				t.Errorf("Expected at least %d results, got %d", tc.expectedCount, len(results))
			}

			for _, expectedFinding := range tc.expectedFindings {
				found := false
				for _, result := range results {
					if strings.Contains(result.Finding, expectedFinding) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected finding containing '%s' not found", expectedFinding)
				}
			}

			for i, result := range results {
				if result.URL == "" {
					t.Errorf("Result %d missing URL", i)
				}
				if result.Category == "" {
					t.Errorf("Result %d missing Category", i)
				}
				if result.Finding == "" {
					t.Errorf("Result %d missing Finding", i)
				}
				if result.RiskLevel == "" {
					t.Errorf("Result %d missing RiskLevel", i)
				}
				if result.Remediation == "" {
					t.Errorf("Result %d missing Remediation", i)
				}
			}

			errors := misconfigScanner.GetErrors()
			if len(errors) > 0 {
				t.Logf("Scanner errors (may be expected): %v", errors)
			}
		})
	}
}

func TestMisconfigScanner_E2E_SensitiveFilesDetailed(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-sensitive-files",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 3,
		Tests:   []string{"files"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	if _, err := misconfigScanner.GetClient().Get(server.URL + "/health"); err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestSensitiveFiles()

	expectedFiles := map[string]struct {
		riskLevel   string
		description string
	}{
		"/.env": {
			riskLevel:   riskHigh,
			description: "Environment configuration file",
		},
		"/config.php": {
			riskLevel:   riskHigh,
			description: "PHP configuration file",
		},
		"/backup.sql": {
			riskLevel:   riskHigh,
			description: "Database backup file",
		},
		"/robots.txt": {
			riskLevel:   "Low",
			description: "Robots exclusion file",
		},
	}

	foundFiles := make(map[string]bool)
	for _, result := range results {
		for expectedPath, expected := range expectedFiles {
			if strings.Contains(result.URL, expectedPath) {
				foundFiles[expectedPath] = true

				if result.RiskLevel != expected.riskLevel {
					t.Errorf("File %s: expected risk level %s, got %s",
						expectedPath, expected.riskLevel, result.RiskLevel)
				}

				if !strings.Contains(result.Finding, expected.description) {
					t.Errorf("File %s: expected finding to contain '%s', got '%s'",
						expectedPath, expected.description, result.Finding)
				}

				if result.Category != categorySensitiveFiles {
					t.Errorf("File %s: expected category '%s', got '%s'",
						expectedPath, categorySensitiveFiles, result.Category)
				}

				if result.Evidence == "" {
					t.Errorf("File %s: missing evidence", expectedPath)
				}
			}
		}
	}

	for expectedPath := range expectedFiles {
		if !foundFiles[expectedPath] {
			t.Errorf("Expected to find sensitive file %s", expectedPath)
		}
	}
}

func TestMisconfigScanner_E2E_SecurityHeaders(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-security-headers",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:     server.URL + "/insecure-headers",
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 1,
		Tests:   []string{"headers"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	if _, err := misconfigScanner.GetClient().Get(server.URL + "/health"); err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestSecurityHeaders()

	expectedMissingHeaders := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Strict-Transport-Security",
	}

	foundHeaders := make(map[string]bool)
	for _, result := range results {
		for _, expectedHeader := range expectedMissingHeaders {
			if strings.Contains(result.Finding, expectedHeader) {
				foundHeaders[expectedHeader] = true

				if result.Category != "headers" {
					t.Errorf("Header %s: expected category 'headers', got '%s'",
						expectedHeader, result.Category)
				}

				if !strings.Contains(result.Finding, "Missing security header") {
					t.Errorf("Header %s: expected finding to contain 'Missing security header', got '%s'",
						expectedHeader, result.Finding)
				}
			}
		}
	}

	for _, expectedHeader := range expectedMissingHeaders {
		if !foundHeaders[expectedHeader] {
			t.Errorf("Expected to find missing header %s", expectedHeader)
		}
	}
}

func TestMisconfigScanner_E2E_DefaultCredentials(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-default-credentials",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 3,
		Tests:   []string{"defaults"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	if _, err := misconfigScanner.GetClient().Get(server.URL + "/health"); err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestDefaultCredentials()

	if len(results) == 0 {
		t.Error("Expected to find default credentials vulnerabilities")
		return
	}

	foundDefaultCreds := false
	for _, result := range results {
		if strings.Contains(result.Finding, "Default credentials accepted") {
			foundDefaultCreds = true

			if result.Category != "defaults" {
				t.Errorf("Expected category 'defaults', got '%s'", result.Category)
			}

			if result.RiskLevel != riskHigh {
				t.Errorf("Expected risk level '%s', got '%s'", riskHigh, result.RiskLevel)
			}

			validCredentials := []string{"admin", "root", "administrator", "guest", "test", "user"}
			foundValidCred := false
			for _, cred := range validCredentials {
				if strings.Contains(result.Evidence, cred) {
					foundValidCred = true
					break
				}
			}
			if !foundValidCred {
				t.Errorf("Expected evidence to contain a valid credential, got '%s'", result.Evidence)
			}
		}
	}

	if !foundDefaultCreds {
		t.Error("Expected to find default credentials acceptance")
	}
}

func TestMisconfigScanner_E2E_DangerousHTTPMethods(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-dangerous-methods",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 2,
		Tests:   []string{"server"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	if _, err := misconfigScanner.GetClient().Get(server.URL + "/health"); err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestHTTPMethods()

	expectedMethods := []string{"PUT", "DELETE", "TRACE"}
	foundMethods := make(map[string]bool)

	for _, result := range results {
		for _, expectedMethod := range expectedMethods {
			if strings.Contains(result.Finding, expectedMethod) {
				foundMethods[expectedMethod] = true

				if result.Category != "server-config" {
					t.Errorf("Method %s: expected category 'server-config', got '%s'",
						expectedMethod, result.Category)
				}

				if result.RiskLevel != riskHigh {
					t.Errorf("Method %s: expected risk level '%s', got '%s'",
						expectedMethod, riskHigh, result.RiskLevel)
				}
			}
		}
	}

	for _, expectedMethod := range expectedMethods {
		if !foundMethods[expectedMethod] {
			t.Errorf("Expected to find dangerous method %s", expectedMethod)
		}
	}
}

func TestMisconfigScanner_E2E_DirectoryListing(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-directory-listing",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	if _, err := misconfigScanner.GetClient().Get(server.URL + "/health"); err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}

	result := misconfigScanner.TestDirectoryListing("/uploads/")

	if result == nil {
		t.Error("Expected to find directory listing vulnerability")
		return
	}

	if result.Category != categorySensitiveFiles {
		t.Errorf("Expected category '%s', got '%s'", categorySensitiveFiles, result.Category)
	}

	if result.Finding != "Directory listing enabled" {
		t.Errorf("Expected finding 'Directory listing enabled', got '%s'", result.Finding)
	}

	if result.RiskLevel != "Medium" {
		t.Errorf("Expected risk level 'Medium', got '%s'", result.RiskLevel)
	}

	if !strings.Contains(result.Evidence, "/uploads") {
		t.Errorf("Expected evidence to contain '/uploads', got '%s'", result.Evidence)
	}
}

func TestMisconfigScanner_E2E_BackupFiles(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getE2EServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createE2ETestHandler(),
		ConfigName: "e2e-backup-files",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 3,
		Tests:   []string{"files"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	if _, err := misconfigScanner.GetClient().Get(server.URL + "/health"); err != nil {
		t.Skip("Test server not available, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestBackupFiles()

	expectedBackupFiles := []string{
		"config.php.bak",
		"index.html.old",
		"database.sql.backup",
	}

	foundFiles := make(map[string]bool)
	for _, result := range results {
		for _, expectedFile := range expectedBackupFiles {
			if strings.Contains(result.URL, expectedFile) {
				foundFiles[expectedFile] = true

				if result.Category != categorySensitiveFiles {
					t.Errorf("Backup file %s: expected category '%s', got '%s'",
						expectedFile, categorySensitiveFiles, result.Category)
				}

				if result.Finding != "Backup file exposed" {
					t.Errorf("Backup file %s: expected finding 'Backup file exposed', got '%s'",
						expectedFile, result.Finding)
				}

				if result.RiskLevel != riskHigh {
					t.Errorf("Backup file %s: expected risk level '%s', got '%s'",
						expectedFile, riskHigh, result.RiskLevel)
				}
			}
		}
	}

	for _, expectedFile := range expectedBackupFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected to find backup file %s", expectedFile)
		}
	}
}
