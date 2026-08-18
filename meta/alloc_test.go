package meta

import (
	"bytes"
	"runtime"
	"testing"
)

// TestMetadataStringLengthAlloc ensures that an oversized string-length field
// in a metadata block body does not trigger an allocation proportional to the
// declared length. The block bodies below are only a handful of bytes but
// declare ~2 GB string lengths; parsing must fail cleanly without allocating
// gigabytes.
func TestMetadataStringLengthAlloc(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			// PICTURE block (type 6, last), length 8. Body: Type(4)=0,
			// MIME length(4)=0x7fffffff.
			name: "picture MIME length",
			input: []byte{
				0x86, 0x00, 0x00, 0x08,
				0x00, 0x00, 0x00, 0x00,
				0x7f, 0xff, 0xff, 0xff,
			},
		},
		{
			// VORBIS_COMMENT block (type 4, last), length 4. Body:
			// vendor length(4, little-endian)=0x7fffffff.
			name: "vorbis comment vendor length",
			input: []byte{
				0x84, 0x00, 0x00, 0x04,
				0xff, 0xff, 0xff, 0x7f,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m0, m1 runtime.MemStats
			runtime.ReadMemStats(&m0)
			if _, err := Parse(bytes.NewReader(tt.input)); err == nil {
				t.Fatal("expected error for truncated block, got nil")
			}
			runtime.ReadMemStats(&m1)
			if delta := m1.TotalAlloc - m0.TotalAlloc; delta > 16<<20 {
				t.Fatalf("excessive allocation on tiny block: %d bytes", delta)
			}
		})
	}
}
