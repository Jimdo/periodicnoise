package main

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var DeadlySignals = []os.Signal{
	os.Signal(syscall.SIGTERM),
	os.Signal(syscall.SIGHUP),
	os.Signal(syscall.SIGINT),
}

func ReceiveDeadlySignals() (on chan os.Signal) {
	on = make(chan os.Signal, 1)
	signal.Notify(on, DeadlySignals...)
	return on
}

func IgnoreDeadlySignals(on chan os.Signal) {
	if on != nil {
		signal.Stop(on)
	}
}

func SignalProcess(p *os.Process, sig os.Signal) error {
	if p == nil {
		return syscall.ESRCH
	}
	err := p.Signal(sig)
	return err
}

func KillProcess(p *os.Process) error {
	return SignalProcess(p, syscall.SIGKILL)
}

var GracefulSignal os.Signal = syscall.SIGTERM

func TerminateProcess(p *os.Process) error {
	return SignalProcess(p, GracefulSignal)
}

var ErrNotLeader = errors.New("Process is not process group leader")

func ProcessGroup(p *os.Process) (grp *os.Process, err error) {
	if p == nil {
		return nil, syscall.ESRCH
	}
	pgid, err := syscall.Getpgid(p.Pid)
	if err != nil {
		return nil, err
	}

	// Pids 0 and 1 will have special meaning, so don't return them.
	if pgid < 2 {
		return nil, ErrNotLeader
	}

	// the process is not the leader?
	if pgid != p.Pid {
		return nil, ErrNotLeader
	}

	// This just creates a process object from a Pid in Unix
	// instead of actually searching it.
	grp, err = os.FindProcess(-pgid)
	return grp, err
}

// Did this command exit gracefully?
func HasNormalExit(cmd *exec.Cmd) bool {
	if cmd.ProcessState == nil {
		return false
	}

	status := cmd.ProcessState.Sys().(syscall.WaitStatus)
	return status.Exited()
}

func processLife(cmd *exec.Cmd, errc chan error) {
	// FIXME(nightlyone) This works neither in Windows nor Plan9.
	// Fix it, once we have users of this platform.
	// NOTE: Cannot setsid and and setpgid in one child. Would need double fork or exec,
	// which makes things very hard.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		errc <- &NotAvailableError{
			args: cmd.Args,
			err:  err,
		}
	} else {
		errc <- cmd.Wait()
	}
}
