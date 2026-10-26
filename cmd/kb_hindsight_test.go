package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/memoryhindsight"
)

type scriptedContributor struct {
	text     string
	err      error
	indexErr error
	queries  []string
	limits   []int
	roots    []string
	fast     bool
}

func (s *scriptedContributor) Search(_ context.Context, query string, limit int) (string, error) {
	s.queries = append(s.queries, query)
	s.limits = append(s.limits, limit)
	return s.text, s.err
}

func (s *scriptedContributor) Index(_ context.Context, roots []string, fast bool) error {
	s.roots = append(s.roots, roots...)
	s.fast = fast
	return s.indexErr
}

func TestMemoryContributorGate(t *testing.T) {
	origOpen := newHindsightIntegration
	origToken := readHindsightAPIToken
	t.Cleanup(func() {
		newHindsightIntegration = origOpen
		readHindsightAPIToken = origToken
	})

	disabled, err := origOpen(hindsight.Settings{}, memoryhindsight.Identity{Username: "ada"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if gate, ok := disabled.(interface{ Enabled() bool }); !ok || gate.Enabled() {
		t.Fatal("disabled settings should not talk to Hindsight")
	}
	if memoryContributor(nil) != nil || memoryContributor(&config.Config{}) != nil {
		t.Fatal("disabled config should stay on the local knowledge base")
	}

	var sawToken string
	newHindsightIntegration = func(_ hindsight.Settings, _ memoryhindsight.Identity, token string) (kb.Contributor, error) {
		sawToken = token
		return &scriptedContributor{text: "remote"}, nil
	}
	readHindsightAPIToken = func(namespace, key string) (string, error) {
		if namespace != "hindsight" || key != "api_token" {
			t.Fatalf("vault ref = %s/%s", namespace, key)
		}
		return "super-secret", nil
	}
	cfg := &config.Config{Memory: config.MemoryConfig{Hindsight: hindsight.Settings{
		Enabled:      true,
		APITokenName: "api_token",
	}}}
	got := memoryContributor(cfg)
	if got == nil || sawToken != "super-secret" {
		t.Fatalf("contributor missing, token = %q", sawToken)
	}

	readHindsightAPIToken = func(string, string) (string, error) {
		return "", errors.New("vault locked")
	}
	failed := memoryContributor(cfg)
	text, err := failed.Search(context.Background(), "q", 1)
	if text != "" || err == nil || strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("degraded search = %q %v", text, err)
	}

	newHindsightIntegration = func(hindsight.Settings, memoryhindsight.Identity, string) (kb.Contributor, error) {
		return nil, errors.New("bad url")
	}
	if _, err := memoryContributor(cfg).Search(context.Background(), "q", 1); err == nil || strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("open error = %v", err)
	}

	cfg.Memory.Hindsight.APITokenName = ""
	newHindsightIntegration = func(_ hindsight.Settings, _ memoryhindsight.Identity, token string) (kb.Contributor, error) {
		if token != "" {
			t.Fatalf("unexpected token %q", token)
		}
		return &memoryhindsightOff{}, nil
	}
	if memoryContributor(cfg) != nil {
		t.Fatal("a disabled integration should be dropped")
	}
	newHindsightIntegration = func(hindsight.Settings, memoryhindsight.Identity, string) (kb.Contributor, error) {
		return nil, nil
	}
	if memoryContributor(cfg) != nil {
		t.Fatal("nil integration should stay local")
	}
	origContributor := newMemoryContributor
	newMemoryContributor = nil
	attachMemoryContributor(nil, cfg)
	newMemoryContributor = origContributor
}

type memoryhindsightOff struct{}

func (memoryhindsightOff) Search(context.Context, string, int) (string, error) { return "", nil }
func (memoryhindsightOff) Index(context.Context, []string, bool) error         { return nil }
func (memoryhindsightOff) Enabled() bool                                       { return false }

