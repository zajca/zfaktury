//go:build !windows

package flock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireAndRelease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	lock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Lock file should exist and contain PID.
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("lock file should contain PID")
	}

	// Release should succeed and remove the file.
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("lock file should be removed after release")
	}

	// Second release is a no-op.
	if err := lock.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}
}

func TestAcquire_AlreadyLocked(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	lock1, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer lock1.Release()

	// Second acquire should fail.
	_, err = Acquire(lockPath)
	if err == nil {
		t.Fatal("second Acquire should fail when lock is held")
	}
}

func TestAcquire_InvalidPath(t *testing.T) {
	_, err := Acquire("/nonexistent/dir/test.lock")
	if err == nil {
		t.Fatal("Acquire should fail for invalid path")
	}
}
