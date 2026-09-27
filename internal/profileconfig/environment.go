package profileconfig

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

var environmentName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Environment resolves only explicitly requested fields. The caller must check
// the vault lock for each secret lookup. No values are interpolated into argv.
func Environment(fields map[string]Field, bindings, inherited []string, getSecret func(string) (string, error)) ([]string, error) {
	resolved := map[string]string{}
	for _, binding := range bindings {
		name, key, ok := strings.Cut(binding, "=")
		if !ok || !environmentName.MatchString(name) || strings.HasPrefix(name, "COCKPIT_") {
			return nil, errors.New("binding must be ENV_NAME=field; COCKPIT_* is reserved")
		}
		if _, exists := resolved[name]; exists {
			return nil, errors.New("duplicate environment binding")
		}
		field, exists := fields[key]
		if !exists {
			return nil, errors.New("binding refers to an unconfigured field")
		}
		var value string
		if field.Value != nil {
			value = *field.Value
		} else {
			var err error
			value, err = getSecret(field.SecretKey)
			if err != nil {
				return nil, errors.New("secret unavailable; verify vault unlock, access and configured key")
			}
		}
		if strings.ContainsRune(value, 0) {
			return nil, errors.New("environment value contains a NUL character")
		}
		resolved[name] = value
	}
	result := make([]string, 0, len(inherited)+len(resolved))
	for _, item := range inherited {
		name, _, _ := strings.Cut(item, "=")
		if _, replaced := resolved[strings.ToUpper(name)]; !replaced {
			result = append(result, item)
		}
	}
	names := make([]string, 0, len(resolved))
	for name := range resolved {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result = append(result, name+"="+resolved[name])
	}
	return result, nil
}
