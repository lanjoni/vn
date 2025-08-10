package cmd

import (
	"fmt"
	"time"

	"vn/internal/scanner"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var misconfigCmd = &cobra.Command{
	Use:   "misconfig [URL]",
	Short: "Test for security misconfigurations",
	Long: `Test endpoints for security misconfigurations including exposed files, 
missing security headers, default credentials, and insecure server configurations.

This command tests for OWASP Top 10 A05:2021 – Security Misconfiguration vulnerabilities.

Examples:
  vn misconfig https://example.com
  vn misconfig https://example.com --tests files,headers
  vn misconfig https://example.com --method POST --headers "Authorization: Bearer token"
  vn misconfig https://example.com --threads 10 --timeout 15`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		method, _ := cmd.Flags().GetString("method")
		headers, _ := cmd.Flags().GetStringSlice("headers")
		timeout, _ := cmd.Flags().GetInt("timeout")
		threads, _ := cmd.Flags().GetInt("threads")
		tests, _ := cmd.Flags().GetStringSlice("tests")

		config := scanner.MisconfigConfig{
			URL:     url,
			Method:  method,
			Headers: headers,
			Timeout: time.Duration(timeout) * time.Second,
			Threads: threads,
			Tests:   tests,
		}

		color.New(color.FgGreen, color.Bold).Printf("🔍 Starting Security Misconfiguration scan on: %s\n", url)
		color.New(color.FgYellow).Printf("⚙️  Method: %s, Timeout: %ds, Threads: %d\n",
			method, timeout, threads)

		if len(tests) > 0 {
			color.New(color.FgYellow).Printf("🎯 Test Categories: %v\n", tests)
		} else {
			color.New(color.FgYellow).Printf("🎯 Test Categories: all\n")
		}
		fmt.Println()

		misconfigScanner := scanner.NewMisconfigScanner(config)
		results := misconfigScanner.Scan()

		DisplayMisconfigResults(results)
	},
}

func DisplayMisconfigResults(results []scanner.MisconfigResult) {
	if len(results) == 0 {
		color.New(color.FgGreen, color.Bold).Println("✅ No security misconfigurations found!")
		return
	}

	sortedResults := SortResultsByRiskAndCategory(results)

	DisplaySummaryStatistics(results)

	color.New(color.FgRed, color.Bold).Printf(
		"🚨 Found %d security misconfigurations:\n\n", len(results))

	DisplayResultsByCategory(sortedResults)

	color.New(color.FgRed, color.Bold).Println("⚠️  Please review and fix these misconfigurations!")
}

func GetRiskColor(riskLevel string) *color.Color {
	switch riskLevel {
	case "High":
		return color.New(color.FgRed, color.Bold)
	case "Medium":
		return color.New(color.FgYellow, color.Bold)
	case "Low":
		return color.New(color.FgGreen, color.Bold)
	default:
		return color.New(color.FgWhite)
	}
}

func SortResultsByRiskAndCategory(results []scanner.MisconfigResult) []scanner.MisconfigResult {
	sortedResults := make([]scanner.MisconfigResult, len(results))
	copy(sortedResults, results)

	for i := 0; i < len(sortedResults)-1; i++ {
		for j := i + 1; j < len(sortedResults); j++ {
			if ShouldSwapResults(sortedResults[i], sortedResults[j]) {
				sortedResults[i], sortedResults[j] = sortedResults[j], sortedResults[i]
			}
		}
	}

	return sortedResults
}

func ShouldSwapResults(a, b scanner.MisconfigResult) bool {
	riskPriority := map[string]int{"High": 3, "Medium": 2, "Low": 1}

	aPriority := riskPriority[a.RiskLevel]
	bPriority := riskPriority[b.RiskLevel]

	if aPriority != bPriority {
		return aPriority < bPriority
	}

	return a.Category > b.Category
}

func DisplaySummaryStatistics(results []scanner.MisconfigResult) {
	categoryStats := make(map[string]map[string]int)
	totalByRisk := map[string]int{"High": 0, "Medium": 0, "Low": 0}

	for _, result := range results {
		if categoryStats[result.Category] == nil {
			categoryStats[result.Category] = make(map[string]int)
		}
		categoryStats[result.Category][result.RiskLevel]++
		totalByRisk[result.RiskLevel]++
	}

	color.New(color.FgCyan, color.Bold).Println("📊 Summary Statistics:")
	fmt.Println()

	color.New(color.FgWhite, color.Bold).Println("Risk Level Distribution:")
	for _, risk := range []string{"High", "Medium", "Low"} {
		if count := totalByRisk[risk]; count > 0 {
			riskColor := GetRiskColor(risk)
			riskColor.Printf("  %s: %d findings\n", risk, count)
		}
	}
	fmt.Println()

	color.New(color.FgWhite, color.Bold).Println("Category Breakdown:")
	for category, risks := range categoryStats {
		categoryName := FormatCategoryName(category)
		color.New(color.FgWhite).Printf("  %s:\n", categoryName)

		for _, risk := range []string{"High", "Medium", "Low"} {
			if count := risks[risk]; count > 0 {
				riskColor := GetRiskColor(risk)
				riskColor.Printf("    %s: %d\n", risk, count)
			}
		}
	}
	fmt.Println()
}

func FormatCategoryName(category string) string {
	switch category {
	case "sensitive-files":
		return "Sensitive Files"
	case "headers":
		return "Security Headers"
	case "defaults":
		return "Default Credentials"
	case "server-config":
		return "Server Configuration"
	default:
		return category
	}
}

func DisplayResultsByCategory(results []scanner.MisconfigResult) {
	currentCategory := ""
	resultIndex := 1

	for _, result := range results {
		if result.Category != currentCategory {
			if currentCategory != "" {
				fmt.Println()
			}
			currentCategory = result.Category
			categoryName := FormatCategoryName(currentCategory)
			color.New(color.FgMagenta, color.Bold).Printf("🔍 %s Issues:\n", categoryName)
			fmt.Println()
		}

		DisplayEnhancedResult(resultIndex, result)
		resultIndex++
	}
}

func DisplayEnhancedResult(index int, result scanner.MisconfigResult) {
	riskColor := GetRiskColor(result.RiskLevel)

	riskColor.Printf("[%d] %s Risk - %s\n", index, result.RiskLevel, result.Finding)

	color.New(color.FgWhite).Printf("   🌐 URL: %s\n", result.URL)

	color.New(color.FgYellow).Printf("   🔍 Evidence: %s\n", result.Evidence)

	color.New(color.FgCyan).Printf("   💡 Remediation: %s\n", result.Remediation)

	fmt.Println()
}

func init() {
	rootCmd.AddCommand(misconfigCmd)

	misconfigCmd.Flags().StringP("method", "m", "GET", "HTTP method for base requests")
	misconfigCmd.Flags().StringSliceP("headers", "H", []string{}, "Custom headers (format: 'Name: Value')")
	misconfigCmd.Flags().IntP("timeout", "t", 10, "Request timeout in seconds")
	misconfigCmd.Flags().IntP("threads", "T", 5, "Number of concurrent threads")
	misconfigCmd.Flags().StringSliceP("tests", "", []string{}, "Specific test categories to run (files,headers,defaults,server)")
}
