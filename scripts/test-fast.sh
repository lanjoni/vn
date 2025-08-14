#!/bin/bash
# Fast test execution script
# Runs unit tests and fast tests with mocks

set -e

echo "🚀 Running Fast Tests..."
echo "========================"

echo "📦 Running unit tests..."
go test -v -race ./cmd ./internal/scanner

echo "⚡ Running fast tests..."
go test -v -race -tags=fast ./tests/shared/...

echo "✅ Fast tests completed successfully!"
echo "⏱️  Total execution time: $(date)"