package main

import (
	"flag"
	"fmt"
	"github.com/nightlyone/lockfile"
	"io"
	"log"
	"log/syslog"
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
	log.Println("Ok")
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

var useSyslog bool

// derive logger
func getLogger() (logger io.Writer, err error) {
	if useSyslog {
		logger, err = syslog.New(syslog.LOG_NOTICE, monitoringEvent)
	} else {
		logger = os.Stderr
	}
	if err != nil {
		log.SetOutput(logger)
	}
	return logger, err
}

// pipe r to logger in the background
func logStream(r io.Reader, logger io.Writer, wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		for {
			if _, err := io.Copy(logger, r); err != nil {
				break
			}
		}
		wg.Done()
	}()
}

func main() {
	var cmd *exec.Cmd
	var interval, timeout time.Duration
	var wg sync.WaitGroup
	var pipeStderr, pipeStdout bool

	// FIXME(mlafeldt) add command-line options for kill or wait on busy
	// state
	log.SetFlags(0)

	flag.DurationVar(&interval, "i", -1,
		"set execution interval for command, e.g. 45s, 2m, 1h30m, default: 1/10 of timeout")
	flag.DurationVar(&timeout, "t", 1*time.Minute,
		"set execution timeout for command, e.g. 45s, 2m, 1h30m, default: 1m")
	flag.BoolVar(&useSyslog, "l", false, "log via syslog")
	flag.BoolVar(&pipeStderr, "e", true, "pipe stderr to log")
	flag.BoolVar(&pipeStdout, "o", true, "pipe stdout to log")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("FATAL: no command to execute")
		return
	}

	command := flag.Arg(0)
	monitoringEvent = filepath.Base(command)
	logger, err := getLogger()
	if err != nil {
		log.Fatal("FATAL: cannot contact syslog")
	}

	if interval >= timeout {
		log.Fatal("FATAL: interval >= timeout, no time left for actual command execution")
		return
	}

	if interval == -1 {
		interval = timeout / 10
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
	lock_dir := os.TempDir()
	os.Mkdir(filepath.Join(lock_dir, monitoringEvent), 0700)
	lock, _ := lockfile.New(filepath.Join(lock_dir, monitoringEvent, monitoringEvent+".lock"))
	if err := lock.TryLock(); err != nil {
		if err != lockfile.ErrBusy {
			log.Printf("ERROR: locking %s: reason: %v\n", lock, err)
		}
		timer.Stop()
		Busy()
		return
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
