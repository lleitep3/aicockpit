package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AssetSyncer syncs and removes package assets to/from canonical cockpit dirs.
type AssetSyncer struct {
	cockpitDir string
}

// NewAssetSyncer creates a new AssetSyncer.
func NewAssetSyncer(cockpitDir string) *AssetSyncer {
	return &AssetSyncer{cockpitDir: cockpitDir}
}

// Sync copies a package's assets (skills, rules, agents, workflows, KB) into the
// cockpit canonical directories so they are available for provider compilation.
func (a *AssetSyncer) Sync(pkg *Package, installPath string) error {
	type assetGroup struct {
		features []Feature
		dir      string
	}

	groups := []assetGroup{
		{features: pkg.Features.Skills, dir: "skills"},
		{features: pkg.Features.Rules, dir: "rules"},
		{features: pkg.Features.Agents, dir: "agents"},
		{features: pkg.Features.Workflows, dir: "workflows"},
	}

	for _, group := range groups {
		for _, f := range group.features {
			src := filepath.Join(installPath, f.Path)
			dst := filepath.Join(a.cockpitDir, group.dir, f.Name)

			info, err := os.Stat(src)
			if os.IsNotExist(err) {
				fmt.Printf("  ⚠ Asset not found, skipping: %s\n", f.Path)
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to stat asset %s: %w", f.Path, err)
			}

			if info.IsDir() {
				if err := os.MkdirAll(dst, 0o755); err != nil {
					return fmt.Errorf("failed to create asset dir %s: %w", dst, err)
				}
				if err := copyDir(src, dst); err != nil {
					return fmt.Errorf("failed to sync asset %s/%s: %w", group.dir, f.Name, err)
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return fmt.Errorf("failed to create asset parent dir %s: %w", filepath.Dir(dst), err)
				}
				if err := copyFile(src, dst); err != nil {
					return fmt.Errorf("failed to sync asset %s/%s: %w", group.dir, f.Name, err)
				}
			}

			fmt.Printf("  ✓ %s/%s synced to canonical dir\n", group.dir, f.Name)
		}
	}

	// Sync KB features
	for _, kb := range pkg.Features.KB {
		src := filepath.Join(installPath, kb.Path)
		dst := filepath.Join(a.cockpitDir, "kb", "packages", pkg.Name, filepath.Base(kb.Path))

		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("  ⚠ KB Asset not found, skipping: %s\n", kb.Path)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create kb asset dir %s: %w", filepath.Dir(dst), err)
		}

		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to stat kb asset %s: %w", src, err)
		}
		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("failed to sync kb asset %s: %w", kb.Path, err)
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("failed to sync kb asset %s: %w", kb.Path, err)
			}
		}

		fmt.Printf("  ✓ kb/packages/%s/%s synced to canonical dir\n", pkg.Name, filepath.Base(kb.Path))
	}

	// Inject gold_rules into ~/.cockpit/COCKPIT.md
	if len(pkg.Features.GoldRules) > 0 {
		if err := a.injectGoldRules(pkg); err != nil {
			return err
		}
	}

	return nil
}

// Remove removes a package's assets from the cockpit canonical directories.
func (a *AssetSyncer) Remove(pkg *Package) error {
	type assetGroup struct {
		features []Feature
		dir      string
	}

	groups := []assetGroup{
		{features: pkg.Features.Skills, dir: "skills"},
		{features: pkg.Features.Rules, dir: "rules"},
		{features: pkg.Features.Agents, dir: "agents"},
		{features: pkg.Features.Workflows, dir: "workflows"},
	}

	for _, group := range groups {
		for _, f := range group.features {
			dst := filepath.Join(a.cockpitDir, group.dir, f.Name)

			if _, err := os.Stat(dst); os.IsNotExist(err) {
				continue
			}

			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("failed to remove asset %s/%s: %w", group.dir, f.Name, err)
			}

			fmt.Printf("  ✓ %s/%s removed from canonical dir\n", group.dir, f.Name)
		}
	}

	// Remove KB features directory for this package
	if len(pkg.Features.KB) > 0 {
		dst := filepath.Join(a.cockpitDir, "kb", "packages", pkg.Name)
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("failed to remove kb package dir %s: %w", dst, err)
			}
			fmt.Printf("  ✓ kb/packages/%s removed from canonical dir\n", pkg.Name)
		}
	}

	// Remove gold_rules from COCKPIT.md
	cockpitMDPath := filepath.Join(a.cockpitDir, "COCKPIT.md")
	if data, err := os.ReadFile(cockpitMDPath); err == nil {
		content := string(data)
		startMarker := fmt.Sprintf("<!-- gold-rule:%s -->", pkg.Name)
		endMarker := fmt.Sprintf("<!-- /gold-rule:%s -->", pkg.Name)

		startIdx := strings.Index(content, startMarker)
		endIdx := strings.Index(content, endMarker)

		if startIdx != -1 && endIdx != -1 {
			endIdx += len(endMarker)
			if endIdx < len(content) && content[endIdx] == '\n' {
				endIdx++
			}
			if startIdx > 0 && content[startIdx-1] == '\n' {
				startIdx--
			}

			newContent := content[:startIdx] + content[endIdx:]
			if err := os.WriteFile(cockpitMDPath, []byte(newContent), 0o644); err != nil {
				return fmt.Errorf("failed to remove gold rules from COCKPIT.md: %w", err)
			}
			fmt.Printf("  ✓ gold_rules removed from COCKPIT.md\n")
		}
	}

	return nil
}

// injectGoldRules appends a package's gold rules to COCKPIT.md.
func (a *AssetSyncer) injectGoldRules(pkg *Package) error {
	cockpitMDPath := filepath.Join(a.cockpitDir, "COCKPIT.md")

	if _, err := os.Stat(cockpitMDPath); os.IsNotExist(err) {
		if err := os.MkdirAll(a.cockpitDir, 0o755); err != nil {
			return fmt.Errorf("failed to create cockpit dir: %w", err)
		}
		baseContent := "# AICockpit — AI Agent Configuration\n\nYou are an AI agent operating with the AICockpit harness.\nAlways use cockpit commands when available.\n\n## Gold Rules\n\nRules injected by installed packages — always follow these:\n\n"
		if err := os.WriteFile(cockpitMDPath, []byte(baseContent), 0o644); err != nil {
			return fmt.Errorf("failed to create base COCKPIT.md: %w", err)
		}
	}

	data, err := os.ReadFile(cockpitMDPath)
	if err != nil {
		return fmt.Errorf("failed to read COCKPIT.md: %w", err)
	}

	content := string(data)
	startMarker := fmt.Sprintf("<!-- gold-rule:%s -->", pkg.Name)
	endMarker := fmt.Sprintf("<!-- /gold-rule:%s -->", pkg.Name)

	if !strings.Contains(content, startMarker) {
		var sb strings.Builder
		sb.WriteString(startMarker + "\n")
		for _, rule := range pkg.Features.GoldRules {
			sb.WriteString(rule + "\n")
		}
		sb.WriteString(endMarker + "\n")

		content += "\n" + sb.String()
		if err := os.WriteFile(cockpitMDPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write gold rules to COCKPIT.md: %w", err)
		}
		fmt.Printf("  ✓ gold_rules injected into COCKPIT.md\n")
	}

	return nil
}
