package frame

import (
	"errors"
	"fmt"
	dbg "fmt"
	"math"

	"github.com/eaburns/bit"
)

// A SubFrame contains the decoded audio data of a channel.
type SubFrame struct {
	// Header specifies the attributes of the subframe, like prediction method
	// and order, residual coding parameters, etc.
	Header *SubHeader
	// Samples contains the decoded audio samples of the channel.
	Samples []Sample
}

// A Sample is an audio sample. The size of each sample is between 4 and 32 bits.
type Sample uint32

// NewSubFrame parses and returns a new subframe, which consists of a subframe
// header and encoded audio samples.
//
// Subframe format (pseudo code):
//
//    type SUBFRAME struct {
//       header      SUBFRAME_HEADER
//       enc_samples SUBFRAME_CONSTANT || SUBFRAME_FIXED || SUBFRAME_LPC || SUBFRAME_VERBATIM
//    }
//
// ref: http://flac.sourceforge.net/format.html#subframe
func (h *Header) NewSubFrame(br *bit.Reader) (subframe *SubFrame, err error) {
	// Parse subframe header.
	subframe = new(SubFrame)
	subframe.Header, err = h.NewSubHeader(br)
	if err != nil {
		return nil, err
	}

	// Decode samples.
	sh := subframe.Header
	switch sh.PredMethod {
	case PredConstant:
		subframe.Samples, err = h.DecodeConstant(br)
	case PredFixed:
		subframe.Samples, err = h.DecodeFixed(br, int(sh.PredOrder))
	case PredLPC:
		subframe.Samples, err = h.DecodeLPC(br, int(sh.PredOrder))
	case PredVerbatim:
		subframe.Samples, err = h.DecodeVerbatim(br)
	default:
		return nil, fmt.Errorf("Header.NewSubFrame: unknown subframe prediction method: %d", sh.PredMethod)
	}
	if err != nil {
		return nil, err
	}

	return subframe, nil
}

// A SubHeader is a subframe header, which contains information about how the
// subframe audio samples are encoded.
type SubHeader struct {
	// PredMethod is the subframe prediction method.
	PredMethod PredMethod
	// WastedBitCount is the number of wasted bits per sample.
	WastedBitCount int8
	// PredOrder is the subframe predictor order, which is used accordingly:
	//    Fixed: Predictor order.
	//    LPC:   LPC order.
	PredOrder int8
}

// PredMethod specifies the subframe prediction method.
type PredMethod int8

// Subframe prediction methods.
const (
	PredConstant PredMethod = iota
	PredFixed
	PredLPC
	PredVerbatim
)

// NewSubHeader parses and returns a new subframe header.
//
// Subframe header format (pseudo code):
//    type SUBFRAME_HEADER struct {
//       _                uint1 // zero-padding, to prevent sync-fooling.
//       type             uint6
//       // 0: no wasted bits-per-sample in source subblock, k = 0.
//       // 1: k wasted bits-per-sample in source subblock, k-1 follows, unary
//       // coded; e.g. k=3 => 001 follows, k=7 => 0000001 follows.
//       wasted_bit_count uint1+k
//    }
//
// ref: http://flac.sourceforge.net/format.html#subframe_header
func (h *Header) NewSubHeader(br *bit.Reader) (sh *SubHeader, err error) {
	// field 0: padding (1 bit)
	// field 1: type    (6 bits)
	fields, err := br.ReadFields(1, 6)
	if err != nil {
		return nil, err
	}

	// Padding.
	if fields[0] != 0 {
		return nil, errors.New("Header.NewSubHeader: invalid padding; must be 0")
	}

	// Subframe prediction method.
	//    000000: SUBFRAME_CONSTANT
	//    000001: SUBFRAME_VERBATIM
	//    00001x: reserved
	//    0001xx: reserved
	//    001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
	//    01xxxx: reserved
	//    1xxxxx: SUBFRAME_LPC, xxxxx=order-1
	sh = new(SubHeader)
	n := fields[1]
	switch {
	case n == 0:
		// 000000: SUBFRAME_CONSTANT
		sh.PredMethod = PredConstant
	case n == 1:
		// 000001: SUBFRAME_VERBATIM
		sh.PredMethod = PredVerbatim
	case n < 8:
		// 00001x: reserved
		// 0001xx: reserved
		return nil, fmt.Errorf("Header.NewSubHeader: invalid subframe prediction method; reserved bit pattern: %06b", n)
	case n < 16:
		// 001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		const predOrderMask = 0x07
		sh.PredOrder = int8(n) & predOrderMask
		if sh.PredOrder > 4 {
			return nil, fmt.Errorf("Header.NewSubHeader: invalid subframe prediction method; reserved bit pattern: %06b", n)
		}
		sh.PredMethod = PredFixed
	case n < 32:
		// 01xxxx: reserved
		return nil, fmt.Errorf("Header.NewSubHeader: invalid subframe prediction method; reserved bit pattern: %06b", n)
	case n < 64:
		// 1xxxxx: SUBFRAME_LPC, xxxxx=order-1
		const predOrderMask = 0x1F
		sh.PredOrder = int8(n)&predOrderMask + 1
		sh.PredMethod = PredLPC
	default:
		// should be unreachable.
		return nil, fmt.Errorf("Header.NewSubHeader: unhandled subframe prediction method; bit pattern: %06b", n)
	}

	// Wasted bits-per-sample, 1+k bits.
	bits, err := br.Read(1)
	if err != nil {
		return nil, err
	}
	if bits != 0 {
		// k wasted bits-per-sample in source subblock, k-1 follows, unary coded;
		// e.g. k=3 => 001 follows, k=7 => 0000001 follows.
		/// ### [ todo ] ###
		///    - verify.
		/// ### [/ todo ] ###
		for {
			sh.WastedBitCount++
			bits, err = br.Read(1)
			if err != nil {
				return nil, err
			}
			if bits == 1 {
				break
			}
		}
	}

	return sh, nil
}

