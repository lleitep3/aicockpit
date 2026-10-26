package hindsight

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPProber(t *testing.T) {
	secretBody := "secret-in-body"
	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{name: "ok", status: http.StatusOK},
		{name: "no content", status: http.StatusNoContent},
		{name: "unavailable", status: http.StatusServiceUnavailable, wantErr: "HTTP 503"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != HealthPath {
					t.Errorf("path = %s", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(secretBody))
			}))
			t.Cleanup(server.Close)
			err := (HTTPProber{Client: server.Client()}).Probe(context.Background(), server.URL+HealthPath)
			if tt.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr) || strings.Contains(err.Error(), secretBody)) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	if err := (HTTPProber{}).Probe(nilCtx(), "http://127.0.0.1"); !errors.Is(err, ErrNilContext) {
		t.Fatalf("nil ctx = %v", err)
	}
	if err := (HTTPProber{}).Probe(context.Background(), "://bad"); err == nil {
		t.Fatal("bad url succeeded")
	}
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed.Close()
	if err := (HTTPProber{Client: closed.Client()}).Probe(context.Background(), closed.URL+"/health"); err == nil {
		t.Fatal("closed server succeeded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)
	if err := (HTTPProber{Client: server.Client()}).Probe(ctx, server.URL+"/health"); err == nil {
		t.Fatal("canceled probe succeeded")
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(slow.Close)
	client := &http.Client{Timeout: time.Millisecond}
	if err := (HTTPProber{Client: client}).Probe(context.Background(), slow.URL+"/health"); err == nil {
		t.Fatal("timeout succeeded")
	}
	if err := (HTTPProber{}).Probe(context.Background(), "http://127.0.0.1:1/health"); err == nil {
		t.Fatal("default client reached a closed port")
	}
}
