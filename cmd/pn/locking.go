package main

import (
	"fmt"
	"github.com/nightlyone/lockfile"
	"io/ioutil"
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

// FIXME(mlafeldt) this should be moved to the lockfile package
func getLockfilePid() (int, error) {
	var pid int

	content, err := ioutil.ReadFile(getLockfileName())
	if err != nil {
		return -1, err
	}

	_, err = fmt.Sscanln(string(content), &pid)
	if err != nil {
		return -1, err
	}

	return pid, nil
}

// FIXME(mlafeldt) this should be moved to the lockfile package
func getLockfileProcess() (*os.Process, error) {
	pid, err := getLockfilePid()
	if err != nil {
		return nil, err
	}

	return os.FindProcess(pid)
}
