package main

import (
	"bytes"
	"log"
	"strings"
	"testing"

	flags "github.com/jessevdk/go-flags"
)

func TestCoreLoopSimple(t *testing.T) {
	oldopts := opts
	defer func() { opts = oldopts }()

	arguments := "-- sleep 1"
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
	} else if _, ok := err.(*TimeoutError); ok {
		t.Log("got", err)
	} else {
		t.Error("want timeout error, got ", err)
	}
}
