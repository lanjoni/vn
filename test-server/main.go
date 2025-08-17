package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultUsername = "admin"
	defaultPassword = "admin"
	rootUsername    = "root"
	rootPassword    = "root"
	passwordValue   = "password"
)

func vulnerableEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	id := r.URL.Query().Get("id")
	username := r.URL.Query().Get("username")
	search := r.URL.Query().Get("search")

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err == nil {
			id = r.FormValue("id")
			username = r.FormValue("username")
			search = r.FormValue("search")
		}
	}

	var response strings.Builder
	response.WriteString("<h1>Vulnerable Test Server</h1>\n")

	switch {
	case strings.Contains(id, "UNION") || strings.Contains(username, "UNION") || strings.Contains(search, "UNION"):
		response.WriteString(`<p style="color: red;">Warning: mysql_fetch_array() expects parameter 1 to be resource</p>`)
	case strings.Contains(id, "SLEEP") || strings.Contains(username, "SLEEP") || strings.Contains(search, "SLEEP"):
		response.WriteString(`<p>Query executed successfully</p>`)
	case strings.Contains(id, "'") || strings.Contains(username, "'") || strings.Contains(search, "'"):
		response.WriteString(`<p style="color: red;">MySQL Error: You have an error in your SQL syntax; ` +
			`check the manual that corresponds to your MySQL server version for the right syntax to use near ''' at line 1</p>`)
	default:
		response.WriteString(`<p>Normal response - no vulnerability detected</p>`)
	}

	if id != "" {
		response.WriteString(fmt.Sprintf("<p>ID: %s</p>\n", id))
	}
	if username != "" {
		response.WriteString(fmt.Sprintf("<p>Username: %s</p>\n", username))
	}
	if search != "" {
		response.WriteString(fmt.Sprintf("<p>Search: %s</p>\n", search))
	}

	fmt.Fprint(w, response.String())
}

func healthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status": "ok", "message": "Test server is running"}`)
}

func sensitiveFilesEndpoint(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/.env":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, `DB_PASSWORD=supersecret123
API_KEY=sk-1234567890abcdef
JWT_SECRET=my-secret-key
DEBUG=true`)
	case "/config.php":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, `<?php
$db_host = "localhost";
$db_user = "root";
$db_pass = "password123";
$api_key = "secret-api-key";
?>`)
	case "/web.config":
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <connectionStrings>
    <add name="DefaultConnection" connectionString="Server=localhost;Database=app;User=sa;Password=Password123;" />
  </connectionStrings>
</configuration>`)
	case "/backup.sql":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, `-- Database backup
CREATE TABLE users (id INT, username VARCHAR(50), password VARCHAR(100));
INSERT INTO users VALUES (1, 'admin', 'admin123');`)
	case "/robots.txt":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, `User-agent: *
Disallow: /admin/
Disallow: /config/
Disallow: /backup/`)
	default:
		http.NotFound(w, r)
	}
}

func directoryListingEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html>
<head><title>Index of /uploads</title></head>
<body>
<h1>Index of /uploads</h1>
<pre>
<a href="../">../</a>
<a href="config.bak">config.bak</a>                    01-Jan-2024 12:00    1024
<a href="database.sql">database.sql</a>                01-Jan-2024 12:00    5120
<a href="passwords.txt">passwords.txt</a>              01-Jan-2024 12:00     256
</pre>
</body>
</html>`)
}

func insecureHeadersEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Server", "Apache/2.4.41 (Ubuntu)")
	fmt.Fprint(w, `<html>
<head><title>Insecure Page</title></head>
<body>
<h1>This page has missing security headers</h1>
		<p>Missing: X-Frame-Options, X-Content-Type-Options, X-XSS-Protection,</p>
		<p>Strict-Transport-Security, Content-Security-Policy</p>
</body>
</html>`)
}

func secureHeadersEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
	fmt.Fprint(w, `<html>
