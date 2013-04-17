package main

import (
	"log"
	"os/exec"
	"strings"
)

var monitoringCalls map[monitoringResult] string
var monitoringEvent string

type monitoringResult string

const (
	monitorOk monitoringResult = "OK"
	monitorCritical monitoringResult = "CRITICAL"
	monitorWarning monitoringResult = "WARNING"
	monitorDebug monitoringResult = "DEBUG"
	monitorUnknown monitoringResult = "UNKNOWN"
)

// Hook for passive monitoring solution
func monitor(state monitoringResult, message string) {
	// sanity check arguments
	switch state {
	case monitorOk,monitorCritical,monitorWarning,monitorDebug,monitorUnknown:
		// These are valid
	default:
		panic("unknown monitoring state")
	}

	call, exists := monitoringCalls[state]
	if !exists {
		return
	}

	call = strings.Replace(call, "%(event)", monitoringEvent, -1)
	call = strings.Replace(call, "%(state)", string(state), -1)
	call = strings.Replace(call, "%(message)", message, -1)
	// do argument interpolation
	cmd := exec.Command("/bin/sh", "-c", call)
	err := cmd.Run()
	if err != nil {
		log.Fatalln("FATAL: Monitoring script failed, hope your monitoring system detects dead clients")
	}
}
