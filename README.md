# VN - Vulnerability Navigator 🛡️

A powerful CLI tool for security testing based on OWASP Top 10 vulnerabilities.

## Features

- **SQL Injection Testing**: Comprehensive SQL injection testing with multiple payload types
- **XSS Testing**: Cross-Site Scripting vulnerability detection
- **Concurrent Testing**: Multi-threaded scanning for improved performance
- **Multiple HTTP Methods**: Support for GET and POST requests
- **Custom Headers**: Add custom headers for authentication and testing
- **Flexible Parameter Testing**: Auto-detect or specify parameters to test
- **Colored Output**: Beautiful, easy-to-read results with risk levels

## Installation

```bash
git clone <your-repo>
cd vn
go build -o vn .
```

## Usage

### SQL Injection Testing

```bash
# Basic SQL injection test
./vn sqli https://example.com/login.php?id=1

# Test POST endpoint with form data
./vn sqli https://api.example.com/login --method POST --data "username=admin&password=secret"

# Test specific parameters
./vn sqli https://example.com/search --params "q,category,type"

# Custom headers and threading
./vn sqli https://example.com/api --headers "Authorization: Bearer token" --threads 10
```

### XSS Testing

```bash
# Basic XSS test
./vn xss https://example.com/search?q=test

# Test POST endpoint for XSS
./vn xss https://example.com/comment --method POST --data "comment=test&author=user"

# Test specific parameters
./vn xss https://example.com/profile --params "name,bio,email"
```

### Available Commands

- `sqli` - Test for SQL injection vulnerabilities
- `xss` - Test for Cross-Site Scripting vulnerabilities

### Global Flags

- `--verbose, -v` - Verbose output
- `--output, -o` - Output format (console, json, xml)
- `--config` - Configuration file path

### Command-Specific Flags

- `--method, -m` - HTTP method (GET, POST)
- `--data, -d` - POST data (form-encoded)
- `--params, -p` - Parameters to test (comma-separated)
- `--headers, -H` - Custom headers
- `--timeout, -t` - Request timeout in seconds
- `--threads, -T` - Number of concurrent threads

## Testing

A vulnerable test server is included for testing purposes:

```bash
# Start the test server
cd test-server
go run main.go

# Test against the vulnerable server
./vn sqli http://localhost:8080/?id=1
./vn sqli http://localhost:8080/login --method POST --data "username=admin&password=secret"
```

## SQL Injection Detection

The tool tests for various types of SQL injection:

- **Error-based**: Detects SQL errors in responses
- **Boolean-based**: Tests logical conditions
- **Time-based**: Detects delays in responses
- **Union-based**: Tests UNION SELECT queries
- **NoSQL**: MongoDB and other NoSQL injection patterns

## XSS Detection

The tool tests for multiple XSS types:

- **Reflected XSS**: Payload reflected in response
- **DOM-based XSS**: Client-side script injection
- **Filter Bypass**: Various encoding and bypass techniques

## OWASP Top 10 Coverage

Currently supports:
- ✅ A01:2021 – Broken Access Control (Partial - SQL Injection)
- ✅ A03:2021 – Injection (SQL Injection, XSS)

Coming soon:
- 🔄 A02:2021 – Cryptographic Failures
- 🔄 A04:2021 – Insecure Design
- 🔄 A05:2021 – Security Misconfiguration
- 🔄 A06:2021 – Vulnerable and Outdated Components
- 🔄 A07:2021 – Identification and Authentication Failures
- 🔄 A08:2021 – Software and Data Integrity Failures
- 🔄 A09:2021 – Security Logging and Monitoring Failures
- 🔄 A10:2021 – Server-Side Request Forgery (SSRF)

## Project Structure

```
vn/
├── main.go                 # Entry point
├── cmd/                    # CLI commands
│   ├── root.go            # Root command
│   ├── sqli.go            # SQL injection command
│   └── xss.go             # XSS command
├── internal/scanner/       # Vulnerability scanners
│   ├── sqli.go            # SQL injection scanner
│   └── xss.go             # XSS scanner
├── test-server/           # Vulnerable test server
│   └── main.go
└── README.md
```

## Examples

### SQL Injection Results
```
🚨 Found 15 potential SQL injection vulnerabilities:

[1] SQL Injection Detected
   URL: http://localhost:8080/?id=1
   Parameter: id
   Payload: ' OR '1'='1
   Method: GET
   Evidence: SQL error detected in response
   Risk Level: High
```

### XSS Results
```
🚨 Found 3 potential XSS vulnerabilities:

[1] XSS Vulnerability Detected
   URL: http://example.com/search
   Parameter: q
   Payload: <script>alert('XSS')</script>
   Type: reflected
   Evidence: Payload reflected in response without proper encoding
   Risk Level: High
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add new vulnerability scanners in `internal/scanner/`
4. Add corresponding CLI commands in `cmd/`
5. Update this README
6. Submit a pull request

## Disclaimer

⚠️ **This tool is for educational and authorized security testing purposes only. Do not use against systems you don't own or don't have explicit permission to test.**

## License

MIT License - see LICENSE file for details. 