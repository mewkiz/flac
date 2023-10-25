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
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/19875.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/44127.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/59996.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/80574.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/172960.flac",
		// TODO: fix: support for prediction method 2 not yet implemented
		//"testdata/189983.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/191885.flac",
		// TODO: fix: support for prediction method 2 not yet implemented
		//"testdata/212768.flac",
		// TODO: fix: support for prediction method 2 not yet implemented
		//"testdata/220014.flac",
		// TODO: fix: support for prediction method 2 not yet implemented
		//"testdata/243749.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/256529.flac",
		// TODO: fix: support for prediction method 3 not yet implemented
		//"testdata/257344.flac",
		// TODO: fix: support for prediction method 2 not yet implemented
		//"testdata/8297-275156-0011.flac",
		// TODO: fix: support for prediction method 2 not yet implemented
		//"testdata/love.flac",
	}
loop:
	for _, path := range paths {
		// Decode source file.
		stream, err := flac.ParseFile(path)
		if err != nil {
			t.Errorf("%q: unable to parse FLAC file; %v", path, err)
			continue
		}
		defer stream.Close()

		// Open encoder for FLAC stream.
		out := new(bytes.Buffer)
		enc, err := flac.NewEncoder(out, stream.Info, stream.Blocks...)
		if err != nil {
			t.Errorf("%q: unable to create encoder for FLAC stream; %v", path, err)
			continue
		}
		// Encode audio samples.
		for {
			frame, err := stream.ParseNext()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("%q: unable to parse audio frame of FLAC stream; %v", path, err)
				continue loop
			}
			if err := enc.WriteFrame(frame); err != nil {
				t.Errorf("%q: unable to encode audio frame of FLAC stream; %v", path, err)
				continue loop
			}
		}
		// Close encoder and flush pending writes.
		if err := enc.Close(); err != nil {
			t.Errorf("%q: unable to close encoder for FLAC stream; %v", path, err)
			continue
		}

		// Compare source and destination FLAC streams.
		want, err := ioutil.ReadFile(path)
		if err != nil {
			t.Errorf("%q: unable to read file; %v", path, err)
			continue
		}
		got := out.Bytes()
		if !bytes.Equal(got, want) {
			t.Errorf("%q: content mismatch; expected % X, got % X", path, want, got)
			continue
		}
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
