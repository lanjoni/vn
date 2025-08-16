# Test Categories and Execution

This document describes the test categorization system and how to run different types of tests.

## Test Categories

### Fast Tests (`fast` build tag)
- **Purpose**: Quick unit tests and tests with minimal dependencies
- **Duration**: < 5 seconds total
- **Dependencies**: In-memory mocks, no external services
- **Examples**: 
  - Unit tests for shared utilities
  - Mock provider tests
  - Build manager tests
  - Resource manager tests

### Integration Tests (`integration` build tag)
- **Purpose**: Medium-speed tests using local test servers
- **Duration**: < 30 seconds total
- **Dependencies**: Local HTTP test servers, shared infrastructure
- **Examples**:
  - CLI integration tests
  - Workflow tests with test servers
  - Shared infrastructure integration tests

### E2E Tests (`e2e` build tag)
- **Purpose**: Full end-to-end testing with comprehensive scenarios
- **Duration**: < 60 seconds total
- **Dependencies**: Complete test environments, full feature workflows
- **Examples**:
  - Complete misconfiguration scanning workflows
  - Full feature integration tests

### Unit Tests (no build tag)
- **Purpose**: Fast unit tests for core functionality
- **Duration**: < 2 seconds total
- **Dependencies**: None (pure unit tests)
- **Examples**:
  - Scanner logic tests
  - Core algorithm tests
  - Utility function tests

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Only Fast Tests
```bash
go test -tags=fast ./...
```

### Run Only Integration Tests
```bash
go test -tags=integration ./...
```

### Run Only E2E Tests
```bash
go test -tags=e2e ./...
```

### Run Fast + Integration Tests
```bash
go test -tags="fast,integration" ./...
```

### Run All Test Categories
```bash
go test -tags="fast,integration,e2e" ./...
```

### Run Tests in Short Mode (skips slow tests)
```bash
go test -short ./...
```

## Test Performance Targets

| Category | Target Duration | Actual Duration |
|----------|----------------|-----------------|
| Unit Tests | < 2s | TBD |
| Fast Tests | < 5s | TBD |
| Integration Tests | < 30s | TBD |
| E2E Tests | < 60s | TBD |
| All Tests | < 2m | TBD |

## CI/CD Integration

### Development Workflow
1. **Pre-commit**: Run fast tests only
2. **Pull Request**: Run fast + integration tests
3. **Main Branch**: Run all test categories

### Example CI Configuration
```yaml
# Fast feedback loop
fast-tests:
  run: go test -tags=fast ./...

# Comprehensive testing
full-tests:
  run: go test -tags="fast,integration,e2e" ./...
```

## Test Organization

```
tests/
├── integration/     # Integration tests (integration tag)
├── e2e/            # End-to-end tests (e2e tag)
└── shared/         # Shared test utilities
    ├── builders/   # Build management (fast tag)
    ├── fixtures/   # Mock providers (fast tag)
    └── testserver/ # Server pool (fast tag)

internal/
└── scanner/        # Unit tests (no tag)
```

## Test Scripts

Convenient execution scripts are available in the `scripts/` directory:

### Execution Scripts
- `scripts/test-fast.sh` - Run fast tests for quick feedback
- `scripts/test-integration.sh` - Run integration tests with local servers
- `scripts/test-e2e.sh` - Run complete end-to-end tests
- `scripts/test-ci.sh` - CI-optimized test execution

### Performance Monitoring
- `scripts/monitor-test-performance.sh` - Monitor test performance against targets
- `make test-performance` - Run performance monitoring via Makefile

### Usage Examples
```bash
# Quick development feedback
./scripts/test-fast.sh

# Monitor performance
./scripts/monitor-test-performance.sh

# CI testing
./scripts/test-ci.sh
```

## Performance Monitoring

The project includes automated performance monitoring to ensure tests remain fast:

```bash
# Check all test categories against performance targets
make test-performance

# Individual performance checks
time make test-unit        # Should complete in < 2s
time make test-fast        # Should complete in < 5s
time make test-integration # Should complete in < 30s
time make test-e2e        # Should complete in < 60s
```

## Best Practices

1. **Tag Selection**: Choose the most restrictive tag that still allows proper testing
2. **Test Isolation**: Ensure tests can run independently within their category
3. **Resource Management**: Use shared infrastructure for integration/e2e tests
4. **Performance Monitoring**: Track test execution times to prevent regression
5. **Parallel Execution**: Use `t.Parallel()` for independent tests within categories
6. **Script Usage**: Use provided scripts for consistent test execution
7. **CI Integration**: Follow the example CI configuration for optimal pipeline performance