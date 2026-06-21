package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GooseAdapter compiles assets for the Goose provider.
type GooseAdapter struct{}

// NewGooseAdapter creates a new GooseAdapter.
func NewGooseAdapter() *GooseAdapter {
	return &GooseAdapter{}
}

// Name returns the provider name.
func (g *GooseAdapter) Name() string {
	return "goose"
}

// Compile compiles .goosehints and skills.
func (g *GooseAdapter) Compile(cockpitHomeDir string, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)

	// 1. Build .goosehints (rules feature)
	rulesConfig, hasRules := provider.Features["rules"]
	if hasRules && rulesConfig.Enabled {
		goosehints, err := buildGoosehints(cockpitHomeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build .goosehints: %w", err)
		}
		files[rulesConfig.Path] = goosehints
	}

	// 2. Build `.agents/skills/` files (skills feature)
	skillsConfig, hasSkills := provider.Features["skills"]
	if hasSkills && skillsConfig.Enabled {
		skillsSrcDir := filepath.Join(cockpitHomeDir, "skills")
		if _, err := os.Stat(skillsSrcDir); err == nil {
			err = filepath.Walk(skillsSrcDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(skillsSrcDir, path)
				if err != nil {
					return err
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				targetPath := filepath.Join(skillsConfig.Path, relPath)
				files[targetPath] = string(content)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk skills: %w", err)
			}
		}
	}

	return files, nil
}

func buildGoosehints(cockpitHomeDir string) (string, error) {
	identityPath := filepath.Join(cockpitHomeDir, "identity.md")
	identityContent := ""
	if _, err := os.Stat(identityPath); err == nil {
		data, err := os.ReadFile(identityPath)
		if err != nil {
			return "", err
		}
		identityContent = string(data)
	}

	rulesDir := filepath.Join(cockpitHomeDir, "rules")
	var rules []string
	if _, err := os.Stat(rulesDir); err == nil {
		entries, err := os.ReadDir(rulesDir)
		if err != nil {
			return "", err
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
				data, err := os.ReadFile(filepath.Join(rulesDir, entry.Name()))
				if err != nil {
					return "", err
				}
				rules = append(rules, string(data))
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(AddGeneratedHeader("", ".goosehints"))

	if identityContent != "" {
		builder.WriteString(identityContent)
		builder.WriteString("\n\n---\n\n")
	}

	builder.WriteString(strings.Join(rules, "\n\n---\n\n"))
	return builder.String(), nil
}
