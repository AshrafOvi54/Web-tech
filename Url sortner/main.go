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

// ১. ডাটা ইন-মেমরিতে সংরক্ষণের জন্য Struct ও Thread-safe Map
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

// ২. ৬ অক্ষরের র‍্যান্ডম সিক্রেট কোড তৈরি করার হেলপার ফাংশন
func generateShortCode() string {
	bytes := make([]byte, 3) // 3 bytes = 6 hex characters
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ৩. POST /api/v1/shorten - URL সংক্ষিপ্ত করার হ্যান্ডলার
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

	// URL-এ http:// বা https:// না থাকলে যোগ করা
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}

	// ইউনিক শর্ট কোড জেনারেট করা
	code := generateShortCode()

	// মেমরিতে শর্ট কোড ও আসল ইউআরএল ম্যাপ করে রাখা
	store.mu.Lock()
	store.store[code] = req.URL
	store.mu.Unlock()

	shortURL := fmt.Sprintf("http://%s/%s", r.Host, code)

	w.WriteHeader(http.StatusCreated) // 201 Created
	_ = json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: shortURL,
		Code:     code,
	})
}

// ৪. GET /{code} - আসল ওয়েবসাইটে রিডাইরেক্ট করার হ্যান্ডলার
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	// Root (/) পাথ এড়িয়ে শুধু কোড ফিল্টার করা
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.Error(w, "Short code required", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	longURL, exists := store.store[code]
	store.mu.RUnlock()

	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound) // 404
		return
	}

	// HTTP 302 Found দিয়ে মূল লিংকে Redirect করা
	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	mux := http.NewServeMux()

	// API Route
	mux.HandleFunc("/api/v1/shorten", shortenHandler)

	// Catch-all Route for redirection (উদা: http://localhost:8080/a1b2c3)
	mux.HandleFunc("/", redirectHandler)

	addr := ":8080"
	log.Printf("URL Shortener service running on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}