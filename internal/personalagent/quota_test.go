package personalagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseCodexQuota(t *testing.T) {
	for _, tt := range []struct {
		name, payload, status string
		windows               int
	}{
		{"available", `{"ordinaryUsageAllowed":true,"rateLimits":{"primary":{"usedPercent":25,"resetsAt":1800000000}}}`, "available", 1},
		{"denied", `{"ordinaryUsageAllowed":false,"rateLimits":{}}`, "exhausted", 0},
		{"empty", `{}`, "unknown", 0},
		{"full is not denial", `{"rateLimits":{"primary":{"usedPercent":100}}}`, "unknown", 1},
		{"spend control", `{"ordinaryUsageAllowed":true,"rateLimits":{"spendControlReached":true}}`, "exhausted", 0},
		{"model bucket", `{"ordinaryUsageAllowed":false,"rateLimits":{"limitId":"spark"}}`, "unknown", 0},
		{"shared bucket", `{"ordinaryUsageAllowed":true,"rateLimits":{"limitId":"spark"},"rateLimitsByLimitId":{"codex":{"primary":{"usedPercent":40}}}}`, "available", 1},
		{"invalid percent", `{"rateLimits":{"primary":{"usedPercent":101},"secondary":{"usedPercent":-1}}}`, "unknown", 0},
		{"missing percent", `{"rateLimits":{"primary":{}}}`, "unknown", 0},
		{"old reset not recovery", `{"ordinaryUsageAllowed":false,"rateLimits":{"primary":{"usedPercent":100,"resetsAt":1}}}`, "exhausted", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parseCodexQuota(json.RawMessage(tt.payload))
			if err != nil || q.Status != tt.status || len(q.Windows) != tt.windows {
				t.Fatalf("quota=%+v err=%v", q, err)
			}
		})
	}
	if _, err := parseCodexQuota(json.RawMessage(`{`)); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	q, err := parseCodexQuota(json.RawMessage(`{"rateLimits":{"primary":{"usedPercent":25,"resetsAt":1800000000},"credits":{"balance":"12.5","unlimited":true}}}`))
	if err != nil || q.Balance == nil || *q.Balance != "12.5" || !q.Unlimited || q.Windows[0].RemainingPercent != 75 || q.Windows[0].ResetsAt.Unix() != 1800000000 {
		t.Fatalf("quota=%+v err=%v", q, err)
	}
}

type quotaTestAdapter struct {
	calls  int
	status string
	err    error
}

func (a *quotaTestAdapter) Quota(context.Context) (ProviderQuota, error) {
	a.calls++
	return ProviderQuota{Status: a.status}, a.err
}
func (a *quotaTestAdapter) Catalog(context.Context) (Catalog, error) { return Catalog{}, nil }
func (a *quotaTestAdapter) RunTurn(context.Context, RunRequest, func(string), func(ApprovalRequest) (string, error)) (RunResult, error) {
	return RunResult{}, nil
}

type blockingQuotaAdapter struct {
	started chan struct{}
	release <-chan struct{}
	calls   atomic.Int32
	once    sync.Once
}

func (a *blockingQuotaAdapter) Quota(ctx context.Context) (ProviderQuota, error) {
	a.calls.Add(1)
	a.once.Do(func() { close(a.started) })
	select {
	case <-a.release:
		return ProviderQuota{Status: "available"}, nil
	case <-ctx.Done():
		return ProviderQuota{}, ctx.Err()
	}
}
func (a *blockingQuotaAdapter) Catalog(context.Context) (Catalog, error) { return Catalog{}, nil }
func (a *blockingQuotaAdapter) RunTurn(context.Context, RunRequest, func(string), func(ApprovalRequest) (string, error)) (RunResult, error) {
	return RunResult{}, nil
}

func setQuotaForTest(t *testing.T, registry *ProviderRegistry, providerID string, quota ProviderQuota) {
	t.Helper()
	for {
		registry.quotaMu.Lock()
		call := registry.quotaCalls[providerID]
		if call == nil {
			if registry.quotas == nil {
				registry.quotas = make(map[string]ProviderQuota)
			}
			registry.quotas[providerID] = quota
			registry.quotaMu.Unlock()
			return
		}
		registry.quotaMu.Unlock()
		<-call.done
	}
}

func TestQuotaCacheAndSelection(t *testing.T) {
	a := &quotaTestAdapter{status: "exhausted"}
	r := &ProviderRegistry{adapters: map[string]Adapter{"codex": a}}
	if r.Quota(context.Background(), "codex").Status != "exhausted" {
		t.Fatal("quota query did not report exhaustion")
	}
	if _, err := r.Catalog(context.Background(), "codex"); err == nil {
		t.Fatal("exhausted provider selectable")
	}
	if r.Quota(context.Background(), "codex").Status != "exhausted" || a.calls != 1 {
		t.Fatal("cache missed")
	}
	r.quotas["codex"] = ProviderQuota{Status: "exhausted", CheckedAt: time.Now().Add(-2 * quotaTTL)}
	a.status = "available"
	if r.Quota(context.Background(), "codex").Status != "available" {
		t.Fatal("expired quota was not refreshed")
	}
	if _, err := r.Catalog(context.Background(), "codex"); err != nil || a.calls != 2 {
		t.Fatalf("recovery failed: %v", err)
	}
	a.err = errors.New("private upstream detail")
	r.quotas = nil
	q := r.Quota(context.Background(), "codex")
	if q.Status != "unknown" || q.Guidance == "private upstream detail" {
		t.Fatal("failure must remain unknown and sanitized")
	}
	if err := r.requireQuota("codex"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"devin", "copilot", "claude", "antigravity", "goose", "other"} {
		if q := r.Quota(context.Background(), id); q.Status != "unknown" || q.Guidance == "" {
			t.Fatalf("missing guidance for %s", id)
		}
	}
}

