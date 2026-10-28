package personalagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var copilotQuotaURL = "https://api.github.com/copilot_internal/user"

var quotaHTTPClient = &http.Client{Timeout: 8 * time.Second}

func quotaHTTPJSON(ctx context.Context, method, url string, headers map[string]string, body []byte) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := quotaHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("quota endpoint returned %s", resp.Status)
	}
	return json.RawMessage(data), nil
}

// --- Copilot: GitHub copilot_internal API ---

func copilotToken(ctx context.Context) (string, error) {
	for _, env := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(env)); token != "" {
			return token, nil
		}
	}
	gh, err := exec.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("no GitHub token available")
	}
	tokenCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(tokenCtx, gh, "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("no GitHub token available")
	}
	if token := strings.TrimSpace(string(output)); token != "" {
		return token, nil
	}
	return "", fmt.Errorf("no GitHub token available")
}

func (a *CopilotAdapter) Quota(ctx context.Context) (ProviderQuota, error) {
	token, err := copilotToken(ctx)
	if err != nil {
		return ProviderQuota{}, err
	}
	data, err := quotaHTTPJSON(ctx, http.MethodGet, copilotQuotaURL, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
	}, nil)
	if err != nil {
		return ProviderQuota{}, err
	}
	return parseCopilotQuota(data)
}

func parseCopilotQuota(data json.RawMessage) (ProviderQuota, error) {
	type snapshot struct {
		PercentRemaining *float64 `json:"percent_remaining"`
		QuotaRemaining   *float64 `json:"quota_remaining"`
		Remaining        *float64 `json:"remaining"`
		Entitlement      *float64 `json:"entitlement"`
		Unlimited        bool     `json:"unlimited"`
		OveragePermitted bool     `json:"overage_permitted"`
	}
	var response struct {
		ResetDate string              `json:"quota_reset_date"`
		ResetUTC  string              `json:"quota_reset_date_utc"`
		Snapshots map[string]snapshot `json:"quota_snapshots"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return ProviderQuota{}, fmt.Errorf("decode Copilot quota: %w", err)
	}
	q := ProviderQuota{Status: "unknown", Source: "github:copilot_internal/user"}

	reset := parseCopilotResetDate(response.ResetUTC, response.ResetDate)
	chat, ok := response.Snapshots["chat"]
	if !ok {
		return q, nil
	}
	window := QuotaWindow{Name: "chat", ResetsAt: reset}
	switch {
	case chat.PercentRemaining != nil && *chat.PercentRemaining >= 0 && *chat.PercentRemaining <= 100:
		window.RemainingPercent = *chat.PercentRemaining
	case chat.Entitlement != nil && chat.QuotaRemaining != nil && *chat.Entitlement > 0:
		window.RemainingPercent = 100 * *chat.QuotaRemaining / *chat.Entitlement
	}
	q.Windows = append(q.Windows, window)
	if chat.Entitlement != nil && *chat.Entitlement > 0 && chat.Remaining != nil {
		balance := fmt.Sprintf("%v/%v créditos", trimFloat(*chat.Remaining), trimFloat(*chat.Entitlement))
		q.Balance = &balance
	}
	q.Unlimited = chat.Unlimited

	switch {
	case chat.Unlimited || chat.OveragePermitted:
		q.Status = "available"
	case chat.Remaining != nil && *chat.Remaining > 0:
		q.Status = "available"
	case chat.Remaining != nil && *chat.Remaining <= 0 && chat.Entitlement != nil && *chat.Entitlement > 0:
		q.Status = "exhausted"
	case chat.QuotaRemaining != nil && *chat.QuotaRemaining <= 0 && chat.Entitlement != nil && *chat.Entitlement > 0:
		q.Status = "exhausted"
	case chat.PercentRemaining != nil && *chat.PercentRemaining > 0:
		q.Status = "available"
	}
	return q, nil
}

func parseCopilotResetDate(utc, date string) *time.Time {
	for _, candidate := range []string{utc, date} {
		if candidate == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if parsed, err := time.Parse(layout, candidate); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

func trimFloat(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// --- Devin: Windsurf seat management (Connect-RPC JSON) ---

var devinCredentialsPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "devin", "credentials.toml")
}

func devinCredentials() (server, token string, err error) {
	data, err := os.ReadFile(devinCredentialsPath())
	if err != nil {
		return "", "", err
	}
	keyRe := regexp.MustCompile(`(?m)^\s*windsurf_api_key\s*=\s*"([^"]+)"`)
	urlRe := regexp.MustCompile(`(?m)^\s*api_server_url\s*=\s*"([^"]+)"`)
	if match := keyRe.FindSubmatch(data); match != nil {
		token = string(match[1])
	}
	if match := urlRe.FindSubmatch(data); match != nil {
		server = string(match[1])
	}
	if token == "" || server == "" {
		return "", "", fmt.Errorf("Devin credentials incomplete")
	}
	return server, token, nil
}

