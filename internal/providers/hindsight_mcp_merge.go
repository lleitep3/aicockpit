package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func mergeMCPDocument(path, existing, generated string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return mergeJSONMCP(existing, generated)
	case ".toml":
		return mergeTOMLMCP(existing, generated), nil
	default:
		return "", fmt.Errorf("hindsight mcp: unsupported config %q", path)
	}
}

func mergeJSONMCP(existing, generated string) (string, error) {
	if strings.TrimSpace(existing) == "" {
		return generated, nil
	}
	if existing == generated {
		return existing, nil
	}
	var have map[string]any
	if err := json.Unmarshal([]byte(existing), &have); err != nil {
		return "", fmt.Errorf("hindsight mcp: existing config: %w", err)
	}
	if have == nil {
		have = map[string]any{}
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(generated), &want); err != nil {
		return "", fmt.Errorf("hindsight mcp: generated config: %w", err)
	}
	haveServers, err := mcpServerObject(have["mcpServers"])
	if err != nil {
		return "", err
	}
	wantServers, err := mcpServerObject(want["mcpServers"])
	if err != nil {
		return "", err
	}
	for _, name := range []string{hindsightProjectServer, hindsightUserServer} {
		delete(haveServers, name)
	}
	for name, cfg := range wantServers {
		haveServers[name] = cfg
	}
	have["mcpServers"] = haveServers
	merged, err := encodeJSON(have)
	if err != nil {
		return "", err
	}
	same, err := jsonSemanticallyEqual(existing, merged)
	if err != nil {
		return "", err
	}
	if same {
		return existing, nil
	}
	return merged, nil
}

func mcpServerObject(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hindsight mcp: mcpServers must be an object")
	}
	return object, nil
}

func encodeJSON(value any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return "", fmt.Errorf("hindsight mcp: encode config: %w", err)
	}
	return buf.String(), nil
}

func jsonSemanticallyEqual(left, right string) (bool, error) {
	canon := func(raw string) (string, error) {
		var value any
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return "", fmt.Errorf("hindsight mcp: compare config: %w", err)
		}
		return encodeJSON(value)
	}
	leftCanon, err := canon(left)
	if err != nil {
		return false, err
	}
	rightCanon, err := canon(right)
	if err != nil {
		return false, err
	}
	return leftCanon == rightCanon, nil
}

func mergeTOMLMCP(existing, generated string) string {
	if strings.TrimSpace(existing) == "" || existing == generated {
		return generated
	}
	remainder := strings.TrimRight(stripOwnedMCPTables(existing), "\n")
	if strings.TrimSpace(remainder) == "" {
		return generated
	}
	if !strings.HasSuffix(generated, "\n") {
		generated += "\n"
	}
	return remainder + "\n\n" + generated
}

func stripOwnedMCPTables(content string) string {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	skipping := false
	for _, line := range lines {
		if isTableHeader(line) {
			skipping = isOwnedMCPTable(line)
		}
		if skipping {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func isTableHeader(line string) bool {
	body := strings.TrimSpace(line)
	return strings.HasPrefix(body, "[") && strings.HasSuffix(body, "]")
}

func isOwnedMCPTable(line string) bool {
	body := strings.TrimSpace(line)
	if !strings.HasPrefix(body, "[") || !strings.HasSuffix(body, "]") || strings.HasPrefix(body, "[[") {
		return false
	}
	inner := strings.TrimSpace(body[1 : len(body)-1])
	for _, name := range []string{hindsightProjectServer, hindsightUserServer} {
		base := "mcp_servers." + strconv.Quote(name)
		if inner == base || strings.HasPrefix(inner, base+".") {
			return true
		}
	}
	return false
}
