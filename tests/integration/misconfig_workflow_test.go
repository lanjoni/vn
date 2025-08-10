package integration

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"vn/internal/scanner"
)

func TestMisconfigScanner_FullWorkflowIntegration(t *testing.T) {
	// Create a comprehensive mock server that simulates various misconfigurations
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// Sensitive files
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret123\nAPI_KEY=abc123"))
		case "/config.php":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<?php $db_pass = 'secret'; ?>"))
		case "/backup.sql":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("CREATE TABLE users (id INT, password VARCHAR(255));"))
		case "/robots.txt":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("User-agent: *\nDisallow: /admin"))
		case "/config.bak":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("database_password=secret123"))
		
		// Directory listing
		case "/uploads/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><title>Index of /uploads</title></head><body><h1>Index of /uploads</h1><pre><a href="../">Parent Directory</a></pre></body></html>`))
		
		// Login endpoints for default credentials testing
		case "/admin":
			if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<form method="post"><input name="username"><input type="password" name="password"></form>`))
			} else if r.Method == "POST" {
				username := r.FormValue("username")
				password := r.FormValue("password")
				if (username == "admin" && password == "admin") || (username == "root" && password == "root") {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`<h1>Welcome to Dashboard</h1><a href="/logout">Logout</a>`))
				} else {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`<h1>Login Failed</h1><p>Invalid credentials</p>`))
				}
			}
		case "/login":
			if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<form action="/login" method="post"><input name="user"><input type="password" name="pass"></form>`))
			} else if r.Method == "POST" {
				user := r.FormValue("user")
				pass := r.FormValue("pass")
				if user == "admin" && pass == "password" {
					w.Header().Set("Location", "/dashboard")
					w.WriteHeader(http.StatusFound)
				} else {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`<p>Login failed</p>`))
				}
			}
		
		// Version disclosure
		case "/info.php":
			w.Header().Set("Server", "Apache/2.4.41 (Ubuntu)")
			w.Header().Set("X-Powered-By", "PHP/7.4.3")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body><h1>PHP Version 7.4.3</h1><p>Apache/2.4.41 Server</p></body></html>`))
		
		// Default installation page
		case "/default.html":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><title>Apache2 Ubuntu Default Page</title></head><body><h1>It works!</h1><p>This is the default welcome page</p></body></html>`))
		
		// HTTP methods testing
		case "/methods-test":
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
		
		// Information leakage
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`<html><body><h1>Application Error</h1><p>Error: Could not open file '/var/www/html/config.php'</p></body></html>`))
		
		// Default response (missing security headers)
		default:
			// Intentionally missing security headers
			w.Header().Set("Server", "Apache/2.4.41")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 5,
		Tests:   []string{"files", "headers", "defaults", "server"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	// Verify comprehensive results
	if len(results) < 10 {
		t.Errorf("Expected at least 10 misconfigurations, got %d", len(results))
	}

	// Categorize results
	categories := make(map[string]int)
	riskLevels := make(map[string]int)
	
	for _, result := range results {
		categories[result.Category]++
		riskLevels[result.RiskLevel]++
		
		// Verify all results have required fields
		if result.URL == "" {
			t.Error("Result missing URL")
		}
		if result.Category == "" {
			t.Error("Result missing Category")
		}
		if result.Finding == "" {
			t.Error("Result missing Finding")
		}
		if result.Evidence == "" {
			t.Error("Result missing Evidence")
		}
		if result.RiskLevel == "" {
			t.Error("Result missing RiskLevel")
		}
		if result.Remediation == "" {
			t.Error("Result missing Remediation")
		}
	}

	// Verify we found issues in all categories
	expectedCategories := []string{"sensitive-files", "headers", "defaults", "server-config"}
	for _, category := range expectedCategories {
		if categories[category] == 0 {
			t.Errorf("Expected to find issues in category '%s'", category)
		}
	}

	// Verify we have different risk levels
	if riskLevels["High"] == 0 {
		t.Error("Expected to find High risk issues")
	}
	if riskLevels["Medium"] == 0 {
		t.Error("Expected to find Medium risk issues")
	}

	// Check for errors
	errors := misconfigScanner.GetErrors()
	if len(errors) > 0 {
		t.Logf("Scanner errors (may be expected): %v", errors)
	}

	t.Logf("Full workflow results: %d total, Categories: %v, Risk levels: %v", 
		len(results), categories, riskLevels)
}

