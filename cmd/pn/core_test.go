package main

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"testing"

	flags "github.com/jessevdk/go-flags"
)

func TestCoreLoopExitSuccess(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- true"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoop(args, &bytes.Buffer{})
	t.Log(output.String())
	if err != nil {
		t.Error("want no error, got", err)
	}
}

func TestCoreLoopExitError(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- false"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoop(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if _, ok := err.(*exec.ExitError); ok {
		t.Log("got", err)
	} else {
		t.Error("want exit error, got", err)
	}
}

func TestCoreLoopWrongCommand(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- /var/run/nonexistant"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoop(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if _, ok := err.(*NotAvailableError); ok {
		t.Log("got", err)
	} else {
		t.Error("want not available error, got", err)
	}
}

func TestCoreLoopHardTimeout(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--timeout=100ms -- sleep 1"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoop(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if timeout, ok := err.(*TimeoutError); ok && !timeout.soft {
		t.Log("got", err)
	} else {
		t.Error("want hard timeout error, got ", err)
	}
}

func TestCoreLoopSoftTimeout(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "--timeout=100ms --grace-time=50ms -- sleep 1"
	args, err := flags.ParseArgs(&opts, strings.Fields(arguments))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	log.SetOutput(&output)

	err = CoreLoop(args, &bytes.Buffer{})
	t.Log(output.String())
	if err == nil {
		t.Error("want error, got nil")
	} else if timeout, ok := err.(*TimeoutError); ok && timeout.soft {
		t.Log("got", err)
	} else {
		t.Error("want soft timeout error, got ", err)
	}
}
