package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

// Report any error passed to monitoring and logging,
// returning the monitoring result code for further processing.
func Report(err error) monitoringResult {
	var message string
	code := monitorDebug

	switch err.(type) {
	case nil:
		code = monitorOk
		if firstbytes == nil {
			message = "OK"
		} else {
			message = string(firstbytes.Bytes())
		}
	case *TimeoutError:
		code = monitorCritical
		message = err.Error()
	case *NotAvailableError:
		code = monitorUnknown
		message = err.Error()
	case *StartupError:
		code = monitorUnknown
		message = err.Error()
	case *LockError:
		code = monitorCritical
		message = err.Error()
	case *exec.ExitError:
		res, s := error2exit(err)
		code = res
		if firstbytes == nil {
			message = s
		} else {
			message = string(firstbytes.Bytes())
		}
	default:
		code = monitorCritical
		message = err.Error()
	}
	log.Printf("%s: %s (considered %s for monitoring)\n", code2prefix[code], message, code)
	monitor(code, message)
	return code
}

var code2prefix = map[monitoringResult]string{
	monitorOk:       "INFO",
	monitorWarning:  "WARNING",
	monitorCritical: "FATAL",
	monitorUnknown:  "FATAL",
	monitorDebug:    "DEBUG",
}
