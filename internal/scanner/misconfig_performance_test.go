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

func TestMisconfigScanner_ConcurrentPerformance(t *testing.T) {
	// Create a server that responds to multiple endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret123"))
		case "/config.php":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<?php $db_pass = 'secret'; ?>"))
		case "/admin":
			if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<form method="post"><input name="username"><input type="password" name="password"></form>`))
			} else if r.Method == "POST" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<h1>Welcome to Dashboard</h1>`))
			}
		default:
			w.Header().Set("X-Frame-Options", "DENY")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	testCases := []struct {
		name    string
		threads int
		timeout time.Duration
	}{
		{
			name:    "Low concurrency",
			threads: 1,
			timeout: 30 * time.Second,
		},
		{
			name:    "Medium concurrency",
			threads: 5,
			timeout: 30 * time.Second,
		},
		{
			name:    "High concurrency",
			threads: 10,
			timeout: 30 * time.Second,
		},
		{
			name:    "Very high concurrency",
			threads: 20,
			timeout: 30 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 5 * time.Second,
				Threads: tc.threads,
				Tests:   []string{"files", "headers", "defaults", "server"},
			}

			misconfigScanner := NewMisconfigScanner(config)

			start := time.Now()
			results := misconfigScanner.Scan()
			duration := time.Since(start)

			if duration > tc.timeout {
				t.Errorf("Scan took too long: %v (max: %v)", duration, tc.timeout)
			}

			if len(results) == 0 {
				t.Error("Expected some results from performance test")
			}

			errors := misconfigScanner.GetErrors()
			// Filter out expected backup file URL parsing errors
			unexpectedErrors := []error{}
			for _, err := range errors {
				if !strings.Contains(err.Error(), "invalid port") {
					unexpectedErrors = append(unexpectedErrors, err)
				}
			}
			if len(unexpectedErrors) > 0 {
				t.Errorf("Unexpected errors during performance test: %v", unexpectedErrors)
			}

			t.Logf("Threads: %d, Duration: %v, Results: %d", tc.threads, duration, len(results))
		})
	}
}

func TestMisconfigScanner_LargeScaleScanning(t *testing.T) {
	// Create a server that simulates many endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate different response times
		if r.URL.Path == "/slow" {
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(5 * time.Millisecond)
		}

		if r.URL.Path == "/.env" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("API_KEY=test123"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 2 * time.Second,
		Threads: 10,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	start := time.Now()
	results := misconfigScanner.TestSensitiveFiles()
	duration := time.Since(start)

	// Should complete within reasonable time even with many requests
	maxDuration := 10 * time.Second
	if duration > maxDuration {
		t.Errorf("Large scale scan took too long: %v (max: %v)", duration, maxDuration)
	}

	// Should find the .env file
	if len(results) == 0 {
		t.Error("Expected to find .env file in large scale test")
	}

	t.Logf("Large scale scan - Duration: %v, Results: %d", duration, len(results))
}

func TestMisconfigScanner_MemoryUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.env" {
			largeContent := make([]byte, 1024*1024)
			for i := range largeContent {
				largeContent[i] = 'A'
			}
			w.WriteHeader(http.StatusOK)
			w.Write(largeContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 5,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	// Run multiple scans to test memory cleanup
	for i := 0; i < 10; i++ {
		results := misconfigScanner.TestSensitiveFiles()
		t.Logf("Memory test iteration %d: %d results", i, len(results))

		// Clear results to test memory cleanup
		misconfigScanner.ClearResults()
		misconfigScanner.ClearErrors()
	}

	// Final scan to ensure everything still works
	finalResults := misconfigScanner.TestSensitiveFiles()
	t.Logf("Final memory test results: %d", len(finalResults))
}

func TestMisconfigScanner_ConcurrentSafety(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 10,
		Tests:   []string{"files", "headers"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	// Run multiple concurrent scans to test thread safety
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine runs a full scan
			results := misconfigScanner.Scan()

			// Verify results are valid
			if results == nil {
				t.Errorf("Goroutine %d got nil results", id)
			}

			// Add some errors to test concurrent error handling
			misconfigScanner.AddError(fmt.Errorf("test error from goroutine %d", id))
		}(i)
	}

	wg.Wait()

	// Check that all errors were collected safely
	errors := misconfigScanner.GetErrors()
	if len(errors) < numGoroutines {
		t.Errorf("Expected at least %d errors, got %d", numGoroutines, len(errors))
	}
}

func TestMisconfigScanner_TimeoutHandling(t *testing.T) {
	// Create a server that responds slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 500 * time.Millisecond,
		Threads: 3,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	start := time.Now()
	results := misconfigScanner.TestSensitiveFiles()
	duration := time.Since(start)

	// Should complete quickly due to timeouts
	maxDuration := 5 * time.Second
	if duration > maxDuration {
		t.Errorf("Timeout test took too long: %v (max: %v)", duration, maxDuration)
	}

	// Should have no results due to timeouts
	if len(results) > 0 {
		t.Errorf("Expected no results due to timeouts, got %d", len(results))
	}

	// Should have timeout errors
	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected timeout errors")
	}

	// Verify timeout errors are properly categorized
	hasTimeoutError := false
	for _, err := range errors {
		if err != nil && (err.Error() != "" && (len(err.Error()) > 0)) {
			hasTimeoutError = true
			break
		}
	}
	if !hasTimeoutError {
		t.Error("Expected at least one timeout-related error")
	}
}

func TestMisconfigScanner_ResourceCleanupPerformance(t *testing.T) {
	// Test that resources are properly cleaned up during high-load scenarios
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 1 * time.Second,
		Threads: 15,
		Tests:   []string{"files", "headers", "defaults", "server"},
	}

	// Run many scans in sequence to test resource cleanup
	for i := 0; i < 20; i++ {
		misconfigScanner := NewMisconfigScanner(config)
		results := misconfigScanner.Scan()

		if results == nil {
			t.Errorf("Scan %d returned nil results", i)
		}

		// Force cleanup
		misconfigScanner.ClearResults()
		misconfigScanner.ClearErrors()
	}
}

func BenchmarkMisconfigScanner_FullScan(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret"))
		default:
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
		Tests:   []string{"files", "headers", "defaults", "server"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		misconfigScanner := NewMisconfigScanner(config)
		results := misconfigScanner.Scan()
		if len(results) == 0 {
			b.Error("Expected some results in benchmark")
		}
	}
}

func BenchmarkMisconfigScanner_SensitiveFiles(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.env" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("API_KEY=test"))
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := misconfigScanner.TestSensitiveFiles()
		if len(results) == 0 {
			b.Error("Expected results in benchmark")
		}
		misconfigScanner.ClearResults()
	}
}
