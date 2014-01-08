package main

import (
	"errors"
	"github.com/nightlyone/lockfile"
	"os"
	"path/filepath"
)

// ErrNeedDirectory means the directory for the lock file actualy not a directory.
var ErrNeedDirectory = errors.New("lockfile directory not a directory")

// ErrNotExclusive means the directory for the lock file is not owned exclusively
// by the requester and thus vulnerable to symlink attacks.
var ErrNotExclusive = errors.New("lockfile directory not owned exclusively")

// create attack safe private directory
// if file creation fails there, then you there is only an ownership problem
// left, but this will be caught anyway now.
func privateSubdir(dirname string) error {
	if err := os.MkdirAll(dirname, 0700); err != nil {
		return err
	}

	fi, err := os.Lstat(dirname)
	if err != nil {
		return err
	}

	if mode := fi.Mode(); !mode.IsDir() {
		return ErrNeedDirectory
	} else if mode.Perm() != 0700 {
		return ErrNotExclusive
	}
	return nil
}

// Create a new lock file. Ensures that only one of these command runs
// concurrently on this machine.  Also cleans up stale locks of dead instances.
func createLock(killRunning bool) (lockfile.Lockfile, error) {
	var zero lockfile.Lockfile
	filename := filepath.Join(os.TempDir(), "periodicnoise-"+
		monitoringEvent, monitoringEvent+".lock")

	dirname := filepath.Dir(filename)
	if err := privateSubdir(dirname); err != nil {
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
			// FIXME(mlafeldt) Remove old lock. Not really safe as
			// the process might not be killed yet. lockfile.Unlock()
			// should check if it actually holds the lock.
			if err := lock.Unlock(); err != nil {
				return zero, err
			}
			// Create new lock
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
