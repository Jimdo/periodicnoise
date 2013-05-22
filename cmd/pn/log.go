package main

import (
	"io"
	"log"
	"log/syslog"
	"os"
	"sync"
)

// With Go 1.1, we can specify the syslog facility in addition to the priority:
// https://code.google.com/p/go/source/browse/src/pkg/log/syslog/syslog.go?name=go1.1
// To stay compatible with older Go versions, set the facility by hand for now.
const LOG_DAEMON = (3 << 3)

// derive logger
func getLogger(useSyslog bool) (logger io.Writer, err error) {
	if useSyslog {
		logger, err = syslog.New(LOG_DAEMON|syslog.LOG_NOTICE, monitoringEvent)
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
		for {
			if _, err := io.Copy(logger, r); err != nil {
				break
			}
		}
		wg.Done()
	}()
}
