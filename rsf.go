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

//Package rsf (Royal Straight fLaC) is used to extract information from flac files.
package rsf

import "bytes"
import "fmt"
import "github.com/karlek/rsf/frame"
import "github.com/karlek/rsf/meta"
import "io"
import "io/ioutil"
import "os"
import dbg "fmt"

const (
	//The first four bytes of all flac files
	FlacSignature = "fLaC"

	//Formatted error strings
	ErrSignatureMismatch         = "invalid flac signature: %s, should be " + FlacSignature
	ErrStreamInfoIsNotFirstBlock = "invalid first block; the first block must be stream info"
)

//The basic structure of a FLAC stream is:
// - The four byte string `fLaC`
// - The STREAMINFO metadata block
// - Zero or more other metadata blocks
// - One or more audio frames
type Stream struct {
	HasSignature bool
	Metadata     []interface{}
	// Frame    []frame.Frame
}

//Extracts the flac stream from a file
func Open(filePath string) (s *Stream, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s, err = NewStream(f)
	if err != nil {
		return nil, err
	}

	return s, nil
}

//Extract the flac stream from a io.Reader
func NewStream(r io.Reader) (s *Stream, err error) {
	s = new(Stream)

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = s.parse(buf)
	if err != nil {
		return nil, err
	}

	return s, nil
}

///A flac stream is valid if:
///	It has the flac signature `fLaC`
///	The first metadata block is the StreamInfo block
///	All optional metadata blocks are valid
///	It has at least one audio frame
///	All audio frames are valid

//Parse a flac stream to a struct
func (s *Stream) parse(block []byte) (err error) {
	buf := bytes.NewBuffer(block)

	//Check `fLaC` signature (size: 4 bytes)
	signature := string(buf.Next(4))
	if signature != FlacSignature {
		return fmt.Errorf(ErrSignatureMismatch, signature)
	}
	s.HasSignature = true

	//Depending on the type number extraced from the metadata header different parse() methods will execute
	var headerTypes = map[uint8]interface{}{
		0: new(meta.StreamInfo),
		// 1: Padding,
		2: new(meta.Application),
		3: new(meta.SeekTable),
		4: new(meta.VorbisComment),
		5: new(meta.CueSheet),
		6: new(meta.Picture),
	}

	//Read Metadata blocks
	isFirstRun := true
	header := meta.DataHeader{}
	for header.IsLast == false {
		//Read Metadata Header (Size: 4 bytes)
		err = header.Parse(buf.Next(4))
		if err != nil {
			return err
		}

		if isFirstRun && header.BlockType != 0 {
			return fmt.Errorf(ErrStreamInfoIsNotFirstBlock)
		} else {
			isFirstRun = false
		}

		///Might have serious bugs with multiple occurences of the same block type. For instance picture blocks
		//Depending on type of block different parse methods are used (size: depends on header.length)
		switch b := headerTypes[header.BlockType].(type) {
		case (*meta.StreamInfo):
			b.Parse(buf.Next(int(header.Length)))
		case (*meta.Application):
			b.Parse(buf.Next(int(header.Length)))
		case (*meta.SeekTable):
			b.Parse(buf.Next(int(header.Length)))
		case (*meta.VorbisComment):
			b.Parse(buf.Next(int(header.Length)))
		case (*meta.CueSheet):
			b.Parse(buf.Next(int(header.Length)))
		case (*meta.Picture):
			b.Parse(buf.Next(int(header.Length)))
		default: //Only when the block type is padding will this code trigger
			buf.Next(int(header.Length))
			continue
		}

		s.Metadata = append(s.Metadata, headerTypes[header.BlockType])
	}

	///Audio frame parsing
	///Flac decoding

	f, err := frame.Decode(buf.Bytes())
	dbg.Println(f)

	return nil
}
