/*
Links:
	http://code.google.com/p/goflac-meta/source/browse/flacmeta.go
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
	MetaBlocks []interface{}
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
	header := new(meta.BlockHeader)
	for !header.IsLast {
		// Read metadata block header.
		header, err = meta.NewBlockHeader(r)
		if err != nil {
			return nil, err
		}

		// Verify block type.
		if isFirst {
			if header.BlockType != meta.TypeStreamInfo {
				// First block type must be StreamInfo.
				return nil, fmt.Errorf("rsf.NewStream: first block type is invalid; expected %d (StreamInfo), got %d.", meta.TypeStreamInfo, header.BlockType)
			}
			isFirst = false
		}

		// Read metadata block.
		lr := &io.LimitedReader{
			R: r,
			N: int64(header.Length),
		}
		switch header.BlockType {
		case meta.TypeStreamInfo:
			si, err := meta.NewStreamInfo(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		case meta.TypePadding:
			err = meta.VerifyPadding(lr)
			if err != nil {
				return nil, err
			}
		case meta.TypeApplication:
			ap, err := meta.NewApplication(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, ap)
		case meta.TypeSeekTable:
			st, err := meta.NewSeekTable(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, st)
		case meta.TypeVorbisComment:
			vc, err := meta.NewVorbisComment(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, vc)
		case meta.TypeCueSheet:
			cs, err := meta.NewCueSheet(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, cs)
		case meta.TypePicture:
			p, err := meta.NewPicture(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, p)
		default:
			return nil, fmt.Errorf("block type '%d' not yet supported.", header.BlockType)
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
