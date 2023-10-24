package bits

import "testing"

func TestIntN(t *testing.T) {
	golden := []struct {
		x    uint64
		n    uint
		want int64
	}{
		{x: 0b011, n: 3, want: 3},
		{x: 0b010, n: 3, want: 2},
		{x: 0b001, n: 3, want: 1},
		{x: 0b000, n: 3, want: 0},
		{x: 0b111, n: 3, want: -1},
		{x: 0b110, n: 3, want: -2},
		{x: 0b101, n: 3, want: -3},
		{x: 0b100, n: 3, want: -4},
	}
	for _, g := range golden {
		got := IntN(g.x, g.n)
		if g.want != got {
			t.Errorf("result mismatch of IntN(x=0b%03b, n=%d); expected %d, got %d", g.x, g.n, g.want, got)
			continue
		}
	}
}
