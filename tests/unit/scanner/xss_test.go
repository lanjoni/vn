package scanner_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	
	"vn/internal/scanner"
)

func TestNewXSSScanner(t *testing.T) {
	config := scanner.XSSConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	
	xssScanner := scanner.NewXSSScanner(config)
	
	if xssScanner.GetConfig().URL != config.URL {
		t.Errorf("Expected URL %s, got %s", config.URL, xssScanner.GetConfig().URL)
	}
	
	if xssScanner.GetConfig().Method != config.Method {
		t.Errorf("Expected Method %s, got %s", config.Method, xssScanner.GetConfig().Method)
	}
	
	if xssScanner.GetClient().Timeout != config.Timeout {
		t.Errorf("Expected Timeout %v, got %v", config.Timeout, xssScanner.GetClient().Timeout)
	}
}

func TestIsPayloadReflected(t *testing.T) {
	config := scanner.XSSConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	xssScanner := scanner.NewXSSScanner(config)
	
	testCases := []struct {
		name     string
		payload  string
		body     string
		expected bool
	}{
		{
			name:     "Exact payload reflection",
			payload:  "<script>alert('XSS')</script>",
			body:     "Your search: <script>alert('XSS')</script> returned no results",
			expected: true,
		},
		{
			name:     "Script tag pattern match",
			payload:  "<script>alert('test')</script>",
			body:     "Found <script> tag in response",
			expected: true,
		},
		{
			name:     "JavaScript URL reflection",
			payload:  "javascript:alert('XSS')",
			body:     "Link: javascript:alert('XSS')",
			expected: true,
		},
		{
			name:     "Event handler reflection",
			payload:  "<img src=x onerror=alert('XSS')>",
			body:     "Image: <img src=x onerror=alert('XSS')>",
			expected: true,
		},
		{
			name:     "Partial dangerous pattern",
			payload:  "<svg onload=alert('XSS')>",
			body:     "SVG element with onload handler detected",
			expected: true,
		},
		{
			name:     "No reflection",
			payload:  "<script>alert('XSS')</script>",
			body:     "Welcome to our safe website",
			expected: false,
		},
		{
			name:     "Case insensitive detection",
			payload:  "<SCRIPT>alert('XSS')</SCRIPT>",
			body:     "Found dangerous <script> content",
			expected: true,
		},
		{
			name:     "Iframe detection",
			payload:  "<iframe src=javascript:alert('XSS')>",
			body:     "Page contains <iframe> element",
			expected: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := xssScanner.IsPayloadReflected(tc.payload, tc.body)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for payload: %s, body: %s", tc.expected, result, tc.payload, tc.body)
			}
		})
	}
}

func TestGetRiskLevel(t *testing.T) {
	config := scanner.XSSConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	xssScanner := scanner.NewXSSScanner(config)
	
	testCases := []struct {
		name        string
		payloadType string
		payload     string
		expected    string
	}{
		{
			name:        "Script tag - High risk",
			payloadType: "reflected",
			payload:     "<script>alert('XSS')</script>",
			expected:    "High",
		},
		{
			name:        "JavaScript URL - High risk",
			payloadType: "reflected",
			payload:     "javascript:alert('XSS')",
			expected:    "High",
		},
		{
			name:        "Event handler - Medium risk",
			payloadType: "reflected",
			payload:     "<img src=x onerror=alert('XSS')>",
			expected:    "Medium",
		},
		{
			name:        "Onload handler - Medium risk",
			payloadType: "reflected",
			payload:     "<body onload=alert('XSS')>",
			expected:    "Medium",
		},
		{
			name:        "Other payload - Low risk",
			payloadType: "reflected",
			payload:     "<div>test</div>",
			expected:    "Low",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := xssScanner.GetRiskLevel(tc.payloadType, tc.payload)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s for payload: %s", tc.expected, result, tc.payload)
			}
		})
	}
}

func TestXSSPayloads(t *testing.T) {
	config := scanner.XSSConfig{
		URL:     "http://example.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	
	xssScanner := scanner.NewXSSScanner(config)
	if xssScanner == nil {
		t.Error("Expected scanner to be created successfully")
	}
	
	if xssScanner.GetConfig().URL != config.URL {
		t.Error("Scanner configuration not set properly")
	}
}

func TestXSSScannerIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		param := r.URL.Query().Get("q")
		
		if param == "<script>alert('XSS')</script>" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Search results for: <script>alert('XSS')</script>"))
			return
		}
		
		if param == "<img src=x onerror=alert('XSS')>" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Image tag detected: onerror handler found"))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Normal search results"))
	}))
	defer server.Close()
	
	config := scanner.XSSConfig{
		URL:     server.URL + "?q=test",
		Method:  "GET",
		Params:  []string{"q"},
		Timeout: 5 * time.Second,
		Threads: 2,
	}
	
	xssScanner := scanner.NewXSSScanner(config)
	results := xssScanner.Scan()
	
	if len(results) == 0 {
		t.Error("Expected to find XSS vulnerabilities, but found none")
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
		if result.Type == "" {
			t.Error("Result Type should not be empty")
		}
		if result.RiskLevel == "" {
			t.Error("Result RiskLevel should not be empty")
		}
	}
}

func TestXSSAnalyzeResponse(t *testing.T) {
	config := scanner.XSSConfig{
		URL:     "http://test.com",
		Method:  "GET",
		Timeout: 10 * time.Second,
		Threads: 5,
	}
	xssScanner := scanner.NewXSSScanner(config)
	
	testCases := []struct {
		name     string
		payload  string
		body     string
		expected bool
	}{
		{
			name:     "Reflected XSS detected",
			payload:  "<script>alert('XSS')</script>",
			body:     "Your search: <script>alert('XSS')</script>",
			expected: true,
		},
		{
			name:     "Partial reflection detected",
			payload:  "<img src=x onerror=alert('XSS')>",
			body:     "Found onerror handler in response",
			expected: true,
		},
		{
			name:     "No vulnerability",
			payload:  "<script>alert('XSS')</script>",
			body:     "Normal response without reflection",
			expected: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := xssScanner.IsPayloadReflected(tc.payload, tc.body)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for payload: %s, body: %s", tc.expected, result, tc.payload, tc.body)
			}
		})
	}
} 