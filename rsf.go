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

// Package rsf (Royal Straight fLaC) implements access to FLAC files.
package rsf

import dbg "fmt"
import "fmt"
import "io"
import "os"

import "github.com/mewkiz/rsf/frame"
import "github.com/mewkiz/rsf/meta"

// A Stream is a FLAC bitstream.
type Stream struct {
	MetaBlocks []*meta.Block
	///Frame      []frame.Frame
}

// Open opens the provided file and returns the parsed FLAC bitstream.
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

// NewStream reads from the provided io.Reader and returns the parsed FLAC
// bitstream.
//
// The basic structure of a FLAC stream is:
//    - The four byte string "fLaC".
//    - The StreamInfo metadata block.
//    - Zero or more other metadata blocks.
//    - One or more audio frames.
func NewStream(r io.ReadSeeker) (s *Stream, err error) {
	// Verify "fLaC" signature (size: 4 bytes).
	buf := make([]byte, 4)
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}
	sig := string(buf)
	if sig != FlacSignature {
		return nil, fmt.Errorf("rsf.NewStream: invalid signature; expected '%s', got '%s'.", FlacSignature, sig)
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

		// Verify block type.
		if isFirst {
			if block.Header.BlockType != meta.TypeStreamInfo {
				// First block type must be StreamInfo.
				return nil, fmt.Errorf("rsf.NewStream: first block type is invalid; expected %d (StreamInfo), got %d.", meta.TypeStreamInfo, block.Header.BlockType)
			}
			isFirst = false
		}

		// Store decoded metadata block.
		s.MetaBlocks = append(s.MetaBlocks, block)
	}

	/// Audio frame parsing.
	/// Flac decoding.

	h, err := frame.NewHeader(r)
	if err != nil {
		return nil, err
	}
	dbg.Printf("frame header: %#v\n", h)

	sh, err := frame.NewSubFrameHeader(r)
	if err != nil {
		return nil, err
	}
	dbg.Printf("subframe header: %#v\n", sh)

	return s, nil
}
