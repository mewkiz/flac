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
	// Number of audio samples in the subframe.
	NSamples int
	// A bit reader, wrapping read operations to r.
	br *bit.Reader
	// Underlying io.Reader.
	r io.Reader
}

// parseSubframe reads and parses the header, and the audio samples of a
// subframe.
func (frame *Frame) parseSubframe(bps uint) (subframe *Subframe, err error) {
	// Parse subframe header.
	br := subframe.br
	subframe = &Subframe{br: br, r: frame.hr}
	err = subframe.parseHeader()
	if err != nil {
		return subframe, err
	}

	// Decode subframe audio samples.
	subframe.NSamples = int(frame.BlockSize)
	subframe.Samples = make([]int32, 0, subframe.NSamples)
	switch subframe.Pred {
	case PredConstant:
		err = subframe.decodeConstant(bps)
	case PredVerbatim:
		err = subframe.decodeVerbatim(bps)
	case PredFixed:
		err = subframe.decodeFixed(bps)
	case PredLPC:
		err = subframe.decodeLPC()
	}
	return subframe, err
}

// A SubHeader specifies the prediction method and order of a subframe.
//
// ref: https://www.xiph.org/flac/format.html#subframe_header
type SubHeader struct {
	// Specifies the prediction method used to encode the audio sample of the
	// subframe.
	Pred Pred
	// Prediction order used by fixed and LPC decoding.
	Order int
}

