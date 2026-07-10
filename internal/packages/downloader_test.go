package packages

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewPackageDownloader(t *testing.T) {
	downloader := NewPackageDownloader()

	if downloader == nil {
		t.Fatal("Expected non-nil downloader")
	}

	if downloader.httpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
}

func TestExtractPackageFromZip(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create a test ZIP file with package structure
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create ZIP file: %v", err)
	}

	// Create ZIP writer
	zipWriter := zip.NewWriter(zipFile)

	// Add test files to ZIP
	// Structure: cockpit-registry-main/hello-world/...
	testFiles := []struct {
		name    string
		content string
		isDir   bool
	}{
		{"cockpit-registry-main/hello-world/", "", true},
		{"cockpit-registry-main/hello-world/README.md", "# Hello World", false},
		{"cockpit-registry-main/hello-world/cockpit-package.yml", "name: hello-world", false},
		{"cockpit-registry-main/hello-world/modules/", "", true},
		{"cockpit-registry-main/hello-world/modules/cmd.go", "package modules", false},
	}

	for _, file := range testFiles {
		header := &zip.FileHeader{
			Name: file.name,
		}

		if file.isDir {
			header.Name += "/"
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			t.Fatalf("Failed to create ZIP entry: %v", err)
		}

		if !file.isDir {
			if _, err := io.WriteString(writer, file.content); err != nil {
				t.Fatalf("Failed to write ZIP content: %v", err)
			}
		}
	}

	zipWriter.Close()
	zipFile.Close()

	// Test extraction
	downloader := NewPackageDownloader()
	destDir := filepath.Join(tmpDir, "extracted")

	err = downloader.extractPackageFromZip(zipPath, "hello-world", destDir, "cockpit-registry", "main")
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// Verify extracted files
	expectedFiles := []string{
		"README.md",
		"cockpit-package.yml",
		"modules/cmd.go",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(destDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file not found: %s", file)
		}
	}

	// Verify file content
	readmeContent, err := os.ReadFile(filepath.Join(destDir, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	if string(readmeContent) != "# Hello World" {
		t.Errorf("Expected README.md content '# Hello World', got '%s'", string(readmeContent))
	}
}

func TestExtractPackageFromZipNotFound(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create a test ZIP file without the expected package
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create ZIP file: %v", err)
	}

	// Create ZIP writer
	zipWriter := zip.NewWriter(zipFile)

	// Add test files to ZIP (but not the package we're looking for)
	header := &zip.FileHeader{
		Name: "cockpit-registry-main/other-package/README.md",
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}

	io.WriteString(writer, "# Other Package")

	zipWriter.Close()
	zipFile.Close()

	// Test extraction
	downloader := NewPackageDownloader()
	destDir := filepath.Join(tmpDir, "extracted")

	err = downloader.extractPackageFromZip(zipPath, "hello-world", destDir, "cockpit-registry", "main")
	if err == nil {
		t.Error("Expected error when package not found in ZIP")
	}
}

func TestExtractPackageFromZipCreateDirectory(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create a test ZIP file with package structure
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create ZIP file: %v", err)
	}

	// Create ZIP writer
	zipWriter := zip.NewWriter(zipFile)

	// Add test files to ZIP
	testFiles := []struct {
		name    string
		content string
	}{
		{"cockpit-registry-main/hello-world/README.md", "# Hello World"},
		{"cockpit-registry-main/hello-world/subdir/file.txt", "content"},
	}

	for _, file := range testFiles {
		header := &zip.FileHeader{
			Name: file.name,
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			t.Fatalf("Failed to create ZIP entry: %v", err)
		}

		io.WriteString(writer, file.content)
	}

	zipWriter.Close()
	zipFile.Close()

	// Test extraction
	downloader := NewPackageDownloader()
	destDir := filepath.Join(tmpDir, "extracted")

	err = downloader.extractPackageFromZip(zipPath, "hello-world", destDir, "cockpit-registry", "main")
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// Verify subdirectory was created
	subdir := filepath.Join(destDir, "subdir")
	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Error("Expected subdirectory not created")
	}

	// Verify file in subdirectory
	filePath := filepath.Join(destDir, "subdir", "file.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected file in subdirectory not found")
	}
}

// newTestZipServer creates a test HTTP server that serves a minimal ZIP containing
// a package directory, and returns the server and a cleanup function.
func newTestZipServer(t *testing.T, packageName string) *httptest.Server {
	t.Helper()

	// Build a ZIP in memory.
	tmpZip, err := os.CreateTemp("", "test-zip-*.zip")
	if err != nil {
		t.Fatalf("failed to create temp zip: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpZip.Name()) })

	zw := zip.NewWriter(tmpZip)
	entry := fmt.Sprintf("repo-main/%s/README.md", packageName)
	w, err := zw.Create(entry)
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := io.WriteString(w, "# test"); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}
	zw.Close()
	tmpZip.Close()

	zipBytes, err := os.ReadFile(tmpZip.Name())
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipBytes) //nolint:errcheck
	}))
	return srv
}

func TestDownloadPackageFromURL_WithExplicitContext(t *testing.T) {
	const pkg = "hello-world"
	srv := newTestZipServer(t, pkg)
	defer srv.Close()

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, pkg)

	downloader := NewPackageDownloader()
	ctx := context.Background()

	err := downloader.DownloadPackageFromURL(ctx, srv.URL+"/pkg.zip", pkg, destDir)
	if err != nil {
		t.Fatalf("expected no error with explicit context, got: %v", err)
	}

	readmePath := filepath.Join(destDir, fmt.Sprintf("repo-main/%s/README.md", pkg))
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Errorf("expected downloaded file at %s", readmePath)
	}
}

func TestDownloadPackageFromURL_WithNoDeadlineContext(t *testing.T) {
	const pkg = "hello-world"
	srv := newTestZipServer(t, pkg)
	defer srv.Close()

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, pkg)

	downloader := NewPackageDownloader()

	// context.TODO() has no deadline — the function must apply defaultDownloadTimeout internally.
	err := downloader.DownloadPackageFromURL(context.TODO(), srv.URL+"/pkg.zip", pkg, destDir)
	if err != nil {
		t.Fatalf("expected no error with no-deadline context, got: %v", err)
	}
}

func TestDownloadPackageFromURL_WithCancelledContext(t *testing.T) {
	// Server that blocks until the client gives up.
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	defer srv.Close()
	defer close(blocked)

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "pkg")

	downloader := NewPackageDownloader()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := downloader.DownloadPackageFromURL(ctx, srv.URL+"/pkg.zip", "pkg", destDir)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestDownloadPackageFromURL_WithExpiredTimeout(t *testing.T) {
	// Server that always hangs.
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	defer srv.Close()
	defer close(blocked)

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "pkg")

	downloader := NewPackageDownloader()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := downloader.DownloadPackageFromURL(ctx, srv.URL+"/pkg.zip", "pkg", destDir)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestDefaultDownloadTimeout(t *testing.T) {
	if defaultDownloadTimeout != 60*time.Second {
		t.Errorf("expected defaultDownloadTimeout = 60s, got %v", defaultDownloadTimeout)
	}
}
