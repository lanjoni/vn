# Test Performance Troubleshooting Guide

This guide helps diagnose and resolve test performance issues in the vulnerability scanner project.

## Quick Diagnostics

### 1. Run Performance Measurement

```bash
./scripts/measure-test-performance.sh
```

Expected results:
- Unit tests: < 15 seconds
- Integration tests: < 5 seconds  
- E2E tests: < 2 seconds
- Total time: < 20 seconds

### 2. Check Test Status

```bash
# Run all tests with timing
time go test -v ./...

# Run specific test categories
time go test -v -tags=integration ./tests/integration
time go test -v -tags=e2e ./tests/e2e
```

### 3. Verify Optimizations Are Active

```bash
# Check if shared infrastructure is working
go test -v ./tests/shared -run TestBuildManager
go test -v ./tests/shared -run TestServerPool
go test -v ./tests/shared -run TestMockProvider
```

## Common Performance Issues

### Issue 1: Tests Taking Too Long

**Symptoms:**
- Total test time > 2 minutes
- Individual tests timing out
- CI pipeline failures due to timeouts

**Diagnosis:**
```bash
# Run with verbose output to identify slow tests
go test -v -tags=integration ./tests/integration

# Check for external service calls
grep -r "httpbin.org" tests/
grep -r "time.Sleep" tests/
```

**Solutions:**
1. **Check timeout configurations:**
   ```bash
   # Look for excessive timeouts
   grep -r "time.Second" tests/ | grep -v "1.*time.Second"
   ```

2. **Verify mock services are used:**
   ```bash
   # Should return no results if properly mocked
   grep -r "httpbin.org" tests/integration/
   ```

3. **Enable parallel execution:**
   ```bash
   # Check for missing t.Parallel() calls
   grep -L "t.Parallel()" tests/integration/*_test.go
   ```

### Issue 2: Build Manager Not Working

**Symptoms:**
- Multiple CLI builds during test execution
- Build-related test failures
- Excessive build times

**Diagnosis:**
```bash
# Test build manager directly
go test -v ./tests/shared -run TestBuildManager

# Check for build errors
go build -o /tmp/test-binary .
```

**Solutions:**
1. **Verify build manager initialization:**
   ```go
   // In test files, ensure this pattern is used:
   binaryPath, err := getSharedBuildManager().BuildOnce("vn", "../../.")
   if err != nil {
       t.Fatalf("Failed to build CLI: %v", err)
   }
   ```

2. **Check build cache:**
   ```bash
   # Clear build cache if corrupted
   go clean -cache
   go clean -testcache
   ```

### Issue 3: Server Pool Issues

**Symptoms:**
- Server startup failures
- Port conflicts
- Connection refused errors

**Diagnosis:**
```bash
# Test server pool directly
go test -v ./tests/shared -run TestServerPool

# Check for port conflicts
lsof -i :8080-8090
```

**Solutions:**
1. **Verify server pool usage:**
   ```go
   // Correct pattern in test files:
   serverPool := getSharedServerPool()
   server, err := serverPool.GetServer(config)
   if err != nil {
       t.Fatalf("Failed to get test server: %v", err)
   }
   defer serverPool.ReleaseServer(server)
   ```

2. **Check server configuration:**
   ```bash
   # Look for hardcoded ports
   grep -r ":8080" tests/
   ```

### Issue 4: Race Conditions

**Symptoms:**
- Intermittent test failures
- Different results on different runs
- "race detected" errors

**Diagnosis:**
```bash
# Run tests with race detector
go test -race -v ./tests/integration

# Check for shared state
grep -r "var.*=" tests/ | grep -v "func"
```

**Solutions:**
1. **Add proper synchronization:**
   ```go
   // Use mutexes for shared resources
   var mu sync.Mutex
   mu.Lock()
   defer mu.Unlock()
   ```

2. **Ensure test isolation:**
   ```go
   // Each test should be independent
   func TestExample(t *testing.T) {
       t.Parallel() // Only if truly independent
       // Test implementation
   }
   ```

### Issue 5: Memory Issues

