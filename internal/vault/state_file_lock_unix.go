//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type stateFileLock struct {
	file *os.File
}

func acquireStateFileLock(path string) (*stateFileLock, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create lock-state directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to secure lock-state directory: %w", err)
	}
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock-state lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to acquire lock-state lock: %w", err)
	}
	return &stateFileLock{file: file}, nil
}

func (lock *stateFileLock) Close() error {
	if err := unix.Flock(int(lock.file.Fd()), unix.LOCK_UN); err != nil {
		_ = lock.file.Close()
		return err
	}
	return lock.file.Close()
}
