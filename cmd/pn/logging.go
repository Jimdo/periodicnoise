package main

import (
	"io"
	"log"
	"log/syslog"
	"os"
	"sync"
)

// derive logger
func getLogger(useSyslog bool) (logger io.Writer, err error) {
	if useSyslog {
		logger, err = syslog.New(syslog.LOG_NOTICE, monitoringEvent)
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
