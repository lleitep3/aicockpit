package personalagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type dashboardOverviewFixture struct {
	overview DashboardOverview
	err      error
}

func (f dashboardOverviewFixture) Overview(context.Context) (DashboardOverview, error) {
	return f.overview, f.err
}

func TestDashboardBridgeReadsOnlyOverviewKPIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/overview/kpi" {
			t.Fatalf("Dashboard API request = %s %s, want GET KPI endpoint", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"projects_active":5,"packages_total":24}`))
	}))
	defer server.Close()

	bridge := &DashboardBridge{baseURL: server.URL, client: server.Client()}
	overview, err := bridge.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	var kpi map[string]int
	if err := json.Unmarshal(overview.KPI, &kpi); err != nil {
		t.Fatalf("decode KPI response: %v", err)
	}
	if kpi["projects_active"] != 5 || kpi["packages_total"] != 24 {
		t.Fatalf("KPI = %#v, want project and package counts", kpi)
	}
}

func TestDashboardBridgeRequiresPreparedWorkspace(t *testing.T) {
	bridge := NewDashboardBridge(filepath.Join(t.TempDir(), "missing-dashboard"))
	if _, err := bridge.Overview(context.Background()); err == nil || !strings.Contains(err.Error(), "backend is not installed") {
		t.Fatalf("Overview() error = %v, want missing backend error", err)
	}
}

func TestDashboardOverviewHTTPUsesPersonalAgentSessionToken(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	overview := DashboardOverview{KPI: json.RawMessage(`{"projects_active":3}`)}
	handler := NewHTTPHandlerWithDashboard(service, "token", host, dashboardOverviewFixture{overview: overview})

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/dashboard/overview", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized dashboard request status = %d, want 401", unauthorized.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/dashboard/overview", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"projects_active":3`) {
		t.Fatalf("authorized dashboard response = %d %s, want overview KPI", response.Code, response.Body.String())
	}
}

func TestDashboardOverviewHTTPReportsUnavailableBridge(t *testing.T) {
	service, _ := newRoutingTestService(t)
	const host = "127.0.0.1:4321"
	handler := NewHTTPHandler(service, "token", host)
	request := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/dashboard/overview", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("dashboard response status = %d, want 503", response.Code)
	}
}

func TestDashboardBridgeCloseWithoutProcess(t *testing.T) {
	bridge := NewDashboardBridge(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bridge.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
