package frame_test

import (
	"bytes"
	"crypto/md5"
	"io"
	"testing"

	"github.com/mewkiz/flac"
)

var golden = []struct {
	name string
}{
	{name: "../testdata/love.flac"},   // i=0
	{name: "../testdata/19875.flac"},  // i=1
	{name: "../testdata/44127.flac"},  // i=2
	{name: "../testdata/59996.flac"},  // i=3
	{name: "../testdata/80574.flac"},  // i=4
	{name: "../testdata/172960.flac"}, // i=5
	{name: "../testdata/189983.flac"}, // i=6
	{name: "../testdata/191885.flac"}, // i=7
	{name: "../testdata/212768.flac"}, // i=8
	{name: "../testdata/220014.flac"}, // i=9
	{name: "../testdata/243749.flac"}, // i=10
	{name: "../testdata/256529.flac"}, // i=11
	{name: "../testdata/257344.flac"}, // i=12
}

func TestFrameHash(t *testing.T) {
	for i, g := range golden {
		stream, err := flac.Open(g.name)
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
				t.Errorf("i=%d, frameNum=%d: error while parsing frame; %v", i, frameNum, err)
				continue
			}
			frame.Hash(md5sum)
		}
		want := stream.Info.MD5sum[:]
		got := md5sum.Sum(nil)
		// Verify the decoded audio samples by comparing the MD5 checksum that is
		// stored in StreamInfo with the computed one.
		if !bytes.Equal(got, want) {
			t.Errorf("i=%d: MD5 checksum mismatch for decoded audio samples; expected %32x, got %32x", i, want, got)
		}
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
