package core

import (
	"path/filepath"
	"testing"

	"github.com/constructspace/loom/internal/storage"
)

func setupCheckpointEnv(t *testing.T) (*CheckpointEngine, *OpWriter, *Stream) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.InitDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := storage.NewObjectStore(filepath.Join(dir, "objects"), db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sm := NewStreamManager(db)
	stream, _ := sm.Create("main")

	writer := NewOpWriter(db, store)
	reader := NewOpReader(db)
	engine := NewCheckpointEngine(db, reader)

	return engine, writer, stream
}

func TestCheckpointEngine_Create(t *testing.T) {
	engine, _, stream := setupCheckpointEnv(t)

	cp, err := engine.Create(CheckpointInput{
		StreamID: stream.ID,
		Title:    "initial",
		Author:   "test",
		Source:   SourceManual,
	})
	if err != nil {
		t.Fatalf("create checkpoint: %v", err)
	}

	if cp.Title != "initial" {
		t.Errorf("expected title initial, got %s", cp.Title)
	}
	if cp.Source != SourceManual {
		t.Errorf("expected source manual, got %s", cp.Source)
	}
	if cp.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCheckpointEngine_CreateWithOps(t *testing.T) {
	engine, writer, stream := setupCheckpointEnv(t)

	// Write some ops
	for _, entity := range []string{"a.go", "b.go", "c.go"} {
		writer.Write(Operation{
			StreamID: stream.ID,
			SpaceID:  "code",
			EntityID: entity,
			Type:     OpModify,
			Path:     entity,
			Author:   "test",
		})
	}

	cp, err := engine.Create(CheckpointInput{
		StreamID: stream.ID,
		Title:    "after changes",
		Author:   "test",
		Source:   SourceManual,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if cp.Seq != 3 {
		t.Errorf("expected seq 3, got %d", cp.Seq)
	}

	// Should have space state for code
	if len(cp.Spaces) != 1 {
		t.Errorf("expected 1 space, got %d", len(cp.Spaces))
	}
	if cp.Spaces[0].SpaceID != "code" {
		t.Errorf("expected space code, got %s", cp.Spaces[0].SpaceID)
	}
}

func TestCheckpointEngine_Get(t *testing.T) {
	engine, _, stream := setupCheckpointEnv(t)

	created, _ := engine.Create(CheckpointInput{
		StreamID: stream.ID,
		Title:    "test get",
		Author:   "test",
		Source:   SourceManual,
	})

	got, err := engine.Get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: %s != %s", got.ID, created.ID)
	}
	if got.Title != "test get" {
		t.Errorf("title mismatch: %s", got.Title)
	}
}

func TestCheckpointEngine_List(t *testing.T) {
	engine, _, stream := setupCheckpointEnv(t)

	engine.Create(CheckpointInput{StreamID: stream.ID, Title: "first", Author: "test", Source: SourceManual})
	engine.Create(CheckpointInput{StreamID: stream.ID, Title: "second", Author: "test", Source: SourceManual})
	engine.Create(CheckpointInput{StreamID: stream.ID, Title: "third", Author: "test", Source: SourceAuto})

	cps, err := engine.List(stream.ID, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(cps) != 3 {
		t.Errorf("expected 3 checkpoints, got %d", len(cps))
	}

	// Should be in reverse order (newest first)
	if cps[0].Title != "third" {
		t.Errorf("expected newest first, got %s", cps[0].Title)
	}
}

func TestCheckpointEngine_Search(t *testing.T) {
	engine, _, stream := setupCheckpointEnv(t)

	engine.Create(CheckpointInput{StreamID: stream.ID, Title: "auth system complete", Author: "test", Source: SourceManual})
	engine.Create(CheckpointInput{StreamID: stream.ID, Title: "database migration", Author: "test", Source: SourceManual})
	engine.Create(CheckpointInput{StreamID: stream.ID, Title: "auth tests added", Author: "test", Source: SourceManual})

	results, err := engine.Search("auth")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 auth results, got %d", len(results))
	}
}

func TestCheckpointEngine_SpaceSummaryBreakdown(t *testing.T) {
	engine, writer, stream := setupCheckpointEnv(t)

	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "a.go", Type: OpCreate, Path: "a.go", Author: "test"})
	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "b.go", Type: OpCreate, Path: "b.go", Author: "test"})
	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "a.go", Type: OpModify, Path: "a.go", Author: "test"})
	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "c.go", Type: OpCreate, Path: "c.go", Author: "test"})
	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "b.go", Type: OpDelete, Path: "b.go", Author: "test"})

	cp, err := engine.Create(CheckpointInput{
		StreamID: stream.ID,
		Title:    "mixed ops",
		Author:   "test",
		Source:   SourceManual,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if len(cp.Spaces) != 1 {
		t.Fatalf("expected 1 space, got %d", len(cp.Spaces))
	}

	s := cp.Spaces[0]
	if s.Summary.EntitiesCreated != 3 {
		t.Errorf("created: expected 3, got %d", s.Summary.EntitiesCreated)
	}
	if s.Summary.EntitiesModified != 1 {
		t.Errorf("modified: expected 1, got %d", s.Summary.EntitiesModified)
	}
	if s.Summary.EntitiesDeleted != 1 {
		t.Errorf("deleted: expected 1, got %d", s.Summary.EntitiesDeleted)
	}
}

func TestCheckpointEngine_ParentChain(t *testing.T) {
	engine, _, stream := setupCheckpointEnv(t)

	cp1, _ := engine.Create(CheckpointInput{StreamID: stream.ID, Title: "first", Author: "test", Source: SourceManual})
	cp2, _ := engine.Create(CheckpointInput{StreamID: stream.ID, Title: "second", Author: "test", Source: SourceManual})

	if cp2.ParentID != cp1.ID {
		t.Errorf("expected parent %s, got %s", cp1.ID, cp2.ParentID)
	}
}
