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
		"meta/testdata/input-SCPAP.flac",
		"meta/testdata/input-SCVA.flac",
		"meta/testdata/input-SCVPAP.flac",
		"meta/testdata/input-VA.flac",
		"meta/testdata/silence.flac",
		"testdata/19875.flac", // prediction method 3 (FIR)
		"testdata/44127.flac", // prediction method 3 (FIR)
		// TODO: fix diff.
		//"testdata/59996.flac",
		"testdata/80574.flac", // prediction method 3 (FIR)
		// TODO: fix diff.
		//"testdata/172960.flac",
		// TODO: fix diff.
		//"testdata/189983.flac",
		// TODO: fix: invalid number of samples per channel; expected >= 16 && <= 65535, got 1
		//"testdata/191885.flac",
		// TODO: fix diff.
		//"testdata/212768.flac",
		"testdata/220014.flac", // prediction method 2 (Fixed)
		"testdata/243749.flac", // prediction method 2 (Fixed)
		// TODO: fix diff.
		//"testdata/256529.flac",
		"testdata/257344.flac", // prediction method 3 (FIR)
		"testdata/8297-275156-0011.flac", // prediction method 3 (FIR)
		// TODO: fix: constant sample mismatch; expected 126, got 125
		//"testdata/love.flac",
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
