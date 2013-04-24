package main

import (
	"github.com/nightlyone/lockfile"
	"os"
	"path/filepath"
)

// Create a new lock file
func createLock() (lockfile.Lockfile, error) {
	filename := filepath.Join(os.TempDir(), "periodicnoise", monitoringEvent, monitoringEvent+".lock")

	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return lockfile.Lockfile(""), err
	}

	return lockfile.New(filename)
}
