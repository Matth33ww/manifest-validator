package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// maxBodyBytes is the biggest request body we accept: 64 KB.
const maxBodyBytes = 64 * 1024

// ValidateResponse is the JSON we send back from POST /validate.
type ValidateResponse struct {
	Valid  bool    `json:"valid"`
	Issues []Issue `json:"issues"`
}

func main() {
	port := getEnv("PORT", "8080")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "http://localhost:4200")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: newHandler(allowedOrigin),
		// Timeouts stop slow clients from keeping a connection open forever.
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("listening on port %s", port)
	log.Fatal(server.ListenAndServe())
}

// getEnv reads an environment variable. If it is not set, it returns the fallback.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// newHandler connects each URL path to its function.
// The tests use it too, so they test the same routes as the real server.
func newHandler(allowedOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", handleValidate(NewHealthChecker()))
	mux.HandleFunc("/healthz", handleHealthz)

	// The routes for registered services and their announcements.
	// One store is made here and lives as long as the server runs.
	NewServicesAPI(NewStore()).Routes(mux)

	return withCORS(allowedOrigin, mux)
}

// handleValidate reads a manifest from the request body and checks it.
// checker runs the live health check (rule 8) when the user asks for it.
func handleValidate(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "use POST to send a manifest")
			return
		}

		// readBody stops reading if the body is bigger than the limit.
		body, ok := readBody(w, r)
		if !ok {
			return
		}

		manifest, err := ParseManifest(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		issues := Validate(manifest)

		// Rule 8 runs only when the user asks for it with ?checkHealth=true,
		// because it calls the service over the network and can be slow.
		// We skip it when the manifest has errors, because the address may be wrong.
		if r.URL.Query().Get("checkHealth") == "true" && !hasErrors(issues) {
			issues = append(issues, checker.Check(r.Context(), manifest)...)
		}
		if issues == nil {
			// Send an empty list "[]" in the JSON, not "null".
			issues = []Issue{}
		}
		writeJSON(w, http.StatusOK, ValidateResponse{
			Valid:  !hasErrors(issues),
			Issues: issues,
		})
	}
}

// handleHealthz tells other tools that the server is running.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// hasErrors is true if at least one issue is an error.
// Warnings do not count, so a manifest with only warnings is still valid.
func hasErrors(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// withCORS lets the frontend call this API from the browser.
// A browser blocks calls to another address unless the server allows it.
func withCORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Vary", "Origin")

		// The browser sends OPTIONS first to ask what is allowed.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON sends a value as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("could not write response: %v", err)
	}
}

// writeError sends an error message as JSON: {"error": "..."}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
