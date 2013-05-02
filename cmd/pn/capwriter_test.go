package main

import (
	"testing"
)

func TestCapWriter(t *testing.T) {
	cw := NewCapWriter(3)
	t.Logf("%T, %+v", cw, cw)
	cw.Write([]byte("he"))
	if string(cw.Bytes()) != "he" {
		t.Errorf("got %s, expected %s", cw.Bytes(), "he")
	}
	cw.Write([]byte("llo"))
	if string(cw.Bytes()) != "hel" {
		t.Errorf("got %s, expected %s", cw.Bytes(), "hel")
	}
}
