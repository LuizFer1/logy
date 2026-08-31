package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrAlreadyRunning is returned when another daemon holds the lock.
var ErrAlreadyRunning = errors.New("logy daemon already running")

// Lock is an exclusive daemon lock file.
type Lock struct {
	path string
	file *os.File
}

// AcquireLock takes an exclusive lock at path.
func AcquireLock(path string) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := exclusiveLock(file); err != nil {
		file.Close()
		return nil, err
	}
	if err := file.Truncate(0); err != nil {
		unlock(file)
		file.Close()
		return nil, fmt.Errorf("truncate lock file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		unlock(file)
		file.Close()
		return nil, fmt.Errorf("seek lock file: %w", err)
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		unlock(file)
		file.Close()
		return nil, fmt.Errorf("write lock pid: %w", err)
	}
	return &Lock{path: path, file: file}, nil
}

// Release drops the exclusive lock.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlock(l.file)
	err := l.file.Close()
	l.file = nil
	return err
}
