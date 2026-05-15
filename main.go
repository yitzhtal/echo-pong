package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Server    string    `json:"server"`
}

var secret string
var ready atomic.Bool

const (
	defaultPort     = "8080"
	shutdownTimeout = 25 * time.Second
	shutdownDrain   = 5 * time.Second
	startupDelay    = 10 * time.Second
)

// CLI flags
var (
	mode     = flag.String("mode", "server", "Mode to run in: 'server' or 'cli'")
	password = flag.String("password", "", "Password for CLI authentication")
	help     = flag.Bool("help", false, "Show help information")
)

// readSecretFromFile reads the secret from the file path specified by environment variable
func readSecretFromFile() error {
	secretPath := os.Getenv("SECRET_FILE_PATH")
	if secretPath == "" {
		return fmt.Errorf("SECRET_FILE_PATH environment variable not set")
	}

	file, err := os.Open(secretPath)
	if err != nil {
		return fmt.Errorf("failed to open secret file %s: %v", secretPath, err)
	}
	defer file.Close()

	secretBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read secret file: %v", err)
	}

	secret = strings.TrimSpace(string(secretBytes))
	if secret == "" {
		return fmt.Errorf("secret file is empty")
	}

	log.Printf("🔐 Secret loaded from %s", secretPath)
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func requireMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		next(w, r)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// authMiddleware checks for valid Authorization header
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Authorization header required",
			})
			log.Printf("🚫 Unauthorized access attempt from %s - missing auth header", r.RemoteAddr)
			return
		}

		// Support both "Bearer <token>" and direct token formats
		token := authHeader
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[len("Bearer "):])
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Invalid authorization token",
			})
			log.Printf("🚫 Unauthorized access attempt from %s - invalid token", r.RemoteAddr)
			return
		}

		next(w, r)
	}
}

// validatePassword checks if the provided password matches the secret
func validatePassword(pwd string) bool {
	return subtle.ConstantTimeCompare([]byte(pwd), []byte(secret)) == 1
}

// runCLI handles CLI mode operations
func runCLI() {
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("❌ No command specified. Available commands: ping, pong")
		printCLIHelp()
		os.Exit(1)
	}

	if *password == "" {
		fmt.Println("❌ Password is required for CLI commands. Use --password flag")
		os.Exit(1)
	}

	if !validatePassword(*password) {
		fmt.Println("❌ Invalid password")
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "ping":
		response := Response{
			Message:   "pong",
			Timestamp: time.Now(),
			Server:    "ping-pong-cli",
		}
		jsonOutput, _ := json.MarshalIndent(response, "", "  ")
		fmt.Printf("🏓 PING → PONG\n%s\n", jsonOutput)

	case "pong":
		response := Response{
			Message:   "ping",
			Timestamp: time.Now(),
			Server:    "ping-pong-cli",
		}
		jsonOutput, _ := json.MarshalIndent(response, "", "  ")
		fmt.Printf("🏓 PONG → PING\n%s\n", jsonOutput)

	default:
		fmt.Printf("❌ Unknown command: %s\n", command)
		fmt.Println("Available commands: ping, pong")
		os.Exit(1)
	}
}

// printCLIHelp shows usage information
func printCLIHelp() {
	fmt.Print(`
🏓 Ping Pong Game - CLI & Server

USAGE:
  # Run as HTTP server (default)
  ./ping-pong-app --mode=server
  
  # Run CLI commands
  ./ping-pong-app --mode=cli --password=<secret> <command>

CLI COMMANDS:
  ping    Send a ping, get a pong response
  pong    Send a pong, get a ping response

FLAGS:
  --mode      Mode to run in: 'server' or 'cli' (default: server)
  --password  Password for CLI authentication (required for CLI mode)
  --help      Show this help information

EXAMPLES:
  ./ping-pong-app --mode=cli --password=mysecret ping
  ./ping-pong-app --mode=cli --password=mysecret pong
  ./ping-pong-app --mode=server

ENVIRONMENT VARIABLES:
  SECRET_FILE_PATH  Path to file containing the secret/password
  PORT             Port for HTTP server mode (default: 8080)
`)
}

