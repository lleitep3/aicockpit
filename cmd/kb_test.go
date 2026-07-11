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

func TestNewKBRebuildCacheCommand_WithExtensions(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Write a doc so the index has something
	doc := "---\ntitle: Rebuild Ext\ntags: [ext]\n---\nContent.\n"
	os.WriteFile(filepath.Join(kbRoot, "rebuild-ext.md"), []byte(doc), 0o644)

	cmd := NewKBRebuildCacheCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--extensions"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("rebuild-cache --extensions error = %v", err)
	}
}

func TestNewKBRebuildCacheCommand_NoRoots(t *testing.T) {
	log, _ := logging.NewManager("")
	cfg := &config.Config{Version: "0.1.0", Language: "en-us", KB: config.KBConfig{Roots: []string{}}}
	tr := i18n.New("en-us")

	cmd := NewKBRebuildCacheCommand(log, cfg, tr)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Errorf("rebuild-cache no roots error = %v", err)
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

func TestKBRootAddCommand_Run_NonExistentPath(t *testing.T) {
	log, cfg, tr, _ := newKBTestDeps(t)
	cmd := NewKBRootAddCommand(log, cfg, tr)
	// Adding a non-existent path should fail
	err := cmd.RunE(cmd, []string{"/nonexistent/path/for/test"})
	if err == nil {
		t.Error("expected error for non-existent path")
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

// ── KB Graph Command ──────────────────────────────────────────────────────

func TestKBGraphCommand_NoRoots(t *testing.T) {
	log, _, _ := newTestDeps(t)
	cfg := &config.Config{Version: "0.1.0", Language: "en-us", KB: config.KBConfig{Roots: []string{}}}
	tr := i18n.New("en-us")
	cmd := NewKBGraphCommand(log, cfg, tr)
	// No roots configured — should return nil (prints message and exits)
	if err := cmd.RunE(cmd, []string{"doc-123"}); err != nil {
		t.Errorf("graph (no roots) error = %v", err)
	}
}

func TestKBGraphCommand_WithDocs(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Write two docs with cross-references using filename-based IDs
	// File "alpha.md" → ID "alpha", File "beta.md" → ID "beta"
	doc1 := "---\ntitle: Alpha Document\nrelated:\n  - beta\n---\nContent of alpha.\n"
	doc2 := "---\ntitle: Beta Document\nrelated:\n  - alpha\n---\nContent of beta.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "alpha.md"), []byte(doc1), 0o644); err != nil {
		t.Fatalf("WriteFile alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kbRoot, "beta.md"), []byte(doc2), 0o644); err != nil {
		t.Fatalf("WriteFile beta: %v", err)
	}

	cmd := NewKBGraphCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--depth", "2", "alpha"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("graph search error = %v", err)
	}
}

func TestKBGraphCommand_WithLongNames(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Use long file names to trigger ID/title truncation in graph output
	doc1 := "---\ntitle: This Is An Extremely Long Document Title That Should Trigger Truncation\nrelated:\n  - very-long-document-name-that-exceeds-twenty-characters\n---\nContent.\n"
	doc2 := "---\ntitle: Another Long Title Here Too\nrelated:\n  - graph-trunc-alpha-doc-with-long-name\n---\nContent.\n"
	os.WriteFile(filepath.Join(kbRoot, "graph-trunc-alpha-doc-with-long-name.md"), []byte(doc1), 0o644)
	os.WriteFile(filepath.Join(kbRoot, "very-long-document-name-that-exceeds-twenty-characters.md"), []byte(doc2), 0o644)

	cmd := NewKBGraphCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--depth", "2", "graph-trunc-alpha-doc-with-long-name"})
	_ = cmd.Execute()
}

func TestKBGraphCommand_DocNotFound(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Write a doc so ListDocuments succeeds
	doc := "---\ntitle: Only Doc\n---\nContent.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "only.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := NewKBGraphCommand(log, cfg, tr)
	cmd.SetArgs([]string{"nonexistent-id"})
	// Should error: document not found
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent doc ID")
	}
}

func TestKBGraphCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewKBGraphCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewKBGraphCommand() returned nil")
	}
	if cmd.Use != "graph <doc-id>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "graph <doc-id>")
	}
	if cmd.Flag("depth") == nil {
		t.Error("expected --depth flag")
	}
}

