package tracking

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/packages"
)

func TestGenerateHeader(t *testing.T) {
	pkg := &packages.Package{
		Name:    "testpkg",
		Version: "1.0.0",
		Metadata: packages.Metadata{
			CreationDate: "2026-01-01T00:00:00Z",
			LastModified: "2026-01-02T00:00:00Z",
		},
	}

	got := GenerateHeader(pkg)
	want := "// package:testpkg version:1.0.0 created:2026-01-01T00:00:00Z updated:2026-01-02T00:00:00Z"
	if got != want {
		t.Fatalf("GenerateHeader mismatch. got %q, want %q", got, want)
	}
}

func TestInjectHeader(t *testing.T) {
	// Setup temporary directory
	dir, err := os.MkdirTemp("", "tracking_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "test.txt")
	originalContent := "line1\nline2"
	if err := os.WriteFile(filePath, []byte(originalContent+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	pkg := &packages.Package{
		Name:     "testpkg",
		Version:  "1.0.0",
		Metadata: packages.Metadata{CreationDate: "2026-01-01T00:00:00Z", LastModified: "2026-01-02T00:00:00Z"},
	}

	if err := InjectHeader(filePath, pkg); err != nil {
		t.Fatalf("InjectHeader returned error: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 {
		t.Fatalf("unexpected number of lines: %d", len(lines))
	}
	expectedHeader := "// package:testpkg version:1.0.0 created:2026-01-01T00:00:00Z updated:2026-01-02T00:00:00Z"
	if lines[0] != expectedHeader {
		t.Fatalf("header mismatch. got %q, want %q", lines[0], expectedHeader)
	}
	if lines[1] != "line1" || lines[2] != "line2" {
		t.Fatalf("content lines shifted or altered: %v", lines[1:3])
	}
}

func TestInjectHeader_ReadFileError(t *testing.T) {
	pkg := &packages.Package{
		Name:    "testpkg",
		Version: "1.0.0",
		Metadata: packages.Metadata{
			CreationDate: "2026-01-01T00:00:00Z",
			LastModified: "2026-01-02T00:00:00Z",
		},
	}

	err := InjectHeader("/nonexistent/path/tracking_test_file.txt", pkg)
	if err == nil {
		t.Fatalf("expected error for non-existent file")
	}
}

func TestInjectHeader_WriteFileError(t *testing.T) {
	dir, err := os.MkdirTemp("", "tracking_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("line1\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Make file read-only so WriteFile fails.
	if err := os.Chmod(filePath, 0o444); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}
	defer os.Chmod(filePath, 0o644)

	pkg := &packages.Package{
		Name:    "testpkg",
		Version: "1.0.0",
		Metadata: packages.Metadata{
			CreationDate: "2026-01-01T00:00:00Z",
			LastModified: "2026-01-02T00:00:00Z",
		},
	}

	err = InjectHeader(filePath, pkg)
	if err == nil {
		t.Fatalf("expected error when directory is read-only")
	}
}