<head><title>Secure Page</title></head>
<body>
<h1>This page has proper security headers</h1>
</body>
</html>`)
}

func defaultCredentialsEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if (username == defaultUsername && password == defaultPassword) ||
			(username == rootUsername && password == rootPassword) ||
			(username == defaultUsername && password == passwordValue) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html>
<head><title>Admin Panel</title></head>
<body>
<h1>Welcome to Admin Panel</h1>
<p>Login successful with default credentials!</p>
</body>
</html>`)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html>
<head><title>Login Failed</title></head>
<body>
<h1>Login Failed</h1>
<p>Invalid credentials</p>
</body>
</html>`)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html>
<head><title>Admin Login</title></head>
<body>
<h1>Admin Login</h1>
<form method="post">
<p>Username: <input type="text" name="username"></p>
<p>Password: <input type="password" name="password"></p>
<p><input type="submit" value="Login"></p>
</form>
</body>
</html>`)
}

func versionDisclosureEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Server", "Apache/2.4.41 (Ubuntu) PHP/7.4.3")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, `<html>
<head><title>500 Internal Server Error</title></head>
<body>
<h1>Internal Server Error</h1>
<p>The server encountered an internal error or misconfiguration and was unable to complete your request.</p>
<hr>
<address>Apache/2.4.41 (Ubuntu) Server at localhost Port 8080</address>
<p>PHP Version: 7.4.3</p>
<p>MySQL Version: 8.0.25</p>
</body>
</html>`)
}

func dangerousMethodsEndpoint(w http.ResponseWriter, r *http.Request) {
	method := r.Method

	switch method {
	case http.MethodPut:
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "PUT method is enabled - file upload successful")
	case http.MethodDelete:
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "DELETE method is enabled - resource deleted")
	case http.MethodTrace:
		w.Header().Set("Content-Type", "message/http")
		fmt.Fprintf(w, "TRACE %s HTTP/1.1\r\n", r.URL.Path)
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Fprintf(w, "%s: %s\r\n", name, value)
			}
		}
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE, TRACE, OPTIONS")
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Dangerous HTTP methods are enabled")
	default:
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html>
<head><title>HTTP Methods Test</title></head>
<body>
<h1>HTTP Methods Test Endpoint</h1>
<p>Try different HTTP methods (PUT, DELETE, TRACE, OPTIONS) to test for dangerous method support</p>
</body>
</html>`)
	}
}

func infoLeakageEndpoint(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")

	if file != "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `<html>
<head><title>Application Error</title></head>
<body>
<h1>Application Error</h1>
<p>Error: Could not open file '/var/www/html/%s'</p>
<p>Stack trace:</p>
<pre>
at FileHandler.readFile(/var/www/html/app.js:123)
at Router.handleRequest(/var/www/html/app.js:456)
at Server.processRequest(/var/www/html/server.js:789)
</pre>
<p>Database connection string: mysql://user:password@localhost:3306/appdb</p>
</body>
</html>`, file)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html>
<head><title>File Reader</title></head>
<body>
<h1>File Reader</h1>
<p>Add ?file=filename to test information leakage</p>
</body>
</html>`)
}

func defaultInstallEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<html>
<head><title>Apache2 Ubuntu Default Page</title></head>
<body>
<h1>Apache2 Ubuntu Default Page</h1>
<p>It works!</p>
		<p>This is the default welcome page used to test the correct operation of the</p>
		<p>Apache2 server after installation on Ubuntu systems.</p>
<p>If you can read this page, it means that the Apache HTTP server installed at this site is working properly.</p>
<hr>
<p>Configuration Overview</p>
<p>Ubuntu's Apache2 default configuration is different from the upstream default configuration.</p>
</body>
</html>`)
}

func backupFilesEndpoint(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/config.bak":
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		configData := "database_password=secret123\napi_key=abc123def456\nserver_config=production"
		if _, err := w.Write([]byte(configData)); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	case "/app.config.old":
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		xmlContent := `<?xml version="1.0"?>
<configuration>
  <connectionStrings>
    <add name="DefaultConnection" connectionString="Server=localhost;Database=prod;User=admin;Password=admin123;" />
  </connectionStrings>
