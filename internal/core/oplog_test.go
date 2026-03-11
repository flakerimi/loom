package core

import (
	"path/filepath"
	"testing"

	"github.com/constructspace/loom/internal/storage"
)

func setupTestEnv(t *testing.T) (*OpWriter, *OpReader, *Stream) {
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
	stream, err := sm.Create("main")
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	writer := NewOpWriter(db, store)
	reader := NewOpReader(db)

	return writer, reader, stream
}

func TestOpWriter_Write(t *testing.T) {
	writer, _, stream := setupTestEnv(t)

	op := Operation{
		StreamID: stream.ID,
		SpaceID:  "code",
		EntityID: "src/main.go",
		Type:     OpCreate,
		Path:     "src/main.go",
		Author:   "test",
		Meta:     OpMeta{Size: 100},
	}

	result, err := writer.Write(op)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	if result.Seq != 1 {
		t.Errorf("expected seq 1, got %d", result.Seq)
	}
	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
	if result.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestOpWriter_SequenceIncrement(t *testing.T) {
	writer, _, stream := setupTestEnv(t)

	for i := 0; i < 5; i++ {
		op := Operation{
			StreamID: stream.ID,
			SpaceID:  "code",
			EntityID: "file.go",
			Type:     OpModify,
			Path:     "file.go",
			Author:   "test",
		}

		result, err := writer.Write(op)
		if err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		if result.Seq != int64(i+1) {
			t.Errorf("write %d: expected seq %d, got %d", i, i+1, result.Seq)
		}
	}
}

func TestOpReader_Head(t *testing.T) {
	writer, reader, stream := setupTestEnv(t)

	// Initially 0
	head, err := reader.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if head != 0 {
		t.Errorf("expected head 0, got %d", head)
	}

	// Write some ops
	for i := 0; i < 3; i++ {
		writer.Write(Operation{
			StreamID: stream.ID,
			SpaceID:  "code",
			EntityID: "file.go",
			Type:     OpModify,
			Path:     "file.go",
			Author:   "test",
		})
	}

	head, err = reader.Head()
	if err != nil {
		t.Fatalf("head after writes: %v", err)
	}
	if head != 3 {
		t.Errorf("expected head 3, got %d", head)
	}
}

func TestOpReader_ReadRange(t *testing.T) {
	writer, reader, stream := setupTestEnv(t)

	// Write 5 ops
	for i := 0; i < 5; i++ {
		writer.Write(Operation{
			StreamID: stream.ID,
			SpaceID:  "code",
			EntityID: "file.go",
			Type:     OpModify,
			Path:     "file.go",
			Author:   "test",
		})
	}

	// Read range [2, 4] (exclusive start, inclusive end)
	ops, err := reader.ReadRange(2, 4)
	if err != nil {
		t.Fatalf("read range: %v", err)
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 ops, got %d", len(ops))
	}
	if ops[0].Seq != 3 {
		t.Errorf("expected first op seq 3, got %d", ops[0].Seq)
	}
	if ops[1].Seq != 4 {
		t.Errorf("expected second op seq 4, got %d", ops[1].Seq)
	}
}

func TestOpReader_ReadBySpace(t *testing.T) {
	writer, reader, stream := setupTestEnv(t)

	// Write mixed space ops
	for _, space := range []string{"code", "docs", "code", "code", "docs"} {
		writer.Write(Operation{
			StreamID: stream.ID,
			SpaceID:  space,
			EntityID: "file",
			Type:     OpModify,
			Path:     "file",
			Author:   "test",
		})
	}

	codeOps, err := reader.ReadBySpace("code", 0, 100)
	if err != nil {
		t.Fatalf("read by space: %v", err)
	}
	if len(codeOps) != 3 {
		t.Errorf("expected 3 code ops, got %d", len(codeOps))
	}

	docsOps, _ := reader.ReadBySpace("docs", 0, 100)
	if len(docsOps) != 2 {
		t.Errorf("expected 2 docs ops, got %d", len(docsOps))
	}
}

func TestOpReader_ReadByEntity(t *testing.T) {
	writer, reader, stream := setupTestEnv(t)

	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "a.go", Type: OpCreate, Path: "a.go", Author: "test"})
	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "b.go", Type: OpCreate, Path: "b.go", Author: "test"})
	writer.Write(Operation{StreamID: stream.ID, SpaceID: "code", EntityID: "a.go", Type: OpModify, Path: "a.go", Author: "test"})

	ops, err := reader.ReadByEntity("a.go")
	if err != nil {
		t.Fatalf("read by entity: %v", err)
	}
	if len(ops) != 2 {
		t.Errorf("expected 2 ops for a.go, got %d", len(ops))
	}
}

func TestOpWriter_WriteBatch(t *testing.T) {
	writer, reader, stream := setupTestEnv(t)

	ops := []Operation{
		{StreamID: stream.ID, SpaceID: "code", EntityID: "a.go", Type: OpCreate, Path: "a.go", Author: "test"},
		{StreamID: stream.ID, SpaceID: "docs", EntityID: "readme.md", Type: OpCreate, Path: "readme.md", Author: "test"},
		{StreamID: stream.ID, SpaceID: "code", EntityID: "b.go", Type: OpCreate, Path: "b.go", Author: "test"},
	}

	result, err := writer.WriteBatch(ops)
	if err != nil {
		t.Fatalf("write batch: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 results, got %d", len(result))
	}

	head, _ := reader.Head()
	if head != 3 {
		t.Errorf("expected head 3, got %d", head)
	}
}
