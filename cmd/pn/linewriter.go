package main

import (
	"bytes"
	"io"
)

type LineWriter struct {
	w io.Writer
}

var lf = []byte("\n")

func (l *LineWriter) Write(p []byte) (n int, err error) {
	for _, line := range bytes.SplitAfter(p, lf) {
		var written int
		written, err = l.w.Write(line)
		n += written
		if err != nil {
			return n, err
		}
	}
	return n, err
}
