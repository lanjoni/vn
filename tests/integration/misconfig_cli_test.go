package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMisconfigCLIFlags(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-test")

	testCases := []struct {
		name           string
		args           []string
		expectError    bool
		expectedOutput []string
	}{
		{
			name: "Basic misconfig scan",
			args: []string{"misconfig", "https://httpbin.org", "--timeout", "5"},
			expectedOutput: []string{
				"Starting Security Misconfiguration scan",
				"Method: GET",
				"Timeout: 5s",
			},
		},
		{
			name: "Misconfig with specific tests",
			args: []string{"misconfig", "https://httpbin.org", "--tests", "files", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [files]",
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "Misconfig with multiple test categories",
			args: []string{"misconfig", "https://httpbin.org", "--tests", "files,headers", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [files headers]",
			},
		},
		{
			name: "Misconfig with custom headers",
			args: []string{"misconfig", "https://httpbin.org", "--headers", "User-Agent: VN-Test", "--timeout", "5"},
			expectedOutput: []string{
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "Misconfig with threading options",
			args: []string{"misconfig", "https://httpbin.org", "--threads", "2", "--timeout", "3"},
			expectedOutput: []string{
				"Threads: 2",
				"Timeout: 3s",
			},
		},
		{
			name: "Misconfig with POST method",
			args: []string{"misconfig", "https://httpbin.org", "--method", "POST", "--timeout", "5"},
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
			cmd := exec.Command("./vn-misconfig-test", tc.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tc.expectError && err == nil {
				t.Errorf("Expected command to fail, but it succeeded. Output: %s", outputStr)
			}

			if !tc.expectError && err != nil && !strings.Contains(outputStr, "no such host") {
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
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-display-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-display-test")

	cmd := exec.Command("./vn-misconfig-display-test", "misconfig", "https://httpbin.org", "--tests", "files", "--timeout", "10")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil && !strings.Contains(outputStr, "no such host") {
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
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-help-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-help-test")

	cmd := exec.Command("./vn-misconfig-help-test", "misconfig", "--help")
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
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-error-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-error-test")

	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "Invalid timeout",
			args: []string{"misconfig", "https://httpbin.org", "--timeout", "invalid"},
		},
		{
			name: "Invalid threads",
			args: []string{"misconfig", "https://httpbin.org", "--threads", "invalid"},
		},
		{
			name: "Unreachable host",
			args: []string{"misconfig", "https://nonexistent-host-12345.com", "--timeout", "2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("./vn-misconfig-error-test", tc.args...)
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
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-advanced-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-advanced-test")

	testCases := []struct {
		name           string
		args           []string
		expectedOutput []string
	}{
		{
			name: "High thread count",
			args: []string{"misconfig", "https://httpbin.org", "--threads", "20", "--timeout", "5"},
			expectedOutput: []string{
				"Threads: 20",
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "Very short timeout",
			args: []string{"misconfig", "https://httpbin.org", "--timeout", "1"},
			expectedOutput: []string{
				"Timeout: 1s",
			},
		},
		{
			name: "Multiple custom headers",
			args: []string{"misconfig", "https://httpbin.org", "--headers", "User-Agent: VN-Test", "--headers", "Authorization: Bearer token", "--timeout", "5"},
			expectedOutput: []string{
				"Starting Security Misconfiguration scan",
			},
		},
		{
			name: "All test categories explicitly",
			args: []string{"misconfig", "https://httpbin.org", "--tests", "files,headers,defaults,server", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [files headers defaults server]",
			},
		},
		{
			name: "Single test category",
			args: []string{"misconfig", "https://httpbin.org", "--tests", "headers", "--timeout", "5"},
			expectedOutput: []string{
				"Test Categories: [headers]",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("./vn-misconfig-advanced-test", tc.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if err != nil && !strings.Contains(outputStr, "no such host") {
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
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-format-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-format-test")

	cmd := exec.Command("./vn-misconfig-format-test", "misconfig", "https://httpbin.org", "--tests", "headers", "--timeout", "10")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil && !strings.Contains(outputStr, "no such host") {
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
	buildCmd := exec.Command("go", "build", "-o", "vn-misconfig-edge-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-misconfig-edge-test")

	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "Zero timeout",
			args: []string{"misconfig", "https://httpbin.org", "--timeout", "0"},
		},
		{
			name: "Zero threads",
			args: []string{"misconfig", "https://httpbin.org", "--threads", "0"},
		},
		{
			name: "Invalid test category",
			args: []string{"misconfig", "https://httpbin.org", "--tests", "invalid"},
		},
		{
			name: "Empty headers",
			args: []string{"misconfig", "https://httpbin.org", "--headers", ""},
		},
		{
			name: "Malformed header",
			args: []string{"misconfig", "https://httpbin.org", "--headers", "InvalidHeader"},
		},
		{
			name: "Very long URL",
			args: []string{"misconfig", "https://httpbin.org/" + strings.Repeat("a", 1000)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("./vn-misconfig-edge-test", tc.args...)
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
