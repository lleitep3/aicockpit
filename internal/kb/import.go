package kb

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ImportOptions selects the destination and the identity to update, if any.
type ImportOptions struct {
	Root     string
	Context  string
	UpdateID string
}

// ImportFile stores Markdown with a stable creation path. Updates are explicit
// so a reused temporary source filename cannot overwrite unrelated knowledge.
func ImportFile(source string, options ImportOptions) (*Document, error) {
	return importFileAt(source, options, time.Now())
}

func importFileAt(source string, options ImportOptions, now time.Time) (*Document, error) {
	doc, err := readImport(source)
	if err != nil {
		return nil, err
	}
	if options.Root == "" {
		return nil, fmt.Errorf("KB destination root is required")
	}
	if err := os.MkdirAll(options.Root, 0o755); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(options.Root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if options.UpdateID != "" {
		return updateImport(root, doc, options, now)
	}
	// Re-adding the managed file itself must retain its identity.
	absSource, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(options.Root)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(absRoot, absSource)
	if err == nil && filepath.IsLocal(rel) {
		options.UpdateID = GenerateDocumentID(filepath.Base(rel))
		return updateImport(root, doc, options, now)
	}
	if strings.TrimSpace(options.Context) != "" {
		doc.Metadata.Context = strings.TrimSpace(options.Context)
	}
	if doc.Metadata.Context == "" {
		return nil, fmt.Errorf("context is required: use --context <project-or-topic>")
	}
	doc.Metadata.Created, doc.Metadata.Modified = now, now
	return createImport(root, doc, now)
}

func readImport(source string) (*Document, error) {
	if strings.ToLower(filepath.Ext(source)) != ".md" {
		return nil, fmt.Errorf("source must be a Markdown (.md) file")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return nil, fmt.Errorf("source must be UTF-8 text")
	}
	id := GenerateDocumentID(filepath.Base(source))
	doc, err := ParseDocument(id, source, string(data))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.Content) == "" {
		return nil, fmt.Errorf("document body is empty")
	}
	fallback, err := ParseDocumentWithoutMetadata(id, source, doc.Content)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.Metadata.Title) == "" {
		doc.Metadata.Title = fallback.Metadata.Title
	}
	if doc.Metadata.Description == "" {
		doc.Metadata.Description = fallback.Metadata.Description
	}
	return doc, nil
}

func documentSlug(title string) string {
	var result strings.Builder
	separator := false
	for _, char := range strings.ToLower(title) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if result.Len() >= 120 {
				break
			}
			if separator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(char)
			separator = false
		} else {
			separator = true
		}
	}
	if result.Len() == 0 {
		return "document"
	}
	return result.String()
}

func createImport(root *os.Root, doc *Document, now time.Time) (*Document, error) {
	dir := now.Format("2006-01-02")
	if err := root.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	base := now.Format("2006-01-02_150405") + "-" + documentSlug(doc.Metadata.Title)
	content, err := SerializeDocument(doc)
	if err != nil {
		return nil, err
	}
	for suffix := 1; ; suffix++ {
		id := base
		if suffix > 1 {
			id = fmt.Sprintf("%s-%d", base, suffix)
		}
		path := filepath.Join(dir, id+".md")
		file, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		err = writeImport(file, content)
		if err != nil {
			_ = root.Remove(path)
			return nil, err
		}
		doc.ID, doc.Path = id, path
		return doc, nil
	}
}

func writeImport(file *os.File, content string) error {
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

func findImport(root *os.Root, id string) (string, error) {
	if id == "" || filepath.Base(id) != id || strings.ContainsAny(id, `/\\`) {
		return "", fmt.Errorf("invalid document ID")
	}
	var match string
	err := fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || entry.Name() != id+".md" {
			return nil
		}
		if match != "" {
			return fmt.Errorf("ambiguous document ID %q; select a more specific --root", id)
		}
		match = path
		return nil
	})
	if err != nil {
		return "", err
	}
	if match == "" {
		return "", fmt.Errorf("document ID %q not found in destination root", id)
	}
	return match, nil
}

func updateImport(root *os.Root, doc *Document, options ImportOptions, now time.Time) (*Document, error) {
	path, err := findImport(root, options.UpdateID)
	if err != nil {
		return nil, err
	}
	data, err := root.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Preserve existing metadata when the replacement supplies only a body.
	merged, err := mergeImportMetadata(string(data), doc, options.Context, now)
	if err != nil {
		return nil, err
	}
	doc.Metadata = merged
	if doc.Metadata.Created.IsZero() {
		doc.Metadata.Created = now
	}
	doc.ID, doc.Path = options.UpdateID, path
	content, err := SerializeDocument(doc)
	if err != nil {
		return nil, err
	}
	// A temporary sibling plus rename keeps the previous version on write errors.
	temp, err := os.CreateTemp(filepath.Join(options.Root, filepath.Dir(path)), ".kb-update-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return nil, err
	}
	if err := writeImport(temp, content); err != nil {
		return nil, err
	}
	tempRel, err := filepath.Rel(options.Root, temp.Name())
	if err != nil {
		return nil, err
	}
	if err := root.Rename(tempRel, path); err != nil {
		return nil, err
	}
	return doc, nil
}

func mergeImportMetadata(oldRaw string, doc *Document, context string, now time.Time) (Metadata, error) {
	old, _, err := extractMetadata(oldRaw)
	if err != nil {
		return Metadata{}, err
	}
	metadata := doc.Metadata
	metadata.Created, metadata.Modified = old.Created, now
	if len(metadata.Tags) == 0 {
		metadata.Tags = old.Tags
	}
	if len(metadata.Related) == 0 {
		metadata.Related = old.Related
	}
	if metadata.Author == "" {
		metadata.Author = old.Author
	}
	if metadata.Version == "" {
		metadata.Version = old.Version
	}
	if metadata.Context == "" {
		metadata.Context = old.Context
	}
	if strings.TrimSpace(context) != "" {
		metadata.Context = strings.TrimSpace(context)
	}
	if metadata.Context == "" {
		return Metadata{}, fmt.Errorf("context is required: use --context <project-or-topic>")
	}
	if metadata.Extra == nil {
		metadata.Extra = make(map[string]interface{})
	}
	for key, value := range old.Extra {
		if _, exists := metadata.Extra[key]; !exists {
			metadata.Extra[key] = value
		}
	}
	return metadata, nil
}
