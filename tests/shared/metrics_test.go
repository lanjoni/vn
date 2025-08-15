package shared

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	collector := &MetricsCollector{
		metrics: make([]TestMetrics, 0),
		enabled: true,
	}

	testMetrics := TestMetrics{
		TestName:      "TestExample",
		ExecutionTime: 100 * time.Millisecond,
		BuildTime:     50 * time.Millisecond,
		NetworkTime:   25 * time.Millisecond,
		Success:       true,
	}

	collector.RecordMetrics(testMetrics)

	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].TestName != "TestExample" {
		t.Errorf("Expected test name 'TestExample', got '%s'", metrics[0].TestName)
	}
}

func TestMetricsCollectorDisabled(t *testing.T) {
	collector := &MetricsCollector{
		metrics: make([]TestMetrics, 0),
		enabled: false,
	}

	testMetrics := TestMetrics{
		TestName: "TestExample",
		Success:  true,
	}

	collector.RecordMetrics(testMetrics)

	metrics := collector.GetMetrics()
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics when disabled, got %d", len(metrics))
	}
}

func TestTestTimer(t *testing.T) {
	timer := StartTestTimer("TestTimerExample")
	
	timer.StartBuild()
	time.Sleep(10 * time.Millisecond)
	timer.EndBuild()
	
	timer.StartSetup()
	time.Sleep(5 * time.Millisecond)
	timer.EndSetup()
	
	timer.AddNetworkTime(15 * time.Millisecond)
	timer.SetSuccess(true)
	
	metrics := timer.Finish()
	
	if metrics.TestName != "TestTimerExample" {
		t.Errorf("Expected test name 'TestTimerExample', got '%s'", metrics.TestName)
	}
	
	if metrics.BuildTime < 10*time.Millisecond {
		t.Errorf("Expected build time >= 10ms, got %v", metrics.BuildTime)
	}
	
	if metrics.SetupTime < 5*time.Millisecond {
		t.Errorf("Expected setup time >= 5ms, got %v", metrics.SetupTime)
	}
	
	if metrics.NetworkTime != 15*time.Millisecond {
		t.Errorf("Expected network time 15ms, got %v", metrics.NetworkTime)
	}
	
	if !metrics.Success {
		t.Error("Expected success to be true")
	}
}

func TestMetricsSaveLoad(t *testing.T) {
	collector := &MetricsCollector{
		metrics: []TestMetrics{
			{
				TestName:      "Test1",
				ExecutionTime: 100 * time.Millisecond,
				Success:       true,
			},
			{
				TestName:      "Test2",
				ExecutionTime: 200 * time.Millisecond,
				Success:       false,
				ErrorMessage:  "test error",
			},
		},
		enabled: true,
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "metrics.json")

	err := collector.SaveToFile(filename)
	if err != nil {
		t.Fatalf("Failed to save metrics: %v", err)
	}

	newCollector := &MetricsCollector{
		metrics: make([]TestMetrics, 0),
		enabled: true,
	}

	err = newCollector.LoadFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load metrics: %v", err)
	}

	loadedMetrics := newCollector.GetMetrics()
	if len(loadedMetrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(loadedMetrics))
	}

	if loadedMetrics[0].TestName != "Test1" {
		t.Errorf("Expected first test name 'Test1', got '%s'", loadedMetrics[0].TestName)
	}

	if loadedMetrics[1].ErrorMessage != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", loadedMetrics[1].ErrorMessage)
	}
}

func TestGetMetricsCollectorEnvironment(t *testing.T) {
	originalValue := os.Getenv("COLLECT_TEST_METRICS")
	defer os.Setenv("COLLECT_TEST_METRICS", originalValue)

	os.Setenv("COLLECT_TEST_METRICS", "true")
	
	collector := GetMetricsCollector()
	if !collector.IsEnabled() {
		t.Error("Expected metrics collector to be enabled when COLLECT_TEST_METRICS=true")
	}
}