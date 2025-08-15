package shared

import (
	"testing"
	"time"

	"vn/tests/shared/fixtures"
	"vn/tests/shared/testserver"
)

func BenchmarkBuildManager(b *testing.B) {
	infra := GetTestInfrastructure()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := infra.BuildManager.GetBinary("vn")
		if err != nil {
			b.Fatalf("Failed to get binary: %v", err)
		}
	}
}

func BenchmarkServerPool(b *testing.B) {
	infra := GetTestInfrastructure()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server, err := infra.ServerPool.GetServer(testserver.ServerConfig{
			Port: 0,
		})
		if err != nil {
			b.Fatalf("Failed to get server: %v", err)
		}
		infra.ServerPool.ReleaseServer(server)
	}
}

func BenchmarkMockProvider(b *testing.B) {
	infra := GetTestInfrastructure()
	
	responses := map[string]fixtures.MockResponse{
		"/test": {
			StatusCode: 200,
			Body:       "test response",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := infra.MockProvider.CreateMockServer(responses)
		server.Close()
	}
}

func BenchmarkMetricsCollection(b *testing.B) {
	collector := GetMetricsCollector()
	collector.Enable()
	
	testMetrics := TestMetrics{
		TestName:      "BenchmarkTest",
		ExecutionTime: 100 * time.Millisecond,
		BuildTime:     50 * time.Millisecond,
		NetworkTime:   25 * time.Millisecond,
		Success:       true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordMetrics(testMetrics)
	}
}

func BenchmarkRegressionDetection(b *testing.B) {
	detector := NewRegressionDetector()
	
	baselineMetrics := []TestMetrics{
		{TestName: "BenchmarkTest", ExecutionTime: 100 * time.Millisecond, Success: true},
		{TestName: "BenchmarkTest", ExecutionTime: 110 * time.Millisecond, Success: true},
		{TestName: "BenchmarkTest", ExecutionTime: 90 * time.Millisecond, Success: true},
	}
	detector.SetBaseline("BenchmarkTest", baselineMetrics)
	
	currentMetrics := TestMetrics{
		TestName:      "BenchmarkTest",
		ExecutionTime: 120 * time.Millisecond,
		Success:       true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.DetectRegression("BenchmarkTest", currentMetrics, 20.0)
		if err != nil {
			b.Fatalf("Failed to detect regression: %v", err)
		}
	}
}

func BenchmarkReportGeneration(b *testing.B) {
	metrics := make([]TestMetrics, 100)
	for i := 0; i < 100; i++ {
		metrics[i] = TestMetrics{
			TestName:      "BenchmarkTest",
			ExecutionTime: time.Duration(i+1) * time.Millisecond,
			Success:       i%10 != 0,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator := NewReportGenerator(metrics)
		_ = generator.GenerateReport()
	}
}