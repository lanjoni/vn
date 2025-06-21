package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func vulnerableEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	id := r.URL.Query().Get("id")
	username := r.URL.Query().Get("username")
	search := r.URL.Query().Get("search")

	if r.Method == "POST" {
		r.ParseForm()
		id = r.FormValue("id")
		username = r.FormValue("username")
		search = r.FormValue("search")
	}

	var response strings.Builder
	response.WriteString("<h1>Vulnerable Test Server</h1>\n")

	switch {
	case strings.Contains(id, "UNION") || strings.Contains(username, "UNION") || strings.Contains(search, "UNION"):
		response.WriteString(`<p style="color: red;">Warning: mysql_fetch_array() expects parameter 1 to be resource</p>`)
	case strings.Contains(id, "SLEEP") || strings.Contains(username, "SLEEP") || strings.Contains(search, "SLEEP"):
		response.WriteString(`<p>Query executed successfully</p>`)
	case strings.Contains(id, "'") || strings.Contains(username, "'") || strings.Contains(search, "'"):
		response.WriteString(`<p style="color: red;">MySQL Error: You have an error in your SQL syntax; ` +
			`check the manual that corresponds to your MySQL server version for the right syntax to use near ''' at line 1</p>`)
	default:
		response.WriteString(`<p>Normal response - no vulnerability detected</p>`)
	}

	if id != "" {
		response.WriteString("<p>ID: " + id + "</p>\n")
	}
	if username != "" {
		response.WriteString("<p>Username: " + username + "</p>\n")
	}
	if search != "" {
		response.WriteString("<p>Search: " + search + "</p>\n")
	}

	w.Write([]byte(response.String()))
}

func healthEndpoint(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok", "message": "Test server is running"}`))
}

func TestVulnerableEndpoint(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		url            string
		formData       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Normal GET request",
			method:         "GET",
			url:            "/?id=1",
			expectedStatus: http.StatusOK,
			expectedBody:   "Normal response - no vulnerability detected",
		},
		{
			name:           "SQL injection with single quote",
			method:         "GET",
			url:            "/?id='",
			expectedStatus: http.StatusOK,
			expectedBody:   "MySQL Error: You have an error in your SQL syntax",
		},
		{
			name:           "SQL injection with UNION",
			method:         "GET",
			url:            "/?id=1' UNION SELECT NULL--",
			expectedStatus: http.StatusOK,
			expectedBody:   "Warning: mysql_fetch_array() expects parameter 1 to be resource",
		},
		{
			name:           "SQL injection with SLEEP",
			method:         "GET",
			url:            "/?id=1 SLEEP(5)",
			expectedStatus: http.StatusOK,
			expectedBody:   "Query executed successfully",
		},
		{
			name:           "Username parameter injection",
			method:         "GET",
			url:            "/?username=admin'",
			expectedStatus: http.StatusOK,
			expectedBody:   "MySQL Error: You have an error in your SQL syntax",
		},
		{
			name:           "Search parameter injection",
			method:         "GET",
			url:            "/?search=test' UNION SELECT 1,2,3--",
			expectedStatus: http.StatusOK,
			expectedBody:   "Warning: mysql_fetch_array() expects parameter 1 to be resource",
		},
		{
			name:           "POST request with injection",
			method:         "POST",
			url:            "/login",
			formData:       "username=admin'&password=secret",
			expectedStatus: http.StatusOK,
			expectedBody:   "MySQL Error: You have an error in your SQL syntax",
		},
		{
			name:           "POST request normal",
			method:         "POST",
			url:            "/login",
			formData:       "username=admin&password=secret",
			expectedStatus: http.StatusOK,
			expectedBody:   "Normal response - no vulnerability detected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tc.method == "POST" {
				req, err = http.NewRequest("POST", tc.url, strings.NewReader(tc.formData))
				if err != nil {
					t.Fatalf("Could not create request: %v", err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req, err = http.NewRequest("GET", tc.url, nil)
				if err != nil {
					t.Fatalf("Could not create request: %v", err)
				}
			}

			recorder := httptest.NewRecorder()
			vulnerableEndpoint(recorder, req)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}

			body := recorder.Body.String()
			if !strings.Contains(body, tc.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tc.expectedBody, body)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	recorder := httptest.NewRecorder()
	healthEndpoint(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	expectedBody := `{"status": "ok", "message": "Test server is running"}`
	body := recorder.Body.String()
	if body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestVulnerableEndpointHeaders(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	recorder := httptest.NewRecorder()
	vulnerableEndpoint(recorder, req)

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected Content-Type 'text/html', got '%s'", contentType)
	}
}

func TestMultipleParameters(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "/?id=1&username=admin'&search=test", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	recorder := httptest.NewRecorder()
	vulnerableEndpoint(recorder, req)

	body := recorder.Body.String()

	if !strings.Contains(body, "MySQL Error") {
		t.Error("Expected SQL error to be detected")
	}

	if !strings.Contains(body, "ID: 1") {
		t.Error("Expected ID parameter to be displayed")
	}

	if !strings.Contains(body, "Username: admin'") {
		t.Error("Expected Username parameter to be displayed")
	}

	if !strings.Contains(body, "Search: test") {
		t.Error("Expected Search parameter to be displayed")
	}
}
