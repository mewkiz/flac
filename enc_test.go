//+build ignore

package flac_test

import (
	"bytes"
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
	for _, path := range paths {
		// Decode source file.
		stream, err := flac.ParseFile(path)
		if err != nil {
			t.Errorf("%q: unable to parse FLAC file; %v", path, err)
			continue
		}
		defer stream.Close()

		// Encode FLAC stream.
		out := new(bytes.Buffer)
		if err := flac.Encode(out, stream); err != nil {
			t.Errorf("%q: unable to encode FLAC stream; %v", path, err)
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
	src, err := flac.ParseFile("testdata/love.flac")
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

	// Encode FLAC file.
	out := new(bytes.Buffer)
	if err := flac.Encode(out, src); err != nil {
		t.Fatalf("unable to encode FLAC file; %v", err)
	}

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
