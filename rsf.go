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

import "bytes"
import dbg "fmt"
import "fmt"
import "io"
import "io/ioutil"
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
func NewStream(r io.Reader) (s *Stream, err error) {
	s = new(Stream)

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	s, err = newStream(buf)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// newStream parses a FLAC bitstream and returns a new Stream. The basic
// structure of a FLAC stream is:
//    - The four byte string "fLaC".
//    - The STREAMINFO metadata block.
//    - Zero or more other metadata blocks.
//    - One or more audio frames.
func newStream(buf []byte) (s *Stream, err error) {
	b := bytes.NewBuffer(buf)

	// Check "fLaC" signature (size: 4 bytes).
	signature := string(b.Next(4))
	if signature != FlacSignature {
		return nil, fmt.Errorf(ErrSignatureMismatch, signature)
	}

	s = new(Stream)

	// Read metadata blocks.
	isFirst := true
	for {
		// Read metadata block header (size: 4 bytes).
		header, err := meta.NewBlockHeader(b.Next(4))
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
		switch header.BlockType {
		case meta.TypeStreamInfo:
			si, err := meta.NewStreamInfo(b.Next(header.Length))
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		case meta.TypePadding:
			// skip padding.
			b.Next(header.Length)
		case meta.TypeApplication:
			si, err := meta.NewApplication(b.Next(header.Length))
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		case meta.TypeSeekTable:
			si, err := meta.NewSeekTable(b.Next(header.Length))
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		case meta.TypeVorbisComment:
			si, err := meta.NewVorbisComment(b.Next(header.Length))
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		case meta.TypeCueSheet:
			si, err := meta.NewCueSheet(b.Next(header.Length))
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		case meta.TypePicture:
			si, err := meta.NewPicture(b.Next(header.Length))
			if err != nil {
				return nil, err
			}
			s.MetaBlocks = append(s.MetaBlocks, si)
		default:
			return nil, fmt.Errorf("block type '%d' not yet supported.", header.BlockType)
		}

		if header.IsLast {
			// Break after last metadata block.
			break
		}
	}

	///Audio frame parsing
	///Flac decoding

	f, err := frame.Decode(b.Bytes())
	dbg.Println(f)

	return s, nil
}
