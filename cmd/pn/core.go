package main

import (
	"io"
	"os/exec"
	"sync"
	"time"
)

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

// CoreLoop handles the core logic of our tool,
// Which is:
//  * wait a random amount of time
//  * lock top ensure single execution
//  * stream output
//  * start timer to limit execution time
//  * wait for output streams
//  * free the lock
//  * report situation via return code
func CoreLoop(args []string, logger io.Writer) error {
	var wg sync.WaitGroup

	now := time.Now()
	SpreadWait(opts.MaxDelay)

	lock, err := createLock(opts.KillRunning)
	if err != nil {
		return &LockError{
			name: string(lock),
			err:  err,
		}
	}
	defer lock.Unlock()

	cmd := exec.Command(args[0], args[1:]...)
	err = connectOutputs(cmd, logger, &wg)
	if err != nil {
		return err
	}

	// central error code channel for asynchronous errors
	errc := make(chan error, 1)

	timer := hardtimer(now, cmd, errc)
	go processLife(cmd, errc)

	err = <-errc

	// We got either triggerd by the timer or finished our try to execute now.
	// So no need for further timeouts here.
	timer.Stop()

	_, isTimeout := err.(*TimeoutError)
	if cmd.Process != nil {
		// we should collect wait state for what we killed
		if isTimeout {
			<-errc
		}
		// wait for output streams to finish
		wg.Wait()
	}
	return err
}
