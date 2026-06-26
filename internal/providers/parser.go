package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseCanonical reads the cockpitHomeDir and returns all canonical models.
func ParseCanonical(cockpitHomeDir string) (*CanonicalEntrypoint, []CanonicalSkill, []CanonicalRule, []CanonicalWorkflow, *CanonicalPermissions, []CanonicalAgent, error) {
	entrypoint, err := parseEntrypoint(cockpitHomeDir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	skills, err := parseSkills(cockpitHomeDir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	rules, err := parseRules(cockpitHomeDir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	workflows, err := parseWorkflows(cockpitHomeDir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	perms, err := parsePermissions(cockpitHomeDir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	agents, err := parseAgents(cockpitHomeDir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return entrypoint, skills, rules, workflows, perms, agents, nil
}

func parseEntrypoint(homeDir string) (*CanonicalEntrypoint, error) {
	entrypoint := &CanonicalEntrypoint{}

	cockpitMDPath := filepath.Join(homeDir, "COCKPIT.md")
	if data, err := os.ReadFile(cockpitMDPath); err == nil {
		entrypoint.ProjectContext = strings.TrimSpace(string(data))
	} else {
		identityPath := filepath.Join(homeDir, "identity.md")
		if data, err := os.ReadFile(identityPath); err == nil {
			entrypoint.ProjectContext = strings.TrimSpace(string(data))
		}
	}

	rulesDir := filepath.Join(homeDir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), "-gold-rules.md") {
				data, err := os.ReadFile(filepath.Join(rulesDir, entry.Name()))
				if err == nil {
					entrypoint.GoldenRules = append(entrypoint.GoldenRules, string(data))
				}
			}
		}
	}

	return entrypoint, nil
}

func parseSkills(homeDir string) ([]CanonicalSkill, error) {
	skillsDir := filepath.Join(homeDir, "skills")
	var skills []CanonicalSkill

	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // expects skill directories
		}
		skillName := entry.Name()
		skillPath := filepath.Join(skillsDir, skillName, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue // skip if SKILL.md doesn't exist
		}

		fm, body := StripFrontmatter(string(data))
		name := fm["name"]
		if name == "" {
			name = skillName
		}

		skill := CanonicalSkill{
			Name:        name,
			Description: fm["description"],
			Content:     body,
			ScriptFiles: make(map[string]string),
		}

		// Optionally load scripts
		scriptsDir := filepath.Join(skillsDir, skillName, "scripts")
		scriptEntries, err := os.ReadDir(scriptsDir)
		if err == nil {
			for _, se := range scriptEntries {
				if !se.IsDir() {
					if sd, err := os.ReadFile(filepath.Join(scriptsDir, se.Name())); err == nil {
						skill.ScriptFiles[filepath.Join("scripts", se.Name())] = string(sd)
					}
				}
			}
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

func parseRules(homeDir string) ([]CanonicalRule, error) {
	rulesDir := filepath.Join(homeDir, "rules")
	var rules []CanonicalRule

	entries, err := os.ReadDir(rulesDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		if strings.HasSuffix(entry.Name(), "-gold-rules.md") {
			continue // handled by entrypoint
		}
		data, err := os.ReadFile(filepath.Join(rulesDir, entry.Name()))
		if err != nil {
			continue
		}
		rules = append(rules, CanonicalRule{
			Name:    entry.Name(),
			Content: string(data),
		})
	}

	return rules, nil
}

func parseWorkflows(homeDir string) ([]CanonicalWorkflow, error) {
	wfDir := filepath.Join(homeDir, "workflows")
	var workflows []CanonicalWorkflow

	entries, err := os.ReadDir(wfDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(wfDir, entry.Name()))
		if err != nil {
			continue
		}

		var wf struct {
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Steps       []string `yaml:"steps"`
		}

		if err := yaml.Unmarshal(data, &wf); err == nil {
			if wf.Name == "" {
				wf.Name = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			}
			workflows = append(workflows, CanonicalWorkflow{
				Name:        wf.Name,
				Description: wf.Description,
				Steps:       wf.Steps,
			})
		}
	}

	return workflows, nil
}

func parsePermissions(homeDir string) (*CanonicalPermissions, error) {
	permsPath := filepath.Join(homeDir, "permissions.yaml")
	perms := &CanonicalPermissions{}

	data, err := os.ReadFile(permsPath)
	if os.IsNotExist(err) {
		return perms, nil
	}
	if err != nil {
		return nil, err
	}

	// Because lowercase field names might not match yaml tag exactly without tags,
	// let's use a struct with yaml tags to unmarshal safely
	var temp struct {
		AllowedCommands []string `yaml:"allowed_commands"`
		DeniedCommands  []string `yaml:"denied_commands"`
		AllowedDirs     []string `yaml:"allowed_dirs"`
		// Devin-specific permissions
		Allow []string `yaml:"allow"`
		Deny  []string `yaml:"deny"`
		Ask   []string `yaml:"ask"`
	}

	if err := yaml.Unmarshal(data, &temp); err != nil {
		return nil, err
	}

	perms.AllowedCommands = temp.AllowedCommands
	perms.DeniedCommands = temp.DeniedCommands
	perms.AllowedDirs = temp.AllowedDirs
	perms.Allow = temp.Allow
	perms.Deny = temp.Deny
	perms.Ask = temp.Ask

	return perms, nil
}

func parseAgents(homeDir string) ([]CanonicalAgent, error) {
	agentsDir := filepath.Join(homeDir, "agents")
	var agents []CanonicalAgent

	entries, err := os.ReadDir(agentsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // expects agent directories
		}
		agentName := entry.Name()
		agentPath := filepath.Join(agentsDir, agentName, "AGENT.md")
		data, err := os.ReadFile(agentPath)
		if err != nil {
			continue // skip if AGENT.md doesn't exist
		}

		fm, body := StripFrontmatter(string(data))
		name := fm["name"]
		if name == "" {
			name = agentName
		}

		// Parse allowed_tools from frontmatter
		var allowedTools []string
		if toolsStr, ok := fm["allowed_tools"]; ok {
			// This is a simplified parsing - in reality you'd want proper YAML parsing
			// For now, we'll store it as a string and let the compiler handle it
			allowedTools = []string{toolsStr}
		}

		agent := CanonicalAgent{
			Name:         name,
			Description:  fm["description"],
			Model:        fm["model"],
			AllowedTools: allowedTools,
			Content:      body,
		}

		// Parse max_nesting if present
		if maxNestingStr, ok := fm["max_nesting"]; ok {
			// Simple parsing - in production you'd want proper error handling
			var maxNesting int
			fmt.Sscanf(maxNestingStr, "%d", &maxNesting)
			agent.MaxNesting = maxNesting
		}

		agents = append(agents, agent)
	}

	return agents, nil
}
