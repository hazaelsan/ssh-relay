package session

import (
	"io"
)

// NewWrapper creates a Wrapper from a reader/writer pair.
func NewWrapper(in io.ReadCloser, out io.WriteCloser) *Wrapper {
	return &Wrapper{
		in:  in,
		out: out,
	}
}

// Wrapper is a wrapper to interface with ssh(1) via a reader/writer pair.
type Wrapper struct {
	in  io.ReadCloser
	out io.WriteCloser
}

// Read is a wrapper to read from the ReadCloser.
func (w *Wrapper) Read(b []byte) (int, error) {
	return w.in.Read(b)
}

// Write is a wrapper to read from the WriteCloser.
func (w *Wrapper) Write(b []byte) (int, error) {
	return w.out.Write(b)
}

// Close closes the WriteCloser and ReadCloser.
func (w *Wrapper) Close() error {
	outErr := w.out.Close()
	if err := w.in.Close(); err != nil {
		return err
	}
	return outErr
}
