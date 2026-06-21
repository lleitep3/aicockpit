package providers

import (
	"fmt"
	"os"
	"path/filepath"
)

// antigravityCockpitPermissions lists the minimum permission grants that cockpit
// needs to operate inside Antigravity sessions.
var antigravityCockpitPermissions = []string{
	"command(cockpit)",
	"command(make)",
	"command(git)",
	"command(go)",
	"command(gh)",
	"read_file(~/.cockpit)",
}

// AntigravityAdapter compiles assets for the Antigravity provider.
type AntigravityAdapter struct{}

// NewAntigravityAdapter creates a new AntigravityAdapter.
func NewAntigravityAdapter() *AntigravityAdapter {
	return &AntigravityAdapter{}
}

// Name returns the provider name.
func (a *AntigravityAdapter) Name() string {
	return "antigravity"
}

// Compile walks through enabled features and maps files from ~/.cockpit/ to their target locations.
func (a *AntigravityAdapter) Compile(cockpitHomeDir string, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)

	features := []string{"skills", "rules", "hooks", "agents", "workflows"}

	for _, feat := range features {
		featConfig, exists := provider.Features[feat]
		if !exists || !featConfig.Enabled {
			continue
		}

		srcDir := filepath.Join(cockpitHomeDir, feat)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Prepends generated header for markdown files
			fileContent := string(content)
			if filepath.Ext(path) == ".md" {
				fileContent = AddGeneratedHeader(fileContent, "antigravity")
			}

			targetPath := filepath.Join(featConfig.Path, relPath)
			files[targetPath] = fileContent

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Handle permissions feature: merge cockpit grants into ~/.gemini/config/config.json
	permConfig, hasPerms := provider.Features["permissions"]
	if hasPerms && permConfig.Enabled {
		if err := a.applyPermissions(permConfig.Path); err != nil {
			return nil, fmt.Errorf("failed to apply antigravity permissions: %w", err)
		}
	}

	return files, nil
}

// applyPermissions reads the Antigravity global config.json, merges cockpit
// permission grants (without removing existing ones), and writes it back.
//
// Structure:
//
//	{
//	  "userSettings": {
//	    "globalPermissionGrants": {
//	      "allow": ["command(git)", ...]
//	    }
//	  }
//	}
func (a *AntigravityAdapter) applyPermissions(configPath string) error {
	expanded, err := expandHome(configPath)
	if err != nil {
		return err
	}

	m, err := readJSONFile(expanded)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", expanded, err)
	}

	// Navigate / create: m["userSettings"]["globalPermissionGrants"]
	userSettings := getNestedMap(m, "userSettings")
	grants := getNestedMap(userSettings, "globalPermissionGrants")

	existing := getStringSliceFromMap(grants, "allow")
	merged := mergeStringSlice(existing, antigravityCockpitPermissions)
	setStringSliceInMap(grants, "allow", merged)

	userSettings["globalPermissionGrants"] = grants
	m["userSettings"] = userSettings

	return writeJSONFile(expanded, m)
}
