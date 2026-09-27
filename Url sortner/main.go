package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// 1. Struct and thread-safe map for storing data in memory.
type URLStore struct {
	mu    sync.RWMutex
	store map[string]string // ShortCode -> LongURL
}

var store = URLStore{
	store: make(map[string]string),
}

// Request & Response Structs
type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
	Code     string `json:"code"`
}

// 2. Helper function to generate a random 6-character code.
func generateShortCode() string {
	bytes := make([]byte, 3) // 3 bytes = 6 hex characters
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// 3. POST /api/v1/shorten - handler for shortening URLs.
func shortenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Only POST allowed"})
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL input"})
		return
	}

	// Add a scheme if the URL does not already include one.
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	// Generate a unique short code.
	code := generateShortCode()

	// Map the short code to the original URL in memory.
	store.mu.Lock()
	store.store[code] = req.URL
	store.mu.Unlock()

	shortURL := fmt.Sprintf("http://%s/%s", r.Host, code)

	w.WriteHeader(http.StatusCreated) // 201 Created.
	_ = json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: shortURL,
		Code:     code,
	})
}

// 4. GET /{code} - handler for redirecting to the original URL.
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	// Ignore the root (/) path and extract the short code.
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.Error(w, "Short code required", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	longURL, exists := store.store[code]
	store.mu.RUnlock()

	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound) // 404 Not Found.
		return
	}

	// Redirect to the original URL with HTTP 302 Found.
	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	mux := http.NewServeMux()

	// API route.
	mux.HandleFunc("/api/v1/shorten", shortenHandler)

	// Catch-all route for redirects (for example, http://localhost:8080/a1b2c3).
	mux.HandleFunc("/", redirectHandler)

	addr := ":8080"
	log.Printf("URL Shortener service running on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
