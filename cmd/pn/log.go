package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/Jimdo/periodicnoise/syslog"
)

// derive logger
func getLogger(useSyslog bool) (logger io.Writer, err error) {
	if useSyslog {
		logger, err = syslog.New(syslog.LOG_DAEMON|syslog.LOG_NOTICE, monitoringEvent)
	} else {
		logger = os.Stderr
		log.SetPrefix(monitoringEvent + ": ")
	}
	if err == nil {
		log.SetOutput(logger)
	}
	return &LineWriter{w: logger}, err
}

// pipe r to logger in the background
func logStream(r io.Reader, logger io.Writer, wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		// Retry writing to logstream until it can be fully written.
		// If io.Copy returns nil, everything has been copied
		// successfully and this routine can be stopped.
		for {
			if _, err := io.Copy(logger, r); err == nil {
				break
			} else {
				fmt.Fprintln(os.Stderr, "pn log error:", err)
			}
		}
		wg.Done()
	}()
}