**Symptoms:**
- Out of memory errors
- Slow garbage collection
- Increasing memory usage during tests

**Diagnosis:**
```bash
# Run with memory profiling
go test -memprofile=mem.prof ./tests/integration
go tool pprof mem.prof

# Check for memory leaks
go test -v ./tests/shared -run TestMemoryUsage
```

**Solutions:**
1. **Verify resource cleanup:**
   ```go
   // Always clean up resources
   defer server.Close()
   defer resp.Body.Close()
   ```

2. **Check for goroutine leaks:**
   ```bash
   # Run tests with goroutine leak detection
   go test -v ./tests/integration | grep "goroutine"
   ```

## Performance Monitoring

### Enable Metrics Collection

```bash
export COLLECT_TEST_METRICS=true
go test ./...
```

### Monitor Resource Usage

```bash
# Monitor during test execution
top -pid $(pgrep -f "go test")

# Check disk usage
du -sh /tmp/vn-test-*
```

### Analyze Performance Trends

```bash
# Generate performance report
./scripts/monitor-test-performance.sh > performance-report.txt

# Compare with baseline
diff baseline-performance.txt performance-report.txt
```

## CI/CD Integration Issues

### GitHub Actions Timeouts

**Problem:** Tests timeout in CI but work locally

**Solutions:**
1. **Increase CI timeout:**
   ```yaml
   - name: Run Tests
     run: go test ./...
     timeout-minutes: 5  # Increase from default
   ```

2. **Use test categories:**
   ```yaml
   - name: Fast Tests
     run: go test -short ./...
   
   - name: Integration Tests  
     run: go test -tags=integration ./tests/integration
   ```

### Resource Constraints

**Problem:** CI environment has limited resources

**Solutions:**
1. **Reduce parallelism:**
   ```bash
   GOMAXPROCS=2 go test ./...
   ```

2. **Skip resource-intensive tests:**
   ```bash
   go test -short ./...
   ```

## Performance Regression Detection

### Establish Baseline

```bash
# Create performance baseline
./scripts/measure-test-performance.sh > baseline.txt
```

### Monitor for Regressions

```bash
# Compare current performance with baseline
./scripts/measure-test-performance.sh > current.txt
diff baseline.txt current.txt
```

### Automated Alerts

Set up CI to fail on performance regressions:

```yaml
- name: Performance Regression Check
  run: |
    ./scripts/measure-test-performance.sh > current.txt
    if [ -f baseline.txt ]; then
      # Fail if tests take 50% longer than baseline
      python scripts/check-regression.py baseline.txt current.txt --threshold 1.5
    fi
```

## Emergency Procedures

### Complete Performance Reset

If all optimizations fail:

```bash
# 1. Clean everything
go clean -cache -testcache -modcache

# 2. Rebuild from scratch
go mod download
go build .

# 3. Run tests sequentially
go test -p 1 ./...

# 4. Re-enable optimizations gradually
go test -v ./tests/shared  # Test infrastructure
go test -v -tags=integration ./tests/integration  # Integration tests
```

### Disable Optimizations Temporarily

```bash
# Run without shared infrastructure
export DISABLE_SHARED_INFRASTRUCTURE=true
go test ./...

# Run without parallel execution
export DISABLE_PARALLEL_TESTS=true
go test ./...
```

## Getting Help

### Debug Information to Collect

When reporting performance issues, include:

1. **System information:**
   ```bash
   go version
   uname -a
   free -h  # Linux
   system_profiler SPHardwareDataType  # macOS
   ```

2. **Test output:**
   ```bash
   go test -v ./... 2>&1 | tee test-output.log
   ```

3. **Performance measurements:**
   ```bash
   ./scripts/measure-test-performance.sh > performance.log
   ```

4. **Resource usage:**
   ```bash
   # During test execution
   ps aux | grep "go test"
   netstat -tulpn | grep :808  # Check port usage
   ```

### Contact Information

- Create an issue in the project repository
- Include all debug information listed above
- Specify your environment (OS, Go version, hardware)
- Describe the expected vs actual performance
