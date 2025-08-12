package scanner

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMisconfigScanner_TestSensitiveFiles(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse map[string]string
		expectedCount  int
		expectedRisk   string
	}{
		{
			name: "detects .env file",
			serverResponse: map[string]string{
				"/.env": "DB_PASSWORD=secret123\nAPI_KEY=abc123",
			},
			expectedCount: 1,
			expectedRisk:  "High",
		},
		{
			name: "detects multiple sensitive files",
			serverResponse: map[string]string{
				"/.env":       "DB_PASSWORD=secret",
				"/config.php": "<?php $db_pass = 'secret'; ?>",
				"/robots.txt": "User-agent: *\nDisallow: /admin",
			},
			expectedCount: 3,
		},
		{
			name:           "no sensitive files found",
			serverResponse: map[string]string{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if content, exists := tt.serverResponse[r.URL.Path]; exists {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(content))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"files"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestSensitiveFiles()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && tt.expectedRisk != "" {
				found := false
				for _, result := range results {
					if result.RiskLevel == tt.expectedRisk {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected risk level %s not found", tt.expectedRisk)
				}
			}
		})
	}
}

func TestMisconfigScanner_TestDirectoryListing(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		statusCode     int
		expectedResult bool
	}{
		{
			name:           "detects Apache directory listing",
			responseBody:   `<html><head><title>Index of /uploads</title></head><body><h1>Index of /uploads</h1><pre><a href="../">Parent Directory</a></pre></body></html>`,
			statusCode:     http.StatusOK,
			expectedResult: true,
		},
		{
			name:           "detects Nginx directory listing",
			responseBody:   `<html><head><title>Index of /files/</title></head><body><h1>Index of /files/</h1><hr><pre><a href="../">../</a></pre></body></html>`,
			statusCode:     http.StatusOK,
			expectedResult: true,
		},
		{
			name:           "detects IIS directory listing",
			responseBody:   `<html><head><title>Directory Listing For /</title></head><body><h1>Directory Listing For /</h1></body></html>`,
			statusCode:     http.StatusOK,
			expectedResult: true,
		},
		{
			name:           "normal page content",
			responseBody:   `<html><head><title>Welcome</title></head><body><h1>Welcome to our site</h1></body></html>`,
			statusCode:     http.StatusOK,
			expectedResult: false,
		},
		{
			name:           "404 not found",
			responseBody:   `<html><head><title>404 Not Found</title></head><body><h1>Not Found</h1></body></html>`,
			statusCode:     http.StatusNotFound,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"files"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			result := misconfigScanner.TestDirectoryListing("/uploads")

			if tt.expectedResult {
				if result == nil {
					t.Error("Expected result but got nil")
				} else {
					if result.Category != "sensitive-files" {
						t.Errorf("Expected category 'sensitive-files', got '%s'", result.Category)
					}
					if result.Finding != "Directory listing enabled" {
						t.Errorf("Expected finding 'Directory listing enabled', got '%s'", result.Finding)
					}
					if result.RiskLevel != "Medium" {
						t.Errorf("Expected risk level 'Medium', got '%s'", result.RiskLevel)
					}
				}
			} else {
				if result != nil {
					t.Error("Expected nil result but got a result")
				}
			}
		})
	}
}

func TestMisconfigScanner_TestBackupFiles(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse map[string]string
		expectedCount  int
	}{
		{
			name: "detects backup files",
			serverResponse: map[string]string{
				"/index.php.bak": "<?php echo 'backup content'; ?>",
				"/config.php~":   "<?php $secret = 'password'; ?>",
			},
			expectedCount: 2,
		},
		{
			name: "detects database backup",
			serverResponse: map[string]string{
				"/backup.sql.old": "CREATE TABLE users (id INT, password VARCHAR(255));",
			},
			expectedCount: 1,
		},
		{
			name:           "no backup files found",
			serverResponse: map[string]string{},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if content, exists := tt.serverResponse[r.URL.Path]; exists {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(content))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"files"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestBackupFiles()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			for _, result := range results {
				if result.Category != "sensitive-files" {
					t.Errorf("Expected category 'sensitive-files', got '%s'", result.Category)
				}
				if result.Finding != "Backup file exposed" {
					t.Errorf("Expected finding 'Backup file exposed', got '%s'", result.Finding)
				}
				if result.RiskLevel != "High" {
					t.Errorf("Expected risk level 'High', got '%s'", result.RiskLevel)
				}
			}
		})
	}
}

func TestMisconfigScanner_DetectDirectoryListing(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "Apache style directory listing",
			body:     `<html><head><title>Index of /uploads</title></head>`,
			expected: true,
		},
		{
			name:     "Nginx style directory listing",
			body:     `<h1>Index of /files/</h1>`,
			expected: true,
		},
		{
			name:     "IIS style directory listing",
			body:     `Directory listing for /admin/`,
			expected: true,
		},
		{
			name:     "Parent directory link",
			body:     `<pre><a href="../">Parent Directory</a>`,
			expected: true,
		},
		{
			name:     "Normal web page",
			body:     `<html><head><title>Welcome</title></head><body>Hello World</body></html>`,
			expected: false,
		},
		{
			name:     "Empty response",
			body:     ``,
			expected: false,
		},
	}

	config := MisconfigConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := misconfigScanner.DetectDirectoryListing(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for body: %s", tt.expected, result, tt.body)
			}
		})
	}
}

