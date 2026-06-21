// Command shorty is a minimal URL shortener.
//
// Phase 1 (walking skeleton): in-memory storage only. No database or cache yet.
// Endpoints:
//
//	POST /shorten  body: {"url":"https://example.com/long"}  -> {"code":"1","short":"http://host/1"}
//	GET  /{code}   -> 302 redirect to the original URL (404 if unknown)
// package main defines this file's package; in Go, executable apps use package main.
package main

import (
	// encoding/json is Go's standard JSON parser/serializer (similar to Python json, Jackson, nlohmann/json).
	"encoding/json"
	// log provides simple logging helpers.
	"log"
	// net/http is the built-in HTTP server/client package.
	"net/http"
	// strings contains string utility functions.
	"strings"
	// sync provides concurrency primitives like Mutex.
	"sync"
)

// const defines an immutable value known at compile time.
// base62 is the alphabet used to encode numeric IDs into short codes.
const base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// store is a tiny in-memory URL store. Phase 2 replaces this with Postgres.
type store struct {
	// mu is a mutex; think of it as a lock to protect shared state across goroutines/threads.
	mu      sync.Mutex
	// counter is the auto-incrementing ID source.
	counter uint64
	// urls maps short code -> original URL (Go map is like dict/HashMap/unordered_map).
	urls    map[string]string // code -> long URL
}

// newStore is a constructor-style helper that allocates and initializes a store.
func newStore() *store {
	// &store creates a struct and returns its pointer (roughly like returning a reference/object handle).
	// make allocates and initializes a map; a nil map cannot be written to.
	return &store{urls: make(map[string]string)}
}

// save records a URL and returns its newly generated short code.
// (s *store) means this is a method with pointer receiver, so it can mutate s.
func (s *store) save(url string) string {
	// Lock acquires the mutex before touching shared fields.
	s.mu.Lock()
	// defer schedules Unlock to run when this function returns (like finally).
	defer s.mu.Unlock()
	// Increment first so first code is 1, not 0.
	s.counter++
	// := declares and initializes a local variable with inferred type.
	code := encodeBase62(s.counter)
	// Store the mapping code -> URL.
	s.urls[code] = url
	// return sends the result back to caller.
	return code
}

// lookup returns the URL for a code and whether it existed.
// Multiple return values are idiomatic in Go (here: value + found flag).
func (s *store) lookup(code string) (string, bool) {
	// Lock while reading shared map for thread safety.
	s.mu.Lock()
	// Ensure unlock always happens.
	defer s.mu.Unlock()
	// Map lookup returns value and ok flag in one step.
	url, ok := s.urls[code]
	// Return both values.
	return url, ok
}

// encodeBase62 converts a positive integer to a base62 string.
func encodeBase62(n uint64) string {
	// Special case: represent zero explicitly.
	if n == 0 {
		return "0"
	}
	// strings.Builder efficiently builds strings by appending bytes/chunks.
	var b strings.Builder
	// Repeatedly take least-significant base62 digit.
	for n > 0 {
		// n%62 picks the next digit index; WriteByte appends one character.
		b.WriteByte(base62[n%62])
		// Integer division shifts right in base62.
		n /= 62
	}
	// reverse
	// Digits were appended least-significant first, so reverse to get final order.
	r := []byte(b.String())
	// Classic two-pointer in-place reverse loop.
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	// Convert byte slice back to string.
	return string(r)
}

// shortenRequest models expected JSON input for POST /shorten.
type shortenRequest struct {
	// Struct tags (`json:"url"`) map this field to JSON key "url".
	// Exported field (capitalized) is required for encoding/json to access it.
	URL string `json:"url"`
}

// shortenResponse models JSON output sent back to the client.
type shortenResponse struct {
	// Code is the generated short code.
	Code  string `json:"code"`
	// Short is the full shortened URL.
	Short string `json:"short"`
}

// main is the program entry point (like main in Java/C++).
func main() {
	// Create application state.
	s := newStore()
	// NewServeMux is a request router/dispatcher.
	mux := http.NewServeMux()

	// Register handler for POST /shorten.
	// Anonymous function acts like a lambda/closure capturing s.
	mux.HandleFunc("POST /shorten", func(w http.ResponseWriter, r *http.Request) {
		// Allocate request struct to decode JSON body into.
		var req shortenRequest
		// Decode request body JSON. In Go, errors are returned explicitly (not exceptions).
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// http.Error writes status code + plain-text error message.
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			// return exits the handler early.
			return
		}
		// Minimal URL validation guard.
		if !validURL(req.URL) {
			http.Error(w, "url must start with http:// or https://", http.StatusBadRequest)
			return
		}
		// Save URL and get generated short code.
		code := s.save(req.URL)
		// Set response content type header.
		w.Header().Set("Content-Type", "application/json")
		// Encode and write JSON response directly to response writer.
		json.NewEncoder(w).Encode(shortenResponse{
			Code:  code,
			// r.Host is host:port from incoming request; build absolute short URL.
			Short: "http://" + r.Host + "/" + code,
		})
	})

	// Register handler for GET /{code}.
	// {code} is a path parameter supported by Go's newer ServeMux patterns.
	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		// Extract path parameter value named "code".
		code := r.PathValue("code")
		// Lookup original URL.
		url, ok := s.lookup(code)
		// If missing, return HTTP 404.
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// Send HTTP 302 redirect to target URL.
		http.Redirect(w, r, url, http.StatusFound)
	})

	// Listen on all interfaces, TCP port 8080.
	addr := ":8080"
	// Log startup message.
	log.Printf("shorty listening on %s", addr)
	// Start HTTP server; log.Fatal prints error and exits if server stops unexpectedly.
	log.Fatal(http.ListenAndServe(addr, mux))
}

// validURL is a minimal check; Phase 2 hardens validation.
func validURL(u string) bool {
	// Accept only absolute URLs beginning with http:// or https://.
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