func main() {
	// Parse CLI flags first
	flag.Parse()

	if *help {
		printCLIHelp()
		os.Exit(0)
	}

	// Load secret from file
	if err := readSecretFromFile(); err != nil {
		log.Fatalf("❌ Failed to load secret: %v", err)
	}

	if *mode == "cli" {
		runCLI()
		os.Exit(0)
	}

	// Server mode continues below
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	mux := http.NewServeMux()

	// Protected endpoints with authentication
	mux.HandleFunc("/ping", requireMethod(http.MethodGet, authMiddleware(pingHandler)))
	mux.HandleFunc("/pong", requireMethod(http.MethodGet, authMiddleware(pongHandler)))

	// Public endpoints
	mux.HandleFunc("/health", requireMethod(http.MethodGet, healthHandler))
	mux.HandleFunc("/ready", requireMethod(http.MethodGet, readyHandler))
	mux.HandleFunc("/", requireMethod(http.MethodGet, rootHandler))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("🏓 Ping-Pong server starting on port %s", port)
	log.Printf("📍 Available endpoints:")
	log.Printf("   GET /ping  - Returns pong (🔐 Auth required)")
	log.Printf("   GET /pong  - Returns ping (🔐 Auth required)")
	log.Printf("   GET /health - Health check")
	log.Printf("   GET /ready - Readiness check")
	log.Printf("   GET /      - API documentation")

	log.Printf("Thinking for 10 seconds before starting the server")
	select {
	case <-time.After(startupDelay):
	case <-ctx.Done():
		log.Printf("Shutdown requested before server startup completed")
		return
	}

	log.Printf("Starting the server")
	ready.Store(true)

	go func() {
		<-ctx.Done()
		// Marking readiness false first lets Kubernetes drain traffic before the
		// process exits during a rolling update or node eviction.
		ready.Store(false)
		time.Sleep(shutdownDrain)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server failed: %v", err)
	}

	log.Printf("Server stopped")
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Message:   "pong",
		Timestamp: time.Now(),
		Server:    "ping-pong-server",
	}

	writeJSON(w, http.StatusOK, response)

	log.Printf("🏓 PING received from %s → responding with PONG", r.RemoteAddr)
}

func pongHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Message:   "ping",
		Timestamp: time.Now(),
		Server:    "ping-pong-server",
	}

	writeJSON(w, http.StatusOK, response)

	log.Printf("🏓 PONG received from %s → responding with PING", r.RemoteAddr)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    "running",
		"service":   "ping-pong-game",
		"ready":     ready.Load(),
	}

	writeJSON(w, http.StatusOK, health)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	if !ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "not ready",
			"timestamp": time.Now(),
			"service":   "ping-pong-game",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
		"service":   "ping-pong-game",
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
		})
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🏓 Ping Pong Game API</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px; 
            margin: 2rem auto; 
            padding: 0 1rem;
            line-height: 1.6;
            color: #333;
        }
        .header { text-align: center; margin-bottom: 2rem; }
        .endpoint { 
            background: #f8f9fa; 
            border-left: 4px solid #007bff; 
            padding: 1rem; 
            margin: 1rem 0; 
            border-radius: 4px;
        }
        .endpoint h3 { margin-top: 0; color: #007bff; }
        .method { 
            background: #28a745; 
            color: white; 
            padding: 0.2rem 0.5rem; 
            border-radius: 3px; 
            font-size: 0.8rem; 
            font-weight: bold;
        }
        .try-it { 
            display: inline-block; 
            background: #007bff; 
            color: white; 
            text-decoration: none; 
            padding: 0.5rem 1rem; 
            border-radius: 4px; 
            margin-top: 0.5rem;
        }
        .try-it:hover { background: #0056b3; }
        .footer { 
            text-align: center; 
            margin-top: 3rem; 
            padding-top: 2rem; 
            border-top: 1px solid #eee; 
            color: #666;
        }
        code {
            background: #f1f1f1;
            padding: 0.2rem 0.4rem;
            border-radius: 3px;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🏓 Ping Pong Game API</h1>
        <p>A simple HTTP API for playing ping pong with token-based authentication!</p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> /ping 🔐</h3>
        <p>Send a ping and get a pong back!</p>
        <p><strong>Authentication required:</strong> Include <code>Authorization</code> header with secret token</p>
        <a href="/ping" class="try-it">Try it →</a>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> /pong 🔐</h3>
        <p>Send a pong and get a ping back!</p>
        <p><strong>Authentication required:</strong> Include <code>Authorization</code> header with secret token</p>
        <a href="/pong" class="try-it">Try it →</a>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> /health</h3>
        <p>Check if the service process is healthy (for Kubernetes liveness probes)</p>
        <a href="/health" class="try-it">Try it →</a>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> /ready</h3>
        <p>Check if the service is ready to receive traffic (for Kubernetes readiness probes)</p>
        <a href="/ready" class="try-it">Try it →</a>
    </div>

    <div class="footer">
        <p>🚀 Built for DevOps Home Assignment</p>
        <p>Focus on containerization, CI/CD, and Kubernetes deployment!</p>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)

	log.Printf("📄 Root page served to %s", r.RemoteAddr)
}
