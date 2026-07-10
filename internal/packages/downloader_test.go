package packages

import (
	"archive/zip"
	"bytes"
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

func TestDownloadPackageFromURL_CancelledDuringCopy(t *testing.T) {
	// Server that streams slowly so cancellation hits during io.Copy.
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PK")) //nolint:errcheck
		<-blocked
	}))
	defer srv.Close()
	defer close(blocked)

	tmpDir := t.TempDir()
	downloader := NewPackageDownloader()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel shortly after the handler starts writing.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := downloader.DownloadPackageFromURL(ctx, srv.URL+"/pkg.zip", "pkg", filepath.Join(tmpDir, "pkg"))
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestDefaultDownloadTimeout(t *testing.T) {
	if defaultDownloadTimeout != 60*time.Second {
		t.Errorf("expected defaultDownloadTimeout = 60s, got %v", defaultDownloadTimeout)
	}
}

func TestNewPackageDownloader_WithToken(t *testing.T) {
	t.Setenv("COCKPIT_GITHUB_TOKEN", "ghp_test_token")
	defer t.Setenv("COCKPIT_GITHUB_TOKEN", "")

	downloader := NewPackageDownloader()
	if downloader == nil {
		t.Fatal("Expected non-nil downloader")
	}
	if downloader.httpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
	if downloader.gitToken != "ghp_test_token" {
		t.Errorf("expected token %q, got %q", "ghp_test_token", downloader.gitToken)
	}
}

func TestNewPackageDownloader_NoEnvToken(t *testing.T) {
	t.Setenv("COCKPIT_GITHUB_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	downloader := NewPackageDownloader()
	if downloader == nil {
		t.Fatal("Expected non-nil downloader")
	}
	if downloader.httpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
	if downloader.httpClient.CheckRedirect == nil {
		t.Error("Expected redirect handler to be configured")
	}
}

func TestGetGitHubToken_EnvVars(t *testing.T) {
	t.Run("COCKPIT_GITHUB_TOKEN", func(t *testing.T) {
		t.Setenv("COCKPIT_GITHUB_TOKEN", "cockpit-token")
		t.Setenv("GITHUB_TOKEN", "")
		if got := getGitHubToken(); got != "cockpit-token" {
			t.Errorf("expected cockpit-token, got %q", got)
		}
	})

	t.Run("GITHUB_TOKEN", func(t *testing.T) {
		t.Setenv("COCKPIT_GITHUB_TOKEN", "")
		t.Setenv("GITHUB_TOKEN", "github-token")
		if got := getGitHubToken(); got != "github-token" {
			t.Errorf("expected github-token, got %q", got)
		}
	})
}

func TestGetGitHubToken_GHCLI(t *testing.T) {
	// When env vars are empty, getGitHubToken falls back to gh CLI.
	// This test simply exercises that fallback path without requiring auth.
	t.Setenv("COCKPIT_GITHUB_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	_ = getGitHubToken()
}

func TestExtractPackageFromZip_Corrupted(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "corrupted.zip")
	if err := os.WriteFile(zipPath, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	downloader := NewPackageDownloader()
	err := downloader.extractPackageFromZip(zipPath, "pkg", filepath.Join(tmpDir, "out"), "repo", "main")
	if err == nil {
		t.Error("Expected error for corrupted ZIP")
	}
}

func TestExtractPackageFromZip_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "empty.zip")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	zw := zip.NewWriter(zipFile)
	zw.Close()
	zipFile.Close()

	downloader := NewPackageDownloader()
	err = downloader.extractPackageFromZip(zipPath, "pkg", filepath.Join(tmpDir, "out"), "repo", "main")
	if err == nil {
		t.Error("Expected error when package not found in empty ZIP")
	}
}

func TestDownloadPackageFromURL_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	downloader := NewPackageDownloader()

	err := downloader.DownloadPackageFromURL(context.Background(), srv.URL+"/pkg.zip", "pkg", filepath.Join(tmpDir, "pkg"))
	if err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}

