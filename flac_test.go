package flac_test

import (
	"io"
	"testing"

	"github.com/mewkiz/flac"
)

func TestSkipID3v2(t *testing.T) {
	if _, err := flac.Open("testdata/id3.flac", flac.BufferedParse); err != nil {
		t.Fatal(err)
	}
}

func TestSkipping(t *testing.T) {
	stream, err := flac.Open("testdata/id3.flac", flac.EnableSeek)
	if err != nil {
		t.Fatal(err)
	}

	pos, err := stream.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	if pos != 0 {
		t.Fatalf("pos %d does not equal %d", pos, 0)
	}

	pos, err = stream.Seek(400000, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	if pos != 438272 {
		t.Fatalf("pos %d does not equal %d", pos, 438272)
	}

	pos, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	if pos != 0 {
		t.Fatalf("pos %d does not equal %d", pos, 0)
	}

	pos, err = stream.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}

	if pos != 8818688 {
		t.Fatalf("pos %d does not equal %d", pos, 8818688)
	}
}
