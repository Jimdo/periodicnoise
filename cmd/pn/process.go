package main

import (
	"os"
	"os/exec"
	"syscall"
)

func SignalProcess(p *os.Process, sig syscall.Signal) error {
	if p == nil {
		return syscall.ESRCH
	}
	err := p.Signal(sig)
	return err
}

func KillProcess(p *os.Process) error {
	return SignalProcess(p, syscall.SIGKILL)
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
