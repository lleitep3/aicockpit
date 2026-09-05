package cmd

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	testHome, err := os.MkdirTemp("", "aicockpit-cmd-tests-")
	if err != nil {
		os.Exit(2)
	}
	_ = os.Setenv("HOME", testHome)
	keyring.MockInit()
	code := m.Run()
	_ = os.RemoveAll(testHome)
	os.Exit(code)
}
