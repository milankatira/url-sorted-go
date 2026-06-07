package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestMux mirrors the route registration in main().
func newTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler)
	mux.HandleFunc("POST /short", ShortURLHandler)
	mux.HandleFunc("GET /redirect/{id}", redirectURLHandler)
	return mux
}

func TestShortURLHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantShort  string
	}{
		{
			name:       "clean url",
			body:       `{"url":"https://github.com/davidbyttow/govips"}`,
			wantStatus: http.StatusOK,
			wantShort:  `"short_url":"f9751de4"`,
		},
		{
			name:       "url wrapped in backticks and spaces is normalized",
			body:       "{\"url\":\" `https://github.com/davidbyttow/govips` \"}",
			wantStatus: http.StatusOK,
			wantShort:  `"short_url":"f9751de4"`,
		},
		{
			name:       "garbage url",
			body:       `{"url":"not a url"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty url",
			body:       `{"url":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing scheme",
			body:       `{"url":"www.baidu.com"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	mux := newTestMux()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/short", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantShort != "" && !strings.Contains(rec.Body.String(), tt.wantShort) {
				t.Errorf("body = %s, want it to contain %s", rec.Body.String(), tt.wantShort)
			}
		})
	}
}

func TestRedirectURLHandler(t *testing.T) {
	mux := newTestMux()

	// Seed an entry via the public endpoint.
	seed := httptest.NewRequest(http.MethodPost, "/short", strings.NewReader(`{"url":"https://github.com/davidbyttow/govips"}`))
	seedRec := httptest.NewRecorder()
	mux.ServeHTTP(seedRec, seed)
	if seedRec.Code != http.StatusOK {
		t.Fatalf("seed POST /short failed: %d %s", seedRec.Code, seedRec.Body.String())
	}

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantLocation string
	}{
		{
			name:         "known id redirects with clean location",
			path:         "/redirect/f9751de4",
			wantStatus:   http.StatusFound,
			wantLocation: "https://github.com/davidbyttow/govips",
		},
		{
			name:       "unknown id returns 404",
			path:       "/redirect/deadbeef",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing id returns 404 without panic",
			path:       "/redirect",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantLocation != "" {
				if got := rec.Header().Get("Location"); got != tt.wantLocation {
					t.Errorf("Location = %q, want %q", got, tt.wantLocation)
				}
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "clean", in: "https://github.com/davidbyttow/govips", want: "https://github.com/davidbyttow/govips"},
		{name: "backticks and spaces", in: " `https://github.com/davidbyttow/govips` ", want: "https://github.com/davidbyttow/govips"},
		{name: "quotes", in: `"https://example.com"`, want: "https://example.com"},
		{name: "empty", in: "", wantErr: true},
		{name: "whitespace only", in: "   ", wantErr: true},
		{name: "no scheme", in: "www.baidu.com", wantErr: true},
		{name: "ftp scheme rejected", in: "ftp://example.com", wantErr: true},
		{name: "garbage", in: "not a url", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeURL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeURL(%q) = %q, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeURL(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
