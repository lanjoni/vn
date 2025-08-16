package shared

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	p95 = 0.95
)

type PerformanceBaseline struct {
	TestName            string        `json:"test_name"`
	MedianExecutionTime time.Duration `json:"median_execution_time"`
	P95ExecutionTime    time.Duration `json:"p95_execution_time"`
	MedianBuildTime     time.Duration `json:"median_build_time"`
	MedianNetworkTime   time.Duration `json:"median_network_time"`
	SampleCount         int           `json:"sample_count"`
	LastUpdated         time.Time     `json:"last_updated"`
}

type RegressionDetector struct {
	baselines map[string]PerformanceBaseline
}

func NewRegressionDetector() *RegressionDetector {
	return &RegressionDetector{
		baselines: make(map[string]PerformanceBaseline),
	}
}

func (rd *RegressionDetector) SetBaseline(testName string, metrics []TestMetrics) {
	if len(metrics) == 0 {
		return
	}

	successfulMetrics := make([]TestMetrics, 0, len(metrics))
	for _, m := range metrics {
		if m.Success {
			successfulMetrics = append(successfulMetrics, m)
		}
	}

	if len(successfulMetrics) == 0 {
		return
	}

	baseline := PerformanceBaseline{
		TestName: testName,
		MedianExecutionTime: calculateMedianDuration(successfulMetrics,
			func(m TestMetrics) time.Duration { return m.ExecutionTime }),
		P95ExecutionTime: calculateP95Duration(successfulMetrics,
			func(m TestMetrics) time.Duration { return m.ExecutionTime }),
		MedianBuildTime: calculateMedianDuration(successfulMetrics,
			func(m TestMetrics) time.Duration { return m.BuildTime }),
		MedianNetworkTime: calculateMedianDuration(successfulMetrics,
			func(m TestMetrics) time.Duration { return m.NetworkTime }),
		SampleCount: len(successfulMetrics),
		LastUpdated: time.Now(),
	}

	rd.baselines[testName] = baseline
}

func (rd *RegressionDetector) GetBaseline(testName string) (PerformanceBaseline, bool) {
	baseline, exists := rd.baselines[testName]
	return baseline, exists
}

func (rd *RegressionDetector) DetectRegression(testName string, currentMetrics TestMetrics,
	thresholdPercent float64) (*RegressionResult, error) {
	baseline, exists := rd.baselines[testName]
	if !exists {
		return nil, fmt.Errorf("no baseline found for test: %s", testName)
	}

	if !currentMetrics.Success {
		return &RegressionResult{
			TestName:  testName,
			Regressed: true,
			Reason:    "Test failed",
			Baseline:  baseline,
			Current:   currentMetrics,
		}, nil
	}

	result := &RegressionResult{
		TestName: testName,
		Baseline: baseline,
		Current:  currentMetrics,
	}

	executionRegression := calculateRegressionPercent(baseline.MedianExecutionTime, currentMetrics.ExecutionTime)
	buildRegression := calculateRegressionPercent(baseline.MedianBuildTime, currentMetrics.BuildTime)
	networkRegression := calculateRegressionPercent(baseline.MedianNetworkTime, currentMetrics.NetworkTime)

	if executionRegression > thresholdPercent {
		result.Regressed = true
		result.Reason = fmt.Sprintf("Execution time regressed by %.1f%% (baseline: %v, current: %v)",
			executionRegression, baseline.MedianExecutionTime, currentMetrics.ExecutionTime)
		result.ExecutionRegression = executionRegression
	}

	if buildRegression > thresholdPercent {
		result.Regressed = true
		if result.Reason != "" {
			result.Reason += "; "
		}
		result.Reason += fmt.Sprintf("Build time regressed by %.1f%% (baseline: %v, current: %v)",
			buildRegression, baseline.MedianBuildTime, currentMetrics.BuildTime)
		result.BuildRegression = buildRegression
	}

	if networkRegression > thresholdPercent {
		result.Regressed = true
		if result.Reason != "" {
			result.Reason += "; "
		}
		result.Reason += fmt.Sprintf("Network time regressed by %.1f%% (baseline: %v, current: %v)",
			networkRegression, baseline.MedianNetworkTime, currentMetrics.NetworkTime)
		result.NetworkRegression = networkRegression
	}

	return result, nil
}

type RegressionResult struct {
	TestName            string              `json:"test_name"`
	Regressed           bool                `json:"regressed"`
	Reason              string              `json:"reason"`
	ExecutionRegression float64             `json:"execution_regression"`
	BuildRegression     float64             `json:"build_regression"`
	NetworkRegression   float64             `json:"network_regression"`
	Baseline            PerformanceBaseline `json:"baseline"`
	Current             TestMetrics         `json:"current"`
}

func calculateMedianDuration(metrics []TestMetrics, extractor func(TestMetrics) time.Duration) time.Duration {
	if len(metrics) == 0 {
		return 0
	}

	durations := make([]time.Duration, len(metrics))
	for i, m := range metrics {
		durations[i] = extractor(m)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	n := len(durations)
	if n%2 == 0 {
		return (durations[n/2-1] + durations[n/2]) / 2
	}
	return durations[n/2]
}

func calculateP95Duration(metrics []TestMetrics, extractor func(TestMetrics) time.Duration) time.Duration {
	if len(metrics) == 0 {
		return 0
	}

	durations := make([]time.Duration, len(metrics))
	for i, m := range metrics {
		durations[i] = extractor(m)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	index := int(math.Ceil(p95*float64(len(durations)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}

func calculateRegressionPercent(baseline, current time.Duration) float64 {
	if baseline == 0 {
		return 0
	}

	return ((float64(current) - float64(baseline)) / float64(baseline)) * 100
}