func TestDownloadPackageFromURL_InvalidZip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not a zip")) //nolint:errcheck
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	downloader := NewPackageDownloader()

	err := downloader.DownloadPackageFromURL(context.Background(), srv.URL+"/pkg.zip", "pkg", filepath.Join(tmpDir, "pkg"))
	if err == nil {
		t.Fatal("expected error for invalid ZIP body, got nil")
	}
}

func TestDownloadPackageFromURL_GitHubTokenHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		// Build a tiny valid zip on the fly.
		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		f, _ := zw.Create("repo-main/pkg/README.md")
		f.Write([]byte("# test")) //nolint:errcheck
		zw.Close()
		w.Write(buf.Bytes()) //nolint:errcheck
	}))
	defer srv.Close()

	t.Setenv("COCKPIT_GITHUB_TOKEN", "secret-token")

	tmpDir := t.TempDir()
	downloader := NewPackageDownloader()

	// URL containing "github.com" triggers the Authorization header.
	githubURL := srv.URL + "/github.com/pkg.zip"
	err := downloader.DownloadPackageFromURL(context.Background(), githubURL, "pkg", filepath.Join(tmpDir, "pkg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "token secret-token" {
		t.Errorf("expected Authorization header %q, got %q", "token secret-token", gotAuth)
	}
}

func TestNewPackageDownloader_PreservesAuthOnRedirect(t *testing.T) {
	finalAuth := ""
	finalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)

		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		f, _ := zw.Create("repo-main/pkg/README.md")
		f.Write([]byte("# test")) //nolint:errcheck
		zw.Close()
		w.Write(buf.Bytes()) //nolint:errcheck
	}))
	defer finalSrv.Close()

	redirectSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalSrv.URL+"/pkg.zip", http.StatusFound)
	}))
	defer redirectSrv.Close()

	t.Setenv("COCKPIT_GITHUB_TOKEN", "redirect-token")

	tmpDir := t.TempDir()
	downloader := NewPackageDownloader()

	// First request URL contains "github.com" so the Authorization header is set.
	err := downloader.DownloadPackageFromURL(context.Background(), redirectSrv.URL+"/github.com/pkg.zip", "pkg", filepath.Join(tmpDir, "pkg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if finalAuth != "token redirect-token" {
		t.Errorf("expected Authorization preserved on redirect, got %q", finalAuth)
	}
}

func TestDownloadPackageFromGitHub_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pkgName := "hello-world"
	repo := "cockpit-registry"
	branch := "main"

	// Build a ZIP with the expected structure: repo-branch/pkgName/...
	zipFile := func() []byte {
		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		files := []struct {
			name    string
			content string
		}{
			{fmt.Sprintf("%s-%s/%s/", repo, branch, pkgName), ""},
			{fmt.Sprintf("%s-%s/%s/README.md", repo, branch, pkgName), "# Hello"},
		}
		for _, f := range files {
			var w io.Writer
			if f.name[len(f.name)-1] == '/' {
				w, _ = zw.CreateHeader(&zip.FileHeader{Name: f.name})
			} else {
				w, _ = zw.Create(f.name)
			}
			w.Write([]byte(f.content)) //nolint:errcheck
		}
		zw.Close()
		return buf.Bytes()
	}()

	oldBase := githubBaseURL
	defer func() { githubBaseURL = oldBase }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/%s/%s/archive/refs/heads/%s.zip", "owner", repo, branch)
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipFile) //nolint:errcheck
	}))
	defer srv.Close()
	githubBaseURL = srv.URL

	downloader := NewPackageDownloader()
	destDir := filepath.Join(tmpDir, pkgName)
	err := downloader.DownloadPackageFromGitHub(context.Background(), "owner", repo, branch, pkgName, destDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	readme := filepath.Join(destDir, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		t.Errorf("expected README.md at %s", readme)
	}
}

