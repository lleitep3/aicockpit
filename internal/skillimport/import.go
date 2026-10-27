package skillimport

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Item describes the action taken, or planned, for one skill.
type Item struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Report summarizes a local skill import without exposing skill contents.
type Report struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	ReviewTarget string `json:"review_target,omitempty"`
	DryRun       bool   `json:"dry_run"`
	Items        []Item `json:"items"`
}

type skill struct {
	name      string
	directory string
	path      string
	files     map[string]file
}

type file struct {
	content []byte
	mode    fs.FileMode
}

// ImportLocal imports skills from a skill directory or a directory containing
// skill directories. Existing skills are never overwritten.
func ImportLocal(sourcePath, targetRoot string, dryRun bool) (Report, error) {
	return ImportLocalWithConflictArchive(sourcePath, targetRoot, "", dryRun)
}

// ImportLocalWithConflictArchive imports local skills and optionally copies
// conflicting skills to a new review directory outside the Cockpit library.
func ImportLocalWithConflictArchive(sourcePath, targetRoot, reviewRoot string, dryRun bool) (Report, error) {
	source, err := filepath.Abs(sourcePath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve skill source: %w", err)
	}
	resolvedSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return Report{}, fmt.Errorf("resolve skill source %s: %w", source, err)
	}
	source = resolvedSource
	target, err := filepath.Abs(targetRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve Cockpit skill directory: %w", err)
	}
	if reviewRoot != "" {
		reviewRoot, err = filepath.Abs(reviewRoot)
		if err != nil {
			return Report{}, fmt.Errorf("resolve skill review directory: %w", err)
		}
		if pathsOverlap(reviewRoot, target) || pathsOverlap(reviewRoot, source) {
			return Report{}, fmt.Errorf("skill review directory must be separate from source and Cockpit skill directories")
		}
		if _, err := os.Lstat(reviewRoot); err == nil {
			return Report{}, fmt.Errorf("skill review directory already exists: %s", reviewRoot)
		} else if !errors.Is(err, os.ErrNotExist) {
			return Report{}, fmt.Errorf("inspect skill review directory %s: %w", reviewRoot, err)
		}
	}

	candidates, err := discoverSkills(source)
	if err != nil {
		return Report{}, err
	}
	if len(candidates) == 0 {
		return Report{}, fmt.Errorf("no skills found in %s; expected SKILL.md files in the directory or its immediate subdirectories", source)
	}

	if err := checkDuplicateCandidates(candidates); err != nil {
		return Report{}, err
	}

	existingByName, err := discoverExistingSkills(target)
	if err != nil {
		return Report{}, err
	}

	report := Report{Source: source, Target: target, ReviewTarget: reviewRoot, DryRun: dryRun}
	reviewRootCreated := false
	for _, candidate := range candidates {
		destination := filepath.Join(target, candidate.directory)
		if existing, ok := existingByName[normalizeName(candidate.name)]; ok {
			if same, err := sameSkill(candidate, existing); err != nil {
				return Report{}, err
			} else if same {
				report.Items = append(report.Items, Item{Name: candidate.name, Status: "unchanged"})
			} else {
				item, err := conflictItem(candidate, fmt.Sprintf("a different skill with this name already exists at %s", existing), reviewRoot, reviewRootCreated, dryRun)
				if err != nil {
					return report, err
				}
				if item.Status == "saved-conflict" {
					reviewRootCreated = true
				}
				report.Items = append(report.Items, item)
			}
			continue
		}

		if info, err := os.Lstat(destination); err == nil {
			reason := "a different skill already occupies the target directory"
			if info.Mode()&os.ModeSymlink != 0 {
				reason = "the target directory is a symbolic link"
			}
			item, err := conflictItem(candidate, reason, reviewRoot, reviewRootCreated, dryRun)
			if err != nil {
				return report, err
			}
			if item.Status == "saved-conflict" {
				reviewRootCreated = true
			}
			report.Items = append(report.Items, item)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return Report{}, fmt.Errorf("inspect target for skill %q: %w", candidate.name, err)
		}

		status := "imported"
		if dryRun {
			status = "would-import"
		} else {
			if err := copySkill(candidate, destination, target); err != nil {
				return report, fmt.Errorf("import skill %q: %w", candidate.name, err)
			}
			existingByName[normalizeName(candidate.name)] = destination
		}
		report.Items = append(report.Items, Item{Name: candidate.name, Status: status})
	}

	return report, nil
}

