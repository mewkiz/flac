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

func TestSkipping(t *testing.T) {
	f, err := os.Open("testdata/172960.flac")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	testPos := []struct {
		seek     int64
		whence   int
		expected int64
	}{
		{seek: 0, whence: io.SeekStart, expected: 0},
		{seek: 9000, whence: io.SeekStart, expected: 8192},
		{seek: 0, whence: io.SeekStart, expected: 0},
		{seek: -6000, whence: io.SeekEnd, expected: 36864},

		// expected: 40960 seems like the wrong answer, it should be
		// before 36864 from the previous seek.
		// Debugging shows that the file offset at this point is 226,
		// which is before the first frame position.
		// Maybe there is some buffering I don't understand?
		// Maybe io.SeekCurrent can't be supported??
		{seek: -8000, whence: io.SeekCurrent, expected: 40960},
		{seek: 0, whence: io.SeekEnd, expected: 40960},
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

	opts := map[string]func(io.Reader) (*flac.Stream, error){
		"new":     flac.New,
		"newSeek": flac.NewSeek,
		"parse":   flac.Parse,
	}

	for _, path := range paths {
		for k, f := range opts {
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
