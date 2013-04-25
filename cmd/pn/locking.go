package main

import (
	"github.com/nightlyone/lockfile"
	"os"
	"path/filepath"
)

// Create a new lock file. Ensures that only one of these command runs
// concurrently on this machine.  Also cleans up stale locks of dead instances.
func createLock(killRunning bool) (lockfile.Lockfile, error) {
	var zero lockfile.Lockfile

	filename := filepath.Join(os.TempDir(), "periodicnoise",
		monitoringEvent, monitoringEvent+".lock")

	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return zero, err
	}

	lock, err := lockfile.New(filename)
	if err != nil {
		return zero, err
	}

	if err := lock.TryLock(); err != nil {
		if err != lockfile.ErrBusy {
			return lock, err
		}

		if killRunning {
			process, err := lock.GetOwner()
			if err != nil {
				return zero, err
			}
			if err := process.Kill(); err != nil {
				return zero, err
			}
			// Remove old lock and create new one
			if err := lock.Unlock(); err != nil {
				return zero, err
			}
			if err := lock.TryLock(); err != nil {
				return zero, err
			}
		} else {
			return zero, err
		}
	}

	// Lock successfully created
	return lock, err
}
