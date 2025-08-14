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
	globalBuildManager   builders.BuildManager
	globalServerPool     testserver.ServerPool
	globalMockProvider   fixtures.MockProvider
	globalResourceManager *ResourceManager
	initOnce             sync.Once
)

type TestInfrastructure struct {
	BuildManager     builders.BuildManager
	ServerPool       testserver.ServerPool
	MockProvider     fixtures.MockProvider
	ResourceManager  *ResourceManager
}

func GetTestInfrastructure() *TestInfrastructure {
	initOnce.Do(func() {
		globalBuildManager = builders.NewBuildManager()
		globalServerPool = testserver.NewServerPool()
		globalMockProvider = fixtures.NewMockProvider()
		globalResourceManager = NewResourceManager()
		
		// Register critical shared resources
		globalResourceManager.RegisterResource("build-manager", 30*time.Second)
		globalResourceManager.RegisterResource("server-pool", 10*time.Second)
	})
	
	return &TestInfrastructure{
		BuildManager:    globalBuildManager,
		ServerPool:      globalServerPool,
		MockProvider:    globalMockProvider,
		ResourceManager: globalResourceManager,
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