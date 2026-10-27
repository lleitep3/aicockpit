// Package codexconfig manages the small portion of Codex configuration owned
// by AICockpit.
package codexconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	sandboxModeLine   = `sandbox_mode = "workspace-write"`
	sandboxTable      = "[sandbox_workspace_write]"
	writableRootsKey  = "writable_roots"
	defaultConfigMode = 0o644
)

// EnsureUserSandboxConfig grants Codex access to AICockpit's log directory.
// It returns the config path and whether the file changed.
func EnsureUserSandboxConfig() (string, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("failed to determine user home: %w", err)
	}

	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(homeDir, ".codex")
	} else if !filepath.IsAbs(codexHome) {
		codexHome, err = filepath.Abs(codexHome)
		if err != nil {
			return "", false, fmt.Errorf("failed to resolve CODEX_HOME: %w", err)
		}
	}

	configPath := filepath.Join(codexHome, "config.toml")
	logsPath := filepath.Join(homeDir, ".cockpit", "logs")
	changed, err := EnsureSandboxWritableRoot(configPath, logsPath)
	if err != nil {
		return configPath, false, err
	}
	return configPath, changed, nil
}

// EnsureSandboxWritableRoot adds writableRoot to Codex's workspace-write
// configuration without rewriting unrelated TOML or comments.
func EnsureSandboxWritableRoot(configPath, writableRoot string) (bool, error) {
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve Codex config path: %w", err)
	}
	writableRoot, err = filepath.Abs(writableRoot)
	if err != nil {
		return false, fmt.Errorf("failed to resolve writable root: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return false, fmt.Errorf("failed to create Codex config directory: %w", err)
	}

	data, mode, err := readConfig(configPath)
	if err != nil {
		return false, err
	}

	lines := splitLines(data)
	changed := false

	lines, modeChanged := ensureSandboxMode(lines)
	changed = changed || modeChanged

	lines, rootsChanged, err := ensureWritableRoot(lines, writableRoot)
	if err != nil {
		return false, err
	}
	changed = changed || rootsChanged

	if !changed {
		return false, nil
	}

	if err := writeConfigAtomically(configPath, []byte(strings.Join(lines, "")), mode); err != nil {
		return false, err
	}
	return true, nil
}

func readConfig(configPath string) ([]byte, os.FileMode, error) {
	info, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return nil, defaultConfigMode, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat Codex config: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read Codex config: %w", err)
	}
	return data, info.Mode().Perm(), nil
}

func splitLines(data []byte) []string {
	contents := strings.ReplaceAll(string(data), "\r\n", "\n")
	if contents == "" {
		return []string{}
	}
	return strings.SplitAfter(contents, "\n")
}

func ensureSandboxMode(lines []string) ([]string, bool) {
	firstTable := firstTableIndex(lines)
	for i := 0; i < firstTable; i++ {
		if assignmentKey(lines[i]) == "sandbox_mode" {
			ending := lineEnding(lines[i])
			if strings.TrimSuffix(lineBody(lines[i]), "\r") == sandboxModeLine {
				return lines, false
			}
			lines[i] = sandboxModeLine + ending
			return lines, true
		}
	}

	prefix := []string{sandboxModeLine + "\n"}
	if len(lines) > 0 && strings.TrimSpace(lineBody(lines[0])) != "" {
		prefix = append(prefix, "\n")
	}
	return append(prefix, lines...), true
}

