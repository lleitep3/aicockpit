package packages

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PackageDownloader handles downloading packages from registries.
type PackageDownloader struct {
	httpClient *http.Client
	gitToken   string
}

// NewPackageDownloader creates a new package downloader.
func NewPackageDownloader() *PackageDownloader {
	downloader := &PackageDownloader{
		httpClient: &http.Client{},
	}

	// Try to get GitHub token from gh CLI
	downloader.gitToken = getGitHubToken()

	// Configure redirect handling to preserve auth headers
	downloader.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Preserve Authorization header on redirects
		if len(via) > 0 && via[0].Header.Get("Authorization") != "" {
			req.Header.Set("Authorization", via[0].Header.Get("Authorization"))
		}
		return nil
	}

	return downloader
}

// getGitHubToken retrieves the GitHub token from gh CLI
func getGitHubToken() string {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// DownloadPackageFromGitHub downloads a package from GitHub as a ZIP file.
// It extracts only the package directory from the repository.
//
// Example:
//
//	owner: "lleitep3"
//	repo: "cockpit-registry"
//	branch: "main"
//	packageName: "hello-world"
//	destDir: "/home/user/.cockpit/packages/hello-world"
func (pd *PackageDownloader) DownloadPackageFromGitHub(owner, repo, branch, packageName, destDir string) error {
	// Construct GitHub API URL for downloading the repository as ZIP
	// Format: https://github.com/{owner}/{repo}/archive/refs/heads/{branch}.zip
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, branch)

	// Download the ZIP file
	fmt.Printf("Downloading package from: %s\n", downloadURL)
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available
	if pd.gitToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", pd.gitToken))
	}

	resp, err := pd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download package: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download package: HTTP %d", resp.StatusCode)
	}

	// Create temporary file for ZIP
	tmpFile, err := os.CreateTemp("", "cockpit-package-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write response body to temporary file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write package file: %w", err)
	}
	tmpFile.Close()

	// Extract package from ZIP
	if err := pd.extractPackageFromZip(tmpFile.Name(), packageName, destDir, repo, branch); err != nil {
		return err
	}

	fmt.Printf("✓ Package downloaded successfully\n")
	return nil
}

// extractPackageFromZip extracts a specific package directory from a ZIP file.
// The ZIP file structure is: {repo}-{branch}/{packageName}/...
func (pd *PackageDownloader) extractPackageFromZip(zipPath, packageName, destDir, repo, branch string) error {
	// Open ZIP file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Expected prefix in ZIP: {repo}-{branch}/{packageName}/
	// Example: cockpit-registry-main/hello-world/
	expectedPrefix := fmt.Sprintf("%s-%s/%s/", repo, branch, packageName)

	// Track if we found the package
	found := false

	// Extract files from ZIP
	for _, file := range reader.File {
		// Check if file is within the package directory
		if !strings.HasPrefix(file.Name, expectedPrefix) {
			continue
		}

		found = true

		// Get relative path (remove the prefix)
		relativePath := strings.TrimPrefix(file.Name, expectedPrefix)

		// Skip the package directory itself
		if relativePath == "" {
			continue
		}

		// Construct destination path
		destPath := filepath.Join(destDir, relativePath)

		// Create directories if needed
		if strings.HasSuffix(file.Name, "/") {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Extract file
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in ZIP: %w", err)
		}

		dstFile, err := os.Create(destPath)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		srcFile.Close()
		dstFile.Close()

		// Preserve file permissions
		if err := os.Chmod(destPath, file.Mode()); err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	if !found {
		return fmt.Errorf("package not found in repository: %s", packageName)
	}

	return nil
}

// DownloadPackageFromURL downloads a package from a custom URL.
// Useful for non-GitHub registries.
func (pd *PackageDownloader) DownloadPackageFromURL(downloadURL, packageName, destDir string) error {
	fmt.Printf("Downloading package from: %s\n", downloadURL)
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available and URL is from GitHub
	if pd.gitToken != "" && strings.Contains(downloadURL, "github.com") {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", pd.gitToken))
	}

	resp, err := pd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download package: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download package: HTTP %d", resp.StatusCode)
	}

	// Create temporary file for ZIP
	tmpFile, err := os.CreateTemp("", "cockpit-package-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write response body to temporary file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write package file: %w", err)
	}
	tmpFile.Close()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract ZIP file
	reader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	// Extract all files
	for _, file := range reader.File {
		destPath := filepath.Join(destDir, file.Name)

		if strings.HasSuffix(file.Name, "/") {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Extract file
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in ZIP: %w", err)
		}

		dstFile, err := os.Create(destPath)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		srcFile.Close()
		dstFile.Close()

		// Preserve file permissions
		if err := os.Chmod(destPath, file.Mode()); err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	fmt.Printf("✓ Package downloaded successfully\n")
	return nil
}
