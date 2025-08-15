package shared

import (
	"fmt"
	"time"
)

type PerformanceTarget struct {
	TestName           string        `json:"test_name"`
	MaxExecutionTime   time.Duration `json:"max_execution_time"`
	MaxBuildTime       time.Duration `json:"max_build_time"`
	MaxNetworkTime     time.Duration `json:"max_network_time"`
	MinSuccessRate     float64       `json:"min_success_rate"`
	MaxRegressionPercent float64     `json:"max_regression_percent"`
}

type ValidationResult struct {
	TestName    string `json:"test_name"`
	Passed      bool   `json:"passed"`
	Violations  []string `json:"violations"`
	ActualMetrics TestMetrics `json:"actual_metrics"`
	Target      PerformanceTarget `json:"target"`
}

type PerformanceValidator struct {
	targets map[string]PerformanceTarget
}

func NewPerformanceValidator() *PerformanceValidator {
	return &PerformanceValidator{
		targets: make(map[string]PerformanceTarget),
	}
}

func (pv *PerformanceValidator) SetTarget(target PerformanceTarget) {
	pv.targets[target.TestName] = target
}

func (pv *PerformanceValidator) SetDefaultTargets() {
	pv.SetTarget(PerformanceTarget{
		TestName:             "integration_tests",
		MaxExecutionTime:     2 * time.Minute,
		MaxBuildTime:         30 * time.Second,
		MaxNetworkTime:       10 * time.Second,
		MinSuccessRate:       0.95,
		MaxRegressionPercent: 25.0,
	})
	
	pv.SetTarget(PerformanceTarget{
		TestName:             "e2e_tests",
		MaxExecutionTime:     1 * time.Minute,
		MaxBuildTime:         30 * time.Second,
		MaxNetworkTime:       5 * time.Second,
		MinSuccessRate:       0.90,
		MaxRegressionPercent: 30.0,
	})
	
	pv.SetTarget(PerformanceTarget{
		TestName:             "unit_tests",
		MaxExecutionTime:     30 * time.Second,
		MaxBuildTime:         10 * time.Second,
		MaxNetworkTime:       1 * time.Second,
		MinSuccessRate:       0.98,
		MaxRegressionPercent: 15.0,
	})
}

func (pv *PerformanceValidator) ValidateMetrics(metrics TestMetrics) ValidationResult {
	target, exists := pv.targets[metrics.TestName]
	if !exists {
		return ValidationResult{
			TestName: metrics.TestName,
			Passed:   true,
			ActualMetrics: metrics,
		}
	}
	
	result := ValidationResult{
		TestName:      metrics.TestName,
		Passed:        true,
		ActualMetrics: metrics,
		Target:        target,
	}
	
	if metrics.ExecutionTime > target.MaxExecutionTime {
		result.Passed = false
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Execution time %v exceeds target %v", 
				metrics.ExecutionTime, target.MaxExecutionTime))
	}
	
	if metrics.BuildTime > target.MaxBuildTime {
		result.Passed = false
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Build time %v exceeds target %v", 
				metrics.BuildTime, target.MaxBuildTime))
	}
	
	if metrics.NetworkTime > target.MaxNetworkTime {
		result.Passed = false
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Network time %v exceeds target %v", 
				metrics.NetworkTime, target.MaxNetworkTime))
	}
	
	return result
}

func (pv *PerformanceValidator) ValidateReport(report PerformanceReport) []ValidationResult {
	var results []ValidationResult
	
	successRate := float64(report.SuccessfulTests) / float64(report.TotalTests)
	
	for testName, target := range pv.targets {
		result := ValidationResult{
			TestName: testName,
			Passed:   true,
			Target:   target,
		}
		
		if testName == "integration_tests" && report.TotalExecutionTime > target.MaxExecutionTime {
			result.Passed = false
			result.Violations = append(result.Violations, 
				fmt.Sprintf("Total execution time %v exceeds target %v", 
					report.TotalExecutionTime, target.MaxExecutionTime))
		}
		
		if successRate < target.MinSuccessRate {
			result.Passed = false
			result.Violations = append(result.Violations, 
				fmt.Sprintf("Success rate %.2f%% below target %.2f%%", 
					successRate*100, target.MinSuccessRate*100))
		}
		
		results = append(results, result)
	}
	
	return results
}

func (pv *PerformanceValidator) ValidateWithBaseline(metrics TestMetrics, detector *RegressionDetector) ValidationResult {
	result := pv.ValidateMetrics(metrics)
	
	target, exists := pv.targets[metrics.TestName]
	if !exists {
		return result
	}
	
	regressionResult, err := detector.DetectRegression(metrics.TestName, metrics, target.MaxRegressionPercent)
	if err == nil && regressionResult.Regressed {
		result.Passed = false
		result.Violations = append(result.Violations, 
			fmt.Sprintf("Performance regression detected: %s", regressionResult.Reason))
	}
	
	return result
}

type PerformanceBaselines struct {
	Baselines map[string]PerformanceBaseline `json:"baselines"`
	UpdatedAt time.Time                      `json:"updated_at"`
}

func EstablishBaselines(metrics []TestMetrics) PerformanceBaselines {
	detector := NewRegressionDetector()
	baselines := make(map[string]PerformanceBaseline)
	
	testGroups := make(map[string][]TestMetrics)
	for _, m := range metrics {
		testGroups[m.TestName] = append(testGroups[m.TestName], m)
	}
	
	for testName, testMetrics := range testGroups {
		detector.SetBaseline(testName, testMetrics)
		if baseline, exists := detector.GetBaseline(testName); exists {
			baselines[testName] = baseline
		}
	}
	
	return PerformanceBaselines{
		Baselines: baselines,
		UpdatedAt: time.Now(),
	}
}