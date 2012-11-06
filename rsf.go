/*
Todo:
	Add padding IsAllZero check?
	Change NewStream() ioutil.ReadAll to bufio.NewReader
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

import "github.com/mewkiz/rsf/frame"
import "github.com/mewkiz/rsf/meta"

// FlacSignature is present at the beginning of each FLAC file.
const FlacSignature = "fLaC"

// Formatted error strings.
const (
	ErrSignatureMismatch         = "invalid flac signature: %s, should be " + FlacSignature
	ErrStreamInfoIsNotFirstBlock = "first block type is invalid: expected '%d' (StreamInfo), got '%d'."
)

// A Stream is a FLAC bitstream.
type Stream struct {
	MetaBlocks []interface{}
	//Frame      []frame.Frame
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

// NewStream reads from the provided io.Reader and returns the parsed FLAC
// bitstream.
//
// The basic structure of a FLAC stream is:
//    - The four byte string "fLaC".
//    - The STREAMINFO metadata block.
//    - Zero or more other metadata blocks.
//    - One or more audio frames.
func NewStream(r io.ReadSeeker) (s *Stream, err error) {
	// Check "fLaC" signature (size: 4 bytes).
	sig := make([]byte, 4)
	_, err = r.Read(sig)
	if err != nil {
		return nil, err
	}
	if string(sig) != FlacSignature {
		return nil, fmt.Errorf(ErrSignatureMismatch, sig)
	}

	s = new(Stream)

	// Read metadata blocks.
	isFirst := true
	header := new(meta.BlockHeader)
	for !header.IsLast {
		// Read metadata block header.
		header, err = meta.NewBlockHeader(r)
		if err != nil {
			return nil, err
		}
		if isFirst {
			if header.BlockType != meta.TypeStreamInfo {
				// first block type has to be StreamInfo
				return nil, fmt.Errorf(ErrStreamInfoIsNotFirstBlock, meta.TypeStreamInfo, header.BlockType)
			}
			isFirst = false
		}
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
			st, err := meta.NewApplication(lr)
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, st)
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
		/**case meta.TypeCueSheet:
			newBlock = meta.NewCueSheet
		case meta.TypePicture:
			newBlock = meta.NewPicture*/
		default:
			return nil, fmt.Errorf("block type '%d' not yet supported.", header.BlockType)
		}
	}

	///Audio frame parsing
	///Flac decoding

	f, err := frame.Decode(r)
	if err != nil {
		return nil, err
	}
	dbg.Println(f)

	return s, nil
}
