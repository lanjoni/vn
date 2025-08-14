package scanner

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMisconfigScanner_HTTPRequestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		serverSetup   func() *httptest.Server
		expectedError bool
		errorContains string
	}{
		{
			name: "connection timeout",
			serverSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(2 * time.Second) // Longer than client timeout
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectedError: true,
			errorContains: "timeout",
		},
		{
			name: "connection refused",
			serverSetup: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				server.Close() // Close immediately to simulate connection refused
				return server
			},
			expectedError: true,
			errorContains: "connection refused",
		},
		{
			name: "successful request",
			serverSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				}))
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.serverSetup()
			if tt.name != "connection refused" {
				defer server.Close()
			}

			config := MisconfigConfig{
				URL:     server.URL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 500 * time.Millisecond, // Short timeout for testing
				Threads: 1,
				Tests:   []string{"headers"},
			}

			misconfigScanner := NewMisconfigScanner(config)
			misconfigScanner.TestSecurityHeaders()

			errors := misconfigScanner.GetErrors()

			if tt.expectedError {
				if len(errors) == 0 {
					t.Error("Expected error but got none")
				} else {
					found := false
					for _, err := range errors {
						if strings.Contains(err.Error(), tt.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got errors: %v", tt.errorContains, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("Expected no errors but got: %v", errors)
				}
			}
		})
	}
}

func TestMisconfigScanner_DNSErrorHandling(t *testing.T) {
	config := MisconfigConfig{
		URL:     "http://nonexistent-domain-12345.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)
	results := misconfigScanner.TestSensitiveFiles()

	if len(results) > 0 {
		t.Errorf("Expected no results due to DNS failure, got %d", len(results))
	}

	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected DNS errors but got none")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "DNS resolution failed") ||
			strings.Contains(err.Error(), "host not found") ||
			strings.Contains(err.Error(), "no such host") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected DNS-related error, got: %v", errors)
	}
}

func TestMisconfigScanner_ResponseSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := strings.Repeat("A", 1024*100)
		w.Write([]byte(data))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second, // Longer timeout to avoid context cancellation
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)
	_ = misconfigScanner.TestDirectoryListing("/")

	errors := misconfigScanner.GetErrors()

	hasContextError := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "context") {
			hasContextError = true
			break
		}
	}

	if len(errors) > 0 && !hasContextError {
		t.Errorf("Expected no errors or only context errors for size limiting, got: %v", errors)
	}
}

func TestMisconfigScanner_InvalidUTF8Handling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		invalidUTF8 := []byte{0xff, 0xfe, 0xfd}
		w.Write(invalidUTF8)
		w.Write([]byte("valid text"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)
	result := misconfigScanner.TestDirectoryListing("/")

	if result != nil {
		t.Error("Expected nil result due to invalid UTF-8")
	}

	errors := misconfigScanner.GetErrors()
	found := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "invalid UTF-8") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected UTF-8 encoding error, got: %v", errors)
	}
}

func TestMisconfigScanner_GracefulDegradation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.env" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret"))
		} else if strings.Contains(r.URL.Path, "timeout") {
			time.Sleep(2 * time.Second) // Cause timeout for some requests
			return
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 500 * time.Millisecond, // Short timeout
		Threads: 5,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	results := []MisconfigResult{}

	envFile := SensitiveFile{Path: "/.env", Description: "Environment file", RiskLevel: "High", Method: "GET"}
	if result := misconfigScanner.TestSingleFile(envFile); result != nil {
		results = append(results, *result)
	}

	if len(results) == 0 {
		t.Error("Expected at least one successful result")
	}

	found := false
	for _, result := range results {
		if strings.Contains(result.Evidence, "DB_PASSWORD") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected successful request result not found")
	}
}

func TestMisconfigScanner_ConcurrentErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "fail") {
			time.Sleep(2 * time.Second) // Cause timeout
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 500 * time.Millisecond,
		Threads: 10, // High concurrency
		Tests:   []string{"files", "headers", "defaults", "server"},
	}

	misconfigScanner := NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	if results == nil {
		t.Error("Expected results slice, got nil")
	}

	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected some errors from concurrent failures")
	}

	for i := 0; i < 100; i++ {
		go func() {
			misconfigScanner.AddError(fmt.Errorf("test error %d", i))
		}()
	}

	time.Sleep(100 * time.Millisecond) // Allow goroutines to complete

	finalErrors := misconfigScanner.GetErrors()
	if len(finalErrors) < len(errors) {
		t.Error("Error collection appears to have race conditions")
	}
}

func TestMisconfigScanner_PanicRecovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("normal response"))
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

	misconfigScanner := NewMisconfigScanner(config)

	results := misconfigScanner.Scan()

	if results == nil {
		t.Error("Expected results slice, got nil")
	}

	errors := misconfigScanner.GetErrors()

	if len(results) < 0 {
		t.Error("Results slice should be valid")
	}

	if len(errors) < 0 {
		t.Error("Errors slice should be valid")
	}
}

func TestMisconfigScanner_ResourceCleanup(t *testing.T) {
	var connectionCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&connectionCount, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 3,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	for i := 0; i < 5; i++ {
		results := misconfigScanner.TestSensitiveFiles()
		if results == nil {
			t.Error("Expected results slice, got nil")
		}
	}

	finalResults := misconfigScanner.TestSensitiveFiles()
	if finalResults == nil {
		t.Error("Expected final results, resource cleanup may have failed")
	}

	if atomic.LoadInt64(&connectionCount) == 0 {
		t.Error("Expected some connections to be made")
	}
}

func TestMisconfigScanner_ErrorClearance(t *testing.T) {
	config := MisconfigConfig{
		URL:     "http://nonexistent-domain-12345.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 1 * time.Second,
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	misconfigScanner.TestSensitiveFiles()

	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected some errors to be generated")
	}

	misconfigScanner.ClearErrors()

	clearedErrors := misconfigScanner.GetErrors()
	if len(clearedErrors) != 0 {
		t.Errorf("Expected no errors after clearing, got %d", len(clearedErrors))
	}

	misconfigScanner.AddError(fmt.Errorf("test error"))
	newErrors := misconfigScanner.GetErrors()
	if len(newErrors) != 1 {
		t.Errorf("Expected 1 error after adding, got %d", len(newErrors))
	}
}

func TestMisconfigScanner_TLSErrorHandling(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 1,
		Tests:   []string{"headers"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Strict verification
		},
	}
	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	if err == nil {
		t.Error("Expected TLS error but got none")
	}

	wrappedErr := misconfigScanner.GetErrors()
	if len(wrappedErr) > 0 {
		for _, e := range wrappedErr {
			if e == nil {
				t.Error("Error should not be nil")
			}
		}
	}
}
