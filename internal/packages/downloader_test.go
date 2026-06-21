package packages

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
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
