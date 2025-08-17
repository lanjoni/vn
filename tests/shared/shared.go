package shared

import (
	"sync"
	"testing"
	"time"

	"vn/tests/shared/builders"
	"vn/tests/shared/fixtures"
	"vn/tests/shared/testserver"
)

var (
	globalBuildManager       builders.BuildManager
	globalServerPool         testserver.ServerPool
	globalMockProvider       fixtures.MockProvider
	globalResourceManager    *ResourceManager
	globalMetricsCollector   *MetricsCollector
	globalRegressionDetector *RegressionDetector
	initOnce                 sync.Once
)

type TestInfrastructure struct {
	BuildManager       builders.BuildManager
	ServerPool         testserver.ServerPool
	MockProvider       fixtures.MockProvider
	ResourceManager    *ResourceManager
	MetricsCollector   *MetricsCollector
	RegressionDetector *RegressionDetector
}

func GetTestInfrastructure() *TestInfrastructure {
	initOnce.Do(func() {
		globalBuildManager = builders.NewBuildManager()
		globalServerPool = testserver.NewServerPool()
		globalMockProvider = fixtures.NewMockProvider()
		globalResourceManager = NewResourceManager()
		globalMetricsCollector = GetMetricsCollector()
		globalRegressionDetector = NewRegressionDetector()

		// Register critical shared resources
		const buildManagerTimeout = 30 * time.Second
		globalResourceManager.RegisterResource("build-manager", buildManagerTimeout)
		globalResourceManager.RegisterResource("server-pool", 10*time.Second)
	})

	return &TestInfrastructure{
		BuildManager:       globalBuildManager,
		ServerPool:         globalServerPool,
		MockProvider:       globalMockProvider,
		ResourceManager:    globalResourceManager,
		MetricsCollector:   globalMetricsCollector,
		RegressionDetector: globalRegressionDetector,
	}
}

func SetupTest(t *testing.T) *TestInfrastructure {
	t.Helper()

	infra := GetTestInfrastructure()

	t.Cleanup(func() {
		// Note: We don't cleanup global resources here as they're shared
		// across tests. Cleanup happens in TestMain or at package level.
	})

	return infra
}

func CleanupGlobalResources() {
	if globalResourceManager != nil {
		globalResourceManager.Cleanup()
	}
	if globalBuildManager != nil {
		globalBuildManager.Cleanup()
	}
	if globalServerPool != nil {
		globalServerPool.Shutdown()
	}
}
