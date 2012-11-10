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

import "fmt"
import "io"
import "os"

///import "github.com/mewkiz/rsf/frame"
import "github.com/mewkiz/rsf/meta"

// A Stream is a FLAC bitstream.
type Stream struct {
	MetaBlocks []MetaBlock
	///Frame      []frame.Frame
}

type MetaBlock struct {
	Header *meta.BlockHeader
	Body   interface{}
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
		var block MetaBlock
		// Read metadata block header.
		block.Header, err = meta.NewBlockHeader(r)
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

		// Read metadata block.
		lr := &io.LimitedReader{
			R: r,
			N: int64(block.Header.Length),
		}
		switch block.Header.BlockType {
		case meta.TypeStreamInfo:
			block.Body, err = meta.NewStreamInfo(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		case meta.TypePadding:
			err = meta.VerifyPadding(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		case meta.TypeApplication:
			block.Body, err = meta.NewApplication(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		case meta.TypeSeekTable:
			block.Body, err = meta.NewSeekTable(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		case meta.TypeVorbisComment:
			block.Body, err = meta.NewVorbisComment(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		case meta.TypeCueSheet:
			block.Body, err = meta.NewCueSheet(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		case meta.TypePicture:
			block.Body, err = meta.NewPicture(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, block)
		default:
			return nil, fmt.Errorf("block type '%d' not yet supported.", block.Header.BlockType)
		}
	}

	/// Audio frame parsing.
	/// Flac decoding.

	/**
	f, err := frame.Decode(r)
	if err != nil {
		return nil, err
	}
	dbg.Println(f)
	*/

	return s, nil
}