func TestMisconfigScanner_TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name             string
		responseHeaders  map[string]string
		expectedCount    int
		expectedFindings []string
	}{
		{
			name:            "missing all required headers",
			responseHeaders: map[string]string{},
			expectedCount:   3,
			expectedFindings: []string{
				"Missing security header: X-Frame-Options",
				"Missing security header: X-Content-Type-Options",
				"Missing security header: Strict-Transport-Security",
			},
		},
		{
			name: "all headers present with valid values",
			responseHeaders: map[string]string{
				"X-Frame-Options":           "DENY",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000",
				"Content-Security-Policy":   "default-src 'self'",
				"X-XSS-Protection":          "1; mode=block",
			},
			expectedCount: 0,
		},
		{
			name: "weak header values",
			responseHeaders: map[string]string{
				"X-Frame-Options":        "ALLOWALL",
				"X-Content-Type-Options": "sniff",
				"X-XSS-Protection":       "0",
			},
			expectedCount: 4,
			expectedFindings: []string{
				"Weak security header value: X-Frame-Options",
				"Weak security header value: X-Content-Type-Options",
				"Weak security header value: X-XSS-Protection",
				"Missing security header: Strict-Transport-Security",
			},
		},
		{
			name: "mixed valid and invalid headers",
			responseHeaders: map[string]string{
				"X-Frame-Options":        "SAMEORIGIN",
				"X-Content-Type-Options": "invalid",
			},
			expectedCount: 2,
			expectedFindings: []string{
				"Weak security header value: X-Content-Type-Options",
				"Missing security header: Strict-Transport-Security",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for name, value := range tt.responseHeaders {
					w.Header().Set(name, value)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"headers"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestSecurityHeaders()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			for _, expectedFinding := range tt.expectedFindings {
				found := false
				for _, result := range results {
					if result.Finding == expectedFinding {
						found = true
						if result.Category != "headers" {
							t.Errorf("Expected category 'headers', got '%s'", result.Category)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected finding '%s' not found", expectedFinding)
				}
			}
		})
	}
}

func TestMisconfigScanner_TestHTTPSEnforcement(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		hstsHeader      string
		expectedResult  bool
		expectedFinding string
		expectedRisk    string
	}{
		{
			name:            "HTTPS site missing HSTS header",
			url:             "https://example.com",
			hstsHeader:      "",
			expectedResult:  true,
			expectedFinding: "HTTPS not properly enforced",
			expectedRisk:    "High",
		},
		{
			name:           "HTTPS site with valid HSTS header",
			url:            "https://example.com",
			hstsHeader:     "max-age=31536000; includeSubDomains",
			expectedResult: false,
		},
		{
			name:            "HTTPS site with HSTS missing max-age",
			url:             "https://example.com",
			hstsHeader:      "includeSubDomains",
			expectedResult:  true,
			expectedFinding: "Weak HSTS configuration",
			expectedRisk:    "Medium",
		},
		{
			name:           "HTTP site (HSTS not applicable)",
			url:            "http://example.com",
			hstsHeader:     "",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.hstsHeader != "" {
					w.Header().Set("Strict-Transport-Security", tt.hstsHeader)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			// For HTTPS tests, we'll use the actual URL from the test case
			// For HTTP tests, we'll use the server URL
			testURL := tt.url
			if tt.url == "http://example.com" {
				testURL = server.URL
			}

			config := MisconfigConfig{
				URL:     testURL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"headers"},
			}

			misconfigScanner := NewMisconfigScanner(config)

			// For HTTPS tests, we need to mock the HTTP response since we can't make real HTTPS calls to test servers
			if tt.url == "https://example.com" {
				// Create a mock HTTP response
				resp := &http.Response{
					StatusCode: 200,
					Header:     make(http.Header),
				}
				if tt.hstsHeader != "" {
					resp.Header.Set("Strict-Transport-Security", tt.hstsHeader)
				}

				// Test the logic directly using the analyzeHTTPSEnforcement helper
				result := misconfigScanner.AnalyzeHTTPSEnforcement(resp, tt.url)

				if tt.expectedResult {
					if result == nil {
						t.Error("Expected result but got nil")
					} else {
						if result.Finding != tt.expectedFinding {
							t.Errorf("Expected finding '%s', got '%s'", tt.expectedFinding, result.Finding)
						}
						if result.RiskLevel != tt.expectedRisk {
							t.Errorf("Expected risk level '%s', got '%s'", tt.expectedRisk, result.RiskLevel)
						}
						if result.Category != "headers" {
							t.Errorf("Expected category 'headers', got '%s'", result.Category)
						}
					}
				} else {
					if result != nil {
						t.Error("Expected nil result but got a result")
					}
				}
			} else {
				// For HTTP tests, we can use the actual method
				result := misconfigScanner.TestHTTPSEnforcement()

				if tt.expectedResult {
					if result == nil {
						t.Error("Expected result but got nil")
					} else {
						if result.Finding != tt.expectedFinding {
							t.Errorf("Expected finding '%s', got '%s'", tt.expectedFinding, result.Finding)
						}
						if result.RiskLevel != tt.expectedRisk {
							t.Errorf("Expected risk level '%s', got '%s'", tt.expectedRisk, result.RiskLevel)
						}
						if result.Category != "headers" {
							t.Errorf("Expected category 'headers', got '%s'", result.Category)
						}
					}
				} else {
					if result != nil {
						t.Error("Expected nil result but got a result")
					}
				}
			}
		})
	}
}

