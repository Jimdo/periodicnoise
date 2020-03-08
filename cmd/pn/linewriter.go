package main

import (
	"bytes"
	"io"
)

// LineWriter is an io.Writer that will pass on line breaks as message boundaries.
type LineWriter struct {
	w io.Writer
}

var lf = []byte("\n")

func (l *LineWriter) Write(p []byte) (n int, err error) {
	for _, line := range bytes.SplitAfter(p, lf) {
		// drop pure whitespace lines and fake successful write of them
		if nospaces := bytes.TrimSpace(line); bytes.Equal(nospaces, lf) {
			n += len(line)
			continue
		}

		var written int
		written, err = l.w.Write(line)
		n += written
		if err != nil {
			return n, err
		}
	}
	return n, err
}
