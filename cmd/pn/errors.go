package main

import (
	"fmt"
	"os"
	"time"
)

// StartupError happens during startup phases
type StartupError struct {
	stage string
	err   error
}

func (e *StartupError) Error() string {
	return fmt.Sprintf("Startup phase: Cannot %s: %s", e.stage, e.err)
}

// NotAvailableError will be reported, when a command to be executed is not found
type NotAvailableError struct {
	args []string // arg[0] is the command
	err  error
}

func (e *NotAvailableError) Error() string {
	return fmt.Sprintf("Command %s not available: %s", e.args[0], e.err)
}

// TimeoutError happens when execution takes too long
type TimeoutError struct {
	soft  bool
	after time.Duration
}

func (e *TimeoutError) Error() string {
	if e.soft {
		return fmt.Sprintf("Soft timeout after %s, killed with %s", e.after, GracefulSignal)
	}
	return fmt.Sprintf("Hard timeout after %s, killed with %s", e.after, os.Kill)
}

// LockError happens, when the file base lock cannot be aquired
type LockError struct {
	name string
	err  error
}

func (e *LockError) Error() string {
	return fmt.Sprintf("cannot get lockfile %s: %s", e.name, e.err)
}