func conflictItem(candidate skill, reason, reviewRoot string, reviewRootCreated, dryRun bool) (Item, error) {
	item := Item{Name: candidate.name, Status: "conflict", Reason: reason}
	if reviewRoot == "" {
		return item, nil
	}
	if dryRun {
		item.Status = "would-save-conflict"
		return item, nil
	}
	if !reviewRootCreated {
		if err := createReviewRoot(reviewRoot); err != nil {
			return Item{}, err
		}
	}
	destination := filepath.Join(reviewRoot, candidate.directory)
	if err := copySkill(candidate, destination, reviewRoot); err != nil {
		return Item{}, fmt.Errorf("save conflicting skill %q for review: %w", candidate.name, err)
	}
	item.Status = "saved-conflict"
	return item, nil
}

func createReviewRoot(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create skill review parent directory: %w", err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("create skill review directory %s: %w", path, err)
	}
	return nil
}

func pathsOverlap(first, second string) bool {
	return pathContains(first, second) || pathContains(second, first)
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func discoverSkills(source string) ([]skill, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("open skill source %s: %w", source, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skill source must be a directory: %s", source)
	}

	if _, err := os.Stat(filepath.Join(source, "SKILL.md")); err == nil {
		candidate, err := loadSkill(source)
		if err != nil {
			return nil, err
		}
		return []skill{candidate}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect skill source %s: %w", source, err)
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, fmt.Errorf("read skill source %s: %w", source, err)
	}

	candidates := make([]skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		skillPath := filepath.Join(source, entry.Name())
		if _, err := os.Stat(filepath.Join(skillPath, "SKILL.md")); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect skill directory %s: %w", skillPath, err)
		}
		candidate, err := loadSkill(skillPath)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].name == candidates[j].name {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].name < candidates[j].name
	})
	return candidates, nil
}

func loadSkill(skillPath string) (skill, error) {
	metadataPath := filepath.Join(skillPath, "SKILL.md")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return skill{}, fmt.Errorf("read skill metadata %s: %w", metadataPath, err)
	}
	name, err := skillName(string(metadataBytes))
	if err != nil {
		return skill{}, fmt.Errorf("invalid skill %s: %w", metadataPath, err)
	}
	base := filepath.Base(skillPath)
	if base == "." || base == ".." || strings.ContainsAny(base, `/\\`) {
		return skill{}, fmt.Errorf("invalid skill directory %q", base)
	}

	loaded := skill{name: name, directory: base, path: skillPath, files: make(map[string]file)}
	err = filepath.WalkDir(skillPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == skillPath {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not supported in skills: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect skill file %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type in skill: %s", path)
		}
		rel, err := filepath.Rel(skillPath, path)
		if err != nil {
			return fmt.Errorf("resolve skill file path %s: %w", path, err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read skill file %s: %w", path, err)
		}
		loaded.files[filepath.ToSlash(rel)] = file{content: content, mode: info.Mode().Perm()}
		return nil
	})
	if err != nil {
		return skill{}, fmt.Errorf("read skill directory %s: %w", skillPath, err)
	}
	return loaded, nil
}

func skillName(document string) (string, error) {
	lines := strings.Split(document, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", fmt.Errorf("SKILL.md must start with YAML frontmatter")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return "", fmt.Errorf("SKILL.md has no closing frontmatter delimiter")
	}

	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &metadata); err != nil {
		return "", fmt.Errorf("parse YAML frontmatter: %w", err)
	}
	if strings.TrimSpace(metadata.Name) == "" {
		return "", fmt.Errorf("SKILL.md frontmatter requires a non-empty name")
	}
	if len([]rune(metadata.Name)) > 64 {
		return "", fmt.Errorf("skill name exceeds 64 characters")
	}
	nameRunes := []rune(metadata.Name)
	if len(nameRunes) == 0 || !unicode.IsLetter(nameRunes[0]) && !unicode.IsDigit(nameRunes[0]) {
		return "", fmt.Errorf("skill name must begin with a letter or number")
	}
	for _, char := range nameRunes {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && !strings.ContainsRune("-_.: ", char) {
			return "", fmt.Errorf("skill name contains a character that is not allowed")
		}
	}
	if strings.TrimSpace(metadata.Name) != metadata.Name {
		return "", fmt.Errorf("skill name cannot start or end with whitespace")
	}
	if strings.TrimSpace(metadata.Description) == "" {
		return "", fmt.Errorf("SKILL.md frontmatter requires a non-empty description")
	}
	if len([]rune(metadata.Description)) > 1024 {
		return "", fmt.Errorf("skill description exceeds 1024 characters")
	}
	return metadata.Name, nil
}

