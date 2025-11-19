# VN - Vulnerability Navigator 🛡️

A powerful CLI tool for security testing based on OWASP Top 10 vulnerabilities.

[![asciicast demo](https://asciinema.org/a/752737.svg)](https://asciinema.org/a/752737)

## Features

- **SQL Injection Testing**: Comprehensive SQL injection testing with multiple payload types
- **XSS Testing**: Cross-Site Scripting vulnerability detection
- **Security Misconfiguration Testing**: Detect exposed files, missing headers, default credentials, and server misconfigurations
- **Concurrent Testing**: Multi-threaded scanning for improved performance
- **Multiple HTTP Methods**: Support for GET and POST requests
- **Custom Headers**: Add custom headers for authentication and testing
- **Flexible Parameter Testing**: Auto-detect or specify parameters to test
- **Colored Output**: Beautiful, easy-to-read results with risk levels

## Installation

You can download the [binary files on releases page](https://github.com/lanjoni/vn/releases) or even clone the repository with:
```bash
git clone https://github.com/lanjoni/vn.git
cd vn
go build -o vn .
```

## Docker

You can also build and run `vn` using Docker. This is a convenient way to run the tool without installing Go locally.

### Build the Image

First, build the Docker image:

```bash
docker build -t vn-cli .
```

### Run Commands

Once the image is built, you can run any `vn` command using the following format:

```bash
docker run --rm vn-cli [command]
```

The `--rm` flag automatically removes the container after the command is executed.

**Examples:**

```bash
# Show the help message
docker run --rm vn-cli --help

# Run a SQL injection scan
docker run --rm vn-cli sqli https://example.com/login.php?id=1
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

### Security Misconfiguration Testing

```bash
# Basic misconfiguration scan
./vn misconfig https://example.com

# Test specific categories
./vn misconfig https://example.com --tests files,headers

# Custom headers and threading
./vn misconfig https://example.com --headers "Authorization: Bearer token" --threads 10

# Extended timeout for slow servers
./vn misconfig https://example.com --timeout 15
```

### Available Commands

- `sqli` - Test for SQL injection vulnerabilities
- `xss` - Test for Cross-Site Scripting vulnerabilities
- `misconfig` - Test for security misconfigurations

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

The project includes a comprehensive testing suite managed via the `Makefile`.

Tests are divided into several categories:

- **Unit Tests**: Fast tests for core logic located in the `internal/` and `cmd/` directories. Run with `make test-unit`.
- **Integration Tests**: Tests that require external services or the local test server. Run with `make test-integration`.
- **E2E Tests**: End-to-end tests that simulate real-world usage of the CLI. Run with `make test-e2e`.

To run the entire test suite, use the `make test` command.

### Vulnerable Test Server

A vulnerable test server is included for testing purposes. To start it, run:

```bash
cd test-server
go run main.go
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

## Security Misconfiguration Detection

The tool tests for various misconfigurations:

- **Sensitive Files**: Exposed configuration files, backups, and environment files
- **Security Headers**: Missing or weak security headers (HSTS, CSP, X-Frame-Options, etc.)
- **Default Credentials**: Common default username/password combinations
- **Server Configuration**: Dangerous HTTP methods, information disclosure, insecure redirects

## OWASP Top 10 Coverage

Currently supports:
- ✅ A01:2021 – Broken Access Control (Partial - SQL Injection)
- ✅ A03:2021 – Injection (SQL Injection, XSS)
- ✅ A05:2021 – Security Misconfiguration

Coming soon:
- 🔄 A02:2021 – Cryptographic Failures
- 🔄 A04:2021 – Insecure Design
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
│   ├── xss.go             # XSS command
│   └── misconfig.go       # Security misconfiguration command
├── internal/scanner/       # Vulnerability scanners
│   ├── sqli.go            # SQL injection scanner
│   ├── xss.go             # XSS scanner
│   └── misconfig.go       # Security misconfiguration scanner
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

### Security Misconfiguration Results
```
🚨 Found 8 security misconfigurations:

[1] Sensitive File Exposed
   URL: http://example.com/.env
   Category: sensitive-files
   Finding: Environment configuration file accessible
   Evidence: HTTP 200 response with configuration data
   Risk Level: High
   Remediation: Remove or restrict access to sensitive files

[2] Missing Security Header
   URL: http://example.com
   Category: headers
   Finding: Missing X-Frame-Options header
   Evidence: Header not present in response
   Risk Level: Medium
   Remediation: Add X-Frame-Options: DENY or SAMEORIGIN header
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

AGPL-3.0 license - see LICENSE file for details. 
