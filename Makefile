# Makefile for VN - Vulnerability Navigator

.PHONY: help build test test-unit test-integration test-e2e test-shared test-all test-coverage clean benchmark performance-check

# Default target
help:
	@echo "Available targets:"
	@echo "  build              - Build the VN binary"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only"
	@echo "  test-e2e           - Run end-to-end tests only"
	@echo "  test-shared        - Run shared infrastructure tests only"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  benchmark          - Run performance benchmarks"
	@echo "  performance-check  - Validate test performance targets"
	@echo "  clean              - Clean build artifacts and test cache"

# Build targets
build:
	@echo "Building VN binary..."
	go build -v -ldflags="-s -w" -o vn .

# Test targets
test: test-unit test-integration test-e2e test-shared

test-unit:
	@echo "Running unit tests..."
	go test -v -race ./internal/... ./cmd/... -count=1

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/integration -count=1

test-e2e:
	@echo "Running E2E tests..."
	go test -v -tags=e2e ./tests/e2e -count=1

test-shared:
	@echo "Running shared infrastructure tests..."
	go test -v ./tests/shared -count=1

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./internal/... ./cmd/... ./tests/shared
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Performance targets
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./tests/shared

performance-check:
	@echo "Validating test performance targets..."
	@chmod +x scripts/simple-performance-validation.sh
	@./scripts/simple-performance-validation.sh

performance-measure:
	@echo "Measuring test performance..."
	@chmod +x scripts/measure-test-performance.sh
	@./scripts/measure-test-performance.sh

# Utility targets
clean:
	@echo "Cleaning build artifacts and test cache..."
	go clean -cache -testcache -modcache
	rm -f vn coverage.out coverage.html
	rm -f performance-*.txt benchmark-*.txt

# Development workflow targets
test-quick: test-unit test-shared
	@echo "Quick tests completed"

test-ci: test-unit test-shared test-integration
	@echo "CI tests completed"

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# Format and lint
fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run

# Security scan
security:
	@echo "Running security scan..."
	gosec ./...

# Full CI pipeline
ci: deps fmt lint security test performance-check
	@echo "Full CI pipeline completed successfully"
