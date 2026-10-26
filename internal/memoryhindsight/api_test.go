package memoryhindsight

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/memory"
)

func TestAPIRoundTrip(t *testing.T) {
	stamp := time.Date(2026, 4, 2, 8, 0, 0, 0, time.UTC)
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, r.URL.Path+" "+string(body))
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/memories/recall"):
			_, _ = w.Write([]byte(`{"results":[{"id":"m1","text":"Alice works at Google","context":"career","document_id":"doc-1","tags":["team"],"metadata":{"source":"kb"},"scores":{"final":0.75}},{"id":"empty","text":"  "}]}`))
		case strings.HasSuffix(r.URL.Path, "/reflect"):
			_, _ = w.Write([]byte(`{"text":" Alice works at Google. ","based_on":{"memories":[{"id":"m1","text":"Alice works at Google","context":"career","tags":["team"],"metadata":{"source":"kb"}},{"text":" "}]}}`))
		case strings.HasSuffix(r.URL.Path, "/memories"):
			_, _ = w.Write([]byte(`{"success":true,"bank_id":"project-demo","items_count":1,"async":false}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	api, err := NewAPI(srv.URL+"/", "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = api.Retain(ctx, "project-demo", []memory.Item{{
		Content:    "Alice works at Google",
		Context:    "career",
		DocumentID: "doc-1",
		Tags:       []string{"team"},
		Metadata:   map[string]string{"source": "kb"},
		Timestamp:  stamp,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := api.Retain(ctx, "project-demo", nil); err != nil {
		t.Fatal(err)
	}
	hits, err := api.Recall(ctx, "project-demo", memory.Query{Text: "where does Alice work", Tags: []string{"team"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Item.ID != "m1" || hits[0].Score != 0.75 || hits[0].Item.DocumentID != "doc-1" {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Item.Metadata["source"] != "kb" || hits[0].Item.Context != "career" {
		t.Fatalf("hit item = %+v", hits[0].Item)
	}
	text, facts, err := api.Reflect(ctx, "project-demo", memory.Query{Text: "where does Alice work", Tags: []string{"team"}})
	if err != nil {
		t.Fatal(err)
	}
	if text != "Alice works at Google." || len(facts) != 1 || facts[0].Item.ID != "m1" {
		t.Fatalf("reflect = %q %+v", text, facts)
	}
	joined := strings.Join(requests, "\n")
	if !strings.Contains(joined, "/v1/default/banks/project-demo/memories") {
		t.Fatalf("paths = %s", joined)
	}
	if !strings.Contains(joined, `"all_strict"`) || !strings.Contains(joined, "Alice works at Google") {
		t.Fatalf("body = %s", joined)
	}
	if !strings.Contains(joined, "2026-04-02") {
		t.Fatalf("timestamp missing: %s", joined)
	}
}

func TestAPIErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		recall  bool
		retain  bool
		reflect bool
		wantErr bool
		empty   bool
	}{
		{name: "recall missing bank", status: http.StatusNotFound, body: `{"detail":"missing"}`, recall: true, empty: true},
		{name: "reflect missing bank", status: http.StatusNotFound, body: `{"detail":"missing"}`, reflect: true, empty: true},
		{name: "recall failure", status: http.StatusInternalServerError, body: `{"detail":"down"}`, recall: true, wantErr: true},
		{name: "retain failure", status: http.StatusBadGateway, body: `{"detail":"down"}`, retain: true, wantErr: true},
		{name: "retain rejected", status: http.StatusOK, body: `{"success":false,"bank_id":"b","items_count":0,"async":false}`, retain: true, wantErr: true},
		{name: "reflect plain", status: http.StatusOK, body: `{"text":"ok"}`, reflect: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			api, err := NewAPI(srv.URL, "")
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			switch {
			case tt.retain:
				err = api.Retain(ctx, "project-demo", []memory.Item{{Content: "fact"}})
			case tt.recall:
				var hits []memory.Hit
				hits, err = api.Recall(ctx, "project-demo", memory.Query{Text: "fact"})
				if tt.empty && (err != nil || hits != nil) {
					t.Fatalf("hits=%+v err=%v", hits, err)
				}
			case tt.reflect:
				var text string
				text, _, err = api.Reflect(ctx, "project-demo", memory.Query{Text: "fact"})
				if !tt.wantErr && text != "ok" && !tt.empty {
					t.Fatalf("text = %q err = %v", text, err)
				}
				if tt.empty && err != nil {
					t.Fatalf("missing bank err = %v", err)
				}
			}
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNewAPIRejectsBadInput(t *testing.T) {
	if _, err := NewAPI("http://user:secret@127.0.0.1:8888", ""); err == nil {
		t.Fatal("userinfo should be rejected")
	}
	if _, err := NewAPI("http://127.0.0.1:9", "bad\ntoken"); err == nil {
		t.Fatal("newline token should be rejected")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	api, err := NewAPI(srv.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	srv.Close()
	if _, err := api.Recall(context.Background(), "project-demo", memory.Query{Text: "x"}); err == nil {
		t.Fatal("closed server should fail")
	}
	var nilAPI *API
	if err := nilAPI.Retain(context.Background(), "b", []memory.Item{{Content: "x"}}); err == nil {
		t.Fatal("nil client should fail")
	}
	if _, err := api.Recall(nil, "b", memory.Query{Text: "x"}); !errors.Is(err, memory.ErrNilContext) { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatalf("nil ctx = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := api.Reflect(ctx, "b", memory.Query{Text: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v", err)
	}
}

func TestTitleFrom(t *testing.T) {
	long := strings.Repeat("a", 90)
	if got := titleFrom("  # " + long); !strings.HasSuffix(got, "...") || len([]rune(got)) != 80 {
		t.Fatalf("title = %q (%d)", got, len([]rune(got)))
	}
	if titleFrom("###") != "memory" {
		t.Fatal("hash-only title")
	}
	if titleFrom("Short title") != "Short title" {
		t.Fatal("short title")
	}
}
