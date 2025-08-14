package fixtures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

type MockProvider interface {
	CreateMockServer(responses map[string]MockResponse) *httptest.Server
	SimulateNetworkDelay(duration time.Duration)
	SimulateNetworkError(errorType string)
	CreateHTTPBinMock() *httptest.Server
}

type MockResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Delay      time.Duration
}

type mockProvider struct {
	networkDelay time.Duration
	errorMode    string
}

func NewMockProvider() MockProvider {
	return &mockProvider{}
}

func (mp *mockProvider) CreateMockServer(responses map[string]MockResponse) *httptest.Server {
	mux := http.NewServeMux()
	
	for path, response := range responses {
		mux.HandleFunc(path, mp.createHandler(response))
	}
	
	return httptest.NewServer(mux)
}

func (mp *mockProvider) createHandler(response MockResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mp.errorMode != "" {
			mp.handleError(w, r)
			return
		}
		
		if response.Delay > 0 || mp.networkDelay > 0 {
			delay := response.Delay
			if mp.networkDelay > 0 {
				delay = mp.networkDelay
			}
			time.Sleep(delay)
		}
		
		for key, value := range response.Headers {
			w.Header().Set(key, value)
		}
		
		w.WriteHeader(response.StatusCode)
		w.Write([]byte(response.Body))
	}
}

func (mp *mockProvider) handleError(w http.ResponseWriter, r *http.Request) {
	switch mp.errorMode {
	case "timeout":
		w.WriteHeader(http.StatusRequestTimeout)
	case "connection_refused":
		w.WriteHeader(http.StatusServiceUnavailable)
	case "internal_error":
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (mp *mockProvider) SimulateNetworkDelay(duration time.Duration) {
	mp.networkDelay = duration
}

func (mp *mockProvider) SimulateNetworkError(errorType string) {
	mp.errorMode = errorType
}

func (mp *mockProvider) CreateHTTPBinMock() *httptest.Server {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"args":    r.URL.Query(),
			"headers": r.Header,
			"origin":  r.RemoteAddr,
			"url":     fmt.Sprintf("http://httpbin.org%s", r.URL.Path),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		
		response := map[string]interface{}{
			"args":    r.URL.Query(),
			"data":    string(body),
			"files":   map[string]string{},
			"form":    map[string]string{},
			"headers": r.Header,
			"json":    nil,
			"origin":  r.RemoteAddr,
			"url":     fmt.Sprintf("http://httpbin.org%s", r.URL.Path),
		}
		
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			var jsonData interface{}
			json.Unmarshal(body, &jsonData)
			response["json"] = jsonData
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		statusStr := strings.TrimPrefix(r.URL.Path, "/status/")
		var statusCode int
		fmt.Sscanf(statusStr, "%d", &statusCode)
		
		if statusCode == 0 {
			statusCode = 200
		}
		
		w.WriteHeader(statusCode)
		if statusCode >= 400 {
			w.Write([]byte(fmt.Sprintf("Status: %d", statusCode)))
		}
	})
	
	mux.HandleFunc("/delay/", func(w http.ResponseWriter, r *http.Request) {
		delayStr := strings.TrimPrefix(r.URL.Path, "/delay/")
		var delaySeconds int
		fmt.Sscanf(delayStr, "%d", &delaySeconds)
		
		if delaySeconds > 0 && delaySeconds <= 2 {
			time.Sleep(time.Duration(delaySeconds*100) * time.Millisecond)
		}
		
		response := map[string]interface{}{
			"args":    r.URL.Query(),
			"headers": r.Header,
			"origin":  r.RemoteAddr,
			"url":     fmt.Sprintf("http://httpbin.org%s", r.URL.Path),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>httpbin mock server</h1></body></html>`))
	})
	
	return httptest.NewServer(mux)
}