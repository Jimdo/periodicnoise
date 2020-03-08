package main

import (
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
	lf.Unlock()

	lf3, err := createLock(false)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else {
		t.Log("got lockfile", lf3)
		lf3.Unlock()
	}
}
