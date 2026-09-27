package profileconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func public(v string) Field { return Field{Value: &v} }

func TestProfilesAndGrants(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if err := s.Set("newrelic", "newrelic", "dev", "account", public("42")); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("newrelic", "newrelic", "prod", "account", public("99")); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("newrelic", "newrelic", "dev", "api_key", Field{SecretKey: SecretKey("dev", "api_key")}); err != nil {
		t.Fatal(err)
	}
	for _, actor := range []string{"other", "", "../bad"} {
		if s.Check(actor, "newrelic", false) == nil {
			t.Fatal("unauthorized access", actor)
		}
	}
	if s.Grant("newrelic", "newrelic", "other", true) == nil {
		t.Fatal("package may not delegate")
	}
	if err := s.Grant(Operator, "newrelic", "other", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Grant(Operator, "newrelic", "other", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Check("other", "newrelic", false); err != nil {
		t.Fatal(err)
	}
	if s.Set("other", "newrelic", "dev", "account", public("bad")) == nil {
		t.Fatal("reader wrote")
	}
	doc, err := s.Read("other", "newrelic")
	if err != nil || len(doc.Readers) != 1 {
		t.Fatal(doc, err)
	}
	fields, err := s.Profile("other", "newrelic", "prod")
	if err != nil || *fields["account"].Value != "99" {
		t.Fatal(fields, err)
	}
	if err := s.Grant(Operator, "newrelic", "other", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("other", "newrelic"); err == nil {
		t.Fatal("revocation ignored")
	}
	if _, err := s.Profile("other", "newrelic", "dev"); err == nil {
		t.Fatal("revocation ignored")
	}
	if _, err := s.Profile("newrelic", "newrelic", "missing"); err == nil {
		t.Fatal("missing profile accepted")
	}
	if s.Set("newrelic", "newrelic", "dev", "api_key", public("plaintext")) == nil {
		t.Fatal("secret downgraded")
	}
	if s.Set("newrelic", "newrelic", "dev", "account", Field{SecretKey: SecretKey("dev", "account")}) == nil {
		t.Fatal("public field changed type")
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, "newrelic.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "plaintext") {
		t.Fatal("secret leaked")
	}
}

func TestValidation(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	for _, name := range []string{"../x", "a/b", "a:b", "UPPER", "", Operator, strings.Repeat("x", 65)} {
		if s.Check("newrelic", name, false) == nil {
			t.Fatal("namespace accepted", name)
		}
		if s.Set("newrelic", "newrelic", name, "key", public("v")) == nil {
			t.Fatal("profile accepted", name)
		}
		if s.Set("newrelic", "newrelic", "dev", name, public("v")) == nil {
			t.Fatal("field accepted", name)
		}
		if s.Grant(Operator, "newrelic", name, true) == nil {
			t.Fatal("reader accepted", name)
		}
	}
	if s.Grant(Operator, "newrelic", "newrelic", true) == nil {
		t.Fatal("self grant")
	}
	for _, f := range []Field{{}, {SecretKey: "other:token"}, {Value: public("v").Value, SecretKey: "key"}} {
		if s.Set("newrelic", "newrelic", "dev", "key", f) == nil {
			t.Fatal("invalid field")
		}
	}
}

func TestMalformedStorage(t *testing.T) {
	for _, data := range []string{"{", `null`, `{"version":2,"readers":[],"profiles":{}}`, `{"version":1,"readers":["../bad"],"profiles":{}}`, `{"version":1,"readers":[],"profiles":{"../bad":{}}}`, `{"version":1,"readers":[],"profiles":{"dev":null}}`, `{"version":1,"readers":[],"profiles":{"dev":{"key":{}}}}`, `{"version":1,"readers":[],"profiles":{"dev":{"key":{"secret_key":"other"}}}}}`, strings.Repeat("x", 1024*1024+1)} {
		s := Store{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(s.Dir, "newrelic.json"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Read(Operator, "newrelic"); err == nil {
			t.Fatal("corrupt metadata accepted")
		}
		if s.Set(Operator, "newrelic", "dev", "key", public("v")) == nil {
			t.Fatal("corruption overwritten")
		}
	}
}

func TestBusyAndFilesystemFailures(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(s.Dir, "newrelic.json.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if s.Set(Operator, "newrelic", "dev", "key", public("v")) == nil {
		t.Fatal("lock ignored")
	}
	if err := os.Mkdir(filepath.Join(s.Dir, "other.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(Operator, "other"); err == nil {
		t.Fatal("directory accepted")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	bad := Store{Dir: file}
	if bad.Set(Operator, "newrelic", "dev", "key", public("v")) == nil {
		t.Fatal("invalid directory accepted")
	}
	if _, err := bad.Read(Operator, "newrelic"); err == nil {
		t.Fatal("invalid directory read")
	}
	if s.Set(Operator, "../bad", "dev", "key", public("v")) == nil {
		t.Fatal("invalid namespace")
	}
}

func TestMetadataContainsOnlyReference(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if err := s.Set(Operator, "newrelic", "dev", "token", Field{SecretKey: SecretKey("dev", "token")}); err != nil {
		t.Fatal(err)
	}
	doc, err := s.Read(Operator, "newrelic")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(doc)
	if strings.Contains(string(encoded), `"value"`) {
		t.Fatal("secret value serialized")
	}
}

func TestEnvironment(t *testing.T) {
	fields := map[string]Field{"account": public("42"), "token": {SecretKey: "profiles/dev/token"}}
	get := func(key string) (string, error) {
		if key != "profiles/dev/token" {
			t.Fatal(key)
		}
		return "private-test-value", nil
	}
	env, err := Environment(fields, []string{"ACCOUNT=account", "TOKEN=token"}, []string{"PATH=/bin", "TOKEN=old", "token=old-lower"}, get)
	if err != nil || strings.Join(env, "|") != "PATH=/bin|ACCOUNT=42|TOKEN=private-test-value" {
		t.Fatal(env, err)
	}
	for _, bindings := range [][]string{{"bad"}, {"lower=account"}, {"COCKPIT_PACKAGE_CONTEXT=account"}, {"X=missing"}, {"X=account", "X=token"}} {
		if _, err := Environment(fields, bindings, nil, get); err == nil {
			t.Fatal("invalid bindings", bindings)
		}
	}
	_, err = Environment(fields, []string{"TOKEN=token"}, nil, func(string) (string, error) { return "", errors.New("sensitive-error-detail") })
	if err == nil || strings.Contains(err.Error(), "sensitive-error-detail") {
		t.Fatal("unsafe error", err)
	}
	if _, err := Environment(map[string]Field{"nul": public("a\x00b")}, []string{"X=nul"}, nil, get); err == nil {
		t.Fatal("NUL accepted")
	}
}

func TestPrivatePermissionsAndSize(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	if err := s.Set(Operator, "newrelic", "dev", "key", public("ok")); err != nil {
		t.Fatal(err)
	}
	if s.Set(Operator, "newrelic", "dev", "large", public(strings.Repeat("x", 1024*1024))) == nil {
		t.Fatal("oversized write accepted")
	}
	fields, err := s.Profile(Operator, "newrelic", "dev")
	if err != nil || *fields["key"].Value != "ok" {
		t.Fatal("failed write damaged profile")
	}
	if runtime.GOOS == "windows" {
		return
	}
	path := filepath.Join(s.Dir, "newrelic.json")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(Operator, "newrelic"); err == nil {
		t.Fatal("publicly readable policy accepted")
	}
	link := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(s.Dir, link); err != nil {
		t.Fatal(err)
	}
	if (Store{Dir: link}).Set(Operator, "newrelic", "dev", "key", public("v")) == nil {
		t.Fatal("symlink directory accepted")
	}
}
