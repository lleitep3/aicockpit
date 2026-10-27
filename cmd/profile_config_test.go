package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/zalando/go-keyring"
)

func unlockTestVault(t *testing.T) {
	t.Helper()
	if err := vault.NewLockManager("").Unlock("test"); err != nil {
		t.Fatal(err)
	}
}

func profileCommand(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()
	c := NewConfigCommand()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetIn(strings.NewReader(input))
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}

func TestConfigCLI(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "")
	keyring.MockInit()
	base := []string{"--namespace", "newrelic", "--profile", "dev"}
	run := func(input string, args ...string) (string, error) {
		return profileCommand(t, input, append(append([]string{}, base...), args...)...)
	}
	if _, err := run("", "set", "account", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("secret-value", "secret", "token", "--stdin"); err == nil {
		t.Fatal("locked secret write allowed")
	}
	unlockTestVault(t)
	if _, err := run("secret-value\n", "secret", "token", "--stdin"); err != nil {
		t.Fatal(err)
	}
	out, err := run("", "show")
	if err != nil || strings.Contains(out, "secret-value") || !strings.Contains(out, "profiles/dev/token") {
		t.Fatal(out, err)
	}
	out, err = run("", "list")
	if err != nil || !strings.Contains(out, "dev") {
		t.Fatal(out, err)
	}
	if _, err := run("", "grant", "consumer"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "consumer")
	if _, err := run("", "show"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"set", "account", "bad"}, {"secret", "token", "--stdin"}, {"grant", "third"}, {"revoke", "third"}} {
		if _, err := run("bad", args...); err == nil {
			t.Fatal("reader mutation allowed", args)
		}
	}
	if checkNamespaceAccess("", false) == nil {
		t.Fatal("package accessed legacy vault")
	}
	if err := checkNamespaceAccess("newrelic", false); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "")
	if _, err := run("", "revoke", "consumer"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "consumer")
	if _, err := run("", "show"); err == nil {
		t.Fatal("revoked reader allowed")
	}
	if checkNamespaceAccess("newrelic", false) == nil {
		t.Fatal("revoked vault reader allowed")
	}
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "newrelic")
	if _, err := profileCommand(t, "", "--profile", "dev", "show"); err != nil {
		t.Fatal(err)
	}
	if err := vault.NewLockManager("").Lock("test"); err != nil {
		t.Fatal(err)
	}
	if checkNamespaceAccess("newrelic", false) == nil {
		t.Fatal("namespace bypassed lock")
	}
	if _, err := run("", "show"); err != nil {
		t.Fatal("public metadata needs no vault", err)
	}
}

func TestConfigCLIValidation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "")
	keyring.MockInit()
	unlockTestVault(t)
	for _, args := range [][]string{{"show"}, {"--namespace", "newrelic", "show"}, {"--namespace", "../bad", "list"}, {"--namespace", "newrelic", "--profile", "dev", "secret", "../bad", "--stdin"}, {"--namespace", "newrelic", "--profile", "missing", "show"}, {"--namespace", "newrelic", "--profile", "dev", "exec", "echo"}} {
		if _, err := profileCommand(t, "", args...); err == nil {
			t.Fatal("invalid arguments accepted", args)
		}
	}
	for _, value := range []string{"", "a\x00b", strings.Repeat("x", 65537)} {
		if _, err := profileCommand(t, value, "--namespace", "newrelic", "--profile", "dev", "secret", "token", "--stdin"); err == nil {
			t.Fatal("invalid secret accepted")
		}
	}
	if _, err := profileCommand(t, "", "--namespace", "newrelic", "--profile", "dev", "secret", "token"); err == nil {
		t.Fatal("no terminal accepted")
	}
}

func TestConfigExecChildOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "")
	keyring.MockInit()
	unlockTestVault(t)
	if _, err := profileCommand(t, "child-private", "--namespace", "newrelic", "--profile", "dev", "secret", "token", "--stdin"); err != nil {
		t.Fatal(err)
	}
	// Use the test binary as a cross-platform child. It checks the injected
	// value and emits only a boolean result, not the secret itself.
	t.Setenv("COCKPIT_PROFILE_HELPER", "1")
	args := []string{"--namespace", "newrelic", "--profile", "dev", "exec", "--env", "TEST_PROFILE_SECRET=token", "--", os.Args[0], "-test.run=^TestProfileChild$"}
	out, err := profileCommand(t, "", args...)
	if err != nil || !strings.Contains(out, "child-ok") {
		t.Fatal(out, err)
	}
	if os.Getenv("TEST_PROFILE_SECRET") != "" {
		t.Fatal("parent environment modified")
	}
	if strings.Contains(out, "child-private") {
		t.Fatal("secret printed")
	}
	if _, err := profileCommand(t, "", append(args[:len(args)-2], "missing-executable-xyz")...); err == nil {
		t.Fatal("missing executable success")
	}
	if err := vault.NewLockManager("").Lock("test"); err != nil {
		t.Fatal(err)
	}
	if _, err := profileCommand(t, "", args...); err == nil {
		t.Fatal("locked execution allowed")
	}
}

func TestProfileChild(t *testing.T) {
	if os.Getenv("COCKPIT_PROFILE_HELPER") != "1" {
		return
	}
	if os.Getenv("TEST_PROFILE_SECRET") != "child-private" {
		os.Exit(21)
	}
	_, _ = os.Stdout.WriteString("child-ok\n")
	os.Exit(0)
}

func TestVaultNamespaceAndStdin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "newrelic")
	keyring.MockInit()
	log, cfg, tr := newTestDeps(t)
	for _, operation := range []string{"get", "set", "remove", "list"} {
		c := NewVaultCommand(log, cfg, tr)
		args := []string{operation, "--namespace", "newrelic"}
		if operation != "list" {
			args = append(args, "token")
		}
		if operation == "set" {
			args = append(args, "--stdin")
		}
		c.SetArgs(args)
		c.SetIn(strings.NewReader("private"))
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		if err := c.Execute(); err == nil {
			t.Fatal("lock bypass", operation)
		}
	}
	unlockTestVault(t)
	c := NewVaultSetCommand(log, cfg, tr)
	c.SetArgs([]string{"token", "--namespace", "newrelic", "--stdin"})
	c.SetIn(strings.NewReader("private\n"))
	var out bytes.Buffer
	c.SetOut(&out)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "private") {
		t.Fatal("secret printed")
	}
	value, err := vault.NewNamespacedVault("newrelic").Get("token")
	if err != nil || value != "private" {
		t.Fatal("secret not saved")
	}
	c = NewVaultSetCommand(log, cfg, tr)
	c.SetArgs([]string{"token", "--namespace", "newrelic", "--stdin", "--value", "bad"})
	if c.Execute() == nil {
		t.Fatal("conflicting secret input")
	}
}

func TestConfigCorruptionAndLockFailures(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COCKPIT_PACKAGE_CONTEXT", "")
	keyring.MockInit()
	s := packageConfigStore()
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "newrelic.json"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := profileCommand(t, "", "--namespace", "newrelic", "list"); err == nil {
		t.Fatal("corrupt list")
	}
	orig := profileVaultAccess
	profileVaultAccess = func(string) error { return errors.New("locked") }
	t.Cleanup(func() { profileVaultAccess = orig })
	if _, err := profileCommand(t, "", "--namespace", "other", "grant", "newrelic"); err == nil {
		t.Fatal("locked grant")
	}
}