func discoverExistingSkills(target string) (map[string]string, error) {
	entries, err := os.ReadDir(target)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Cockpit skill directory %s: %w", target, err)
	}

	existing := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		skillPath := filepath.Join(target, entry.Name())
		candidate, err := loadSkill(skillPath)
		if err != nil {
			continue
		}
		key := normalizeName(candidate.name)
		if _, duplicate := existing[key]; !duplicate {
			existing[key] = skillPath
		}
	}
	return existing, nil
}

func sameSkill(candidate skill, existingPath string) (bool, error) {
	existing, err := loadSkill(existingPath)
	if err != nil {
		return false, fmt.Errorf("inspect existing skill %s: %w", existingPath, err)
	}
	if len(candidate.files) != len(existing.files) {
		return false, nil
	}
	for path, candidateFile := range candidate.files {
		existingFile, ok := existing.files[path]
		if !ok || candidateFile.mode != existingFile.mode {
			return false, nil
		}
		candidateContent := candidateFile.content
		existingContent := existingFile.content
		if path == "SKILL.md" {
			candidateContent, err = normalizeSkillDocument(candidateContent)
			if err != nil {
				return false, fmt.Errorf("normalize incoming SKILL.md: %w", err)
			}
			existingContent, err = normalizeSkillDocument(existingContent)
			if err != nil {
				return false, fmt.Errorf("normalize existing SKILL.md: %w", err)
			}
		}
		if !bytes.Equal(candidateContent, existingContent) {
			return false, nil
		}
	}
	return true, nil
}

func normalizeSkillDocument(document []byte) ([]byte, error) {
	frontmatter, body, ok := splitSkillFrontmatter(string(document))
	if !ok {
		return document, nil
	}
	var metadata map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, fmt.Errorf("parse YAML frontmatter: %w", err)
	}
	canonicalFrontmatter, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode YAML frontmatter: %w", err)
	}
	body = stripGeneratedHeader(body)
	var normalized bytes.Buffer
	normalized.WriteString("---\n")
	normalized.Write(canonicalFrontmatter)
	normalized.WriteString("---\n")
	normalized.WriteString(body)
	return normalized.Bytes(), nil
}

func splitSkillFrontmatter(document string) (string, string, bool) {
	lines := strings.Split(document, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", document, false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), true
		}
	}
	return "", document, false
}

func stripGeneratedHeader(body string) string {
	lines := strings.SplitN(body, "\n", 3)
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "<!-- GENERATED by AICockpit ") || !strings.HasPrefix(lines[1], "<!-- GENERATED:") {
		return body
	}
	return strings.TrimPrefix(lines[2], "\n")
}

func copySkill(candidate skill, destination, targetRoot string) error {
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return fmt.Errorf("create skill import directory: %w", err)
	}
	temporary, err := os.MkdirTemp(targetRoot, ".skill-import-")
	if err != nil {
		return fmt.Errorf("create import staging directory: %w", err)
	}
	defer os.RemoveAll(temporary) // Best-effort cleanup of this command's staging files.

	paths := make([]string, 0, len(candidate.files))
	for rel := range candidate.files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		clean := filepath.Clean(filepath.FromSlash(rel))
		if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe skill file path %q", rel)
		}
		entry := candidate.files[rel]
		outPath := filepath.Join(temporary, clean)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("create imported skill resource directory: %w", err)
		}
		mode := entry.mode
		if err := os.WriteFile(outPath, entry.content, mode); err != nil {
			return fmt.Errorf("write imported skill file %s: %w", rel, err)
		}
		if err := os.Chmod(outPath, mode); err != nil {
			return fmt.Errorf("preserve permissions for imported skill file %s: %w", rel, err)
		}
	}
	if err := os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("install imported skill directory: %w", err)
	}
	return nil
}

func checkDuplicateCandidates(candidates []skill) error {
	seen := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		key := normalizeName(candidate.name)
		if previous, exists := seen[key]; exists {
			return fmt.Errorf("source contains duplicate skill name %q in %s and %s", candidate.name, previous, candidate.path)
		}
		seen[key] = candidate.path
	}
	return nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