func TestMisconfigScanner_ValidateHeaderValue(t *testing.T) {
	tests := []struct {
		name            string
		headerName      string
		headerValue     string
		expectedResult  bool
		expectedFinding string
	}{
		{
			name:           "valid X-Frame-Options DENY",
			headerName:     "X-Frame-Options",
			headerValue:    "DENY",
			expectedResult: false,
		},
		{
			name:           "valid X-Frame-Options SAMEORIGIN",
			headerName:     "X-Frame-Options",
			headerValue:    "SAMEORIGIN",
			expectedResult: false,
		},
		{
			name:            "invalid X-Frame-Options value",
			headerName:      "X-Frame-Options",
			headerValue:     "ALLOWALL",
			expectedResult:  true,
			expectedFinding: "Invalid security header value: X-Frame-Options",
		},
		{
			name:           "valid X-Content-Type-Options",
			headerName:     "X-Content-Type-Options",
			headerValue:    "nosniff",
			expectedResult: false,
		},
		{
			name:            "invalid X-Content-Type-Options value",
			headerName:      "X-Content-Type-Options",
			headerValue:     "sniff",
			expectedResult:  true,
			expectedFinding: "Invalid security header value: X-Content-Type-Options",
		},
		{
			name:           "valid X-XSS-Protection",
			headerName:     "X-XSS-Protection",
			headerValue:    "1; mode=block",
			expectedResult: false,
		},
		{
			name:            "invalid X-XSS-Protection value",
			headerName:      "X-XSS-Protection",
			headerValue:     "0",
			expectedResult:  true,
			expectedFinding: "Invalid security header value: X-XSS-Protection",
		},
		{
			name:           "header without validation rules",
			headerName:     "Strict-Transport-Security",
			headerValue:    "max-age=31536000",
			expectedResult: false,
		},
		{
			name:           "unknown header",
			headerName:     "Unknown-Header",
			headerValue:    "some-value",
			expectedResult: false,
		},
	}

	config := MisconfigConfig{
		URL:     "https://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{"headers"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := misconfigScanner.ValidateHeaderValue(tt.headerName, tt.headerValue)

			if tt.expectedResult {
				if result == nil {
					t.Error("Expected result but got nil")
				} else {
					if result.Finding != tt.expectedFinding {
						t.Errorf("Expected finding '%s', got '%s'", tt.expectedFinding, result.Finding)
					}
					if result.Category != "headers" {
						t.Errorf("Expected category 'headers', got '%s'", result.Category)
					}
					if result.RiskLevel != "Medium" {
						t.Errorf("Expected risk level 'Medium', got '%s'", result.RiskLevel)
					}
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				}
			}
		})
	}
}

func TestMisconfigScanner_DetectLoginForm(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "login form with action",
			body:     `<form action="/login" method="post"><input type="text" name="username"><input type="password" name="password"></form>`,
			expected: true,
		},
		{
			name:     "form with password field",
			body:     `<form method="post"><input type="text" name="user"><input type="password" name="pass"></form>`,
			expected: true,
		},
		{
			name:     "form with username field",
			body:     `<form><input type="text" name="username"><input type="submit"></form>`,
			expected: true,
		},
		{
			name:     "form with user field",
			body:     `<form><input type="text" name="user"><input type="submit"></form>`,
			expected: true,
		},
		{
			name:     "form with email and password",
			body:     `<form><input type="email" name="email"><input type="password" name="password"></form>`,
			expected: true,
		},
		{
			name:     "regular form without login fields",
			body:     `<form><input type="text" name="search"><input type="submit"></form>`,
			expected: false,
		},
		{
			name:     "no form present",
			body:     `<html><body><h1>Welcome</h1><p>No forms here</p></body></html>`,
			expected: false,
		},
		{
			name:     "empty body",
			body:     ``,
			expected: false,
		},
	}

	config := MisconfigConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{"defaults"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := misconfigScanner.DetectLoginForm(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for body: %s", tt.expected, result, tt.body)
			}
		})
	}
}

