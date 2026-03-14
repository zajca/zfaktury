package flock

import (
	"fmt"
	"os"
	"syscall"
)

// Lock represents an exclusive file lock.
type Lock struct {
	path string
	f    *os.File
}

// Acquire tries to obtain an exclusive lock on the given path.
// Returns an error if the lock is already held by another process.
func Acquire(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file %s: %w", path, err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another ZFaktury instance is already running (lock file: %s)", path)
	}

	// Write PID for debugging
	_ = f.Truncate(0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())

	return &Lock{path: path, f: f}, nil
}

// Release releases the lock and removes the lock file.
func (l *Lock) Release() error {
	if l.f == nil {
		return nil
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	_ = l.f.Close()
	_ = os.Remove(l.path)
	l.f = nil
	return nil
}
