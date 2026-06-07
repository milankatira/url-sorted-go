package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type URL struct {
	ID           string    `json:"id"`
	OriginalURL  string    `json:"original_url"`
	ShortURL     string    `json:"short_url"`
	CreatingDate time.Time `json:"creating_date"`
}

var (
	urlDB = make(map[string]URL)
	mu    sync.RWMutex
)

// normalizeURL trims whitespace and stray quotes/backticks, then validates
// that the result is an absolute http(s) URL.
func normalizeURL(raw string) (string, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.Trim(cleaned, "`'\"")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", fmt.Errorf("url is empty")
	}
	parsed, err := url.ParseRequestURI(cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("url must be absolute http or https")
	}
	return cleaned, nil
}

func generateShortURL(OriginalURL string) string {
	hasher := md5.New()
	hasher.Write([]byte(OriginalURL))
	data := hasher.Sum(nil)
	hash := hex.EncodeToString(data)
	return hash[:8]
}

func createURL(OriginalURL string) string {
	ShortURL := generateShortURL(OriginalURL)
	id := ShortURL
	mu.Lock()
	defer mu.Unlock()
	urlDB[id] = URL{
		ID:           id,
		OriginalURL:  OriginalURL,
		ShortURL:     ShortURL,
		CreatingDate: time.Now(),
	}
	return ShortURL
}

func getURL(id string) (URL, error) {
	mu.RLock()
	defer mu.RUnlock()
	url, ok := urlDB[id]
	if !ok {
		return URL{}, fmt.Errorf("url not found")
	}
	return url, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func ShortURLHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		URL string `json:"url"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cleanURL, err := normalizeURL(data.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortURL := createURL(cleanURL)

	response := struct {
		ShortURL string `json:"short_url"`
	}{
		ShortURL: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func redirectURLHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	url, err := getURL(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}

func main() {
	http.HandleFunc("GET /health", handler)
	http.HandleFunc("POST /short", ShortURLHandler)
	http.HandleFunc("GET /redirect/{id}", redirectURLHandler)

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Println("Error on staring server", err)
	}
}
