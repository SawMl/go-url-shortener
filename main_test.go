// Tests for the in-memory store, base62 encoding, and input validation (Phase 4).
package main

import (
	"strings"
	"testing"
)

func TestEncodeBase62(t *testing.T) {
	cases := map[uint64]string{
		0:  "0",
		1:  "1",
		61: "Z",
		62: "10",
	}
	for in, want := range cases {
		if got := encodeBase62(in); got != want {
			t.Errorf("encodeBase62(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestMemoryStoreSaveAndLookup(t *testing.T) {
	s := newMemoryStore()
	code, err := s.Save("https://example.com")
	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}

	got, ok, err := s.Lookup(code)
	if err != nil {
		t.Fatalf("Lookup() returned error: %v", err)
	}
	if !ok {
		t.Fatalf("Lookup(%q) returned ok=false, want true", code)
	}
	if got != "https://example.com" {
		t.Errorf("Lookup(%q) = %q, want %q", code, got, "https://example.com")
	}

	_, ok, err = s.Lookup("does-not-exist")
	if err != nil {
		t.Fatalf("Lookup() returned error: %v", err)
	}
	if ok {
		t.Error("Lookup of unknown code returned ok=true, want false")
	}
}

func TestValidURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		// Valid URLs
		{"https://example.com", true},
		{"http://example.com/path", true},
		{"https://example.com/path?query=value", true},
		{"https://sub.example.co.uk:8080/path", true},

		// Invalid: malformed
		{"not a url", false},
		{"http://", false},
		{"https://", false},

		// Invalid: wrong scheme
		{"ftp://example.com", false},
		{"file:///etc/passwd", false},
		{"", false},

		// Invalid: localhost/loopback (redirect loop prevention)
		{"http://localhost/path", false},
		{"http://127.0.0.1/path", false},
		{"http://[::1]/path", false},

		// Invalid: private IPs (internal only)
		{"http://10.0.0.1/path", false},
		{"http://192.168.1.1/path", false},
		{"http://172.16.0.1/path", false},

		// Invalid: size limit (>2048 chars)
		{"https://example.com/" + strings.Repeat("a", 2100), false},
	}

	for _, c := range cases {
		if got := validURL(c.url); got != c.want {
			t.Errorf("validURL(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}
