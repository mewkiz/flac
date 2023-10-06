package flac_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	stream "github.com/mewkiz/flac/stream"
)

func TestSkipID3v2(t *testing.T) {
	if _, err := stream.ParseFile("testdata/id3.flac"); err != nil {
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
		seek     uint64
		expected uint64
		err      string
	}{
		{seek: 0, expected: 0},
		{seek: 9000, expected: 8192},
		{seek: 0, expected: 0},
		{seek: 8000, expected: 4096},
		{seek: 0, expected: 0},
		{seek: 50000, expected: 0, err: "unable to seek to sample number 50000"},
		{seek: 100, expected: 0},
	}

	stream, err := stream.NewSeek(f)
	if err != nil {
		t.Fatal(err)
	}

	for i, pos := range testPos {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			p, err := stream.Seek(pos.seek)
			if err != nil {
				if err.Error() != pos.err {
					t.Fatal(err)
				}
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

	funcs := map[string]func(io.Reader) (*stream.Stream, error){
		"new":     stream.New,
		"newSeek": func(r io.Reader) (*stream.Stream, error) { return stream.NewSeek(r.(io.ReadSeeker)) },
		"parse":   stream.Parse,
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
