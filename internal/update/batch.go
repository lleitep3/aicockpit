package update

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lleitep3/aicockpit/internal/packages"
	"golang.org/x/mod/semver"
)

type Outcome struct {
	Component string `json:"component"`
	Name      string `json:"name"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}
type Report struct {
	SchemaVersion int       `json:"schema_version"`
	Check         bool      `json:"check"`
	Outcomes      []Outcome `json:"outcomes"`
}

func (r Report) Failed() bool {
	for _, o := range r.Outcomes {
		if o.Status == "failed" || o.Status == "blocked" {
			return true
		}
	}
	return false
}

type packageOperations interface {
	UpgradePackage(string, string) error
	TriggerDeploy(string) error
}
type PackageBatch struct {
	root       string
	registries []packages.RegistryConfig
	operations packageOperations
}

func NewPackageBatch(root string, registries []packages.RegistryConfig) *PackageBatch {
	ordered := append([]packages.RegistryConfig(nil), registries...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Priority < ordered[j].Priority })
	return &PackageBatch{root: root, registries: ordered, operations: packages.NewPackageManager(root)}
}

type registrySnapshot struct {
	index *packages.PackageIndex
	err   error
}

func (b *PackageBatch) Run(check bool) Report {
	report := Report{SchemaVersion: 1, Check: check, Outcomes: []Outcome{}}
	entries, err := os.ReadDir(filepath.Join(b.root, "packages"))
	if os.IsNotExist(err) {
		return report
	}
	if err != nil {
		report.Outcomes = append(report.Outcomes, Outcome{Component: "packages", Name: "inventory", Status: "failed", Message: err.Error()})
		return report
	}
	cache := packages.NewRegistryCache(b.root)
	snapshots := map[string]registrySnapshot{}
	updated := false
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			report.Outcomes = append(report.Outcomes, Outcome{Component: "package", Name: entry.Name(), Status: "blocked", Message: "symbolic installed package requires manual reconciliation"})
			continue
		}
		if !entry.IsDir() {
			continue
		}
		outcome := Outcome{Component: "package", Name: entry.Name(), Status: "failed"}
		installed, err := packages.LoadPackage(filepath.Join(b.root, "packages", entry.Name()))
		if err != nil {
			outcome.Message = err.Error()
		} else if installed.Name != entry.Name() {
			outcome.Message = "installed manifest identity mismatch"
		} else {
			outcome = b.updateOne(installed, check, cache, snapshots)
		}
		if outcome.Status == "updated" {
			updated = true
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	if updated {
		outcome := Outcome{Component: "deploy", Name: "providers", Status: "updated"}
		if err := b.operations.TriggerDeploy(""); err != nil {
			outcome.Status = "failed"
			outcome.Message = "packages activated; retry cockpit deploy: " + err.Error()
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	return report
}
func (b *PackageBatch) resolve(name string, cache *packages.RegistryCache, snapshots map[string]registrySnapshot) (*packages.PackageIndexEntry, string, error) {
	for _, registry := range b.registries {
		if !registry.Enabled {
			continue
		}
		snapshot, exists := snapshots[registry.Name]
		if !exists {
			snapshot.err = cache.EnsureRegistry(registry)
			if snapshot.err == nil {
				snapshot.index, snapshot.err = cache.LoadPackageIndexFromCache(registry.Name)
			}
			snapshots[registry.Name] = snapshot
		}
		if snapshot.err != nil {
			return nil, "", fmt.Errorf("registry %s refresh failed: %w", registry.Name, snapshot.err)
		}
		if entry := snapshot.index.GetPackageByName(name); entry != nil {
			path := entry.Path
			if path == "" {
				path = name
			}
			source, err := cache.GetPackageFromCache(registry.Name, path)
			return entry, source, err
		}
	}
	return nil, "", fmt.Errorf("package absent from enabled registries")
}
func (b *PackageBatch) updateOne(installed *packages.Package, check bool, cache *packages.RegistryCache, snapshots map[string]registrySnapshot) Outcome {
	outcome := Outcome{Component: "package", Name: installed.Name, From: installed.Version, Status: "failed"}
	entry, source, err := b.resolve(installed.Name, cache, snapshots)
	if err != nil {
		outcome.Message = err.Error()
		return outcome
	}
	outcome.To = entry.Version
	current, available := "v"+strings.TrimPrefix(installed.Version, "v"), "v"+strings.TrimPrefix(entry.Version, "v")
	if !semver.IsValid(current) || !semver.IsValid(available) {
		outcome.Message = "cannot compare invalid semantic versions"
		return outcome
	}
	comparison := semver.Compare(available, current)
	if comparison < 0 {
		outcome.Status = "blocked"
		outcome.Message = "automatic downgrade refused"
		return outcome
	}
	if comparison == 0 {
		outcome.Status = "current"
		return outcome
	}
	candidate, err := packages.LoadPackage(source)
	if err != nil {
		outcome.Message = err.Error()
		return outcome
	}
	if candidate.Name != installed.Name || candidate.Version != entry.Version {
		outcome.Message = "candidate identity/version differs from registry index"
		return outcome
	}
	if err := candidate.Validate(source); err != nil {
		outcome.Message = err.Error()
		return outcome
	}
	if check {
		outcome.Status = "available"
		return outcome
	}
	if err := b.operations.UpgradePackage(installed.Name, source); err != nil {
		outcome.Message = err.Error()
		return outcome
	}
	outcome.Status = "updated"
	return outcome
}
