package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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

func main() {
	http.HandleFunc("/", vulnerableEndpoint)
	http.HandleFunc("/login", vulnerableEndpoint)
	http.HandleFunc("/search", vulnerableEndpoint)
	http.HandleFunc("/user", vulnerableEndpoint)
	http.HandleFunc("/health", healthEndpoint)

	fmt.Println("🚨 VULNERABLE TEST SERVER RUNNING ON :8080")
	fmt.Println("⚠️  This server is intentionally vulnerable for testing purposes!")
	fmt.Println("   Available endpoints:")
	fmt.Println("   - http://localhost:8080/?id=1")
	fmt.Println("   - http://localhost:8080/login")
	fmt.Println("   - http://localhost:8080/search?q=test")
	fmt.Println("   - http://localhost:8080/user?username=admin")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
