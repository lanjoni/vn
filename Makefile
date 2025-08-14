# VN - Vulnerability Navigator Makefile

.PHONY: help build test test-unit test-integration test-coverage clean lint fmt vet security run-server benchmark install-tools

# Variables
BINARY_NAME=vn
VERSION?=dev
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Default target
help: ## Display this help message
	@echo "VN - Vulnerability Navigator"
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} .

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe .

test: test-unit test-fast test-integration test-e2e ## Run all tests

test-unit: ## Run unit tests (no build tags)
	@echo "Running unit tests..."
	go test -v -race ./cmd ./internal/scanner

test-fast: ## Run fast tests with mocks and minimal dependencies
	@echo "Running fast tests..."
	go test -v -race -tags=fast ./tests/shared/...

test-integration: ## Run integration tests with local servers
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/integration/... ./tests/shared/...

test-e2e: ## Run end-to-end tests with full workflows
	@echo "Running E2E tests..."
	go test -v -tags=e2e ./tests/e2e/...

test-all-categories: ## Run all test categories (unit, fast, integration, e2e)
	@echo "Running all test categories..."
	go test -v -race -tags="fast,integration,e2e" ./...

test-quick: test-unit test-fast ## Run quick tests for development feedback
	@echo "Quick tests completed!"

test-ci: test-unit test-fast test-integration ## Run CI-appropriate tests (excludes slow e2e)
	@echo "CI tests completed!"

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -tags="fast,integration,e2e" -coverprofile=${COVERAGE_FILE} ./...
	go tool cover -html=${COVERAGE_FILE} -o ${COVERAGE_HTML}
	@echo "Coverage report generated: ${COVERAGE_HTML}"

test-coverage-fast: ## Run fast tests with coverage
	@echo "Running fast tests with coverage..."
	go test -v -race -tags=fast -coverprofile=${COVERAGE_FILE} ./...
	go tool cover -html=${COVERAGE_FILE} -o ${COVERAGE_HTML}
	@echo "Fast test coverage report generated: ${COVERAGE_HTML}"

test-short: ## Run short tests only
	@echo "Running short tests..."
	go test -short -v ./...

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -tags=integration ./tests/integration/...

test-performance: ## Monitor test performance against targets
	@echo "Monitoring test performance..."
	./scripts/monitor-test-performance.sh

lint: ## Run linting
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

security: ## Run security scan
	@echo "Running security scan..."
	gosec ./...

clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME}-*
	rm -f ${COVERAGE_FILE}
	rm -f ${COVERAGE_HTML}
	rm -f vn-test
	rm -f vn-bench

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest

run-server: ## Start the vulnerable test server
	@echo "Starting vulnerable test server on :8080..."
	cd test-server && go run main.go

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

pre-commit: fmt vet lint test-quick ## Run pre-commit checks
	@echo "Pre-commit checks passed!"

ci: deps vet lint test-ci benchmark ## Run CI pipeline locally
	@echo "CI pipeline completed!"

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing ${BINARY_NAME}..."
	go install ${LDFLAGS} .

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t vn:${VERSION} .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it vn:${VERSION}

release-test: ## Test release process
	@echo "Testing release process..."
	goreleaser release --snapshot --clean

# Development helpers
dev-setup: install-tools deps ## Set up development environment
	@echo "Development environment setup complete!"

check: fmt vet lint test-unit ## Quick development check
	@echo "Quick check passed!"

all: clean deps fmt vet lint test build ## Build everything from scratch
	@echo "Build complete!"

# Help target should be first
.DEFAULT_GOAL := help 