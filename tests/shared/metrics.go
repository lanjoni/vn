package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TestMetrics struct {
	TestName      string        `json:"test_name"`
	ExecutionTime time.Duration `json:"execution_time"`
	BuildTime     time.Duration `json:"build_time"`
	NetworkTime   time.Duration `json:"network_time"`
	SetupTime     time.Duration `json:"setup_time"`
	TeardownTime  time.Duration `json:"teardown_time"`
	ParallelTests int           `json:"parallel_tests"`
	TotalTests    int           `json:"total_tests"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Success       bool          `json:"success"`
	ErrorMessage  string        `json:"error_message,omitempty"`
}

type MetricsCollector struct {
	metrics []TestMetrics
	mu      sync.RWMutex
	enabled bool
}

var (
	metricsOnce sync.Once
)

func GetMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make([]TestMetrics, 0),
		enabled: os.Getenv("COLLECT_TEST_METRICS") == "true",
	}
}

func (mc *MetricsCollector) IsEnabled() bool {
	return mc.enabled
}

func (mc *MetricsCollector) Enable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = true
}

func (mc *MetricsCollector) Disable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = false
}

func (mc *MetricsCollector) RecordMetrics(metrics TestMetrics) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = append(mc.metrics, metrics)
}

func (mc *MetricsCollector) GetMetrics() []TestMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make([]TestMetrics, len(mc.metrics))
	copy(result, mc.metrics)
	return result
}

func (mc *MetricsCollector) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = mc.metrics[:0]
}

func (mc *MetricsCollector) SaveToFile(filename string) error {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(mc.metrics); err != nil {
		return fmt.Errorf("failed to encode metrics: %w", err)
	}

	return nil
}

func (mc *MetricsCollector) LoadFromFile(filename string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&mc.metrics); err != nil {
		return fmt.Errorf("failed to decode metrics: %w", err)
	}

	return nil
}

type TestTimer struct {
	testName    string
	startTime   time.Time
	buildStart  time.Time
	buildEnd    time.Time
	networkTime time.Duration
	setupStart  time.Time
	setupEnd    time.Time
	success     bool
	errorMsg    string
	collector   *MetricsCollector
}

func StartTestTimer(testName string) *TestTimer {
	return &TestTimer{
		testName:  testName,
		startTime: time.Now(),
		collector: GetMetricsCollector(),
	}
}

func (tt *TestTimer) StartBuild() {
	tt.buildStart = time.Now()
}

func (tt *TestTimer) EndBuild() {
	tt.buildEnd = time.Now()
}

func (tt *TestTimer) StartSetup() {
	tt.setupStart = time.Now()
}

func (tt *TestTimer) EndSetup() {
	tt.setupEnd = time.Now()
}

func (tt *TestTimer) AddNetworkTime(duration time.Duration) {
	tt.networkTime += duration
}

func (tt *TestTimer) SetSuccess(success bool) {
	tt.success = success
}

func (tt *TestTimer) SetError(err error) {
	tt.success = false
	if err != nil {
		tt.errorMsg = err.Error()
	}
}

func (tt *TestTimer) Finish() TestMetrics {
	endTime := time.Now()
	
	var buildTime time.Duration
	if !tt.buildEnd.IsZero() && !tt.buildStart.IsZero() {
		buildTime = tt.buildEnd.Sub(tt.buildStart)
	}
	
	var setupTime time.Duration
	if !tt.setupEnd.IsZero() && !tt.setupStart.IsZero() {
		setupTime = tt.setupEnd.Sub(tt.setupStart)
	}

	metrics := TestMetrics{
		TestName:      tt.testName,
		ExecutionTime: endTime.Sub(tt.startTime),
		BuildTime:     buildTime,
		NetworkTime:   tt.networkTime,
		SetupTime:     setupTime,
		StartTime:     tt.startTime,
		EndTime:       endTime,
		Success:       tt.success,
		ErrorMessage:  tt.errorMsg,
	}

	tt.collector.RecordMetrics(metrics)
	return metrics
}