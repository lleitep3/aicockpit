package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/lleitep3/aicockpit/internal/agents"
	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	testHome, err := os.MkdirTemp("", "aicockpit-cmd-tests-")
	if err != nil {
		os.Exit(2)
	}
	_ = os.Setenv("HOME", testHome)
	keyring.MockInit()
	// Doctor tests must not execute a developer machine's claude or codex.
	detectAgentCLIs = stubAgentCLIs
	code := m.Run()
	_ = os.RemoveAll(testHome)
	os.Exit(code)
}

func stubAgentCLIs(context.Context) []agents.Status {
	return []agents.Status{
		{ID: "claude", Name: "Claude CLI", Binary: "claude"},
		{ID: "codex", Name: "Codex CLI", Binary: "codex"},
	}
}
