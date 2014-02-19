package main

import (
	"log"
	"path/filepath"
)

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
	Report(err)
}