func (a *DevinAdapter) Quota(ctx context.Context) (ProviderQuota, error) {
	server, token, err := devinCredentials()
	if err != nil {
		return ProviderQuota{}, err
	}
	url := strings.TrimRight(server, "/") + "/exa.seat_management_pb.SeatManagementService/GetPlanStatus"
	data, err := quotaHTTPJSON(ctx, http.MethodPost, url, map[string]string{
		"X-Auth-Token":             token,
		"Connect-Protocol-Version": "1",
	}, []byte("{}"))
	if err != nil {
		return ProviderQuota{}, err
	}
	return parseDevinPlanStatus(data)
}

func parseDevinPlanStatus(data json.RawMessage) (ProviderQuota, error) {
	var response struct {
		PlanStatus struct {
			PlanInfo struct {
				PlanName        string `json:"planName"`
				BillingStrategy string `json:"billingStrategy"`
			} `json:"planInfo"`
			PlanEnd                     string   `json:"planEnd"`
			AvailablePromptCredits      int64    `json:"availablePromptCredits"`
			WeeklyQuotaRemainingPercent *float64 `json:"weeklyQuotaRemainingPercent"`
			OverageBalanceMicros        string   `json:"overageBalanceMicros"`
			DailyQuotaResetAtUnix       string   `json:"dailyQuotaResetAtUnix"`
			WeeklyQuotaResetAtUnix      string   `json:"weeklyQuotaResetAtUnix"`
		} `json:"planStatus"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return ProviderQuota{}, fmt.Errorf("decode Devin plan status: %w", err)
	}
	ps := response.PlanStatus
	q := ProviderQuota{Status: "unknown", Source: "devin:SeatManagementService/GetPlanStatus"}

	var overageMicros int64
	if ps.OverageBalanceMicros != "" {
		if v, err := strconv.ParseInt(ps.OverageBalanceMicros, 10, 64); err == nil {
			overageMicros = v
		}
	}
	if overageMicros > 0 {
		balance := fmt.Sprintf("$%.2f créditos excedentes", float64(overageMicros)/1e6)
		q.Balance = &balance
	}
	if reset := unixTimePtr(ps.WeeklyQuotaResetAtUnix); reset != nil {
		window := QuotaWindow{Name: "weekly", ResetsAt: reset}
		if ps.WeeklyQuotaRemainingPercent != nil && *ps.WeeklyQuotaRemainingPercent >= 0 {
			window.RemainingPercent = float64(*ps.WeeklyQuotaRemainingPercent)
		}
		q.Windows = append(q.Windows, window)
	}

	switch {
	case ps.AvailablePromptCredits < 0 && ps.WeeklyQuotaRemainingPercent == nil:
		q.Status = "unknown"
	case ps.WeeklyQuotaRemainingPercent == nil:
		if ps.AvailablePromptCredits > 0 || overageMicros > 0 {
			q.Status = "available"
		}
	case *ps.WeeklyQuotaRemainingPercent > 0:
		q.Status = "available"
	case overageMicros > 0 || ps.AvailablePromptCredits > 0 || ps.AvailablePromptCredits < 0:
		// Weekly exhausted but overage balance or unlimited prompt credits remain.
		q.Status = "available"
	default:
		q.Status = "exhausted"
	}
	return q, nil
}

func unixTimePtr(raw string) *time.Time {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return nil
	}
	t := time.Unix(v, 0).UTC()
	return &t
}

// --- Antigravity: `agy --print "/usage"` ---

func (a *AntigravityAdapter) Quota(ctx context.Context) (ProviderQuota, error) {
	cmd := exec.CommandContext(ctx, a.binary, "--print", "/usage", "--print-timeout", "25s")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ProviderQuota{}, ctx.Err()
		}
		return ProviderQuota{}, fmt.Errorf("read Antigravity quota: %w", err)
	}
	return parseAntigravityUsage(string(output)), nil
}

var agyUsageLine = regexp.MustCompile(`^(.+?)\t(.+?)\t(\d+(?:\.\d+)?)%\t(\S+)\s*$`)

func parseAntigravityUsage(raw string) ProviderQuota {
	clean := ansiEscapeRegex.ReplaceAllString(raw, "")
	q := ProviderQuota{Status: "unknown", Source: "agy:/usage"}
	for _, line := range strings.Split(clean, "\n") {
		match := agyUsageLine.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		percent, err := strconv.ParseFloat(match[3], 64)
		if err != nil || percent < 0 || percent > 100 {
			continue
		}
		window := QuotaWindow{
			Name:             strings.TrimSpace(match[1] + " " + match[2]),
			RemainingPercent: percent,
		}
		if reset, err := time.Parse(time.RFC3339, match[4]); err == nil {
			window.ResetsAt = &reset
		}
		q.Windows = append(q.Windows, window)
	}
	if len(q.Windows) == 0 {
		return q
	}
	available := false
	for _, window := range q.Windows {
		if window.RemainingPercent > 0 {
			available = true
			break
		}
	}
	if available {
		q.Status = "available"
	}
	return q
}
