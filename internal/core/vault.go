package core

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/constructspace/loom/internal/storage"
)

const (
	LoomDir    = ".loom"
	DBFile     = "loom.db"
	ConfigFile = "config.toml"
	ObjectsDir = "objects"
)

var ErrNotInit = errors.New("not a loom project (run 'loom init')")

// Vault represents an initialized Loom project.
type Vault struct {
	ProjectPath string
	LoomPath    string
	Config      *ProjectConfig
	DB          *sql.DB
	Store       *storage.ObjectStore
	Streams     *StreamManager
	OpWriter    *OpWriter
	OpReader    *OpReader
	Checkpoints *CheckpointEngine
}

// InitVault initializes a new Loom project at the given path.
func InitVault(projectPath string) (*Vault, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	loomPath := filepath.Join(absPath, LoomDir)

	// Check if already initialized
	if _, err := os.Stat(loomPath); err == nil {
		return nil, fmt.Errorf("already initialized at %s", loomPath)
	}

	// Create directory structure
	dirs := []string{
		loomPath,
		filepath.Join(loomPath, ObjectsDir),
		filepath.Join(loomPath, "locks"),
		filepath.Join(loomPath, "hooks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	// Initialize database
	db, err := storage.InitDB(filepath.Join(loomPath, DBFile))
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// Initialize object store
	store, err := storage.NewObjectStore(filepath.Join(loomPath, ObjectsDir), db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init object store: %w", err)
	}

	// Detect author from git config
	author := detectAuthor()

	// Detect spaces
	spaces := detectSpaces(absPath)

	// Build config
	cfg := &ProjectConfig{
		Project: ProjectInfo{
			Name:    filepath.Base(absPath),
			Version: 1,
		},
		Author: author,
		Spaces: spaces,
		Watch: WatchConfig{
			Enabled:    true,
			DebounceMs: 500,
			Ignore:     []string{".git", "node_modules", "dist", "build", ".loom", "*.tmp", "*.swp"},
		},
		CPoint: CheckpointConfig{
			Auto:                true,
			IntervalOps:         50,
			IntervalSeconds:     300,
			OnSignificantChange: true,
		},
	}

	// Write config
	configPath := filepath.Join(loomPath, ConfigFile)
	f, err := os.Create(configPath)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create config: %w", err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		db.Close()
		return nil, fmt.Errorf("write config: %w", err)
	}
	f.Close()

	// Build vault
	v := &Vault{
		ProjectPath: absPath,
		LoomPath:    loomPath,
		Config:      cfg,
		DB:          db,
		Store:       store,
		Streams:     NewStreamManager(db),
		OpWriter:    NewOpWriter(db, store),
		OpReader:    NewOpReader(db),
	}
	v.Checkpoints = NewCheckpointEngine(db, v.OpReader)

	// Create main stream
	stream, err := v.Streams.Create("main")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create main stream: %w", err)
	}
	if err := v.Streams.SetActive("main"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set active stream: %w", err)
	}

	// Scan and record initial entities
	entityCount := v.scanEntities(stream)
	_ = entityCount

	return v, nil
}

// OpenVault opens an existing Loom project.
func OpenVault(projectPath string) (*Vault, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// Walk up to find .loom directory
	loomPath, err := findLoomDir(absPath)
	if err != nil {
		return nil, err
	}

	// Open database
	db, err := storage.OpenDB(filepath.Join(loomPath, DBFile))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Open object store
	store, err := storage.NewObjectStore(filepath.Join(loomPath, ObjectsDir), db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open object store: %w", err)
	}

	// Read config
	cfg := &ProjectConfig{}
	configPath := filepath.Join(loomPath, ConfigFile)
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		db.Close()
		return nil, fmt.Errorf("read config: %w", err)
	}

	v := &Vault{
		ProjectPath: filepath.Dir(loomPath),
		LoomPath:    loomPath,
		Config:      cfg,
		DB:          db,
		Store:       store,
		Streams:     NewStreamManager(db),
		OpWriter:    NewOpWriter(db, store),
		OpReader:    NewOpReader(db),
	}
	v.Checkpoints = NewCheckpointEngine(db, v.OpReader)

	return v, nil
}

// Close releases all resources.
func (v *Vault) Close() error {
	if v.DB != nil {
		return v.DB.Close()
	}
	return nil
}

// ActiveStream returns the currently active stream.
func (v *Vault) ActiveStream() (*Stream, error) {
	name, err := v.Streams.ActiveName()
	if err != nil {
		return nil, err
	}
	return v.Streams.GetByName(name)
}

// EntityCount returns the number of tracked entities per space.
func (v *Vault) EntityCount() (map[string]int, error) {
	rows, err := v.DB.Query("SELECT space_id, COUNT(*) FROM entities WHERE status = 'active' GROUP BY space_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var space string
		var count int
		rows.Scan(&space, &count)
		counts[space] = count
	}
	return counts, nil
}

