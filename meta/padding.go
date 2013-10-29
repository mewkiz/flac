package meta

import (
	"errors"
	"io"
)

// VerifyPadding verifies that the padding metadata block only contains 0 bits.
// The provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
func VerifyPadding(r io.Reader) (err error) {
	// Verify up to 4 kb of padding each iteration.
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !isAllZero(buf[:n]) {
			return errors.New("meta.VerifyPadding: invalid padding; must contain only zeroes")
		}
	}
	return nil
}

/// ### [ note ] ###
///    - Might trigger unnecessary errors.
/// ### [/ note ] ###

// isAllZero returns true if the value of each byte in the provided slice is 0,
// and false otherwise.
func isAllZero(buf []byte) bool {
	for _, b := range buf {
		if b != 0 {
			return false
		}
	}
	return true
}
