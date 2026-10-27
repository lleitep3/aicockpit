package vault

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var ErrMigrationConfirmation = errors.New("vault state migration confirmation required")

// MigrateState converts the forgeable v1 state to a fresh, blocked v2 state.
// It intentionally does not read, write, or delete credential data.
func MigrateState(path string, keyStore KeyStore, confirmed bool) (string, error) {
	if path == "" {
		path = DefaultLockStatePath()
	}
	encryptor := NewStateEncryptorAt(path, keyStore)
	var backupPath string
	err := withStateFileLock(path, func() error {
		data, readErr := os.ReadFile(path)
		if errors.Is(readErr, os.ErrNotExist) {
			if !confirmed {
				return ErrMigrationConfirmation
			}
			if err := encryptor.initializeLockedState(); err != nil {
				return err
			}
			if _, err := encryptor.DecryptAndVerify(); err != nil {
				return fmt.Errorf("failed to verify initialized lock state: %w", err)
			}
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("failed to read legacy lock state: %w", readErr)
		}
		if len(data) > maxStateFileSize {
			return fmt.Errorf("lock state exceeds maximum size")
		}
		legacy, err := encryptor.readEnvelope()
		if err != nil {
			return fmt.Errorf("cannot classify lock state for migration: %w", err)
		}
		if legacy.Version == stateVersionV2 {
			if _, err := encryptor.DecryptAndVerify(); err != nil {
				return fmt.Errorf("existing v2 lock state is unavailable: %w", err)
			}
			return nil
		}
		if legacy.Version != "v1" && legacy.Salt == "" && legacy.Signature == "" {
			return fmt.Errorf("unsupported lock state version %q", legacy.Version)
		}
		if !confirmed {
			return ErrMigrationConfirmation
		}

		backupPath, err = createExclusiveBackup(path, data)
		if err != nil {
			return err
		}
		if err := encryptor.initializeLockedState(); err != nil {
			return fmt.Errorf("failed to write migrated lock state: %w", err)
		}
		if _, err := encryptor.DecryptAndVerify(); err != nil {
			_ = atomicWriteFile(path, data, 0o600)
			return fmt.Errorf("failed to verify migrated lock state: %w", err)
		}
		return nil
	})
	if err != nil {
		return backupPath, err
	}
	if backupPath == "" {
		return "", nil
	}
	return backupPath, nil
}

func createExclusiveBackup(path string, data []byte) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create migration directory: %w", err)
	}
	file, err := os.CreateTemp(dir, filepath.Base(path)+".v1-backup-*")
	if err != nil {
		return "", fmt.Errorf("failed to create legacy state backup: %w", err)
	}
	name := file.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(name)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("failed to secure legacy state backup: %w", err)
	}
	if _, err := io.Copy(file, bytes.NewReader(data)); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("failed to write legacy state backup: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("failed to sync legacy state backup: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("failed to close legacy state backup: %w", err)
	}
	keep = true
	return name, nil
}
