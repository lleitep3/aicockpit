package kb

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lleitep3/aicockpit/internal/packages"
)

// Contributor is an in-process knowledge-base extension.
// Package scripts stay on disk. memory-hindsight implements Contributor.
// A nil Contributor is skipped.
type Contributor interface {
	// Search returns extra recall text for query. limit caps the hits.
	// Zero limit uses the contributor default. Disabled contributors return
	// "", nil. A non-nil error means the contributor was enabled but failed.
	// Callers keep the local knowledge base.
	Search(ctx context.Context, query string, limit int) (string, error)
	// Index retains documents under roots. fast is the rebuild-cache fast
	// flag. A non-nil error does not discard the local index.
	Index(ctx context.Context, roots []string, fast bool) error
}

// SearchExtensionResult is the output of package scripts and in-process
// contributors. Supplement is filled even when the error is non-nil, so
// callers can fall back to local search and still show Hindsight hits.
type SearchExtensionResult struct {
	// Script is the first successful package search script.
	Script string
	// ScriptOK reports whether a package script succeeded.
	ScriptOK bool
	// Supplement is extra text from in-process contributors.
	Supplement string
}

// RunSearchExtensions runs installed kb search scripts and optional
// in-process contributors such as memory-hindsight.
func RunSearchExtensions(ctx context.Context, pm *packages.PackageManager, query string, limit int, contributors ...Contributor) (SearchExtensionResult, error) {
	script, ok := firstSearchScript(ctx, pm, query)
	result := SearchExtensionResult{
		Script:     script,
		ScriptOK:   ok,
		Supplement: SearchSupplement(ctx, query, limit, contributors...),
	}
	if !ok {
		return result, fmt.Errorf("no search extensions found or all failed")
	}
	return result, nil
}

// SearchSupplement runs in-process search contributors.
// A contributor error is reported on stdout and does not discard text that
// another contributor already produced.
func SearchSupplement(ctx context.Context, query string, limit int, contributors ...Contributor) string {
	text, err := runContributorSearch(ctx, query, limit, contributors...)
	if err != nil && strings.TrimSpace(text) == "" {
		fmt.Printf("Warning: memory-hindsight search skipped: %v\n", err)
	}
	return strings.TrimSpace(text)
}

func runContributorSearch(ctx context.Context, query string, limit int, contributors ...Contributor) (string, error) {
	var b strings.Builder
	var first error
	for _, contributor := range contributors {
		if contributor == nil {
			continue
		}
		if ctx != nil && ctx.Err() != nil {
			return b.String(), fmt.Errorf("kb search extension: %w", ctx.Err())
		}
		text, err := contributor.Search(ctx, query, limit)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(text)
	}
	return b.String(), first
}

func firstSearchScript(ctx context.Context, pm *packages.PackageManager, query string) (string, bool) {
	if pm == nil {
		return "", false
	}
	pkgs, err := pm.ListInstalledPackages()
	if err != nil {
		return "", false
	}
	for _, pkg := range pkgs {
		kbExts, ok := pkg.Extensions["kb"]
		if !ok {
			continue
		}
		searchCmd, ok := kbExts["search"]
		if !ok {
			continue
		}
		scriptPath := filepath.Join(pm.GetPackageInstallPath(pkg.Name), searchCmd)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}
		fmt.Printf("Running %s search extension...\n", pkg.Name)
		cmd := exec.CommandContext(ctx, "sh", scriptPath, query)
		cmd.Dir = pm.GetPackageInstallPath(pkg.Name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: %s search extension failed: %v\nOutput: %s\n", pkg.Name, err, string(out))
			continue
		}
		return string(out), true
	}
	return "", false
}

// RunIndexExtensions runs installed kb index scripts and optional in-process
// contributors. Contributor failures are warnings. The local index is kept.
func RunIndexExtensions(ctx context.Context, pm *packages.PackageManager, roots []string, fast bool, contributors ...Contributor) error {
	listErr := runIndexScripts(ctx, pm, roots, fast)
	for _, contributor := range contributors {
		if contributor == nil {
			continue
		}
		if err := contributor.Index(ctx, roots, fast); err != nil {
			fmt.Printf("Warning: memory-hindsight index skipped: %v\n", err)
		}
	}
	return listErr
}

func runIndexScripts(ctx context.Context, pm *packages.PackageManager, roots []string, fast bool) error {
	if pm == nil {
		return nil
	}
	pkgs, err := pm.ListInstalledPackages()
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		kbExts, ok := pkg.Extensions["kb"]
		if !ok {
			continue
		}
		indexCmd, ok := kbExts["index"]
		if !ok {
			continue
		}
		scriptPath := filepath.Join(pm.GetPackageInstallPath(pkg.Name), indexCmd)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}
		fmt.Printf("Running %s index extension...\n", pkg.Name)
		args := []string{scriptPath}
		if fast {
			args = append(args, "--fast")
		}
		args = append(args, roots...)
		cmd := exec.CommandContext(ctx, "sh", args...)
		cmd.Dir = pm.GetPackageInstallPath(pkg.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: %s index extension failed: %v\n", pkg.Name, err)
		}
	}
	return nil
}
