package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitVault(t *testing.T) {
	dir := t.TempDir()

	// Create some project files
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "readme.md"), []byte("# Hello"), 0644)

	vault, err := InitVault(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer vault.Close()

	// .loom directory should exist
	if _, err := os.Stat(filepath.Join(dir, ".loom")); os.IsNotExist(err) {
		t.Error(".loom directory not created")
	}

	// Config should exist
	if _, err := os.Stat(filepath.Join(dir, ".loom", "config.toml")); os.IsNotExist(err) {
		t.Error("config.toml not created")
	}

	// Database should exist
	if _, err := os.Stat(filepath.Join(dir, ".loom", "loom.db")); os.IsNotExist(err) {
		t.Error("loom.db not created")
	}

	// Should detect code and docs spaces
	if _, ok := vault.Config.Spaces["code"]; !ok {
		t.Error("code space not detected")
	}
	if _, ok := vault.Config.Spaces["docs"]; !ok {
		t.Error("docs space not detected")
	}

	// Main stream should exist
	stream, err := vault.ActiveStream()
	if err != nil {
		t.Fatalf("active stream: %v", err)
	}
	if stream.Name != "main" {
		t.Errorf("expected main stream, got %s", stream.Name)
	}
}

func TestInitVault_AlreadyInit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	vault, err := InitVault(dir)
	if err != nil {
		t.Fatalf("first init: %v", err)
	}
	vault.Close()

	// Second init should fail
	_, err = InitVault(dir)
	if err == nil {
		t.Error("expected error on double init")
	}
}

func TestOpenVault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	v1, err := InitVault(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	v1.Close()

	v2, err := OpenVault(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer v2.Close()

	if v2.Config.Project.Name != filepath.Base(dir) {
		t.Errorf("project name mismatch: %s", v2.Config.Project.Name)
	}
}

func TestOpenVault_NotInit(t *testing.T) {
	dir := t.TempDir()
	_, err := OpenVault(dir)
	if err == nil {
		t.Error("expected error for non-loom directory")
	}
}

func TestVault_EntityCount(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "docs", "readme.md"), []byte("# Hello"), 0644)

	vault, err := InitVault(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer vault.Close()

	counts, err := vault.EntityCount()
	if err != nil {
		t.Fatalf("entity count: %v", err)
	}

	// Should have entities in code and docs
	totalCode := counts["code"]
	totalDocs := counts["docs"]
	if totalCode == 0 {
		t.Error("expected code entities")
	}
	if totalDocs == 0 {
		t.Error("expected docs entities")
	}
}

func TestFindLoomDir_WalkUp(t *testing.T) {
	dir := t.TempDir()

	// Create project with nested dirs
	os.MkdirAll(filepath.Join(dir, "src", "pkg", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	vault, err := InitVault(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	vault.Close()

	// Open from a nested directory
	v2, err := OpenVault(filepath.Join(dir, "src", "pkg", "deep"))
	if err != nil {
		t.Fatalf("open from nested: %v", err)
	}
	defer v2.Close()

	if v2.ProjectPath != dir {
		t.Errorf("expected project path %s, got %s", dir, v2.ProjectPath)
	}
}
