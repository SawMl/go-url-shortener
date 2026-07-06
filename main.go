// Command shorty is a minimal URL shortener.
//
// Phase 1 (walking skeleton): in-memory storage only. No database or cache yet.
// Endpoints:
//
//	POST /shorten  body: {"url":"https://example.com/long"}  -> {"code":"1","short":"http://host/1"}
//	GET  /{code}   -> 302 redirect to the original URL (404 if unknown)
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
)

const base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// store is a tiny in-memory URL store. Phase 2 replaces this with Postgres.
type store struct {
	mu      sync.Mutex
	counter uint64
	urls    map[string]string
}

func newStore() *store {
	return &store{urls: make(map[string]string)}
}

// save records a URL and returns its newly generated short code.
func (s *store) save(url string) string {

	s.mu.Lock()

	defer s.mu.Unlock()

	s.counter++

	code := encodeBase62(s.counter)

	s.urls[code] = url

	return code
}

// lookup returns the URL for a code and whether it existed.
func (s *store) lookup(code string) (string, bool) {

	s.mu.Lock()

	defer s.mu.Unlock()

	url, ok := s.urls[code]

	return url, ok
}

// encodeBase62 converts a positive integer to a base62 string.
func encodeBase62(n uint64) string {

	if n == 0 {
		return "0"
	}

	var b strings.Builder

	for n > 0 {
		b.WriteByte(base62[n%62])
		n /= 62
	}

	r := []byte(b.String())

	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}

	return string(r)
}

// shortenRequest models expected JSON input for POST /shorten.
type shortenRequest struct {
	URL string `json:"url"`
}

// shortenResponse models JSON output sent back to the client.
type shortenResponse struct {
	Code  string `json:"code"`
	Short string `json:"short"`
}

func main() {
	s := newStore()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", func(w http.ResponseWriter, r *http.Request) {
		var req shortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if !validURL(req.URL) {
			http.Error(w, "url must start with http:// or https://", http.StatusBadRequest)
			return
		}
		code := s.save(req.URL)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shortenResponse{
			Code:  code,
			Short: "http://" + r.Host + "/" + code,
		})
	})

	// {code} is a path parameter supported by Go's newer ServeMux patterns.
	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		url, ok := s.lookup(code)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Redirect(w, r, url, http.StatusFound)
	})

	addr := ":8080"
	log.Printf("shorty listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// validURL is a minimal check; Phase 2 hardens validation.
func validURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
