package memoryhindsight

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/memory"
)

func TestResolveBanks(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/acme/Demo_App\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("a", 80)

	tests := []struct {
		name    string
		id      Identity
		cfg     hindsight.BanksConfig
		project string
		user    string
		wantErr error
	}{
		{
			name:    "module and user",
			id:      Identity{WorkDir: dir, Username: "Ada Lovelace"},
			project: "project-demo_app",
			user:    "user-ada-lovelace",
		},
		{
			name:    "configured names",
			id:      Identity{Username: "ignored"},
			cfg:     hindsight.BanksConfig{Project: "My App", User: "carol"},
			project: "my-app",
			user:    "carol",
		},
		{
			name:    "directory when module is absent",
			id:      Identity{WorkDir: filepath.Join(dir, "Local Project"), Username: "bob"},
			project: "project-local-project",
			user:    "user-bob",
		},
		{
			name:    "fallbacks",
			id:      Identity{},
			project: "project-workspace",
			user:    "user-unknown",
		},
		{
			name:    "reserved project name",
			cfg:     hindsight.BanksConfig{Project: "project", User: "carol"},
			wantErr: memory.ErrInvalidBank,
		},
		{
			name:    "same bank",
			cfg:     hindsight.BanksConfig{Project: "shared", User: "shared"},
			wantErr: errors.New("must differ"),
		},
		{
			name:    "long derived name is capped",
			id:      Identity{Username: long},
			cfg:     hindsight.BanksConfig{Project: "crew"},
			project: "crew",
			user:    "user-" + strings.Repeat("a", 59),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveBanks(tt.id, tt.cfg)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error")
				}
				if errors.Is(tt.wantErr, memory.ErrInvalidBank) {
					if !errors.Is(err, memory.ErrInvalidBank) {
						t.Fatalf("error = %v", err)
					}
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveBanks: %v", err)
			}
			if got.Project != tt.project || got.User != tt.user {
				t.Fatalf("banks = %+v, want project %s user %s", got, tt.project, tt.user)
			}
			if len(got.Project) > maxBankLen || len(got.User) > maxBankLen {
				t.Fatalf("bank longer than %d: %+v", maxBankLen, got)
			}
		})
	}
}

func TestCurrentIdentity(t *testing.T) {
	t.Setenv("USER", "bank-tester")
	id := CurrentIdentity()
	if id.WorkDir == "" && id.Username == "" {
		t.Fatal("identity is empty")
	}
}

func TestValidateAPIURL(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "http://127.0.0.1:8888"},
		{in: "https://memory.example"},
		{in: "ftp://127.0.0.1", wantErr: true},
		{in: "http://user:pass@127.0.0.1", wantErr: true},
		{in: "not a url", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		err := validateAPIURL(tt.in)
		if tt.wantErr && err == nil {
			t.Errorf("%q: expected error", tt.in)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%q: %v", tt.in, err)
		}
	}
}
