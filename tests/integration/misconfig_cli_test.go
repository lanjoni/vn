//go:build integration
// +build integration

package integration

import (
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"vn/tests/shared/fixtures"
)

var (
	sharedMockServer *httptest.Server
	mockServerOnce   sync.Once
)

func getSharedMockServer() *httptest.Server {
	mockServerOnce.Do(func() {
		provider := fixtures.NewMockProvider()
		sharedMockServer = provider.CreateHTTPBinMock()
	})
	return sharedMockServer
}

func TestMisconfigCLIFlags(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	mockServer := getSharedMockServer()

	testCases := []struct {
		name           string
		args           []string
		expectError    bool
		expectedOutput []string
	}{
		{
			name: "Basic misconfig scan",
			args: []string{"misconfig", mockServer.URL, "--timeout", "5"},
			expectedOutput: []string{
				"Starting Security Misconfiguration scan",
				"Method: GET",
				"Timeout: 5s",
			},
		},
		{
			name: "Misconfig with specific tests",
			args: []string{"misconfig", mockServer.URL, "--tests", "files", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [files]",
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "Misconfig with multiple test categories",
			args: []string{"misconfig", mockServer.URL, "--tests", "files,headers", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [files headers]",
			},
		},
		{
			name: "Misconfig with custom headers",
			args: []string{"misconfig", mockServer.URL, "--headers", "User-Agent: VN-Test", "--timeout", "5"},
			expectedOutput: []string{
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "Misconfig with threading options",
			args: []string{"misconfig", mockServer.URL, "--threads", "2", "--timeout", "3"},
			expectedOutput: []string{
				"Threads: 2",
				"Timeout: 3s",
			},
		},
		{
			name: "Misconfig with POST method",
			args: []string{"misconfig", mockServer.URL, "--method", "POST", "--timeout", "5"},
			expectedOutput: []string{
				"Method: POST",
			},
		},
		{
			name:        "Missing URL argument",
			args:        []string{"misconfig"},
			expectError: true,
		},
		{
			name:        "Invalid URL",
			args:        []string{"misconfig", "not-a-url"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tc.expectError && err == nil {
				t.Errorf("Expected command to fail, but it succeeded. Output: %s", outputStr)
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected command to succeed, but it failed: %v. Output: %s", err, outputStr)
			}

			for _, expected := range tc.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, outputStr)
				}
			}

			if strings.Contains(outputStr, "panic") {
				t.Errorf("Command panicked: %s", outputStr)
			}
		})
	}
}

func TestMisconfigCLIResultDisplay(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	mockServer := getSharedMockServer()

	cmd := exec.Command(binaryPath, "misconfig", mockServer.URL,
		"--tests", "files", "--timeout", "10")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Errorf("Command failed: %v. Output: %s", err, outputStr)
	}

	expectedElements := []string{
		"Security Misconfiguration scan",
		"Test Categories:",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, outputStr)
		}
	}

	if strings.Contains(outputStr, "Security Misconfiguration Detected") {
		resultElements := []string{
			"URL:",
			"Category:",
			"Finding:",
			"Evidence:",
			"Risk Level:",
			"Remediation:",
		}

		for _, element := range resultElements {
			if !strings.Contains(outputStr, element) {
				t.Errorf("Expected result output to contain '%s', but it didn't.\nOutput: %s", element, outputStr)
			}
		}
	}
}

func TestMisconfigCLIHelp(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	cmd := exec.Command(binaryPath, "misconfig", "--help")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Errorf("Help command failed: %v", err)
	}

	expectedHelpElements := []string{
		"Test endpoints for security misconfigurations",
		"OWASP Top 10 A05:2021",
		"--method",
		"--headers",
		"--timeout",
		"--threads",
		"--tests",
		"files,headers,defaults,server",
		"Examples:",
		"vn misconfig https://example.com",
	}

	for _, expected := range expectedHelpElements {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected help output to contain '%s', but it didn't.\nOutput: %s", expected, outputStr)
		}
	}
}

func TestMisconfigCLIErrorHandling(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	mockServer := getSharedMockServer()

	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "Invalid timeout",
			args: []string{"misconfig", mockServer.URL, "--timeout", "invalid"},
		},
		{
			name: "Invalid threads",
			args: []string{"misconfig", mockServer.URL, "--threads", "invalid"},
		},
		{
			name: "Unreachable host",
			args: []string{"misconfig", "https://nonexistent-host-12345.com", "--timeout", "2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if strings.Contains(outputStr, "panic") {
				t.Errorf("Command panicked: %s", outputStr)
			}

			if tc.name == "Unreachable host" && err == nil {
				if !strings.Contains(outputStr, "No security misconfigurations found") {
					t.Logf("Expected no results for unreachable host, got: %s", outputStr)
				}
			}
		})
	}
}