// scanEntities performs initial scan of all spaces and records entities.
func (v *Vault) scanEntities(stream *Stream) int {
	total := 0
	for spaceID, spaceCfg := range v.Config.Spaces {
		spacePath := filepath.Join(v.ProjectPath, spaceCfg.Path)
		if _, err := os.Stat(spacePath); os.IsNotExist(err) {
			continue
		}

		entities := scanDirectory(spacePath, spaceID, v.Config.Watch.Ignore)
		for _, e := range entities {
			// Read content and store in object store
			fullPath := filepath.Join(spacePath, e.Path)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			hash, err := v.Store.Write(content, e.Meta["content_type"])
			if err != nil {
				continue
			}

			op := Operation{
				StreamID:  stream.ID,
				SpaceID:   spaceID,
				EntityID:  e.Path,
				Type:      OpCreate,
				Path:      e.Path,
				ObjectRef: hash,
				Author:    v.Config.Author.Name,
				Meta: OpMeta{
					Size:   int64(len(content)),
					Source: "init",
				},
			}

			v.OpWriter.Write(op)
			total++
		}
	}
	return total
}

// scanDirectory walks a directory and returns entity states.
func scanDirectory(root, spaceID string, ignoreRules []string) []EntityState {
	var entities []EntityState

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			// Check if directory should be ignored
			if info != nil && info.IsDir() {
				for _, rule := range ignoreRules {
					if matched, _ := filepath.Match(rule, info.Name()); matched {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		// Check file ignore rules
		for _, rule := range ignoreRules {
			if matched, _ := filepath.Match(rule, info.Name()); matched {
				return nil
			}
		}

		rel, _ := filepath.Rel(root, path)
		entities = append(entities, EntityState{
			ID:      rel,
			SpaceID: spaceID,
			Kind:    "file",
			Path:    rel,
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05Z"),
			Meta:    map[string]string{"content_type": detectMIME(rel)},
		})
		return nil
	})

	return entities
}

func findLoomDir(startPath string) (string, error) {
	current := startPath
	for {
		loomPath := filepath.Join(current, LoomDir)
		if info, err := os.Stat(loomPath); err == nil && info.IsDir() {
			return loomPath, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrNotInit
		}
		current = parent
	}
}

func detectAuthor() AuthorInfo {
	author := AuthorInfo{Name: "local-user"}

	if name, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		author.Name = strings.TrimSpace(string(name))
	}
	if email, err := exec.Command("git", "config", "user.email").Output(); err == nil {
		author.Email = strings.TrimSpace(string(email))
	}

	return author
}

func detectSpaces(projectPath string) map[string]SpaceConfig {
	spaces := make(map[string]SpaceConfig)

	// Code (git repo or common project files)
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); err == nil {
		spaces["code"] = SpaceConfig{Adapter: "git", Path: "."}
	} else {
		indicators := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "Makefile", "CMakeLists.txt"}
		for _, f := range indicators {
			if _, err := os.Stat(filepath.Join(projectPath, f)); err == nil {
				spaces["code"] = SpaceConfig{Adapter: "filesystem", Path: "."}
				break
			}
		}
	}

	// Docs
	docDirs := []string{"docs", "doc", "documentation"}
	for _, d := range docDirs {
		if info, err := os.Stat(filepath.Join(projectPath, d)); err == nil && info.IsDir() {
			spaces["docs"] = SpaceConfig{Adapter: "filesystem", Path: d}
			break
		}
	}

	// Design
	designDirs := []string{"design", "ui", ".design"}
	for _, d := range designDirs {
		if info, err := os.Stat(filepath.Join(projectPath, d)); err == nil && info.IsDir() {
			spaces["design"] = SpaceConfig{Adapter: "design", Path: d}
			break
		}
	}

	// Notes
	noteDirs := []string{"notes", "journal", ".notes"}
	for _, d := range noteDirs {
		if info, err := os.Stat(filepath.Join(projectPath, d)); err == nil && info.IsDir() {
			spaces["notes"] = SpaceConfig{Adapter: "filesystem", Path: d}
			break
		}
	}

	return spaces
}

func detectMIME(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mimeMap := map[string]string{
		".go":   "text/x-go",
		".js":   "text/javascript",
		".ts":   "text/typescript",
		".py":   "text/x-python",
		".rs":   "text/x-rust",
		".md":   "text/markdown",
		".json": "application/json",
		".yaml": "text/yaml",
		".yml":  "text/yaml",
		".toml": "text/toml",
		".html": "text/html",
		".css":  "text/css",
		".sql":  "text/x-sql",
		".txt":  "text/plain",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".svg":  "image/svg+xml",
		".gif":  "image/gif",
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
