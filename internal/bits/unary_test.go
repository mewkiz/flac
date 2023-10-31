package bits_test

import (
	"bytes"
	"github.com/icza/bitio"
	"github.com/mewkiz/flac/internal/bits"
	"testing"
)

func TestUnary(t *testing.T) {
	w := new(bytes.Buffer)
	bw := bitio.NewWriter(w)

	var want uint64
	for ; want < 1000; want++ {
		// Write unary
		if err := bits.WriteUnary(bw, want); err != nil {
			t.Fatalf("error writing unary: %v", err)
		}
		// Flush buffer
		if err := bw.Close(); err != nil {
			t.Fatalf("error closing the buffer: %v", err)
		}

		// Read written unary
		r := bits.NewReader(w)
		got, err := r.ReadUnary()
		if err != nil {
			t.Fatalf("error reading unary: %v", err)
		}

		if got != want {
			t.Fatalf("the written and read unary doesn't match the original. got: %v, expected: %v", got, want)
		}
	}
}
