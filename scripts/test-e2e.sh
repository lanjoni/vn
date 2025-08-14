#!/bin/bash
# End-to-end test execution script
# Runs full E2E tests with complete workflows

set -e

echo "🎯 Running End-to-End Tests..."
echo "=============================="

echo "📦 Running unit tests..."
go test -v -race ./cmd ./internal/scanner

echo "⚡ Running fast tests..."
go test -v -race -tags=fast ./tests/shared/...

echo "🌐 Running integration tests..."
go test -v -tags=integration ./tests/integration/... ./tests/shared/...

echo "🔄 Running E2E tests..."
go test -v -tags=e2e ./tests/e2e/...

echo "✅ All tests completed successfully!"
echo "⏱️  Total execution time: $(date)"