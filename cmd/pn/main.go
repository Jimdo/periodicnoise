package main

import (
	"flag"
	"fmt"
	"io"
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

// LockError states that we could not get the lock.
func LockError(err error) {
	s := fmt.Sprintln("Failed to get lock: ", err)
	log.Println("FATAL:", s)
	monitor(monitorCritical, s)
}

var firstbytes *CapWriter
var interval, timeout time.Duration
var pipeStderr, pipeStdout bool
var useSyslog bool
var wrapNagios bool
var killRunning bool
var command string

func parseFlags() {
	flag.DurationVar(&interval, "i", -1,
		"set maximum execution start delay for command, e.g. 45s, 2m, 1h30m, default: 1/10 of timeout")
	flag.DurationVar(&timeout, "t", 1*time.Minute,
		"set execution timeout for command, e.g. 45s, 2m, 1h30m, default: 1m")
	flag.BoolVar(&useSyslog, "s", false, "log via syslog")
	flag.BoolVar(&wrapNagios, "n", false, "wrap nagios plugin (pass on return codes, pass first 8KiB of stdout as message)")
	flag.BoolVar(&pipeStderr, "e", true, "pipe stderr to log")
	flag.StringVar(&monitoringEvent, "E", "", "monitoring event (defaults to check_foo for /path/check_foo.sh ")
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

	// default check_foo for /path/check_foo.sh
	if monitoringEvent == "" {
		monitoringEvent = filepath.Base(command)
		if ext := filepath.Ext(command); ext != "" {
			monitoringEvent = monitoringEvent[0 : len(monitoringEvent)-len("."+ext)]
		}
	}

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

	lock, err := createLock(killRunning)
	if err != nil {
		timer.Stop()
		LockError(err)
		return
	}
	defer lock.Unlock()

	cmd = exec.Command(command, flag.Args()[1:]...)

	if pipeStdout {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		if wrapNagios {
			firstbytes = NewCapWriter(8192)
			stdout := io.TeeReader(stdout, firstbytes)
			logStream(stdout, logger, &wg)
		} else {
			logStream(stdout, logger, &wg)
		}
	} else if wrapNagios {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		firstbytes = NewCapWriter(8192)
		go io.Copy(firstbytes, stdout)
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
