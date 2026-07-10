package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/logging"
)

// newKBTestDeps creates test deps with a cockpit dir in tmpDir/.cockpit so
// cfg.Save() can persist during tests.
func newKBTestDeps(t *testing.T) (*logging.Manager, *config.Config, *i18n.Translator, string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll cockpit: %v", err)
	}
	log, err := logging.NewManager(filepath.Join(cockpitDir, "logs"))
	if err != nil {
		t.Fatalf("logging.NewManager: %v", err)
	}
	kbRoot := filepath.Join(cockpitDir, "kb")
	if err := os.MkdirAll(kbRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll kb: %v", err)
	}
	cfg := &config.Config{
		Version:  "0.1.0",
		Language: "en-us",
		KB:       config.KBConfig{Roots: []string{kbRoot}},
	}
	return log, cfg, i18n.New("en-us"), kbRoot
}

func TestNewKBCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	cmd := NewKBCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBCommand() returned nil")
	}

	if cmd.Use != "kb" {
		t.Errorf("NewKBCommand() Use = %s, want kb", cmd.Use)
	}

	// Check subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) < 4 {
		t.Errorf("NewKBCommand() has %d subcommands, want at least 4", len(subcommands))
	}
}

func TestNewKBSearchCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	cmd := NewKBSearchCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBSearchCommand() returned nil")
	}

	if cmd.Use != "search <query>" {
		t.Errorf("NewKBSearchCommand() Use = %s, want search <query>", cmd.Use)
	}
}

func TestNewKBListCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	cmd := NewKBListCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBListCommand() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("NewKBListCommand() Use = %s, want list", cmd.Use)
	}
}

func TestNewKBAddCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	cmd := NewKBAddCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBAddCommand() returned nil")
	}

	if cmd.Use != "add <file>" {
		t.Errorf("NewKBAddCommand() Use = %s, want add <file>", cmd.Use)
	}
}

func TestNewKBRemoveCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	cmd := NewKBRemoveCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBRemoveCommand() returned nil")
	}

	if cmd.Use != "remove <id>" {
		t.Errorf("NewKBRemoveCommand() Use = %s, want remove <id>", cmd.Use)
	}
}

func TestOutputJSON(t *testing.T) {
	results := &kb.SearchResults{
		Query: "test",
		Results: []kb.SearchResult{
			{
				ID:    "doc1",
				Title: "Test Document",
				Score: 0.95,
			},
		},
		Total: 1,
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(results)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("outputJSON() error = %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("outputJSON() produced no output")
	}

	if !bytes.Contains([]byte(output), []byte("test")) {
		t.Error("outputJSON() output missing query")
	}
}

func TestOutputDefault(t *testing.T) {
	results := &kb.SearchResults{
		Query: "test",
		Results: []kb.SearchResult{
			{
				ID:    "doc1",
				Title: "Test Document",
				Score: 0.95,
			},
		},
		Total: 1,
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputDefault(results)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("outputDefault() error = %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("outputDefault() produced no output")
	}

	if !bytes.Contains([]byte(output), []byte("Test Document")) {
		t.Error("outputDefault() output missing title")
	}
}

func TestOutputTable(t *testing.T) {
	results := &kb.SearchResults{
		Query: "test",
		Results: []kb.SearchResult{
			{
				ID:    "doc1",
				Title: "Test Document",
				Score: 0.95,
				Path:  "guides/test.md",
			},
		},
		Total: 1,
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputTable(results)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("outputTable() error = %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("outputTable() produced no output")
	}

	if !bytes.Contains([]byte(output), []byte("Test Document")) {
		t.Error("outputTable() output missing title")
	}
}

func TestOutputDocumentsJSON(t *testing.T) {
	documents := []*kb.Document{
		{
			ID:   "doc1",
			Path: "guides/test.md",
			Metadata: kb.Metadata{
				Title:       "Test Document",
				Description: "A test",
				Tags:        []string{"test"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
		},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputDocumentsJSON(documents)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("outputDocumentsJSON() error = %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("outputDocumentsJSON() produced no output")
	}

	if !bytes.Contains([]byte(output), []byte("Test Document")) {
		t.Error("outputDocumentsJSON() output missing title")
	}
}

func TestOutputDocumentsDefault(t *testing.T) {
	documents := []*kb.Document{
		{
			ID:   "doc1",
			Path: "guides/test.md",
			Metadata: kb.Metadata{
				Title:       "Test Document",
				Description: "A test",
				Tags:        []string{"test"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
		},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputDocumentsDefault(documents)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("outputDocumentsDefault() error = %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("outputDocumentsDefault() produced no output")
	}

	if !bytes.Contains([]byte(output), []byte("Test Document")) {
		t.Error("outputDocumentsDefault() output missing title")
	}
}

func TestOutputDocumentsTable(t *testing.T) {
	documents := []*kb.Document{
		{
			ID:   "doc1",
			Path: "guides/test.md",
			Metadata: kb.Metadata{
				Title:       "Test Document",
				Description: "A test",
				Tags:        []string{"test"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
		},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputDocumentsTable(documents)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("outputDocumentsTable() error = %v", err)
	}

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("outputDocumentsTable() produced no output")
	}

	if !bytes.Contains([]byte(output), []byte("Test Document")) {
		t.Error("outputDocumentsTable() output missing title")
	}
}

func TestKBSearchCommand_Integration(t *testing.T) {
	// Create temporary KB directory
	tmpDir := t.TempDir()

	// Create a test document
	repo := kb.NewFileRepository(tmpDir)
	doc := &kb.Document{
		ID:   "test",
		Path: "guides/test.md",
		Metadata: kb.Metadata{
			Title:       "Test Document",
			Description: "A test document",
			Tags:        []string{"test"},
			Created:     time.Now(),
			Modified:    time.Now(),
		},
		Content: "This is test content about logging.",
	}

	if err := repo.SaveDocument(doc); err != nil {
		t.Fatalf("Failed to save test document: %v", err)
	}

	// Verify document was saved
	loaded, err := repo.LoadDocument(doc.Path)
	if err != nil {
		t.Fatalf("Failed to load test document: %v", err)
	}

	if loaded.Metadata.Title != "Test Document" {
		t.Errorf("Loaded document title = %s, want Test Document", loaded.Metadata.Title)
	}
}

func TestNewKBRootCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{
		Version:  "0.1.0",
		Language: "en-us",
		KB:       config.KBConfig{Roots: []string{"/tmp/kb"}},
	}
	translator := i18n.New("en-us")

	cmd := NewKBRootCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBRootCommand() returned nil")
	}

	if cmd.Use != "root" {
		t.Errorf("NewKBRootCommand() Use = %s, want root", cmd.Use)
	}

	// Check subcommands exist
	subcommands := cmd.Commands()
	if len(subcommands) != 3 {
		t.Errorf("NewKBRootCommand() has %d subcommands, want 3", len(subcommands))
	}
}

func TestNewKBRootListCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{
		Version:  "0.1.0",
		Language: "en-us",
		KB:       config.KBConfig{Roots: []string{"/tmp/kb1", "/tmp/kb2"}},
	}
	translator := i18n.New("en-us")

	cmd := NewKBRootListCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBRootListCommand() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("NewKBRootListCommand() Use = %s, want list", cmd.Use)
	}

	// Test execution - just verify it doesn't error
	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("NewKBRootListCommand() RunE() error = %v", err)
	}
}

func TestNewKBRebuildCacheCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Version:  "0.1.0",
		Language: "en-us",
		KB:       config.KBConfig{Roots: []string{tmpDir}},
	}
	translator := i18n.New("en-us")

	cmd := NewKBRebuildCacheCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewKBRebuildCacheCommand() returned nil")
	}

	if cmd.Use != "rebuild-cache" {
		t.Errorf("NewKBRebuildCacheCommand() Use = %s, want rebuild-cache", cmd.Use)
	}

	// Test execution - just verify it doesn't error
	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Errorf("NewKBRebuildCacheCommand() RunE() error = %v", err)
	}
}

// ── RunE execution tests ──────────────────────────────────────────────────

func TestKBSearchCommand_NoRoots(t *testing.T) {
	log, _, _ := newTestDeps(t)
	cfg := &config.Config{Version: "0.1.0", Language: "en-us", KB: config.KBConfig{Roots: []string{}}}
	tr := i18n.New("en-us")
	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "query"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search (no roots) error = %v", err)
	}
}

