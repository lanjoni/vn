package scanner

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	riskHigh   = "High"
	riskMedium = "Medium"
	riskLow    = "Low"

	categoryHeaders        = "headers"
	categoryDefaults       = "defaults"
	categoryServerConfig   = "server-config"
	categorySensitiveFiles = "sensitive-files"

	httpMethodGET  = "GET"
	httpMethodPOST = "POST"

	timeoutSeconds   = 30
	keepAliveSeconds = 30
	idleConnTimeout  = 90

	maxResponseSize  = 1024 * 1024 // 1MB
	fileContentLimit = 10 * 1024   // 10KB
	directoryLimit   = 50 * 1024   // 50KB
	loginPageLimit   = 100 * 1024  // 100KB
	serverInfoLimit  = 200         // 200 bytes
)

type MisconfigConfig struct {
	URL     string
	Method  string
	Headers []string
	Timeout time.Duration
	Threads int
	Tests   []string
}

type MisconfigResult struct {
	URL         string
	Category    string
	Finding     string
	Evidence    string
	RiskLevel   string
	Remediation string
}

type MisconfigScanner struct {
	config     MisconfigConfig
	client     *http.Client
	results    []MisconfigResult
	mutex      sync.Mutex
	errors     []error
	errorMutex sync.Mutex
}

type SensitiveFileScanner interface {
	TestSensitiveFiles() []MisconfigResult
	TestDirectoryListing(path string) *MisconfigResult
	TestBackupFiles() []MisconfigResult
}

type SecurityHeaderScanner interface {
	TestSecurityHeaders() []MisconfigResult
	TestHTTPSEnforcement() *MisconfigResult
	ValidateHeaderValue(headerName, headerValue string) *MisconfigResult
}

type DefaultCredentialScanner interface {
	TestDefaultCredentials() []MisconfigResult
	TestDefaultPages() []MisconfigResult
	DetectVersionDisclosure(response string) *MisconfigResult
}

type ServerConfigScanner interface {
	TestHTTPMethods() []MisconfigResult
	TestServerBanner() *MisconfigResult
	TestErrorMessages() []MisconfigResult
	TestInsecureRedirects() []MisconfigResult
}

type SensitiveFile struct {
	Path        string
	Description string
	RiskLevel   string
	Method      string
}

type SecurityHeader struct {
	Name        string
	Required    bool
	ValidValues []string
	RiskLevel   string
	Description string
}

type DefaultCredential struct {
	Username string
	Password string
	Context  string
}

type HTTPMethod struct {
	Method      string
	Dangerous   bool
	Description string
}

func NewMisconfigScanner(config MisconfigConfig) *MisconfigScanner {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeoutSeconds * time.Second,
			KeepAlive: keepAliveSeconds * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       idleConnTimeout * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Required for testing
		},
	}

	return &MisconfigScanner{
		config: config,
		client: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
		results: make([]MisconfigResult, 0),
		errors:  make([]error, 0),
	}
}

func (m *MisconfigScanner) GetConfig() MisconfigConfig {
	return m.config
}

func (m *MisconfigScanner) GetClient() *http.Client {
	return m.client
}

func (m *MisconfigScanner) GetResults() []MisconfigResult {
	return m.results
}

func (m *MisconfigScanner) AddResult(result MisconfigResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.results = append(m.results, result)
}

func (m *MisconfigScanner) ClearResults() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.results = make([]MisconfigResult, 0)
}

func (m *MisconfigScanner) AddError(err error) {
	m.errorMutex.Lock()
	defer m.errorMutex.Unlock()
	m.errors = append(m.errors, err)
}

func (m *MisconfigScanner) GetErrors() []error {
	m.errorMutex.Lock()
	defer m.errorMutex.Unlock()
	return append([]error(nil), m.errors...)
}

func (m *MisconfigScanner) ClearErrors() {
	m.errorMutex.Lock()
	defer m.errorMutex.Unlock()
	m.errors = make([]error, 0)
}

func (m *MisconfigScanner) makeHTTPRequest(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, m.handleHTTPError(err)
	}

	return resp, nil
}

func (m *MisconfigScanner) handleHTTPError(err error) error {
	var netErr net.Error
	var dnsErr *net.DNSError
	var tlsErr tls.RecordHeaderError

	switch {
	case errors.As(err, &netErr) && netErr.Timeout():
		wrappedErr := fmt.Errorf("HTTP request timeout: %w", err)
		m.AddError(wrappedErr)
		return wrappedErr
	case errors.As(err, &dnsErr):
		wrappedErr := fmt.Errorf("DNS resolution failed: %w", err)
		m.AddError(wrappedErr)
		return wrappedErr
	case errors.As(err, &tlsErr):
		wrappedErr := fmt.Errorf("TLS/SSL error: %w", err)
		m.AddError(wrappedErr)
		return wrappedErr
	case strings.Contains(err.Error(), "connection refused"):
		wrappedErr := fmt.Errorf("connection refused: %w", err)
		m.AddError(wrappedErr)
		return wrappedErr
	case strings.Contains(err.Error(), "no such host"):
		wrappedErr := fmt.Errorf("host not found: %w", err)
		m.AddError(wrappedErr)
		return wrappedErr
	default:
		wrappedErr := fmt.Errorf("HTTP request failed: %w", err)
		m.AddError(wrappedErr)
		return wrappedErr
	}
}

