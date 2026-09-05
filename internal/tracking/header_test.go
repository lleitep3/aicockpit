package tracking

import (
	"github.com/lleitep3/aicockpit/internal/packages"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectHeader(t *testing.T) {
	// Setup temporary directory
	dir, err := ioutil.TempDir("", "tracking_test")
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