func TestMisconfigScanner_TestDefaultCredentials(t *testing.T) {
	tests := []struct {
		name            string
		loginPaths      []string
		responses       map[string]map[string]string // path -> method -> response
		statusCodes     map[string]map[string]int    // path -> method -> status code
		expectedCount   int
		expectedFinding string
	}{
		{
			name:       "successful login with default credentials",
			loginPaths: []string{"/login"},
			responses: map[string]map[string]string{
				"/login": {
					"GET":  `<form method="post"><input name="username"><input type="password" name="password"></form>`,
					"POST": `<html><body><h1>Welcome to Dashboard</h1><a href="/logout">Logout</a></body></html>`,
				},
			},
			statusCodes: map[string]map[string]int{
				"/login": {
					"GET":  200,
					"POST": 200,
				},
			},
			expectedCount:   8, // Number of default credentials in CommonDefaultCredentials
			expectedFinding: "Default credentials accepted",
		},
		{
			name:       "failed login attempts",
			loginPaths: []string{"/login"},
			responses: map[string]map[string]string{
				"/login": {
					"GET":  `<form method="post"><input name="username"><input type="password" name="password"></form>`,
					"POST": `<html><body><h1>Login Failed</h1><p>Invalid credentials</p></body></html>`,
				},
			},
			statusCodes: map[string]map[string]int{
				"/login": {
					"GET":  200,
					"POST": 200,
				},
			},
			expectedCount: 0,
		},
		{
			name:       "redirect on successful login",
			loginPaths: []string{"/admin"},
			responses: map[string]map[string]string{
				"/admin": {
					"GET":  `<form action="/admin" method="post"><input name="username"><input type="password" name="password"></form>`,
					"POST": `Redirecting...`,
				},
				"/": {
					"GET": `<html><body><h1>Home</h1></body></html>`,
				},
			},
			statusCodes: map[string]map[string]int{
				"/admin": {
					"GET":  200,
					"POST": 302,
				},
				"/": {
					"GET": 200,
				},
			},
			expectedCount:   8,
			expectedFinding: "Default credentials accepted",
		},
		{
			name:       "no login forms found",
			loginPaths: []string{},
			responses: map[string]map[string]string{
				"/": {
					"GET": `<html><body><h1>Welcome</h1></body></html>`,
				},
			},
			statusCodes: map[string]map[string]int{
				"/": {
					"GET": 200,
				},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if pathResponses, exists := tt.responses[r.URL.Path]; exists {
					if response, methodExists := pathResponses[r.Method]; methodExists {
						if statusCodes, statusExists := tt.statusCodes[r.URL.Path]; statusExists {
							if statusCode, codeExists := statusCodes[r.Method]; codeExists {
								if statusCode == 302 {
									w.Header().Set("Location", "/dashboard")
								}
								w.WriteHeader(statusCode)
								w.Write([]byte(response))
								return
							}
						}
						w.WriteHeader(200)
						w.Write([]byte(response))
						return
					}
				}
				w.WriteHeader(404)
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"defaults"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestDefaultCredentials()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && tt.expectedFinding != "" {
				found := false
				for _, result := range results {
					if result.Finding == tt.expectedFinding {
						found = true
						if result.Category != "defaults" {
							t.Errorf("Expected category 'defaults', got '%s'", result.Category)
						}
						if result.RiskLevel != "High" {
							t.Errorf("Expected risk level 'High', got '%s'", result.RiskLevel)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected finding '%s' not found", tt.expectedFinding)
				}
			}
		})
	}
}

func TestMisconfigScanner_TestDefaultPages(t *testing.T) {
	tests := []struct {
		name            string
		responses       map[string]string
		expectedCount   int
		expectedFinding string
	}{
		{
			name: "default installation page detected",
			responses: map[string]string{
				"/": `<html><head><title>Welcome to Apache Installation</title></head><body><h1>Congratulations! You have successfully installed Apache</h1></body></html>`,
			},
			expectedCount:   1,
			expectedFinding: "Default installation page detected",
		},
		{
			name: "version disclosure detected",
			responses: map[string]string{
				"/info.php": `<html><body><h1>PHP Version 7.4.3</h1><p>Apache/2.4.41 Server</p></body></html>`,
			},
			expectedCount:   1,
			expectedFinding: "Version information disclosed",
		},
		{
			name: "multiple default pages",
			responses: map[string]string{
				"/":         `<html><head><title>Welcome to Nginx Installation</title></head></html>`,
				"/info.php": `<html><body>MySQL 8.0.25</body></html>`,
			},
			expectedCount: 2,
		},
		{
			name: "no default pages found",
			responses: map[string]string{
				"/": `<html><body><h1>Custom Application</h1></body></html>`,
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if response, exists := tt.responses[r.URL.Path]; exists {
					w.WriteHeader(200)
					w.Write([]byte(response))
				} else {
					w.WriteHeader(404)
				}
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"defaults"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestDefaultPages()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && tt.expectedFinding != "" {
				found := false
				for _, result := range results {
					if result.Finding == tt.expectedFinding {
						found = true
						if result.Category != "defaults" {
							t.Errorf("Expected category 'defaults', got '%s'", result.Category)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected finding '%s' not found", tt.expectedFinding)
				}
			}
		})
	}
}

func TestMisconfigScanner_DetectVersionDisclosure(t *testing.T) {
	tests := []struct {
		name             string
		response         string
		expectedResult   bool
		expectedSoftware string
		expectedVersion  string
	}{
		{
			name:             "Apache version disclosure",
			response:         "Server: Apache/2.4.41 (Ubuntu)",
			expectedResult:   true,
			expectedSoftware: "Apache",
			expectedVersion:  "2.4.41",
		},
		{
			name:             "Nginx version disclosure",
			response:         "Server: nginx/1.18.0",
			expectedResult:   true,
			expectedSoftware: "nginx",
			expectedVersion:  "1.18.0",
		},
		{
			name:             "PHP version disclosure",
			response:         "X-Powered-By: PHP/7.4.3",
			expectedResult:   true,
			expectedSoftware: "PHP",
			expectedVersion:  "7.4.3",
		},
		{
			name:             "MySQL version disclosure",
			response:         "MySQL Server version: 8.0.25",
			expectedResult:   true,
			expectedSoftware: "MySQL",
			expectedVersion:  "8.0.25",
		},
		{
			name:             "WordPress version disclosure",
			response:         "WordPress 5.8.1 installation",
			expectedResult:   true,
			expectedSoftware: "WordPress",
			expectedVersion:  "5.8.1",
		},
		{
			name:           "no version disclosure",
			response:       "Server: CustomServer",
			expectedResult: false,
		},
		{
			name:           "empty response",
			response:       "",
			expectedResult: false,
		},
	}

	config := MisconfigConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{"defaults"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := misconfigScanner.DetectVersionDisclosure(tt.response)

			if tt.expectedResult {
				if result == nil {
					t.Error("Expected result but got nil")
				} else {
					if result.Finding != "Version information disclosed" {
						t.Errorf("Expected finding 'Version information disclosed', got '%s'", result.Finding)
					}
					if result.Category != "defaults" {
						t.Errorf("Expected category 'defaults', got '%s'", result.Category)
					}
					if result.RiskLevel != "Low" {
						t.Errorf("Expected risk level 'Low', got '%s'", result.RiskLevel)
					}
					if !strings.Contains(result.Evidence, tt.expectedVersion) {
						t.Errorf("Expected evidence to contain version '%s', got '%s'", tt.expectedVersion, result.Evidence)
					}
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				}
			}
		})
	}
}

func TestMisconfigScanner_DetectDefaultInstallation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "Apache test page",
			body:     `<html><head><title>Apache2 Ubuntu Default Page</title></head><body><h1>Apache2 Test Page</h1></body></html>`,
			expected: true,
		},
		{
			name:     "Nginx welcome page",
			body:     `<html><head><title>Welcome to nginx!</title></head><body><h1>Welcome to nginx!</h1></body></html>`,
			expected: true,
		},
		{
			name:     "IIS welcome page",
			body:     `<html><head><title>IIS Windows Server</title></head><body><h1>IIS Welcome</h1></body></html>`,
			expected: true,
		},
		{
			name:     "XAMPP dashboard",
			body:     `<html><head><title>XAMPP Dashboard</title></head><body><h1>Welcome to XAMPP</h1></body></html>`,
			expected: true,
		},
		{
			name:     "WAMP server page",
			body:     `<html><head><title>WAMP Server Homepage</title></head><body><h1>WAMP Server</h1></body></html>`,
			expected: true,
		},
		{
			name:     "successful installation message",
			body:     `<html><body><h1>Congratulations! You have successfully installed the software</h1></body></html>`,
			expected: true,
		},
		{
			name:     "welcome installation message",
			body:     `<html><body><h1>Welcome to the installation wizard</h1></body></html>`,
			expected: true,
		},
		{
			name:     "custom application page",
			body:     `<html><head><title>My Custom App</title></head><body><h1>Welcome to My App</h1></body></html>`,
			expected: false,
		},
		{
			name:     "empty body",
			body:     ``,
			expected: false,
		},
	}

	config := MisconfigConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{"defaults"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := misconfigScanner.DetectDefaultInstallation(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for body: %s", tt.expected, result, tt.body)
			}
		})
	}
}

