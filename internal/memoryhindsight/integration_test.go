package memoryhindsight

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/memory"
)

func TestIntegrationSearchAndIndex(t *testing.T) {
	fake := &Fake{
		RecallHits: map[string][]memory.Hit{
			"project-demo": {{Item: memory.Item{ID: "m1", Title: "Guide", Content: strings.Repeat("aqueduct ", 80)}, Score: 0.8}},
		},
	}
	backend, err := New(fake, testBanks())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	body := "---\ntitle: Guide\ndescription: Water\ntags: [crew]\n---\nThe aqueduct is open.\n"
	if err := os.MkdirAll(filepath.Join(root, "guides"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "guides", "open.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "empty.md"), []byte("   \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	integration := &Integration{mem: backend, banks: testBanks(), source: fileSource{}, enabled: true}
	ctx := context.Background()
	if err := integration.Index(ctx, []string{root}, true); err != nil {
		t.Fatal(err)
	}
	calls := fake.Snapshot()
	if len(calls) != 1 || calls[0].Op != "retain" || calls[0].Bank != "project-demo" {
		t.Fatalf("calls = %+v", calls)
	}
	item := calls[0].Items[0]
	if !strings.Contains(item.Content, "aqueduct") || !strings.Contains(item.Content, "# Guide") {
		t.Fatalf("content = %q", item.Content)
	}
	if item.DocumentID == "" || item.Metadata["source"] != "kb" || item.Context != "knowledge base" {
		t.Fatalf("item = %+v", item)
	}

	text, err := integration.Search(ctx, "aqueduct", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Hindsight recall") || !strings.Contains(text, "project-demo") || !strings.HasSuffix(strings.TrimSpace(text), "...") {
		t.Fatalf("search = %q", text)
	}
	if strings.Count(text, "aqueduct") > 40 {
		t.Fatalf("snippet was not truncated: %s", text)
	}
}

func TestIntegrationDegrades(t *testing.T) {
	var disabled *Integration
	if text, err := disabled.Search(context.Background(), "q", 1); text != "" || err != nil {
		t.Fatalf("nil search = %q %v", text, err)
	}
	if err := disabled.Index(context.Background(), []string{"/tmp"}, false); err != nil {
		t.Fatal(err)
	}
	off, err := Open(hindsight.Settings{}, Identity{}, "")
	if err != nil || off.Enabled() {
		t.Fatalf("disabled open = %+v %v", off, err)
	}
	if _, err := Open(hindsight.Settings{Enabled: true, APIURL: "ftp://nope"}, Identity{Username: "ada"}, ""); err == nil {
		t.Fatal("bad url should fail")
	}
	if _, err := Open(hindsight.Settings{Enabled: true, Banks: hindsight.BanksConfig{Project: "project", User: "ada"}}, Identity{}, ""); !errors.Is(err, memory.ErrInvalidBank) {
		t.Fatalf("reserved bank = %v", err)
	}

	dir := t.TempDir()
	opened, err := Open(hindsight.Settings{Enabled: true}, Identity{WorkDir: dir, Username: "Ada"}, "")
	if err != nil || !opened.Enabled() {
		t.Fatalf("open = %+v %v", opened, err)
	}
	if opened.Banks().Project == "" || opened.Banks().User != "user-ada" {
		t.Fatalf("banks = %+v", opened.Banks())
	}

	failed := Failed(nil)
	if _, err := failed.Search(context.Background(), "q", 1); err == nil {
		t.Fatal("failed search should error")
	}
	if err := failed.Index(context.Background(), nil, false); err == nil {
		t.Fatal("failed index should error")
	}
	if FormatRecall(nil) != "" || FormatRecall(&memory.RecallResult{}) != "" {
		t.Fatal("empty recall should stay quiet")
	}
	if _, err := Open(hindsight.Settings{Enabled: true, APIPort: 80, UIPort: 80}, Identity{Username: "ada"}, ""); err == nil {
		t.Fatal("identical ports should fail before any request")
	}
	var missing *Integration
	if missing.Banks() != (Names{}) {
		t.Fatal("nil banks")
	}
	quiet := FormatRecall(&memory.RecallResult{Hits: []memory.Hit{{Item: memory.Item{Content: "fact"}, Score: 1}}})
	if !strings.Contains(quiet, "memory") {
		t.Fatalf("untitled hit = %q", quiet)
	}
}

func TestIntegrationOpenUsesAPI(t *testing.T) {
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/memories/recall") {
			_, _ = w.Write([]byte(`{"results":[{"id":"m1","text":"from server","scores":{"final":0.5}}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	opened, err := Open(hindsight.Settings{Enabled: true, APIURL: srv.URL, Banks: hindsight.BanksConfig{Project: "crew", User: "ada"}}, Identity{}, "tok")
	if err != nil {
		t.Fatal(err)
	}
	text, err := opened.Search(context.Background(), "server", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "from server") || !strings.Contains(text, "crew") {
		t.Fatalf("text = %q", text)
	}
	fail = true
	if _, err := opened.Search(context.Background(), "server", 3); err == nil {
		t.Fatal("unreachable hindsight should fail the recall")
	}
}

func TestIntegrationIndexErrors(t *testing.T) {
	fake := &Fake{RetainErr: map[string]error{"project-demo": errors.New("down")}}
	backend, err := New(fake, testBanks())
	if err != nil {
		t.Fatal(err)
	}
	integration := &Integration{
		mem:     backend,
		banks:   testBanks(),
		source:  stubDocs{docs: []*kb.Document{{Path: "a.md", Content: "hello", Metadata: kb.Metadata{Title: "A", Modified: time.Now()}}, nil, {Content: "  "}}},
		enabled: true,
	}
	if err := integration.Index(context.Background(), []string{"root"}, false); err == nil || !strings.Contains(err.Error(), "down") {
		t.Fatalf("index err = %v", err)
	}
	integration.mem = nil
	if err := integration.Index(context.Background(), []string{"root"}, false); err == nil {
		t.Fatal("nil memory should fail")
	}
	if _, err := integration.Search(context.Background(), "q", 1); err == nil {
		t.Fatal("nil memory search should fail")
	}
	integration.enabled = false
	if err := integration.Index(context.Background(), []string{"root"}, false); err != nil {
		t.Fatal(err)
	}
	missing := &Integration{mem: backend, banks: testBanks(), source: fileSource{}, enabled: true}
	if err := missing.Index(context.Background(), []string{filepath.Join(t.TempDir(), "missing")}, false); err == nil {
		t.Fatal("missing root should fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	integration.enabled = true
	integration.mem = backend
	if err := integration.Index(ctx, []string{"root"}, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v", err)
	}
	if err := integration.Index(nil, []string{"root"}, false); !errors.Is(err, memory.ErrNilContext) { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatalf("nil ctx = %v", err)
	}
}

type stubDocs struct {
	docs []*kb.Document
	err  error
}

func (s stubDocs) Documents([]string) ([]*kb.Document, error) {
	return s.docs, s.err
}

func TestDocumentToken(t *testing.T) {
	tests := []struct {
		doc  *kb.Document
		want string
	}{
		{doc: &kb.Document{Path: "guides/open.md"}, want: "guides-open-md"},
		{doc: &kb.Document{Path: "1-start.md"}, want: "doc-1-start-md"},
		{doc: &kb.Document{ID: "Alpha"}, want: "alpha"},
	}
	for _, tt := range tests {
		if got := documentToken(tt.doc); got != tt.want {
			t.Errorf("documentToken(%+v) = %q, want %q", tt.doc, got, tt.want)
		}
	}
	long := &kb.Document{Path: strings.Repeat("abc/", 40) + "end.md"}
	if got := documentToken(long); !strings.HasPrefix(got, "doc-") || len(got) > 40 {
		t.Fatalf("long token = %q", got)
	}
	if oneLine("a\nb  c", 10) != "a b c" {
		t.Fatalf("oneLine = %q", oneLine("a\nb  c", 10))
	}
	if oneLine("abcdef", 3) != "abc" {
		t.Fatalf("oneLine trim = %q", oneLine("abcdef", 3))
	}
}
