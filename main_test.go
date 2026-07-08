// Tests for the in-memory store and base62 encoding (Phase 2).
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

func TestMemoryStoreCodesAreUnique(t *testing.T) {
	s := newMemoryStore()
	a, err := s.Save("https://a.com")
	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	b, err := s.Save("https://b.com")
	if err != nil {
		t.Fatalf("Save() returned error: %v", err)
	}
	if a == b {
		t.Errorf("expected unique codes, got %q twice", a)
	}
}