func TestMisconfigScanner_TestHTTPMethods(t *testing.T) {
	tests := []struct {
		name            string
		allowedMethods  []string
		expectedCount   int
		expectedFinding string
		expectedRisk    string
	}{
		{
			name:            "dangerous methods enabled",
			allowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "TRACE"},
			expectedCount:   3, // PUT, DELETE, TRACE are dangerous
			expectedFinding: "Dangerous HTTP method enabled",
			expectedRisk:    "High",
		},
		{
			name:           "only safe methods enabled",
			allowedMethods: []string{"GET", "POST", "HEAD"},
			expectedCount:  0,
		},
		{
			name:            "OPTIONS method enabled",
			allowedMethods:  []string{"GET", "POST", "OPTIONS"},
			expectedCount:   1,
			expectedFinding: "Dangerous HTTP method enabled: OPTIONS",
			expectedRisk:    "Medium",
		},
		{
			name:            "PATCH method enabled",
			allowedMethods:  []string{"GET", "POST", "PATCH"},
			expectedCount:   1,
			expectedFinding: "Dangerous HTTP method enabled: PATCH",
			expectedRisk:    "Medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methodAllowed := false
				for _, method := range tt.allowedMethods {
					if r.Method == method {
						methodAllowed = true
						break
					}
				}

				if methodAllowed {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				} else {
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"server-config"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestHTTPMethods()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && tt.expectedFinding != "" {
				found := false
				for _, result := range results {
					if strings.Contains(result.Finding, tt.expectedFinding) {
						found = true
						if result.Category != "server-config" {
							t.Errorf("Expected category 'server-config', got '%s'", result.Category)
						}
						if result.RiskLevel != tt.expectedRisk {
							t.Errorf("Expected risk level '%s', got '%s'", tt.expectedRisk, result.RiskLevel)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected finding containing '%s' not found", tt.expectedFinding)
				}
			}
		})
	}
}

func TestMisconfigScanner_TestServerBanner(t *testing.T) {
	tests := []struct {
		name            string
		serverHeader    string
		expectedResult  bool
		expectedFinding string
		expectedRisk    string
	}{
		{
			name:            "Apache version disclosure",
			serverHeader:    "Apache/2.4.41 (Ubuntu)",
			expectedResult:  true,
			expectedFinding: "Server version disclosed in banner",
			expectedRisk:    "Low",
		},
		{
			name:            "Nginx version disclosure",
			serverHeader:    "nginx/1.18.0",
			expectedResult:  true,
			expectedFinding: "Server version disclosed in banner",
			expectedRisk:    "Low",
		},
		{
			name:            "IIS version disclosure",
			serverHeader:    "Microsoft-IIS/10.0",
			expectedResult:  true,
			expectedFinding: "Server version disclosed in banner",
			expectedRisk:    "Low",
		},
		{
			name:            "Apache without version",
			serverHeader:    "Apache",
			expectedResult:  true,
			expectedFinding: "Server software disclosed in banner",
			expectedRisk:    "Low",
		},
		{
			name:            "Nginx without version",
			serverHeader:    "nginx",
			expectedResult:  true,
			expectedFinding: "Server software disclosed in banner",
			expectedRisk:    "Low",
		},
		{
			name:           "generic server header",
			serverHeader:   "CustomServer/1.0",
			expectedResult: false,
		},
		{
			name:           "no server header",
			serverHeader:   "",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverHeader != "" {
					w.Header().Set("Server", tt.serverHeader)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"server-config"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			result := misconfigScanner.TestServerBanner()

			if tt.expectedResult {
				if result == nil {
					t.Error("Expected result but got nil")
				} else {
					if result.Finding != tt.expectedFinding {
						t.Errorf("Expected finding '%s', got '%s'", tt.expectedFinding, result.Finding)
					}
					if result.RiskLevel != tt.expectedRisk {
						t.Errorf("Expected risk level '%s', got '%s'", tt.expectedRisk, result.RiskLevel)
					}
					if result.Category != "server-config" {
						t.Errorf("Expected category 'server-config', got '%s'", result.Category)
					}
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				}
			}
		})
	}
}

func TestMisconfigScanner_TestErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		errorResponses  map[string]string
		expectedCount   int
		expectedFinding string
	}{
		{
			name: "stack trace in error",
			errorResponses: map[string]string{
				"/nonexistent-page-12345": `<html><body><h1>Internal Server Error</h1><pre>Stack trace:
at com.example.Controller.handleRequest(Controller.java:42)
at com.example.Servlet.doGet(Servlet.java:15)</pre></body></html>`,
			},
			expectedCount:   1,
			expectedFinding: "Information leakage in error messages",
		},
		{
			name: "database error disclosure",
			errorResponses: map[string]string{
				"/admin/secret": `<html><body><h1>Database Error</h1><p>MySQL Error: Table 'users' doesn't exist in query SELECT * FROM users WHERE id=1</p></body></html>`,
			},
			expectedCount:   1,
			expectedFinding: "Information leakage in error messages",
		},
		{
			name: "PHP fatal error",
			errorResponses: map[string]string{
				"/database/config": `<html><body><h1>Fatal Error</h1><p>Fatal error: Call to undefined function mysql_connect() in /var/www/html/config.php on line 15</p></body></html>`,
			},
			expectedCount:   1,
			expectedFinding: "Information leakage in error messages",
		},
		{
			name: "version disclosure in error",
			errorResponses: map[string]string{
				"/api/v1/nonexistent": `<html><body><h1>404 Not Found</h1><p>Apache/2.4.41 Server at example.com Port 80</p></body></html>`,
			},
			expectedCount:   1,
			expectedFinding: "Version information disclosed in error page",
		},
		{
			name: "multiple error pages with issues",
			errorResponses: map[string]string{
				"/nonexistent-page-12345": `<html><body><h1>Exception occurred</h1><p>Error on line 42</p></body></html>`,
				"/admin/secret":           `<html><body><h1>Debug information</h1><p>Debug mode enabled</p></body></html>`,
			},
			expectedCount: 2,
		},
		{
			name: "clean error pages",
			errorResponses: map[string]string{
				"/nonexistent-page-12345": `<html><body><h1>404 Not Found</h1><p>The requested page was not found.</p></body></html>`,
				"/admin/secret":           `<html><body><h1>403 Forbidden</h1><p>Access denied.</p></body></html>`,
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if response, exists := tt.errorResponses[r.URL.Path]; exists {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(response))
				} else {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				}
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"server-config"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestErrorMessages()

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && tt.expectedFinding != "" {
				found := false
				for _, result := range results {
					if result.Finding == tt.expectedFinding {
						found = true
						if result.Category != "server-config" {
							t.Errorf("Expected category 'server-config', got '%s'", result.Category)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected finding '%s' not found", tt.expectedFinding)
				}
			}
		})
	}
}

