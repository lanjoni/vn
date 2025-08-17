package shared

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type PerformanceReport struct {
	TotalTests           int           `json:"total_tests"`
	SuccessfulTests      int           `json:"successful_tests"`
	FailedTests          int           `json:"failed_tests"`
	TotalExecutionTime   time.Duration `json:"total_execution_time"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	MedianExecutionTime  time.Duration `json:"median_execution_time"`
	P95ExecutionTime     time.Duration `json:"p95_execution_time"`
	TotalBuildTime       time.Duration `json:"total_build_time"`
	TotalNetworkTime     time.Duration `json:"total_network_time"`
	SlowestTests         []TestMetrics `json:"slowest_tests"`
	FastestTests         []TestMetrics `json:"fastest_tests"`
	FailedTestNames      []string      `json:"failed_test_names"`
	GeneratedAt          time.Time     `json:"generated_at"`
}

type ReportGenerator struct {
	metrics []TestMetrics
}

func NewReportGenerator(metrics []TestMetrics) *ReportGenerator {
	return &ReportGenerator{
		metrics: metrics,
	}
}

func (rg *ReportGenerator) GenerateReport() PerformanceReport {
	if len(rg.metrics) == 0 {
		return PerformanceReport{
			GeneratedAt: time.Now(),
		}
	}

	report := PerformanceReport{
		TotalTests:  len(rg.metrics),
		GeneratedAt: time.Now(),
	}

	var totalExecutionTime time.Duration
	var totalBuildTime time.Duration
	var totalNetworkTime time.Duration
	var successfulExecutionTime time.Duration
	var successfulMetrics []TestMetrics
	var failedTestNames []string

	for _, m := range rg.metrics {
		totalExecutionTime += m.ExecutionTime
		totalBuildTime += m.BuildTime
		totalNetworkTime += m.NetworkTime

		if m.Success {
			report.SuccessfulTests++
			successfulExecutionTime += m.ExecutionTime
			successfulMetrics = append(successfulMetrics, m)
		} else {
			report.FailedTests++
			failedTestNames = append(failedTestNames, m.TestName)
		}
	}

	report.TotalExecutionTime = totalExecutionTime
	report.TotalBuildTime = totalBuildTime
	report.TotalNetworkTime = totalNetworkTime
	report.FailedTestNames = failedTestNames

	if len(successfulMetrics) > 0 {
		report.AverageExecutionTime = successfulExecutionTime / time.Duration(len(successfulMetrics))
		report.MedianExecutionTime = calculateMedianDuration(successfulMetrics,
			func(m TestMetrics) time.Duration { return m.ExecutionTime })
		report.P95ExecutionTime = calculateP95Duration(successfulMetrics,
			func(m TestMetrics) time.Duration { return m.ExecutionTime })
	}

	sortedMetrics := make([]TestMetrics, len(successfulMetrics))
	copy(sortedMetrics, successfulMetrics)
	sort.Slice(sortedMetrics, func(i, j int) bool {
		return sortedMetrics[i].ExecutionTime > sortedMetrics[j].ExecutionTime
	})

	slowestCount := 5
	if len(sortedMetrics) < slowestCount {
		slowestCount = len(sortedMetrics)
	}
	report.SlowestTests = sortedMetrics[:slowestCount]

	fastestCount := 5
	if len(sortedMetrics) < fastestCount {
		fastestCount = len(sortedMetrics)
	}
	if fastestCount > 0 {
		fastestTests := make([]TestMetrics, fastestCount)
		copy(fastestTests, sortedMetrics[len(sortedMetrics)-fastestCount:])
		sort.Slice(fastestTests, func(i, j int) bool {
			return fastestTests[i].ExecutionTime < fastestTests[j].ExecutionTime
		})
		report.FastestTests = fastestTests
	}

	return report
}

func (rg *ReportGenerator) GenerateTextReport(writer io.Writer) error {
	report := rg.GenerateReport()

	fmt.Fprintf(writer, "Test Performance Report\n")
	fmt.Fprintf(writer, "=======================\n")
	fmt.Fprintf(writer, "Generated at: %s\n\n", report.GeneratedAt.Format(time.RFC3339))

	fmt.Fprintf(writer, "Summary:\n")
	fmt.Fprintf(writer, "  Total Tests: %d\n", report.TotalTests)
	fmt.Fprintf(writer, "  Successful: %d\n", report.SuccessfulTests)
	fmt.Fprintf(writer, "  Failed: %d\n", report.FailedTests)
	fmt.Fprintf(writer, "  Success Rate: %.1f%%\n\n", float64(report.SuccessfulTests)/float64(report.TotalTests)*100)

	fmt.Fprintf(writer, "Execution Times:\n")
	fmt.Fprintf(writer, "  Total: %v\n", report.TotalExecutionTime)
	fmt.Fprintf(writer, "  Average: %v\n", report.AverageExecutionTime)
	fmt.Fprintf(writer, "  Median: %v\n", report.MedianExecutionTime)
	fmt.Fprintf(writer, "  95th Percentile: %v\n\n", report.P95ExecutionTime)

	fmt.Fprintf(writer, "Resource Usage:\n")
	fmt.Fprintf(writer, "  Total Build Time: %v\n", report.TotalBuildTime)
	fmt.Fprintf(writer, "  Total Network Time: %v\n\n", report.TotalNetworkTime)

	if len(report.SlowestTests) > 0 {
		fmt.Fprintf(writer, "Slowest Tests:\n")
		for i, test := range report.SlowestTests {
			fmt.Fprintf(writer, "  %d. %s: %v\n", i+1, test.TestName, test.ExecutionTime)
		}
		fmt.Fprintf(writer, "\n")
	}

	if len(report.FastestTests) > 0 {
		fmt.Fprintf(writer, "Fastest Tests:\n")
		for i, test := range report.FastestTests {
			fmt.Fprintf(writer, "  %d. %s: %v\n", i+1, test.TestName, test.ExecutionTime)
		}
		fmt.Fprintf(writer, "\n")
	}

	if len(report.FailedTestNames) > 0 {
		fmt.Fprintf(writer, "Failed Tests:\n")
		for _, testName := range report.FailedTestNames {
			fmt.Fprintf(writer, "  - %s\n", testName)
		}
		fmt.Fprintf(writer, "\n")
	}

	return nil
}

func (rg *ReportGenerator) GenerateCSVReport(writer io.Writer) error {
	fmt.Fprintf(writer, "test_name,execution_time_ms,build_time_ms,network_time_ms,setup_time_ms,success,error_message\n")

	for _, m := range rg.metrics {
		errorMsg := strings.ReplaceAll(m.ErrorMessage, ",", ";")
		errorMsg = strings.ReplaceAll(errorMsg, "\n", " ")

		fmt.Fprintf(writer, "%s,%d,%d,%d,%d,%t,%s\n",
			m.TestName,
			m.ExecutionTime.Milliseconds(),
			m.BuildTime.Milliseconds(),
			m.NetworkTime.Milliseconds(),
			m.SetupTime.Milliseconds(),
			m.Success,
			errorMsg,
		)
	}

	return nil
}

func (rg *ReportGenerator) GenerateRegressionReport(detector *RegressionDetector,
	thresholdPercent float64, writer io.Writer) error {
	fmt.Fprintf(writer, "Performance Regression Report\n")
	fmt.Fprintf(writer, "============================\n")
	fmt.Fprintf(writer, "Threshold: %.1f%%\n\n", thresholdPercent)

	regressionCount := 0

	for _, m := range rg.metrics {
		result, err := detector.DetectRegression(m.TestName, m, thresholdPercent)
		if err != nil {
			fmt.Fprintf(writer, "Error checking %s: %v\n", m.TestName, err)
			continue
		}

		if result.Regressed {
			regressionCount++
			fmt.Fprintf(writer, "REGRESSION: %s\n", result.TestName)
			fmt.Fprintf(writer, "  Reason: %s\n", result.Reason)
			if result.ExecutionRegression > 0 {
				fmt.Fprintf(writer, "  Execution Time: %.1f%% slower\n", result.ExecutionRegression)
			}
			if result.BuildRegression > 0 {
				fmt.Fprintf(writer, "  Build Time: %.1f%% slower\n", result.BuildRegression)
			}
			if result.NetworkRegression > 0 {
				fmt.Fprintf(writer, "  Network Time: %.1f%% slower\n", result.NetworkRegression)
			}
			fmt.Fprintf(writer, "\n")
		}
	}

	if regressionCount == 0 {
		fmt.Fprintf(writer, "No performance regressions detected.\n")
	} else {
		fmt.Fprintf(writer, "Total regressions found: %d\n", regressionCount)
	}

	return nil
}
