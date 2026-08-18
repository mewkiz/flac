package meta

import "io"

// readString reads and returns exactly n bytes from the provided io.Reader.
//
// The error is io.EOF only if no bytes were read. If an io.EOF happens after
// reading some but not all the bytes, ReadFull returns io.ErrUnexpectedEOF. On
// return, n == len(buf) if and only if err == nil.
func readString(r io.Reader, n int) (string, error) {
	// Guard against a huge up-front allocation driven by an untrusted length.
	// Metadata block bodies are always read through an io.LimitedReader, so a
	// declared length larger than the bytes remaining in the block is invalid;
	// report a truncated read instead of allocating n bytes. A negative n can
	// only arise from a uint32 length overflowing int on 32-bit platforms and
	// is likewise invalid.
	if n < 0 {
		return "", io.ErrUnexpectedEOF
	}
	if lr, ok := r.(*io.LimitedReader); ok && int64(n) > lr.N {
		return "", io.ErrUnexpectedEOF
	}
	// readBuf is the local buffer used by readBytes.
	var backingArray [4096]byte // hopefully allocated on stack.
	readBuf := backingArray[:]
	if n > len(readBuf) {
		// The local buffer is initially 4096 bytes and will grow automatically if
		// so required.
		readBuf = make([]byte, n)
	}
	_, err := io.ReadFull(r, readBuf[:n])
	if err != nil {
		return "", err
	}
	return string(readBuf[:n]), nil
}
