package cmd

import (
	"fmt"
	"time"

	"vn/internal/scanner"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var xssCmd = &cobra.Command{
	Use:   "xss [URL]",
	Short: "Test for Cross-Site Scripting (XSS) vulnerabilities",
	Long: `Test endpoints for XSS vulnerabilities using various payloads.
This command tests for reflected, stored, and DOM-based XSS vulnerabilities.

Examples:
  vn xss https://example.com/search?q=test
  vn xss https://example.com/comment --method POST --data "comment=test&name=user"
  vn xss https://example.com/profile --params "name,email,bio"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		method, _ := cmd.Flags().GetString("method")
		data, _ := cmd.Flags().GetString("data")
		params, _ := cmd.Flags().GetStringSlice("params")
		headers, _ := cmd.Flags().GetStringSlice("headers")
		timeout, _ := cmd.Flags().GetInt("timeout")
		threads, _ := cmd.Flags().GetInt("threads")

		config := scanner.XSSConfig{
			URL:     url,
			Method:  method,
			Data:    data,
			Params:  params,
			Headers: headers,
			Timeout: time.Duration(timeout) * time.Second,
			Threads: threads,
		}

		color.New(color.FgGreen, color.Bold).Printf("🔍 Starting XSS scan on: %s\n", url)  // #nosec G104
		color.New(color.FgYellow).Printf("⚙️  Method: %s, Timeout: %ds, Threads: %d\n\n", // #nosec G104
			method, timeout, threads)

		xssScanner := scanner.NewXSSScanner(config)
		results := xssScanner.Scan()

		displayXSSResults(results)
	},
}

func displayXSSResults(results []scanner.XSSResult) {
	if len(results) == 0 {
		color.New(color.FgGreen, color.Bold).Println("✅ No XSS vulnerabilities found!") // #nosec G104
		return
	}

	color.New(color.FgRed, color.Bold).Printf("🚨 Found %d potential XSS vulnerabilities:\n\n", len(results)) // #nosec G104

	for i, result := range results {
		color.New(color.FgRed, color.Bold).Printf("[%d] XSS Vulnerability Detected\n", i+1) // #nosec G104
		color.New(color.FgWhite).Printf("   URL: %s\n", result.URL)                         // #nosec G104
		color.New(color.FgWhite).Printf("   Parameter: %s\n", result.Parameter)             // #nosec G104
		color.New(color.FgWhite).Printf("   Payload: %s\n", result.Payload)                 // #nosec G104
		color.New(color.FgWhite).Printf("   Type: %s\n", result.Type)                       // #nosec G104
		color.New(color.FgYellow).Printf("   Evidence: %s\n", result.Evidence)              // #nosec G104
		color.New(color.FgCyan).Printf("   Risk Level: %s\n", result.RiskLevel)             // #nosec G104
		fmt.Println()
	}

	color.New(color.FgRed, color.Bold).Println("⚠️  Please review and fix these vulnerabilities!") // #nosec G104
}

func init() {
	rootCmd.AddCommand(xssCmd)

	xssCmd.Flags().StringP("method", "m", "GET", "HTTP method (GET, POST)")
	xssCmd.Flags().StringP("data", "d", "", "POST data (form-encoded)")
	xssCmd.Flags().StringSliceP("params", "p", []string{}, "Parameters to test (comma-separated)")
	xssCmd.Flags().StringSliceP("headers", "H", []string{}, "Custom headers to test")
	xssCmd.Flags().IntP("timeout", "t", 10, "Request timeout in seconds")
	xssCmd.Flags().IntP("threads", "T", 5, "Number of concurrent threads")
}