func TestMisconfigCLIAdvancedFeatures(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	mockServer := getSharedMockServer()

	testCases := []struct {
		name           string
		args           []string
		expectedOutput []string
	}{
		{
			name: "High thread count",
			args: []string{"misconfig", mockServer.URL, "--threads", "20", "--timeout", "5"},
			expectedOutput: []string{
				"Threads: 20",
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "Very short timeout",
			args: []string{"misconfig", mockServer.URL, "--timeout", "1"},
			expectedOutput: []string{
				"Timeout: 1s",
			},
		},
		{
			name: "Multiple custom headers",
			args: []string{"misconfig", mockServer.URL, "--headers", "User-Agent: VN-Test",
				"--headers", "Authorization: Bearer token", "--timeout", "5"},
			expectedOutput: []string{
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "All test categories explicitly",
			args: []string{"misconfig", mockServer.URL, "--tests", "files,headers,defaults,server", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [files headers defaults server]",
			},
		},
		{
			name: "Single test category",
			args: []string{"misconfig", mockServer.URL, "--tests", "headers", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [headers]",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if err != nil {
				t.Errorf("Command failed: %v. Output: %s", err, outputStr)
			}

			for _, expected := range tc.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, outputStr)
				}
			}

			if strings.Contains(outputStr, "panic") {
				t.Errorf("Command panicked: %s", outputStr)
			}
		})
	}
}

func TestMisconfigCLIOutputFormatting(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	mockServer := getSharedMockServer()

	cmd := exec.Command(binaryPath, "misconfig", mockServer.URL,
		"--tests", "headers", "--timeout", "10")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Errorf("Command failed: %v. Output: %s", err, outputStr)
	}

	expectedFormatElements := []string{
		"🔍 Starting Security Misconfiguration scan",
		"⚙️  Method:",
		"🎯 Test Categories:",
	}

	for _, element := range expectedFormatElements {
		if !strings.Contains(outputStr, element) {
			t.Errorf("Expected output formatting to contain '%s', but it didn't.\nOutput: %s", element, outputStr)
		}
	}

	if strings.Contains(outputStr, "Security Misconfiguration Detected") {
		resultFormatElements := []string{
			"[1] Security Misconfiguration Detected",
			"URL:",
			"Category:",
			"Finding:",
			"Evidence:",
			"Risk Level:",
			"Remediation:",
		}

		for _, element := range resultFormatElements {
			if !strings.Contains(outputStr, element) {
				t.Errorf("Expected result formatting to contain '%s', but it didn't.\nOutput: %s", element, outputStr)
			}
		}
	}
}

func TestMisconfigCLIEdgeCases(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	mockServer := getSharedMockServer()

	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "Zero timeout",
			args: []string{"misconfig", mockServer.URL, "--timeout", "0"},
		},
		{
			name: "Zero threads",
			args: []string{"misconfig", mockServer.URL, "--threads", "0"},
		},
		{
			name: "Invalid test category",
			args: []string{"misconfig", mockServer.URL, "--tests", "invalid"},
		},
		{
			name: "Empty headers",
			args: []string{"misconfig", mockServer.URL, "--headers", ""},
		},
		{
			name: "Malformed header",
			args: []string{"misconfig", mockServer.URL, "--headers", "InvalidHeader"},
		},
		{
			name: "Very long URL",
			args: []string{"misconfig", mockServer.URL + "/" + strings.Repeat("a", 1000)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tc.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if strings.Contains(outputStr, "panic") {
				t.Errorf("Command panicked: %s", outputStr)
			}

			if tc.expectError && err == nil {
				t.Errorf("Expected command to fail for edge case '%s', but it succeeded", tc.name)
			}

			if !strings.Contains(outputStr, "Security Misconfiguration scan") &&
				!strings.Contains(outputStr, "Error") &&
				!strings.Contains(outputStr, "Usage:") {
				t.Errorf("Expected some meaningful output for edge case '%s', got: %s", tc.name, outputStr)
			}
		})
	}
}

func TestMisconfigCLITimeoutHandling(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	provider := fixtures.NewMockProvider()
	provider.SimulateNetworkDelay(3000)
	delayedServer := provider.CreateHTTPBinMock()
	defer delayedServer.Close()

	cmd := exec.Command(binaryPath, "misconfig", delayedServer.URL, "--timeout", "1")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if strings.Contains(outputStr, "panic") {
		t.Errorf("Command panicked during timeout test: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Starting Security Misconfiguration scan") {
		t.Errorf("Expected scan to start even with timeout, got: %s", outputStr)
	}
}

func TestMisconfigCLIErrorSimulation(t *testing.T) {
	t.Parallel()
	binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
	if err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}

	provider := fixtures.NewMockProvider()
	provider.SimulateNetworkError("internal_error")
	errorServer := provider.CreateHTTPBinMock()
	defer errorServer.Close()

	cmd := exec.Command(binaryPath, "misconfig", errorServer.URL, "--timeout", "5")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if strings.Contains(outputStr, "panic") {
		t.Errorf("Command panicked during error simulation: %s", outputStr)
	}

	if !strings.Contains(outputStr, "Starting Security Misconfiguration scan") {
		t.Errorf("Expected scan to start even with server errors, got: %s", outputStr)
	}
}
