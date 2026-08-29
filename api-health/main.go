package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

type Result struct {
	URL        string `json:"url"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
}

func check(url string) Result {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return Result{URL: url, OK: false, StatusCode: 0}
	}
	defer resp.Body.Close()
	return Result{URL: url, OK: resp.StatusCode >= 200 && resp.StatusCode <= 299, StatusCode: resp.StatusCode}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var urls []string

	if r.Method == http.MethodPost {
		var req struct{ URLs []string `json:"urls"` }
		_ = json.NewDecoder(r.Body).Decode(&req)
		urls = req.URLs
	} else {
		// GET Request হলে urls.txt ফাইল থেকে পড়বে
		file, err := os.Open("urls.txt")
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				if line := strings.TrimSpace(scanner.Text()); line != "" {
					urls = append(urls, line)
				}
			}
			file.Close()
		}
	}

	results := []Result{} // Empty slice to avoid null response
	for _, u := range urls {
		results = append(results, check(u))
	}

	json.NewEncoder(w).Encode(results)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.ListenAndServe(":8081", nil)
}