</configuration>`
		if _, err := w.Write([]byte(xmlContent)); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	case "/database.sql.backup":
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		sqlContent := `-- Database backup
CREATE TABLE users (
    id INT PRIMARY KEY,
    username VARCHAR(50),
    password VARCHAR(255)
);
INSERT INTO users VALUES (1, 'admin', 'admin123');`
		if _, err := w.Write([]byte(sqlContent)); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	default:
		http.NotFound(w, r)
	}
}

func main() {
	http.HandleFunc("/", vulnerableEndpoint)
	http.HandleFunc("/login", vulnerableEndpoint)
	http.HandleFunc("/search", vulnerableEndpoint)
	http.HandleFunc("/user", vulnerableEndpoint)
	http.HandleFunc("/health", healthEndpoint)

	http.HandleFunc("/.env", sensitiveFilesEndpoint)
	http.HandleFunc("/config.php", sensitiveFilesEndpoint)
	http.HandleFunc("/web.config", sensitiveFilesEndpoint)
	http.HandleFunc("/backup.sql", sensitiveFilesEndpoint)
	http.HandleFunc("/robots.txt", sensitiveFilesEndpoint)
	http.HandleFunc("/uploads/", directoryListingEndpoint)

	http.HandleFunc("/insecure-headers", insecureHeadersEndpoint)
	http.HandleFunc("/secure-headers", secureHeadersEndpoint)

	http.HandleFunc("/admin", defaultCredentialsEndpoint)
	http.HandleFunc("/admin/login", defaultCredentialsEndpoint)
	http.HandleFunc("/version-error", versionDisclosureEndpoint)
	http.HandleFunc("/default-install", defaultInstallEndpoint)
	http.HandleFunc("/methods-test", dangerousMethodsEndpoint)
	http.HandleFunc("/info-leak", infoLeakageEndpoint)

	http.HandleFunc("/config.bak", backupFilesEndpoint)
	http.HandleFunc("/app.config.old", backupFilesEndpoint)
	http.HandleFunc("/database.sql.backup", backupFilesEndpoint)

	fmt.Println("🚨 VULNERABLE TEST SERVER RUNNING ON :8080")
	fmt.Println("⚠️  This server is intentionally vulnerable for testing purposes!")
	fmt.Println("   Original endpoints:")
	fmt.Println("   - http://localhost:8080/?id=1")
	fmt.Println("   - http://localhost:8080/login")
	fmt.Println("   - http://localhost:8080/search?q=test")
	fmt.Println("   - http://localhost:8080/user?username=admin")
	fmt.Println("")
	fmt.Println("   Misconfiguration test endpoints:")
	fmt.Println("   Sensitive Files:")
	fmt.Println("   - http://localhost:8080/.env")
	fmt.Println("   - http://localhost:8080/config.php")
	fmt.Println("   - http://localhost:8080/web.config")
	fmt.Println("   - http://localhost:8080/backup.sql")
	fmt.Println("   - http://localhost:8080/robots.txt")
	fmt.Println("   - http://localhost:8080/uploads/")
	fmt.Println("   - http://localhost:8080/config.bak")
	fmt.Println("   - http://localhost:8080/app.config.old")
	fmt.Println("   - http://localhost:8080/database.sql.backup")
	fmt.Println("   Security Headers:")
	fmt.Println("   - http://localhost:8080/insecure-headers")
	fmt.Println("   - http://localhost:8080/secure-headers")
	fmt.Println("   Default Credentials:")
	fmt.Println("   - http://localhost:8080/admin (try admin/admin)")
	fmt.Println("   - http://localhost:8080/admin/login")
	fmt.Println("   - http://localhost:8080/version-error")
	fmt.Println("   - http://localhost:8080/default-install")
	fmt.Println("   Server Configuration:")
	fmt.Println("   - http://localhost:8080/methods-test (try PUT/DELETE/TRACE)")
	fmt.Println("   - http://localhost:8080/info-leak?file=test.txt")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
