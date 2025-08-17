# CLAUDE - Comprehensive Language And Universal Documentation Engine

This document provides a detailed overview of the `vn.go` project, a Go-based security scanner.

## 1. Project Overview

`vn.go` (Vulnerability Navigator) is a command-line interface (CLI) tool designed for fast and efficient web application security auditing. It is written in Go, which allows for high concurrency and performance. The tool scans for common vulnerabilities based on the OWASP Top 10, including SQL Injection (SQLi), Cross-Site Scripting (XSS), and Security Misconfigurations.

### Core Features

*   **Vulnerability Scanning**: Detects SQLi, XSS, and misconfigurations.
*   **High Performance**: Leverages Go's concurrency to run scans quickly.
*   **Flexible Configuration**: Supports various command-line flags for customizing scans, including HTTP methods, custom headers, and target parameters.
*   **CI/CD Integration**: Includes a robust set of scripts and configurations for continuous integration and performance monitoring.
*   **Comprehensive Testing**: A multi-layered testing strategy ensures code quality and reliability.

## 2. Project Structure

The project is organized into several key directories:

```
/
├── .github/              # CI/CD workflows (GitHub Actions)
├── cmd/                  # CLI command definitions (root, sqli, xss, misconfig)
├── docs/                 # Project documentation
├── internal/             # Core application logic
│   └── scanner/          # Vulnerability scanning implementation
├── scripts/              # Automation and utility scripts
├── test-server/          # A vulnerable server for testing purposes
├── tests/                # E2E and integration tests
├── go.mod                # Go module definition
├── main.go               # Application entry point
└── Makefile              # Build, test, and utility commands
```

## 3. Getting Started

### Prerequisites

*   Go (version 1.18 or later)
*   Git

### Build

To build the application, clone the repository and use the `make build` command:

```bash
git clone <repository-url>
cd vn.go
make build
```

This command compiles the source code and creates an executable file named `vn` in the project's root directory.

## 4. Usage

The tool is operated via subcommands, each targeting a specific type of vulnerability.

### Global Flags

*   `--verbose, -v`: Enables detailed output.
*   `--output, -o`: Sets the output format (e.g., `json`, `xml`).

### Commands

#### a. `sqli` - SQL Injection

Scans for SQL injection vulnerabilities.

```bash
# Scan a GET endpoint
./vn sqli "http://example.com/product?id=1"

# Scan a POST endpoint with data
./vn sqli http://example.com/login --method POST --data "user=admin&pass=123"
```

#### b. `xss` - Cross-Site Scripting

Scans for XSS vulnerabilities.

```bash
# Scan a URL parameter
./vn xss "http://example.com/search?query=<script>alert(1)</script>"
```

#### c. `misconfig` - Security Misconfiguration

Checks for common server and application misconfigurations.

```bash
# Run a full misconfiguration scan
./vn misconfig http://example.com
```

## 5. Development and Testing

The project includes a streamlined testing suite managed via the `Makefile`.

- **Unit Tests (`make test-unit`)**: These are located alongside the source code in `internal/scanner/` and `cmd/`. They are self-contained and test core logic in isolation.
- **End-to-End Tests (`make test-e2e`)**: A small set of tests in `tests/e2e/` that run the compiled binary against a live test server to ensure the commands work as expected from a user's perspective.
- **Integration Tests (`make test-integration`)**: Tests that involve interactions between different components, such as the CLI and the test server.

To run the full suite, simply use `make test`.

### Test Server

A dedicated test server is located in the `test-server/` directory. To run it:

```bash
cd test-server/
go run main.go
```
The server provides vulnerable endpoints to validate the scanner's effectiveness.

## 6. Performance

Performance is a critical aspect of `vn.go`. The project includes scripts and workflows for performance testing and regression analysis.

*   **`make benchmark`**: Runs Go benchmarks.
*   **`scripts/measure-test-performance.sh`**: Measures the execution time of the test suite.
*   **`scripts/check-performance-regression.py`**: Compares current performance against a baseline.

The `.github/workflows/performance-monitoring.yml` file defines a GitHub Action that automatically checks for performance regressions on each commit.

## 7. Continuous Integration (CI)

The CI pipeline is defined in `.github/workflows/ci.yml`. It automates the following tasks:

1.  **Dependency Installation**: `go mod download`
2.  **Code Formatting**: `go fmt`
3.  **Linting**: `golangci-lint run`
4.  **Security Scanning**: `gosec ./...`
5.  **Testing**: `make test-ci`

This ensures that all code merged into the main branch adheres to the project's quality and security standards.