func (m *MisconfigScanner) safeReadResponse(resp *http.Response, maxSize int64) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("invalid response or nil body")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			m.AddError(fmt.Errorf("failed to close response body: %w", err))
		}
	}()

	if maxSize <= 0 {
		maxSize = maxResponseSize
	}

	limitedReader := io.LimitReader(resp.Body, maxSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to read response body: %w", err)
		m.AddError(wrappedErr)
		return nil, wrappedErr
	}

	if !utf8.Valid(body) {
		m.AddError(fmt.Errorf("response contains invalid UTF-8 encoding"))
		body = []byte(strings.ToValidUTF8(string(body), "�"))
	}

	return body, nil
}

func (m *MisconfigScanner) createHTTPRequest(method, url string, body io.Reader) (*http.Request, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to create HTTP request: %w", err)
		m.AddError(wrappedErr)
		return nil, wrappedErr
	}

	for _, header := range m.config.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	return req, nil
}

var CommonSensitiveFiles = []SensitiveFile{
	{Path: "/.env", Description: "Environment configuration file", RiskLevel: "High", Method: "GET"},
	{Path: "/config.php", Description: "PHP configuration file", RiskLevel: "High", Method: "GET"},
	{Path: "/web.config", Description: "IIS configuration file", RiskLevel: "High", Method: "GET"},
	{Path: "/robots.txt", Description: "Robots exclusion file", RiskLevel: "Low", Method: "GET"},
	{Path: "/.git/config", Description: "Git configuration file", RiskLevel: "High", Method: "GET"},
	{Path: "/backup.sql", Description: "Database backup file", RiskLevel: "High", Method: "GET"},
	{Path: "/config.json", Description: "JSON configuration file", RiskLevel: "Medium", Method: "GET"},
	{Path: "/settings.php", Description: "PHP settings file", RiskLevel: "High", Method: "GET"},
}

var BackupExtensions = []string{".bak", ".old", ".backup", ".orig", "~", ".tmp", ".swp"}

var RequiredSecurityHeaders = []SecurityHeader{
	{
		Name:        "X-Frame-Options",
		Required:    true,
		ValidValues: []string{"DENY", "SAMEORIGIN"},
		RiskLevel:   "Medium",
		Description: "Prevents clickjacking attacks",
	},
	{
		Name:        "X-Content-Type-Options",
		Required:    true,
		ValidValues: []string{"nosniff"},
		RiskLevel:   "Low",
		Description: "Prevents MIME type sniffing",
	},
	{
		Name:        "Strict-Transport-Security",
		Required:    true,
		ValidValues: []string{},
		RiskLevel:   "High",
		Description: "Enforces HTTPS connections",
	},
	{
		Name:        "Content-Security-Policy",
		Required:    false,
		ValidValues: []string{},
		RiskLevel:   "Medium",
		Description: "Prevents XSS and data injection attacks",
	},
	{
		Name:        "X-XSS-Protection",
		Required:    false,
		ValidValues: []string{"1; mode=block"},
		RiskLevel:   "Low",
		Description: "Enables XSS filtering in browsers",
	},
}

var CommonDefaultCredentials = []DefaultCredential{
	{Username: "admin", Password: "admin", Context: "admin panel"},
	{Username: "root", Password: "root", Context: "system login"},
	{Username: "admin", Password: "password", Context: "admin panel"},
	{Username: "admin", Password: "", Context: "admin panel"},
	{Username: "administrator", Password: "administrator", Context: "admin panel"},
	{Username: "guest", Password: "guest", Context: "guest account"},
	{Username: "test", Password: "test", Context: "test account"},
	{Username: "user", Password: "user", Context: "user account"},
}

var DangerousHTTPMethods = []HTTPMethod{
	{Method: "PUT", Dangerous: true, Description: "Allows file uploads and modifications"},
	{Method: "DELETE", Dangerous: true, Description: "Allows resource deletion"},
	{Method: "TRACE", Dangerous: true, Description: "Can be used for XSS attacks"},
	{Method: "CONNECT", Dangerous: true, Description: "Can be used for tunneling"},
	{Method: "PATCH", Dangerous: false, Description: "Allows partial resource updates"},
	{Method: "OPTIONS", Dangerous: false, Description: "Reveals available methods"},
}

var directoryListingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<title>.*index of.*</title>`),
	regexp.MustCompile(`(?i)<h1>index of`),
	regexp.MustCompile(`(?i)directory listing for`),
	regexp.MustCompile(`(?i)<pre><a href="\.\./"`),
	regexp.MustCompile(`(?i)parent directory</a>`),
	regexp.MustCompile(`(?i)<th><a href="\?C=N;O=D">name</a></th>`),
}

func (m *MisconfigScanner) Scan() []MisconfigResult {
	m.ClearResults()
	m.ClearErrors()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.config.Threads)

	testCategories := []struct {
		name string
		fn   func()
	}{
		{"sensitive-files", m.runSensitiveFilesTests},
		{"security-headers", m.runSecurityHeadersTests},
		{"default-credentials", m.runDefaultCredentialsTests},
		{"server-config", m.runServerConfigTests},
	}

	for _, testCategory := range testCategories {
		wg.Add(1)
		go func(category string, test func()) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					m.AddError(fmt.Errorf("panic in %s test: %v", category, r))
				}
			}()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			func() {
				defer func() {
					if r := recover(); r != nil {
						m.AddError(fmt.Errorf("recovered from panic in %s: %v", category, r))
					}
				}()
				test()
			}()
		}(testCategory.name, testCategory.fn)
	}

	wg.Wait()
	return m.results
}

func (m *MisconfigScanner) runSensitiveFilesTests() {
	if m.shouldRunTest("files") {
		m.TestSensitiveFiles()
		m.TestBackupFiles()

		commonDirs := []string{"/", "/admin", "/uploads", "/files", "/backup"}
		for _, dir := range commonDirs {
			if result := m.TestDirectoryListing(dir); result != nil {
				m.AddResult(*result)
			}
		}
	}
}

func (m *MisconfigScanner) runSecurityHeadersTests() {
	if m.shouldRunTest("headers") {
		m.TestSecurityHeaders()
		if result := m.TestHTTPSEnforcement(); result != nil {
			m.AddResult(*result)
		}
	}
}

func (m *MisconfigScanner) runDefaultCredentialsTests() {
	if m.shouldRunTest("defaults") {
		m.TestDefaultCredentials()
		m.TestDefaultPages()
	}
}

func (m *MisconfigScanner) runServerConfigTests() {
	if m.shouldRunTest("server") {
		m.TestHTTPMethods()
		if result := m.TestServerBanner(); result != nil {
			m.AddResult(*result)
		}
		m.TestErrorMessages()
		m.TestInsecureRedirects()
	}
}

func (m *MisconfigScanner) shouldRunTest(category string) bool {
	if len(m.config.Tests) == 0 {
		return true
	}

	for _, test := range m.config.Tests {
		if test == category {
			return true
		}
	}
	return false
}

func (m *MisconfigScanner) ShouldRunTest(category string) bool {
	return m.shouldRunTest(category)
}

func (m *MisconfigScanner) TestSensitiveFiles() []MisconfigResult {
	var results []MisconfigResult
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.config.Threads)
	resultsChan := make(chan MisconfigResult, len(CommonSensitiveFiles))

	for _, file := range CommonSensitiveFiles {
		wg.Add(1)
		go func(sensitiveFile SensitiveFile) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if result := m.testSingleFile(sensitiveFile); result != nil {
				resultsChan <- *result
			}
		}(file)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
		m.AddResult(result)
	}

	return results
}

func (m *MisconfigScanner) TestSingleFile(file SensitiveFile) *MisconfigResult {
	return m.testSingleFile(file)
}

func (m *MisconfigScanner) testSingleFile(file SensitiveFile) *MisconfigResult {
	targetURL := strings.TrimSuffix(m.config.URL, "/") + file.Path

	req, err := m.createHTTPRequest(file.Method, targetURL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}

	if resp.StatusCode == http.StatusOK {
		body, err := m.safeReadResponse(resp, fileContentLimit)
		if err != nil {
			return nil
		}

		bodyStr := string(body)
		evidence := fmt.Sprintf("File accessible at %s (Status: %d)", file.Path, resp.StatusCode)

		if len(bodyStr) > 100 {
			evidence += fmt.Sprintf(", Content preview: %s...", bodyStr[:100])
		} else if len(bodyStr) > 0 {
			evidence += fmt.Sprintf(", Content: %s", bodyStr)
		}

		return &MisconfigResult{
			URL:         targetURL,
			Category:    categorySensitiveFiles,
			Finding:     fmt.Sprintf("Sensitive file exposed: %s", file.Description),
			Evidence:    evidence,
			RiskLevel:   file.RiskLevel,
			Remediation: fmt.Sprintf("Remove or restrict access to %s", file.Path),
		}
	} else {
		resp.Body.Close()
	}

	return nil
}

func (m *MisconfigScanner) TestDirectoryListing(path string) *MisconfigResult {
	targetURL := strings.TrimSuffix(m.config.URL, "/") + path

	req, err := m.createHTTPRequest(httpMethodGET, targetURL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}

	if resp.StatusCode == http.StatusOK {
		body, err := m.safeReadResponse(resp, directoryLimit)
		if err != nil {
			return nil
		}

		bodyStr := string(body)
		if m.detectDirectoryListing(bodyStr) {
			return &MisconfigResult{
				URL:         targetURL,
				Category:    categorySensitiveFiles,
				Finding:     "Directory listing enabled",
				Evidence:    fmt.Sprintf("Directory listing detected at %s", path),
				RiskLevel:   "Medium",
				Remediation: "Disable directory browsing in web server configuration",
			}
		}
	} else {
		resp.Body.Close()
	}

	return nil
}

func (m *MisconfigScanner) detectDirectoryListing(body string) bool {
	for _, pattern := range directoryListingPatterns {
		if pattern.MatchString(body) {
			return true
		}
	}
	return false
}

func (m *MisconfigScanner) DetectDirectoryListing(body string) bool {
	return m.detectDirectoryListing(body)
}

func (m *MisconfigScanner) TestBackupFiles() []MisconfigResult {
	var results []MisconfigResult
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.config.Threads)
	resultsChan := make(chan MisconfigResult, len(CommonSensitiveFiles)*len(BackupExtensions))

	parsedURL, err := url.Parse(m.config.URL)
	if err != nil {
		return results
	}

	basePaths := []string{
		"/index.php",
		"/index.html",
		"/config.php",
		"/admin.php",
		"/login.php",
		"/database.sql",
		"/backup.sql",
	}
	
	// Add the parsed URL path if it's not empty and starts with /
	if parsedURL.Path != "" && strings.HasPrefix(parsedURL.Path, "/") {
		basePaths = append(basePaths, parsedURL.Path)
	}

	for _, basePath := range basePaths {
		for _, ext := range BackupExtensions {
			wg.Add(1)
			go func(path, extension string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Ensure the path starts with / to avoid malformed URLs
				if !strings.HasPrefix(path, "/") {
					path = "/" + path
				}
				backupPath := path + extension
				if result := m.testBackupFile(backupPath); result != nil {
					resultsChan <- *result
				}
			}(basePath, ext)
		}
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
		m.AddResult(result)
	}

	return results
}

func (m *MisconfigScanner) testBackupFile(backupPath string) *MisconfigResult {
	targetURL := strings.TrimSuffix(m.config.URL, "/") + backupPath

	req, err := m.createHTTPRequest(httpMethodGET, targetURL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}

	if resp.StatusCode == http.StatusOK {
		body, err := m.safeReadResponse(resp, fileContentLimit)
		if err != nil {
			return nil
		}

		bodyStr := string(body)
		if len(bodyStr) > 0 {
			evidence := fmt.Sprintf("Backup file accessible at %s (Status: %d)", backupPath, resp.StatusCode)
			if len(bodyStr) > 100 {
				evidence += fmt.Sprintf(", Content preview: %s...", bodyStr[:100])
			}

			return &MisconfigResult{
				URL:         targetURL,
				Category:    categorySensitiveFiles,
				Finding:     "Backup file exposed",
				Evidence:    evidence,
				RiskLevel:   "High",
				Remediation: fmt.Sprintf("Remove backup file %s or restrict access", backupPath),
			}
		}
	} else {
		resp.Body.Close()
	}

	return nil
}

func (m *MisconfigScanner) TestSecurityHeaders() []MisconfigResult {
	var results []MisconfigResult

	req, err := m.createHTTPRequest(m.config.Method, m.config.URL, nil)
	if err != nil {
		return results
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return results
	}
	defer resp.Body.Close()

	for _, secHeader := range RequiredSecurityHeaders {
		if result := m.analyzeSecurityHeader(resp, secHeader); result != nil {
			results = append(results, *result)
			m.AddResult(*result)
		}
	}

	return results
}

func (m *MisconfigScanner) analyzeSecurityHeader(resp *http.Response, secHeader SecurityHeader) *MisconfigResult {
	headerValue := resp.Header.Get(secHeader.Name)

	if headerValue == "" {
		if secHeader.Required {
			return &MisconfigResult{
				URL:         m.config.URL,
				Category:    categoryHeaders,
				Finding:     fmt.Sprintf("Missing security header: %s", secHeader.Name),
				Evidence:    fmt.Sprintf("Header '%s' not found in response", secHeader.Name),
				RiskLevel:   secHeader.RiskLevel,
				Remediation: fmt.Sprintf("Add '%s' header to prevent %s", secHeader.Name, strings.ToLower(secHeader.Description)),
			}
		}
		return nil
	}

	if len(secHeader.ValidValues) > 0 {
		isValid := false
		for _, validValue := range secHeader.ValidValues {
			if strings.EqualFold(headerValue, validValue) ||
				strings.Contains(strings.ToLower(headerValue), strings.ToLower(validValue)) {
				isValid = true
				break
			}
		}

		if !isValid {
			return &MisconfigResult{
				URL:      m.config.URL,
				Category: categoryHeaders,
				Finding:  fmt.Sprintf("Weak security header value: %s", secHeader.Name),
				Evidence: fmt.Sprintf("Header '%s' has value '%s', expected one of: %v",
					secHeader.Name, headerValue, secHeader.ValidValues),
				RiskLevel:   "Medium",
				Remediation: fmt.Sprintf("Set '%s' header to a secure value: %v", secHeader.Name, secHeader.ValidValues),
			}
		}
	}

	return nil
}

func (m *MisconfigScanner) TestHTTPSEnforcement() *MisconfigResult {
	req, err := m.createHTTPRequest(m.config.Method, m.config.URL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	return m.AnalyzeHTTPSEnforcement(resp, m.config.URL)
}

func (m *MisconfigScanner) AnalyzeHTTPSEnforcement(resp *http.Response, targetURL string) *MisconfigResult {
	hstsHeader := resp.Header.Get("Strict-Transport-Security")
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}

	if parsedURL.Scheme == "https" && hstsHeader == "" {
		return &MisconfigResult{
			URL:         targetURL,
			Category:    categoryHeaders,
			Finding:     "HTTPS not properly enforced",
			Evidence:    "HTTPS site missing Strict-Transport-Security header",
			RiskLevel:   "High",
			Remediation: "Add 'Strict-Transport-Security: max-age=31536000; includeSubDomains' header to enforce HTTPS",
		}
	}

	if parsedURL.Scheme == "https" && hstsHeader != "" {
		if !strings.Contains(hstsHeader, "max-age=") {
			return &MisconfigResult{
				URL:         targetURL,
				Category:    categoryHeaders,
				Finding:     "Weak HSTS configuration",
				Evidence:    fmt.Sprintf("HSTS header missing max-age directive: %s", hstsHeader),
				RiskLevel:   "Medium",
				Remediation: "Include max-age directive in HSTS header with appropriate value (e.g., max-age=31536000)",
			}
		}
	}

	return nil
}

func (m *MisconfigScanner) ValidateHeaderValue(headerName, headerValue string) *MisconfigResult {
	for _, secHeader := range RequiredSecurityHeaders {
		if strings.EqualFold(secHeader.Name, headerName) {
			if len(secHeader.ValidValues) > 0 {
				isValid := false
				for _, validValue := range secHeader.ValidValues {
					if strings.EqualFold(headerValue, validValue) ||
						strings.Contains(strings.ToLower(headerValue), strings.ToLower(validValue)) {
						isValid = true
						break
					}
				}

				if !isValid {
					return &MisconfigResult{
						URL:      m.config.URL,
						Category: categoryHeaders,
						Finding:  fmt.Sprintf("Invalid security header value: %s", headerName),
						Evidence: fmt.Sprintf("Header '%s' has invalid value '%s', expected one of: %v",
							headerName, headerValue, secHeader.ValidValues),
						RiskLevel:   "Medium",
						Remediation: fmt.Sprintf("Set '%s' header to a valid value: %v", headerName, secHeader.ValidValues),
					}
				}
			}
			break
		}
	}
	return nil
}

var loginFormPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<form[^>]*action[^>]*login[^>]*>`),
	regexp.MustCompile(`(?i)<form[^>]*>[^<]*<input[^>]*type[^>]*password[^>]*>`),
	regexp.MustCompile(`(?i)<input[^>]*name[^>]*password[^>]*>`),
	regexp.MustCompile(`(?i)<input[^>]*name[^>]*username[^>]*>`),
	regexp.MustCompile(`(?i)<input[^>]*name[^>]*user[^>]*>`),
	regexp.MustCompile(`(?i)<input[^>]*name[^>]*email[^>]*>.*<input[^>]*type[^>]*password[^>]*>`),
}

var versionDisclosurePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)apache/(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)nginx/(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)microsoft-iis/(\d+\.\d+)`),
	regexp.MustCompile(`(?i)php/(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)mysql.*(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)postgresql.*(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)wordpress.*(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)drupal.*(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?i)joomla.*(\d+\.\d+\.\d+)`),
}

var defaultInstallationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)welcome to.*installation`),
	regexp.MustCompile(`(?i)default.*installation.*page`),
	regexp.MustCompile(`(?i)congratulations.*successfully installed`),
	regexp.MustCompile(`(?i)apache.*test page`),
	regexp.MustCompile(`(?i)welcome to nginx`),
	regexp.MustCompile(`(?i)nginx.*welcome`),
	regexp.MustCompile(`(?i)iis.*welcome`),
	regexp.MustCompile(`(?i)xampp.*dashboard`),
	regexp.MustCompile(`(?i)wamp.*server`),
}

func (m *MisconfigScanner) TestDefaultCredentials() []MisconfigResult {
	var results []MisconfigResult
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.config.Threads)
	resultsChan := make(chan MisconfigResult, len(CommonDefaultCredentials)*10)

	loginEndpoints := m.discoverLoginEndpoints()

	for _, endpoint := range loginEndpoints {
		for _, cred := range CommonDefaultCredentials {
			wg.Add(1)
			go func(loginURL string, credential DefaultCredential) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if result := m.testCredential(loginURL, credential); result != nil {
					resultsChan <- *result
				}
			}(endpoint, cred)
		}
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
		m.AddResult(result)
	}

	return results
}

