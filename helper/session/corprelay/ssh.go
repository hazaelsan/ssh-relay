package corprelay

import (
	"io"
)

// sshWrapper is a wrapper to interface with ssh(1) via a reader/writer pair.
type sshWrapper struct {
	in  io.ReadCloser
	out io.WriteCloser
}

// Read is a wrapper to read from the ReadCloser.
func (s *sshWrapper) Read(b []byte) (int, error) {
	return s.in.Read(b)
}

// Write is a wrapper to read from the WriteCloser.
func (s *sshWrapper) Write(b []byte) (int, error) {
	return s.out.Write(b)
}

// Close closes the WriteCloser and ReadCloser.
func (s *sshWrapper) Close() error {
	outErr := s.out.Close()
	if err := s.in.Close(); err != nil {
		return err
	}
	return outErr
}
