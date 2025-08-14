# Shared Test Infrastructure

This package provides shared utilities for optimizing test performance across the VN project.

## Components

### BuildManager
Manages CLI binary builds with caching to avoid redundant compilation.

```go
import "vn/tests/shared"

func TestExample(t *testing.T) {
    infra := shared.SetupTest(t)
    
    // Get cached binary (builds once per test session)
    binaryPath, err := infra.BuildManager.GetBinary("vn")
    if err != nil {
        t.Fatalf("Failed to get binary: %v", err)
    }
    
    // Use binary in tests...
}
```

### ServerPool
Manages HTTP test servers with reuse capabilities.

```go
import (
    "net/http"
    "vn/tests/shared"
    "vn/tests/shared/testserver"
)

func TestServerExample(t *testing.T) {
    infra := shared.SetupTest(t)
    
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    })
    
    config := testserver.ServerConfig{
        Handler:    handler,
        ConfigName: "my-test-server",
    }
    
    server, err := infra.ServerPool.GetServer(config)
    if err != nil {
        t.Fatalf("Failed to get server: %v", err)
    }
    defer infra.ServerPool.ReleaseServer(server)
    
    // Use server.URL in tests...
}
```

### MockProvider
Provides mock servers to replace external service dependencies.

```go
import (
    "vn/tests/shared"
    "vn/tests/shared/fixtures"
)

func TestMockExample(t *testing.T) {
    infra := shared.SetupTest(t)
    
    // Create httpbin.org mock
    mockServer := infra.MockProvider.CreateHTTPBinMock()
    defer mockServer.Close()
    
    // Use mockServer.URL instead of httpbin.org
    
    // Or create custom mock responses
    responses := map[string]fixtures.MockResponse{
        "/api/test": {
            StatusCode: 200,
            Headers:    map[string]string{"Content-Type": "application/json"},
            Body:       `{"status": "ok"}`,
        },
    }
    
    customMock := infra.MockProvider.CreateMockServer(responses)
    defer customMock.Close()
}
```

## Global Cleanup

For test packages that need to clean up global resources, add this to your test main:

```go
func TestMain(m *testing.M) {
    code := m.Run()
    shared.CleanupGlobalResources()
    os.Exit(code)
}
```

## Performance Benefits

- **Binary Caching**: Builds CLI binary once per test session instead of per test
- **Server Reuse**: Reuses HTTP servers across tests with same configuration
- **Mock Services**: Eliminates external network dependencies
- **Resource Pooling**: Efficient resource management and cleanup