func (m *MisconfigScanner) discoverLoginEndpoints() []string {
	commonLoginPaths := []string{
		"/login",
		"/admin",
		"/admin/login",
		"/administrator",
		"/wp-admin",
		"/wp-login.php",
		"/user/login",
		"/auth/login",
		"/signin",
		"/admin.php",
		"/login.php",
		"/admin/index.php",
		"/manager/html",
		"/phpmyadmin",
	}

	var endpoints []string
	baseURL := strings.TrimSuffix(m.config.URL, "/")

	for _, path := range commonLoginPaths {
		loginURL := baseURL + path
		if m.hasLoginForm(loginURL) {
			endpoints = append(endpoints, loginURL)
		}
	}

	if len(endpoints) == 0 {
		endpoints = append(endpoints, baseURL)
	}

	return endpoints
}

func (m *MisconfigScanner) hasLoginForm(loginURL string) bool {
	req, err := m.createHTTPRequest(httpMethodGET, loginURL, nil)
	if err != nil {
		return false
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return false
	}

	body, err := m.safeReadResponse(resp, loginPageLimit)
	if err != nil {
		return false
	}

	return m.detectLoginForm(string(body))
}

func (m *MisconfigScanner) detectLoginForm(body string) bool {
	for _, pattern := range loginFormPatterns {
		if pattern.MatchString(body) {
			return true
		}
	}
	return false
}

