package frame

import "io"

// A Subframe contains the encoded audio samples from one channel of an audio
// block (a part of the audio stream).
//
// ref: https://www.xiph.org/flac/format.html#subframe
type Subframe struct {
	// Subframe header.
	SubHeader
	// Unencoded audio samples. Samples is initially nil, and gets populated by a
	// call to Frame.Parse.
	Samples []int32
	// Underlying io.Reader.
	r io.Reader
}

func (frame *Frame) parseSubframe() (subframe *Subframe, err error) {
	err = subframe.parseHeader()
	if err != nil {
		return subframe, err
	}
	panic("not yet implemented.")
}

// A SubHeader specifies the prediction method and order of a subframe.
//
// ref: https://www.xiph.org/flac/format.html#subframe_header
type SubHeader struct {
	// Specifies the prediction method used to encode the audio sample of the
	// subframe.
	Pred Pred
}

func (subframe *Subframe) parseHeader() error {
	panic("not yet implemented.")
}

// Pred specifies the prediction method used to encode the audio samples of a
// subframe.
type Pred uint8

// Prediction methods.
const (
	PredConstant Pred = iota
	PredVerbatim
	PredFixed
	PredLPC
)
