package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Avoid thundering herd problem on remote services used by this command.
// maxDelay will be 0 if this is not an issue.
func SpreadWait(maxDelay time.Duration) {
	if maxDelay > 0 {
		// Seed random generator with current process ID
		rand.Seed(int64(os.Getpid()))
		// Sleep for random amount of time within maxDelay
		time.Sleep(time.Duration(rand.Int63n(int64(maxDelay))))
	}
}

// Ok states that execution went well. Logs debug output and reports ok to
// monitoring.
func Ok() {
	var message string
	log.Println("OK")
	if firstbytes == nil {
		message = "OK"
	} else {
		message = string(firstbytes.Bytes())
	}
	monitor(monitorOk, message)
}

// NotAvailable states that the command could not be started successfully. It
// might not be installed or has other problems.
func NotAvailable(err error) {
	s := fmt.Sprintln("Cannot start command: ", err)
	log.Println("FATAL:", s)
	monitor(monitorUnknown, s)
}

// TimedOut states that the command took too long and reports failure to the
// monitoring.
func TimedOut(err error) {
	s := fmt.Sprintln(err)
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

// Busy states that the command hangs and reports failure to the monitoring.
// Those tasks should be automatically killed, if it happens often.
func Busy() {
	s := "previous invocation of command still running"
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

// Failed states that the command didn't execute successfully and reports
// failure to the monitoring. Also Logs error output.
func Failed(err error) {
	var message string
	code, s := error2exit(err)
	log.Println("FATAL:", s)
	if firstbytes == nil {
		message = s
	} else {
		message = string(firstbytes.Bytes())
	}
	monitor(code, message)
}

// Locked states that we could not get the lock.
func Locked(err error) {
	s := fmt.Sprintln("Failed to get lock: ", err)
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

var firstbytes *CapWriter

//hardtimer provides a hard deadline, after which cmd will not run anymore
func hardtimer(now time.Time, cmd *exec.Cmd, errc chan error) *time.Timer {
	return time.AfterFunc(opts.Timeout-time.Since(now), func() {
		errc <- &TimeoutError{
			after: opts.Timeout,
		}
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
			// FIXME(nightlyone) log the kill
		}
	})
}

func main() {
	var cmd *exec.Cmd
	var wg sync.WaitGroup

	log.SetFlags(0)
	args := parseFlags()

	// exit program after --help
	if len(args) == 0 {
		return
	}

	command := args[0]

	monitoringEvent = opts.MonitoringEvent
	if monitoringEvent == "" {
		// event for command /path/check_foo.sh will be check_foo
		monitoringEvent = filepath.Base(command)
		if ext := filepath.Ext(command); ext != "" {
			monitoringEvent = monitoringEvent[0 : len(monitoringEvent)-len(ext)]
		}
	}

	logger, err := getLogger(opts.UseSyslog)
	if err != nil {
		log.Fatal("FATAL: cannot contact syslog")
		return
	}

	loadMonitoringCommands()

	now := time.Now()

	SpreadWait(opts.MaxDelay)

	// central error code channel for asynchronous errors
	errc := make(chan error, 1)

	lock, err := createLock(opts.KillRunning)
	if err != nil {
		Locked(&LockError{
			name: string(lock),
			err:  err,
		})
		return
	}
	defer lock.Unlock()

	cmd = exec.Command(command, args[1:]...)
	err = connectOutputs(cmd, logger, &wg)
	if err == nil {
		timer := hardtimer(now, cmd, errc)
		go processLife(cmd, errc)
		err = <-errc
		// wait for output streams to finish in case we exit normally
		if _, ok := err.(*TimeoutError); !ok {
			wg.Wait()
		}
		timer.Stop()
	}

	if err == nil {
		// best case
		Ok()
	} else {
		// now handle any errors
		switch e := err.(type) {
		case *TimeoutError:
			// we should collect wait state for what we killed
			if cmd != nil && cmd.Process != nil {
				<-errc
				// wait for output streams to finish in case got killed
				wg.Wait()
			}
			TimedOut(e)
		case *NotAvailableError:
			NotAvailable(e)
		case *StartupError:
			NotAvailable(e)
		case *exec.ExitError:
			Failed(e)
		case *LockError:
			Locked(e)
		default:
			// is unknown error really a fail? Shouldn't happend anyway!
			Failed(e)
		}
	}
}
