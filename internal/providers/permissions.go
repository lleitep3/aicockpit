package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// expandHome expands a leading ~ to the user's home directory.
func expandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}

// mergeStringSlice merges additions into existing, returning a deduplicated slice.
func mergeStringSlice(existing, additions []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(additions))
	result := make([]string, 0, len(existing)+len(additions))
	for _, s := range existing {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	for _, s := range additions {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}

// readJSONFile reads a JSON file into a generic map.
// If the file does not exist it returns an empty map without error.
func readJSONFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]interface{}), nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// writeJSONFile writes a generic map to a JSON file (indented).
func writeJSONFile(path string, m map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// getNestedMap safely walks and creates nested map keys in a generic map.
func getNestedMap(m map[string]interface{}, keys ...string) map[string]interface{} {
	cur := m
	for _, k := range keys {
		v, ok := cur[k]
		if !ok {
			next := make(map[string]interface{})
			cur[k] = next
			cur = next
			continue
		}
		next, ok := v.(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			cur[k] = next
		}
		cur = next
	}
	return cur
}

// getStringSliceFromMap extracts a []string from a generic map key.
// Returns empty slice if key is absent or has unexpected type.
func getStringSliceFromMap(m map[string]interface{}, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// setStringSliceInMap stores a []string into a generic map key.
func setStringSliceInMap(m map[string]interface{}, key string, values []string) {
	iface := make([]interface{}, len(values))
	for i, v := range values {
		iface[i] = v
	}
	m[key] = iface
}