func (m *MisconfigScanner) DetectLoginForm(body string) bool {
	return m.detectLoginForm(body)
}

func (m *MisconfigScanner) testCredential(loginURL string, credential DefaultCredential) *MisconfigResult {
	formData := url.Values{}
	formData.Set("username", credential.Username)
	formData.Set("user", credential.Username)
	formData.Set("email", credential.Username)
	formData.Set("password", credential.Password)
	formData.Set("pass", credential.Password)

	req, err := m.createHTTPRequest(httpMethodPOST, loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	noRedirectClient := &http.Client{
		Timeout:   m.client.Timeout,
		Transport: m.client.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		if handleErr := m.handleHTTPError(err); handleErr != nil {
			m.AddError(handleErr)
		}
		return nil
	}
	defer resp.Body.Close()

	body, err := m.safeReadResponse(resp, directoryLimit)
	if err != nil {
		return nil
	}

	bodyStr := string(body)

	if m.isSuccessfulLogin(resp, bodyStr) {
		return &MisconfigResult{
			URL:      loginURL,
			Category: categoryDefaults,
			Finding:  "Default credentials accepted",
			Evidence: fmt.Sprintf("Login successful with %s:%s for %s",
				credential.Username, credential.Password, credential.Context),
			RiskLevel:   "High",
			Remediation: "Change default credentials immediately and implement strong password policies",
		}
	}

	return nil
}

func (m *MisconfigScanner) isSuccessfulLogin(resp *http.Response, body string) bool {
	successIndicators := []string{
		"dashboard",
		"welcome",
		"logout",
		"profile",
		"admin panel",
		"control panel",
		"administration",
	}

	failureIndicators := []string{
		"invalid",
		"incorrect",
		"failed",
		"error",
		"denied",
		"unauthorized",
		"forbidden",
	}

	bodyLower := strings.ToLower(body)

	for _, indicator := range failureIndicators {
		if strings.Contains(bodyLower, indicator) {
			return false
		}
	}

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther ||
		resp.StatusCode == http.StatusMovedPermanently {
		location := resp.Header.Get("Location")
		locationLower := strings.ToLower(location)
		if strings.Contains(locationLower, "dashboard") ||
			strings.Contains(locationLower, "admin") ||
			strings.Contains(locationLower, "home") ||
			strings.Contains(locationLower, "main") ||
			strings.Contains(locationLower, "index") ||
			location != "" {
			return true
		}
	}

	for _, indicator := range successIndicators {
		if strings.Contains(bodyLower, indicator) {
			return true
		}
	}

	return false
}

func (m *MisconfigScanner) TestDefaultPages() []MisconfigResult {
	var results []MisconfigResult

	defaultPages := []string{
		"/",
		"/index.html",
		"/index.php",
		"/default.html",
		"/default.asp",
		"/welcome.html",
		"/info.php",
		"/phpinfo.php",
		"/server-info",
		"/server-status",
	}

	baseURL := strings.TrimSuffix(m.config.URL, "/")

	for _, page := range defaultPages {
		pageURL := baseURL + page
		if result := m.testDefaultPage(pageURL); result != nil {
			results = append(results, *result)
			m.AddResult(*result)
		}
	}

	return results
}

func (m *MisconfigScanner) testDefaultPage(pageURL string) *MisconfigResult {
	req, err := m.createHTTPRequest(httpMethodGET, pageURL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil
	}

	body, err := m.safeReadResponse(resp, loginPageLimit)
	if err != nil {
		return nil
	}

	bodyStr := string(body)

	if m.detectDefaultInstallation(bodyStr) {
		return &MisconfigResult{
			URL:         pageURL,
			Category:    categoryDefaults,
			Finding:     "Default installation page detected",
			Evidence:    fmt.Sprintf("Default installation page found at %s", pageURL),
			RiskLevel:   "Medium",
			Remediation: "Remove or customize default installation pages to prevent information disclosure",
		}
	}

	if versionResult := m.DetectVersionDisclosure(bodyStr); versionResult != nil {
		versionResult.URL = pageURL
		return versionResult
	}

	return nil
}

func (m *MisconfigScanner) detectDefaultInstallation(body string) bool {
	for _, pattern := range defaultInstallationPatterns {
		if pattern.MatchString(body) {
			return true
		}
	}
	return false
}

func (m *MisconfigScanner) DetectDefaultInstallation(body string) bool {
	return m.detectDefaultInstallation(body)
}

func (m *MisconfigScanner) DetectVersionDisclosure(response string) *MisconfigResult {
	for _, pattern := range versionDisclosurePatterns {
		if matches := pattern.FindStringSubmatch(response); len(matches) > 1 {
			version := matches[1]
			software := strings.Split(matches[0], "/")[0]

			return &MisconfigResult{
				URL:         m.config.URL,
				Category:    categoryDefaults,
				Finding:     "Version information disclosed",
				Evidence:    fmt.Sprintf("Software version disclosed: %s version %s", software, version),
				RiskLevel:   "Low",
				Remediation: "Configure server to hide version information in responses and error pages",
			}
		}
	}
	return nil
}

func (m *MisconfigScanner) TestHTTPMethods() []MisconfigResult {
	var results []MisconfigResult
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.config.Threads)
	resultsChan := make(chan MisconfigResult, len(DangerousHTTPMethods))

	for _, method := range DangerousHTTPMethods {
		wg.Add(1)
		go func(httpMethod HTTPMethod) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if result := m.testHTTPMethod(httpMethod); result != nil {
				resultsChan <- *result
			}
		}(method)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
		m.AddResult(result)
	}

	return results
}

