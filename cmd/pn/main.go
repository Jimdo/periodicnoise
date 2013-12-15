package main

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

// Ok states that execution went well. Logs debug output and reports ok to
// monitoring.
func Ok() {
	var message string
	log.Println("OK")
	if firstbytes == nil {
		message = "OK"
	} else {
		message = string(firstbytes.Bytes())
	}
	monitor(monitorOk, message)
}

// NotAvailable states that the command could not be started successfully. It
// might not be installed or has other problems.
func NotAvailable(err error) {
	s := fmt.Sprint("Cannot start command: ", err)
	log.Println("FATAL:", s)
	monitor(monitorUnknown, s)
}

// TimedOut states that the command took too long and reports failure to the
// monitoring.
func TimedOut(err error) {
	s := fmt.Sprint(err)
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

// Busy states that the command hangs and reports failure to the monitoring.
// Those tasks should be automatically killed, if it happens often.
func Busy() {
	s := "previous invocation of command still running"
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

// Failed states that the command didn't execute successfully and reports
// failure to the monitoring. Also Logs error output.
func Failed(err error) {
	var message string
	code, s := error2exit(err)
	log.Println("FATAL:", s)
	if firstbytes == nil {
		message = s
	} else {
		message = string(firstbytes.Bytes())
	}
	monitor(code, message)
}

// Locked states that we could not get the lock.
func Locked(err error) {
	s := fmt.Sprint("Failed to get lock: ", err)
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

var firstbytes *CapWriter

func main() {
	log.SetFlags(0)
	args, e := parseFlags()

	if e != nil {
		log.Fatalf("FATAL: invalid arguments, %s", e)
		return
	}

	// exit program after --help
	if len(args) == 0 {
		return
	}

	monitoringEvent = opts.MonitoringEvent
	if monitoringEvent == "" {
		command := args[0]
		// event for command /path/check_foo.sh will be check_foo
		monitoringEvent = filepath.Base(command)
		if ext := filepath.Ext(command); ext != "" {
			monitoringEvent = monitoringEvent[0 : len(monitoringEvent)-len(ext)]
		}
	}

	logger, err := getLogger(opts.UseSyslog)
	if err != nil {
		log.Fatal("FATAL: cannot contact syslog")
		return
	}

	loadMonitoringCommands()

	err = CoreLoop(args, logger)
	if err == nil {
		// best case
		Ok()
	} else {
		// now handle any errors
		switch e := err.(type) {
		case *TimeoutError:
			TimedOut(e)
		case *NotAvailableError:
			NotAvailable(e)
		case *StartupError:
			NotAvailable(e)
		case *exec.ExitError:
			Failed(e)
		case *LockError:
			Locked(e)
		default:
			// is unknown error really a fail? Shouldn't happend anyway!
			Failed(e)
		}
	}
}
