package main

import (
	"fmt"
	"os"
	"time"
)

type StartupError struct {
	stage string
	err   error
}

func (e *StartupError) Error() string {
	return fmt.Sprintf("Startup phase: Cannot %s: %s", e.stage, e.err)
}

type NotAvailableError struct {
	args []string // arg[0] is the command
	err  error
}

func (e *NotAvailableError) Error() string {
	return fmt.Sprintf("Command %s not available: %s", e.args[0], e.err)
}

type TimeoutError struct {
	after time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("Hard timeout after %s, killed with %s", e.after, os.Kill)
}

type LockError struct {
	name string
	err  error
}

func (e *LockError) Error() string {
	return fmt.Sprintf("cannot get lockfile %s: %s", e.name, e.err)
}
