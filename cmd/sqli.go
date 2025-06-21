package cmd

import (
	"fmt"
	"time"

	"vn/internal/scanner"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var sqliCmd = &cobra.Command{
	Use:   "sqli [URL]",
	Short: "Test for SQL injection vulnerabilities",
	Long: `Test endpoints for SQL injection vulnerabilities using various payloads.
This command tests GET parameters, POST data, headers, and cookies for SQL injection.

Examples:
  vn sqli https://example.com/login.php
  vn sqli https://api.example.com/users --method POST --data "username=test&password=test"
  vn sqli https://example.com/search?q=test --params "id,username,search"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		method, _ := cmd.Flags().GetString("method")
		data, _ := cmd.Flags().GetString("data")
		params, _ := cmd.Flags().GetStringSlice("params")
		headers, _ := cmd.Flags().GetStringSlice("headers")
		timeout, _ := cmd.Flags().GetInt("timeout")
		threads, _ := cmd.Flags().GetInt("threads")

		config := scanner.SQLiConfig{
			URL:     url,
			Method:  method,
			Data:    data,
			Params:  params,
			Headers: headers,
			Timeout: time.Duration(timeout) * time.Second,
			Threads: threads,
		}

		color.New(color.FgGreen, color.Bold).Printf("🔍 Starting SQL Injection scan on: %s\n", url)
		color.New(color.FgYellow).Printf("⚙️  Method: %s, Timeout: %ds, Threads: %d\n\n", method, timeout, threads)

		sqlScanner := scanner.NewSQLiScanner(config)
		results := sqlScanner.Scan()

		displaySQLiResults(results)
	},
}

func displaySQLiResults(results []scanner.SQLiResult) {
	if len(results) == 0 {
		color.New(color.FgGreen, color.Bold).Println("✅ No SQL injection vulnerabilities found!")
		return
	}

	color.New(color.FgRed, color.Bold).Printf("🚨 Found %d potential SQL injection vulnerabilities:\n\n", len(results))

	for i, result := range results {
		color.New(color.FgRed, color.Bold).Printf("[%d] SQL Injection Detected\n", i+1)
		color.New(color.FgWhite).Printf("   URL: %s\n", result.URL)
		color.New(color.FgWhite).Printf("   Parameter: %s\n", result.Parameter)
		color.New(color.FgWhite).Printf("   Payload: %s\n", result.Payload)
		color.New(color.FgWhite).Printf("   Method: %s\n", result.Method)
		color.New(color.FgYellow).Printf("   Evidence: %s\n", result.Evidence)
		color.New(color.FgCyan).Printf("   Risk Level: %s\n", result.RiskLevel)
		fmt.Println()
	}

	color.New(color.FgRed, color.Bold).Println("⚠️  Please review and fix these vulnerabilities!")
}

func init() {
	rootCmd.AddCommand(sqliCmd)

	sqliCmd.Flags().StringP("method", "m", "GET", "HTTP method (GET, POST)")
	sqliCmd.Flags().StringP("data", "d", "", "POST data (form-encoded)")
	sqliCmd.Flags().StringSliceP("params", "p", []string{}, "Parameters to test (comma-separated)")
	sqliCmd.Flags().StringSliceP("headers", "H", []string{}, "Custom headers to test")
	sqliCmd.Flags().IntP("timeout", "t", 10, "Request timeout in seconds")
	sqliCmd.Flags().IntP("threads", "T", 5, "Number of concurrent threads")
}
