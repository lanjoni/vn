package shared

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestReportGenerator(t *testing.T) {
	metrics := []TestMetrics{
		{
			TestName:      "Test1",
			ExecutionTime: 100 * time.Millisecond,
			BuildTime:     50 * time.Millisecond,
			NetworkTime:   25 * time.Millisecond,
			Success:       true,
		},
		{
			TestName:      "Test2",
			ExecutionTime: 200 * time.Millisecond,
			BuildTime:     75 * time.Millisecond,
			NetworkTime:   30 * time.Millisecond,
			Success:       true,
		},
		{
			TestName:     "Test3",
			Success:      false,
			ErrorMessage: "test failed",
		},
	}

	generator := NewReportGenerator(metrics)
	report := generator.GenerateReport()

	if report.TotalTests != 3 {
		t.Errorf("Expected 3 total tests, got %d", report.TotalTests)
	}

	if report.SuccessfulTests != 2 {
		t.Errorf("Expected 2 successful tests, got %d", report.SuccessfulTests)
	}

	if report.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", report.FailedTests)
	}

	expectedTotalTime := 300 * time.Millisecond
	if report.TotalExecutionTime != expectedTotalTime {
		t.Errorf("Expected total execution time %v, got %v", expectedTotalTime, report.TotalExecutionTime)
	}

	expectedAverage := 150 * time.Millisecond
	if report.AverageExecutionTime != expectedAverage {
		t.Errorf("Expected average execution time %v, got %v", expectedAverage, report.AverageExecutionTime)
	}

	if len(report.FailedTestNames) != 1 || report.FailedTestNames[0] != "Test3" {
		t.Errorf("Expected failed test names ['Test3'], got %v", report.FailedTestNames)
	}

	if len(report.SlowestTests) != 2 {
		t.Errorf("Expected 2 slowest tests, got %d", len(report.SlowestTests))
	}

	if report.SlowestTests[0].TestName != "Test2" {
		t.Errorf("Expected slowest test to be 'Test2', got '%s'", report.SlowestTests[0].TestName)
	}
}

func TestGenerateTextReport(t *testing.T) {
	metrics := []TestMetrics{
		{
			TestName:      "Test1",
			ExecutionTime: 100 * time.Millisecond,
			Success:       true,
		},
		{
			TestName:     "Test2",
			Success:      false,
			ErrorMessage: "test failed",
		},
	}

	generator := NewReportGenerator(metrics)
	var buffer bytes.Buffer
	err := generator.GenerateTextReport(&buffer)
	if err != nil {
		t.Fatalf("Failed to generate text report: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "Test Performance Report") {
		t.Error("Expected report to contain title")
	}

	if !strings.Contains(output, "Total Tests: 2") {
		t.Error("Expected report to contain total test count")
	}

	if !strings.Contains(output, "Successful: 1") {
		t.Error("Expected report to contain successful test count")
	}

	if !strings.Contains(output, "Failed: 1") {
		t.Error("Expected report to contain failed test count")
	}
}

func TestGenerateCSVReport(t *testing.T) {
	metrics := []TestMetrics{
		{
			TestName:      "Test1",
			ExecutionTime: 100 * time.Millisecond,
			BuildTime:     50 * time.Millisecond,
			Success:       true,
		},
		{
			TestName:     "Test2",
			Success:      false,
			ErrorMessage: "test failed",
		},
	}

	generator := NewReportGenerator(metrics)
	var buffer bytes.Buffer
	err := generator.GenerateCSVReport(&buffer)
	if err != nil {
		t.Fatalf("Failed to generate CSV report: %v", err)
	}

	output := buffer.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines (header + 2 data), got %d", len(lines))
	}

	header := lines[0]
	if !strings.Contains(header, "test_name") {
		t.Error("Expected CSV header to contain test_name")
	}

	if !strings.Contains(lines[1], "Test1") {
		t.Error("Expected first data line to contain Test1")
	}

	if !strings.Contains(lines[2], "Test2") {
		t.Error("Expected second data line to contain Test2")
	}
}

func TestGenerateRegressionReport(t *testing.T) {
	detector := NewRegressionDetector()
	
	baselineMetrics := []TestMetrics{
		{TestName: "Test1", ExecutionTime: 100 * time.Millisecond, Success: true},
	}
	detector.SetBaseline("Test1", baselineMetrics)

	currentMetrics := []TestMetrics{
		{TestName: "Test1", ExecutionTime: 200 * time.Millisecond, Success: true},
	}

	generator := NewReportGenerator(currentMetrics)
	var buffer bytes.Buffer
	err := generator.GenerateRegressionReport(detector, 20.0, &buffer)
	if err != nil {
		t.Fatalf("Failed to generate regression report: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "Performance Regression Report") {
		t.Error("Expected report to contain title")
	}

	if !strings.Contains(output, "REGRESSION: Test1") {
		t.Error("Expected report to contain regression for Test1")
	}

	if !strings.Contains(output, "100.0% slower") {
		t.Error("Expected report to contain regression percentage")
	}
}