func TestMisconfigScanner_WorkflowWithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for custom headers
		if r.Header.Get("Authorization") == "Bearer test-token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Authenticated response"))
		} else if r.Header.Get("User-Agent") == "VN-Scanner" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Custom user agent detected"))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
		}
	}))
	defer server.Close()

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{
			"Authorization: Bearer test-token",
			"User-Agent: VN-Scanner",
		},
		Timeout: 5 * time.Second,
		Threads: 3,
		Tests:   []string{"headers"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.TestSecurityHeaders()

	// Should be able to make requests with custom headers
	errors := misconfigScanner.GetErrors()
	if len(errors) > 0 {
		t.Errorf("Unexpected errors with custom headers: %v", errors)
	}

	// Should find missing security headers
	if len(results) == 0 {
		t.Error("Expected to find missing security headers")
	}
}

func TestMisconfigScanner_WorkflowWithDifferentMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("POST response"))
		case "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("PUT response"))
		case "DELETE":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DELETE response"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("GET response"))
		}
	}))
	defer server.Close()

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	
	for _, method := range methods {
		t.Run("Method_"+method, func(t *testing.T) {
			config := scanner.MisconfigConfig{
				URL:     server.URL,
				Method:  method,
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: 2,
				Tests:   []string{"server"},
			}

			misconfigScanner := scanner.NewMisconfigScanner(config)
			results := misconfigScanner.TestHTTPMethods()

			// Should detect dangerous methods regardless of base method
			if method == "PUT" || method == "DELETE" {
				if len(results) == 0 {
					t.Errorf("Expected to detect dangerous method %s", method)
				}
			}
		})
	}
}

func TestMisconfigScanner_WorkflowErrorRecovery(t *testing.T) {
	// Create a server that fails for some requests but succeeds for others
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		
		// Fail every 3rd request
		if requestCount%3 == 0 {
			time.Sleep(2 * time.Second) // Cause timeout
			return
		}
		
		// Succeed for other requests
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("API_KEY=test123"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 500 * time.Millisecond, // Short timeout to trigger failures
		Threads: 5,
		Tests:   []string{"files", "headers"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	// Should have some results despite errors
	if len(results) == 0 {
		t.Error("Expected some results despite intermittent failures")
	}

	// Should have collected errors
	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected some errors from failed requests")
	}

	// Verify scanner continues working after errors
	misconfigScanner.ClearErrors()
	newResults := misconfigScanner.TestSensitiveFiles()
	
	// Should still be able to get results
	if len(newResults) == 0 {
		t.Error("Scanner should continue working after errors")
	}
}

func TestMisconfigScanner_WorkflowWithSpecificTests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("SECRET=test"))
		case "/admin":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Dashboard"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<form method="post"><input name="username"><input type="password" name="password"></form>`))
			}
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	testCases := []struct {
		name          string
		tests         []string
		expectFiles   bool
		expectHeaders bool
		expectDefaults bool
		expectServer  bool
	}{
		{
			name:        "Files only",
			tests:       []string{"files"},
			expectFiles: true,
		},
		{
			name:          "Headers only",
			tests:         []string{"headers"},
			expectHeaders: true,
		},
		{
			name:           "Defaults only",
			tests:          []string{"defaults"},
			expectDefaults: true,
		},
		{
			name:         "Server only",
			tests:        []string{"server"},
			expectServer: true,
		},
		{
			name:          "Files and headers",
			tests:         []string{"files", "headers"},
			expectFiles:   true,
			expectHeaders: true,
		},
		{
			name:           "All tests",
			tests:          []string{"files", "headers", "defaults", "server"},
			expectFiles:    true,
			expectHeaders:  true,
			expectDefaults: true,
			expectServer:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := scanner.MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 10 * time.Second,
				Threads: 3,
				Tests:   tc.tests,
			}

			misconfigScanner := scanner.NewMisconfigScanner(config)
			results := misconfigScanner.Scan()

			categories := make(map[string]bool)
			for _, result := range results {
				categories[result.Category] = true
			}

			if tc.expectFiles && !categories["sensitive-files"] {
				t.Error("Expected sensitive-files results")
			}
			if tc.expectHeaders && !categories["headers"] {
				t.Error("Expected headers results")
			}
			if tc.expectDefaults && !categories["defaults"] {
				t.Error("Expected defaults results")
			}
			if tc.expectServer && !categories["server-config"] {
				t.Error("Expected server-config results")
			}

			// Verify only expected categories are present
			if !tc.expectFiles && categories["sensitive-files"] {
				t.Error("Unexpected sensitive-files results")
			}
			if !tc.expectHeaders && categories["headers"] {
				t.Error("Unexpected headers results")
			}
			if !tc.expectDefaults && categories["defaults"] {
				t.Error("Unexpected defaults results")
			}
			if !tc.expectServer && categories["server-config"] {
				t.Error("Unexpected server-config results")
			}
		})
	}
}

