//go:build windows

package flock

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
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

	if err := lockFileEx(syscall.Handle(f.Fd())); err != nil {
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
	_ = unlockFileEx(syscall.Handle(l.f.Fd()))
	_ = l.f.Close()
	_ = os.Remove(l.path)
	l.f = nil
	return nil
}

func lockFileEx(h syscall.Handle) error {
	ol := new(syscall.Overlapped)
	r1, _, err := procLockFileEx.Call(
		uintptr(h),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func unlockFileEx(h syscall.Handle) error {
	ol := new(syscall.Overlapped)
	r1, _, err := procUnlockFileEx.Call(
		uintptr(h),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}
