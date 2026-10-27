package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lleitep3/aicockpit/internal/kb"
)

// kbStore is the subset of the knowledge-base manager this backend uses.
// *kb.Manager implements it. Tests substitute a fake.
type kbStore interface {
	Search(query string) (*kb.SearchResults, error)
	ListDocuments() ([]*kb.Document, error)
	AddDocument(doc *kb.Document, rootIndex int) error
	RebuildIndex() error
}

// KB is a Memory backed by the local knowledge base. It is not safe for
// concurrent use, matching kb.Manager.
//
// Retain writes one markdown file per item under the first configured root:
//
//	memory/<bank>/<id>.md
//
// The file uses the standard knowledge-base metadata header. description is
// the item context, so keyword search can see it. tags always include
// "memory" and "bank:<bank>", plus the caller tags. related stores
// "doc:<document id>" and "meta:<key>=<value>" entries. author is
// "aicockpit-memory". Retain then rebuilds the index so the next Recall sees
// the file. The same id and bank overwrite the previous file.
//
// Recall uses knowledge-base search across every configured root, then loads
// the full documents. Documents that Retain did not write are returned too:
// title, body, and description become the item title, content, and context.
// Query.Bank keeps only documents tagged for that bank. Query.Tags keeps
// documents that carry every requested tag. Query.Context is added to the
// search text and is not an exact-match filter.
//
// Reflect recalls and joins the matching excerpts. It does not call a model.
// Backends that cannot summarize return ErrReflectNotSupported instead.
type KB struct {
	store kbStore
}

var (
	_ Memory  = (*KB)(nil)
	_ Memory  = (*InMemory)(nil)
	_ Memory  = (*Fake)(nil)
	_ kbStore = (*kb.Manager)(nil)
)

// NewKB returns a Memory that stores items in the local knowledge base.
// roots are the knowledge-base roots; the first root must already be a
// directory and receives retained files. indexPath is the index cache file.
// Search still covers every root.
func NewKB(roots []string, indexPath string) (*KB, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("memory kb: no roots configured")
	}
	if strings.TrimSpace(indexPath) == "" {
		return nil, fmt.Errorf("memory kb: empty index path")
	}
	info, err := os.Stat(roots[0])
	if err != nil {
		return nil, fmt.Errorf("memory kb: first root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("memory kb: first root is not a directory")
	}
	return newKB(kb.NewManager(roots, indexPath)), nil
}

// NewKBWithManager uses an existing knowledge-base manager. The manager must
// already have at least one root and an index path. Retained files are written
// to the first root.
func NewKBWithManager(m *kb.Manager) (*KB, error) {
	if m == nil {
		return nil, fmt.Errorf("memory kb: nil manager")
	}
	if len(m.GetRoots()) == 0 {
		return nil, fmt.Errorf("memory kb: no roots configured")
	}
	if strings.TrimSpace(m.GetIndexPath()) == "" {
		return nil, fmt.Errorf("memory kb: empty index path")
	}
	return newKB(m), nil
}

func newKB(store kbStore) *KB {
	return &KB{store: store}
}

// Retain writes item into the knowledge base and rebuilds the index.
func (k *KB) Retain(ctx context.Context, item Item) (Item, error) {
	if err := checkCtx(ctx, "retain"); err != nil {
		return Item{}, err
	}
	stored, err := NormalizeItem(item)
	if err != nil {
		return Item{}, err
	}
	if err := k.store.AddDocument(documentFromItem(stored), 0); err != nil {
		return Item{}, fmt.Errorf("memory retain: %w", err)
	}
	if err := k.store.RebuildIndex(); err != nil {
		return Item{}, fmt.Errorf("memory retain: %w", err)
	}
	return stored, nil
}

// Recall searches the knowledge base.
func (k *KB) Recall(ctx context.Context, query Query) (*RecallResult, error) {
	return k.recall(ctx, "recall", query)
}

// Reflect searches the knowledge base and joins the matching excerpts.
func (k *KB) Reflect(ctx context.Context, query Query) (*ReflectResult, error) {
	recalled, err := k.recall(ctx, "reflect", query)
	if err != nil {
		return nil, err
	}
	return summarize(recalled.Query, recalled.Hits), nil
}

