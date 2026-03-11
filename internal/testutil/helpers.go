package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/constructspace/loom/internal/core"
	"github.com/constructspace/loom/internal/storage"
)

// NewTestDB creates a temporary SQLite database with schema.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.InitDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("init test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// NewTestObjectStore creates a temporary object store.
func NewTestObjectStore(t *testing.T, db *sql.DB) *storage.ObjectStore {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewObjectStore(filepath.Join(dir, "objects"), db)
	if err != nil {
		t.Fatalf("init test object store: %v", err)
	}
	return store
}

// NewTestVault creates a full temporary vault with a sample project.
func NewTestVault(t *testing.T) *core.Vault {
	t.Helper()
	dir := t.TempDir()

	// Create sample project structure
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "readme.md"), []byte("# Test Project\n\nA test project.\n"), 0644)

	vault, err := core.InitVault(dir)
	if err != nil {
		t.Fatalf("init test vault: %v", err)
	}
	t.Cleanup(func() { vault.Close() })

	return vault
}

// MakeOp creates a test operation for the given stream.
func MakeOp(streamID, spaceID, entityPath string) core.Operation {
	return core.Operation{
		StreamID: streamID,
		SpaceID:  spaceID,
		EntityID: entityPath,
		Type:     core.OpModify,
		Path:     entityPath,
		Author:   "test",
		Meta: core.OpMeta{
			Source: "test",
			Size:   100,
		},
	}
}
