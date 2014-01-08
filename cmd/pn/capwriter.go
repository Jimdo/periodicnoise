package main

// CapWriter implements an io.Writer, that will silently ignore any write after a certain limit.
type CapWriter []byte

// NewCapWriter returns a CapWriter that will silently ignore any write after consuming limit bytes of output.
func NewCapWriter(limit int) *CapWriter {
	cw := make(CapWriter, 0, limit)
	return &cw
}

// Bytes returns underlying byte slice of of CapWriter
func (cw *CapWriter) Bytes() []byte { return []byte(*cw) }

func (cw *CapWriter) Write(p []byte) (n int, err error) {
	c := cap(*cw) - len(*cw)
	if len(p) < c {
		c = len(p)
	}
	(*cw) = append((*cw), p[:c]...)
	return len(p), nil
}