func TestKBSearchIncludesHindsightAndDegrades(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	doc := "---\ntitle: Hello World\ntags: [hello]\n---\nHello world content.\n"
	if err := os.WriteFile(filepath.Join(kbRoot, "hello.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Memory.Hindsight.Enabled = true
	scripted := &scriptedContributor{text: "Hindsight recall for \"hello\" (1):\n1. Ada prefers mornings"}
	withContributor(t, scripted)

	out := captureCmdStdout(t, func() {
		cmd := NewKBSearchCommand(log, cfg, tr)
		cmd.SetArgs([]string{"--limit", "3", "hello"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("search: %v", err)
		}
	})
	if !strings.Contains(out, "Hello World") || !strings.Contains(out, "Ada prefers mornings") {
		t.Fatalf("output = %s", out)
	}
	if len(scripted.limits) != 1 || scripted.limits[0] != 3 {
		t.Fatalf("limits = %v", scripted.limits)
	}

	scripted.err = errors.New("connection refused")
	scripted.text = ""
	out = captureCmdStdout(t, func() {
		cmd := NewKBSearchCommand(log, cfg, tr)
		cmd.SetArgs([]string{"--bm25", "--format", "json", "hello"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("degraded search: %v", err)
		}
	})
	if !strings.Contains(out, "memory-hindsight search skipped") || !strings.Contains(out, "Hello World") {
		t.Fatalf("degraded output = %s", out)
	}
	if strings.Contains(out, `"hindsight"`) {
		t.Fatalf("json included an empty hindsight field: %s", out)
	}
}

func TestKBIndexRetainsThroughExtension(t *testing.T) {
	log, cfg, tr, kbRoot := newKBTestDeps(t)
	if err := os.WriteFile(filepath.Join(kbRoot, "note.md"), []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Memory.Hindsight.Enabled = true
	scripted := &scriptedContributor{}
	withContributor(t, scripted)
	captureCmdStdout(t, func() {
		cmd := NewKBRebuildCacheCommand(log, cfg, tr)
		cmd.SetArgs([]string{"--fast"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if len(scripted.roots) == 0 || !scripted.fast {
		t.Fatalf("index was not called: %+v", scripted)
	}
	scripted.indexErr = errors.New("unreachable")
	out := captureCmdStdout(t, func() {
		cmd := NewKBRebuildCacheCommand(log, cfg, tr)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "memory-hindsight index skipped") {
		t.Fatalf("output = %s", out)
	}
}

func TestKBSearchNoRootsStillShowsRecall(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cfg.KB.Roots = nil
	withContributor(t, &scriptedContributor{text: "Hindsight recall for \"q\" (1):\n1. fact"})
	out := captureCmdStdout(t, func() {
		cmd := NewKBSearchCommand(log, cfg, tr)
		cmd.SetArgs([]string{"q"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "No knowledge base roots configured") || !strings.Contains(out, "fact") {
		t.Fatalf("output = %s", out)
	}
}

func TestWriteSearchResultsIncludesRecall(t *testing.T) {
	results := &kb.SearchResults{
		Query:   "q",
		Results: []kb.SearchResult{{Title: "Local", Path: "guides/local.md", Score: 1}},
		Total:   1,
	}
	for _, format := range []string{"table", "default", "json", "other"} {
		out := captureCmdStdout(t, func() {
			if err := writeSearchResults(format, results, "hindsight line"); err != nil {
				t.Fatal(err)
			}
		})
		if !strings.Contains(out, "Local") || !strings.Contains(out, "hindsight line") {
			t.Fatalf("%s output = %s", format, out)
		}
	}
}

func withContributor(t *testing.T, contributor kb.Contributor) {
	t.Helper()
	orig := newMemoryContributor
	t.Cleanup(func() { newMemoryContributor = orig })
	newMemoryContributor = func(*config.Config) kb.Contributor { return contributor }
}

func captureCmdStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
