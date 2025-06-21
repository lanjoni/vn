package integration

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildCmd := exec.Command("go", "build", "-o", "vn-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-test")

	serverCmd := exec.Command("go", "run", "../../test-server/main.go")
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/health", nil)
	client := &http.Client{}
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
			args:        []string{"sqli", "http://localhost:8080/?id=1"},
			expectError: false,
			expectVulns: true,
		},
		{
			name: "SQL injection POST scan",
			args: []string{"sqli", "http://localhost:8080/login", "--method", "POST",
				"--data", "username=admin&password=secret"},
			expectError: false,
			expectVulns: true,
		},
		{
			name:        "SQL injection with specific params",
			args:        []string{"sqli", "http://localhost:8080/", "--params", "id,username"},
			expectError: false,
			expectVulns: true,
		},
		{
			name:        "XSS scan",
			args:        []string{"xss", "http://localhost:8080/?q=test"},
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
			cmd := exec.Command("./vn-test", tc.args...)
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
	buildCmd := exec.Command("go", "build", "-o", "vn-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-test")

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("./vn-test", tc.args...)
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
	buildCmd := exec.Command("go", "build", "-o", "vn-test", "../../.")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-test")

	cmd := exec.Command("./vn-test")
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

	buildCmd := exec.Command("go", "build", "-o", "vn-bench", "../../.")
	if err := buildCmd.Run(); err != nil {
		b.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("vn-bench")

	serverCmd := exec.Command("go", "run", "../../test-server/main.go")
	if err := serverCmd.Start(); err != nil {
		b.Fatalf("Failed to start test server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	time.Sleep(2 * time.Second)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command("./vn-bench", "sqli", "http://localhost:8080/?id=1", "--threads", "2")
		if err := cmd.Run(); err != nil {
			b.Errorf("Benchmark iteration %d failed: %v", i, err)
		}
	}
}