func TestMisconfigScanner_TestInsecureRedirects(t *testing.T) {
	tests := []struct {
		name             string
		redirects        map[string]string // path -> redirect location
		expectedMinCount int
		expectedFinding  string
	}{
		{
			name: "open redirect detected",
			redirects: map[string]string{
				"/redirect?url=http://evil.com":        "http://evil.com",
				"/redirect?redirect=http://evil.com":   "http://evil.com",
				"/redirect?return=http://evil.com":     "http://evil.com",
				"/redirect?next=http://evil.com":       "http://evil.com",
				"/login?return_to=http://evil.com":     "http://evil.com",
				"/logout?redirect_uri=http://evil.com": "http://evil.com",
			},
			expectedMinCount: 1,
			expectedFinding:  "Insecure redirect configuration detected",
		},
		{
			name: "safe redirects only",
			redirects: map[string]string{
				"/redirect?url=http://evil.com":        "/dashboard",
				"/redirect?redirect=http://evil.com":   "/login",
				"/redirect?return=http://evil.com":     "/home",
				"/redirect?next=http://evil.com":       "/profile",
				"/login?return_to=http://evil.com":     "/dashboard",
				"/logout?redirect_uri=http://evil.com": "/login",
			},
			expectedMinCount: 0,
		},
		{
			name: "mixed redirects",
			redirects: map[string]string{
				"/redirect?url=http://evil.com":        "http://evil.com",
				"/redirect?redirect=http://evil.com":   "/login",
				"/redirect?return=http://evil.com":     "/home",
				"/redirect?next=http://evil.com":       "/profile",
				"/login?return_to=http://evil.com":     "/dashboard",
				"/logout?redirect_uri=http://evil.com": "/login",
			},
			expectedMinCount: 1,
			expectedFinding:  "Insecure redirect configuration detected",
		},
		{
			name:             "no redirects",
			redirects:        map[string]string{},
			expectedMinCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fullPath := r.URL.Path
				if r.URL.RawQuery != "" {
					fullPath += "?" + r.URL.RawQuery
				}

				if location, exists := tt.redirects[fullPath]; exists {
					w.Header().Set("Location", location)
					w.WriteHeader(http.StatusFound)
				} else {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				}
			}))
			defer server.Close()

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"server-config"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			results := misconfigScanner.TestInsecureRedirects()

			if len(results) < tt.expectedMinCount {
				t.Errorf("Expected at least %d results, got %d", tt.expectedMinCount, len(results))
			}

			if tt.expectedMinCount > 0 && tt.expectedFinding != "" {
				found := false
				for _, result := range results {
					if result.Finding == tt.expectedFinding {
						found = true
						if result.Category != "server-config" {
							t.Errorf("Expected category 'server-config', got '%s'", result.Category)
						}
						if result.RiskLevel != "Medium" {
							t.Errorf("Expected risk level 'Medium', got '%s'", result.RiskLevel)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected finding '%s' not found", tt.expectedFinding)
				}
			}
		})
	}
}

