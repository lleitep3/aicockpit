package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed default_cockpit/*
var defaultCockpitFS embed.FS

// RestoreAssets copies the embedded default config to the given destination folder
func RestoreAssets(destinationPath string) error {
	// Walk the embedded filesystem
	err := fs.WalkDir(defaultCockpitFS, "default_cockpit", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel("default_cockpit", path)
		if err != nil {
			return err
		}

		dest := filepath.Join(destinationPath, relPath)

		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		// Check if file already exists in destination
		if _, err := os.Stat(dest); err == nil {
			// File exists, do not overwrite user config
			return nil
		}

		// Read from embedded FS
		data, err := defaultCockpitFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Write to disk
		return os.WriteFile(dest, data, 0644)
	})

	if err != nil {
		return fmt.Errorf("failed to restore assets: %w", err)
	}

	return nil
}
