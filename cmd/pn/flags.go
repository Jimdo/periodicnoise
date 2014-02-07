package main

import (
	"fmt"
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
	MonitorOk        []uint8       `long:"monitor-ok" description:"add exit code to consider as no failure."`
	MonitorWarning   []uint8       `long:"monitor-warning" description:"add exit code to warn about"`
	MonitorCritical  []uint8       `long:"monitor-critical" description:"add exit code to consider as critical failure"`
	MonitorUnknown   []uint8       `long:"monitor-unknown" description:"add exit code to consider as state not known"`
	SendAs           string        `long:"send-as" description:"send monitoring events masquerading as this entity"`
	SendTo           string        `long:"send-to" description:"send monitoring events to this service"`
}

// FlagConstraintError happens when command line arguments make no sense or contradict each other
type FlagConstraintError struct {
	Constraint string
}

func (c *FlagConstraintError) Error() string {
	return fmt.Sprintf(c.Constraint)
}

func validateOptionConstraints() (err error) {
	if opts.MaxDelay >= opts.Timeout {
		return &FlagConstraintError{Constraint: "max delay >= timeout, no time left for actual command execution"}
	}

	// Setup constraint that exit code 0 is ALWAYS considered ok ...
	unique := map[uint8]monitoringResult{
		uint8(monitorOk): monitorOk,
	}

	// ... and check for duplicate definitions of severity <=> exit code mappings.
	ForEachResultMapping(func(severity monitoringResult, code uint8) bool {
		if duplicate, exists := unique[code]; exists && duplicate != severity {
			err = &FlagConstraintError{
				Constraint: fmt.Sprintf("exit code %d is already considered %s instead of %s",
					code, monitoringResults[duplicate], monitoringResults[severity]),
			}
			return false
		}
		unique[code] = severity
		return true
	})

	// check for identity mappings (ok == monitorOk is enforced above)
	if _, exists := unique[uint8(monitorWarning)]; !exists {
		opts.MonitorWarning = append(opts.MonitorWarning, uint8(monitorWarning))
	}
	if _, exists := unique[uint8(monitorCritical)]; !exists {
		opts.MonitorCritical = append(opts.MonitorCritical, uint8(monitorCritical))
	}
	if _, exists := unique[uint8(monitorUnknown)]; !exists {
		opts.MonitorUnknown = append(opts.MonitorUnknown, uint8(monitorUnknown))
	}
	return err
}

func parseFlags() ([]string, error) {
	p := flags.NewParser(&opts, flags.Default)

	// display nice usage message
	p.Usage = "[OPTIONS]... COMMAND\n\nSafely wrap execution of COMMAND in e.g. a cron job"

	args, err := p.Parse()
	if err != nil {
		// --help is not an error
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, nil
		}
		return nil, err
	}

	if len(args) < 1 {
		return nil, &FlagConstraintError{Constraint: "no command to execute"}
	}

	if err := validateOptionConstraints(); err != nil {
		return nil, err
	}

	return args, nil
}
