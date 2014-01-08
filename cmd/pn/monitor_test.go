package main

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func setupMonitoringCalls() {
	monitoringCalls = map[monitoringResult]string{
		monitorOk:       `printf "somehost.example.com;%(event);0;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
		monitorWarning:  `printf "somehost.example.com;%(event);1;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
		monitorCritical: `printf "somehost.example.com;%(event);2;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
		monitorUnknown:  `printf "somehost.example.com;%(event);3;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
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

	setupMonitoringCalls()
	monitoringEvent = "tests"
	ce := &mockCommanderExecutor{
		want: `/bin/sh -c printf "somehost.example.com;tests;0;OK\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"`,
	}

	commander = Commander(ce)
	monitor(monitorOk, "OK")
	if ce.got != ce.want {
		t.Errorf("got '%v', want '%v'", ce.got, ce.want)
	} else {
		t.Logf("got '%v', want '%v'", ce.got, ce.want)
	}
}

func TestNoMonitoring(t *testing.T) {
	oldCalls := monitoringCalls
	oldEvent := monitoringEvent
	oldCommander := commander
	oldOpts := opts
	defer func() {
		monitoringCalls = oldCalls
		monitoringEvent = oldEvent
		commander = oldCommander
		opts = oldOpts
	}()

	setupMonitoringCalls()
	opts.NoMonitoring = true
	monitoringEvent = "tests"
	ce := &mockCommanderExecutor{}

	commander = Commander(ce)
	monitor(monitorOk, "OK")
	if ce.got != ce.want {
		t.Errorf("got '%v', want '%v'", ce.got, ce.want)
	} else {
		t.Logf("got '%v', want '%v'", ce.got, ce.want)
	}
}

var escapeTest = [...]string{
	// stuff bash interprets, space
	"# & ; ` | * ? ~ < > ^ ( ) [ ] { } $ \x0a ' \" %",

	// stuff bash interprets, no space
	"#&;`|*?~<>^()[]{}$\x0a'\"%",

	// printf codes:
	"% (used to form %.0#-*+d, or \a \b \f \n \r \t \v \" \062 \062 \x32 \u0032 and \U00000032)",

	// extra case (bash and dash bet to differ here in certain cases)
	"\\",
}

func TestShellEscaping(t *testing.T) {
	oldCalls := monitoringCalls
	oldEvent := monitoringEvent
	oldCommander := commander
	oldOpts := opts
	defer func() {
		monitoringCalls = oldCalls
		monitoringEvent = oldEvent
		commander = oldCommander
		opts = oldOpts
	}()

	monitoringCalls = map[monitoringResult]string{
		monitorOk: `printf "somehost.example.com;%(event);0;%(message) \n" | cat`,
	}
	opts.NoMonitoring = false
	monitoringEvent = "escape"

	for i, sample := range escapeTest {
		ce := &capturingCommanderExecutor{
			want: "somehost.example.com;escape;0;" + sample + " \n",
		}

		commander = Commander(ce)

		monitor(monitorOk, sample)
		if !reflect.DeepEqual(ce.err, ce.xfail) {
			t.Errorf("%d: got error '%v', want '%v'", i, ce.err, ce.xfail)
		} else if ce.err != nil {
			t.Logf("%d: got expected error '%v'", i, ce.err)
		}

		if ce.got != ce.want {
			t.Errorf("%d: got %q, want %q", i, ce.got, ce.want)
		} else {
			t.Logf("%d: ok", i)
		}
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

//  version of executor capturing output and stderr
type capturingCommanderExecutor struct {
	got, want  string
	err, xfail error
	cmd        *exec.Cmd
}

func (e *capturingCommanderExecutor) Command(name string, args ...string) Executor {
	e.cmd = exec.Command(name, args...)
	return e
}

func (e *capturingCommanderExecutor) Run() error {
	res, err := e.cmd.CombinedOutput()
	if res != nil {
		e.got = string(res)
	} else {
		e.got = ""
	}
	e.err = err
	return err
}
