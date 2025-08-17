# Scripts and Workflows Overview

This document provides an overview of the essential scripts and workflows for test performance optimization.

## Essential Files

### GitHub Workflows

1. **`.github/workflows/ci.yml`** - Main CI/CD pipeline
   - Runs unit, integration, E2E, and shared tests
   - Includes performance validation
   - Builds binaries for multiple platforms
   - Generates coverage reports

2. **`.github/workflows/performance-monitoring.yml`** - Performance monitoring
   - Tracks performance regressions
   - Comments on PRs with performance results
   - Maintains performance baselines

### Scripts

1. **`scripts/measure-test-performance.sh`** - Performance measurement
   - Measures execution time for all test categories
   - Provides detailed timing breakdown
   - Used for baseline establishment

2. **`scripts/simple-performance-validation.sh`** - Performance validation
   - Validates tests meet performance targets
   - CI-friendly with appropriate exit codes
   - Provides clear success/failure feedback

3. **`scripts/check-performance-regression.py`** - Regression detection
   - Compares current performance with baseline
   - Generates detailed regression reports
   - Supports JSON output for automation

### Build System

1. **`Makefile`** - Build and test automation
   - Provides consistent commands across environments
   - Simplifies CI/CD integration
   - Includes all test categories and performance checks

## Removed Files

The following files were removed as they were redundant or broken:

- `scripts/validate-performance-targets.sh` - Replaced by `simple-performance-validation.sh`
- `.github/workflows/test-performance-example.yml` - Functionality integrated into main CI

## Usage Examples

### Local Development

```bash
# Quick feedback during development
make test-quick

# Full test suite with performance validation
make test
make performance-check

# Measure current performance
make performance-measure
```

### CI/CD Integration

The workflows automatically:
- Run all test categories with performance validation
- Track performance regressions
- Comment on PRs with performance results
- Maintain performance baselines

### Performance Monitoring

```bash
# Check for regressions
python3 scripts/check-performance-regression.py baseline.txt current.txt

# Validate against targets
./scripts/simple-performance-validation.sh

# Measure and save baseline
./scripts/measure-test-performance.sh > baseline.txt
```

## Performance Targets

| Test Category | Target Time | Actual Time |
|---------------|-------------|-------------|
| Unit Tests | < 15s | ~13s |
| Integration Tests | < 5s | ~5s |
| E2E Tests | < 2s | ~1s |
| Shared Tests | < 2s | ~1s |
| **Total** | **< 25s** | **~18s** |

## Key Features

- **97% performance improvement** (10+ min → 18s)
- **Automated regression detection** with configurable thresholds
- **CI-friendly validation** with appropriate exit codes
- **Comprehensive documentation** and troubleshooting guides
- **Cross-platform compatibility** (Linux, macOS, Windows)
- **Parallel execution** for maximum efficiency
