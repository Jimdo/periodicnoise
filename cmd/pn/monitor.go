package main

import (
	"log"
	"os/exec"
	"strconv"
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

// ForEachResultMapping iterate through the set of exit-code <=> monitoring severity mapping,
// calling the visitor function with each severity and code combination at least once.
// If visitor returns false, the iteration will be aborted.
func ForEachResultMapping(visitor func(severity monitoringResult, code uint8) bool) {
	monitoringCodes := map[monitoringResult][]uint8{
		monitorOk:       opts.MonitorOk,
		monitorWarning:  opts.MonitorWarning,
		monitorCritical: opts.MonitorCritical,
		monitorUnknown:  opts.MonitorUnknown,
		monitorDebug:    nil,
	}
	for severity, filter := range monitoringCodes {
		for _, code := range filter {
			if !visitor(severity, code) {
				return
			}
		}
	}
}

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
	status := uint8(exitstate.ExitStatus())
	result := monitorCritical
	if opts.WrapNagiosPlugin {
		result = monitorUnknown
	}

	ForEachResultMapping(func(severity monitoringResult, code uint8) bool {
		if status == code {
			result = severity
			return false
		}
		return true
	})
	return result, err.Error()
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

var shellEscaper = strings.NewReplacer(
	// pairs of replacements, s. http://godoc.org/strings#Replacer for details
	`$`, `\$`,
	"`", "\\`",
	"%", "%%",
)

func shellEscape(s string) string {
	res := shellEscaper.Replace(strconv.Quote(s))
	// undo the quote chars
	return res[1 : len(res)-1]
}

// Hook for passive monitoring solution
func monitor(state monitoringResult, message string) {
	if _, exists := monitoringResults[state]; !exists {
		panic("unknown monitoring state")
	}

	call, exists := monitoringCalls[state]
	if !exists || opts.NoMonitoring {
		return
	}

	call = strings.Replace(call, "%(event)", shellEscape(monitoringEvent), -1)
	call = strings.Replace(call, "%(state)", state.String(), -1)
	call = strings.Replace(call, "%(message)", shellEscape(message), -1)
	// do argument interpolation
	cmd := commander.Command("/bin/sh", "-c", call)
	err := cmd.Run()
	if err != nil {
		log.Fatalln("FATAL: Monitoring script failed with: ", err)
	}
}

// Executor provides infrastructure for dependency injection for os.exec Command and run
type Executor interface {
	Run() error
}

// Commander provides infrastructure for dependency injection for os.exec Command and run
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
