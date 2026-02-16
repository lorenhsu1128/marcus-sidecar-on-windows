//go:build windows

package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

// lockTimeout is the maximum time to wait for file lock acquisition (td-984ead)
const lockTimeout = 5 * time.Second

// lockRetryInterval is how often to retry lock acquisition
const lockRetryInterval = 10 * time.Millisecond

// acquireManifestLock acquires an advisory lock on the manifest file with timeout.
// exclusive=true for writes, false for reads.
// On Windows, uses LockFileEx for file locking.
func acquireManifestLock(path string, exclusive bool) (*os.File, error) {
	lockPath := path + ".lock"

	// Ensure directory exists for lock file
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}

	ol := new(windows.Overlapped)

	// Try non-blocking lock with timeout (td-984ead)
	deadline := time.Now().Add(lockTimeout)
	for {
		err := windows.LockFileEx(
			windows.Handle(lockFile.Fd()),
			flags,
			0,           // reserved
			1,           // lock 1 byte
			0,           // high bits
			ol,
		)
		if err == nil {
			return lockFile, nil
		}
		if time.Now().After(deadline) {
			_ = lockFile.Close()
			return nil, fmt.Errorf("lock acquisition timeout after %v", lockTimeout)
		}
		time.Sleep(lockRetryInterval)
	}
}

// releaseManifestLock releases the advisory lock.
func releaseManifestLock(lockFile *os.File) {
	if lockFile == nil {
		return
	}
	ol := new(windows.Overlapped)
	_ = windows.UnlockFileEx(
		windows.Handle(lockFile.Fd()),
		0,
		1,
		0,
		ol,
	)
	_ = lockFile.Close()
}