func ensureWritableRoot(lines []string, writableRoot string) ([]string, bool, error) {
	rootLiteral := strconv.Quote(writableRoot)
	tableStart := findTable(lines, sandboxTable)
	if tableStart < 0 {
		if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
			lines[len(lines)-1] += "\n"
		}
		if len(lines) > 0 && strings.TrimSpace(lineBody(lines[len(lines)-1])) != "" {
			lines = append(lines, "\n")
		}
		lines = append(lines, sandboxTable+"\n", writableRootsKey+" = ["+rootLiteral+"]\n")
		return lines, true, nil
	}

	tableEnd := nextTableIndex(lines, tableStart+1)
	keyStart := findAssignment(lines, tableStart+1, tableEnd, writableRootsKey)
	if keyStart < 0 {
		insertAt := tableStart + 1
		lines = insertLine(lines, insertAt, writableRootsKey+" = ["+rootLiteral+"]\n")
		return lines, true, nil
	}

	keyEnd, err := arrayEnd(lines, keyStart, tableEnd)
	if err != nil {
		return nil, false, err
	}
	for _, line := range lines[keyStart : keyEnd+1] {
		if strings.Contains(line, rootLiteral) {
			return lines, false, nil
		}
	}

	if keyStart == keyEnd {
		body := lineBody(lines[keyStart])
		closeIndex := strings.LastIndex(body, "]")
		openIndex := strings.Index(body, "[")
		if openIndex < 0 || closeIndex < openIndex {
			return nil, false, fmt.Errorf("invalid %s array in Codex config", writableRootsKey)
		}
		values := strings.TrimSpace(body[openIndex+1 : closeIndex])
		separator := ""
		if values != "" {
			separator = ", "
		}
		lines[keyStart] = body[:closeIndex] + separator + rootLiteral + body[closeIndex:] + lineEnding(lines[keyStart])
		return lines, true, nil
	}

	previous := keyEnd - 1
	for previous > keyStart && strings.TrimSpace(lineBody(lines[previous])) == "" {
		previous--
	}
	if previous > keyStart {
		lines[previous] = addTrailingComma(lines[previous])
	}
	indent := leadingWhitespace(lineBody(lines[keyStart])) + "    "
	lines = insertLine(lines, keyEnd, indent+rootLiteral+",\n")
	return lines, true, nil
}

func findTable(lines []string, table string) int {
	for i, line := range lines {
		if strings.TrimSpace(lineBody(line)) == table {
			return i
		}
	}
	return -1
}

func firstTableIndex(lines []string) int {
	return nextTableIndex(lines, 0)
}

func nextTableIndex(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lineBody(lines[i]))
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			return i
		}
	}
	return len(lines)
}

func findAssignment(lines []string, start, end int, key string) int {
	for i := start; i < end && i < len(lines); i++ {
		if assignmentKey(lines[i]) == key {
			return i
		}
	}
	return -1
}

func assignmentKey(line string) string {
	trimmed := strings.TrimSpace(lineBody(line))
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	separator := strings.IndexByte(trimmed, '=')
	if separator < 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[:separator])
}

func arrayEnd(lines []string, start, limit int) (int, error) {
	balance := 0
	started := false
	for i := start; i < limit && i < len(lines); i++ {
		body := lineBody(lines[i])
		open := strings.Index(body, "[")
		if open >= 0 {
			started = true
		}
		if started {
			balance += strings.Count(body, "[")
			balance -= strings.Count(body, "]")
			if balance <= 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated %s array in Codex config", writableRootsKey)
}

func insertLine(lines []string, index int, line string) []string {
	lines = append(lines, "")
	copy(lines[index+1:], lines[index:])
	lines[index] = line
	return lines
}

func addTrailingComma(line string) string {
	ending := lineEnding(line)
	body := strings.TrimSuffix(lineBody(line), "\r")
	commentIndex := strings.Index(body, "#")
	content, comment := body, ""
	if commentIndex >= 0 {
		content, comment = body[:commentIndex], body[commentIndex:]
	}
	trimmed := strings.TrimRight(content, " \t")
	if strings.HasSuffix(trimmed, ",") || strings.HasSuffix(trimmed, "[") {
		return body + ending
	}
	return trimmed + "," + content[len(trimmed):] + comment + ending
}

func lineBody(line string) string {
	return strings.TrimSuffix(line, "\n")
}

func lineEnding(line string) string {
	if strings.HasSuffix(line, "\n") {
		return "\n"
	}
	return ""
}

func leadingWhitespace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func writeConfigAtomically(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config.toml.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary Codex config: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to preserve Codex config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temporary Codex config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync temporary Codex config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary Codex config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace Codex config: %w", err)
	}
	return nil
}
