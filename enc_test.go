package flac_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

func TestEncode(t *testing.T) {
	paths := []string{
		// metadata test cases.
		"meta/testdata/input-SCPAP.flac",
		"meta/testdata/input-SCVA.flac",
		"meta/testdata/input-SCVPAP.flac",
		"meta/testdata/input-VA.flac",
		"meta/testdata/input-SCVAUP.flac", // empty metadata block (of type 0x7e)
		"meta/testdata/input-SVAUP.flac",  // empty metadata block (of type 0x7e)
		"meta/testdata/silence.flac",
		// flac test cases.
		"testdata/19875.flac", // prediction method 3 (FIR)
		"testdata/44127.flac", // prediction method 3 (FIR)
		"testdata/59996.flac",
		"testdata/80574.flac", // prediction method 3 (FIR)
		"testdata/172960.flac",
		"testdata/189983.flac",
		"testdata/191885.flac",
		"testdata/212768.flac",
		"testdata/220014.flac", // prediction method 2 (Fixed)
		"testdata/243749.flac", // prediction method 2 (Fixed)
		"testdata/256529.flac",
		"testdata/257344.flac",           // prediction method 3 (FIR)
		"testdata/8297-275156-0011.flac", // prediction method 3 (FIR)
		"testdata/love.flac",             // wasted bits
		// IETF test cases.
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
		// NOTE: the only diff is that "26 - ...flac" uses `block_size: 0b111
		// (end of header (16 bit))` to encode the block size at the end of the
		// header, whereas mewkiz/flac encodes it directly `block_size: 4096
		// (0b1100)`. Notably, the computed md5 hash of the decoded audio samples
		// is identical (MD5: 3b2939b39ae7369b80451c77865e60c1). Thus, ignore the
		// test case.
		//"testdata/flac-test-files/subset/26 - variable blocksize file created with CUETools.Flake 2.1.6.flac",
		// NOTE: the only diff is that "27 - ...flac" uses `block_size: 0b111
		// (end of header (16 bit))` to encode the block size at the end of the
		// header, whereas mewkiz/flac encodes it directly `block_size: 4608
		// (0b101)`. Notably, the computed md5 hash of the decoded audio samples
		// is identical (MD5: 9fb66177d2f735d4b1f501a5af1320a3). Thus, ignore the
		// test case.
		//"testdata/flac-test-files/subset/27 - old format variable blocksize file created with Flake 0.11.flac",
		"testdata/flac-test-files/subset/28 - high resolution audio, default settings.flac",
		"testdata/flac-test-files/subset/29 - high resolution audio, blocksize 16384.flac",
		"testdata/flac-test-files/subset/30 - high resolution audio, blocksize 13456.flac",
		"testdata/flac-test-files/subset/31 - high resolution audio, using only 32nd order predictors.flac",
		"testdata/flac-test-files/subset/32 - high resolution audio, partition order 8 containing escaped partitions.flac",
		"testdata/flac-test-files/subset/33 - samplerate 192kHz.flac",
		// NOTE: the only diff is that "34 - ...flac" uses `0b1100 (end of header
		// (8 bit*1000))` to encode the sample rate at the end of the header,
		// whereas mewkiz/flac encodes it directly `192000 (0b11)`. Notably, the
		// computed md5 hash of the decoded audio samples is identical
		// (MD5: 942f56e503437dfd4c269c331774b2e3). Thus, ignore the test case.
		//"testdata/flac-test-files/subset/34 - samplerate 192kHz, using only 32nd order predictors.flac",
		"testdata/flac-test-files/subset/35 - samplerate 134560Hz.flac",
		"testdata/flac-test-files/subset/36 - samplerate 384kHz.flac",
		"testdata/flac-test-files/subset/37 - 20 bit per sample.flac",
		"testdata/flac-test-files/subset/38 - 3 channels (3.0).flac",
		"testdata/flac-test-files/subset/39 - 4 channels (4.0).flac",
		"testdata/flac-test-files/subset/40 - 5 channels (5.0).flac",
		"testdata/flac-test-files/subset/41 - 6 channels (5.1).flac",
		"testdata/flac-test-files/subset/42 - 7 channels (6.1).flac",
		"testdata/flac-test-files/subset/43 - 8 channels (7.1).flac",
		// NOTE: the only diff is that "44 - ...flac" uses `0b1100 (end of header
		// (8 bit*1000))` to encode the sample rate at the end of the header,
		// whereas mewkiz/flac encodes it directly `192000 (0b11)`. Notably, the
		// computed md5 hash of the decoded audio samples is identical
		// (MD5: cdf531d4d4b95233986bc499518a89db). Thus, ignore the test case.
		//"testdata/flac-test-files/subset/44 - 8-channel surround, 192kHz, 24 bit, using only 32nd order predictors.flac",
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
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// Decode source file.
			stream, err := flac.ParseFile(path)
			if err != nil {
				t.Fatalf("%q: unable to parse FLAC file; %v", path, err)
			}
			defer stream.Close()

			// Open encoder for FLAC stream.
			out := new(bytes.Buffer)
			enc, err := flac.NewEncoder(out, stream.Info, stream.Blocks...)
			if err != nil {
				t.Fatalf("%q: unable to create encoder for FLAC stream; %v", path, err)
			}
			enc.EnablePredictionAnalysis(false) // disable prediction analysis to support round-trip decode/encode test.
			// Encode audio samples.
			for {
				frame, err := stream.ParseNext()
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatalf("%q: unable to parse audio frame of FLAC stream; %v", path, err)
				}
				if err := enc.WriteFrame(frame); err != nil {
					t.Fatalf("%q: unable to encode audio frame of FLAC stream; %v", path, err)
				}
			}
			// Close encoder and flush pending writes.
			if err := enc.Close(); err != nil {
				t.Fatalf("%q: unable to close encoder for FLAC stream; %v", path, err)
			}

			// Compare source and destination FLAC streams.
			want, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatalf("%q: unable to read file; %v", path, err)
			}
			got := out.Bytes()
			if !bytes.Equal(got, want) {
				t.Fatalf("%q: content mismatch; expected % X, got % X", path, want, got)
			}
		})
	}
}

