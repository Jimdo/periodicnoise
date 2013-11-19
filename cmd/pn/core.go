package main

import (
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

func timerChannel(timer *time.Timer) <-chan time.Time {
	if timer == nil {
		return nil
	} else {
		return timer.C
	}
}

func disableTimer(timer *time.Timer) *time.Timer {
	if timer != nil {
		timer.Stop()
	}
	return nil
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
//  * handles signals
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

	// error code channel for asynchronous errors from processLife
	errc := make(chan error, 1)
	go processLife(cmd, errc)

	// hardlimit provides a hard deadline, after which cmd will not run anymore
	hardlimit := time.NewTimer(opts.Timeout - time.Since(now))

	sigc := ReceiveDeadlySignals()
	defer IgnoreDeadlySignals(sigc)

	for errc != nil {
		select {
		case signal := <-sigc:
			// clear timer
			hardlimit = disableTimer(hardlimit)
			log.Println("INFO: Received signal", signal)

			if grp, _ := ProcessGroup(cmd.Process); KillProcess(grp) == nil {
				log.Println("INFO: Killed process group, because we have been signalled")
			} else if KillProcess(cmd.Process) == nil {
				log.Println("INFO: Killed process, because we have been signalled")
			} else {
				// normal case for fast kill
				log.Println("INFO: Killed before the process even started?")

				// and we are done here, so terminate the loop
				errc = nil
			}
		case cerr := <-errc:
			// we record only ONE error. Timeouts might set an error before we come here.
			if err == nil {
				err = cerr
			}

			// disable error channel, since we are only allowed to receive once here
			// and like to leave the for loop now.
			errc = nil

			// clear timer
			hardlimit = disableTimer(hardlimit)

			// wait for output streams to finish
			wg.Wait()

		case timeo := <-timerChannel(hardlimit):
			err = &TimeoutError{
				after: timeo.Sub(now),
			}
			// cancel timers, but collect return code from error channel in next iteration
			hardlimit = disableTimer(hardlimit)

			// block signals, since we exit anyway now
			sigc = nil

			// now terminate process tree, if it exists
			if grp, _ := ProcessGroup(cmd.Process); KillProcess(grp) == nil {
				log.Println("INFO: Killed process tree")
			} else if KillProcess(cmd.Process) == nil {
				log.Println("INFO: Killed process")
			} else {
				// very fishy, should never get here, but we still handle that crap. Better Fatal exit?
				log.Println("FATAL: Timeout before the process even started? Please increase the timeout!")

				// and we are done here, so terminate the loop
				errc = nil
			}
		}
	}

	return err
}
