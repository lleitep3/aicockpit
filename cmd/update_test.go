package cmd

import (
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func TestNewUpdateCommand(t *testing.T) {
	log, err := logging.NewManager("/tmp/test-cockpit")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	cfg := &config.Config{
		Version:    "0.1.0",
		Language:   "en-us",
		LogLevel:   "info",
		AIProvider: "antigravity",
	}
	translator := i18n.New("en-us")

	cmd := NewUpdateCommand(log, cfg, translator)

	if cmd == nil {
		t.Fatal("NewUpdateCommand() returned nil")
	}

	if cmd.Use != "update" {
		t.Errorf("NewUpdateCommand() Use = %v, want %v", cmd.Use, "update")
	}
}
