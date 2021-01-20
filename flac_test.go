package flac_test

import (
	"fmt"
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

func TestDecode(t *testing.T) {
	paths := []string{
		"meta/testdata/input-SCPAP.flac",
		"meta/testdata/input-SCVA.flac",
		"meta/testdata/input-SCVPAP.flac",
		"meta/testdata/input-VA.flac",
		"meta/testdata/silence.flac",
		"testdata/19875.flac",
		"testdata/44127.flac",
		"testdata/59996.flac",
		"testdata/80574.flac",
		"testdata/172960.flac",
		"testdata/189983.flac",
		"testdata/191885.flac",
		"testdata/212768.flac",
		"testdata/220014.flac",
		"testdata/243749.flac",
		"testdata/256529.flac",
		"testdata/257344.flac",
		"testdata/8297-275156-0011.flac",
		"testdata/love.flac",
	}

	opts := map[string]flac.Option{
		"buffered":        flac.Buffered,
		"bufferedParse":   flac.BufferedParse,
		"enableSeek":      flac.EnableSeek,
		"enableSeekParse": flac.EnableSeekParse,
	}

	for _, path := range paths {
		for k, opt := range opts {
			t.Run(fmt.Sprintf("%s/%s", k, path), func(t *testing.T) {
				_, err := flac.Open(path, opt)
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
