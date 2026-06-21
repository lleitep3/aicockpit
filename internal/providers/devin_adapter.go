package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DevinAdapter compiles assets for the Devin provider.
type DevinAdapter struct{}

// NewDevinAdapter creates a new DevinAdapter.
func NewDevinAdapter() *DevinAdapter {
	return &DevinAdapter{}
}

// Name returns the provider name.
func (d *DevinAdapter) Name() string {
	return "devin"
}

// Compile compiles AGENTS.md, custom skills, and config.yaml.
func (d *DevinAdapter) Compile(cockpitHomeDir string, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)

	// 1. Build AGENTS.md (rules feature)
	rulesConfig, hasRules := provider.Features["rules"]
	if hasRules && rulesConfig.Enabled {
		agentsMd, err := buildAgentsMd(cockpitHomeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build AGENTS.md: %w", err)
		}
		files[rulesConfig.Path] = agentsMd
	}

	// 2. Build `.agents/skills/` files (skills feature)
	skillsConfig, hasSkills := provider.Features["skills"]
	var skillDirs []string
	if hasSkills && skillsConfig.Enabled {
		skillsSrcDir := filepath.Join(cockpitHomeDir, "skills")
		if _, err := os.Stat(skillsSrcDir); err == nil {
			err = filepath.Walk(skillsSrcDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					if path != skillsSrcDir {
						rel, _ := filepath.Rel(skillsSrcDir, path)
						// Keep track of top-level skill directories
						if !strings.Contains(rel, string(filepath.Separator)) {
							skillDirs = append(skillDirs, rel)
						}
					}
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

	// 3. Build `.devin/config.yaml` (workflows feature)
	wfConfig, hasWorkflows := provider.Features["workflows"]
	if hasWorkflows && wfConfig.Enabled {
		configYaml := buildDevinConfigYaml(skillDirs, skillsConfig.Path)
		files[wfConfig.Path] = configYaml
	}

	return files, nil
}

func buildAgentsMd(cockpitHomeDir string) (string, error) {
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
	builder.WriteString(AddGeneratedHeader("", "devin"))

	if identityContent != "" {
		builder.WriteString(identityContent)
		builder.WriteString("\n\n---\n\n")
	}

	builder.WriteString(strings.Join(rules, "\n\n---\n\n"))
	return builder.String(), nil
}

func buildDevinConfigYaml(skillDirs []string, skillsPath string) string {
	var skillTools strings.Builder
	sort.Strings(skillDirs)
	for _, skillDir := range skillDirs {
		name := strings.ReplaceAll(skillDir, "-", "_")
		//nolint:staticcheck // strings.Title is deprecated but acceptable for simple text transforms
		readable := strings.Title(strings.ReplaceAll(skillDir, "-", " "))
		skillTools.WriteString(fmt.Sprintf(`  - name: %s
    command: "cat %s"
    description: "Follow the %s skill guide for this task."

`, name, filepath.Join(skillsPath, skillDir, "SKILL.md"), readable))
	}

	content := fmt.Sprintf(`tools:
  - name: run_tests
    command: "make test"
    description: "Run the full test suite and return results."

  - name: lint_check
    command: "make lint"
    description: "Run all linters and check formatting."

%s`, skillTools.String())

	return AddGeneratedHeader(content, "config.yaml")
}
