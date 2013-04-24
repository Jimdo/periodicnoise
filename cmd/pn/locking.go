package main

import (
	"github.com/nightlyone/lockfile"
	"os"
	"path/filepath"
)

// Get name of lock file, which is derived from the monitoring event name
func getLockfileName() string {
	return filepath.Join(os.TempDir(), monitoringEvent, monitoringEvent+".lock")
}

// Create a new lock file
func createLock() (lockfile.Lockfile, error) {
	filename := getLockfileName()
	os.Mkdir(filepath.Dir(filename), 0700)
	return lockfile.New(filename)
}