func TestQuotaSnapshotRefreshesProvidersWithoutBlocking(t *testing.T) {
	release := make(chan struct{})
	codex := &blockingQuotaAdapter{started: make(chan struct{}), release: release}
	devin := &blockingQuotaAdapter{started: make(chan struct{}), release: release}
	r := &ProviderRegistry{adapters: map[string]Adapter{
		"codex": codex,
		"devin": devin,
	}}

	startedAt := time.Now()
	if got := r.QuotaSnapshot("codex").Status; got != "unknown" {
		t.Fatalf("initial Codex quota status = %q, want unknown", got)
	}
	if got := r.QuotaSnapshot("codex").Status; got != "unknown" {
		t.Fatalf("repeated Codex quota status = %q, want unknown", got)
	}
	if got := r.QuotaSnapshot("devin").Status; got != "unknown" {
		t.Fatalf("initial Devin quota status = %q, want unknown", got)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("quota snapshots blocked for %s", elapsed)
	}

	for _, started := range []<-chan struct{}{codex.started, devin.started} {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("provider quota refresh did not start independently")
		}
	}
	if codex.calls.Load() != 1 || devin.calls.Load() != 1 {
		t.Fatalf("duplicate quota refreshes: Codex=%d Devin=%d", codex.calls.Load(), devin.calls.Load())
	}
	close(release)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, providerID := range []string{"codex", "devin"} {
		if got := r.Quota(ctx, providerID).Status; got != "available" {
			t.Fatalf("quota status for %s = %q, want available", providerID, got)
		}
	}
}

func TestQuotaBlocksConversationAndTurn(t *testing.T) {
	service, _ := newRoutingTestService(t)
	ctx := context.Background()
	conversation, err := service.CreateConversation(ctx, "codex", "model-fast", "", "on-request", "manual")
	if err != nil {
		t.Fatal(err)
	}
	setQuotaForTest(t, service.providers, "codex", ProviderQuota{Status: "exhausted", CheckedAt: time.Now()})
	if _, err := service.CreateConversation(ctx, "codex", "model-fast", "", "on-request", "manual"); err == nil {
		t.Fatal("creation bypassed quota")
	}
	if _, err := service.StartTurn(conversation.ID, "hello", nil); err == nil {
		t.Fatal("turn bypassed quota")
	}
	for _, provider := range service.Providers() {
		if provider.ID == "codex" && provider.Quota.Status != "exhausted" {
			t.Fatal("API did not expose exhaustion")
		}
	}
	if _, _, _, ok := service.selectFallbackAdapter(ctx, "copilot"); ok {
		t.Fatal("fallback selected exhausted provider")
	}
}

func TestCodexQuotaRPC(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"success", "initialize", "read"} {
		t.Run(mode, func(t *testing.T) {
			wrapper := filepath.Join(t.TempDir(), "codex")
			script := fmt.Sprintf("#!/bin/sh\nCOCKPIT_QUOTA_HELPER=%s exec %q -test.run=TestQuotaRPCProcess -- \"$@\"\n", mode, binary)
			if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), quotaTimeout)
			defer cancel()
			q, err := NewCodexAdapter(wrapper).Quota(ctx)
			if mode == "success" {
				if err != nil || q.Status != "available" {
					t.Fatalf("quota=%+v err=%v", q, err)
				}
			} else if err == nil {
				t.Fatal("RPC failure accepted")
			}
		})
	}
	if _, err := NewCodexAdapter(filepath.Join(t.TempDir(), "absent")).Quota(context.Background()); err == nil {
		t.Fatal("missing binary accepted")
	}
}

func TestQuotaRPCProcess(t *testing.T) {
	mode := os.Getenv("COCKPIT_QUOTA_HELPER")
	if mode == "" {
		return
	}
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := decoder.Decode(&request); err != nil {
			os.Exit(0)
		}
		if len(request.ID) == 0 {
			continue
		}
		result := map[string]any{}
		if request.Method == "account/rateLimits/read" {
			result["ordinaryUsageAllowed"] = true
		}
		response := map[string]any{"id": request.ID, "result": result}
		if (mode == "initialize" && request.Method == "initialize") || (mode == "read" && request.Method == "account/rateLimits/read") {
			response = map[string]any{"id": request.ID, "error": map[string]any{"code": -1, "message": "unavailable"}}
		}
		if err := encoder.Encode(response); err != nil {
			os.Exit(1)
		}
	}
}

func TestLiveCodexQuota(t *testing.T) {
	if os.Getenv("COCKPIT_TEST_LIVE_QUOTA") != "1" {
		t.Skip("explicit live opt-in required")
	}
	binary, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), quotaTimeout)
	defer cancel()
	q, err := NewCodexAdapter(binary).Quota(ctx)
	if err != nil {
		t.Fatal("live Codex quota query failed")
	}
	t.Logf("status=%s windows=%d balance_present=%t", q.Status, len(q.Windows), q.Balance != nil)
}
