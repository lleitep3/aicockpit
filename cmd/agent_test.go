package cmd

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFindAgentDesktopDir(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, "internal", "personalagent", "desktop")
	if err := os.MkdirAll(desktop, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(desktop, "package.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		workingDir    string
		executableDir string
		configuredDir string
		want          string
		wantErr       string
	}{
		{
			name:       "discovers checkout from working directory",
			workingDir: filepath.Join(root, "cmd"),
			want:       desktop,
		},
		{
			name:          "discovers checkout from executable directory",
			workingDir:    t.TempDir(),
			executableDir: filepath.Join(root, "bin"),
			want:          desktop,
		},
		{
			name:          "configured directory takes precedence",
			workingDir:    t.TempDir(),
			executableDir: t.TempDir(),
			configuredDir: desktop,
			want:          desktop,
		},
		{
			name:       "reports actionable error when missing",
			workingDir: t.TempDir(),
			wantErr:    "COCKPIT_AGENT_DESKTOP_DIR",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := findAgentDesktopDir(test.workingDir, test.executableDir, test.configuredDir)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("findAgentDesktopDir() error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("findAgentDesktopDir() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestAgentCommandRejectsConflictingInterfaces(t *testing.T) {
	command := NewAgentCommand()
	command.SetArgs([]string{"--desktop", "--tui"})
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "--desktop and --tui cannot be used together") {
		t.Fatalf("Execute() error = %v, want conflicting interface error", err)
	}
}