// parseHeader reads and parses the header of a subframe.
func (subframe *Subframe) parseHeader() error {
	// 1 bit: zero-padding.
	br := subframe.br
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
		order := int(x & 0x07)
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
		subframe.Order = int(x&0x1F) + 1
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

// signExtend interprets x as a signed n-bit integer value and sign extends it
// to 32 bits.
func signExtend(x uint64, n uint) int32 {
	// x is signed if its most significant bit is set.
	if x&(1<<(n-1)) != 0 {
		// Sign extend x.
		return int32(x | ^uint64(0)<<n)
	}
	return int32(x)
}

// decodeConstant reads an unencoded audio sample of the subframe. Each sample
// of the subframe has this constant value. The constant encoding can be thought
// of as run-length encoding.
//
// ref: https://www.xiph.org/flac/format.html#subframe_constant
func (subframe *Subframe) decodeConstant(bps uint) error {
	// (bits-per-sample) bits: Unencoded constant value of the subblock.
	br := subframe.br
	x, err := br.Read(bps)
	if err != nil {
		return err
	}

	// Each sample of the subframe has the same constant value.
	sample := signExtend(x, bps)
	for i := 0; i < subframe.NSamples; i++ {
		subframe.Samples = append(subframe.Samples, sample)
	}

	return nil
}

// decodeVerbatim reads the unencoded audio samples of the subframe.
//
// ref: https://www.xiph.org/flac/format.html#subframe_verbatim
func (subframe *Subframe) decodeVerbatim(bps uint) error {
	// Parse the unencoded audio samples of the subframe.
	br := subframe.br
	for i := 0; i < subframe.NSamples; i++ {
		// (bits-per-sample) bits: Unencoded constant value of the subblock.
		x, err := br.Read(bps)
		if err != nil {
			return err
		}
		sample := signExtend(x, bps)
		subframe.Samples = append(subframe.Samples, sample)
	}
	return nil
}

// fixedCoeffs maps from prediction order to the LPC coefficients used in fixed
// encoding.
//
//    x_0[n] = 0
//    x_1[n] = x[n-1]
//    x_2[n] = 2*x[n-1] - x[n-2]
//    x_3[n] = 3*x[n-1] - 3*x[n-2] + x[n-3]
//    x_4[n] = 4*x[n-1] - 6*x[n-2] + 4*x[n-3] - x[n-4]
var fixedCoeffs = [...][]int32{
	// ref: Section 2.2 of http://www.hpl.hp.com/techreports/1999/HPL-1999-144.pdf
	1: {1},
	2: {2, -1},
	3: {3, -3, 1},
	// ref: Data Compression: The Complete Reference (7.10.1)
	4: {4, -6, 4, -1},
}

// decodeFixed decodes the linear prediction coded samples of the subframe,
// using a fixed set of predefined polynomial coefficients.
//
// ref: https://www.xiph.org/flac/format.html#subframe_fixed
func (subframe *Subframe) decodeFixed(bps uint) error {
	// Parse unencoded warmup samples.
	br := subframe.br
	for i := 0; i < subframe.Order; i++ {
		// (bits-per-sample) bits: Unencoded warmup sample.
		x, err := br.Read(bps)
		if err != nil {
			return err
		}
		sample := signExtend(x, bps)
		subframe.Samples = append(subframe.Samples, sample)
	}

	return subframe.decodeResidual()
}

// decodeLPC decodes the linear prediction coded samples of the subframe, using
// polynomial coefficients stored in the stream.
//
// ref: https://www.xiph.org/flac/format.html#subframe_lpc
func (subframe *Subframe) decodeLPC() error {
	panic("not yet implemented.")
}

// decodeResidual decodes the encoded residuals (prediction method error
// signals) of the subframe.
//
// ref: https://www.xiph.org/flac/format.html#residual
func (subframe *Subframe) decodeResidual() error {
	// 2 bits: Residual coding method.
	br := subframe.br
	x, err := br.Read(2)
	if err != nil {
		return err
	}
	// The 2 bits are used to specify the residual coding method as follows:
	//    00: Rice coding with a 4-bit Rice parameter.
	//    01: Rice coding with a 5-bit Rice parameter.
	//    10: reserved.
	//    11: reserved.
	switch x {
	case 0x0:
		return subframe.decodeRicePart(4)
	case 0x1:
		return subframe.decodeRicePart(5)
	default:
		return fmt.Errorf("frame.Subframe.decodeResidual: reserved residual coding method bit pattern (%02b)", x)
	}
}

// decodeRicePart decodes a Rice partition of encoded residuals from the
// subframe, using a Rice parameter of the specified size in bits.
//
// ref: https://www.xiph.org/flac/format.html#partitioned_rice
// ref: https://www.xiph.org/flac/format.html#partitioned_rice2
func (subframe *Subframe) decodeRicePart(paramSize uint) error {
	// 4 bits: Partition order.
	br := subframe.br
	x, err := br.Read(4)
	if err != nil {
		return err
	}
	partOrder := x

	// Parse Rice partitions; in total 2^partOrder partitions.
	//
	// ref: https://www.xiph.org/flac/format.html#rice_partition
	// ref: https://www.xiph.org/flac/format.html#rice2_partition
	nparts := 1 << partOrder
	for i := 0; i < nparts; i++ {
		// (4 or 5) bits: Rice parameter.
		x, err = br.Read(paramSize)
		if err != nil {
			return err
		}
		if paramSize == 4 && x == 0xF || paramSize == 4 && x == 0x1F {
			// 1111 or 11111: Escape code, meaning the partition is in unencoded
			// binary form using n bits per sample; n follows as a 5-bit number.
			panic("not yet implemented; Rice parameter escape code.")
		}
		param := uint(x)

		// Determine the number of Rice encoded samples in the partition.
		var nsamples int
		if partOrder == 0 {
			nsamples = subframe.NSamples - subframe.Order
		} else if i != 0 {
			nsamples = subframe.NSamples / nparts
		} else {
			nsamples = subframe.NSamples/nparts - subframe.Order
		}

		// Decode the Rice encoded residuals of the partition.
		for j := 0; j < nsamples; j++ {
			err = subframe.decodeRice(param)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// decodeRice decodes a Rice encoded residual (error signal).
func (subframe *Subframe) decodeRice(k uint) error {
	panic("not yet implemented.")
}
