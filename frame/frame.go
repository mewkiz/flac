// Package frame implements access to FLAC audio frames.
package frame

import "io"

// A Frame contains the header and subframes of an audio frame. It holds the
// encoded samples from a block (a part) of the audio stream. Each subframe
// holding the samples from one of its channel.
//
// ref: https://www.xiph.org/flac/format.html#frame
type Frame struct {
	// Audio frame header.
	Header
	// One subframe per channel, containing encoded audio samples.
	Subframes []Subframe
	// Underlying io.Reader.
	r io.Reader
}

// New creates a new Frame for accessing the audio samples of r. It reads and
// parses an audio frame header. Call Frame.Parse to parse the audio samples of
// its subframes, and Frame.Next to gain access to the next audio frame.
func New(r io.Reader) (frame *Frame, err error) {
	frame = &Frame{r: r}
	err = frame.parseHeader()
	if err != nil {
		return nil, err
	}
	panic("not yet implemented.")
}

// Next returns access to the next audio frame. It reads and parses an audio
// frame header. Call Frame.Parse to parse the audio samples of its subframes.
func (frame *Frame) Next() (next *Frame, err error) {
	panic("not yet implemented.")
}

// Parse reads and parses the audio samples of each subframe. If the samples are
// interchannel correlated between the subframes, it decorrelates them.
//
// ref: https://www.xiph.org/flac/format.html#interchannel
func (frame *Frame) Parse() error {
	panic("not yet implemented.")
}

// A Header contains the basic properties of an audio frame, such as its sample
// rate and channel count. To facilitate random access decoding each frame
// header starts with a sync-code. This allows the decoder to synchronize and
// locate the start of a frame header.
//
// ref: https://www.xiph.org/flac/format.html#frame_header
type Header struct{}

// parseHeader reads and parses the header of an audio frame.
func (frame *Frame) parseHeader() error {
	panic("not yet implemented.")
}
