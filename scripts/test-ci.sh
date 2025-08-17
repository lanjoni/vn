#!/bin/bash
# CI test execution script
# Optimized for CI environments with appropriate timeouts and parallelism

set -e

echo "🤖 Running CI Tests..."
echo "====================="

# Set CI-optimized environment variables
export GOMAXPROCS=${GOMAXPROCS:-4}
export GO_TEST_TIMEOUT=${GO_TEST_TIMEOUT:-10m}

echo "📦 Running unit tests..."
go test -v -race -timeout=${GO_TEST_TIMEOUT} ./cmd ./internal/scanner

echo "⚡ Running fast tests..."
go test -v -race -tags=fast -timeout=${GO_TEST_TIMEOUT} ./tests/shared/...

echo "🌐 Running integration tests..."
go test -v -tags=integration -timeout=${GO_TEST_TIMEOUT} ./tests/integration/... ./tests/shared/...

echo "📊 Running benchmarks..."
go test -bench=. -benchmem -tags=integration -timeout=${GO_TEST_TIMEOUT} ./tests/integration/...

echo "✅ CI tests completed successfully!"
echo "⏱️  Total execution time: $(date)"
echo "🔧 GOMAXPROCS: ${GOMAXPROCS}"
echo "⏰ Timeout: ${GO_TEST_TIMEOUT}"