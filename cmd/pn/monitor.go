package main

import (
	"log"
	"os/exec"
	"strings"
	"syscall"
)

var monitoringCalls = map[monitoringResult]string{}
var monitoringEvent string

type monitoringResult int

const (
	monitorOk monitoringResult = iota
	monitorWarning
	monitorCritical
	monitorUnknown
	monitorDebug
	monitorLast  = monitorDebug
	monitorFirst = monitorOk
)

/* return codes besides Sucess and failure are unix specific, so only use there */
func error2exit(err error) (monitoringResult, string) {
	if err == nil {
		return monitorOk, "OK"
	}

	exiterr, ok := err.(*exec.ExitError)
	if !ok {
		return monitorUnknown, err.Error()
	}
	exitstate, ok := exiterr.Sys().(syscall.WaitStatus)
	if !ok {
		return monitorUnknown, err.Error()
	}
	switch exitstate.ExitStatus() {
	case 0, 1, 2, 3:
		return monitoringResult(exitstate.ExitStatus()), err.Error()
	}
	return monitorUnknown, err.Error()
}

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
