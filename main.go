package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

type URL struct {
	ID           string    `json:"id"`
	OriginalURL  string    `json:"original_url"`
	ShortURL     string    `json:"short_url"`
	CreatingDate time.Time `json:"creating_date"`
}

var urlDB = make(map[string]URL)

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
	urlDB[id] = URL{
		ID:           id,
		OriginalURL:  OriginalURL,
		ShortURL:     ShortURL,
		CreatingDate: time.Now(),
	}
	return ShortURL
}

func getURL(id string) (URL, error) {
	url, ok := urlDB[id]
	if !ok {
		return URL{}, fmt.Errorf("url not found")
	}
	return url, nil
}

func main() {
	fmt.Println("Hello, World!")
	OriginalURL := "https://www.baidu.com"
	ShortURL := generateShortURL(OriginalURL)
	fmt.Println(ShortURL)
}
