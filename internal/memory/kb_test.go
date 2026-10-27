package memory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/kb"
)

func newTestKB(t *testing.T) *KB {
	t.Helper()
	mem, _ := newTestKBRoot(t)
	return mem
}

func newTestKBRoot(t *testing.T) (*KB, string) {
	t.Helper()
	root := t.TempDir()
	index := filepath.Join(t.TempDir(), "index.json")
	mem, err := NewKB([]string{root}, index)
	if err != nil {
		t.Fatalf("NewKB: %v", err)
	}
	return mem, root
}

func TestNewKBRejectsBadConfig(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tests := []struct {
		name    string
		roots   []string
		index   string
		want    string
		wantErr error
	}{
		{name: "no roots", index: "idx", want: "no roots"},
		{name: "empty index", roots: []string{dir}, index: "  ", want: "empty index path"},
		{name: "missing root", roots: []string{filepath.Join(dir, "missing")}, index: "idx", wantErr: os.ErrNotExist},
		{name: "file root", roots: []string{file}, index: "idx", want: "not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKB(tt.roots, tt.index)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.want != "" && !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want text %q", err, tt.want)
			}
		})
	}
}

func TestNewKBWithManager(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		mgr  *kb.Manager
		want string
	}{
		{name: "nil", want: "nil manager"},
		{name: "no roots", mgr: kb.NewManager(nil, filepath.Join(dir, "index.json")), want: "no roots"},
		{name: "blank index", mgr: kb.NewManager([]string{dir}, " "), want: "empty index path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKBWithManager(tt.mgr)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want text %q", err, tt.want)
			}
		})
	}

	mgr := kb.NewManager([]string{dir}, filepath.Join(dir, "idx", "index.json"))
	mem, err := NewKBWithManager(mgr)
	if err != nil {
		t.Fatalf("NewKBWithManager: %v", err)
	}
	if _, err := mem.Retain(context.Background(), Item{Content: "manager wired aqueduct"}); err != nil {
		t.Fatalf("Retain: %v", err)
	}
	res, err := mem.Recall(context.Background(), Query{Text: "aqueduct"})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if res.Total != 1 || res.Hits[0].Path != "memory/default/"+res.Hits[0].Item.ID+".md" {
		t.Fatalf("recall = %+v", res)
	}
}

func TestKBRetainPersistsMarkdown(t *testing.T) {
	mem, root := newTestKBRoot(t)
	stamp := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	stored, err := mem.Retain(context.Background(), Item{
		ID:         "note1",
		Title:      "Inspection",
		Content:    "Remember the aqueduct inspection.",
		Context:    "site visit: north arch",
		Tags:       []string{"civil", "memory", "bank:nope"},
		Bank:       "Field",
		DocumentID: "doc-9",
		Metadata:   map[string]string{"zone": "north", "source": "crew"},
		Timestamp:  stamp,
	})
	if err != nil {
		t.Fatalf("Retain: %v", err)
	}

	path := filepath.Join(root, "memory", "field", "note1.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memory file: %v", err)
	}
	if !strings.HasPrefix(string(data), "---\n") {
		t.Fatalf("file = %s", data)
	}
	doc, err := kb.ParseDocument(stored.ID, "memory/field/note1.md", string(data))
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if doc.Metadata.Author != memoryAuthor || doc.Metadata.Version != memoryVersion {
		t.Fatalf("metadata = %+v", doc.Metadata)
	}
	if doc.Metadata.Description != "site visit: north arch" || doc.Content != stored.Content {
		t.Fatalf("doc = %+v", doc)
	}
	if strings.Join(doc.Metadata.Tags, ",") != "memory,bank:field,civil" {
		t.Fatalf("tags = %#v", doc.Metadata.Tags)
	}
	if strings.Join(doc.Metadata.Related, ",") != "doc:doc-9,meta:source=crew,meta:zone=north" {
		t.Fatalf("related = %#v", doc.Metadata.Related)
	}

	res, err := mem.Recall(context.Background(), Query{Text: "aqueduct", Bank: "field"})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if res.Total != 1 || res.Hits[0].Path != "memory/field/note1.md" {
		t.Fatalf("recall = %+v", res)
	}
}

