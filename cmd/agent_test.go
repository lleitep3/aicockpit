package cmd

import (
	"path/filepath"
	"testing"
)

func TestResolveAgentWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name      string
		workspace string
		want      string
	}{
		{
			name: "defaults to cockpit home",
			want: filepath.Join(home, ".cockpit", "workspace"),
		},
		{
			name:      "preserves explicit workspace",
			workspace: filepath.Join(home, "projects", "agent"),
			want:      filepath.Join(home, "projects", "agent"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveAgentWorkspace(test.workspace); got != test.want {
				t.Fatalf("resolveAgentWorkspace(%q) = %q, want %q", test.workspace, got, test.want)
			}
		})
	}
}