func TestMisconfigScanner_Scan(t *testing.T) {
	tests := []struct {
		name               string
		config             MisconfigConfig
		serverResponses    map[string]string
		serverHeaders      map[string]string
		expectedCount      int
		expectedCategories []string
	}{
		{
			name: "full scan with all test categories",
			config: MisconfigConfig{
				URL:     "",
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 2,
				Tests:   []string{}, // Empty means run all tests
			},
			serverResponses: map[string]string{
				"/.env":  "DB_PASSWORD=secret123",
				"/login": `<form method="post"><input name="username"><input type="password" name="password"></form>`,
			},
			serverHeaders: map[string]string{
				// Missing security headers will be detected
			},
			expectedCount:      4, // At least: .env file, missing headers (3), login form detected
			expectedCategories: []string{"sensitive-files", "headers"},
		},
		{
			name: "scan with specific test categories",
			config: MisconfigConfig{
				URL:     "",
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 2,
				Tests:   []string{"files", "headers"},
			},
			serverResponses: map[string]string{
				"/.env": "DB_PASSWORD=secret123",
			},
			serverHeaders:      map[string]string{},
			expectedCount:      4, // .env file + 3 missing headers
			expectedCategories: []string{"sensitive-files", "headers"},
		},
		{
			name: "scan with only files test",
			config: MisconfigConfig{
				URL:     "",
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 2,
				Tests:   []string{"files"},
			},
			serverResponses: map[string]string{
				"/.env":       "DB_PASSWORD=secret123",
				"/config.php": "<?php $secret = 'password'; ?>",
			},
			serverHeaders:      map[string]string{},
			expectedCount:      2, // Only file-related findings
			expectedCategories: []string{"sensitive-files"},
		},
		{
			name: "concurrent execution with multiple threads",
			config: MisconfigConfig{
				URL:     "",
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   []string{"files"},
			},
			serverResponses: map[string]string{
				"/.env":       "DB_PASSWORD=secret123",
				"/config.php": "<?php $secret = 'password'; ?>",
				"/robots.txt": "User-agent: *",
			},
			serverHeaders:      map[string]string{},
			expectedCount:      3,
			expectedCategories: []string{"sensitive-files"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set headers
				for name, value := range tt.serverHeaders {
					w.Header().Set(name, value)
				}

				// Handle responses
				if content, exists := tt.serverResponses[r.URL.Path]; exists {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(content))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Update config with server URL
			tt.config.URL = server.URL

			misconfigScanner := NewMisconfigScanner(tt.config)
			results := misconfigScanner.Scan()

			if len(results) < tt.expectedCount {
				t.Errorf("Expected at least %d results, got %d", tt.expectedCount, len(results))
			}

			// Check that expected categories are present
			foundCategories := make(map[string]bool)
			for _, result := range results {
				foundCategories[result.Category] = true
			}

			for _, expectedCategory := range tt.expectedCategories {
				if !foundCategories[expectedCategory] {
					t.Errorf("Expected category '%s' not found in results", expectedCategory)
				}
			}

			// Verify results are properly aggregated
			allResults := misconfigScanner.GetResults()
			if len(allResults) != len(results) {
				t.Errorf("Results not properly aggregated: expected %d, got %d", len(results), len(allResults))
			}
		})
	}
}

