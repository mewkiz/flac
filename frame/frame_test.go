package frame_test

import (
	"bytes"
	"crypto/md5"
	"io"
	"testing"

	"github.com/mewkiz/flac"
)

var golden = []struct {
	path string
}{
	{path: "../testdata/love.flac"},
	{path: "../testdata/19875.flac"},
	{path: "../testdata/44127.flac"},
	{path: "../testdata/59996.flac"},
	{path: "../testdata/80574.flac"},
	{path: "../testdata/172960.flac"},
	{path: "../testdata/189983.flac"},
	{path: "../testdata/191885.flac"},
	{path: "../testdata/212768.flac"},
	{path: "../testdata/220014.flac"},
	{path: "../testdata/243749.flac"},
	{path: "../testdata/256529.flac"},
	{path: "../testdata/257344.flac"},
	{path: "../testdata/24000-tts-sf.flac"},

	// IETF test cases.
	{path: "../testdata/flac-test-files/subset/01 - blocksize 4096.flac"},
	{path: "../testdata/flac-test-files/subset/02 - blocksize 4608.flac"},
	{path: "../testdata/flac-test-files/subset/03 - blocksize 16.flac"},
	{path: "../testdata/flac-test-files/subset/04 - blocksize 192.flac"},
	{path: "../testdata/flac-test-files/subset/05 - blocksize 254.flac"},
	{path: "../testdata/flac-test-files/subset/06 - blocksize 512.flac"},
	{path: "../testdata/flac-test-files/subset/07 - blocksize 725.flac"},
	{path: "../testdata/flac-test-files/subset/08 - blocksize 1000.flac"},
	{path: "../testdata/flac-test-files/subset/09 - blocksize 1937.flac"},
	{path: "../testdata/flac-test-files/subset/10 - blocksize 2304.flac"},
	{path: "../testdata/flac-test-files/subset/11 - partition order 8.flac"},
	{path: "../testdata/flac-test-files/subset/12 - qlp precision 15 bit.flac"},
	{path: "../testdata/flac-test-files/subset/13 - qlp precision 2 bit.flac"},
	{path: "../testdata/flac-test-files/subset/14 - wasted bits.flac"},
	{path: "../testdata/flac-test-files/subset/15 - only verbatim subframes.flac"},
	{path: "../testdata/flac-test-files/subset/16 - partition order 8 containing escaped partitions.flac"},
	{path: "../testdata/flac-test-files/subset/17 - all fixed orders.flac"},
	{path: "../testdata/flac-test-files/subset/18 - precision search.flac"},
	{path: "../testdata/flac-test-files/subset/19 - samplerate 35467Hz.flac"},
	{path: "../testdata/flac-test-files/subset/20 - samplerate 39kHz.flac"},
	{path: "../testdata/flac-test-files/subset/21 - samplerate 22050Hz.flac"},
	{path: "../testdata/flac-test-files/subset/22 - 12 bit per sample.flac"},
	{path: "../testdata/flac-test-files/subset/23 - 8 bit per sample.flac"},
	{path: "../testdata/flac-test-files/subset/24 - variable blocksize file created with flake revision 264.flac"},
	{path: "../testdata/flac-test-files/subset/25 - variable blocksize file created with flake revision 264, modified to create smaller blocks.flac"},
	{path: "../testdata/flac-test-files/subset/26 - variable blocksize file created with CUETools.Flake 2.1.6.flac"},
	{path: "../testdata/flac-test-files/subset/27 - old format variable blocksize file created with Flake 0.11.flac"},
	{path: "../testdata/flac-test-files/subset/28 - high resolution audio, default settings.flac"},
	{path: "../testdata/flac-test-files/subset/29 - high resolution audio, blocksize 16384.flac"},
	{path: "../testdata/flac-test-files/subset/30 - high resolution audio, blocksize 13456.flac"},
	{path: "../testdata/flac-test-files/subset/31 - high resolution audio, using only 32nd order predictors.flac"},
	{path: "../testdata/flac-test-files/subset/32 - high resolution audio, partition order 8 containing escaped partitions.flac"},
	{path: "../testdata/flac-test-files/subset/33 - samplerate 192kHz.flac"},
	{path: "../testdata/flac-test-files/subset/34 - samplerate 192kHz, using only 32nd order predictors.flac"},
	{path: "../testdata/flac-test-files/subset/35 - samplerate 134560Hz.flac"},
	{path: "../testdata/flac-test-files/subset/36 - samplerate 384kHz.flac"},
	{path: "../testdata/flac-test-files/subset/37 - 20 bit per sample.flac"},
	{path: "../testdata/flac-test-files/subset/38 - 3 channels (3.0).flac"},
	{path: "../testdata/flac-test-files/subset/39 - 4 channels (4.0).flac"},
	{path: "../testdata/flac-test-files/subset/40 - 5 channels (5.0).flac"},
	{path: "../testdata/flac-test-files/subset/41 - 6 channels (5.1).flac"},
	{path: "../testdata/flac-test-files/subset/42 - 7 channels (6.1).flac"},
	{path: "../testdata/flac-test-files/subset/43 - 8 channels (7.1).flac"},
	{path: "../testdata/flac-test-files/subset/44 - 8-channel surround, 192kHz, 24 bit, using only 32nd order predictors.flac"},
	{path: "../testdata/flac-test-files/subset/45 - no total number of samples set.flac"},
	{path: "../testdata/flac-test-files/subset/46 - no min-max framesize set.flac"},
	{path: "../testdata/flac-test-files/subset/47 - only STREAMINFO.flac"},
	{path: "../testdata/flac-test-files/subset/48 - Extremely large SEEKTABLE.flac"},
	{path: "../testdata/flac-test-files/subset/49 - Extremely large PADDING.flac"},
	{path: "../testdata/flac-test-files/subset/50 - Extremely large PICTURE.flac"},
	{path: "../testdata/flac-test-files/subset/51 - Extremely large VORBISCOMMENT.flac"},
	{path: "../testdata/flac-test-files/subset/52 - Extremely large APPLICATION.flac"},
	{path: "../testdata/flac-test-files/subset/53 - CUESHEET with very many indexes.flac"},
	{path: "../testdata/flac-test-files/subset/54 - 1000x repeating VORBISCOMMENT.flac"},
	{path: "../testdata/flac-test-files/subset/55 - file 48-53 combined.flac"},
	{path: "../testdata/flac-test-files/subset/56 - JPG PICTURE.flac"},
	{path: "../testdata/flac-test-files/subset/57 - PNG PICTURE.flac"},
	{path: "../testdata/flac-test-files/subset/58 - GIF PICTURE.flac"},
	{path: "../testdata/flac-test-files/subset/59 - AVIF PICTURE.flac"},
	{path: "../testdata/flac-test-files/subset/60 - mono audio.flac"},
	{path: "../testdata/flac-test-files/subset/61 - predictor overflow check, 16-bit.flac"},
	{path: "../testdata/flac-test-files/subset/62 - predictor overflow check, 20-bit.flac"},
	// TODO: fix decoding of "subset/63 - ...flac": MD5 checksum mismatch for decoded audio samples; expected e4e4a6b3a672a849a3e2157c11ad23c6, got a0343afaaaa6229266d78ccf3175eb8d
	{path: "../testdata/flac-test-files/subset/63 - predictor overflow check, 24-bit.flac"},
	{path: "../testdata/flac-test-files/subset/64 - rice partitions with escape code zero.flac"},
}