func TestKBRecallIncludesExistingDocuments(t *testing.T) {
	root := t.TempDir()
	guide := filepath.Join(root, "guides")
	notes := filepath.Join(root, "notes")
	if err := os.MkdirAll(guide, 0o755); err != nil {
		t.Fatalf("mkdir guide: %v", err)
	}
	if err := os.MkdirAll(notes, 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}
	guideBody := `---
title: "Field notes"
description: "Notes from the ridge"
tags: ["maps"]
---
The cartography of the ridge is incomplete.
`
	if err := os.WriteFile(filepath.Join(guide, "maps.md"), []byte(guideBody), 0o644); err != nil {
		t.Fatalf("write guide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(notes, "plain.md"), []byte("A plain memorandum about ballast."), 0o644); err != nil {
		t.Fatalf("write plain: %v", err)
	}

	mem, err := NewKB([]string{root}, filepath.Join(t.TempDir(), "index.json"))
	if err != nil {
		t.Fatalf("NewKB: %v", err)
	}
	if _, err := mem.Retain(context.Background(), Item{Content: "The souffle collapsed in the oven.", Bank: "kitchen"}); err != nil {
		t.Fatalf("Retain: %v", err)
	}

	guideHits, err := mem.Recall(context.Background(), Query{Text: "cartography"})
	if err != nil {
		t.Fatalf("Recall guide: %v", err)
	}
	if guideHits.Total != 1 || guideHits.Hits[0].Item.Title != "Field notes" || guideHits.Hits[0].Path != "guides/maps.md" {
		t.Fatalf("guide = %+v", guideHits)
	}
	if guideHits.Hits[0].Item.Bank != "" || guideHits.Hits[0].Item.Context != "Notes from the ridge" {
		t.Fatalf("guide item = %+v", guideHits.Hits[0].Item)
	}
	if !strings.Contains(guideHits.Hits[0].Item.Content, "cartography") {
		t.Fatalf("guide content = %q", guideHits.Hits[0].Item.Content)
	}

	filtered, err := mem.Recall(context.Background(), Query{Text: "cartography", Bank: "kitchen"})
	if err != nil {
		t.Fatalf("Recall filtered: %v", err)
	}
	if filtered.Total != 0 {
		t.Fatalf("filtered = %+v", filtered)
	}

	cooked, err := mem.Recall(context.Background(), Query{Text: "souffle", Bank: "kitchen"})
	if err != nil {
		t.Fatalf("Recall souffle: %v", err)
	}
	if cooked.Total != 1 || cooked.Hits[0].Item.Bank != "kitchen" {
		t.Fatalf("cooked = %+v", cooked)
	}

	plain, err := mem.Recall(context.Background(), Query{Text: "ballast"})
	if err != nil {
		t.Fatalf("Recall plain: %v", err)
	}
	if plain.Total != 1 || plain.Hits[0].Item.Title != "" || plain.Hits[0].Path != "notes/plain.md" {
		t.Fatalf("plain = %+v", plain)
	}
	if !strings.Contains(plain.Hits[0].Item.Content, "ballast") {
		t.Fatalf("plain content = %q", plain.Hits[0].Item.Content)
	}
}

func TestItemFromDocumentAndRelated(t *testing.T) {
	id, meta := decodeRelated([]string{
		"doc:first",
		"doc:second",
		"meta:broken",
		"meta:=novalue",
		"note",
		"meta:source=crew",
	})
	if id != "first" {
		t.Fatalf("doc id = %q", id)
	}
	if len(meta) != 1 || meta["source"] != "crew" {
		t.Fatalf("meta = %#v", meta)
	}

	generic := itemFromDocument(&kb.Document{
		ID:      "maps",
		Content: "body",
		Metadata: kb.Metadata{
			Title:       "Maps",
			Description: "ridge",
			Tags:        []string{"maps"},
			Author:      "someone",
			Related:     []string{"meta:source=nope", "doc:ignored"},
		},
	}, kb.SearchResult{Excerpt: "not used"})
	if generic.Content != "body" || generic.DocumentID != "" || generic.Metadata != nil {
		t.Fatalf("generic = %+v", generic)
	}
	if len(generic.Tags) != 1 || generic.Tags[0] != "maps" || generic.Context != "ridge" {
		t.Fatalf("generic fields = %+v", generic)
	}

	fallback := itemFromDocument(nil, kb.SearchResult{
		ID:          "missing",
		Title:       "Missing",
		Description: "only description",
		Tags:        []string{"Bank:Field", "memory", "Civil"},
	})
	if fallback.Content != "only description" || fallback.Bank != "field" || fallback.Title != "Missing" {
		t.Fatalf("fallback = %+v", fallback)
	}
	if len(fallback.Tags) != 1 || fallback.Tags[0] != "Civil" {
		t.Fatalf("fallback tags = %#v", fallback.Tags)
	}

	excerpt := itemFromDocument(nil, kb.SearchResult{Excerpt: "excerpt text", Description: "desc"})
	if excerpt.Content != "excerpt text" || excerpt.Context != "desc" {
		t.Fatalf("excerpt = %+v", excerpt)
	}
}

func TestKBStoreFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("retain errors", func(t *testing.T) {
		stub := &stubStore{}
		mem := newKB(stub)
		if _, err := mem.Retain(ctx, Item{Content: " "}); !errors.Is(err, ErrEmptyContent) {
			t.Fatalf("empty = %v", err)
		}
		if len(stub.added) != 0 {
			t.Fatalf("added = %d", len(stub.added))
		}

		stub.addErr = errors.New("add failed")
		if _, err := mem.Retain(ctx, Item{Content: "hello memory"}); !errors.Is(err, stub.addErr) || stub.rebuilds != 0 {
			t.Fatalf("add error = %v rebuilds = %d", err, stub.rebuilds)
		}

		stub.addErr = nil
		stub.rebuildErr = errors.New("rebuild failed")
		if _, err := mem.Retain(ctx, Item{ID: "note1", Content: "hello memory", Metadata: map[string]string{"b": "2", "a": "1"}}); !errors.Is(err, stub.rebuildErr) {
			t.Fatalf("rebuild error = %v", err)
		}
		if stub.rebuilds != 1 || len(stub.added) != 2 {
			t.Fatalf("added = %d rebuilds = %d", len(stub.added), stub.rebuilds)
		}
		doc := stub.added[1]
		if doc.Path != "memory/default/note1.md" || doc.Metadata.Author != memoryAuthor {
			t.Fatalf("doc = %+v", doc)
		}
		if strings.Join(doc.Metadata.Related, ",") != "meta:a=1,meta:b=2" {
			t.Fatalf("related = %#v", doc.Metadata.Related)
		}
	})

	t.Run("recall errors and fallback", func(t *testing.T) {
		canceled, cancel := context.WithCancel(context.Background())
		stub := &stubStore{onSearch: cancel, results: &kb.SearchResults{}}
		mem := newKB(stub)
		if _, err := mem.Recall(canceled, Query{Text: "hello"}); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel = %v", err)
		}
		if stub.lists != 0 {
			t.Fatalf("lists = %d", stub.lists)
		}

		stub = &stubStore{searchErr: errors.New("search down")}
		mem = newKB(stub)
		if _, err := mem.Recall(ctx, Query{Text: "hello"}); !errors.Is(err, stub.searchErr) {
			t.Fatalf("search = %v", err)
		}
		if _, err := mem.Reflect(ctx, Query{Text: "hello"}); !errors.Is(err, stub.searchErr) {
			t.Fatalf("reflect search = %v", err)
		}

		stub = &stubStore{listErr: errors.New("list down"), results: &kb.SearchResults{}}
		mem = newKB(stub)
		if _, err := mem.Recall(ctx, Query{Text: "hello"}); !errors.Is(err, stub.listErr) {
			t.Fatalf("list = %v", err)
		}

		stub = &stubStore{}
		mem = newKB(stub)
		empty, err := mem.Recall(ctx, Query{Text: "hello"})
		if err != nil || empty.Total != 0 {
			t.Fatalf("nil results = %+v err = %v", empty, err)
		}

		stub = &stubStore{
			results: &kb.SearchResults{Results: []kb.SearchResult{{
				ID:          "missing",
				Title:       "Missing",
				Description: "desc",
				Path:        "memory/default/missing.md",
				Excerpt:     "excerpt text",
				Tags:        []string{"memory", "bank:default", "civil"},
				Score:       0.4,
			}}},
			docs: []*kb.Document{nil},
		}
		mem = newKB(stub)
		fallback, err := mem.Recall(ctx, Query{Text: "excerpt", Tags: []string{"civil"}})
		if err != nil {
			t.Fatalf("fallback: %v", err)
		}
		if fallback.Total != 1 || fallback.Hits[0].Item.Content != "excerpt text" || fallback.Hits[0].Path != "memory/default/missing.md" {
			t.Fatalf("fallback = %+v", fallback)
		}
		if fallback.Hits[0].Item.Bank != "default" {
			t.Fatalf("bank = %q", fallback.Hits[0].Item.Bank)
		}

		full := &kb.Document{
			ID:      "note",
			Path:    "memory/default/note.md",
			Content: "full aqueduct body",
			Metadata: kb.Metadata{
				Title:   "Note",
				Author:  memoryAuthor,
				Tags:    []string{"memory", "bank:default"},
				Created: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}
		stub = &stubStore{
			results: &kb.SearchResults{Results: []kb.SearchResult{
				{ID: "note", Path: "memory/default/note.md", Excerpt: "short", Score: 0.8, Tags: []string{"memory", "bank:default"}},
				{ID: "other", Path: "guides/other.md", Excerpt: "other aqueduct", Score: 0.2, Tags: []string{"maps"}},
			}},
			docs: []*kb.Document{nil, full},
		}
		mem = newKB(stub)
		got, err := mem.Recall(ctx, Query{Text: "aqueduct", Bank: "default", Limit: 5})
		if err != nil {
			t.Fatalf("full: %v", err)
		}
		if got.Total != 1 || got.Hits[0].Item.Content != "full aqueduct body" {
			t.Fatalf("full recall = %+v", got)
		}
	})
}

type stubStore struct {
	searchErr  error
	listErr    error
	addErr     error
	rebuildErr error
	results    *kb.SearchResults
	docs       []*kb.Document
	added      []*kb.Document
	rebuilds   int
	lists      int
	onSearch   func()
}

func (s *stubStore) Search(string) (*kb.SearchResults, error) {
	if s.onSearch != nil {
		s.onSearch()
	}
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	return s.results, nil
}

func (s *stubStore) ListDocuments() ([]*kb.Document, error) {
	s.lists++
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.docs, nil
}

func (s *stubStore) AddDocument(doc *kb.Document, rootIndex int) error {
	if rootIndex != 0 {
		return errors.New("unexpected root")
	}
	s.added = append(s.added, doc)
	return s.addErr
}

func (s *stubStore) RebuildIndex() error {
	s.rebuilds++
	return s.rebuildErr
}
