package shared

import (
	"testing"
	"time"
)

func TestRegressionDetector(t *testing.T) {
	detector := NewRegressionDetector()

	baselineMetrics := []TestMetrics{
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
		{TestName: "TestExample", ExecutionTime: 110 * time.Millisecond, Success: true},
		{TestName: "TestExample", ExecutionTime: 90 * time.Millisecond, Success: true},
		{TestName: "TestExample", ExecutionTime: 105 * time.Millisecond, Success: true},
		{TestName: "TestExample", ExecutionTime: 95 * time.Millisecond, Success: true},
	}

	detector.SetBaseline("TestExample", baselineMetrics)

	baseline, exists := detector.GetBaseline("TestExample")
	if !exists {
		t.Fatal("Expected baseline to exist")
	}

	if baseline.TestName != "TestExample" {
		t.Errorf("Expected baseline test name 'TestExample', got '%s'", baseline.TestName)
	}

	if baseline.MedianExecutionTime != 100*time.Millisecond {
		t.Errorf("Expected median execution time 100ms, got %v", baseline.MedianExecutionTime)
	}
}

func TestRegressionDetection(t *testing.T) {
	detector := NewRegressionDetector()

	baselineMetrics := []TestMetrics{
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
	}

	detector.SetBaseline("TestExample", baselineMetrics)

	currentMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 150 * time.Millisecond,
		Success:       true,
	}

	result, err := detector.DetectRegression("TestExample", currentMetrics, 20.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Regressed {
		t.Error("Expected regression to be detected (50% increase > 20% threshold)")
	}

	if result.ExecutionRegression != 50.0 {
		t.Errorf("Expected 50%% regression, got %.1f%%", result.ExecutionRegression)
	}
}

func TestNoRegressionDetection(t *testing.T) {
	detector := NewRegressionDetector()

	baselineMetrics := []TestMetrics{
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
	}

	detector.SetBaseline("TestExample", baselineMetrics)

	currentMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 110 * time.Millisecond,
		Success:       true,
	}

	result, err := detector.DetectRegression("TestExample", currentMetrics, 20.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Regressed {
		t.Error("Expected no regression to be detected (10% increase < 20% threshold)")
	}
}

func TestFailedTestRegression(t *testing.T) {
	detector := NewRegressionDetector()

	baselineMetrics := []TestMetrics{
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
	}

	detector.SetBaseline("TestExample", baselineMetrics)

	currentMetrics := TestMetrics{
		TestName: "TestExample",
		Success:  false,
	}

	result, err := detector.DetectRegression("TestExample", currentMetrics, 20.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Regressed {
		t.Error("Expected regression to be detected for failed test")
	}

	if result.Reason != "Test failed" {
		t.Errorf("Expected reason 'Test failed', got '%s'", result.Reason)
	}
}

func TestCalculateMedianDuration(t *testing.T) {
	metrics := []TestMetrics{
		{ExecutionTime: 100 * time.Millisecond},
		{ExecutionTime: 200 * time.Millisecond},
		{ExecutionTime: 300 * time.Millisecond},
		{ExecutionTime: 400 * time.Millisecond},
		{ExecutionTime: 500 * time.Millisecond},
	}

	median := calculateMedianDuration(metrics, func(m TestMetrics) time.Duration { return m.ExecutionTime })
	expected := 300 * time.Millisecond

	if median != expected {
		t.Errorf("Expected median %v, got %v", expected, median)
	}
}

func TestCalculateP95Duration(t *testing.T) {
	metrics := make([]TestMetrics, 100)
	for i := 0; i < 100; i++ {
		metrics[i] = TestMetrics{ExecutionTime: time.Duration(i+1) * time.Millisecond}
	}

	p95 := calculateP95Duration(metrics, func(m TestMetrics) time.Duration { return m.ExecutionTime })
	expected := 95 * time.Millisecond

	if p95 != expected {
		t.Errorf("Expected P95 %v, got %v", expected, p95)
	}
}
