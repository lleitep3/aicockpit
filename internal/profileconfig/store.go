// Package profileconfig stores public package settings and references to vault
// entries. Its access policy is a CLI boundary, not a sandbox for same-user code.
package profileconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
)

const Operator = "cockpit-cli"

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type Field struct {
	Value     *string `json:"value,omitempty"`
	SecretKey string  `json:"secret_key,omitempty"`
}

type Document struct {
	Version  int                         `json:"version"`
	Readers  []string                    `json:"readers"`
	Profiles map[string]map[string]Field `json:"profiles"`
}

type Store struct{ Dir string }

func ValidName(name string) bool { return identifier.MatchString(name) && name != Operator }

func (s Store) path(namespace string) (string, error) {
	if !ValidName(namespace) {
		return "", errors.New("invalid package namespace")
	}
	return filepath.Join(s.Dir, namespace+".json"), nil
}

func (s Store) load(namespace string) (Document, error) {
	empty := Document{Version: 1, Readers: []string{}, Profiles: map[string]map[string]Field{}}
	path, err := s.path(namespace)
	if err != nil {
		return empty, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return empty, nil
	}
	if err != nil {
		return empty, errors.New("cannot inspect package configuration")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return empty, errors.New("configuration permissions must be 0600")
	}
	if !info.Mode().IsRegular() {
		return empty, errors.New("configuration must be a regular file")
	}
	if info.Size() > 1024*1024 {
		return empty, errors.New("configuration exceeds 1 MiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty, errors.New("cannot read package configuration")
	}
	var doc Document
	if json.Unmarshal(data, &doc) != nil || doc.Version != 1 || doc.Profiles == nil || doc.Readers == nil {
		return empty, errors.New("invalid package configuration; restore a valid version before retrying")
	}
	for _, reader := range doc.Readers {
		if !ValidName(reader) {
			return empty, errors.New("invalid reader in configuration")
		}
	}
	for profile, fields := range doc.Profiles {
		if !ValidName(profile) || fields == nil {
			return empty, errors.New("invalid profile in configuration")
		}
		for key, field := range fields {
			if !ValidName(key) || (field.Value == nil) == (field.SecretKey == "") {
				return empty, errors.New("invalid field in configuration")
			}
			if field.SecretKey != "" && field.SecretKey != SecretKey(profile, key) {
				return empty, errors.New("invalid vault reference in configuration")
			}
		}
	}
	return doc, nil
}

func authorize(doc Document, actor, namespace string, write bool) error {
	if actor == Operator || actor == namespace {
		return nil
	}
	if ValidName(actor) && !write && slices.Contains(doc.Readers, actor) {
		return nil
	}
	return errors.New("namespace access denied; request an explicit read grant from the operator")
}

func (s Store) Check(actor, namespace string, write bool) error {
	doc, err := s.load(namespace)
	if err != nil {
		return err
	}
	return authorize(doc, actor, namespace, write)
}

func (s Store) Read(actor, namespace string) (Document, error) {
	doc, err := s.load(namespace)
	if err != nil {
		return doc, err
	}
	if err := authorize(doc, actor, namespace, false); err != nil {
		return Document{}, err
	}
	return doc, nil
}

// update serializes read-modify-write across processes. A stale lock after a
// crash intentionally fails closed; an operator must reconcile it before retry.
func (s Store) update(namespace string, change func(*Document) error) error {
	path, err := s.path(namespace)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return errors.New("cannot create configuration directory")
	}
	info, err := os.Lstat(s.Dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("invalid configuration directory")
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("configuration is busy; if a previous process crashed, reconcile its .lock file")
	}
	defer os.Remove(path + ".lock")
	if err := lock.Close(); err != nil {
		return errors.New("cannot close configuration lock")
	}
	doc, err := s.load(namespace)
	if err != nil {
		return err
	}
	if err := change(&doc); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return errors.New("cannot encode configuration")
	}
	if len(data) > 1024*1024 {
		return errors.New("configuration exceeds 1 MiB")
	}
	f, err := os.CreateTemp(s.Dir, ".profile-*")
	if err != nil {
		return errors.New("cannot create configuration file")
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("cannot persist configuration")
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return errors.New("cannot commit configuration")
	}
	return nil
}

func SecretKey(profile, key string) string { return "profiles/" + profile + "/" + key }

func (s Store) Set(actor, namespace, profile, key string, field Field) error {
	if !ValidName(profile) || !ValidName(key) {
		return errors.New("invalid profile or field name")
	}
	if (field.Value == nil) == (field.SecretKey == "") {
		return errors.New("provide a public value or vault reference, never both")
	}
	if field.SecretKey != "" && field.SecretKey != SecretKey(profile, key) {
		return errors.New("vault reference must belong to this profile and field")
	}
	return s.update(namespace, func(doc *Document) error {
		if err := authorize(*doc, actor, namespace, true); err != nil {
			return err
		}
		if doc.Profiles[profile] == nil {
			doc.Profiles[profile] = map[string]Field{}
		}
		if previous, ok := doc.Profiles[profile][key]; ok && (previous.Value == nil) != (field.Value == nil) {
			return errors.New("field type cannot change between public and secret")
		}
		doc.Profiles[profile][key] = field
		return nil
	})
}

func (s Store) Grant(actor, namespace, reader string, allow bool) error {
	if actor != Operator {
		return errors.New("only the operator may change grants")
	}
	if !ValidName(reader) || reader == namespace {
		return errors.New("invalid reader package")
	}
	return s.update(namespace, func(doc *Document) error {
		doc.Readers = slices.DeleteFunc(doc.Readers, func(v string) bool { return v == reader })
		if allow {
			doc.Readers = append(doc.Readers, reader)
		}
		slices.Sort(doc.Readers)
		return nil
	})
}

func (s Store) Profile(actor, namespace, profile string) (map[string]Field, error) {
	doc, err := s.Read(actor, namespace)
	if err != nil {
		return nil, err
	}
	fields, ok := doc.Profiles[profile]
	if !ok {
		return nil, fmt.Errorf("profile %q is not configured", profile)
	}
	return fields, nil
}
