package main

import (
	"strings"
	"testing"
)

func setup_monitoringCalls() {
	monitoringCalls = map[monitoringResult]string{
		monitorOk:       `echo -e "somehost.example.com;%(event);0;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
		monitorWarning:  `echo -e "somehost.example.com;%(event);1;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
		monitorCritical: `echo -e "somehost.example.com;%(event);2;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
		monitorUnknown:  `echo -e "somehost.example.com;%(event);3;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
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
		want: `/bin/sh -c echo -e "somehost.example.com;tests;0;OK\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
	}

	commander = Commander(ce)
	monitor(monitorOk, "OK")
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
