package scanner_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vn/internal/scanner"
)

func TestNewSQLiScanner(t *testing.T) {
	config := scanner.SQLiConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}

	sqliScanner := scanner.NewSQLiScanner(config)

	if sqliScanner.GetConfig().URL != config.URL {
		t.Errorf("Expected URL %s, got %s", config.URL, sqliScanner.GetConfig().URL)
	}

	if sqliScanner.GetConfig().Method != config.Method {
		t.Errorf("Expected Method %s, got %s", config.Method, sqliScanner.GetConfig().Method)
	}

	if sqliScanner.GetClient().Timeout != config.Timeout {
		t.Errorf("Expected Timeout %v, got %v", config.Timeout, sqliScanner.GetClient().Timeout)
	}
}

func TestDetectSQLError(t *testing.T) {
	config := scanner.SQLiConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	sqliScanner := scanner.NewSQLiScanner(config)

	testCases := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "MySQL error detected",
			body:     "You have an error in your SQL syntax",
			expected: true,
		},
		{
			name:     "PostgreSQL error detected",
			body:     "syntax error at or near",
			expected: true,
		},
		{
			name:     "MSSQL error detected",
			body:     "Microsoft OLE DB Provider for SQL Server",
			expected: true,
		},
		{
			name:     "Oracle error detected",
			body:     "ORA-00933: SQL command not properly ended",
			expected: true,
		},
		{
			name:     "SQLite error detected",
			body:     "SQLite error: syntax error near",
			expected: true,
		},
		{
			name:     "Generic SQL error detected",
			body:     "Database error occurred",
			expected: true,
		},
		{
			name:     "No SQL error",
			body:     "Welcome to our website",
			expected: false,
		},
		{
			name:     "Case insensitive detection",
			body:     "MYSQL_ERROR: Invalid query",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sqliScanner.DetectSQLError(tc.body)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for body: %s", tc.expected, result, tc.body)
			}
		})
	}
}

func TestDetectUnionSuccess(t *testing.T) {
	config := scanner.SQLiConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	sqliScanner := scanner.NewSQLiScanner(config)

	testCases := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "MySQL fetch error",
			body:     "Warning: mysql_fetch_array() expects parameter 1 to be resource",
			expected: true,
		},
		{
			name:     "MySQL fetch function",
			body:     "mysql_fetch_row(): supplied argument is not a valid MySQL result",
			expected: true,
		},
		{
			name:     "Column count mismatch",
			body:     "The used SELECT statements have a different number of columns",
			expected: true,
		},
		{
			name:     "No union indicators",
			body:     "Normal page content",
			expected: false,
		},
		{
			name:     "Case insensitive",
			body:     "WARNING: MYSQL_FETCH_ARRAY() expects parameter",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sqliScanner.DetectUnionSuccess(tc.body)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for body: %s", tc.expected, result, tc.body)
			}
		})
	}
}

func TestDetectNoSQLError(t *testing.T) {
	config := scanner.SQLiConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	sqliScanner := scanner.NewSQLiScanner(config)

	testCases := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "MongoDB error",
			body:     "MongoError: invalid query",
			expected: true,
		},
		{
			name:     "BSON error",
			body:     "BSON error: invalid document",
			expected: true,
		},
		{
			name:     "CouchDB error",
			body:     "CouchDB error occurred",
			expected: true,
		},
		{
			name:     "JSON parse error",
			body:     "JSON parse error: unexpected token",
			expected: true,
		},
		{
			name:     "No NoSQL error",
			body:     "Everything is working fine",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sqliScanner.DetectNoSQLError(tc.body)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for body: %s", tc.expected, result, tc.body)
			}
		})
	}
}

func TestSQLiScannerIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		param := r.URL.Query().Get("id")

		if param == "'" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("You have an error in your SQL syntax"))
			return
		}

		if param == "1' UNION SELECT NULL--" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Warning: mysql_fetch_array() expects parameter 1 to be resource"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Normal response"))
	}))
	defer server.Close()

	config := scanner.SQLiConfig{
		URL:     server.URL + "?id=1",
		Method:  "GET",
		Params:  []string{"id"},
		Timeout: 5 * time.Second,
		Threads: 2,
	}

	sqliScanner := scanner.NewSQLiScanner(config)
	results := sqliScanner.Scan()

	if len(results) == 0 {
		t.Error("Expected to find SQL injection vulnerabilities, but found none")
	}

	for _, result := range results {
		if result.URL == "" {
			t.Error("Result URL should not be empty")
		}
		if result.Parameter == "" {
			t.Error("Result Parameter should not be empty")
		}
		if result.Payload == "" {
			t.Error("Result Payload should not be empty")
		}
		if result.RiskLevel == "" {
			t.Error("Result RiskLevel should not be empty")
		}
	}
}

func TestSQLiPayloads(t *testing.T) {
	config := scanner.SQLiConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}

	sqliScanner := scanner.NewSQLiScanner(config)
	if sqliScanner == nil {
		t.Error("Expected scanner to be created successfully")
	}

	if sqliScanner.GetConfig().URL != config.URL {
		t.Error("Scanner configuration not set properly")
	}
}

func TestSQLiAnalyzeResponse(t *testing.T) {
	config := scanner.SQLiConfig{
		URL:     "http://test.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	sqliScanner := scanner.NewSQLiScanner(config)

	testCases := []struct {
		name        string
		body        string
		expectError bool
		expectUnion bool
	}{
		{
			name:        "Error-based detection",
			body:        "You have an error in your SQL syntax",
			expectError: true,
			expectUnion: false,
		},
		{
			name:        "Union-based detection",
			body:        "The used SELECT statements have a different number of columns",
			expectError: false,
			expectUnion: true,
		},
		{
			name:        "No vulnerability",
			body:        "Normal response",
			expectError: false,
			expectUnion: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errorResult := sqliScanner.DetectSQLError(tc.body)
			unionResult := sqliScanner.DetectUnionSuccess(tc.body)

			if errorResult != tc.expectError {
				t.Errorf("Expected error detection %v, got %v for body: %s", tc.expectError, errorResult, tc.body)
			}

			if unionResult != tc.expectUnion {
				t.Errorf("Expected union detection %v, got %v for body: %s", tc.expectUnion, unionResult, tc.body)
			}
		})
	}
}