func TestMisconfigScanner_WorkflowResultAggregation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return different responses for different paths
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret"))
		case "/config.php":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<?php $pass = 'secret'; ?>"))
		case "/admin":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Welcome"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<form method="post"><input name="username"><input type="password" name="password"></form>`))
			}
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 5,
		Tests:   []string{"files", "headers", "defaults"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Run scan multiple times to test result aggregation
	allResults := []scanner.MisconfigResult{}
	
	for i := 0; i < 3; i++ {
		misconfigScanner.ClearResults()
		results := misconfigScanner.Scan()
		allResults = append(allResults, results...)
	}

	// Verify results are consistent across runs
	if len(allResults) == 0 {
		t.Error("Expected results from multiple scan runs")
	}

	// Group results by finding to check consistency
	findingCounts := make(map[string]int)
	for _, result := range allResults {
		findingCounts[result.Finding]++
	}

	// Each finding should appear consistently across runs (may be more than 3 due to multiple instances)
	// The key is that the count should be divisible by 3 (number of runs)
	for finding, count := range findingCounts {
		if count%3 != 0 {
			t.Errorf("Finding '%s' appeared %d times, expected multiple of 3", finding, count)
		}
	}
}

func TestMisconfigScanner_WorkflowThreadSafety(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add small delay to increase chance of race conditions
		time.Sleep(10 * time.Millisecond)
		
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("SECRET=test"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 10, // High concurrency
		Tests:   []string{"files", "headers"},
	}

	// Run multiple scanners concurrently to test thread safety
	numScanners := 5
	results := make([][]scanner.MisconfigResult, numScanners)
	errors := make([][]error, numScanners)
	
	var wg sync.WaitGroup
	for i := 0; i < numScanners; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			misconfigScanner := scanner.NewMisconfigScanner(config)
			results[index] = misconfigScanner.Scan()
			errors[index] = misconfigScanner.GetErrors()
		}(i)
	}
	
	wg.Wait()

	// Verify all scanners completed successfully
	for i, result := range results {
		if result == nil {
			t.Errorf("Scanner %d returned nil results", i)
		}
		if len(result) == 0 {
			t.Errorf("Scanner %d returned no results", i)
		}
	}

	// Results should be consistent across scanners
	expectedResultCount := len(results[0])
	for i := 1; i < numScanners; i++ {
		if len(results[i]) != expectedResultCount {
			t.Errorf("Scanner %d returned %d results, expected %d", 
				i, len(results[i]), expectedResultCount)
		}
	}
}