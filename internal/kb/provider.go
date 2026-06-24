package kb

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/packages"
)

// RunSearchExtensions iterates over all installed packages and runs their kb search extensions.
// Returns the output of the first extension that succeeds.
func RunSearchExtensions(ctx context.Context, pm *packages.PackageManager, query string) (string, error) {
	pkgs, err := pm.ListInstalledPackages()
	if err != nil {
		return "", err
	}

	for _, pkg := range pkgs {
		if kbExts, ok := pkg.Extensions["kb"]; ok {
			if searchCmd, ok := kbExts["search"]; ok {
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

				return string(out), nil
			}
		}
	}

	return "", fmt.Errorf("no search extensions found or all failed")
}

// RunIndexExtensions iterates over all installed packages and runs their kb index extensions.
func RunIndexExtensions(ctx context.Context, pm *packages.PackageManager, roots []string, fast bool) error {
	pkgs, err := pm.ListInstalledPackages()
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if kbExts, ok := pkg.Extensions["kb"]; ok {
			if indexCmd, ok := kbExts["index"]; ok {
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

				err := cmd.Run()
				if err != nil {
					fmt.Printf("Warning: %s index extension failed: %v\n", pkg.Name, err)
				}
			}
		}
	}

	return nil
}
