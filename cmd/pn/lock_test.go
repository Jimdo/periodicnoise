package main

import (
	"github.com/nightlyone/lockfile"
	"testing"
)

func TestLocking(t *testing.T) {
	monitoringEvent = "TestLocking"
	lf, err := createLock(false)
	if err != nil {
		t.Fatal("environment problem: ", err)
		return
	}
	t.Log("Got lockfile")
	if lf2, err := createLock(false); err == nil {
		t.Errorf("got lockfile, but expected '%v'", lockfile.ErrBusy)
		lf2.Unlock()
	} else if err != lockfile.ErrBusy {
		t.Errorf("bad error got '%v', want '%v'", err, lockfile.ErrBusy)
	} else {
		t.Logf("got expected '%v'", err)
	}
	lf.Unlock()

	lf3, err := createLock(false)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else {
		t.Log("got lockfile", lf3)
		lf3.Unlock()
	}
}
