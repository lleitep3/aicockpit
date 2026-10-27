package providers

import (
	"bytes"
	"fmt"
	"text/template"
)

// GenericYAMLCompiler implements Compiler by rendering artifacts from an
// AdapterManifest's template declarations.
type GenericYAMLCompiler struct {
	manifest *AdapterManifest
}

// NewGenericYAMLCompiler creates a compiler backed by the given manifest.
func NewGenericYAMLCompiler(manifest *AdapterManifest) *GenericYAMLCompiler {
	return &GenericYAMLCompiler{manifest: manifest}
}

// Name returns the provider name from the manifest.
func (c *GenericYAMLCompiler) Name() string {
	return c.manifest.Name
}

// render executes a Go text/template with the given data, returning the result.
func (c *GenericYAMLCompiler) render(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// buildOutput returns a (map[string]string, error) where key=path, value=rendered content.
// If the artifact is disabled or has no path, returns empty map.
func (c *GenericYAMLCompiler) buildOutput(artifactKey string, data interface{}) (map[string]string, error) {
	cfg, ok := c.manifest.Artifacts[artifactKey]
	if !ok || !cfg.Enabled || cfg.Path == "" {
		return map[string]string{}, nil
	}
	content, err := c.render(cfg.Template, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render %s template: %w", artifactKey, err)
	}
	return map[string]string{cfg.Path: content}, nil
}

// CompileEntrypoint renders the entrypoint template.
// Template data: struct{ GoldenRules []string; ProjectContext string }
func (c *GenericYAMLCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	return c.buildOutput("entrypoint", entrypoint)
}

// CompileSkills renders the skills template.
// Template data: []CanonicalSkill
func (c *GenericYAMLCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	if len(skills) == 0 {
		return map[string]string{}, nil
	}
	return c.buildOutput("skills", skills)
}

// CompileRules renders the rules template.
// Template data: []CanonicalRule (each has Name, Content fields)
func (c *GenericYAMLCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
	if len(rules) == 0 {
		return map[string]string{}, nil
	}
	return c.buildOutput("rules", rules)
}

// CompileWorkflows renders the workflows template.
// Template data: []CanonicalWorkflow
func (c *GenericYAMLCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	if len(workflows) == 0 {
		return map[string]string{}, nil
	}
	return c.buildOutput("workflows", workflows)
}

// CompilePermissions renders the permissions template.
// Template data: *CanonicalPermissions
func (c *GenericYAMLCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	return c.buildOutput("permissions", perms)
}

// CompileAgents renders the agents template.
// Template data: []CanonicalAgent
func (c *GenericYAMLCompiler) CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error) {
	if len(agents) == 0 {
		return map[string]string{}, nil
	}
	return c.buildOutput("agents", agents)
}
