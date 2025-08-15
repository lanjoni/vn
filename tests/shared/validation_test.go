package shared

import (
	"testing"
	"time"
)

func TestPerformanceValidator(t *testing.T) {
	validator := NewPerformanceValidator()
	
	target := PerformanceTarget{
		TestName:         "TestExample",
		MaxExecutionTime: 200 * time.Millisecond,
		MaxBuildTime:     100 * time.Millisecond,
		MaxNetworkTime:   50 * time.Millisecond,
		MinSuccessRate:   0.95,
	}
	validator.SetTarget(target)
	
	passingMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 150 * time.Millisecond,
		BuildTime:     80 * time.Millisecond,
		NetworkTime:   30 * time.Millisecond,
		Success:       true,
	}
	
	result := validator.ValidateMetrics(passingMetrics)
	if !result.Passed {
		t.Errorf("Expected validation to pass, but got violations: %v", result.Violations)
	}
	
	failingMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 300 * time.Millisecond,
		BuildTime:     150 * time.Millisecond,
		NetworkTime:   80 * time.Millisecond,
		Success:       true,
	}
	
	result = validator.ValidateMetrics(failingMetrics)
	if result.Passed {
		t.Error("Expected validation to fail for metrics exceeding targets")
	}
	
	if len(result.Violations) != 3 {
		t.Errorf("Expected 3 violations, got %d: %v", len(result.Violations), result.Violations)
	}
}

func TestPerformanceValidatorDefaultTargets(t *testing.T) {
	validator := NewPerformanceValidator()
	validator.SetDefaultTargets()
	
	integrationMetrics := TestMetrics{
		TestName:      "integration_tests",
		ExecutionTime: 90 * time.Second,
		BuildTime:     20 * time.Second,
		NetworkTime:   5 * time.Second,
		Success:       true,
	}
	
	result := validator.ValidateMetrics(integrationMetrics)
	if !result.Passed {
		t.Errorf("Expected integration test validation to pass, but got violations: %v", result.Violations)
	}
}

func TestValidateReport(t *testing.T) {
	validator := NewPerformanceValidator()
	validator.SetDefaultTargets()
	
	report := PerformanceReport{
		TotalTests:         10,
		SuccessfulTests:    10,
		FailedTests:        0,
		TotalExecutionTime: 90 * time.Second,
	}
	
	results := validator.ValidateReport(report)
	
	allPassed := true
	for _, result := range results {
		if !result.Passed {
			allPassed = false
			t.Logf("Validation failed for %s: %v", result.TestName, result.Violations)
		}
	}
	
	if !allPassed {
		t.Error("Some validations failed")
	}
}

func TestValidateWithBaseline(t *testing.T) {
	validator := NewPerformanceValidator()
	target := PerformanceTarget{
		TestName:             "TestExample",
		MaxExecutionTime:     200 * time.Millisecond,
		MaxRegressionPercent: 20.0,
	}
	validator.SetTarget(target)
	
	detector := NewRegressionDetector()
	baselineMetrics := []TestMetrics{
		{TestName: "TestExample", ExecutionTime: 100 * time.Millisecond, Success: true},
	}
	detector.SetBaseline("TestExample", baselineMetrics)
	
	currentMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 150 * time.Millisecond,
		Success:       true,
	}
	
	result := validator.ValidateWithBaseline(currentMetrics, detector)
	if result.Passed {
		t.Error("Expected validation to fail (50% increase > 20% threshold)")
	}
	
	regressedMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 200 * time.Millisecond,
		Success:       true,
	}
	
	result = validator.ValidateWithBaseline(regressedMetrics, detector)
	if result.Passed {
		t.Error("Expected validation to fail for regressed metrics (100% increase > 20% threshold)")
	}
}

func TestEstablishBaselines(t *testing.T) {
	metrics := []TestMetrics{
		{TestName: "Test1", ExecutionTime: 100 * time.Millisecond, Success: true},
		{TestName: "Test1", ExecutionTime: 110 * time.Millisecond, Success: true},
		{TestName: "Test1", ExecutionTime: 90 * time.Millisecond, Success: true},
		{TestName: "Test2", ExecutionTime: 200 * time.Millisecond, Success: true},
		{TestName: "Test2", ExecutionTime: 220 * time.Millisecond, Success: true},
	}
	
	baselines := EstablishBaselines(metrics)
	
	if len(baselines.Baselines) != 2 {
		t.Errorf("Expected 2 baselines, got %d", len(baselines.Baselines))
	}
	
	test1Baseline, exists := baselines.Baselines["Test1"]
	if !exists {
		t.Error("Expected Test1 baseline to exist")
	}
	
	if test1Baseline.MedianExecutionTime != 100*time.Millisecond {
		t.Errorf("Expected Test1 median execution time 100ms, got %v", test1Baseline.MedianExecutionTime)
	}
	
	test2Baseline, exists := baselines.Baselines["Test2"]
	if !exists {
		t.Error("Expected Test2 baseline to exist")
	}
	
	if test2Baseline.MedianExecutionTime != 210*time.Millisecond {
		t.Errorf("Expected Test2 median execution time 210ms, got %v", test2Baseline.MedianExecutionTime)
	}
}