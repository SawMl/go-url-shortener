// Command shorty is a minimal URL shortener.
//
// Phase 4: Postgres persistence with hardened input validation.
// Endpoints:
//
//	POST /shorten  body: {"url":"https://example.com/long"}  -> {"code":"1","short":"http://host/1"}
//	GET  /{code}   -> 302 redirect to the original URL (404 if unknown)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

const base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Store defines the interface for URL storage backends.
type Store interface {
	Save(url string) (string, error)
	Lookup(code string) (string, bool, error)
}

// memoryStore is a tiny in-memory URL store. Useful for testing and local development.
type memoryStore struct {
	mu      sync.Mutex
	counter uint64
	urls    map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{urls: make(map[string]string)}
}

// Save records a URL and returns its newly generated short code.
func (s *memoryStore) Save(url string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	code := encodeBase62(s.counter)
	s.urls[code] = url

	return code, nil
}

// Lookup returns the URL for a code and whether it existed.
func (s *memoryStore) Lookup(code string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	url, ok := s.urls[code]
	return url, ok, nil
}

// postgresStore persists URLs to a Postgres database.
type postgresStore struct {
	db *sql.DB
}

func newPostgresStore(db *sql.DB) *postgresStore {
	return &postgresStore{db: db}
}

// Save records a URL and returns its newly generated short code.
func (s *postgresStore) Save(url string) (string, error) {
	var id uint64
	var code string
	err := s.db.QueryRowContext(
		context.Background(),
		"INSERT INTO urls (url) VALUES ($1) RETURNING id",
		url,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	code = encodeBase62(id)

	_, err = s.db.ExecContext(
		context.Background(),
		"UPDATE urls SET code = $1 WHERE id = $2",
		code,
		id,
	)
	if err != nil {
		return "", err
	}

	return code, nil
}

// Lookup returns the URL for a code and whether it existed.
func (s *postgresStore) Lookup(code string) (string, bool, error) {
	var url string
	err := s.db.QueryRowContext(
		context.Background(),
		"SELECT url FROM urls WHERE code = $1",
		code,
	).Scan(&url)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return url, true, nil
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

// initDB opens the database connection and runs migrations.
func initDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create urls table if it doesn't exist.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id BIGSERIAL PRIMARY KEY,
			code VARCHAR(32) UNIQUE,
			url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return nil, err
	}

	return db, nil
}

func main() {
	var store Store

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("DATABASE_URL not set; using in-memory store")
		store = newMemoryStore()
	} else {
		db, err := initDB(dbURL)
		if err != nil {
			log.Fatalf("failed to initialize database: %v", err)
		}
		defer db.Close()
		store = newPostgresStore(db)
	}

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
		code, err := store.Save(req.URL)
		if err != nil {
			log.Printf("save error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shortenResponse{
			Code:  code,
			Short: "http://" + r.Host + "/" + code,
		})
	})

	// {code} is a path parameter supported by Go's newer ServeMux patterns.
	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		url, ok, err := store.Lookup(code)
		if err != nil {
			log.Printf("lookup error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
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

// validURL checks whether u is a valid HTTP(S) URL with a reasonable size limit.
// Rejects: malformed URLs, localhost/loopback, non-absolute URLs, oversized URLs.
func validURL(u string) bool {
	// Size limit: 2048 characters (most browsers support 2000+)
	const maxURLLength = 2048
	if len(u) == 0 || len(u) > maxURLLength {
		return false
	}

	// Parse as URL to validate structure.
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	// Must be http or https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// Must have a host
	if parsed.Host == "" {
		return false
	}

	// Reject localhost, 127.0.0.1, etc. (prevent redirect loops to self)
	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}

	// Reject private IP ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	ip := net.ParseIP(host)
	if ip != nil && ip.IsPrivate() {
		return false
	}

	return true
}
