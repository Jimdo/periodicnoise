package main

import (
	"flag"
	"fmt"
	"github.com/nightlyone/lockfile"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Avoid thundering herd problem on remote services used by this command.
// Interval will be 0, if this is not an issue.
func SpreadWait(interval time.Duration) {
	if interval > 0 {
		// Seed random generator with current process ID
		rand.Seed(int64(os.Getpid()))
		// Sleep for random amount of time within interval
		time.Sleep(time.Duration(rand.Int63n(int64(interval))))
	}
}

// Ok states that execution went well. Logs debug output and reports ok to
// monitoring.
func Ok() {
	log.Println("OK")
	monitor(monitorOk, "")
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
func TimedOut() {
	s := "execution took too long"
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
	s := fmt.Sprintln("Failed to execute: ", err)
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

var interval, timeout time.Duration
var pipeStderr, pipeStdout bool
var useSyslog bool
var killRunning bool
var command string

func parseFlags() {
	flag.DurationVar(&interval, "i", -1,
		"set execution interval for command, e.g. 45s, 2m, 1h30m, default: 1/10 of timeout")
	flag.DurationVar(&timeout, "t", 1*time.Minute,
		"set execution timeout for command, e.g. 45s, 2m, 1h30m, default: 1m")
	flag.BoolVar(&useSyslog, "s", false, "log via syslog")
	flag.BoolVar(&pipeStderr, "e", true, "pipe stderr to log")
	flag.BoolVar(&pipeStdout, "o", true, "pipe stdout to log")
	flag.BoolVar(&killRunning, "k", false, "kill already running instance of command")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("FATAL: no command to execute")
	}

	command = flag.Arg(0)

	if interval >= timeout {
		log.Fatal("FATAL: interval >= timeout, no time left for actual command execution")
	}

	if interval == -1 {
		interval = timeout / 10
	}
}

func main() {
	var cmd *exec.Cmd
	var wg sync.WaitGroup

	log.SetFlags(0)
	parseFlags()

	monitoringEvent = filepath.Base(command)
	logger, err := getLogger(useSyslog)
	if err != nil {
		log.Fatal("FATAL: cannot contact syslog")
		return
	}

	loadMonitoringCommands()

	// FIXME(nightlyone) try two intervals instead of one?
	timer := time.AfterFunc(timeout, func() {
		TimedOut()
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
			// FIXME(nightlyone) log the kill
		}
		os.Exit(0)
	})

	SpreadWait(interval)

	// Ensures that only one of these command runs concurrently on this
	// machine.  Also cleans up stale locks of dead instances.
	lock, err := createLock()
	if err != nil {
		log.Fatal(err)
	}
	if err := lock.TryLock(); err != nil {
		if err != lockfile.ErrBusy {
			log.Printf("ERROR: locking %s: reason: %v\n", lock, err)
		}

		if killRunning {
			process, err := getLockfileProcess()
			if err != nil {
				log.Fatal(err)
			}
			if err := process.Signal(os.Kill); err != nil {
				log.Fatal(err)
			}

			// Remove old lock and create new one
			lock.Unlock()
			if err := lock.TryLock(); err != nil {
				log.Fatal(err)
			}
		} else {
			timer.Stop()
			Busy()
			return
		}
	}
	defer lock.Unlock()

	cmd = exec.Command(command, flag.Args()[1:]...)

	if pipeStdout {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		logStream(stdout, logger, &wg)
	}

	if pipeStderr {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		logStream(stderr, logger, &wg)
	}

	if err := cmd.Start(); err != nil {
		timer.Stop()
		NotAvailable(err)
		return
	}

	if err := cmd.Wait(); err != nil {
		timer.Stop()
		Failed(err)
	} else {
		timer.Stop()
		Ok()
	}

	wg.Wait()
}
