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
	"time"
)

// FIXME(nightlyone) Hook up passive monitoring solution here
func monitor(state, msg string) {}

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
	monitor("OK", "")
}

// NotAvailable states that the command could not be started successfully. It
// might not be installed or has other problems.
func NotAvailable(err error) {
	s := fmt.Sprintln("Cannot start command: ", err)
	log.Println("FATAL:", s)
	monitor("UNKNOWN", s)
}

// TimedOut states that the command took too long and reports failure to the
// monitoring.
func TimedOut() {
	s := "execution took too long"
	log.Println("FATAL:", s)
	monitor("CRITICAL", s)
}

// Busy states that the command hangs and reports failure to the monitoring.
// Those tasks should be automatically killed, if it happens often.
func Busy() {
	s := "previous invocation of command still running"
	log.Println("FATAL:", s)
	monitor("CRITICAL", s)
}

// Failed states that the command didn't execute successfully and reports
// failure to the monitoring. Also Logs error output.
func Failed(err error) {
	s := fmt.Sprintln("Failed to execute: ", err)
	log.Println("FATAL:", s)
	monitor("CRITICAL", s)
}

func main() {
	var cmd *exec.Cmd
	var interval, timeout time.Duration

	// FIXME(mlafeldt) add command-line options for
	//                 - monitoring command (optional)
	//                 - kill or wait on busy state (optional)
	log.SetFlags(0)

	flag.DurationVar(&interval, "i", -1,
		"set execution interval for command, e.g. 45s, 2m, 1h30m, default: 1/10 of timeout")
	flag.DurationVar(&timeout, "t", 1*time.Minute,
		"set execution timeout for command, e.g. 45s, 2m, 1h30m, default: 1m")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("FATAL: no command to execute")
		return
	}

	command := flag.Arg(0)

	if interval >= timeout {
		log.Fatal("FATAL: interval >= timeout, no time left for actual command execution")
		return
	}

	if interval == -1 {
		interval = timeout / 10
	}

	// FIXME(nightlyone) try two intervals instead of one?
	timer := time.AfterFunc(timeout, func() {
		TimedOut()
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
		os.Exit(0)
	})

	SpreadWait(interval)

	// Ensures that only one of these command runs concurrently on this
	// machine.  Also cleans up stale locks of dead instances.
	base := filepath.Base(command)
	lock_dir := os.TempDir()
	os.Mkdir(filepath.Join(lock_dir, base), 0700)
	lock, _ := lockfile.New(filepath.Join(lock_dir, base, base+".lock"))
	if err := lock.TryLock(); err != nil {
		if err != lockfile.ErrBusy {
			log.Printf("ERROR: locking %s: reason: %v\n", lock, err)
		}
		timer.Stop()
		Busy()
		return
	}
	defer lock.Unlock()

	// FIXME(nightlyone) capture at least cmd.Stderr, and optionally
	// cmd.Stdout
	cmd = exec.Command(command, flag.Args()[1:]...)

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
}
