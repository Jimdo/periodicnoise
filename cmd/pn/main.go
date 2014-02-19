package main

import (
	"log"
	"os/exec"
	"path/filepath"
)

var code2prefix = map[monitoringResult]string{
	monitorOk:       "INFO",
	monitorWarning:  "WARNING",
	monitorCritical: "FATAL",
	monitorUnknown:  "FATAL",
	monitorDebug:    "DEBUG",
}

func report(err error) monitoringResult {
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

	err = CoreLoopRetry(args, logger)
	report(err)
}
