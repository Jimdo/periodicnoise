package main

import (
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"time"
)

func timerChannel(timer *time.Timer) <-chan time.Time {
	if timer != nil {
		return timer.C
	}
	return nil
}

func disableTimer(timer *time.Timer) *time.Timer {
	if timer != nil {
		timer.Stop()
	}
	return nil
}

// ScatterWait avoids the thundering herd problem
// on remote services used by this command.
// Set maxDelay to 0, if this is not an issue.
func ScatterWait(maxDelay time.Duration) {
	if maxDelay > 0 {
		// Seed random generator with current process ID
		rand.Seed(int64(os.Getpid()))
		// Sleep for random amount of time within maxDelay
		time.Sleep(time.Duration(rand.Int63n(int64(maxDelay))))
	}
}

// CoreLoopRetry encapsulates retries, so flaky commands can be handled, too.
func CoreLoopRetry(args []string, logger io.Writer) (err error) {
	for i := uint(0); i < opts.Retries+1; i++ {
		err = CoreLoopOnce(args, logger)
		if err == nil {
			return nil
		}

		// Don't retry on funky exit codes, which our user considers ok.
		if _, ok := err.(*exec.ExitError); ok {
			if code, _ := error2exit(err); code == monitorOk {
				return nil
			}
		}
	}
	return err
}

// CoreLoopOnce handles the core logic of our tool,
// Which is:
//  * wait a random amount of time
//  * lock top ensure single execution
//  * stream output
//  * start timer to limit execution time
//  * wait for output streams
//  * free the lock
//  * report situation via return code
//  * handles signals
func CoreLoopOnce(args []string, logger io.Writer) error {
	var wg sync.WaitGroup

	now := time.Now()
	ScatterWait(opts.MaxDelay)

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

	//softlimit provides a softer deadline, after which cmd will by signalled,
	//but has a chance to catch the signal
	softlimit := time.NewTimer(opts.Timeout - time.Since(now) - opts.GraceTime)
	if opts.GraceTime == 0 || opts.GraceTime >= opts.Timeout {
		softlimit = disableTimer(softlimit)
	}

	sigc := ReceiveDeadlySignals()
	defer IgnoreDeadlySignals(sigc)

	for errc != nil {
		select {
		case signal := <-sigc:
			// clear timers
			hardlimit = disableTimer(hardlimit)
			softlimit = disableTimer(softlimit)
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
			softlimit = disableTimer(softlimit)

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
		case timeo := <-timerChannel(softlimit):
			// report error, since we DID timeout already
			err = &TimeoutError{
				soft:  true,
				after: timeo.Sub(now),
			}

			// cancel soft timer, but keep rest intact
			softlimit = disableTimer(softlimit)

			// now terminate process tree, if it exists
			if grp, _ := ProcessGroup(cmd.Process); TerminateProcess(grp) == nil {
				log.Println("INFO: Terminated process tree")
			} else if TerminateProcess(cmd.Process) == nil {
				log.Println("INFO: Terminated process")
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