func TestMisconfigScanner_ConcurrentExecution(t *testing.T) {
	// Test concurrent execution safety
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		responses := map[string]string{
			"/.env":       "DB_PASSWORD=secret123",
			"/config.php": "<?php $secret = 'password'; ?>",
			"/robots.txt": "User-agent: *",
		}

		if content, exists := responses[r.URL.Path]; exists {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 10, // High concurrency
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	// Run multiple scans concurrently to test thread safety
	var wg sync.WaitGroup
	results := make([][]MisconfigResult, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = misconfigScanner.Scan()
		}(i)
	}

	wg.Wait()

	// Verify all scans produced results
	for i, result := range results {
		if len(result) == 0 {
			t.Errorf("Scan %d produced no results", i)
		}
	}

	// Verify the final aggregated results contain all findings
	finalResults := misconfigScanner.GetResults()
	if len(finalResults) == 0 {
		t.Error("Final aggregated results are empty")
	}
}

func TestMisconfigScanner_ShouldRunTest(t *testing.T) {
	tests := []struct {
		name         string
		configTests  []string
		testCategory string
		expected     bool
	}{
		{
			name:         "empty config runs all tests",
			configTests:  []string{},
			testCategory: "files",
			expected:     true,
		},
		{
			name:         "specific test included",
			configTests:  []string{"files", "headers"},
			testCategory: "files",
			expected:     true,
		},
		{
			name:         "specific test not included",
			configTests:  []string{"files", "headers"},
			testCategory: "defaults",
			expected:     false,
		},
		{
			name:         "single test specified",
			configTests:  []string{"server"},
			testCategory: "server",
			expected:     true,
		},
		{
			name:         "single test not matching",
			configTests:  []string{"server"},
			testCategory: "files",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := MisconfigConfig{
				URL:     "http://example.com",
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 5,
				Tests:   tt.configTests,
			}

			misconfigScanner := NewMisconfigScanner(config)
			result := misconfigScanner.ShouldRunTest(tt.testCategory)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for category '%s' with config tests %v",
					tt.expected, result, tt.testCategory, tt.configTests)
			}
		})
	}
}

func TestMisconfigScanner_ResultAggregation(t *testing.T) {
	config := MisconfigConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{},
	}

	misconfigScanner := NewMisconfigScanner(config)

	// Test initial state
	results := misconfigScanner.GetResults()
	if len(results) != 0 {
		t.Errorf("Expected 0 initial results, got %d", len(results))
	}

	// Test adding results
	result1 := MisconfigResult{
		URL:         "http://example.com/.env",
		Category:    "sensitive-files",
		Finding:     "Sensitive file exposed",
		Evidence:    "File accessible",
		RiskLevel:   "High",
		Remediation: "Remove file",
	}

	result2 := MisconfigResult{
		URL:         "http://example.com",
		Category:    "headers",
		Finding:     "Missing security header",
		Evidence:    "X-Frame-Options not found",
		RiskLevel:   "Medium",
		Remediation: "Add header",
	}

	misconfigScanner.AddResult(result1)
	misconfigScanner.AddResult(result2)

	results = misconfigScanner.GetResults()
	if len(results) != 2 {
		t.Errorf("Expected 2 results after adding, got %d", len(results))
	}

	// Test clearing results
	misconfigScanner.ClearResults()
	results = misconfigScanner.GetResults()
	if len(results) != 0 {
		t.Errorf("Expected 0 results after clearing, got %d", len(results))
	}
}

func TestMisconfigScanner_ThreadSafety(t *testing.T) {
	config := MisconfigConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 5,
		Tests:   []string{},
	}

	misconfigScanner := NewMisconfigScanner(config)

	// Test concurrent result addition
	var wg sync.WaitGroup
	numGoroutines := 100
	resultsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < resultsPerGoroutine; j++ {
				result := MisconfigResult{
					URL:         fmt.Sprintf("http://example.com/test-%d-%d", goroutineID, j),
					Category:    "test",
					Finding:     fmt.Sprintf("Test finding %d-%d", goroutineID, j),
					Evidence:    "Test evidence",
					RiskLevel:   "Low",
					Remediation: "Test remediation",
				}
				misconfigScanner.AddResult(result)
			}
		}(i)
	}

	wg.Wait()

	results := misconfigScanner.GetResults()
	expectedCount := numGoroutines * resultsPerGoroutine
	if len(results) != expectedCount {
		t.Errorf("Expected %d results from concurrent addition, got %d", expectedCount, len(results))
	}
}
