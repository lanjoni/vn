package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestMisconfigTestServerEndpoints(t *testing.T) {
	// Test cases for misconfiguration test server endpoints
	testCases := []struct {
		name           string
		endpoint       string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		// Sensitive files testing (Requirement 1.1)
		{
			name:           "Exposed .env file",
			endpoint:       "/.env",
			method:         "GET",
			expectedStatus: 200,
			expectedBody:   "DB_PASSWORD=supersecret123",
		},
		{
			name:           "Exposed config.php file",
			endpoint:       "/config.php",
			method:         "GET",
			expectedStatus: 200,
			expectedBody:   "$db_pass = \"password123\"",
		},
		{
			name:           "Directory listing",
			endpoint:       "/uploads/",
			method:         "GET",
			expectedStatus: 200,
			expectedBody:   "Index of /uploads",
		},
		{
			name:           "Backup file exposure",
			endpoint:       "/config.bak",
			method:         "GET",
			expectedStatus: 200,
			expectedBody:   "database_password=secret123",
		},
		
		// Security headers testing (Requirement 2.1)
		{
			name:           "Missing security headers",
			endpoint:       "/insecure-headers",
			method:         "GET",
			expectedStatus: 200,
			expectedBody:   "missing security headers",
		},
		
		// Default credentials testing (Requirement 3.1)
		{
			name:           "Admin login form",
			endpoint:       "/admin",
			method:         "GET",
			expectedStatus: 200,
			expectedBody:   "Admin Login",
		},
		{
			name:           "Version disclosure",
			endpoint:       "/version-error",
			method:         "GET",
			expectedStatus: 500,
			expectedBody:   "Apache/2.4.41 (Ubuntu)",
		},
		
		// Server configuration testing (Requirement 4.1)
		{
			name:           "Information leakage",
			endpoint:       "/info-leak?file=test.txt",
			method:         "GET",
			expectedStatus: 500,
			expectedBody:   "/var/www/html/test.txt",
		},
	}

	baseURL := "http://localhost:8080"
	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, baseURL+tc.endpoint, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("Test server not running, skipping test: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			bodyStr := string(body[:n])

			if tc.expectedBody != "" && !contains(bodyStr, tc.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tc.expectedBody, bodyStr)
			}
		})
	}
}

func TestMisconfigTestServerHTTPMethods(t *testing.T) {
	// Test dangerous HTTP methods (Requirement 4.1)
	testCases := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "PUT method enabled",
			method:         "PUT",
			expectedStatus: 200,
			expectedBody:   "PUT method is enabled",
		},
		{
			name:           "DELETE method enabled",
			method:         "DELETE",
			expectedStatus: 200,
			expectedBody:   "DELETE method is enabled",
		},
		{
			name:           "TRACE method enabled",
			method:         "TRACE",
			expectedStatus: 200,
			expectedBody:   "TRACE /methods-test HTTP/1.1",
		},
		{
			name:           "OPTIONS method shows dangerous methods",
			method:         "OPTIONS",
			expectedStatus: 200,
			expectedBody:   "Dangerous HTTP methods are enabled",
		},
	}

	baseURL := "http://localhost:8080/methods-test"
	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, baseURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("Test server not running, skipping test: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			bodyStr := string(body[:n])

			if tc.expectedBody != "" && !contains(bodyStr, tc.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tc.expectedBody, bodyStr)
			}

			// Special check for OPTIONS method to verify Allow header contains dangerous methods
			if tc.method == "OPTIONS" {
				allowHeader := resp.Header.Get("Allow")
				expectedMethods := []string{"PUT", "DELETE", "TRACE"}
				for _, method := range expectedMethods {
					if !contains(allowHeader, method) {
						t.Errorf("Expected Allow header to contain '%s', got '%s'", method, allowHeader)
					}
				}
			}
		})
	}
}

func TestMisconfigTestServerSecurityHeaders(t *testing.T) {
	// Test security headers scenarios (Requirement 2.1)
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("Insecure headers endpoint missing security headers", func(t *testing.T) {
		resp, err := client.Get("http://localhost:8080/insecure-headers")
		if err != nil {
			t.Skipf("Test server not running, skipping test: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check that security headers are missing
		securityHeaders := []string{
			"X-Frame-Options",
			"X-Content-Type-Options",
			"X-XSS-Protection",
			"Strict-Transport-Security",
			"Content-Security-Policy",
		}

		for _, header := range securityHeaders {
			if resp.Header.Get(header) != "" {
				t.Errorf("Expected %s header to be missing, but found: %s", header, resp.Header.Get(header))
			}
		}

		// Should have Server header for version disclosure
		if resp.Header.Get("Server") == "" {
			t.Error("Expected Server header to be present for version disclosure testing")
		}
	})

	t.Run("Secure headers endpoint has all security headers", func(t *testing.T) {
		resp, err := client.Get("http://localhost:8080/secure-headers")
		if err != nil {
			t.Skipf("Test server not running, skipping test: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check that all security headers are present
		expectedHeaders := map[string]string{
			"X-Frame-Options":           "DENY",
			"X-Content-Type-Options":    "nosniff",
			"X-XSS-Protection":          "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'self'",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := resp.Header.Get(header)
			if actualValue != expectedValue {
				t.Errorf("Expected %s header to be '%s', got '%s'", header, expectedValue, actualValue)
			}
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}