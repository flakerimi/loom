package storage

import (
	"path/filepath"
	"testing"
)

func TestInitDB(t *testing.T) {
	dir := t.TempDir()
	db, err := InitDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	// Verify schema was applied
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='operations'").Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected operations table, got count %d", count)
	}
}

func TestInitDB_SeqCounter(t *testing.T) {
	dir := t.TempDir()
	db, err := InitDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	var val string
	err = db.QueryRow("SELECT value FROM metadata WHERE key = 'seq_counter'").Scan(&val)
	if err != nil {
		t.Fatalf("query seq: %v", err)
	}
	if val != "0" {
		t.Errorf("expected seq_counter=0, got %s", val)
	}
}

func TestNextSeq(t *testing.T) {
	dir := t.TempDir()
	db, err := InitDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	seq1, err := NextSeq(db)
	if err != nil {
		t.Fatalf("next seq 1: %v", err)
	}
	if seq1 != 1 {
		t.Errorf("expected seq 1, got %d", seq1)
	}

	seq2, err := NextSeq(db)
	if err != nil {
		t.Fatalf("next seq 2: %v", err)
	}
	if seq2 != 2 {
		t.Errorf("expected seq 2, got %d", seq2)
	}

	seq3, err := NextSeq(db)
	if err != nil {
		t.Fatalf("next seq 3: %v", err)
	}
	if seq3 != 3 {
		t.Errorf("expected seq 3, got %d", seq3)
	}
}

func TestOpenDB_NotFound(t *testing.T) {
	_, err := OpenDB("/nonexistent/path.db")
	if err == nil {
		t.Error("expected error for nonexistent db")
	}
}
