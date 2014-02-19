package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

var errorList = []struct {
	err    error
	logged string
	code   monitoringResult
}{
	{
		logged: "INFO: OK (considered OK for monitoring)\n",
	},
	{
		logged: "FATAL: Soft timeout after 5s, killed with " + GracefulSignal.String() + " (considered CRITICAL for monitoring)\n",
		code:   monitorCritical,
		err: &TimeoutError{
			soft:  true,
			after: 5 * time.Second,
		},
	},
	{
		logged: "FATAL: Command missing not available: missing command (considered UNKNOWN for monitoring)\n",
		code:   monitorUnknown,
		err: &NotAvailableError{
			args: []string{"missing"},
			err:  errors.New("missing command"),
		},
	},
	{
		logged: "FATAL: Startup phase: Cannot start: test error (considered UNKNOWN for monitoring)\n",
		code:   monitorUnknown,
		err: &StartupError{
			stage: "start",
			err:   errors.New("test error"),
		},
	},
	{
		logged: "FATAL: cannot get lockfile /tmp/periodicnoise-toaster: " + os.ErrPermission.Error() + " (considered CRITICAL for monitoring)\n",
		code:   monitorCritical,
		err: &LockError{
			name: "/tmp/periodicnoise-toaster",
			err:  os.ErrPermission,
		},
	},
	{
		logged: "FATAL: unexpected error, not yet handled (considered CRITICAL for monitoring)\n",
		code:   monitorCritical,
		err:    errors.New("unexpected error, not yet handled"),
	},
	{
		// This is a little bit strange: Go exec/wait functions returns error == nil for successful execution,
		// But we fake an artifical error where we cannot initialize the exit code. That's why it is 0.
		// Thus we ensure here that any error for exec.ExitError is actually considered CRITICAL.
		logged: "FATAL: exit status 0 (considered CRITICAL for monitoring)\n",
		code:   monitorCritical,
		err: &exec.ExitError{
			ProcessState: &os.ProcessState{},
		},
	},
}

func TestReport(t *testing.T) {
	oldopts := opts
	oldCommander := commander
	oldFlags := log.Flags()
	oldFirstBytes := firstbytes
	defer func() {
		opts = oldopts
		commander = oldCommander
		log.SetOutput(os.Stderr)
		log.SetFlags(oldFlags)
		firstbytes = oldFirstBytes
	}()

	commander = &ignorantCommanderExecutor{}
	log.SetFlags(0)
	firstbytes = nil
	for i, test := range errorList {
		var output bytes.Buffer
		log.SetOutput(&output)

		code := Report(test.err)
		logged := output.String()
		if test.code != code {
			t.Errorf("%d: monitoring codes: got %s, expected %s", i, code, test.code)
		} else {
			t.Logf("%d: monitoring codes: got %s", i, code)
		}
		if test.logged != logged {
			t.Errorf("%d: log lines: got %q, expected %q", i, logged, test.logged)
		} else {
			t.Logf("%d: log lines: got %q", i, logged)
		}
	}
}

// completely ignore monitoring
type ignorantCommanderExecutor struct{}

func (e *ignorantCommanderExecutor) Command(string, ...string) Executor { return e }
func (e *ignorantCommanderExecutor) Run() error                         { return nil }
