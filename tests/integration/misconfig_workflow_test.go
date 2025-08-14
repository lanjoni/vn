//go:build integration
// +build integration

package integration

import (
	"net/http"
	"sync"
	"testing"

	"vn/internal/scanner"
	"vn/tests/shared"
	"vn/tests/shared/testserver"
)

func createMisconfigTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
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

		case "/uploads/":
			w.WriteHeader(http.StatusOK)
			htmlContent := `<html><head><title>Index of /uploads</title></head><body>` +
				`<h1>Index of /uploads</h1><pre><a href="../">Parent Directory</a></pre></body></html>`
			w.Write([]byte(htmlContent))

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

		case "/info.php":
			w.Header().Set("Server", "Apache/2.4.41 (Ubuntu)")
			w.Header().Set("X-Powered-By", "PHP/7.4.3")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body><h1>PHP Version 7.4.3</h1><p>Apache/2.4.41 Server</p></body></html>`))

		case "/default.html":
			w.WriteHeader(http.StatusOK)
			defaultPageContent := `<html><head><title>Apache2 Ubuntu Default Page</title></head><body>` +
				`<h1>It works!</h1><p>This is the default welcome page</p></body></html>`
			w.Write([]byte(defaultPageContent))

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

		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			errorContent := `<html><body><h1>Application Error</h1><p>Error: ` +
				`Could not open file '/var/www/html/config.php'</p></body></html>`
			w.Write([]byte(errorContent))

		default:
			// Intentionally missing security headers
			w.Header().Set("Server", "Apache/2.4.41")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})
}

func TestMisconfigScanner_FullWorkflowIntegration(t *testing.T) {
	timeouts := shared.GetOptimizedTimeouts()
	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    createMisconfigTestHandler(),
		ConfigName: "misconfig-full-workflow",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)
	
	shared.WaitForServerReady(t, server)

	config := scanner.MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: timeouts.HTTPRequest,
		Threads: 5,
		Tests:   []string{"files", "headers", "defaults", "server"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	if len(results) < 10 {
		t.Errorf("Expected at least 10 misconfigurations, got %d", len(results))
	}

	categories := make(map[string]int)
	riskLevels := make(map[string]int)

	for _, result := range results {
		categories[result.Category]++
		riskLevels[result.RiskLevel]++

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

	expectedCategories := []string{"sensitive-files", "headers", "defaults", "server-config"}
	for _, category := range expectedCategories {
		if categories[category] == 0 {
			t.Errorf("Expected to find issues in category '%s'", category)
		}
	}

	if riskLevels["High"] == 0 {
		t.Error("Expected to find High risk issues")
	}
	if riskLevels["Medium"] == 0 {
		t.Error("Expected to find Medium risk issues")
	}

	errors := misconfigScanner.GetErrors()
	if len(errors) > 0 {
		t.Logf("Scanner errors (may be expected): %v", errors)
	}

	t.Logf("Full workflow results: %d total, Categories: %v, Risk levels: %v",
		len(results), categories, riskLevels)
}

func TestMisconfigScanner_WorkflowWithCustomHeaders(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	customHeadersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})

	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    customHeadersHandler,
		ConfigName: "misconfig-custom-headers",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	config := scanner.MisconfigConfig{
		URL:    server.URL,
		Method: "GET",
		Headers: []string{
			"Authorization: Bearer test-token",
			"User-Agent: VN-Scanner",
		},
		Timeout: timeouts.HTTPRequest,
		Threads: 3,
		Tests:   []string{"headers"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.TestSecurityHeaders()

	errors := misconfigScanner.GetErrors()
	if len(errors) > 0 {
		t.Errorf("Unexpected errors with custom headers: %v", errors)
	}

	if len(results) == 0 {
		t.Error("Expected to find missing security headers")
	}
}

func TestMisconfigScanner_WorkflowWithDifferentMethods(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	methodsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})

	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    methodsHandler,
		ConfigName: "misconfig-methods",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run("Method_"+method, func(t *testing.T) {
			config := scanner.MisconfigConfig{
				URL:     server.URL,
				Method:  method,
				Headers: []string{},
				Timeout: timeouts.HTTPRequest,
				Threads: 2,
				Tests:   []string{"server"},
			}

			misconfigScanner := scanner.NewMisconfigScanner(config)
			results := misconfigScanner.TestHTTPMethods()

			if method == "PUT" || method == "DELETE" {
				if len(results) == 0 {
					t.Errorf("Expected to detect dangerous method %s", method)
				}
			}
		})
	}
}

func TestMisconfigScanner_WorkflowErrorRecovery(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	requestCount := 0
	errorRecoveryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount%3 == 0 {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}

		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("API_KEY=test123"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})

	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    errorRecoveryHandler,
		ConfigName: "misconfig-error-recovery",
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
		Threads: 5,
		Tests:   []string{"files", "headers"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	if len(results) == 0 {
		t.Error("Expected some results despite intermittent failures")
	}

	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected some errors from failed requests")
	}

	misconfigScanner.ClearErrors()
	newResults := misconfigScanner.TestSensitiveFiles()

	if len(newResults) == 0 {
		t.Error("Scanner should continue working after errors")
	}
}

func TestMisconfigScanner_WorkflowWithSpecificTests(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	specificTestsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})

	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    specificTestsHandler,
		ConfigName: "misconfig-specific-tests",
		Timeout:    timeouts.ServerStart,
	}

	server, err := serverPool.GetServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	testCases := []struct {
		name           string
		tests          []string
		expectFiles    bool
		expectHeaders  bool
		expectDefaults bool
		expectServer   bool
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
				Timeout: timeouts.HTTPRequest,
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
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	aggregationHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})

	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    aggregationHandler,
		ConfigName: "misconfig-aggregation",
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
		Threads: 5,
		Tests:   []string{"files", "headers", "defaults"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)

	allResults := []scanner.MisconfigResult{}

	for i := 0; i < 3; i++ {
		misconfigScanner.ClearResults()
		results := misconfigScanner.Scan()
		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		t.Error("Expected results from multiple scan runs")
	}

	findingCounts := make(map[string]int)
	for _, result := range allResults {
		findingCounts[result.Finding]++
	}

	for finding, count := range findingCounts {
		if count%3 != 0 {
			t.Errorf("Finding '%s' appeared %d times, expected multiple of 3", finding, count)
		}
	}
}

func TestMisconfigScanner_WorkflowThreadSafety(t *testing.T) {
	t.Parallel()
	timeouts := shared.GetOptimizedTimeouts()
	threadSafetyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("SECRET=test"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})

	serverPool := getSharedServerPool()
	serverConfig := testserver.ServerConfig{
		Handler:    threadSafetyHandler,
		ConfigName: "misconfig-thread-safety",
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
		Threads: 10,
		Tests:   []string{"files", "headers"},
	}

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

	for i, result := range results {
		if result == nil {
			t.Errorf("Scanner %d returned nil results", i)
		}
		if len(result) == 0 {
			t.Errorf("Scanner %d returned no results", i)
		}
	}

	expectedResultCount := len(results[0])
	for i := 1; i < numScanners; i++ {
		if len(results[i]) != expectedResultCount {
			t.Errorf("Scanner %d returned %d results, expected %d",
				i, len(results[i]), expectedResultCount)
		}
	}
}