func (k *KB) recall(ctx context.Context, op string, query Query) (*RecallResult, error) {
	if err := checkCtx(ctx, op); err != nil {
		return nil, err
	}
	normalized, search, err := NormalizeQuery(op, query)
	if err != nil {
		return nil, err
	}
	results, err := k.store.Search(search)
	if err != nil {
		return nil, fmt.Errorf("memory %s: %w", op, err)
	}
	if err := checkCtx(ctx, op); err != nil {
		return nil, err
	}
	docs, err := k.store.ListDocuments()
	if err != nil {
		return nil, fmt.Errorf("memory %s: %w", op, err)
	}
	byPath := make(map[string]*kb.Document, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		byPath[filepath.ToSlash(doc.Path)] = doc
	}
	if results == nil {
		results = &kb.SearchResults{}
	}
	hits := make([]Hit, 0)
	for _, result := range results.Results {
		doc := byPath[filepath.ToSlash(result.Path)]
		item := itemFromDocument(doc, result)
		if normalized.Bank != "" && item.Bank != normalized.Bank {
			continue
		}
		if !hasTags(item.Tags, normalized.Tags) {
			continue
		}
		hits = append(hits, Hit{
			Item:  item,
			Score: result.Score,
			Path:  filepath.ToSlash(result.Path),
		})
	}
	limited, total := limitHits(hits, normalized.Limit)
	return &RecallResult{Query: displayQuery(normalized), Hits: limited, Total: total}, nil
}

func documentFromItem(item Item) *kb.Document {
	tags := make([]string, 0, 2+len(item.Tags))
	tags = append(tags, tagMemory, bankTagPrefix+item.Bank)
	tags = append(tags, item.Tags...)

	related := make([]string, 0)
	if item.DocumentID != "" {
		related = append(related, docRefPrefix+item.DocumentID)
	}
	if len(item.Metadata) > 0 {
		keys := make([]string, 0, len(item.Metadata))
		for key := range item.Metadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			related = append(related, metaRefPrefix+key+"="+item.Metadata[key])
		}
	}

	return &kb.Document{
		ID:   item.ID,
		Path: filepath.ToSlash(filepath.Join(memoryDir, item.Bank, item.ID+".md")),
		Metadata: kb.Metadata{
			Title:       item.Title,
			Description: item.Context,
			Tags:        tags,
			Created:     item.Timestamp,
			Modified:    item.Timestamp,
			Related:     related,
			Author:      memoryAuthor,
			Version:     memoryVersion,
		},
		Content: item.Content,
	}
}

func itemFromDocument(doc *kb.Document, result kb.SearchResult) Item {
	if doc == nil {
		content := result.Excerpt
		if content == "" {
			content = result.Description
		}
		return Item{
			ID:        result.ID,
			Title:     result.Title,
			Content:   content,
			Context:   result.Description,
			Tags:      normalizeTags(result.Tags),
			Bank:      bankFromTags(result.Tags),
			Timestamp: result.Created,
		}
	}
	item := Item{
		ID:        doc.ID,
		Title:     doc.Metadata.Title,
		Content:   doc.Content,
		Context:   doc.Metadata.Description,
		Timestamp: doc.Metadata.Created,
	}
	if doc.Metadata.Author == memoryAuthor {
		item.Bank = bankFromTags(doc.Metadata.Tags)
		item.Tags = normalizeTags(doc.Metadata.Tags)
		item.DocumentID, item.Metadata = decodeRelated(doc.Metadata.Related)
		return item
	}
	item.Tags = append([]string(nil), doc.Metadata.Tags...)
	return item
}

func bankFromTags(tags []string) string {
	for _, tag := range tags {
		lower := strings.ToLower(tag)
		if strings.HasPrefix(lower, bankTagPrefix) {
			return strings.TrimPrefix(lower, bankTagPrefix)
		}
	}
	return ""
}

func decodeRelated(related []string) (string, map[string]string) {
	var docID string
	var meta map[string]string
	for _, entry := range related {
		switch {
		case strings.HasPrefix(entry, docRefPrefix):
			if docID == "" {
				docID = strings.TrimPrefix(entry, docRefPrefix)
			}
		case strings.HasPrefix(entry, metaRefPrefix):
			rest := strings.TrimPrefix(entry, metaRefPrefix)
			key, value, ok := strings.Cut(rest, "=")
			if !ok || key == "" {
				continue
			}
			if meta == nil {
				meta = make(map[string]string)
			}
			meta[key] = value
		}
	}
	return docID, meta
}
