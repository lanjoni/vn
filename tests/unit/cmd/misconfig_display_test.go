package cmd_test

import (
	"testing"

	"vn/cmd"
	"vn/internal/scanner"
)

func TestGetRiskColor(t *testing.T) {
	tests := []struct {
		name      string
		riskLevel string
	}{
		{"High risk", "High"},
		{"Medium risk", "Medium"},
		{"Low risk", "Low"},
		{"Unknown risk", "Unknown"},
		{"Empty risk", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := cmd.GetRiskColor(tt.riskLevel)
			if color == nil {
				t.Errorf("GetRiskColor(%s) returned nil", tt.riskLevel)
			}
		})
	}
}

func TestSortResultsByRiskAndCategory(t *testing.T) {
	results := []scanner.MisconfigResult{
		{Category: "headers", RiskLevel: "Low", Finding: "Low headers"},
		{Category: "sensitive-files", RiskLevel: "High", Finding: "High files"},
		{Category: "defaults", RiskLevel: "Medium", Finding: "Medium defaults"},
		{Category: "headers", RiskLevel: "High", Finding: "High headers"},
		{Category: "sensitive-files", RiskLevel: "Low", Finding: "Low files"},
	}

	sorted := cmd.SortResultsByRiskAndCategory(results)

	if sorted[0].RiskLevel != "High" || sorted[1].RiskLevel != "High" {
		t.Errorf("High risk items should come first, got: %s, %s", sorted[0].RiskLevel, sorted[1].RiskLevel)
	}

	if sorted[2].RiskLevel != "Medium" {
		t.Errorf("Medium risk should come after High, got: %s", sorted[2].RiskLevel)
	}

	if sorted[3].RiskLevel != "Low" || sorted[4].RiskLevel != "Low" {
		t.Errorf("Low risk items should come last, got: %s, %s", sorted[3].RiskLevel, sorted[4].RiskLevel)
	}
}

func TestShouldSwapResults(t *testing.T) {
	tests := []struct {
		name     string
		a        scanner.MisconfigResult
		b        scanner.MisconfigResult
		expected bool
	}{
		{
			name:     "High should come before Medium",
			a:        scanner.MisconfigResult{RiskLevel: "Medium"},
			b:        scanner.MisconfigResult{RiskLevel: "High"},
			expected: true,
		},
		{
			name:     "Medium should come before Low",
			a:        scanner.MisconfigResult{RiskLevel: "Low"},
			b:        scanner.MisconfigResult{RiskLevel: "Medium"},
			expected: true,
		},
		{
			name:     "Same risk level, sort by category",
			a:        scanner.MisconfigResult{RiskLevel: "High", Category: "z-category"},
			b:        scanner.MisconfigResult{RiskLevel: "High", Category: "a-category"},
			expected: true,
		},
		{
			name:     "No swap needed",
			a:        scanner.MisconfigResult{RiskLevel: "High"},
			b:        scanner.MisconfigResult{RiskLevel: "Medium"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.ShouldSwapResults(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("ShouldSwapResults() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFormatCategoryName(t *testing.T) {
	tests := []struct {
		category string
		expected string
	}{
		{"sensitive-files", "Sensitive Files"},
		{"headers", "Security Headers"},
		{"defaults", "Default Credentials"},
		{"server-config", "Server Configuration"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := cmd.FormatCategoryName(tt.category)
			if result != tt.expected {
				t.Errorf("FormatCategoryName(%s) = %s, expected %s", tt.category, result, tt.expected)
			}
		})
	}
}

func TestDisplayFunctionsDoNotPanic(t *testing.T) {
	results := []scanner.MisconfigResult{
		{
			URL:         "https://example.com/.env",
			Category:    "sensitive-files",
			Finding:     "Sensitive file exposed",
			Evidence:    "File accessible at /.env",
			RiskLevel:   "High",
			Remediation: "Remove or restrict access to .env file",
		},
		{
			URL:         "https://example.com",
			Category:    "headers",
			Finding:     "Missing security header",
			Evidence:    "X-Frame-Options header not found",
			RiskLevel:   "Medium",
			Remediation: "Add X-Frame-Options header",
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Display functions panicked: %v", r)
		}
	}()

	cmd.DisplaySummaryStatistics(results)
	cmd.DisplayMisconfigResults(results)
	cmd.DisplayResultsByCategory(results)
	cmd.DisplayEnhancedResult(1, results[0])

	var emptyResults []scanner.MisconfigResult
	cmd.DisplayMisconfigResults(emptyResults)
	cmd.DisplaySummaryStatistics(emptyResults)
	cmd.DisplayResultsByCategory(emptyResults)
}

func TestSortingLogic(t *testing.T) {
	results := []scanner.MisconfigResult{
		{Category: "z-category", RiskLevel: "Low", Finding: "Should be last"},
		{Category: "a-category", RiskLevel: "High", Finding: "Should be first"},
		{Category: "m-category", RiskLevel: "Medium", Finding: "Should be middle"},
		{Category: "b-category", RiskLevel: "High", Finding: "Should be second"},
		{Category: "n-category", RiskLevel: "Medium", Finding: "Should be after first medium"},
	}

	sorted := cmd.SortResultsByRiskAndCategory(results)

	if sorted[0].RiskLevel != "High" || sorted[1].RiskLevel != "High" {
		t.Errorf("First two should be High risk, got: %s, %s", sorted[0].RiskLevel, sorted[1].RiskLevel)
	}

	if sorted[0].Category != "a-category" || sorted[1].Category != "b-category" {
		t.Errorf("High risk items should be sorted by category, got: %s, %s", sorted[0].Category, sorted[1].Category)
	}

	if sorted[2].RiskLevel != "Medium" || sorted[3].RiskLevel != "Medium" {
		t.Errorf("Medium risk items should come after High, got: %s, %s", sorted[2].RiskLevel, sorted[3].RiskLevel)
	}

	if sorted[4].RiskLevel != "Low" {
		t.Errorf("Low risk should come last, got: %s", sorted[4].RiskLevel)
	}
}

func TestCategoryNameFormatting(t *testing.T) {
	categories := map[string]string{
		"sensitive-files": "Sensitive Files",
		"headers":         "Security Headers",
		"defaults":        "Default Credentials",
		"server-config":   "Server Configuration",
	}

	for input, expected := range categories {
		result := cmd.FormatCategoryName(input)
		if result != expected {
			t.Errorf("FormatCategoryName(%s) = %s, expected %s", input, result, expected)
		}
	}

	unknown := "unknown-category"
	result := cmd.FormatCategoryName(unknown)
	if result != unknown {
		t.Errorf("FormatCategoryName(%s) = %s, expected %s", unknown, result, unknown)
	}
}
