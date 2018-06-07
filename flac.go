// TODO(u): Evaluate storing the samples (and residuals) during frame audio
// decoding in a buffer allocated for the stream. This buffer would be allocated
// using BlockSize and NChannels from the StreamInfo block, and it could be
// reused in between calls to Next and ParseNext. This should reduce GC
// pressure.

// Package flac provides access to FLAC (Free Lossless Audio Codec) streams.
//
// A brief introduction of the FLAC stream format [1] follows. Each FLAC stream
// starts with a 32-bit signature ("fLaC"), followed by one or more metadata
// blocks, and then one or more audio frames. The first metadata block
// (StreamInfo) describes the basic properties of the audio stream and it is the
// only mandatory metadata block. Subsequent metadata blocks may appear in an
// arbitrary order.
//
// Please refer to the documentation of the meta [2] and the frame [3] packages
// for a brief introduction of their respective formats.
//
//    [1]: https://www.xiph.org/flac/format.html#stream
//    [2]: https://godoc.org/github.com/mewkiz/flac/meta
//    [3]: https://godoc.org/github.com/mewkiz/flac/frame
package flac

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// A Stream contains the metadata blocks and provides access to the audio frames
// of a FLAC stream.
//
// ref: https://www.xiph.org/flac/format.html#stream
type Stream struct {
	// The StreamInfo metadata block describes the basic properties of the FLAC
	// audio stream.
	Info *meta.StreamInfo
	// Zero or more metadata blocks.
	Blocks []*meta.Block
	// Underlying io.Reader.
	r io.Reader
	// Underlying io.Closer of file if opened with Open and ParseFile, and nil
	// otherwise.
	c io.Closer
}

// New creates a new Stream for accessing the audio samples of r. It reads and
// parses the FLAC signature and the StreamInfo metadata block, but skips all
// other metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame, and call
// Stream.ParseNext to parse the entire next frame including audio samples.
func New(r io.Reader) (stream *Stream, err error) {
	// Verify FLAC signature and parse the StreamInfo metadata block.
	br := bufio.NewReader(r)
	stream = &Stream{r: br}
	isLast, err := stream.parseStreamInfo()
	if err != nil {
		return nil, err
	}

	// Skip the remaining metadata blocks.
	for !isLast {
		block, err := meta.New(br)
		if err != nil && err != meta.ErrReservedType {
			return stream, err
		}
		if err = block.Skip(); err != nil {
			return stream, err
		}
		isLast = block.IsLast
	}

	return stream, nil
}

// flacSignature marks the beginning of a FLAC stream.
var flacSignature = []byte("fLaC")

// id3Signature marks the beginning of an ID3 stream, used to skip over ID3 data.
var id3Signature = []byte("ID3")

// parseStreamInfo verifies the signature which marks the beginning of a FLAC
// stream, and parses the StreamInfo metadata block. It returns a boolean value
// which specifies if the StreamInfo block was the last metadata block of the
// FLAC stream.
func (stream *Stream) parseStreamInfo() (isLast bool, err error) {
	// Verify FLAC signature.
	r := stream.r
	var buf [4]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return false, err
	}

	// Skip prepended ID3v2 data.
	if bytes.Equal(buf[:3], id3Signature) {
		if err := stream.skipID3v2(); err != nil {
			return false, err
		}

		// Second attempt at verifying signature.
		if _, err = io.ReadFull(r, buf[:]); err != nil {
			return false, err
		}
	}

	if !bytes.Equal(buf[:], flacSignature) {
		return false, fmt.Errorf("flac.parseStreamInfo: invalid FLAC signature; expected %q, got %q", flacSignature, buf)
	}

	// Parse StreamInfo metadata block.
	block, err := meta.Parse(r)
	if err != nil {
		return false, err
	}
	si, ok := block.Body.(*meta.StreamInfo)
	if !ok {
		return false, fmt.Errorf("flac.parseStreamInfo: incorrect type of first metadata block; expected *meta.StreamInfo, got %T", si)
	}
	stream.Info = si
	return block.IsLast, nil
}

// skipID3v2 skips ID3v2 data prepended to flac files.
func (stream *Stream) skipID3v2() error {
	r := bufio.NewReader(stream.r)

	// Discard unnecessary data from the ID3v2 header.
	if _, err := r.Discard(2); err != nil {
		return err
	}

	// Read the size from the ID3v2 header.
	var sizeBuf [4]byte
	if _, err := r.Read(sizeBuf[:]); err != nil {
		return err
	}
	// The size is encoded as a synchsafe integer.
	size := int(sizeBuf[0])<<21 | int(sizeBuf[1])<<14 | int(sizeBuf[2])<<7 | int(sizeBuf[3])

	_, err := r.Discard(size)
	return err
}

// Parse creates a new Stream for accessing the metadata blocks and audio
// samples of r. It reads and parses the FLAC signature and all metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame, and call
// Stream.ParseNext to parse the entire next frame including audio samples.
func Parse(r io.Reader) (stream *Stream, err error) {
	// Verify FLAC signature and parse the StreamInfo metadata block.
	br := bufio.NewReader(r)
	stream = &Stream{r: br}
	isLast, err := stream.parseStreamInfo()
	if err != nil {
		return nil, err
	}

	// Parse the remaining metadata blocks.
	for !isLast {
		block, err := meta.Parse(br)
		if err != nil {
			if err != meta.ErrReservedType {
				return stream, err
			}
			// Skip the body of unknown (reserved) metadata blocks, as stated by
			// the specification.
			//
			// ref: https://www.xiph.org/flac/format.html#format_overview
			if err = block.Skip(); err != nil {
				return stream, err
			}
		}
		stream.Blocks = append(stream.Blocks, block)
		isLast = block.IsLast
	}

	return stream, nil
}

// Open creates a new Stream for accessing the audio samples of path. It reads
// and parses the FLAC signature and the StreamInfo metadata block, but skips
// all other metadata blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame, and call
// Stream.ParseNext to parse the entire next frame including audio samples.
//
// Note: The Close method of the stream must be called when finished using it.
func Open(path string) (stream *Stream, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stream, err = New(f)
	if err != nil {
		return nil, err
	}
	stream.c = f
	return stream, err
}

// ParseFile creates a new Stream for accessing the metadata blocks and audio
// samples of path. It reads and parses the FLAC signature and all metadata
// blocks.
//
// Call Stream.Next to parse the frame header of the next audio frame, and call
// Stream.ParseNext to parse the entire next frame including audio samples.
//
// Note: The Close method of the stream must be called when finished using it.
func ParseFile(path string) (stream *Stream, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stream, err = Parse(f)
	if err != nil {
		return nil, err
	}
	stream.c = f
	return stream, err
}

// Close closes the stream if opened through a call to Open or ParseFile, and
// performs no operation otherwise.
func (stream *Stream) Close() error {
	if stream.c != nil {
		return stream.c.Close()
	}
	return nil
}

// Next parses the frame header of the next audio frame. It returns io.EOF to
// signal a graceful end of FLAC stream.
//
// Call Frame.Parse to parse the audio samples of its subframes.
func (stream *Stream) Next() (f *frame.Frame, err error) {
	return frame.New(stream.r)
}

// ParseNext parses the entire next frame including audio samples. It returns
// io.EOF to signal a graceful end of FLAC stream.
func (stream *Stream) ParseNext() (f *frame.Frame, err error) {
	return frame.Parse(stream.r)
}

// Seek is not implement yet.
func (stream *Stream) Seek(offset int64, whence int) (read int64, err error) {
	return 0, nil
}
