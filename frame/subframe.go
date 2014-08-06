package frame

import (
	"errors"
	"fmt"
	"io"

	"github.com/mewkiz/pkg/bit"
	"github.com/mewkiz/pkg/bitutil"
)

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

// parseSubframe reads and parses the header, and the audio samples of a
// subframe.
func (frame *Frame) parseSubframe() (subframe *Subframe, err error) {
	subframe = &Subframe{r: frame.hr}
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
	// Prediction order used by fixed and LPC decoding.
	Order uint8
}

// parseHeader reads and parses the header of a subframe.
func (subframe *Subframe) parseHeader() error {
	// 1 bit: zero-padding.
	br := bit.NewReader(subframe.r)
	x, err := br.Read(1)
	if err != nil {
		return err
	}
	if x != 0 {
		return errors.New("frame.Subframe.parseHeader: non-zero padding")
	}

	// 6 bits: Pred.
	x, err = br.Read(6)
	if err != nil {
		return err
	}
	// The 6 bits are used to specify the prediction method and order as follows:
	//    000000: Constant prediction method.
	//    000001: Verbatim prediction method.
	//    00001x: reserved.
	//    0001xx: reserved.
	//    001xxx:
	//       if (xxx <= 4)
	//          Fixed prediction method; xxx=order
	//       else
	//          reserved.
	//    01xxxx: reserved.
	//    1xxxxx: LPC prediction method; xxxxx=order-1
	switch {
	case x < 1:
		// 000000: Constant prediction method.
		subframe.Pred = PredConstant
	case x < 2:
		// 000001: Verbatim prediction method.
		subframe.Pred = PredVerbatim
	case x < 8:
		// 00001x: reserved.
		// 0001xx: reserved.
		return fmt.Errorf("frame.Subframe.parseHeader: reserved prediction method bit pattern (%06b)", x)
	case x < 16:
		// 001xxx:
		//    if (xxx <= 4)
		//       Fixed prediction method; xxx=order
		//    else
		//       reserved.
		order := uint8(x & 0x07)
		if order > 4 {
			return fmt.Errorf("frame.Subframe.parseHeader: reserved prediction method bit pattern (%06b)", x)
		}
		subframe.Pred = PredFixed
		subframe.Order = order
	case x < 32:
		// 01xxxx: reserved.
		return fmt.Errorf("frame.Subframe.parseHeader: reserved prediction method bit pattern (%06b)", x)
	default:
		// 1xxxxx: LPC prediction method; xxxxx=order-1
		subframe.Pred = PredLPC
		subframe.Order = uint8(x&0x1F) + 1
	}

	// 1 bit: hasWastedBits.
	x, err = br.Read(1)
	if err != nil {
		return err
	}
	if x != 0 {
		// The number of wasted bits-per-sample is unary coded.
		_, err = bitutil.DecodeUnary(br)
		if err != nil {
			return err
		}
		panic("Never seen a FLAC file contain wasted-bits-per-sample before. Not really a reason to panic, but I want to dissect one of those files. Please send it to me :)")
	}

	return nil
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
