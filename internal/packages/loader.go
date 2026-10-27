package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackageLoader loads and manages installed packages
type PackageLoader struct {
	packagesDir string
}

// NewPackageLoader creates a new package loader
func NewPackageLoader(cockpitDir string) *PackageLoader {
	return &PackageLoader{
		packagesDir: filepath.Join(cockpitDir, "packages"),
	}
}

// LoadInstalledPackages returns list of installed packages
func (pl *PackageLoader) LoadInstalledPackages() ([]string, error) {
	entries, err := os.ReadDir(pl.packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read packages directory: %w", err)
	}

	var packages []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a valid package (has cockpit-package.yml)
		manifestPath := filepath.Join(pl.packagesDir, entry.Name(), "cockpit-package.yml")
		if _, err := os.Stat(manifestPath); err == nil {
			packages = append(packages, entry.Name())
		}
	}

	return packages, nil
}

// GetPackagePath returns the path to an installed package
func (pl *PackageLoader) GetPackagePath(packageName string) string {
	return filepath.Join(pl.packagesDir, packageName)
}

// LoadPackageManifest loads the manifest of an installed package
func (pl *PackageLoader) LoadPackageManifest(packageName string) (*Package, error) {
	packagePath := pl.GetPackagePath(packageName)
	return LoadPackage(packagePath)
}

// ExecutePackageCommand executes a command from a package
// This loads the package's module and executes the command
func (pl *PackageLoader) ExecutePackageCommand(packageName, commandName string, args []string) error {
	packagePath := pl.GetPackagePath(packageName)
	modulesPath := filepath.Join(packagePath, "modules")

	// Check if modules directory exists
	if _, err := os.Stat(modulesPath); err != nil {
		return fmt.Errorf("package %s has no modules", packageName)
	}

	// Build the command to execute the package module
	// This would require the package to have a compiled binary or script
	// For now, we'll return an error indicating this needs to be implemented
	return fmt.Errorf("package command execution not yet implemented")
}

// GetPackageCommands returns the list of commands provided by a package
func (pl *PackageLoader) GetPackageCommands(packageName string) ([]string, error) {
	pkg, err := pl.LoadPackageManifest(packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to load package manifest: %w", err)
	}

	// Extract command names from package features
	var commands []string

	// Check if package has modules (which means it has commands)
	if len(pkg.Features.Modules) > 0 {
		// For now, assume the package name is the command name
		commands = append(commands, packageName)
	}

	return commands, nil
}

// CompilePackageModules compiles the Go modules of a package
// This is needed to make the commands available
func (pl *PackageLoader) CompilePackageModules(packageName string) error {
	packagePath := pl.GetPackagePath(packageName)
	modulesPath := filepath.Join(packagePath, "modules")

	// Check if modules directory exists
	if _, err := os.Stat(modulesPath); err != nil {
		return fmt.Errorf("package %s has no modules", packageName)
	}

	// For now, we'll skip compilation as it requires complex setup
	// In the future, this could compile the modules into a shared library
	return nil
}

// DiscoverPackageCommands discovers available commands in a package
// by scanning the modules directory for command definitions
func (pl *PackageLoader) DiscoverPackageCommands(packageName string) ([]string, error) {
	packagePath := pl.GetPackagePath(packageName)
	modulesPath := filepath.Join(packagePath, "modules")

	// Check if modules directory exists
	if _, err := os.Stat(modulesPath); err != nil {
		return []string{}, nil
	}

	// Look for cmd.go files
	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read modules directory: %w", err)
	}

	var commands []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Look for cmd.go or *_cmd.go files
		if strings.HasPrefix(entry.Name(), "cmd") && strings.HasSuffix(entry.Name(), ".go") {
			// Extract command name from file
			// For now, assume cmd.go contains a NewXxxCommand function
			// We'd need to parse the Go file to extract the actual command name
			commands = append(commands, packageName)
		}
	}

	return commands, nil
}

// SymlinkPackageModules creates symlinks to package modules in the main codebase
// This allows the package commands to be loaded dynamically
func (pl *PackageLoader) SymlinkPackageModules(packageName string) error {
	packagePath := pl.GetPackagePath(packageName)
	modulesPath := filepath.Join(packagePath, "modules")

	// Check if modules directory exists
	if _, err := os.Stat(modulesPath); err != nil {
		return fmt.Errorf("package %s has no modules", packageName)
	}

	// For now, we'll skip symlinking as it requires careful setup
	// In the future, this could create symlinks to make modules available
	return nil
}

// RegisterPackageCommands registers commands from installed packages
// This should be called during CLI initialization
func (pl *PackageLoader) RegisterPackageCommands() (map[string]string, error) {
	packages, err := pl.LoadInstalledPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to load installed packages: %w", err)
	}

	// Map of command name to package name
	commands := make(map[string]string)

	for _, pkgName := range packages {
		// Try to discover commands in this package
		cmds, err := pl.DiscoverPackageCommands(pkgName)
		if err != nil {
			// Log warning but continue
			fmt.Printf("Warning: failed to discover commands in package %s: %v\n", pkgName, err)
			continue
		}

		for _, cmd := range cmds {
			commands[cmd] = pkgName
		}
	}

	return commands, nil
}
