package e2e

import (
	"strings"
	"testing"
	"time"

	"vn/internal/scanner"
)

func TestMisconfigScanner_E2E_TestServer(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	// Skip if test server is not running
	config := scanner.MisconfigConfig{
		URL:     baseURL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 5,
		Tests:   []string{"files"},
	}
	
	testScanner := scanner.NewMisconfigScanner(config)
	
	// Quick connectivity test
	req, err := testScanner.GetClient().Get(baseURL + "/health")
	if err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}
	req.Body.Close()

	testCases := []struct {
		name          string
		tests         []string
		expectedCount int
		expectedFindings []string
	}{
		{
			name:          "Sensitive files detection",
			tests:         []string{"files"},
			expectedCount: 5, // .env, config.php, backup.sql, robots.txt, config.bak
			expectedFindings: []string{
				"Sensitive file exposed: Environment configuration file",
				"Sensitive file exposed: PHP configuration file",
				"Backup file exposed",
			},
		},
		{
			name:          "Security headers analysis",
			tests:         []string{"headers"},
			expectedCount: 3, // Missing X-Frame-Options, X-Content-Type-Options, Strict-Transport-Security
			expectedFindings: []string{
				"Missing security header: X-Frame-Options",
				"Missing security header: X-Content-Type-Options",
				"Missing security header: Strict-Transport-Security",
			},
		},
		{
			name:          "Default credentials testing",
			tests:         []string{"defaults"},
			expectedCount: 8, // Default credentials accepted for admin/admin, root/root, etc.
			expectedFindings: []string{
				"Default credentials accepted",
			},
		},
		{
			name:          "Server configuration testing",
			tests:         []string{"server"},
			expectedCount: 4, // PUT, DELETE, TRACE methods enabled
			expectedFindings: []string{
				"Dangerous HTTP method enabled: PUT",
				"Dangerous HTTP method enabled: DELETE",
				"Dangerous HTTP method enabled: TRACE",
			},
		},
		{
			name:          "Full comprehensive scan",
			tests:         []string{"files", "headers", "defaults", "server"},
			expectedCount: 15, // Sum of all individual test results
			expectedFindings: []string{
				"Sensitive file exposed",
				"Missing security header",
				"Default credentials accepted",
				"Dangerous HTTP method enabled",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := scanner.MisconfigConfig{
				URL:     baseURL,
				Method:  "GET",
				Headers: []string{},
				Timeout: 15 * time.Second,
				Threads: 5,
				Tests:   tc.tests,
			}

			misconfigScanner := scanner.NewMisconfigScanner(config)
			results := misconfigScanner.Scan()

			if len(results) < tc.expectedCount {
				t.Errorf("Expected at least %d results, got %d", tc.expectedCount, len(results))
			}

			// Verify expected findings are present
			for _, expectedFinding := range tc.expectedFindings {
				found := false
				for _, result := range results {
					if strings.Contains(result.Finding, expectedFinding) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected finding containing '%s' not found", expectedFinding)
				}
			}

			// Verify all results have required fields
			for i, result := range results {
				if result.URL == "" {
					t.Errorf("Result %d missing URL", i)
				}
				if result.Category == "" {
					t.Errorf("Result %d missing Category", i)
				}
				if result.Finding == "" {
					t.Errorf("Result %d missing Finding", i)
				}
				if result.RiskLevel == "" {
					t.Errorf("Result %d missing RiskLevel", i)
				}
				if result.Remediation == "" {
					t.Errorf("Result %d missing Remediation", i)
				}
			}

			// Check for errors
			errors := misconfigScanner.GetErrors()
			if len(errors) > 0 {
				t.Logf("Scanner errors (may be expected): %v", errors)
			}
		})
	}
}