func (m *MisconfigScanner) testHTTPMethod(method HTTPMethod) *MisconfigResult {
	req, err := m.createHTTPRequest(method.Method, m.config.URL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent ||
		resp.StatusCode == http.StatusAccepted {
		riskLevel := riskMedium
		if method.Dangerous {
			riskLevel = riskHigh
		}

		return &MisconfigResult{
			URL:         m.config.URL,
			Category:    categoryServerConfig,
			Finding:     fmt.Sprintf("Dangerous HTTP method enabled: %s", method.Method),
			Evidence:    fmt.Sprintf("HTTP %s method returned status %d", method.Method, resp.StatusCode),
			RiskLevel:   riskLevel,
			Remediation: fmt.Sprintf("Disable %s method if not required. %s", method.Method, method.Description),
		}
	}

	return nil
}

func (m *MisconfigScanner) TestServerBanner() *MisconfigResult {
	req, err := m.createHTTPRequest(m.config.Method, m.config.URL, nil)
	if err != nil {
		return nil
	}

	resp, err := m.makeHTTPRequest(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	return m.analyzeServerBanner(resp)
}

func (m *MisconfigScanner) analyzeServerBanner(resp *http.Response) *MisconfigResult {
	serverHeader := resp.Header.Get("Server")
	if serverHeader == "" {
		return nil
	}

	for _, pattern := range versionDisclosurePatterns {
		if matches := pattern.FindStringSubmatch(serverHeader); len(matches) > 1 {
			version := matches[1]
			software := strings.Split(matches[0], "/")[0]

			return &MisconfigResult{
				URL:         m.config.URL,
				Category:    categoryServerConfig,
				Finding:     "Server version disclosed in banner",
				Evidence:    fmt.Sprintf("Server header reveals: %s version %s", software, version),
				RiskLevel:   "Low",
				Remediation: "Configure server to hide version information in Server header",
			}
		}
	}

	if strings.Contains(strings.ToLower(serverHeader), "apache") ||
		strings.Contains(strings.ToLower(serverHeader), "nginx") ||
		strings.Contains(strings.ToLower(serverHeader), "iis") ||
		strings.Contains(strings.ToLower(serverHeader), "microsoft") {
		return &MisconfigResult{
			URL:         m.config.URL,
			Category:    categoryServerConfig,
			Finding:     "Server software disclosed in banner",
			Evidence:    fmt.Sprintf("Server header reveals software: %s", serverHeader),
			RiskLevel:   "Low",
			Remediation: "Configure server to use generic Server header or remove it entirely",
		}
	}

	return nil
}

func (m *MisconfigScanner) TestErrorMessages() []MisconfigResult {
	var results []MisconfigResult

	errorPaths := []string{
		"/nonexistent-page-12345",
		"/admin/secret",
		"/database/config",
		"/api/v1/nonexistent",
		"/../../../etc/passwd",
		"/wp-admin/nonexistent",
		"/phpmyadmin/nonexistent",
	}

	for _, path := range errorPaths {
		if result := m.testErrorMessage(path); result != nil {
			results = append(results, *result)
			m.AddResult(*result)
		}
	}

	return results
}

func (m *MisconfigScanner) testErrorMessage(path string) *MisconfigResult {
	targetURL := strings.TrimSuffix(m.config.URL, "/") + path

	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, httpMethodGET, targetURL, nil)
	if err != nil {
		return nil
	}

	for _, header := range m.config.Headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 600 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}

		bodyStr := string(body)
		return m.analyzeErrorMessage(targetURL, resp.StatusCode, bodyStr)
	}

	return nil
}

var informationLeakagePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)stack trace|stacktrace`),
	regexp.MustCompile(`(?i)exception|error.*line \d+`),
	regexp.MustCompile(`(?i)mysql.*error|postgresql.*error|oracle.*error`),
	regexp.MustCompile(`(?i)warning.*include.*failed to open stream`),
	regexp.MustCompile(`(?i)fatal error.*in.*on line \d+`),
	regexp.MustCompile(`(?i)debug.*information|debug.*mode`),
	regexp.MustCompile(`(?i)internal server error.*details`),
	regexp.MustCompile(`(?i)application.*error.*details`),
	regexp.MustCompile(`(?i)file not found.*path.*[/\\]`),
	regexp.MustCompile(`(?i)access denied.*path.*[/\\]`),
}

func (m *MisconfigScanner) analyzeErrorMessage(url string, statusCode int, body string) *MisconfigResult {
	for _, pattern := range informationLeakagePatterns {
		if pattern.MatchString(body) {
			evidence := fmt.Sprintf("Error page at %s (Status: %d) contains sensitive information", url, statusCode)
			if len(body) > serverInfoLimit {
				evidence += fmt.Sprintf(", Content preview: %s...", body[:serverInfoLimit])
			} else {
				evidence += fmt.Sprintf(", Content: %s", body)
			}

			return &MisconfigResult{
				URL:         url,
				Category:    categoryServerConfig,
				Finding:     "Information leakage in error messages",
				Evidence:    evidence,
				RiskLevel:   "Medium",
				Remediation: "Configure custom error pages that don't reveal system information",
			}
		}
	}

	if versionResult := m.DetectVersionDisclosure(body); versionResult != nil {
		versionResult.URL = url
		versionResult.Category = categoryServerConfig
		versionResult.Finding = "Version information disclosed in error page"
		versionResult.Remediation = "Configure custom error pages to hide version information"
		return versionResult
	}

	return nil
}

func (m *MisconfigScanner) TestInsecureRedirects() []MisconfigResult {
	var results []MisconfigResult

	redirectTests := []struct {
		path        string
		description string
	}{
		{"/redirect?url=http://evil.com", "Open redirect via url parameter"},
		{"/redirect?redirect=http://evil.com", "Open redirect via redirect parameter"},
		{"/redirect?return=http://evil.com", "Open redirect via return parameter"},
		{"/redirect?next=http://evil.com", "Open redirect via next parameter"},
		{"/login?return_to=http://evil.com", "Open redirect in login form"},
		{"/logout?redirect_uri=http://evil.com", "Open redirect in logout"},
	}

	for _, test := range redirectTests {
		if result := m.testInsecureRedirect(test.path, test.description); result != nil {
			results = append(results, *result)
			m.AddResult(*result)
		}
	}

	return results
}

func (m *MisconfigScanner) testInsecureRedirect(path, description string) *MisconfigResult {
	targetURL := strings.TrimSuffix(m.config.URL, "/") + path

	noRedirectClient := &http.Client{
		Timeout:   m.client.Timeout,
		Transport: m.client.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := m.createHTTPRequest(httpMethodGET, targetURL, nil)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		if handleErr := m.handleHTTPError(err); handleErr != nil {
			m.AddError(handleErr)
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" && (strings.HasPrefix(location, "http://evil.com") || strings.Contains(location, "evil.com")) {
			return &MisconfigResult{
				URL:         targetURL,
				Category:    categoryServerConfig,
				Finding:     "Insecure redirect configuration detected",
				Evidence:    fmt.Sprintf("%s - redirects to: %s", description, location),
				RiskLevel:   "Medium",
				Remediation: "Implement redirect validation to only allow trusted domains",
			}
		}
	}

	return nil
}