func TestEncodeComment(t *testing.T) {
	// Decode FLAC file.
	const path = "meta/testdata/input-VA.flac"
	src, err := flac.ParseFile(path)
	if err != nil {
		t.Fatalf("unable to parse input FLAC file; %v", err)
	}
	defer src.Close()

	// Add custom vorbis comment.
	const want = "FLAC encoding test case"
	for _, block := range src.Blocks {
		if comment, ok := block.Body.(*meta.VorbisComment); ok {
			comment.Vendor = want
		}
	}

	// Open encoder for FLAC stream.
	out := new(bytes.Buffer)
	enc, err := flac.NewEncoder(out, src.Info, src.Blocks...)
	if err != nil {
		t.Fatalf("%q: unable to create encoder for FLAC stream; %v", path, err)
	}
	// Encode audio samples.
	for {
		frame, err := src.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("%q: unable to parse audio frame of FLAC stream; %v", path, err)
		}
		if err := enc.WriteFrame(frame); err != nil {
			t.Fatalf("%q: unable to encode audio frame of FLAC stream; %v", path, err)
		}
	}
	// Close encoder and flush pending writes.
	if err := enc.Close(); err != nil {
		t.Fatalf("%q: unable to close encoder for FLAC stream; %v", path, err)
	}

	// Parse encoded FLAC file.
	stream, err := flac.Parse(out)
	if err != nil {
		t.Fatalf("unable to parse output FLAC file; %v", err)
	}
	defer stream.Close()

	// Add custom vorbis comment.
	for _, block := range stream.Blocks {
		if comment, ok := block.Body.(*meta.VorbisComment); ok {
			got := comment.Vendor
			if got != want {
				t.Errorf("Vorbis comment mismatch; expected %q, got %q", want, got)
				continue
			}
		}
	}
}
