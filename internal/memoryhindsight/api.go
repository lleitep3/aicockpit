package memoryhindsight

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/memory"
	hsclient "github.com/vectorize-io/hindsight/hindsight-clients/go"
)

const apiTimeout = 30 * time.Second

// API adapts the official Hindsight Go client to Client.
type API struct {
	raw *hsclient.APIClient
}

// NewAPI returns a client for baseURL. token is sent as a Bearer token when
// it is not empty. The token is not stored on API and must not be logged.
func NewAPI(baseURL, token string) (*API, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if err := validateAPIURL(baseURL); err != nil {
		return nil, err
	}
	if strings.ContainsAny(token, "\r\n\x00") {
		return nil, errors.New("hindsight api token contains a newline")
	}
	cfg := hsclient.NewConfiguration()
	cfg.UserAgent = "aicockpit-memory-hindsight"
	cfg.Servers = hsclient.ServerConfigurations{{URL: baseURL}}
	cfg.HTTPClient = &http.Client{Timeout: apiTimeout}
	if token != "" {
		cfg.AddDefaultHeader("Authorization", "Bearer "+token)
	}
	return &API{raw: hsclient.NewAPIClient(cfg)}, nil
}

// Retain stores items in bank.
func (a *API) Retain(ctx context.Context, bank string, items []memory.Item) error {
	if err := a.ready(ctx, "retain"); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	payload := make([]hsclient.MemoryItem, 0, len(items))
	for _, item := range items {
		payload = append(payload, memoryItem(item))
	}
	req := hsclient.NewRetainRequest(payload)
	resp, httpResp, err := a.raw.MemoryAPI.RetainMemories(ctx, bank).RetainRequest(*req).Execute()
	code := discard(httpResp)
	if code == http.StatusNotFound {
		return fmt.Errorf("hindsight retain %s: bank not found", bank)
	}
	if err != nil {
		return fmt.Errorf("hindsight retain %s: %w", bank, err)
	}
	if resp == nil || !resp.Success {
		return fmt.Errorf("hindsight retain %s: unsuccessful", bank)
	}
	return nil
}

// Recall searches one bank.
func (a *API) Recall(ctx context.Context, bank string, query memory.Query) ([]memory.Hit, error) {
	if err := a.ready(ctx, "recall"); err != nil {
		return nil, err
	}
	req := hsclient.NewRecallRequest(query.Text)
	if len(query.Tags) > 0 {
		req.Tags = append([]string(nil), query.Tags...)
		match := "all_strict"
		req.TagsMatch = &match
	}
	resp, httpResp, err := a.raw.MemoryAPI.RecallMemories(ctx, bank).RecallRequest(*req).Execute()
	code := discard(httpResp)
	if code == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("hindsight recall %s: %w", bank, err)
	}
	if resp == nil {
		return nil, fmt.Errorf("hindsight recall %s: empty response", bank)
	}
	hits := make([]memory.Hit, 0, len(resp.Results))
	for _, result := range resp.Results {
		hit, ok := hitFromRecall(bank, result)
		if ok {
			hits = append(hits, hit)
		}
	}
	return hits, nil
}

// Reflect asks one bank for a synthesized answer.
func (a *API) Reflect(ctx context.Context, bank string, query memory.Query) (string, []memory.Hit, error) {
	if err := a.ready(ctx, "reflect"); err != nil {
		return "", nil, err
	}
	req := hsclient.NewReflectRequest(query.Text)
	if len(query.Tags) > 0 {
		req.Tags = append([]string(nil), query.Tags...)
		match := "all_strict"
		req.TagsMatch = &match
	}
	resp, httpResp, err := a.raw.MemoryAPI.Reflect(ctx, bank).ReflectRequest(*req).Execute()
	code := discard(httpResp)
	if code == http.StatusNotFound {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("hindsight reflect %s: %w", bank, err)
	}
	if resp == nil {
		return "", nil, fmt.Errorf("hindsight reflect %s: empty response", bank)
	}
	hits := reflectHits(bank, resp)
	return strings.TrimSpace(resp.GetText()), hits, nil
}

func (a *API) ready(ctx context.Context, op string) error {
	if a == nil || a.raw == nil {
		return fmt.Errorf("hindsight %s: nil client", op)
	}
	if ctx == nil {
		return fmt.Errorf("hindsight %s: %w", op, memory.ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("hindsight %s: %w", op, err)
	}
	return nil
}

func memoryItem(item memory.Item) hsclient.MemoryItem {
	mem := hsclient.MemoryItem{Content: hsclient.TextContent(item.Content)}
	if item.Context != "" {
		mem.Context = *hsclient.NewNullableString(hsclient.PtrString(item.Context))
	}
	if item.DocumentID != "" {
		mem.DocumentId = *hsclient.NewNullableString(hsclient.PtrString(item.DocumentID))
	}
	if !item.Timestamp.IsZero() {
		stamp := item.Timestamp.UTC()
		mem.Timestamp = *hsclient.NewNullableTimestamp(&hsclient.Timestamp{TimeTime: hsclient.PtrTime(stamp)})
	}
	if len(item.Tags) > 0 {
		mem.Tags = append([]string(nil), item.Tags...)
	}
	if len(item.Metadata) > 0 {
		mem.Metadata = copyMeta(item.Metadata)
	}
	return mem
}

func hitFromRecall(bank string, result hsclient.RecallResult) (memory.Hit, bool) {
	text := strings.TrimSpace(result.GetText())
	if text == "" {
		return memory.Hit{}, false
	}
	hit := memory.Hit{
		Item: memory.Item{
			ID:         result.GetId(),
			Title:      titleFrom(text),
			Content:    text,
			Context:    result.GetContext(),
			Tags:       copyTags(result.GetTags()),
			Bank:       bank,
			DocumentID: result.GetDocumentId(),
			Metadata:   copyMeta(result.GetMetadata()),
		},
	}
	if scores, ok := result.GetScoresOk(); ok && scores != nil {
		hit.Score = float64(scores.GetFinal())
	}
	return hit, true
}

func reflectHits(bank string, resp *hsclient.ReflectResponse) []memory.Hit {
	based, ok := resp.GetBasedOnOk()
	if !ok || based == nil {
		return nil
	}
	facts := based.GetMemories()
	if len(facts) == 0 {
		return nil
	}
	hits := make([]memory.Hit, 0, len(facts))
	for _, fact := range facts {
		text := strings.TrimSpace(fact.GetText())
		if text == "" {
			continue
		}
		hits = append(hits, memory.Hit{
			Item: memory.Item{
				ID:         fact.GetId(),
				Title:      titleFrom(text),
				Content:    text,
				Context:    fact.GetContext(),
				Tags:       copyTags(fact.GetTags()),
				Bank:       bank,
				DocumentID: fact.GetDocumentId(),
				Metadata:   copyMeta(fact.GetMetadata()),
			},
		})
	}
	if len(hits) == 0 {
		return nil
	}
	return hits
}

func titleFrom(text string) string {
	line := text
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		line = text[:i]
	}
	line = strings.TrimSpace(strings.TrimLeft(line, "#"))
	if line == "" {
		return "memory"
	}
	runes := []rune(line)
	if len(runes) > 80 {
		return string(runes[:77]) + "..."
	}
	return line
}

func copyTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	return append([]string(nil), tags...)
}

func copyMeta(meta map[string]string) map[string]string {
	if len(meta) == 0 {
		return nil
	}
	out := make(map[string]string, len(meta))
	for key, value := range meta {
		out[key] = value
	}
	return out
}

func discard(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	if resp.Body != nil {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	}
	return resp.StatusCode
}
