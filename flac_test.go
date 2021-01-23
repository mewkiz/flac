package flac_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/mewkiz/flac"
)

func TestSkipID3v2(t *testing.T) {
	if _, err := flac.ParseFile("testdata/id3.flac"); err != nil {
		t.Fatal(err)
	}
}

func TestSeek(t *testing.T) {
	f, err := os.Open("testdata/172960.flac")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	//Seek Table:
	// {SampleNum:0 Offset:8283 NSamples:4096}
	// {SampleNum:4096 Offset:17777 NSamples:4096}
	// {SampleNum:8192 Offset:27141 NSamples:4096}
	// {SampleNum:12288 Offset:36665 NSamples:4096}
	// {SampleNum:16384 Offset:46179 NSamples:4096}
	// {SampleNum:20480 Offset:55341 NSamples:4096}
	// {SampleNum:24576 Offset:64690 NSamples:4096}
	// {SampleNum:28672 Offset:74269 NSamples:4096}
	// {SampleNum:32768 Offset:81984 NSamples:4096}
	// {SampleNum:36864 Offset:86656 NSamples:4096}
	// {SampleNum:40960 Offset:89596 NSamples:2723}

	testPos := []struct {
		seek     int64
		whence   int
		expected int64
	}{
		{seek: 0, whence: io.SeekStart, expected: 0},
		{seek: 9000, whence: io.SeekStart, expected: 8192},
		{seek: 0, whence: io.SeekStart, expected: 0},
		{seek: -6000, whence: io.SeekEnd, expected: 36864},
		{seek: -8000, whence: io.SeekCurrent, expected: 32768},
		{seek: 8000, whence: io.SeekCurrent, expected: 40960},
		{seek: 0, whence: io.SeekEnd, expected: 40960},
		{seek: 50000, whence: io.SeekStart, expected: 40960},
		{seek: 100, whence: io.SeekEnd, expected: 40960},
		{seek: -100, whence: io.SeekStart, expected: 0},
	}

	stream, err := flac.NewSeek(f)
	if err != nil {
		t.Fatal(err)
	}

	for i, pos := range testPos {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			p, err := stream.Seek(pos.seek, pos.whence)
			if err != nil {
				t.Fatal(err)
			}

			if p != pos.expected {
				t.Fatalf("pos %d does not equal %d", p, pos.expected)
			}

			_, err = stream.ParseNext()
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}
		})

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

	funcs := map[string]func(io.Reader) (*flac.Stream, error){
		"new":     flac.New,
		"newSeek": func(r io.Reader) (*flac.Stream, error) { return flac.NewSeek(r.(io.ReadSeeker)) },
		"parse":   flac.Parse,
	}

	for _, path := range paths {
		for k, f := range funcs {
			t.Run(fmt.Sprintf("%s/%s", k, path), func(t *testing.T) {
				file, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}

				stream, err := f(file)
				if err != nil {
					t.Fatal(err)
				}

				_, err = stream.ParseNext()
				if err != nil {
					t.Fatal(err)
				}

				file.Close()
			})
		}
	}
}