func TestFrameHash(t *testing.T) {
	for _, g := range golden {
		t.Run(g.path, func(t *testing.T) {
			stream, err := flac.Open(g.path)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()

			md5sum := md5.New()
			for frameNum := 0; ; frameNum++ {
				frame, err := stream.ParseNext()
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Errorf("path=%q, frameNum=%d: error while parsing frame; %v", g.path, frameNum, err)
					continue
				}
				frame.Hash(md5sum)
			}
			want := stream.Info.MD5sum[:]
			got := md5sum.Sum(nil)
			// Verify the decoded audio samples by comparing the MD5 checksum that is
			// stored in StreamInfo with the computed one.
			if !bytes.Equal(got, want) {
				t.Errorf("path=%q: MD5 checksum mismatch for decoded audio samples; expected %32x, got %32x", g.path, want, got)
			}
		})
	}
}

func BenchmarkFrameParse(b *testing.B) {
	// The file 151185.flac is a 119.5 MB public domain FLAC file used to
	// benchmark the flac library. Because of its size, it has not been included
	// in the repository, but is available for download at
	//
	//    http://freesound.org/people/jarfil/sounds/151185/
	for i := 0; i < b.N; i++ {
		stream, err := flac.Open("../testdata/benchmark/151185.flac")
		if err != nil {
			b.Fatal(err)
		}
		for {
			_, err := stream.ParseNext()
			if err != nil {
				if err == io.EOF {
					break
				}
				stream.Close()
				b.Fatal(err)
			}
		}
		stream.Close()
	}
}

func BenchmarkFrameHash(b *testing.B) {
	// The file 151185.flac is a 119.5 MB public domain FLAC file used to
	// benchmark the flac library. Because of its size, it has not been included
	// in the repository, but is available for download at
	//
	//    http://freesound.org/people/jarfil/sounds/151185/
	for i := 0; i < b.N; i++ {
		stream, err := flac.Open("../testdata/benchmark/151185.flac")
		if err != nil {
			b.Fatal(err)
		}
		md5sum := md5.New()
		for {
			frame, err := stream.ParseNext()
			if err != nil {
				if err == io.EOF {
					break
				}
				stream.Close()
				b.Fatal(err)
			}
			frame.Hash(md5sum)
		}
		stream.Close()
		want := stream.Info.MD5sum[:]
		got := md5sum.Sum(nil)
		// Verify the decoded audio samples by comparing the MD5 checksum that is
		// stored in StreamInfo with the computed one.
		if !bytes.Equal(got, want) {
			b.Fatalf("MD5 checksum mismatch for decoded audio samples; expected %32x, got %32x", want, got)
		}
	}
}
