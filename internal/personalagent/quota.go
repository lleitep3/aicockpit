package personalagent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const quotaTTL = 30 * time.Second
const quotaTimeout = 10 * time.Second

var quotaTimeouts = map[string]time.Duration{
	"antigravity": 40 * time.Second,
}

type ProviderQuota struct {
	Status    string        `json:"status"`
	Source    string        `json:"source,omitempty"`
	CheckedAt time.Time     `json:"checked_at,omitempty"`
	Windows   []QuotaWindow `json:"windows,omitempty"`
	Balance   *string       `json:"balance,omitempty"`
	Unlimited bool          `json:"unlimited"`
	Guidance  string        `json:"guidance,omitempty"`
}

type QuotaWindow struct {
	Name             string     `json:"name"`
	RemainingPercent float64    `json:"remaining_percent"`
	ResetsAt         *time.Time `json:"resets_at,omitempty"`
}

type quotaReader interface {
	Quota(context.Context) (ProviderQuota, error)
}

type quotaCall struct {
	done chan struct{}
}

func quotaGuidance(id string) string {
	switch id {
	case "codex":
		return "Codex: /status; consulta automática via app-server account/rateLimits/read."
	case "devin":
		return "Devin: consulta automática via SeatManagementService/GetPlanStatus (quota semanal, resets e saldo excedente)."
	case "copilot":
		return "Copilot: consulta automática via api.github.com/copilot_internal/user (créditos de chat, completions e data de renovação)."
	case "claude":
		return "Claude: sem consulta automática de cota; /usage mostra limites do plano no CLI interativo."
	case "antigravity":
		return "Antigravity: consulta automática via `agy --print /usage` (cotas por grupo de modelos e renovação)."
	case "goose":
		return "Goose: cota pertence ao backend configurado; sem saldo universal da CLI."
	default:
		return "Consulta de cota não disponível."
	}
}

func (r *ProviderRegistry) Quota(ctx context.Context, id string) ProviderQuota {
	return r.requestQuota(ctx, id, true)
}

func (r *ProviderRegistry) QuotaSnapshot(id string) ProviderQuota {
	return r.requestQuota(context.Background(), id, false)
}

func (r *ProviderRegistry) requestQuota(ctx context.Context, id string, wait bool) ProviderQuota {
	if ctx == nil {
		ctx = context.Background()
	}
	r.quotaMu.Lock()
	now := time.Now().UTC()
	q, cached := r.quotas[id]
	if cached && now.Sub(q.CheckedAt) < quotaTTL {
		r.quotaMu.Unlock()
		return q
	}
	if !cached {
		q = ProviderQuota{Status: "unknown", CheckedAt: now, Guidance: quotaGuidance(id)}
	}
	if r.quotaCalls == nil {
		r.quotaCalls = make(map[string]*quotaCall)
	}
	call := r.quotaCalls[id]
	start := call == nil
	if start {
		call = &quotaCall{done: make(chan struct{})}
		r.quotaCalls[id] = call
	}
	r.quotaMu.Unlock()

	if start {
		if wait {
			return r.refreshQuota(ctx, id, call)
		}
		go r.refreshQuota(context.Background(), id, call)
		return q
	}
	if !wait {
		return q
	}
	select {
	case <-call.done:
		r.quotaMu.Lock()
		q = r.quotas[id]
		r.quotaMu.Unlock()
		return q
	case <-ctx.Done():
		return q
	}
}

