package scanner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMisconfigScanner_IntegratedErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.env":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("DB_PASSWORD=secret123"))
		case "/timeout":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		case "/large":
			data := strings.Repeat("A", 1024*200)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(data))
		case "/invalid-utf8":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{0xff, 0xfe, 0xfd})
			w.Write([]byte("valid text"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 500 * time.Millisecond,
		Threads: 3,
		Tests:   []string{"files", "headers", "defaults", "server"},
	}

	misconfigScanner := NewMisconfigScanner(config)
	results := misconfigScanner.Scan()

	if len(results) == 0 {
		t.Error("Expected some results despite various errors")
	}

	errors := misconfigScanner.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected some errors from problematic requests")
	}

	found := false
	for _, result := range results {
		if strings.Contains(result.Evidence, "DB_PASSWORD") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected successful .env file detection despite other errors")
	}

	hasTimeoutError := false
	hasUTF8Error := false
	hasHTTPError := false

	for _, err := range errors {
		errStr := err.Error()
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context") {
			hasTimeoutError = true
		}
		if strings.Contains(errStr, "UTF-8") {
			hasUTF8Error = true
		}
		if strings.Contains(errStr, "HTTP request") {
			hasHTTPError = true
		}
	}

	if !hasTimeoutError && !hasUTF8Error && !hasHTTPError {
		t.Errorf("Expected at least one type of error (timeout, UTF-8, or HTTP), got: %v", errors)
	}

	if hasTimeoutError {
		t.Log("Timeout-related error detected as expected")
	}
	if hasUTF8Error {
		t.Log("UTF-8 encoding error detected as expected")
	}
	if hasHTTPError {
		t.Log("HTTP error detected as expected")
	}
}

func TestMisconfigScanner_ErrorRecoveryAndContinuation(t *testing.T) {
	config := MisconfigConfig{
		URL:     "http://nonexistent-domain-12345.com",
		Method:  "GET",
		Headers: []string{},
		Timeout: 1 * time.Second,
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := NewMisconfigScanner(config)

	results1 := misconfigScanner.TestSensitiveFiles()
	errors1 := misconfigScanner.GetErrors()

	if len(results1) > 0 {
		t.Error("Expected no results from nonexistent domain")
	}
	if len(errors1) == 0 {
		t.Error("Expected errors from nonexistent domain")
	}

	misconfigScanner.ClearErrors()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.env" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("API_KEY=test123"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	newConfig := MisconfigConfig{
		URL:     server.URL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 5 * time.Second,
		Threads: 1,
		Tests:   []string{"files"},
	}

	newScanner := NewMisconfigScanner(newConfig)
	results2 := newScanner.TestSensitiveFiles()
	errors2 := newScanner.GetErrors()

	if len(results2) == 0 {
		t.Error("Expected results from working server")
	}
	if len(errors2) > 0 {
		t.Errorf("Expected no errors from working server, got: %v", errors2)
	}

	found := false
	for _, result := range results2 {
		if strings.Contains(result.Evidence, "API_KEY") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected API_KEY in results")
	}
}
