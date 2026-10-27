package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/services"
)

func TestKBAddPersistsAndSearches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	log, cfg, tr := newTestDeps(t)
	cfg.KB.Roots = nil
	source := filepath.Join(t.TempDir(), "lesson.md")
	if err := os.WriteFile(source, []byte("# Cockpit persistence\nquartzdiagnostic searchable lesson"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := NewKBAddCommand(log, cfg, tr)
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{source, "--context", "aicockpit"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Saved:") || !strings.Contains(output.String(), "ID:") {
		t.Fatal(output.String())
	}
	loadedConfig, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	manager := kb.NewManager(loadedConfig.KB.Roots, filepath.Join(config.GetCockpitDir(), ".kb-index.json"))
	results, err := manager.Search("quartzdiagnostic")
	if err != nil || results.Total != 1 {
		t.Fatalf("search after import: %+v %v", results, err)
	}
	original := results.Results[0]
	if err := os.WriteFile(source, []byte("# Updated lesson\nupdateddiagnostic searchable lesson"), 0o600); err != nil {
		t.Fatal(err)
	}
	update := NewKBAddCommand(log, loadedConfig, tr)
	update.SetArgs([]string{source, "--update", original.ID})
	if err := update.Execute(); err != nil {
		t.Fatal(err)
	}
	manager = kb.NewManager(loadedConfig.KB.Roots, filepath.Join(config.GetCockpitDir(), ".kb-index.json"))
	results, err = manager.Search("updateddiagnostic")
	if err != nil || results.Total != 1 || results.Results[0].Path != original.Path || !results.Results[0].Created.Equal(original.Created) {
		t.Fatalf("search after update: %+v %v", results, err)
	}
}

type kbSearchProbe struct {
	services.KBService
	extensions int
	searches   int
}

func (p *kbSearchProbe) Search(query string) (*kb.SearchResults, error) {
	p.searches++
	return &kb.SearchResults{Query: query}, nil
}

func (p *kbSearchProbe) RunSearchExtensions(ctx context.Context, query string) (string, error) {
	p.extensions++
	return "", fmt.Errorf("extension not configured")
}

func TestKBSearchLocalByDefault(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		flags                []string
		searches, extensions int
		wantErr              bool
	}{
		{"default", nil, 1, 0, false},
		{"bm25", []string{"--bm25"}, 1, 0, false},
		{"graphify", []string{"--graphify"}, 0, 1, true},
		{"conflict", []string{"--graphify", "--bm25"}, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probe := &kbSearchProbe{}
			original := newKBServiceFunc
			newKBServiceFunc = func([]string, string, *logging.Manager) services.KBService { return probe }
			t.Cleanup(func() { newKBServiceFunc = original })
			log, cfg, tr := newTestDeps(t)
			cfg.KB.Roots = []string{t.TempDir()}
			cmd := NewKBSearchCommand(log, cfg, tr)
			cmd.SetArgs(append(tc.flags, "query"))
			err := cmd.Execute()
			if (err != nil) != tc.wantErr || probe.searches != tc.searches || probe.extensions != tc.extensions {
				t.Fatalf("calls=%+v err=%v", probe, err)
			}
		})
	}
}

func TestKBAddReportsPartialFailure(t *testing.T) {
	for _, tc := range []struct{ name, obstruction, message string }{
		{"index", ".kb-index.json", "indexing failed"},
		{"config", "config.yaml", "saving roots failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			log, cfg, tr := newTestDeps(t)
			if err := os.MkdirAll(filepath.Join(config.GetCockpitDir(), tc.obstruction), 0o755); err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(t.TempDir(), "knowledge")
			if tc.name == "index" {
				cfg.KB.Roots = []string{root}
			}
			source := filepath.Join(t.TempDir(), "lesson.md")
			if err := os.WriteFile(source, []byte("# Durable knowledge\nConfirmed procedure"), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := NewKBAddCommand(log, cfg, tr)
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{source, "--root", root, "--context", "test"})
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.message) || !strings.Contains(err.Error(), "document saved at") {
				t.Fatalf("partial failure: %v", err)
			}
			if strings.Contains(output.String(), "Saved:") {
				t.Fatal("reported success")
			}
			docs, err := kb.NewFileRepository(root).ListDocuments(".")
			if err != nil || len(docs) != 1 {
				t.Fatalf("saved file missing: %v", err)
			}
		})
	}
}