// ── KB Search with formats ────────────────────────────────────────────────

func TestKBSearchCommand_WithLongTitles(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Write doc with very long title to trigger truncation in table output
	doc := "---\ntitle: This Is A Very Long Title That Exceeds Thirty Characters Easily\ntags: [truncation]\n---\nContent with long path that needs truncation test.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "very-long-path-name-for-testing-truncation-behavior.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Test table format (exercises title/path truncation)
	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "--format", "table", "truncation"})
	_ = cmd.Execute()
}

func TestKBListCommand_WithLongTitles(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)

	// Write doc with long title and long ID (via long filename)
	doc := "---\ntitle: This Title Is Also Extremely Long and Should Trigger Truncation\ntags: [long]\n---\nContent.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "very-long-document-identifier-name-here.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Test table format
	cmd := NewKBListCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--format", "table"})
	_ = cmd.Execute()
}

func TestKBSearchCommand_WithExtensions(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	doc := "---\ntitle: Extension Test\ntags: [ext]\n---\nExtension test content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "ext.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Without --bm25, tries RunSearchExtensions first (will fail → falls back to BM25)
	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"extension"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search without --bm25 error = %v", err)
	}
}

func TestKBSearchCommand_JSONFormat(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	doc := "---\ntitle: Format Test\ntags: [json]\n---\nJSON format test content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "format.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "--format", "json", "format"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search --format json error = %v", err)
	}
}

func TestKBSearchCommand_TableFormat(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	doc := "---\ntitle: Table Test\ntags: [table]\n---\nTable format test content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "table.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "--format", "table", "table"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search --format table error = %v", err)
	}
}

func TestKBSearchCommand_WithLimit(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	// Create multiple docs that match the same query to trigger limit truncation
	doc1 := "---\ntitle: Limit Doc One\ntags: [limit]\n---\nLimit test content one about searching.\n"
	doc2 := "---\ntitle: Limit Doc Two\ntags: [limit]\n---\nLimit test content two about searching.\n"
	doc3 := "---\ntitle: Limit Doc Three\ntags: [limit]\n---\nLimit test content three about searching.\n"
	os.WriteFile(filepath.Join(kbRoot, "limit1.md"), []byte(doc1), 0o644)
	os.WriteFile(filepath.Join(kbRoot, "limit2.md"), []byte(doc2), 0o644)
	os.WriteFile(filepath.Join(kbRoot, "limit3.md"), []byte(doc3), 0o644)

	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "--limit", "1", "limit"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search --limit 1 error = %v", err)
	}
}

func TestKBSearchCommand_NoResults(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	// Create a doc that won't match the query
	doc := "---\ntitle: Unrelated\ntags: [other]\n---\nNothing relevant here.\n"
	os.WriteFile(filepath.Join(kbRoot, "unrelated.md"), []byte(doc), 0o644)

	cmd := NewKBSearchCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--bm25", "xyznonexistentterm123456"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("search no results error = %v", err)
	}
}

// ── KB List with formats ──────────────────────────────────────────────────

func TestKBListCommand_JSONFormat(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	doc := "---\ntitle: List JSON\ntags: [list]\n---\nList content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "listjson.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := NewKBListCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("list --format json error = %v", err)
	}
}

func TestKBListCommand_TableFormat(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	doc := "---\ntitle: List Table\ntags: [list]\n---\nList content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "listtable.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := NewKBListCommand(log, cfg, tr)
	cmd.SetArgs([]string{"--format", "table"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("list --format table error = %v", err)
	}
}
