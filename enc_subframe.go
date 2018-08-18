package flac

import (
	"github.com/icza/bitio"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/pkg/errutil"
)

// encodeSubframe encodes the given samples to a subframe of the output stream.
func (enc *Encoder) encodeSubframe(bw bitio.Writer, hdr *frame.Header, samples []int32) error {
	// Encode subframe header.
	subHdr := &frame.SubHeader{
		// Specifies the prediction method used to encode the audio sample of the
		// subframe.
		Pred: frame.PredVerbatim,
		// Prediction order used by fixed and FIR linear prediction decoding.
		Order: 0,
		// Wasted bits-per-sample.
		Wasted: 0,
	}
	if err := enc.encodeSubframeHeader(bw, subHdr); err != nil {
		return errutil.Err(err)
	}

	switch subHdr.Pred {
	//case frame.PredConstant:
	//	if err := enc.encodeConstantSamples(bw, samples); err != nil {
	//		return errutil.Err(err)
	//	}
	case frame.PredVerbatim:
		if err := enc.encodeVerbatimSamples(bw, hdr, samples); err != nil {
			return errutil.Err(err)
		}
	//case frame.PredFixed:
	//	if err := enc.encodeFixedSamples(bw, samples, subHdr.Order); err != nil {
	//		return errutil.Err(err)
	//	}
	//case frame.PredFIR:
	//	if err := enc.encodeFIRSamples(bw, samples, subHdr.Order); err != nil {
	//		return errutil.Err(err)
	//	}
	default:
		return errutil.Newf("support for prediction method %v not yet implemented", subHdr.Pred)
	}

	return nil
}

// encodeSubframeHeader encodes the given subframe header to the output stream.
func (enc *Encoder) encodeSubframeHeader(bw bitio.Writer, subHdr *frame.SubHeader) error {
	// Zero bit padding, to prevent sync-fooling string of 1s.
	if err := bw.WriteBits(0x0, 1); err != nil {
		return errutil.Err(err)
	}

	// Subframe type:
	//     000000 : SUBFRAME_CONSTANT
	//     000001 : SUBFRAME_VERBATIM
	//     00001x : reserved
	//     0001xx : reserved
	//     001xxx : if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
	//     01xxxx : reserved
	//     1xxxxx : SUBFRAME_LPC, xxxxx=order-1
	var bits uint64
	switch subHdr.Pred {
	case frame.PredConstant:
		// 000000 : SUBFRAME_CONSTANT
		bits = 0x00
	case frame.PredVerbatim:
		// 000001 : SUBFRAME_VERBATIM
		bits = 0x01
	case frame.PredFixed:
		// 001xxx : if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		bits = 0x08 | uint64(subHdr.Order)
	case frame.PredFIR:
		// 1xxxxx : SUBFRAME_LPC, xxxxx=order-1
		order := uint64(0)
		bits = 0x20 | uint64(order-1)
	}
	if err := bw.WriteBits(bits, 6); err != nil {
		return errutil.Err(err)
	}

	// <1+k> 'Wasted bits-per-sample' flag:
	//
	//     0 : no wasted bits-per-sample in source subblock, k=0
	//     1 : k wasted bits-per-sample in source subblock, k-1 follows, unary coded; e.g. k=3 => 001 follows, k=7 => 0000001 follows.
	hasWastedBits := subHdr.Wasted > 0
	if err := bw.WriteBool(hasWastedBits); err != nil {
		return errutil.Err(err)
	}
	if hasWastedBits {
		if err := writeUnary(bw, uint64(subHdr.Wasted)); err != nil {
			return errutil.Err(err)
		}
	}

	return nil
}

// encodeVerbatimSamples stores the given samples verbatim, uncompressed.
func (enc *Encoder) encodeVerbatimSamples(bw bitio.Writer, hdr *frame.Header, samples []int32) error {
	// Unencoded subblock; n = frame's bits-per-sample, i = frame's blocksize.
	if int(hdr.BlockSize) != len(samples) {
		return errutil.Newf("invalid number of samples in block; expected %d, got %d", hdr.BlockSize, len(samples))
	}
	// TODO: remove debug printout.
	//fmt.Println("verbatim")
	for i := 0; i < int(hdr.BlockSize); i++ {
		//fmt.Println("   sample:", samples[i])
		if err := bw.WriteBits(uint64(samples[i]), byte(hdr.BitsPerSample)); err != nil {
			return errutil.Err(err)
		}
	}
	return nil
}

// TODO: move writeUnary to internal/bits when bit writer is moved there.

// writeUnary encodes x as an unary coded integer, whose value is represented by
// the number of leading zeros before a one.
//
// Examples of unary coded binary on the left and decoded decimal on the right:
//
//    0 => 1
//    1 => 01
//    2 => 001
//    3 => 0001
//    4 => 00001
//    5 => 000001
//    6 => 0000001
func writeUnary(bw bitio.Writer, x uint64) error {
	bits := uint64(1)
	n := byte(1)
	for ; x > 0; x-- {
		n++
	}
	if err := bw.WriteBits(bits, n); err != nil {
		return errutil.Err(err)
	}
	return nil
}
