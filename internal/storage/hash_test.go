package storage

import (
	"testing"
)

func TestHashContent_Deterministic(t *testing.T) {
	content := []byte("hello world")
	hash1 := HashContent(content)
	hash2 := HashContent(content)

	if hash1 != hash2 {
		t.Errorf("hash not deterministic: %s != %s", hash1, hash2)
	}
}

func TestHashContent_DifferentContent(t *testing.T) {
	hash1 := HashContent([]byte("hello"))
	hash2 := HashContent([]byte("world"))

	if hash1 == hash2 {
		t.Error("different content produced same hash")
	}
}

func TestHashContent_EmptyContent(t *testing.T) {
	hash := HashContent([]byte{})
	if hash == "" {
		t.Error("empty content produced empty hash")
	}
	if len(hash) != 64 {
		t.Errorf("expected 64 char hex hash, got %d chars", len(hash))
	}
}

func TestHashContent_Length(t *testing.T) {
	hash := HashContent([]byte("test content"))
	if len(hash) != 64 {
		t.Errorf("expected 64 char SHA-256 hex, got %d", len(hash))
	}
}
