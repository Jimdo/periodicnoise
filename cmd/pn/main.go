package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
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

var opts struct {
	MaxDelay         time.Duration `short:"d" long:"max-start-delay" description:"optional maximum execution start delay for command, e.g. 45s, 2m, 1h30m"`
	Timeout          time.Duration `short:"t" long:"timeout" default:"1m" description:"set execution timeout for command, e.g. 45s, 2m, 1h30m"`
	UseSyslog        bool          `short:"s" long:"use-syslog" description:"log via syslog instead of stderr"`
	WrapNagiosPlugin bool          `short:"n" long:"wrap-nagios-plugin" description:"wrap nagios plugin (pass on return codes, pass first 8KiB of stdout as message)"`
	NoPipeStderr     bool          `long:"no-stream-stderr" description:"do not stream stderr to log"`
	NoPipeStdout     bool          `long:"no-stream-stdout" description:"do not stream stdout to log"`
	MonitoringEvent  string        `short:"E" long:"monitor-event" description:"monitoring event (defaults to check_foo for /path/check_foo.sh)"`
	KillRunning      bool          `short:"k" long:"kill-running" description:"kill already running instance of command"`
	NoMonitoring     bool          `long:"no-monitoring" description:"wrap command without sending monitoring events"`
}

func parseFlags() []string {
	p := flags.NewParser(&opts, flags.Default)

	// display nice usage message
	p.Usage = "[OPTIONS]... COMMAND\n\nSafely wrap execution of COMMAND in e.g. a cron job"

	args, err := p.Parse()
	if err != nil {
		// --help is not an error
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil
		} else {
			log.Fatal("FATAL: invalid arguments")
		}
	}

	if len(args) < 1 {
		log.Fatal("FATAL: no command to execute")
	}

	if opts.MaxDelay >= opts.Timeout {
		log.Fatal("FATAL: max delay >= timeout, no time left for actual command execution")
	}

	return args
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

	// FIXME(nightlyone) try two intervals instead of one?
	timer := time.AfterFunc(opts.Timeout, func() {
		TimedOut(errors.New("execution took too long"))
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
			// FIXME(nightlyone) log the kill
		}
		os.Exit(0)
	})

	SpreadWait(opts.MaxDelay)

	lock, err := createLock(opts.KillRunning)
	if err != nil {
		timer.Stop()
		Locked(err)
		return
	}
	defer lock.Unlock()

	cmd = exec.Command(command, args[1:]...)

	if !opts.NoPipeStdout {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		if opts.WrapNagiosPlugin {
			firstbytes = NewCapWriter(8192)
			stdout := io.TeeReader(stdout, firstbytes)
			logStream(stdout, logger, &wg)
		} else {
			logStream(stdout, logger, &wg)
		}
	} else if opts.WrapNagiosPlugin {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		firstbytes = NewCapWriter(8192)
		logStream(stdout, firstbytes, &wg)
	}

	if !opts.NoPipeStderr {
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

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		timer.Stop()
		Failed(err)
	} else {
		timer.Stop()
		Ok()
	}
}
