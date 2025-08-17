#!/bin/bash
# Integration test execution script
# Runs integration tests with local test servers

set -e

echo "🔧 Running Integration Tests..."
echo "==============================="

echo "📦 Running unit tests..."
go test -v -race ./cmd ./internal/scanner

echo "⚡ Running fast tests..."
go test -v -race -tags=fast ./tests/shared/...

echo "🌐 Running integration tests..."
go test -v -tags=integration ./tests/integration/... ./tests/shared/...

echo "✅ Integration tests completed successfully!"
echo "⏱️  Total execution time: $(date)"