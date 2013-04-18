package main

import (
	"log"
	"os/exec"
	"strings"
)

var monitoringCalls = map[monitoringResult]string{}
var monitoringEvent string

type monitoringResult int

const (
	monitorOk monitoringResult = iota
	monitorCritical
	monitorWarning
	monitorDebug
	monitorUnknown
	monitorLast  = monitorUnknown
	monitorFirst = monitorOk
)

var monitoringResults = map[monitoringResult]string{
	monitorOk:       "OK",
	monitorCritical: "CRITICAL",
	monitorWarning:  "WARNING",
	monitorDebug:    "DEBUG",
	monitorUnknown:  "UNKNOWN",
}

func (m monitoringResult) String() string {
	return monitoringResults[m]
}

// Hook for passive monitoring solution
func monitor(state monitoringResult, message string) {
	if _, exists := monitoringResults[state]; !exists {
		panic("unknown monitoring state")
	}

	call, exists := monitoringCalls[state]
	if !exists {
		return
	}

	call = strings.Replace(call, "%(event)", monitoringEvent, -1)
	call = strings.Replace(call, "%(state)", state.String(), -1)
	call = strings.Replace(call, "%(message)", message, -1)
	// do argument interpolation
	cmd := commander.Command("/bin/sh", "-c", call)
	err := cmd.Run()
	if err != nil {
		log.Fatalln("FATAL: Monitoring script failed, hope your monitoring system detects dead clients")
	}
}

// infrastructure for dependency injection for os.exec Command and run
type Executor interface {
	Run() error
}

type Commander interface {
	Command(name string, args ...string) Executor
}

var commander Commander = execCommander{}

// default implementations
type execExecutor struct {
	*exec.Cmd
}

type execCommander struct{}

func (e execCommander) Command(name string, args ...string) Executor {
	return execExecutor{exec.Command(name, args...)}
}
