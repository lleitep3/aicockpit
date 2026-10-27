package agents

import (
	"os"
	"path/filepath"
)

func writeDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeTempFile(t interface {
	TempDir() string
	Helper()
}, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		panic(err)
	}
	return path
}

func joinParts(parts []string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += "\x00"
		}
		out += part
	}
	return out
}
