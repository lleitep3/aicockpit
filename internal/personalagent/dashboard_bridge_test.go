package personalagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDashboardBridgeStartsAndStopsLoopbackBackend(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fake uvicorn launcher requires a POSIX shell")
	}
	t.Setenv("COCKPIT_DASHBOARD_HELPER", "")

	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	backend := filepath.Join(workspace, "app", "backend")
	uvicorn := dashboardUvicornPath(workspace)
	if err := os.MkdirAll(filepath.Join(backend, "app"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backend, "app", "main.py"), []byte("app = None\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf(
		"#!/bin/sh\nCOCKPIT_DASHBOARD_HELPER=1 exec %q -test.run=TestDashboardBridgeSubprocess -- \"$@\"\n",
		binary,
	)
	if err := os.MkdirAll(filepath.Dir(uvicorn), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uvicorn, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	bridge := NewDashboardBridge(workspace)
	ctx, cancel := context.WithTimeout(context.Background(), dashboardStartupTimeout+5*time.Second)
	defer cancel()
	overview, err := bridge.Overview(ctx)
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if !strings.Contains(string(overview.KPI), `"projects_active":7`) {
		t.Fatalf("KPI response = %s, want local test data", overview.KPI)
	}

	bridge.mu.Lock()
	address := bridge.baseURL
	bridge.mu.Unlock()
	if !strings.HasPrefix(address, "http://127.0.0.1:") {
		t.Fatalf("Dashboard API address = %q, want loopback", address)
	}
	if err := bridge.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := http.Get(address + "/health"); err == nil {
		t.Fatal("Dashboard API remained available after bridge shutdown")
	}
}

func TestDashboardBridgeInstalledWorkspace(t *testing.T) {
	workspace := os.Getenv("COCKPIT_TEST_DASHBOARD_WORKSPACE")
	if workspace == "" {
		t.Skip("explicit installed Dashboard workspace opt-in required")
	}
	bridge := NewDashboardBridge(workspace)
	ctx, cancel := context.WithTimeout(context.Background(), dashboardStartupTimeout+10*time.Second)
	defer cancel()
	defer func() {
		if err := bridge.Close(ctx); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	overview, err := bridge.Overview(ctx)
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if !json.Valid(overview.KPI) {
		t.Fatalf("Dashboard KPI is invalid JSON")
	}
}

func TestDashboardBridgeSubprocess(t *testing.T) {
	if os.Getenv("COCKPIT_DASHBOARD_HELPER") != "1" {
		return
	}
	port := ""
	for index, arg := range os.Args {
		if arg == "--port" && index+1 < len(os.Args) {
			port = os.Args[index+1]
			break
		}
	}
	if _, err := strconv.Atoi(port); err != nil {
		os.Exit(2)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v1/overview/kpi", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"projects_active":7}`))
	})
	if err := http.ListenAndServe("127.0.0.1:"+port, mux); err != nil {
		os.Exit(1)
	}
}
