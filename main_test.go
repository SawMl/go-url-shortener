// Tests for the in-memory store and base62 encoding (Phase 1).
package main

import "testing"

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

func TestStoreSaveAndLookup(t *testing.T) {
	s := newStore()
	code := s.save("https://example.com")

	got, ok := s.lookup(code)
	if !ok {
		t.Fatalf("lookup(%q) returned ok=false, want true", code)
	}
	if got != "https://example.com" {
		t.Errorf("lookup(%q) = %q, want %q", code, got, "https://example.com")
	}

	if _, ok := s.lookup("does-not-exist"); ok {
		t.Error("lookup of unknown code returned ok=true, want false")
	}
}

func TestStoreCodesAreUnique(t *testing.T) {
	s := newStore()
	a := s.save("https://a.com")
	b := s.save("https://b.com")
	if a == b {
		t.Errorf("expected unique codes, got %q twice", a)
	}
}
