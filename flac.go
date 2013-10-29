/*
Links:
	http://code.google.com/p/goflac-meta/source/browse/flacmeta_test.go
	http://flac.sourceforge.net/api/hierarchy.html
	http://flac.sourceforge.net/documentation_format_overview.html
	http://flac.sourceforge.net/format.html
	http://jflac.sourceforge.net/
	http://ffmpeg.org/doxygen/trunk/libavcodec_2flacdec_8c-source.html#l00485
	http://mi.eng.cam.ac.uk/reports/svr-ftp/auto-pdf/robinson_tr156.pdf
*/

// Package flac provides access to FLAC [1] (Free Lossless Audio Codec) files.
//
// [1]: http://flac.sourceforge.net/format.html
package flac

import (
	"fmt"
	"io"
	"os"

	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// A Stream is a FLAC bitstream.
type Stream struct {
	// Metadata blocks.
	MetaBlocks []*meta.Block
	// Audio frames.
	Frames []*frame.Frame
}

// Open opens the provided file and returns a parsed FLAC bitstream.
func Open(filePath string) (s *Stream, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return NewStream(f)
}

// FlacSignature is present at the beginning of each FLAC file.
const FlacSignature = "fLaC"

// NewStream reads from the provided io.ReadSeeker and returns a parsed FLAC
// bitstream.
//
// The basic structure of a FLAC stream is:
//    - The four byte string "fLaC".
//    - The STREAMINFO metadata block.
//    - Zero or more other metadata blocks.
//    - One or more audio frames.
func NewStream(r io.ReadSeeker) (s *Stream, err error) {
	// Verify "fLaC" signature (size: 4 bytes).
	buf := make([]byte, 4)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	sig := string(buf)
	if sig != FlacSignature {
		return nil, fmt.Errorf("flac.NewStream: invalid signature; expected %q, got %q", FlacSignature, sig)
	}

	// Read metadata blocks.
	s = new(Stream)
	isFirst := true
	var isLast bool
	for !isLast {
		// Read metadata block.
		block, err := meta.NewBlock(r)
		if err != nil {
			return nil, err
		}
		if block.Header.IsLast {
			isLast = true
		}

		// The first block type must be StreamInfo.
		if isFirst {
			if block.Header.BlockType != meta.TypeStreamInfo {
				return nil, fmt.Errorf("flac.NewStream: first block type is invalid; expected %d (StreamInfo), got %d", meta.TypeStreamInfo, block.Header.BlockType)
			}
			isFirst = false
		}

		// Store the decoded metadata block.
		s.MetaBlocks = append(s.MetaBlocks, block)
	}

	// The first block is always a StreamInfo block.
	si := s.MetaBlocks[0].Body.(*meta.StreamInfo)

	// Read audio frames.
	/// ### [ todo ] ###
	///   - check for int overflow.
	/// ### [/ todo ] ###
	var i uint64
	for i < si.SampleCount {
		f, err := frame.NewFrame(r)
		if err != nil {
			return nil, err
		}
		s.Frames = append(s.Frames, f)
		i += uint64(len(f.SubFrames[0].Samples))
	}

	return s, nil
}
