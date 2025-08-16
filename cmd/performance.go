package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"vn/tests/shared"
)

const (
	defaultThreshold = 25.0
	dirMode          = 0755
)

var performanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "Performance validation and reporting tools",
	Long:  "Tools for validating test performance, generating reports, and detecting regressions",
}

var validateCmd = &cobra.Command{
	Use:   "validate [metrics-file]",
	Short: "Validate performance metrics against targets",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

var reportCmd = &cobra.Command{
	Use:   "report [metrics-file]",
	Short: "Generate performance report from metrics",
	Args:  cobra.ExactArgs(1),
	RunE:  runReport,
}

var baselineCmd = &cobra.Command{
	Use:   "baseline [metrics-file] [baseline-file]",
	Short: "Establish performance baselines from metrics",
	Args:  cobra.ExactArgs(2),
	RunE:  runBaseline,
}

var regressionCmd = &cobra.Command{
	Use:   "regression [metrics-file] [baseline-file]",
	Short: "Check for performance regressions",
	Args:  cobra.ExactArgs(2),
	RunE:  runRegression,
}

var (
	outputFormat     string
	thresholdPercent float64
	failOnViolation  bool
)

func init() {
	rootCmd.AddCommand(performanceCmd)
	performanceCmd.AddCommand(validateCmd)
	performanceCmd.AddCommand(reportCmd)
	performanceCmd.AddCommand(baselineCmd)
	performanceCmd.AddCommand(regressionCmd)

	validateCmd.Flags().BoolVar(&failOnViolation, "fail-on-violation", false,
		"Exit with non-zero code if validation fails")
	reportCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format: text, csv, json")
	regressionCmd.Flags().Float64Var(&thresholdPercent, "threshold", defaultThreshold, "Regression threshold percentage")
	regressionCmd.Flags().BoolVar(&failOnViolation, "fail-on-regression", false,
		"Exit with non-zero code if regression detected")
}

func runValidate(cmd *cobra.Command, args []string) error {
	metricsFile := args[0]

	metrics, err := loadMetrics(metricsFile)
	if err != nil {
		return fmt.Errorf("failed to load metrics: %w", err)
	}

	validator := shared.NewPerformanceValidator()
	validator.SetDefaultTargets()

	hasViolations := false
	for _, metric := range metrics {
		result := validator.ValidateMetrics(metric)
		if !result.Passed {
			hasViolations = true
			fmt.Printf("VIOLATION: %s\n", result.TestName)
			for _, violation := range result.Violations {
				fmt.Printf("  - %s\n", violation)
			}
		} else {
			fmt.Printf("PASS: %s\n", result.TestName)
		}
	}

	if hasViolations {
		fmt.Printf("\nValidation completed with violations\n")
		if failOnViolation {
			os.Exit(1)
		}
	} else {
		fmt.Printf("\nAll validations passed\n")
	}

	return nil
}

func runReport(cmd *cobra.Command, args []string) error {
	metricsFile := args[0]

	metrics, err := loadMetrics(metricsFile)
	if err != nil {
		return fmt.Errorf("failed to load metrics: %w", err)
	}

	generator := shared.NewReportGenerator(metrics)

	switch outputFormat {
	case "text":
		return generator.GenerateTextReport(os.Stdout)
	case "csv":
		return generator.GenerateCSVReport(os.Stdout)
	case "json":
		report := generator.GenerateReport()
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func runBaseline(cmd *cobra.Command, args []string) error {
	metricsFile := args[0]
	baselineFile := args[1]

	metrics, err := loadMetrics(metricsFile)
	if err != nil {
		return fmt.Errorf("failed to load metrics: %w", err)
	}

	baselines := shared.EstablishBaselines(metrics)

	if mkdirErr := os.MkdirAll(filepath.Dir(baselineFile), dirMode); mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	file, err := os.Create(baselineFile)
	if err != nil {
		return fmt.Errorf("failed to create baseline file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(baselines); err != nil {
		return fmt.Errorf("failed to encode baselines: %w", err)
	}

	fmt.Printf("Established baselines for %d tests in %s\n", len(baselines.Baselines), baselineFile)
	return nil
}

func runRegression(cmd *cobra.Command, args []string) error {
	metricsFile := args[0]
	baselineFile := args[1]

	metrics, err := loadMetrics(metricsFile)
	if err != nil {
		return fmt.Errorf("failed to load metrics: %w", err)
	}

	baselines, err := loadBaselines(baselineFile)
	if err != nil {
		return fmt.Errorf("failed to load baselines: %w", err)
	}

	detector := shared.NewRegressionDetector()
	for testName, baseline := range baselines.Baselines {
		detector.SetBaseline(testName, []shared.TestMetrics{
			{
				TestName:      baseline.TestName,
				ExecutionTime: baseline.MedianExecutionTime,
				BuildTime:     baseline.MedianBuildTime,
				NetworkTime:   baseline.MedianNetworkTime,
				Success:       true,
			},
		})
	}

	hasRegressions := false
	for _, metric := range metrics {
		result, err := detector.DetectRegression(metric.TestName, metric, thresholdPercent)
		if err != nil {
			fmt.Printf("ERROR: %s - %v\n", metric.TestName, err)
			continue
		}

		if result.Regressed {
			hasRegressions = true
			fmt.Printf("REGRESSION: %s\n", result.TestName)
			fmt.Printf("  Reason: %s\n", result.Reason)
		} else {
			fmt.Printf("PASS: %s\n", result.TestName)
		}
	}

	if hasRegressions {
		fmt.Printf("\nRegressions detected\n")
		if failOnViolation {
			os.Exit(1)
		}
	} else {
		fmt.Printf("\nNo regressions detected\n")
	}

	return nil
}

func loadMetrics(filename string) ([]shared.TestMetrics, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metrics []shared.TestMetrics
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func loadBaselines(filename string) (shared.PerformanceBaselines, error) {
	file, err := os.Open(filename)
	if err != nil {
		return shared.PerformanceBaselines{}, err
	}
	defer file.Close()

	var baselines shared.PerformanceBaselines
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&baselines); err != nil {
		return shared.PerformanceBaselines{}, err
	}

	return baselines, nil
}
