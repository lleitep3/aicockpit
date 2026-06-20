package cmd

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/kb"
	"github.com/lleite/aicockpit/internal/logging"
)

func TestNewKBCommand(t *testing.T) {
	log, err := logging.NewManager("")
	if err != nil {
		t.Fatalf("Failed to create logging manager: %v", err)
	}
	cfg := &config.Config{Version: "0.1.0", Language: "en-us"}
	translator := i18n.New("en-us")

	cmd := NewKBCommand(log, cfg, translator)

	if cmd == nil {
		t.Error("NewKBCommand() returned nil")
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
		t.Error("NewKBSearchCommand() returned nil")
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
		t.Error("NewKBListCommand() returned nil")
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
		t.Error("NewKBAddCommand() returned nil")
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
		t.Error("NewKBRemoveCommand() returned nil")
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
		t.Error("NewKBRootCommand() returned nil")
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
		t.Error("NewKBRootListCommand() returned nil")
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
		t.Error("NewKBRebuildCacheCommand() returned nil")
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
