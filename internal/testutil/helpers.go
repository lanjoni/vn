package testutil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func MockServer(responses map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path + "?" + r.URL.RawQuery
		
		if response, exists := responses[key]; exists {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
			return
		}
		
		for pattern, response := range responses {
			if strings.Contains(key, pattern) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
				return
			}
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Default response"))
	}))
}

func AssertContains(t *testing.T, actual, expected, context string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Errorf("%s: expected to contain '%s', but got '%s'", context, expected, actual)
	}
}

func AssertNotContains(t *testing.T, actual, unwanted, context string) {
	t.Helper()
	if strings.Contains(actual, unwanted) {
		t.Errorf("%s: expected NOT to contain '%s', but got '%s'", context, unwanted, actual)
	}
}

func AssertEqual(t *testing.T, actual, expected interface{}, context string) {
	t.Helper()
	if actual != expected {
		t.Errorf("%s: expected '%v', but got '%v'", context, expected, actual)
	}
}

func VulnerableServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		
		var params []string
		for key, values := range r.Form {
			for _, value := range values {
				params = append(params, key+"="+value)
			}
		}
		
		allParams := strings.Join(params, "&")
		
		if strings.Contains(allParams, "'") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("MySQL Error: You have an error in your SQL syntax"))
			return
		}
		
		if strings.Contains(allParams, "UNION") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Warning: mysql_fetch_array() expects parameter 1 to be resource"))
			return
		}
		
		if strings.Contains(allParams, "<script>") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Search results: " + allParams))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Normal response"))
	}))
}

func SkipIfShort(t *testing.T, reason string) {
	t.Helper()
	if testing.Short() {
		t.Skipf("Skipping %s in short mode", reason)
	}
} 