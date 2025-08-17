#!/bin/bash
# Test performance monitoring script
# Measures and validates test execution times against targets

set -e

echo "📊 Test Performance Monitor"
echo "=========================="

# Performance targets (in seconds)
UNIT_TARGET=2
FAST_TARGET=5
INTEGRATION_TARGET=30
E2E_TARGET=60
ALL_TARGET=120

# Metrics collection
METRICS_DIR="test-metrics"
METRICS_FILE="$METRICS_DIR/performance-$(date +%Y%m%d-%H%M%S).json"
BASELINE_FILE="$METRICS_DIR/baseline.json"
REPORT_FILE="$METRICS_DIR/latest-report.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to measure execution time with metrics collection
measure_time() {
    local test_name=$1
    shift
    local start_time=$(date +%s)
    
    # Enable metrics collection
    export COLLECT_TEST_METRICS=true
    
    "$@"
    local exit_code=$?
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Collect metrics if available
    if [ -f "test-metrics.json" ]; then
        mkdir -p "$METRICS_DIR"
        mv "test-metrics.json" "$METRICS_FILE"
    fi
    
    echo $duration
    return $exit_code
}

# Function to check performance target
check_target() {
    local actual=$1
    local target=$2
    local test_name=$3
    
    if [ $actual -le $target ]; then
        echo -e "${GREEN}✅ $test_name: ${actual}s (target: ${target}s)${NC}"
        return 0
    else
        echo -e "${RED}❌ $test_name: ${actual}s (target: ${target}s) - EXCEEDED${NC}"
        return 1
    fi
}

# Track overall results
overall_result=0

echo "🔍 Measuring test performance..."
echo

# Unit tests
echo "📦 Running unit tests..."
unit_time=$(measure_time "unit_tests" make test-unit)
check_target $unit_time $UNIT_TARGET "Unit Tests" || overall_result=1

echo

# Fast tests
echo "⚡ Running fast tests..."
fast_time=$(measure_time "fast_tests" make test-fast)
check_target $fast_time $FAST_TARGET "Fast Tests" || overall_result=1

echo

# Integration tests
echo "🌐 Running integration tests..."
integration_time=$(measure_time "integration_tests" make test-integration)
check_target $integration_time $INTEGRATION_TARGET "Integration Tests" || overall_result=1

echo

# E2E tests
echo "🎯 Running E2E tests..."
e2e_time=$(measure_time "e2e_tests" make test-e2e)
check_target $e2e_time $E2E_TARGET "E2E Tests" || overall_result=1

echo

# All tests
echo "🚀 Running all tests..."
all_time=$(measure_time "all_tests" make test)
check_target $all_time $ALL_TARGET "All Tests" || overall_result=1

echo
echo "📈 Performance Summary"
echo "====================="
echo "Unit Tests:        ${unit_time}s / ${UNIT_TARGET}s"
echo "Fast Tests:        ${fast_time}s / ${FAST_TARGET}s"
echo "Integration Tests: ${integration_time}s / ${INTEGRATION_TARGET}s"
echo "E2E Tests:         ${e2e_time}s / ${E2E_TARGET}s"
echo "All Tests:         ${all_time}s / ${ALL_TARGET}s"

# Calculate total improvement
total_measured=$((unit_time + fast_time + integration_time + e2e_time))
echo "Total Measured:    ${total_measured}s"

# Performance recommendations
echo
echo "💡 Performance Recommendations"
echo "=============================="

if [ $unit_time -gt $UNIT_TARGET ]; then
    echo "- Unit tests are slow. Consider reducing test complexity or adding more mocks."
fi

if [ $fast_time -gt $FAST_TARGET ]; then
    echo "- Fast tests are slow. Ensure all external dependencies are mocked."
fi

if [ $integration_time -gt $INTEGRATION_TARGET ]; then
    echo "- Integration tests are slow. Check server startup times and optimize timeouts."
fi

if [ $e2e_time -gt $E2E_TARGET ]; then
    echo "- E2E tests are slow. Consider reducing test scope or improving parallelization."
fi

if [ $overall_result -eq 0 ]; then
    echo -e "${GREEN}🎉 All performance targets met!${NC}"
else
    echo -e "${YELLOW}⚠️  Some performance targets were exceeded. Consider optimization.${NC}"
fi

# Generate performance report if metrics are available
if [ -f "$METRICS_FILE" ]; then
    echo
    echo "📊 Generating performance report..."
    
    # Build the performance tool if it doesn't exist
    if [ ! -f "./vn" ]; then
        go build -o vn .
    fi
    
    # Generate report
    ./vn performance report "$METRICS_FILE" --format text > "$REPORT_FILE"
    echo "Report saved to: $REPORT_FILE"
    
    # Check for regressions if baseline exists
    if [ -f "$BASELINE_FILE" ]; then
        echo
        echo "🔍 Checking for performance regressions..."
        ./vn performance regression "$METRICS_FILE" "$BASELINE_FILE" --threshold 25.0
    else
        echo
        echo "📈 Establishing performance baseline..."
        ./vn performance baseline "$METRICS_FILE" "$BASELINE_FILE"
        echo "Baseline established: $BASELINE_FILE"
    fi
    
    # Validate against targets
    echo
    echo "✅ Validating against performance targets..."
    ./vn performance validate "$METRICS_FILE"
fi

echo
echo "⏱️  Monitoring completed at $(date)"

exit $overall_result