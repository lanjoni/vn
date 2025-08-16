# VN Test Server - Misconfiguration Testing Extensions

This test server has been extended to support comprehensive security misconfiguration testing scenarios. It provides vulnerable endpoints that simulate real-world misconfigurations for testing the VN misconfiguration scanner.

## Misconfiguration Test Endpoints

### Sensitive Files Testing (Requirement 1.1)

**Exposed Configuration Files:**
- `/.env` - Environment configuration with database passwords and API keys
- `/config.php` - PHP configuration file with database credentials
- `/web.config` - IIS configuration with connection strings
- `/backup.sql` - Database backup file with user data
- `/robots.txt` - Robots exclusion file revealing hidden directories

**Backup Files:**
- `/config.bak` - Backup configuration file
- `/app.config.old` - Old application configuration
- `/database.sql.backup` - Database backup with sensitive data

**Directory Listings:**
- `/uploads/` - Directory listing showing exposed files

### Security Headers Testing (Requirement 2.1)

**Missing Security Headers:**
- `/insecure-headers` - Page missing all security headers
  - Missing: X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, HSTS, CSP
  - Exposes Server header for version disclosure

**Proper Security Headers:**
- `/secure-headers` - Page with all required security headers
  - X-Frame-Options: DENY
  - X-Content-Type-Options: nosniff
  - X-XSS-Protection: 1; mode=block
  - Strict-Transport-Security: max-age=31536000; includeSubDomains
  - Content-Security-Policy: default-src 'self'

### Default Credentials Testing (Requirement 3.1)

**Admin Interfaces:**
- `/admin` - Admin login form accepting default credentials
  - Accepts: admin/admin, root/root, admin/password
- `/admin/login` - Alternative admin login endpoint

**Version Disclosure:**
- `/version-error` - Error page revealing server and technology versions
  - Exposes Apache, PHP, and MySQL versions
- `/default-install` - Default Apache installation page

### Server Configuration Testing (Requirement 4.1)

**Dangerous HTTP Methods:**
- `/methods-test` - Endpoint accepting dangerous HTTP methods
  - PUT: File upload simulation
  - DELETE: Resource deletion simulation
  - TRACE: HTTP trace method (security risk)
  - OPTIONS: Shows all allowed methods in Allow header

**Information Leakage:**
- `/info-leak?file=<filename>` - Error page revealing system paths and database connection strings
  - Exposes file system paths
  - Reveals database connection details
  - Shows application stack traces

## Usage Examples

### Starting the Test Server

```bash
cd test-server
go run main.go
```

The server will start on port 8080 and display all available endpoints.

### Testing Sensitive File Discovery

```bash
# Test for exposed configuration files
curl http://localhost:8080/.env
curl http://localhost:8080/config.php
curl http://localhost:8080/web.config

# Test for backup files
curl http://localhost:8080/config.bak
curl http://localhost:8080/app.config.old
curl http://localhost:8080/database.sql.backup

# Test for directory listings
curl http://localhost:8080/uploads/
```

### Testing Security Headers

```bash
# Check for missing security headers
curl -I http://localhost:8080/insecure-headers

# Verify proper security headers
curl -I http://localhost:8080/secure-headers
```

### Testing Default Credentials

```bash
# Test admin login with default credentials
curl -d "username=admin&password=admin" http://localhost:8080/admin
curl -d "username=root&password=root" http://localhost:8080/admin

# Check version disclosure
curl http://localhost:8080/version-error
```

### Testing Dangerous HTTP Methods

```bash
# Test dangerous HTTP methods
curl -X PUT http://localhost:8080/methods-test
curl -X DELETE http://localhost:8080/methods-test
curl -X TRACE http://localhost:8080/methods-test
curl -X OPTIONS -I http://localhost:8080/methods-test

# Test information leakage
curl "http://localhost:8080/info-leak?file=config.txt"
```

## Integration with Misconfiguration Scanner

These endpoints are designed to be detected by the VN misconfiguration scanner. Each endpoint simulates a specific type of security misconfiguration that the scanner should identify and report.

The test server can be used for:
- Unit testing scanner components
- Integration testing full scanner workflow
- End-to-end testing with known vulnerable configurations
- Performance testing with concurrent requests

## Security Notice

⚠️ **WARNING**: This test server is intentionally vulnerable and should NEVER be deployed in a production environment. It is designed solely for testing security scanning tools in controlled environments.