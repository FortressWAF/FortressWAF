package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Comment struct {
	Author  string `json:"author"`
	Message string `json:"message"`
}

var (
	comments []Comment
	mu       sync.Mutex
	users    = map[string]string{"admin": "supersecret", "user": "password123"}
)

func main() {
	port := os.Getenv("VULN_APP_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/search", searchHandler)
	mux.HandleFunc("/comment", commentHandler)
	mux.HandleFunc("/comments", getCommentsHandler)
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/file", fileHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/reset", resetHandler)

	fmt.Printf("Vulnerable test app listening on :%s\n", port)
	http.ListenAndServe(":"+port, mux)
}

func renderPage(w http.ResponseWriter, title, content string) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>%s</title>
<style>
body{font-family:monospace;max-width:800px;margin:40px auto;padding:20px;background:#1a1a2e;color:#eee;}
h1{color:#e94560;}
a{color:#0f3460;background:#e94560;padding:4px 12px;border-radius:4px;text-decoration:none;margin:4px;}
a:hover{background:#ff6b81;}
nav{margin:20px 0;}
form{margin:20px 0;padding:20px;background:#16213e;border-radius:8px;}
input,textarea{display:block;width:100%%;margin:8px 0;padding:8px;background:#0f3460;border:1px solid #e94560;color:#eee;border-radius:4px;}
button{background:#e94560;color:#fff;border:none;padding:8px 20px;border-radius:4px;cursor:pointer;}
pre{background:#0f3460;padding:10px;border-radius:4px;overflow-x:auto;}
.result{background:#2d1b2e;border:1px solid #e94560;padding:10px;margin:10px 0;border-radius:4px;}
</style></head><body>
<h1>🔓 VulnApp — Test Target</h1>
<nav><a href="/">Home</a><a href="/login">Login</a><a href="/search">Search</a><a href="/comment">Comment</a><a href="/ping">Ping</a><a href="/file">File</a></nav>
%s
<hr><small>FortressWAF Test Target — DO NOT DEPLOY TO PRODUCTION</small>
</body></html>`, title, content)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	content := `<div class="result">
<p>This is a <strong>deliberately vulnerable</strong> web application for testing FortressWAF.</p>
<p>Try these attacks:</p>
<ul>
<li><strong>SQLi:</strong> <code>' OR '1'='1</code> on Login or Search</li>
<li><strong>XSS:</strong> <code>&lt;script&gt;alert(1)&lt;/script&gt;</code> on Search or Comment</li>
<li><strong>CMDi:</strong> <code>; cat /etc/passwd</code> on Ping</li>
<li><strong>LFI:</strong> <code>../../../etc/passwd</code> on File</li>
</ul>
</div>`
	renderPage(w, "VulnApp Home", content)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		content := `<form method="POST" action="/login">
<label>Username: <input type="text" name="username" placeholder="Enter username"></label>
<label>Password: <input type="password" name="password" placeholder="Enter password"></label>
<button type="submit">Login</button>
</form>`
		renderPage(w, "Login", content)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	query := fmt.Sprintf("SELECT * FROM users WHERE username='%s' AND password='%s'", username, password)

	authenticated := false
	for u, p := range users {
		if strings.Contains(username, "'") || strings.Contains(password, "'") {
			authenticated = true
			break
		}
		if u == username && p == password {
			authenticated = true
			break
		}
	}

	result := fmt.Sprintf(`<div class="result">
<h3>Login Result</h3>
<pre>Query: %s</pre>
<p><strong>Status:</strong> %s</p>
</div>`, template.HTMLEscapeString(query), map[bool]string{true: "✅ ACCESS GRANTED", false: "❌ ACCESS DENIED"}[authenticated])

	if authenticated {
		result += `<p style="color:#4ecca3">Welcome! You have access to all user data.</p>`
	} else {
		result += `<p style="color:#e94560">Invalid credentials.</p>`
	}

	result += `<p><a href="/login">Try again</a></p>`
	renderPage(w, "Login Result", result)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		content := `<form method="GET" action="/search">
<label>Search: <input type="text" name="q" placeholder="Search products..."></label>
<button type="submit">Search</button>
</form>`
		renderPage(w, "Search", content)
		return
	}

	q := r.URL.Query().Get("q")
	results := "No results found"

	lower := strings.ToLower(q)
	if strings.Contains(lower, "union") || strings.Contains(lower, "'") || strings.Contains(lower, "select") || strings.Contains(lower, "or ") {
		results = "ALL USERS: admin, editor, viewer, guest"
	} else if q != "" {
		results = fmt.Sprintf("Product: %s — $19.99", q)
	}

	content := fmt.Sprintf(`<div class="result">
<h3>Search Results for: %s</h3>
<pre>Query: SELECT * FROM products WHERE name LIKE '%%%s%%'</pre>
<p><strong>Result:</strong> %s</p>
</div>`, template.HTMLEscapeString(q), template.HTMLEscapeString(q), results)

	renderPage(w, "Search Results", content)
}

func commentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		content := `<form method="POST" action="/comment">
<label>Author: <input type="text" name="author" placeholder="Your name"></label>
<label>Message:<br><textarea name="message" rows="4" cols="50" placeholder="Write a comment..."></textarea></label>
<button type="submit">Post Comment</button>
</form>`
		renderPage(w, "Leave a Comment", content)
		return
	}

	author := r.PostFormValue("author")
	message := r.PostFormValue("message")

	mu.Lock()
	comments = append(comments, Comment{Author: author, Message: message})
	mu.Unlock()

	http.Redirect(w, r, "/comments", http.StatusFound)
}

func getCommentsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var items string
	if len(comments) == 0 {
		items = "<p>No comments yet.</p>"
	} else {
		for i, c := range comments {
			items += fmt.Sprintf(`<div class="result">
<strong>#%d — %s</strong>
<p>%s</p>
</div>`, i+1, c.Author, c.Message)
		}
	}

	content := fmt.Sprintf(`<h3>Comments (%d)</h3>
%s
<p><a href="/comment">Add comment</a></p>`, len(comments), items)
	renderPage(w, "Comments", content)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		content := `<form method="POST" action="/ping">
<label>Ping an address: <input type="text" name="addr" placeholder="8.8.8.8" value="8.8.8.8"></label>
<button type="submit">Ping</button>
</form>`
		renderPage(w, "Ping", content)
		return
	}

	addr := r.PostFormValue("addr")
	cmd := exec.Command("sh", "-c", "ping -c 1 "+addr)
	output, err := cmd.CombinedOutput()

	content := fmt.Sprintf(`<div class="result">
<h3>Ping Result</h3>
<pre>Command: ping -c 1 %s</pre>
<pre>%s</pre>
</div>`, template.HTMLEscapeString(addr), template.HTMLEscapeString(string(output)))

	if err != nil {
		content += fmt.Sprintf(`<p style="color:#e94560">Error: %s</p>`, err)
	}

	renderPage(w, "Ping Result", content)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		content := `<form method="GET" action="/file">
<label>Filename: <input type="text" name="name" placeholder="e.g. notes.txt"></label>
<button type="submit">Read File</button>
</form>`
		renderPage(w, "File Reader", content)
		return
	}

	name := r.URL.Query().Get("name")

	safeDir, _ := filepath.Abs("files")
	reqPath := filepath.Join(safeDir, name)

	data, err := os.ReadFile(reqPath)

	content := fmt.Sprintf(`<div class="result">
<h3>File Read</h3>
<pre>Path: %s</pre>
`, template.HTMLEscapeString(reqPath))

	if err == nil {
		content += fmt.Sprintf(`<pre>%s</pre>`, template.HTMLEscapeString(string(data)))
	} else {
		content += fmt.Sprintf(`<p style="color:#e94560">Error: %s</p>`, err)
	}
	content += `</div>`

	renderPage(w, "File Reader", content)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"app":    "vuln-app",
		"version": "1.0.0",
	})
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<form method="POST"><button type="submit">Reset All Data</button></form>`))
		return
	}

	mu.Lock()
	comments = nil
	mu.Unlock()

	users = map[string]string{"admin": "supersecret", "user": "password123"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reset", "message": "All data has been reset"})
}

func init() {
	os.MkdirAll("files", 0755)
	os.WriteFile("files/notes.txt", []byte("This is a secret note.\nFlag: FORTRESS{test_flag_123}"), 0644)
	os.WriteFile("files/readme.md", []byte("# Welcome\nThis is a safe file."), 0644)
}
