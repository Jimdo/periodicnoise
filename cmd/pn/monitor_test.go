package main

import (
	"strings"
	"testing"
)

func setup_monitoringCalls() {
	monitoringCalls = map[monitoringResult]string{
		monitorOk:       `send_nsca "%(event): [OK] %(message)"`,
		monitorCritical: `send_nsca "%(event): [CRITICAL] %(message)"`,
		monitorWarning:  `send_nsca "%(event): [WARNING] %(message)"`,
		monitorDebug:    `send_nsca "%(event): [DEBUG] %(message)"`,
		monitorUnknown:  `send_nsca "%(event): [UNKNOWN] %(message)"`,
	}
}

func TestMonitorOk(t *testing.T) {
	oldCalls := monitoringCalls
	oldEvent := monitoringEvent
	oldCommander := commander
	defer func() {
		monitoringCalls = oldCalls
		monitoringEvent = oldEvent
		commander = oldCommander
	}()

	setup_monitoringCalls()
	monitoringEvent = "tests"
	ce := &mockCommanderExecutor{
		want: `/bin/sh -c send_nsca "tests: [OK] "`,
	}

	commander = Commander(ce)
	monitor(monitorOk, "")
	if ce.got != ce.want {
		t.Errorf("got '%v', want '%v'", ce.got, ce.want)
	} else {
		t.Logf("got '%v', want '%v'", ce.got, ce.want)
	}
}

// mock infrastructure for os.exec Command and run
type mockCommanderExecutor struct {
	got, want string
	xfail     error
}

func (e *mockCommanderExecutor) Command(name string, args ...string) Executor {
	cmd := []string{name}
	cmd = append(cmd, args...)
	e.got = strings.Join(cmd, " ")
	return e
}

func (e *mockCommanderExecutor) Run() error { return e.xfail }