func TestDownloadPackageFromGitHub_NotFound(t *testing.T) {
	oldBase := githubBaseURL
	defer func() { githubBaseURL = oldBase }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	githubBaseURL = srv.URL

	downloader := NewPackageDownloader()
	err := downloader.DownloadPackageFromGitHub(context.Background(), "owner", "repo", "main", "pkg", t.TempDir())
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestDownloadPackageFromGitHub_PackageMissingInZip(t *testing.T) {
	oldBase := githubBaseURL
	defer func() { githubBaseURL = oldBase }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve a valid ZIP that does not contain the requested package.
		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		zw.Create("repo-main/other-pkg/README.md") //nolint:errcheck
		zw.Close()
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes()) //nolint:errcheck
	}))
	defer srv.Close()
	githubBaseURL = srv.URL

	downloader := NewPackageDownloader()
	err := downloader.DownloadPackageFromGitHub(context.Background(), "owner", "repo", "main", "missing-pkg", t.TempDir())
	if err == nil {
		t.Fatal("expected error when package is missing in ZIP")
	}
}

func TestDownloadPackageFromGitHub_Timeout(t *testing.T) {
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	defer srv.Close()
	defer close(blocked)

	oldBase := githubBaseURL
	defer func() { githubBaseURL = oldBase }()
	githubBaseURL = srv.URL

	tmpDir := t.TempDir()
	downloader := NewPackageDownloader()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := downloader.DownloadPackageFromGitHub(ctx, "owner", "repo", "main", "pkg", tmpDir)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExtractPackageFromZip_WithDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	zw := zip.NewWriter(zipFile)
	files := []struct {
		name    string
		content string
		isDir   bool
	}{
		{"repo-main/pkg/", "", true},
		{"repo-main/pkg/subdir/", "", true},
		{"repo-main/pkg/subdir/file.txt", "nested", false},
	}
	for _, f := range files {
		header := &zip.FileHeader{Name: f.name}
		if f.isDir {
			header.Name += "/"
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatalf("setup: %v", err)
		}
		if !f.isDir {
			w.Write([]byte(f.content)) //nolint:errcheck
		}
	}
	zw.Close()
	zipFile.Close()

	downloader := NewPackageDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = downloader.extractPackageFromZip(zipPath, "pkg", destDir, "repo", "main")
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}

	nested := filepath.Join(destDir, "subdir", "file.txt")
	data, err := os.ReadFile(nested)
	if err != nil {
		t.Fatalf("expected nested file: %v", err)
	}
	if string(data) != "nested" {
		t.Errorf("expected nested content 'nested', got %q", string(data))
	}
}

func TestExtractPackageFromZip_PreservesExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	zw := zip.NewWriter(zipFile)
	header := &zip.FileHeader{Name: "repo-main/pkg/run.sh"}
	header.SetMode(0o755)
	w, _ := zw.CreateHeader(header)
	w.Write([]byte("#!/bin/sh\n")) //nolint:errcheck
	zw.Close()
	zipFile.Close()

	downloader := NewPackageDownloader()
	destDir := filepath.Join(tmpDir, "extracted")
	err = downloader.extractPackageFromZip(zipPath, "pkg", destDir, "repo", "main")
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(destDir, "run.sh"))
	if err != nil {
		t.Fatalf("expected executable file: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected executable permissions, got %o", info.Mode().Perm())
	}
}

func TestExtractPackageFromZip_DestDirNotWritable(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	zw := zip.NewWriter(zipFile)
	zw.Create("repo-main/pkg/file.txt") //nolint:errcheck
	zw.Close()
	zipFile.Close()

	// Create a read-only destination directory.
	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer os.Chmod(roDir, 0o755) //nolint:errcheck

	downloader := NewPackageDownloader()
	err = downloader.extractPackageFromZip(zipPath, "pkg", roDir, "repo", "main")
	if err == nil {
		t.Error("Expected error when destination directory is not writable")
	}
}

func TestDownloadPackageFromURL_DestDirNotWritable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		zw.Create("file.txt") //nolint:errcheck
		zw.Close()
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes()) //nolint:errcheck
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer os.Chmod(roDir, 0o755) //nolint:errcheck

	downloader := NewPackageDownloader()
	err := downloader.DownloadPackageFromURL(context.Background(), srv.URL+"/pkg.zip", "pkg", roDir)
	if err == nil {
		t.Error("Expected error when destination directory is not writable")
	}
}
