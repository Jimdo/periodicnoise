package main

type CapWriter []byte

// Return writer, that will silently ignore any writer after limit
func NewCapWriter(limit int) *CapWriter {
	cw := make(CapWriter, 0, limit)
	return &cw
}

// return underlying byte slice of of CapWriter
func (cw *CapWriter) Bytes() []byte { return []byte(*cw) }

// satisfy io.Writer interface
func (cw *CapWriter) Write(p []byte) (n int, err error) {
	c := cap(*cw) - len(*cw)
	if len(p) < c {
		c = len(p)
	}
	(*cw) = append((*cw), p[:c]...)
	return len(p), nil
}
