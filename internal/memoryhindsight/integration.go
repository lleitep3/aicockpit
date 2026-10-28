package memoryhindsight

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/memory"
)

// Integration plugs Hindsight into knowledge-base search and index.
// Disabled and failed integrations leave the local knowledge base in place.
type Integration struct {
	mem     memory.Memory
	banks   Names
	source  DocumentSource
	enabled bool
}

// DocumentSource lists knowledge-base documents to retain.
type DocumentSource interface {
	// Documents returns markdown documents under roots.
	Documents(roots []string) ([]*kb.Document, error)
}

var _ kb.Contributor = (*Integration)(nil)
var _ kb.Contributor = failedContributor{}

// Open builds the integration from cockpit settings.
// A disabled server returns an integration whose Search and Index do nothing.
// token is an optional API bearer token and is not logged.
func Open(settings hindsight.Settings, id Identity, token string) (*Integration, error) {
	if !settings.Enabled {
		return &Integration{}, nil
	}
	resolved, err := settings.Resolve()
	if err != nil {
		return nil, fmt.Errorf("memory-hindsight: %w", err)
	}
	base := strings.TrimSpace(settings.APIURL)
	if base == "" {
		base = resolved.APIURL()
	}
	banks, err := ResolveBanks(id, settings.Banks)
	if err != nil {
		return nil, err
	}
	client, err := NewAPI(base, token)
	if err != nil {
		return nil, fmt.Errorf("memory-hindsight: %w", err)
	}
	backend, err := New(client, banks)
	if err != nil {
		return nil, err
	}
	return &Integration{mem: backend, banks: banks, source: fileSource{}, enabled: true}, nil
}

// Enabled reports whether Search and Index talk to Hindsight.
func (in *Integration) Enabled() bool {
	return in != nil && in.enabled
}

// Memory returns the Hindsight memory backend.
// It is nil when the integration is disabled or was not opened.
func (in *Integration) Memory() memory.Memory {
	if in == nil || !in.enabled {
		return nil
	}
	return in.mem
}

// Banks returns the resolved project and user bank ids.
func (in *Integration) Banks() Names {
	if in == nil {
		return Names{}
	}
	return in.banks
}

// Search recalls both banks and returns text to append to local results.
// An empty string means there is nothing to add. Errors mean Hindsight was
// enabled but the recall did not succeed.
func (in *Integration) Search(ctx context.Context, query string, limit int) (string, error) {
	if in == nil || !in.enabled {
		return "", nil
	}
	if in.mem == nil {
		return "", errors.New("memory-hindsight: nil memory")
	}
	recalled, err := in.mem.Recall(ctx, memory.Query{Text: query, Limit: limit})
	if err != nil {
		return "", err
	}
	return FormatRecall(recalled), nil
}

// Index retains markdown documents into the project bank.
// fast is accepted for the knowledge-base extension contract. Retention is a
// full upsert keyed by document id either way.
func (in *Integration) Index(ctx context.Context, roots []string, fast bool) error {
	if in == nil || !in.enabled {
		return nil
	}
	if in.mem == nil {
		return errors.New("memory-hindsight: nil memory")
	}
	if len(roots) == 0 {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("memory index: %w", memory.ErrNilContext)
	}
	source := in.source
	if source == nil {
		source = fileSource{}
	}
	docs, readErr := source.Documents(roots)
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i] == nil {
			return false
		}
		if docs[j] == nil {
			return true
		}
		return docs[i].Path < docs[j].Path
	})
	var retainErr error
	for _, doc := range docs {
		if err := ctx.Err(); err != nil {
			retainErr = err
			break
		}
		item, ok := itemFromKB(doc)
		if !ok {
			continue
		}
		if _, err := in.mem.Retain(ctx, item); err != nil && retainErr == nil {
			retainErr = err
		}
	}
	return errors.Join(readErr, retainErr)
}

// Failed returns a contributor that reports err from Search and Index.
// kb search and kb index treat that error as a skip and keep local results.
func Failed(err error) kb.Contributor {
	if err == nil {
		err = errors.New("memory-hindsight unavailable")
	}
	return failedContributor{err: err}
}

type failedContributor struct {
	err error
}

func (f failedContributor) Search(context.Context, string, int) (string, error) {
	return "", f.err
}

func (f failedContributor) Index(context.Context, []string, bool) error {
	return f.err
}

// FormatRecall renders recall hits for the knowledge-base search output.
// No hits produce an empty string so local search stays unchanged.
func FormatRecall(result *memory.RecallResult) string {
	if result == nil || len(result.Hits) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Hindsight recall for %q (%d):\n", result.Query, len(result.Hits))
	for i, hit := range result.Hits {
		if i > 0 {
			b.WriteByte('\n')
		}
		title := strings.TrimSpace(hit.Item.Title)
		if title == "" {
			title = hit.Item.ID
		}
		if title == "" {
			title = "memory"
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, title)
		fmt.Fprintf(&b, "   bank: %s\n", hit.Item.Bank)
		fmt.Fprintf(&b, "   score: %.2f\n", hit.Score)
		if snippet := snippet(hit.Item.Content); snippet != "" {
			fmt.Fprintf(&b, "   %s\n", snippet)
		}
	}
	return b.String()
}

func snippet(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) > 240 {
		return string(runes[:237]) + "..."
	}
	return text
}

func itemFromKB(doc *kb.Document) (memory.Item, bool) {
	if doc == nil {
		return memory.Item{}, false
	}
	title := strings.TrimSpace(doc.Metadata.Title)
	body := strings.TrimSpace(doc.Content)
	desc := strings.TrimSpace(doc.Metadata.Description)
	var b strings.Builder
	if title != "" {
		fmt.Fprintf(&b, "# %s\n\n", title)
	}
	if desc != "" {
		fmt.Fprintf(&b, "%s\n\n", desc)
	}
	b.WriteString(body)
	content := strings.TrimSpace(b.String())
	if content == "" {
		return memory.Item{}, false
	}
	meta := map[string]string{"source": "kb"}
	if doc.Path != "" {
		meta["path"] = oneLine(doc.Path, 180)
	}
	item := memory.Item{
		Title:      title,
		Content:    content,
		Context:    "knowledge base",
		Tags:       doc.Metadata.Tags,
		DocumentID: documentToken(doc),
		Metadata:   meta,
		Timestamp:  doc.Metadata.Modified,
	}
	if item.Timestamp.IsZero() {
		item.Timestamp = doc.Metadata.Created
	}
	return item, true
}

func documentToken(doc *kb.Document) string {
	raw := strings.TrimSpace(doc.Path)
	if raw == "" {
		raw = strings.TrimSpace(doc.ID)
	}
	part := sanitizePart(strings.NewReplacer("\\", "-", "/", "-").Replace(raw))
	if part == "" || len(part) > 120 {
		sum := sha256.Sum256([]byte(doc.Path + "\n" + doc.ID))
		return "doc-" + hex.EncodeToString(sum[:8])
	}
	if part[0] >= '0' && part[0] <= '9' {
		part = "doc-" + part
	}
	if len(part) > 120 {
		part = part[:120]
	}
	return part
}

func oneLine(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

type fileSource struct{}

func (fileSource) Documents(roots []string) ([]*kb.Document, error) {
	var docs []*kb.Document
	var errs []error
	for _, root := range roots {
		repo := kb.NewFileRepository(root)
		found, err := repo.ListDocuments(".")
		if err != nil {
			errs = append(errs, fmt.Errorf("read %s: %w", root, err))
			continue
		}
		docs = append(docs, found...)
	}
	return docs, errors.Join(errs...)
}
