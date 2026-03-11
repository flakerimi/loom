package core

import (
	"path/filepath"
	"testing"

	"github.com/constructspace/loom/internal/storage"
)

func newTestStreamManager(t *testing.T) *StreamManager {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.InitDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStreamManager(db)
}

func TestStreamManager_Create(t *testing.T) {
	sm := newTestStreamManager(t)

	s, err := sm.Create("main")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if s.Name != "main" {
		t.Errorf("expected name main, got %s", s.Name)
	}
	if s.HeadSeq != 0 {
		t.Errorf("expected head_seq 0, got %d", s.HeadSeq)
	}
	if s.Status != "active" {
		t.Errorf("expected status active, got %s", s.Status)
	}
}

func TestStreamManager_GetByName(t *testing.T) {
	sm := newTestStreamManager(t)

	created, _ := sm.Create("main")
	got, err := sm.GetByName("main")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: %s != %s", got.ID, created.ID)
	}
}

func TestStreamManager_GetByName_NotFound(t *testing.T) {
	sm := newTestStreamManager(t)

	_, err := sm.GetByName("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent stream")
	}
}

func TestStreamManager_Fork(t *testing.T) {
	sm := newTestStreamManager(t)

	sm.Create("main")
	forked, err := sm.Fork("main", "feature/auth")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}

	if forked.Name != "feature/auth" {
		t.Errorf("expected name feature/auth, got %s", forked.Name)
	}
	if forked.ParentID == "" {
		t.Error("expected parent ID to be set")
	}
}

func TestStreamManager_List(t *testing.T) {
	sm := newTestStreamManager(t)

	sm.Create("main")
	sm.Create("dev")
	sm.Create("feature/x")

	streams, err := sm.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(streams) != 3 {
		t.Errorf("expected 3 streams, got %d", len(streams))
	}
}

func TestStreamManager_ActiveStream(t *testing.T) {
	sm := newTestStreamManager(t)

	sm.Create("main")
	sm.Create("dev")

	// Default active
	name, _ := sm.ActiveName()
	if name != "main" {
		t.Errorf("expected default active main, got %s", name)
	}

	// Switch
	sm.SetActive("dev")
	name, _ = sm.ActiveName()
	if name != "dev" {
		t.Errorf("expected active dev, got %s", name)
	}
}
