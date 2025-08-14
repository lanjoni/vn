package integration

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"vn/tests/shared/builders"
	"vn/tests/shared/testserver"
)

var (
	sharedBuildManager builders.BuildManager
	buildManagerOnce   sync.Once
	sharedServerPool   testserver.ServerPool
	serverPoolOnce     sync.Once
)

func getSharedBuildManager() builders.BuildManager {
	buildManagerOnce.Do(func() {
		sharedBuildManager = builders.NewBuildManager()
	})
	return sharedBuildManager
}

func getSharedServerPool() testserver.ServerPool {
	serverPoolOnce.Do(func() {
		sharedServerPool = testserver.NewServerPool()
	})
	return sharedServerPool
}

func createVulnerableTestHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
			response.WriteString(`<p style="color: red;">MySQL Error: You have an error in your SQL syntax</p>`)
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
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "ok", "message": "Test server is running"}`)
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			password := r.FormValue("password")
			if username == "admin" && password == "secret" {
				fmt.Fprint(w, `<p>Login successful</p>`)
			} else {
				fmt.Fprint(w, `<p>Login failed</p>`)
			}
		} else {
			fmt.Fprint(w, `<form method="post"><input name="username"><input type="password" name="password"></form>`)
		}
	})

	return mux
}