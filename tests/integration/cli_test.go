//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"vn/tests/shared"
	"vn/tests/shared/testserver"
)

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	serverPool := getSharedServerPool()
	config := testserver.ServerConfig{
		Handler:    createVulnerableTestHandler(),
		ConfigName: "vulnerable-test-server",
		Timeout:    10 * time.Second,
	}

	server, err := serverPool.GetServer(config)
	if err != nil {
		t.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	shared.WaitForServerReady(t, server)

	timeouts := shared.GetOptimizedTimeouts()
	ctx, cancel := context.WithTimeout(context.Background(), timeouts.HealthCheck)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/health", nil)
	client := server.Client()
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("Test server not available: %v", err)
	}
	resp.Body.Close()

	testCases := []struct {
		name        string
		args        []string
		expectError bool
		expectVulns bool
	}{
		{
			name:        "SQL injection scan with vulnerabilities",
			args:        []string{"sqli", server.URL + "/?id=1"},
			expectError: false,
			expectVulns: true,
		},
		{
			name: "SQL injection POST scan",
			args: []string{"sqli", server.URL + "/login", "--method", "POST",
				"--data", "username=admin&password=secret"},
			expectError: false,
			expectVulns: true,
		},
		{
			name:        "SQL injection with specific params",
			args:        []string{"sqli", server.URL + "/", "--params", "id,username"},
			expectError: false,
			expectVulns: true,
		},
		{
			name:        "XSS scan",
			args:        []string{"xss", server.URL + "/?q=test"},
			expectError: false,
			expectVulns: false,
		},
		{
			name:        "Misconfiguration scan",
			args:        []string{"misconfig", server.URL + "/"},
			expectError: false,
			expectVulns: false,
		},
		{
			name:        "Misconfiguration scan with specific tests",
			args:        []string{"misconfig", server.URL + "/", "--tests", "files,headers"},
			expectError: false,
			expectVulns: false,
		},
		{
			name:        "Misconfiguration scan with custom headers",
			args:        []string{"misconfig", server.URL + "/", "--headers", "User-Agent: VN-Scanner"},
			expectError: false,
			expectVulns: false,
		},
		{
			name:        "Misconfiguration scan with threading",
			args:        []string{"misconfig", server.URL + "/", "--threads", "3", "--timeout", "5"},
			expectError: false,
			expectVulns: false,
		},
		{
			name:        "Invalid URL",
			args:        []string{"sqli", "invalid-url"},
			expectError: false,
			expectVulns: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()

			if tc.expectError && err == nil {
				t.Error("Expected command to fail, but it succeeded")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected command to succeed, but it failed: %v", err)
			}

			outputStr := string(output)

			if tc.expectVulns {
				if !strings.Contains(outputStr, "potential") && !strings.Contains(outputStr, "vulnerabilities") {
					t.Errorf("Expected to find vulnerabilities, but output was: %s", outputStr)
				}
			}

			if strings.Contains(outputStr, "panic") {
				t.Errorf("Command panicked: %s", outputStr)
			}
		})
	}
}

func TestCLIHelp(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	testCases := []struct {
		name           string
		args           []string
		expectedOutput []string
	}{
		{
			name: "Root help",
			args: []string{"--help"},
			expectedOutput: []string{
				"VN - Vulnerability Navigator",
				"OWASP Top 10",
				"sqli",
				"xss",
				"misconfig",
			},
		},
		{
			name: "SQL injection help",
			args: []string{"sqli", "--help"},
			expectedOutput: []string{
				"SQL injection vulnerabilities",
				"--method",
				"--data",
				"--params",
			},
		},
		{
			name: "XSS help",
			args: []string{"xss", "--help"},
			expectedOutput: []string{
				"XSS vulnerabilities",
				"--method",
				"--data",
				"--params",
			},
		},
		{
			name: "Misconfiguration help",
			args: []string{"misconfig", "--help"},
			expectedOutput: []string{
				"security misconfigurations",
				"--method",
				"--headers",
				"--tests",
				"--threads",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Errorf("Help command failed: %v", err)
			}

			outputStr := string(output)

			for _, expected := range tc.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected help output to contain '%s', but it didn't.\nOutput: %s", expected, outputStr)
				}
			}
		})
	}
}

func TestCLIBasicFunctionality(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	cmd := exec.Command(binaryPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "panic") {
			t.Errorf("CLI panicked: %s", outputStr)
		}
	}
}

func BenchmarkCLIPerformance(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		b.Fatalf("Failed to build CLI: %v", err)
	}

	serverPool := getSharedServerPool()
	config := testserver.ServerConfig{
		Handler:    createVulnerableTestHandler(),
		ConfigName: "vulnerable-test-server-bench",
		Timeout:    10 * time.Second,
	}

	server, err := serverPool.GetServer(config)
	if err != nil {
		b.Fatalf("Failed to get test server: %v", err)
	}
	defer serverPool.ReleaseServer(server)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "sqli", server.URL+"/?id=1", "--threads", "2")
		if err := cmd.Run(); err != nil {
			b.Errorf("Benchmark iteration %d failed: %v", i, err)
		}
	}
}
