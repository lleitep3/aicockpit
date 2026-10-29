package personalagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCopilotQuota(t *testing.T) {
	for _, tt := range []struct {
		name, payload, status string
		windows               int
		balance               bool
	}{
		{"available", `{"quota_snapshots":{"chat":{"percent_remaining":80,"remaining":160,"entitlement":200,"quota_remaining":160}}}`, "available", 1, true},
		{"exhausted", `{"quota_snapshots":{"chat":{"percent_remaining":0,"remaining":-1,"entitlement":200,"quota_remaining":-0.1}}}`, "exhausted", 1, true},
		{"unlimited", `{"quota_snapshots":{"chat":{"unlimited":true}}}`, "available", 1, false},
		{"overage permitted", `{"quota_snapshots":{"chat":{"remaining":0,"entitlement":200,"overage_permitted":true}}}`, "available", 1, true},
		{"missing chat", `{"quota_snapshots":{"completions":{"percent_remaining":100}}}`, "unknown", 0, false},
		{"percent only", `{"quota_snapshots":{"chat":{"percent_remaining":42}}}`, "available", 1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parseCopilotQuota(json.RawMessage(tt.payload))
			if err != nil || q.Status != tt.status || len(q.Windows) != tt.windows {
				t.Fatalf("quota=%+v err=%v", q, err)
			}
			if tt.balance != (q.Balance != nil) {
				t.Fatalf("balance mismatch: %+v", q)
			}
		})
	}
	if _, err := parseCopilotQuota(json.RawMessage(`{`)); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	q, err := parseCopilotQuota(json.RawMessage(`{"quota_reset_date":"2026-11-01","quota_snapshots":{"chat":{"remaining":10,"entitlement":200}}}`))
	if err != nil || q.Windows[0].ResetsAt == nil || q.Windows[0].ResetsAt.Format("2006-01-02") != "2026-11-01" {
		t.Fatalf("reset date lost: %+v err=%v", q, err)
	}
}

func TestParseDevinPlanStatus(t *testing.T) {
	for _, tt := range []struct {
		name, payload, status string
		balance               bool
	}{
		{"weekly remaining", `{"planStatus":{"weeklyQuotaRemainingPercent":50,"weeklyQuotaResetAtUnix":"1793520000","overageBalanceMicros":"9590188"}}`, "available", true},
		{"exhausted no overage", `{"planStatus":{"weeklyQuotaRemainingPercent":0,"weeklyQuotaResetAtUnix":"1793520000","availablePromptCredits":0,"overageBalanceMicros":"0"}}`, "exhausted", false},
		{"exhausted but overage", `{"planStatus":{"weeklyQuotaRemainingPercent":0,"overageBalanceMicros":"5000000"}}`, "available", true},
		{"unlimited credits", `{"planStatus":{"weeklyQuotaRemainingPercent":0,"availablePromptCredits":-1}}`, "available", false},
		{"missing fields", `{}`, "unknown", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parseDevinPlanStatus(json.RawMessage(tt.payload))
			if err != nil || q.Status != tt.status {
				t.Fatalf("quota=%+v err=%v", q, err)
			}
			if tt.balance != (q.Balance != nil) {
				t.Fatalf("balance mismatch: %+v", q)
			}
		})
	}
	if _, err := parseDevinPlanStatus(json.RawMessage(`{`)); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}

func TestParseAntigravityUsage(t *testing.T) {
	sample := "Gemini Models\tWeekly Limit Remaining\t97%\t2026-11-03T17:30:06Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t100%\t2026-10-28T04:47:00Z\n" +
		"Claude and GPT models\tWeekly Limit Remaining\t78%\t2026-10-29T18:23:59Z\n"
	q := parseAntigravityUsage(sample)
	if q.Status != "available" || len(q.Windows) != 3 {
		t.Fatalf("quota=%+v", q)
	}
	if q.Windows[0].ResetsAt == nil || q.Windows[0].Name != "Gemini Models Weekly Limit Remaining" {
		t.Fatalf("window=%+v", q.Windows[0])
	}
	if q := parseAntigravityUsage("A\tB\t0%\t2026-11-03T17:30:06Z\nC\tD\t0%\t2026-10-28T04:47:00Z\n"); q.Status != "unknown" {
		t.Fatalf("status=%s", q.Status)
	}
	if q := parseAntigravityUsage("garbage\n"); q.Status != "unknown" || len(q.Windows) != 0 {
		t.Fatalf("quota=%+v", q)
	}
}

func TestCopilotTokenSources(t *testing.T) {
	for _, env := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(env, "tok-"+env)
		if token, err := copilotToken(context.Background()); err != nil || token != "tok-"+env {
			t.Fatalf("%s failed: %v", env, err)
		}
		t.Setenv(env, "")
	}
	dir := t.TempDir()
	gh := filepath.Join(dir, "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\nif [ \"$1\" = auth ]; then echo gh-token; exit 0; fi\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if token, err := copilotToken(context.Background()); err != nil || token != "gh-token" {
		t.Fatalf("gh token failed: %v", err)
	}
	if err := os.WriteFile(gh, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := copilotToken(context.Background()); err == nil {
		t.Fatal("failing gh accepted")
	}
	if err := os.Remove(gh); err != nil {
		t.Fatal(err)
	}
	if _, err := copilotToken(context.Background()); err == nil {
		t.Fatal("missing gh accepted")
	}
}

func TestCopilotQuotaHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok-env" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"quota_snapshots":{"chat":{"remaining":5,"entitlement":200}}}`)
	}))
	defer server.Close()
	t.Setenv("GH_TOKEN", "tok-env")
	old := copilotQuotaURL
	copilotQuotaURL = server.URL
	defer func() { copilotQuotaURL = old }()
	q, err := NewCopilotAdapter("unused").Quota(context.Background())
	if err != nil || q.Status != "available" || q.Balance == nil {
		t.Fatalf("quota=%+v err=%v", q, err)
	}
}

func TestCopilotQuotaErrors(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := NewCopilotAdapter("unused").Quota(context.Background()); err == nil {
		t.Fatal("quota without token accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Setenv("GH_TOKEN", "tok")
	old := copilotQuotaURL
	copilotQuotaURL = server.URL
	defer func() { copilotQuotaURL = old }()
	if _, err := NewCopilotAdapter("unused").Quota(context.Background()); err == nil {
		t.Fatal("HTTP 500 accepted")
	}
	q, err := parseCopilotQuota(json.RawMessage(`{"quota_snapshots":{"chat":{"remaining":12.5,"entitlement":200}}}`))
	if err != nil || q.Balance == nil || *q.Balance != "12.50/200 créditos" {
		t.Fatalf("fractional balance lost: %+v", q)
	}
}

func TestDevinQuotaHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "dev-tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"planStatus":{"weeklyQuotaRemainingPercent":50,"weeklyQuotaResetAtUnix":"1793520000"}}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	creds := filepath.Join(dir, "credentials.toml")
	if err := os.WriteFile(creds, []byte("windsurf_api_key = \"dev-tok\"\napi_server_url = \""+server.URL+"\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	old := devinCredentialsPath
	devinCredentialsPath = func() string { return creds }
	defer func() { devinCredentialsPath = old }()
	q, err := NewDevinAdapter("unused").Quota(context.Background())
	if err != nil || q.Status != "available" || len(q.Windows) != 1 || q.Windows[0].RemainingPercent != 50 {
		t.Fatalf("quota=%+v err=%v", q, err)
	}
}

func TestDevinQuotaErrors(t *testing.T) {
	old := devinCredentialsPath
	defer func() { devinCredentialsPath = old }()
	devinCredentialsPath = func() string { return filepath.Join(t.TempDir(), "absent.toml") }
	if _, err := NewDevinAdapter("unused").Quota(context.Background()); err == nil {
		t.Fatal("missing credentials accepted")
	}
	creds := filepath.Join(t.TempDir(), "credentials.toml")
	if err := os.WriteFile(creds, []byte("windsurf_api_key = \"x\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	devinCredentialsPath = func() string { return creds }
	if _, err := NewDevinAdapter("unused").Quota(context.Background()); err == nil {
		t.Fatal("incomplete credentials accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	if err := os.WriteFile(creds, []byte("windsurf_api_key = \"x\"\napi_server_url = \""+server.URL+"\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDevinAdapter("unused").Quota(context.Background()); err == nil {
		t.Fatal("HTTP error accepted")
	}
}

func TestAntigravityQuotaCommand(t *testing.T) {
	wrapper := filepath.Join(t.TempDir(), "agy")
	script := "#!/bin/sh\nif [ \"$1\" = \"--print\" ]; then printf 'G\\tW\\t80%%\\t2026-11-03T17:30:06Z\\n'; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	q, err := NewAntigravityAdapter(wrapper).Quota(context.Background())
	if err != nil || q.Status != "available" || len(q.Windows) != 1 {
		t.Fatalf("quota=%+v err=%v", q, err)
	}
	if _, err := NewAntigravityAdapter(filepath.Join(t.TempDir(), "absent")).Quota(context.Background()); err == nil {
		t.Fatal("missing binary accepted")
	}
}

func TestLiveProviderQuotas(t *testing.T) {
	if os.Getenv("COCKPIT_TEST_LIVE_QUOTA") != "1" {
		t.Skip("explicit live opt-in required")
	}
	r := NewProviderRegistry()
	for _, id := range []string{"copilot", "devin", "antigravity"} {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		q := r.Quota(ctx, id)
		cancel()
		t.Logf("%s: status=%s windows=%d source=%s", id, q.Status, len(q.Windows), q.Source)
		for _, w := range q.Windows {
			t.Logf("  %s: %.0f%% resets=%v", w.Name, w.RemainingPercent, w.ResetsAt)
		}
	}
}