func TestKBSearchCommand_WithResults(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Write a markdown doc so search returns at least one result.
	doc := "---\ntitle: Hello World\ntags: [hello]\n---\nHello world content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "hello.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "hello"})
	// May return 0 results if index not built — still must not error.
	if err := cmd.Execute(); err != nil {
		t.Errorf("search with doc error = %v", err)
	}
}

func TestKBListCommand_NoRoots(t *testing.T) {
	log, _, _ := newTestDeps(t)
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")
	cmd := NewKBListCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Errorf("list (no roots) error = %v", err)
	}
}

func TestKBListCommand_EmptyRoot(t *testing.T) {
	log, cfg, tr, _ := newKBTestDeps(t)
	cmd := NewKBListCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Errorf("list (empty root) error = %v", err)
	}
}

func TestKBAddCommand_Run(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewKBAddCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{"/some/file.md"}); err != nil {
		t.Errorf("add run error = %v", err)
	}
}

func TestKBRemoveCommand_Run(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewKBRemoveCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{"some-id"}); err != nil {
		t.Errorf("remove run error = %v", err)
	}
}

func TestKBRootAddCommand_Run(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	newRoot := filepath.Join(kbRoot, "extra")
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cmd := NewKBRootAddCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{newRoot}); err != nil {
		t.Errorf("root add error = %v", err)
	}
}

func TestKBRootRemoveCommand_Run(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	cmd := NewKBRootRemoveCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{kbRoot}); err != nil {
		t.Errorf("root remove error = %v", err)
	}
}

func TestKBRootRemoveCommand_NotFound(t *testing.T) {
	log, cfg, tr, _ := newKBTestDeps(t)
	cmd := NewKBRootRemoveCommand(log, cfg, tr)
	err := cmd.RunE(cmd, []string{"/nonexistent/path"})
	if err == nil {
		t.Error("root remove nonexistent should return error")
	}
}

func TestKBRootListCommand_NoRoots(t *testing.T) {
	log, _, _ := newTestDeps(t)
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	tr := i18n.New("en-us")
	cmd := NewKBRootListCommand(log, cfg, tr)
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Errorf("root list (no roots) error = %v", err)
	}
}