func (r *ProviderRegistry) refreshQuota(ctx context.Context, id string, call *quotaCall) ProviderQuota {
	now := time.Now().UTC()
	q := ProviderQuota{Status: "unknown", CheckedAt: now, Guidance: quotaGuidance(id)}
	timeout := quotaTimeouts[id]
	if timeout == 0 {
		timeout = quotaTimeout
	}
	var read func(context.Context) (ProviderQuota, error)
	if query, ok := r.adapters[id].(quotaReader); ok {
		read = query.Quota
	}
	if read != nil {
		queryCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		result, err := read(queryCtx)
		if err == nil {
			q = result
			q.Guidance = quotaGuidance(id)
		} else {
			q.Guidance = "Falha ao consultar cota; tente novamente. " + q.Guidance
		}
	}
	q.CheckedAt = time.Now().UTC()
	r.quotaMu.Lock()
	if r.quotas == nil {
		r.quotas = make(map[string]ProviderQuota)
	}
	r.quotas[id] = q
	delete(r.quotaCalls, id)
	close(call.done)
	r.quotaMu.Unlock()
	return q
}

func (r *ProviderRegistry) requireQuota(id string) error {
	if r.QuotaSnapshot(id).Status == "exhausted" {
		return fmt.Errorf("provider %q está sem cota disponível; aguarde renovação ou selecione outro provider", id)
	}
	return nil
}

func (a *CodexAdapter) Quota(ctx context.Context) (ProviderQuota, error) {
	client, err := a.startAppServer()
	if err != nil {
		return ProviderQuota{}, err
	}
	defer client.Close()
	if err := client.Initialize(ctx); err != nil {
		return ProviderQuota{}, err
	}
	data, err := client.Call(ctx, "account/rateLimits/read", map[string]any{})
	if err != nil {
		return ProviderQuota{}, err
	}
	return parseCodexQuota(data)
}

func parseCodexQuota(data json.RawMessage) (ProviderQuota, error) {
	type window struct {
		UsedPercent *float64 `json:"usedPercent"`
		ResetsAt    *int64   `json:"resetsAt"`
	}
	type snapshot struct {
		LimitID   string  `json:"limitId"`
		Primary   *window `json:"primary"`
		Secondary *window `json:"secondary"`
		Credits   *struct {
			Balance    *string `json:"balance"`
			Unlimited  bool    `json:"unlimited"`
			HasCredits *bool   `json:"hasCredits"`
		} `json:"credits"`
		SpendControlReached *bool `json:"spendControlReached"`
	}
	var response struct {
		Allowed *bool               `json:"ordinaryUsageAllowed"`
		Limits  snapshot            `json:"rateLimits"`
		ByID    map[string]snapshot `json:"rateLimitsByLimitId"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return ProviderQuota{}, fmt.Errorf("decode Codex quota: %w", err)
	}
	q := ProviderQuota{Status: "unknown", Source: "codex-app-server:account/rateLimits/read"}
	limits := response.Limits
	if shared, ok := response.ByID["codex"]; ok {
		limits = shared
	}
	if limits.LimitID != "" && limits.LimitID != "codex" {
		return q, nil
	}
	for _, entry := range []struct {
		name  string
		value *window
	}{{"primary", limits.Primary}, {"secondary", limits.Secondary}} {
		w := entry.value
		if w == nil || w.UsedPercent == nil || *w.UsedPercent < 0 || *w.UsedPercent > 100 {
			continue
		}
		item := QuotaWindow{Name: entry.name, RemainingPercent: 100 - *w.UsedPercent}
		if w.ResetsAt != nil && *w.ResetsAt > 0 {
			reset := time.Unix(*w.ResetsAt, 0).UTC()
			item.ResetsAt = &reset
		}
		q.Windows = append(q.Windows, item)
	}
	if limits.Credits != nil {
		q.Balance = limits.Credits.Balance
		q.Unlimited = limits.Credits.Unlimited
	}
	if response.Allowed != nil {
		if *response.Allowed {
			q.Status = "available"
		} else {
			q.Status = "exhausted"
		}
	}
	if limits.Credits != nil && (limits.Credits.Unlimited || (limits.Credits.HasCredits != nil && *limits.Credits.HasCredits)) {
		q.Status = "available"
	}
	if limits.SpendControlReached != nil && *limits.SpendControlReached {
		q.Status = "exhausted"
	}
	return q, nil
}
