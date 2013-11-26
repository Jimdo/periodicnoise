package main

import (
	"log"
	"time"

	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	MaxDelay         time.Duration `short:"d" long:"max-start-delay" description:"optional maximum execution start delay for command, e.g. 45s, 2m, 1h30m"`
	Timeout          time.Duration `short:"t" long:"timeout" default:"1m" description:"set hard execution timeout for command, e.g. 45s, 2m, 1h30m"`
	UseSyslog        bool          `short:"s" long:"use-syslog" description:"log via syslog instead of stderr"`
	WrapNagiosPlugin bool          `short:"n" long:"wrap-nagios-plugin" description:"wrap nagios plugin (pass on return codes, pass first 8KiB of stdout as message)"`
	NoPipeStderr     bool          `long:"no-stream-stderr" description:"do not stream stderr to log"`
	NoPipeStdout     bool          `long:"no-stream-stdout" description:"do not stream stdout to log"`
	MonitoringEvent  string        `short:"E" long:"monitor-event" description:"monitoring event (defaults to check_foo for /path/check_foo.sh)"`
	KillRunning      bool          `short:"k" long:"kill-running" description:"kill already running instance of command"`
	NoMonitoring     bool          `long:"no-monitoring" description:"wrap command without sending monitoring events"`
	GraceTime        time.Duration `long:"grace-time" default:"10s" description:"time left until TIMEOUT, before sending SIGTERM to command, e.g. 45s, 2m, 1h30m"`
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

	if opts.GraceTime >= opts.Timeout {
		log.Fatal("FATAL: grace time >= timeout, no time left for actual command execution")
	}

	return args
}