// DecodeConstant decodes and returns a slice of samples. The first sample is
// constant throughout the entire subframe.
//
// ref: http://flac.sourceforge.net/format.html#subframe_constant
func (h *Header) DecodeConstant(br *bit.Reader) (samples []Sample, err error) {
	// Read constant sample.
	bits, err := br.Read(uint(h.SampleSize))
	if err != nil {
		return nil, err
	}
	sample := Sample(bits)
	dbg.Println("Constant sample:", sample)

	// Duplicate the constant sample, sample count number of times.
	for i := uint16(0); i < h.SampleCount; i++ {
		samples = append(samples, sample)
	}

	return samples, nil
}

// DecodeFixed decodes and returns a slice of samples.
/// ### [ todo ] ###
///    - add more details.
/// ### [/ todo ] ###
//
// ref: http://flac.sourceforge.net/format.html#subframe_fixed
func (h *Header) DecodeFixed(br *bit.Reader, predOrder int) (samples []Sample, err error) {
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * predictor order
	for i := 0; i < predOrder; i++ {
		bits, err := br.Read(uint(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits)
		dbg.Println("Fixed warm-up sample:", sample)
		samples = append(samples, sample)
	}

	residuals, err := h.DecodeResidual(br, predOrder)
	if err != nil {
		return nil, err
	}
	_ = residuals
	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, errors.New("not yet implemented; Fixed encoding")
}

// DecodeLPC decodes and returns a slice of samples.
/// ### [ todo ] ###
///    - add more details.
/// ### [/ todo ] ###
//
// ref: http://flac.sourceforge.net/format.html#subframe_lpc
func (h *Header) DecodeLPC(br *bit.Reader, lpcOrder int) (samples []Sample, err error) {
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * lpc order
	for i := 0; i < lpcOrder; i++ {
		bits, err := br.Read(uint(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits)
		dbg.Println("LPC warm-up sample:", sample)
		samples = append(samples, sample)
	}

	// (Quantized linear predictor coefficients' precision in bits) - 1.
	n, err := br.Read(4)
	if err != nil {
		return nil, err
	}
	if n == 0x0F {
		// 1111: invalid.
		return nil, errors.New("Header.DecodeLPC: invalid quantized lpc precision; reserved bit pattern: 1111")
	}
	qlpcPrec := int(n) + 1

	// Quantized linear predictor coefficient shift needed in bits.
	qlpcShift, err := br.Read(5)
	if err != nil {
		return nil, err
	}
	/// ### [ todo ] ###
	///    - NOTE: this number is signed two's-complement.
	///    - special case for negative numbers required?
	///    - the same goes for qlpcPrec.
	/// ### [/ todo ] ###
	_ = qlpcShift

	// Unencoded predictor coefficients.
	for i := 0; i < lpcOrder; i++ {
		pc, err := br.Read(uint(qlpcPrec))
		if err != nil {
			return nil, err
		}
		dbg.Println("pc:", pc)
		_ = pc
	}

	residuals, err := h.DecodeResidual(br, lpcOrder)
	if err != nil {
		return nil, err
	}
	_ = residuals
	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, errors.New("not yet implemented; LPC encoding")
}

// DecodeVerbatim decodes and returns a slice of samples. The samples are stored
// unencoded.
//
// ref: http://flac.sourceforge.net/format.html#subframe_verbatim
func (h *Header) DecodeVerbatim(br *bit.Reader) (samples []Sample, err error) {
	// Read unencoded samples.
	for i := uint16(0); i < h.SampleCount; i++ {
		bits, err := br.Read(uint(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits)
		dbg.Println("Verbatim sample:", sample)
		samples = append(samples, sample)
	}

	return samples, nil
}

// DecodeResidual decodes and returns a slice of residuals.
/// ### [ todo ] ###
///    - add more details.
/// ### [/ todo ] ###
//
// ref: http://flac.sourceforge.net/format.html#residual
func (h *Header) DecodeResidual(br *bit.Reader, predOrder int) (residuals []int, err error) {
	// Residual coding method.
	method, err := br.Read(2)
	if err != nil {
		return nil, err
	}
	switch method {
	case 0:
		// 00: partitioned Rice coding with 4-bit Rice parameter;
		//     RESIDUAL_CODING_METHOD_PARTITIONED_RICE follows
		return h.DecodeRice(br, predOrder)
	case 1:
		// 01: partitioned Rice coding with 5-bit Rice parameter;
		//     RESIDUAL_CODING_METHOD_PARTITIONED_RICE2 follows
		return h.DecodeRice2(br, predOrder)
	}
	// 1x: reserved
	return nil, fmt.Errorf("Header.DecodeResidual: invalid residual coding method; reserved bit pattern: %02b", method)
}

// DecodeRice decodes and returns a slice of residuals. The residual coding
// method used is partitioned Rice coding with a 4-bit Rice parameter.
//
// ref: http://flac.sourceforge.net/format.html#partitioned_rice
func (h *Header) DecodeRice(br *bit.Reader, predOrder int) (residuals []int, err error) {
	// Partition order.
	partOrder, err := br.Read(4)
	if err != nil {
		return nil, err
	}

	// Rice partitions.
	partCount := int(math.Pow(2, float64(partOrder)))
	partSampleCount := int(h.SampleCount) / partCount
	for partNum := 0; partNum < partCount; partNum++ {
		// Encoding parameter.
		n, err := br.Read(4)
		if err != nil {
			return nil, err
		}
		if n == 0x0F {
			// 1111: Escape code, meaning the partition is in unencoded binary form
			// using n bits per sample; n follows as a 5-bit number.
			/// ### [ todo ] ###
			///    - not yet implemented.
			/// ### [/ todo ] ###
			return nil, errors.New("not yet implemented; rice encoding parameter escape code")
		}
		riceParam := n
		_ = riceParam

		dbg.Println("riceParam:", riceParam)

		// Encoded residual.
		sampleCount := partSampleCount
		if partNum == 0 {
			sampleCount -= predOrder
		}

		dbg.Println("sampleCount:", sampleCount)
	}

	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, errors.New("not yet implemented; rice coding method 0")
}

// DecodeRice2 decodes and returns a slice of residuals. The residual coding
// method used is partitioned Rice coding with a 5-bit Rice parameter.
//
// ref: http://flac.sourceforge.net/format.html#partitioned_rice2
func (h *Header) DecodeRice2(br *bit.Reader, predOrder int) (residuals []int, err error) {
	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, errors.New("not yet implemented; rice coding method 1")
}

/**
type SubFrameLpc struct {
	Precision             uint8
	ShiftNeeded           uint8
	PredictorCoefficients []byte
}

type Rice struct {
	Partitions     []RicePartition
}

type Rice2 struct {
	Partitions     []Rice2Partition
}

type RicePartition struct {
	EncodingParameter uint16
}

type Rice2Partition struct{}
*/
