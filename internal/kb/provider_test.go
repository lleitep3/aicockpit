package kb

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/packages"
)

type fakeContributor struct {
	text     string
	err      error
	indexErr error
	query    string
	limit    int
	roots    []string
	fast     bool
	calls    int
}

func (f *fakeContributor) Search(ctx context.Context, query string, limit int) (string, error) {
	if ctx != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}
	f.calls++
	f.query = query
	f.limit = limit
	return f.text, f.err
}

func (f *fakeContributor) Index(ctx context.Context, roots []string, fast bool) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	f.calls++
	f.roots = append([]string(nil), roots...)
	f.fast = fast
	return f.indexErr
}

func TestRunSearchExtensions(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, "a-bad", "search.sh", "#!/bin/sh\necho failed\nexit 1\n")
	writePkg(t, dir, "b-missing", "", "")
	writePkg(t, dir, "c-good", "search.sh", "#!/bin/sh\necho script-ok\n")
	pm := packages.NewPackageManager(dir)
	contributor := &fakeContributor{text: "hindsight-hit"}
	out := captureStdout(t, func() {
		result, err := RunSearchExtensions(context.Background(), pm, "aqueduct", 4, nil, contributor)
		if err != nil {
			t.Fatalf("search extensions: %v", err)
		}
		if !result.ScriptOK || !strings.Contains(result.Script, "script-ok") {
			t.Fatalf("script = %q", result.Script)
		}
		if result.Supplement != "hindsight-hit" {
			t.Fatalf("supplement = %q", result.Supplement)
		}
	})
	if !strings.Contains(out, "a-bad search extension failed") || !strings.Contains(out, "Running c-good") {
		t.Fatalf("output = %s", out)
	}
	if contributor.query != "aqueduct" || contributor.limit != 4 || contributor.calls != 1 {
		t.Fatalf("contributor = %+v", contributor)
	}
}

func TestSearchSupplementDegrades(t *testing.T) {
	contributor := &fakeContributor{err: errors.New("unreachable")}
	out := captureStdout(t, func() {
		result, err := RunSearchExtensions(context.Background(), nil, "q", 0, contributor)
		if err == nil || result.ScriptOK {
			t.Fatal("expected no package script")
		}
		if result.Supplement != "" {
			t.Fatalf("supplement = %q", result.Supplement)
		}
	})
	if !strings.Contains(out, "memory-hindsight search skipped") || !strings.Contains(out, "unreachable") {
		t.Fatalf("output = %s", out)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runContributorSearch(ctx, "q", 1, &fakeContributor{text: "late"}); err == nil {
		t.Fatal("canceled context should fail")
	}
	if text := SearchSupplement(context.Background(), "q", 1, &fakeContributor{text: "  kept  "}, &fakeContributor{text: "second"}); text != "kept\nsecond" {
		t.Fatalf("text = %q", text)
	}
}

func TestRunIndexExtensions(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, "idx", "index.sh", "#!/bin/sh\necho indexing\nexit 1\n")
	pm := packages.NewPackageManager(dir)
	contributor := &fakeContributor{indexErr: errors.New("down")}
	out := captureStdout(t, func() {
		if err := RunIndexExtensions(context.Background(), pm, []string{dir}, true, nil, contributor); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "index extension failed") || !strings.Contains(out, "memory-hindsight index skipped") {
		t.Fatalf("output = %s", out)
	}
	if !contributor.fast || len(contributor.roots) != 1 {
		t.Fatalf("index call = %+v", contributor)
	}
	if err := RunIndexExtensions(context.Background(), nil, nil, false); err != nil {
		t.Fatal(err)
	}
}

func writePkg(t *testing.T, cockpitDir, name, scriptName, script string) {
	t.Helper()
	dir := filepath.Join(cockpitDir, "packages", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "name: " + name + "\nversion: 0.0.1\nextensions:\n  kb:\n"
	if scriptName == "" {
		manifest += "    search: missing.sh\n"
	} else if strings.HasPrefix(scriptName, "index") {
		manifest += "    index: " + scriptName + "\n"
	} else {
		manifest += "    search: " + scriptName + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "cockpit-package.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if scriptName != "" {
		if err := os.WriteFile(filepath.Join(dir, scriptName), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
