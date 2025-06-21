# Testing Structure

This document explains how tests are organized in the VN project following Go best practices.

## 🏗️ Go Testing Conventions

### **1. Unit Tests (Go Standard)**
- **Location**: Same directory as source code with `_test.go` suffix
- **Package**: Uses `package foo` (internal) or `package foo_test` (external)
- **Purpose**: Test individual functions/methods in isolation
- **Examples**:
  - `internal/scanner/sqli_test.go` - Tests SQL injection scanner
  - `internal/scanner/xss_test.go` - Tests XSS scanner
  - `cmd/root_test.go` - Tests CLI root command

### **2. Integration Tests**
- **Location**: `tests/integration/` directory
- **Package**: Uses `package integration` 
- **Purpose**: Test full system behavior, CLI integration, server interaction
- **Examples**:
  - `tests/integration/cli_test.go` - Full CLI testing
  - `tests/integration/server_test.go` - Server behavior testing

### **3. Test Data**
- **Location**: `testdata/` directory
- **Purpose**: Store test fixtures, payloads, expected responses
- **Auto-ignored**: Go tooling automatically ignores `testdata` directories
- **Examples**:
  - `testdata/payloads/sqli_payloads.json` - SQL injection test payloads
  - `testdata/responses/vulnerable_responses.json` - Expected server responses

### **4. Test Utilities**
- **Location**: `internal/testutil/` directory
- **Purpose**: Shared test helpers and utilities
- **Examples**:
  - Mock servers, assertion helpers, test setup functions

## 📊 Test Categories

### **Unit Tests**
```bash
# Run all unit tests
make test-unit
# or
go test -v ./internal/... ./cmd/... ./test-server/...
```

### **Integration Tests**
```bash
# Run integration tests
make test-integration
# or
go test -v ./tests/integration/...
```

### **All Tests**
```bash
# Run everything
make test
# or
go test -v ./...
```

## 🎯 Test Organization Benefits

### **✅ Pros of This Structure:**

1. **Go Idiomatic**: Unit tests follow Go convention (same directory)
2. **Clear Separation**: Integration tests separate from unit tests
3. **Shared Utilities**: Common test helpers in `testutil`
4. **Test Data**: Organized fixtures in `testdata`
5. **CI/CD Friendly**: Different test types can run independently
6. **Performance**: Unit tests run fast, integration tests run when needed

### **🚀 Common Go Testing Patterns:**

1. **Table-Driven Tests**: Most Go tests use this pattern
2. **Test Helpers**: Functions with `t.Helper()` for better error reporting
3. **Setup/Teardown**: Using defer for cleanup
4. **Parallel Tests**: Using `t.Parallel()` for concurrent test execution
5. **Build Tags**: Using `//go:build integration` for conditional compilation

## 🛠️ Development Workflow

### **Quick Development Check:**
```bash
make check  # fmt + vet + lint + unit tests
```

### **Pre-Commit:**
```bash
make pre-commit  # Full check before committing
```

### **CI Pipeline:**
```bash
make ci  # Complete CI pipeline locally
```

## 📁 Directory Structure

```
vn/
├── cmd/                     # CLI commands
│   ├── root.go
│   ├── root_test.go        # ✅ Unit tests (same dir)
│   ├── sqli.go
│   └── xss.go
├── internal/
│   ├── scanner/            # Core scanners
│   │   ├── sqli.go
│   │   ├── sqli_test.go    # ✅ Unit tests (same dir)
│   │   ├── xss.go
│   │   └── xss_test.go     # ✅ Unit tests (same dir)
│   └── testutil/           # 🛠️ Test utilities
│       └── helpers.go      # Shared test helpers
├── test-server/            # Vulnerable test server
│   ├── main.go
│   └── main_test.go        # ✅ Unit tests (same dir)
├── tests/                  # 🧪 Integration tests
│   ├── integration/
│   │   ├── cli_test.go     # CLI integration tests
│   │   └── server_test.go  # Server integration tests
│   └── README.md           # This file
├── testdata/               # 📊 Test fixtures
│   ├── payloads/
│   │   └── sqli_payloads.json
│   └── responses/
│       └── vulnerable_responses.json
└── ...
```

## 🎓 Go Testing Best Practices

1. **Keep unit tests close to source** - Go convention
2. **Use table-driven tests** - Clean and scalable
3. **Test behavior, not implementation** - Focus on outcomes
4. **Use meaningful test names** - `TestFunctionName_Condition_ExpectedResult`
5. **Parallel where possible** - Use `t.Parallel()` for independent tests
6. **Clean up resources** - Use `defer` for cleanup
7. **Use test utilities** - Share common setup/teardown logic
8. **Separate unit from integration** - Different purposes, different speeds

This structure makes development happier by providing clear organization while following Go idioms! 🎉 