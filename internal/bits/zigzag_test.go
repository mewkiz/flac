package bits

import (
	"testing"
)

func TestDecodeZigZag(t *testing.T) {
	golden := []struct {
		x    uint32
		want int32
	}{
		{x: 0, want: 0},
		{x: 1, want: -1},
		{x: 2, want: 1},
		{x: 3, want: -2},
		{x: 4, want: 2},
		{x: 5, want: -3},
		{x: 6, want: 3},
	}
	for _, g := range golden {
		got := DecodeZigZag(g.x)
		if g.want != got {
			t.Errorf("result mismatch of DecodeZigZag(x=%d); expected %d, got %d", g.x, g.want, got)
			continue
		}
	}
}

func TestEncodeZigZag(t *testing.T) {
	golden := []struct {
		x    int32
		want uint32
	}{
		{x: 0, want: 0},
		{x: -1, want: 1},
		{x: 1, want: 2},
		{x: -2, want: 3},
		{x: 2, want: 4},
		{x: -3, want: 5},
		{x: 3, want: 6},
	}
	for _, g := range golden {
		got := EncodeZigZag(g.x)
		if g.want != got {
			t.Errorf("result mismatch of EncodeZigZag(x=%d); expected %d, got %d", g.x, g.want, got)
			continue
		}
	}
}
