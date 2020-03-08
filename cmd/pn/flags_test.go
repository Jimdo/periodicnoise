package main

import (
	"strings"
	"testing"

	flags "github.com/jessevdk/go-flags"
)

func TestDefaulSettingsOk(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- true"
	_, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	err = validateOptionConstraints()
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}

}

func TestZeroTimeout(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--timeout=0ms -- true"
	_, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	err = validateOptionConstraints()
	if err == nil {
		t.Error("want error, got nil")
	} else if e, ok := err.(*FlagConstraintError); ok {
		want := "max delay >= timeout, no time left for actual command execution"
		if e.Constraint != want {
			t.Errorf("want %s, got %s", want, err)
		} else {
			t.Log("got", err)
		}
	} else {
		t.Error("want flag constraint error, got", err)
	}
}

func TestTooBigMaxDelay(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--timeout=1s --max-start-delay=2s -- true"
	_, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	err = validateOptionConstraints()
	if err == nil {
		t.Error("want error, got nil")
	} else if e, ok := err.(*FlagConstraintError); ok {
		want := "max delay >= timeout, no time left for actual command execution"
		if e.Constraint != want {
			t.Errorf("want %s, got %s", want, err)
		} else {
			t.Log("got", err)
		}
	} else {
		t.Error("want flag constraint error, got", err)
	}
}

func TestPuppetDetailedExitCodes(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--monitor-ok=2 --monitor-warning=4 --monitor-warning=6 --monitor-critical=1 -- true"
	_, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	err = validateOptionConstraints()
	if err != nil {
		t.Errorf("want no error, got %s", err)
	}
}

func TestDuplicateExitCodes(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--monitor-ok=2 --monitor-warning=4 --monitor-critical=2 -- true"
	_, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	err = validateOptionConstraints()
	if err == nil {
		t.Error("want error, got nil")
	} else if e, ok := err.(*FlagConstraintError); ok {
		want := "is already considered"
		if !strings.Contains(e.Constraint, want) {
			t.Errorf("want %s, got %s", want, err)
		} else {
			t.Log("got", err)
		}
	} else {
		t.Error("want flag constraint error, got", err)
	}
}
