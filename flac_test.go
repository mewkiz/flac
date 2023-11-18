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

	stream, err := flac.NewSeek(f)
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
		"testdata/24000-tts-sf.flac",
		"testdata/love.flac",
		// IETF test cases.
		//
		// ref: https://github.com/ietf-wg-cellar/flac-test-files/tree/main/subset
		"testdata/flac-test-files/subset/01 - blocksize 4096.flac",
		"testdata/flac-test-files/subset/02 - blocksize 4608.flac",
		"testdata/flac-test-files/subset/03 - blocksize 16.flac",
		"testdata/flac-test-files/subset/04 - blocksize 192.flac",
		"testdata/flac-test-files/subset/05 - blocksize 254.flac",
		"testdata/flac-test-files/subset/06 - blocksize 512.flac",
		"testdata/flac-test-files/subset/07 - blocksize 725.flac",
		"testdata/flac-test-files/subset/08 - blocksize 1000.flac",
		"testdata/flac-test-files/subset/09 - blocksize 1937.flac",
		"testdata/flac-test-files/subset/10 - blocksize 2304.flac",
		"testdata/flac-test-files/subset/11 - partition order 8.flac",
		"testdata/flac-test-files/subset/12 - qlp precision 15 bit.flac",
		"testdata/flac-test-files/subset/13 - qlp precision 2 bit.flac",
		"testdata/flac-test-files/subset/14 - wasted bits.flac",
		"testdata/flac-test-files/subset/15 - only verbatim subframes.flac",
		"testdata/flac-test-files/subset/16 - partition order 8 containing escaped partitions.flac",
		"testdata/flac-test-files/subset/17 - all fixed orders.flac",
		"testdata/flac-test-files/subset/18 - precision search.flac",
		"testdata/flac-test-files/subset/19 - samplerate 35467Hz.flac",
		"testdata/flac-test-files/subset/20 - samplerate 39kHz.flac",
		"testdata/flac-test-files/subset/21 - samplerate 22050Hz.flac",
		"testdata/flac-test-files/subset/22 - 12 bit per sample.flac",
		"testdata/flac-test-files/subset/23 - 8 bit per sample.flac",
		"testdata/flac-test-files/subset/24 - variable blocksize file created with flake revision 264.flac",
		"testdata/flac-test-files/subset/25 - variable blocksize file created with flake revision 264, modified to create smaller blocks.flac",
		"testdata/flac-test-files/subset/26 - variable blocksize file created with CUETools.Flake 2.1.6.flac",
		"testdata/flac-test-files/subset/27 - old format variable blocksize file created with Flake 0.11.flac",
		"testdata/flac-test-files/subset/28 - high resolution audio, default settings.flac",
		"testdata/flac-test-files/subset/29 - high resolution audio, blocksize 16384.flac",
		"testdata/flac-test-files/subset/30 - high resolution audio, blocksize 13456.flac",
		"testdata/flac-test-files/subset/31 - high resolution audio, using only 32nd order predictors.flac",
		"testdata/flac-test-files/subset/32 - high resolution audio, partition order 8 containing escaped partitions.flac",
		"testdata/flac-test-files/subset/33 - samplerate 192kHz.flac",
		"testdata/flac-test-files/subset/34 - samplerate 192kHz, using only 32nd order predictors.flac",
		"testdata/flac-test-files/subset/35 - samplerate 134560Hz.flac",
		"testdata/flac-test-files/subset/36 - samplerate 384kHz.flac",
		"testdata/flac-test-files/subset/37 - 20 bit per sample.flac",
		"testdata/flac-test-files/subset/38 - 3 channels (3.0).flac",
		"testdata/flac-test-files/subset/39 - 4 channels (4.0).flac",
		"testdata/flac-test-files/subset/40 - 5 channels (5.0).flac",
		"testdata/flac-test-files/subset/41 - 6 channels (5.1).flac",
		"testdata/flac-test-files/subset/42 - 7 channels (6.1).flac",
		"testdata/flac-test-files/subset/43 - 8 channels (7.1).flac",
		"testdata/flac-test-files/subset/44 - 8-channel surround, 192kHz, 24 bit, using only 32nd order predictors.flac",
		"testdata/flac-test-files/subset/45 - no total number of samples set.flac",
		"testdata/flac-test-files/subset/46 - no min-max framesize set.flac",
		"testdata/flac-test-files/subset/47 - only STREAMINFO.flac",
		"testdata/flac-test-files/subset/48 - Extremely large SEEKTABLE.flac",
		"testdata/flac-test-files/subset/49 - Extremely large PADDING.flac",
		"testdata/flac-test-files/subset/50 - Extremely large PICTURE.flac",
		"testdata/flac-test-files/subset/51 - Extremely large VORBISCOMMENT.flac",
		"testdata/flac-test-files/subset/52 - Extremely large APPLICATION.flac",
		"testdata/flac-test-files/subset/53 - CUESHEET with very many indexes.flac",
		"testdata/flac-test-files/subset/54 - 1000x repeating VORBISCOMMENT.flac",
		"testdata/flac-test-files/subset/55 - file 48-53 combined.flac",
		"testdata/flac-test-files/subset/56 - JPG PICTURE.flac",
		"testdata/flac-test-files/subset/57 - PNG PICTURE.flac",
		"testdata/flac-test-files/subset/58 - GIF PICTURE.flac",
		"testdata/flac-test-files/subset/59 - AVIF PICTURE.flac",
		"testdata/flac-test-files/subset/60 - mono audio.flac",
		"testdata/flac-test-files/subset/61 - predictor overflow check, 16-bit.flac",
		"testdata/flac-test-files/subset/62 - predictor overflow check, 20-bit.flac",
		"testdata/flac-test-files/subset/63 - predictor overflow check, 24-bit.flac",
		"testdata/flac-test-files/subset/64 - rice partitions with escape code zero.flac",
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
