package providers

import (
	"os"
	"path/filepath"
)

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

	return files, nil
}