func TestMisconfigScanner_E2E_SensitiveFilesDetailed(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	config := scanner.MisconfigConfig{
		URL:     baseURL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 3,
		Tests:   []string{"files"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Test connectivity first
	if _, err := misconfigScanner.GetClient().Get(baseURL + "/health"); err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestSensitiveFiles()

	expectedFiles := map[string]struct {
		riskLevel   string
		description string
	}{
		"/.env": {
			riskLevel:   "High",
			description: "Environment configuration file",
		},
		"/config.php": {
			riskLevel:   "High", 
			description: "PHP configuration file",
		},
		"/backup.sql": {
			riskLevel:   "High",
			description: "Database backup file",
		},
		"/robots.txt": {
			riskLevel:   "Low",
			description: "Robots exclusion file",
		},
	}

	foundFiles := make(map[string]bool)
	for _, result := range results {
		for expectedPath, expected := range expectedFiles {
			if strings.Contains(result.URL, expectedPath) {
				foundFiles[expectedPath] = true
				
				if result.RiskLevel != expected.riskLevel {
					t.Errorf("File %s: expected risk level %s, got %s", 
						expectedPath, expected.riskLevel, result.RiskLevel)
				}
				
				if !strings.Contains(result.Finding, expected.description) {
					t.Errorf("File %s: expected finding to contain '%s', got '%s'", 
						expectedPath, expected.description, result.Finding)
				}
				
				if result.Category != "sensitive-files" {
					t.Errorf("File %s: expected category 'sensitive-files', got '%s'", 
						expectedPath, result.Category)
				}
				
				if result.Evidence == "" {
					t.Errorf("File %s: missing evidence", expectedPath)
				}
			}
		}
	}

	for expectedPath := range expectedFiles {
		if !foundFiles[expectedPath] {
			t.Errorf("Expected to find sensitive file %s", expectedPath)
		}
	}
}

func TestMisconfigScanner_E2E_SecurityHeaders(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	config := scanner.MisconfigConfig{
		URL:     baseURL + "/insecure-headers",
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 1,
		Tests:   []string{"headers"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Test connectivity first
	if _, err := misconfigScanner.GetClient().Get(baseURL + "/health"); err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestSecurityHeaders()

	expectedMissingHeaders := []string{
		"X-Frame-Options",
		"X-Content-Type-Options", 
		"Strict-Transport-Security",
	}

	foundHeaders := make(map[string]bool)
	for _, result := range results {
		for _, expectedHeader := range expectedMissingHeaders {
			if strings.Contains(result.Finding, expectedHeader) {
				foundHeaders[expectedHeader] = true
				
				if result.Category != "headers" {
					t.Errorf("Header %s: expected category 'headers', got '%s'", 
						expectedHeader, result.Category)
				}
				
				if !strings.Contains(result.Finding, "Missing security header") {
					t.Errorf("Header %s: expected finding to contain 'Missing security header', got '%s'", 
						expectedHeader, result.Finding)
				}
			}
		}
	}

	for _, expectedHeader := range expectedMissingHeaders {
		if !foundHeaders[expectedHeader] {
			t.Errorf("Expected to find missing header %s", expectedHeader)
		}
	}
}

func TestMisconfigScanner_E2E_DefaultCredentials(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	config := scanner.MisconfigConfig{
		URL:     baseURL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 15 * time.Second,
		Threads: 3,
		Tests:   []string{"defaults"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Test connectivity first
	if _, err := misconfigScanner.GetClient().Get(baseURL + "/health"); err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestDefaultCredentials()

	if len(results) == 0 {
		t.Error("Expected to find default credentials vulnerabilities")
		return
	}

	// Verify that default credentials were accepted
	foundDefaultCreds := false
	for _, result := range results {
		if strings.Contains(result.Finding, "Default credentials accepted") {
			foundDefaultCreds = true
			
			if result.Category != "defaults" {
				t.Errorf("Expected category 'defaults', got '%s'", result.Category)
			}
			
			if result.RiskLevel != "High" {
				t.Errorf("Expected risk level 'High', got '%s'", result.RiskLevel)
			}
			
			if !strings.Contains(result.Evidence, "admin") {
				t.Errorf("Expected evidence to contain 'admin', got '%s'", result.Evidence)
			}
		}
	}

	if !foundDefaultCreds {
		t.Error("Expected to find default credentials acceptance")
	}
}

func TestMisconfigScanner_E2E_DangerousHTTPMethods(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	config := scanner.MisconfigConfig{
		URL:     baseURL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 2,
		Tests:   []string{"server"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Test connectivity first
	if _, err := misconfigScanner.GetClient().Get(baseURL + "/health"); err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestHTTPMethods()

	expectedMethods := []string{"PUT", "DELETE", "TRACE"}
	foundMethods := make(map[string]bool)

	for _, result := range results {
		for _, expectedMethod := range expectedMethods {
			if strings.Contains(result.Finding, expectedMethod) {
				foundMethods[expectedMethod] = true
				
				if result.Category != "server-config" {
					t.Errorf("Method %s: expected category 'server-config', got '%s'", 
						expectedMethod, result.Category)
				}
				
				if result.RiskLevel != "High" {
					t.Errorf("Method %s: expected risk level 'High', got '%s'", 
						expectedMethod, result.RiskLevel)
				}
			}
		}
	}

	for _, expectedMethod := range expectedMethods {
		if !foundMethods[expectedMethod] {
			t.Errorf("Expected to find dangerous method %s", expectedMethod)
		}
	}
}

func TestMisconfigScanner_E2E_DirectoryListing(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	config := scanner.MisconfigConfig{
		URL:     baseURL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 1,
		Tests:   []string{"files"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Test connectivity first
	if _, err := misconfigScanner.GetClient().Get(baseURL + "/health"); err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}

	result := misconfigScanner.TestDirectoryListing("/uploads/")

	if result == nil {
		t.Error("Expected to find directory listing vulnerability")
		return
	}

	if result.Category != "sensitive-files" {
		t.Errorf("Expected category 'sensitive-files', got '%s'", result.Category)
	}

	if result.Finding != "Directory listing enabled" {
		t.Errorf("Expected finding 'Directory listing enabled', got '%s'", result.Finding)
	}

	if result.RiskLevel != "Medium" {
		t.Errorf("Expected risk level 'Medium', got '%s'", result.RiskLevel)
	}

	if !strings.Contains(result.Evidence, "/uploads") {
		t.Errorf("Expected evidence to contain '/uploads', got '%s'", result.Evidence)
	}
}

func TestMisconfigScanner_E2E_BackupFiles(t *testing.T) {
	baseURL := "http://localhost:8080"
	
	config := scanner.MisconfigConfig{
		URL:     baseURL,
		Method:  "GET",
		Headers: []string{},
		Timeout: 10 * time.Second,
		Threads: 3,
		Tests:   []string{"files"},
	}

	misconfigScanner := scanner.NewMisconfigScanner(config)
	
	// Test connectivity first
	if _, err := misconfigScanner.GetClient().Get(baseURL + "/health"); err != nil {
		t.Skip("Test server not running, skipping E2E tests")
		return
	}

	results := misconfigScanner.TestBackupFiles()

	expectedBackupFiles := []string{
		"config.bak",
		"app.config.old", 
		"database.sql.backup",
	}

	foundFiles := make(map[string]bool)
	for _, result := range results {
		for _, expectedFile := range expectedBackupFiles {
			if strings.Contains(result.URL, expectedFile) {
				foundFiles[expectedFile] = true
				
				if result.Category != "sensitive-files" {
					t.Errorf("Backup file %s: expected category 'sensitive-files', got '%s'", 
						expectedFile, result.Category)
				}
				
				if result.Finding != "Backup file exposed" {
					t.Errorf("Backup file %s: expected finding 'Backup file exposed', got '%s'", 
						expectedFile, result.Finding)
				}
				
				if result.RiskLevel != "High" {
					t.Errorf("Backup file %s: expected risk level 'High', got '%s'", 
						expectedFile, result.RiskLevel)
				}
			}
		}
	}

	for _, expectedFile := range expectedBackupFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected to find backup file %s", expectedFile)
		}
	}
}