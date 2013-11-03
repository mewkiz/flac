package meta

import (
	"io"
)

// readBuf is the local buffer used by readBytes.
var readBuf = make([]byte, 4096)

// readBytes reads and returns exactly n bytes from the provided io.Reader. The
// local buffer is reused in between calls to reduce generation of garbage. It
// is the callers responsibility to make a copy of the returned data.
//
// The local buffer is initially 4096 bytes and will grow automatically if so
// required.
func readBytes(r io.Reader, n int) ([]byte, error) {
	if n > len(readBuf) {
		readBuf = make([]byte, n)
	}
	_, err := io.ReadFull(r, readBuf[:n])
	if err != nil {
		return nil, err
	}
	return readBuf[:n], nil
}
