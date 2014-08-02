// TODO(u): Get rid of all panics :)

package frame

import (
	"errors"
	"fmt"
	"math"

	"github.com/eaburns/bit"
	"github.com/mewkiz/pkg/bitutil"
	"github.com/mewkiz/pkg/dbg"
)

func init() {
	dbg.Debug = false
}

// A SubFrame contains the decoded audio data of a channel.
type SubFrame struct {
	// Header specifies the attributes of the subframe, like prediction method
	// and order, residual coding parameters, etc.
	Header *SubHeader
	// Samples contains the decoded audio samples of the channel.
	Samples []Sample
}

// A Sample is an audio sample. The size of each sample is between 4 and 32
// bits.
type Sample int32

// NewSubFrame parses and returns a new subframe, which consists of a subframe
// header and encoded audio samples.
//
// Subframe format (pseudo code):
//
//    type SUBFRAME struct {
//       header      SUBFRAME_HEADER
//       enc_samples SUBFRAME_CONSTANT || SUBFRAME_FIXED || SUBFRAME_LPC ||
//                   SUBFRAME_VERBATIM
//    }
//
// ref: http://flac.sourceforge.net/format.html#subframe
func (h *Header) NewSubFrame(br *bit.Reader, bps uint) (subframe *SubFrame, err error) {
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
		subframe.Samples, err = h.DecodeConstant(br, bps)
	case PredFixed:
		subframe.Samples, err = h.DecodeFixed(br, int(sh.PredOrder), bps)
	case PredLPC:
		subframe.Samples, err = h.DecodeLPC(br, int(sh.PredOrder), bps)
	case PredVerbatim:
		subframe.Samples, err = h.DecodeVerbatim(br, bps)
	default:
		return nil, fmt.Errorf("frame.Header.NewSubFrame: unknown subframe prediction method: %d", sh.PredMethod)
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
	// field 0: padding (1 bit)
	if fields[0] != 0 {
		return nil, errors.New("frame.Header.NewSubHeader: invalid padding; must be 0")
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
	// field 1: type (6 bits)
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
		return nil, fmt.Errorf("frame.Header.NewSubHeader: invalid subframe prediction method; reserved bit pattern: %06b", n)
	case n < 16:
		// 001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		const predOrderMask = 0x07
		sh.PredOrder = int8(n) & predOrderMask
		if sh.PredOrder > 4 {
			return nil, fmt.Errorf("frame.Header.NewSubHeader: invalid subframe prediction method; reserved bit pattern: %06b", n)
		}
		sh.PredMethod = PredFixed
	case n < 32:
		// 01xxxx: reserved
		return nil, fmt.Errorf("frame.Header.NewSubHeader: invalid subframe prediction method; reserved bit pattern: %06b", n)
	case n < 64:
		// 1xxxxx: SUBFRAME_LPC, xxxxx=order-1
		const predOrderMask = 0x1F
		sh.PredOrder = int8(n)&predOrderMask + 1
		sh.PredMethod = PredLPC
	default:
		// should be unreachable.
		panic(fmt.Errorf("frame.Header.NewSubHeader: unhandled subframe prediction method; bit pattern: %06b", n))
	}

	// Wasted bits-per-sample, 1+k bits.
	hasWastedBits, err := br.Read(1)
	if err != nil {
		return nil, err
	}
	if hasWastedBits != 0 {
		// k wasted bits-per-sample in source subblock, k-1 follows, unary coded;
		// e.g. k=3 => 001 follows, k=7 => 0000001 follows.
		// TODO(u): Verify that the unary decoding is correct.
		n, err := bitutil.DecodeUnary(br)
		if err != nil {
			return nil, err
		}
		sh.WastedBitCount = int8(n) + 1
		dbg.Println("wasted bits-per-sample:", sh.WastedBitCount)
		panic("not yet implemented; wasted bits")
	}

	return sh, nil
}

// DecodeConstant decodes and returns a slice of samples. The first sample is
// constant throughout the entire subframe.
//
// ref: http://flac.sourceforge.net/format.html#subframe_constant
func (h *Header) DecodeConstant(br *bit.Reader, bps uint) (samples []Sample, err error) {
	// Read constant sample.
	x, err := br.Read(bps)
	if err != nil {
		return nil, err
	}
	sample := Sample(signExtend(x, bps))
	dbg.Println("Constant sample:", sample)

	// Duplicate the constant sample, sample count number of times.
	samples = make([]Sample, h.SampleCount)
	for i := range samples {
		samples[i] = sample
	}

	return samples, nil
}

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

// fixedCoeffs maps from prediction order to the LPC coefficients used in fixed
// encoding.
//
//    x_0[n] = 0
//    x_1[n] = x[n-1]
//    x_2[n] = 2*x[n-1] - x[n-2]
//    x_3[n] = 3*x[n-1] - 3*x[n-2] + x[n-3]
//
// ref: Section 2.2 of http://www.hpl.hp.com/techreports/1999/HPL-1999-144.pdf
var fixedCoeffs = [...][]int32{
	1: {1},
	2: {2, -1},
	3: {3, -3, 1},
	// TODO(u): Verify the definition of the coefficients for prediction order 4.
	4: {4, -6, 4, -1},
}

// DecodeFixed decodes and returns a slice of samples.
//
// TODO(u): Add more detailed documentation.
//
// ref: http://flac.sourceforge.net/format.html#subframe_fixed
func (h *Header) DecodeFixed(br *bit.Reader, predOrder int, bps uint) (samples []Sample, err error) {
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * predictor order
	warm := make([]Sample, predOrder)
	dbg.Println("Fixed prediction order:", predOrder)
	for i := range warm {
		x, err := br.Read(bps)
		if err != nil {
			return nil, err
		}
		sample := Sample(signExtend(x, bps))
		dbg.Println("Fixed warm-up sample:", sample)
		warm[i] = sample
	}

	residuals, err := h.DecodeResidual(br, predOrder)
	if err != nil {
		return nil, err
	}
	dbg.Println("residuals:", residuals)
	dbg.Println("coeff:", fixedCoeffs[predOrder])
	return lpcDecode(fixedCoeffs[predOrder], warm, residuals, 0), nil
}

// lpcDecode decodes a set of samples using LPC (Linear Predictive Coding) with
// FIR (Finite Impulse Response) predictors.
func lpcDecode(coeffs []int32, warm []Sample, residuals []int32, shift uint) (samples []Sample) {
	samples = make([]Sample, len(warm)+len(residuals))
	copy(samples, warm)
	// Note: The following code is borrowed from https://github.com/eaburns/flac/blob/master/decode.go#L751
	for i := len(warm); i < len(samples); i++ {
		var sum int32
		for j, coeff := range coeffs {
			sum += coeff * int32(samples[i-j-1])
			samples[i] = Sample(residuals[i-len(warm)] + sum>>shift)
		}
	}
	dbg.Println("samples:", samples)
	return samples
}

// DecodeLPC decodes and returns a slice of samples.
//
// TODO(u): Add more detailed documentation.
//
// ref: http://flac.sourceforge.net/format.html#subframe_lpc
func (h *Header) DecodeLPC(br *bit.Reader, lpcOrder int, bps uint) (samples []Sample, err error) {
	dbg.Println("lpcOrder:", lpcOrder)
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * lpc order
	dbg.Println("bps:", bps)
	warm := make([]Sample, lpcOrder)
	for i := range warm {
		x, err := br.Read(bps)
		if err != nil {
			return nil, err
		}
		sample := Sample(signExtend(x, bps))
		dbg.Println("LPC warm-up sample:", sample)
		warm[i] = sample
	}
	dbg.Println("warm:", warm)

	// (Quantized linear predictor coefficients' precision in bits) - 1.
	x, err := br.Read(4)
	if err != nil {
		return nil, err
	}
	if x == 0xF {
		// 1111: invalid.
		return nil, errors.New("frame.Header.DecodeLPC: invalid quantized lpc precision; reserved bit pattern: 1111")
	}
	qlpcPrec := int(x) + 1
	dbg.Println("qlpcPrec:", qlpcPrec)

	// Quantized linear predictor coefficient shift needed in bits.
	x, err = br.Read(5)
	if err != nil {
		return nil, err
	}
	qlpcShift := signExtend(x, 5)
	if qlpcShift < 0 {
		panic("qlpcShift is negative")
	}
	dbg.Println("qlpcShift:", qlpcShift)

	// Unencoded predictor coefficients.
	coeffs := make([]int32, lpcOrder)
	for i := range coeffs {
		x, err := br.Read(uint(qlpcPrec))
		if err != nil {
			return nil, err
		}
		coeff := int32(signExtend(x, uint(qlpcPrec)))
		dbg.Println("coeff:", coeff)
		coeffs[i] = coeff
	}
	dbg.Println("coeffs:", coeffs)

	residuals, err := h.DecodeResidual(br, lpcOrder)
	if err != nil {
		return nil, err
	}
	_ = residuals
	dbg.Println("residuals:", residuals)

	return lpcDecode(coeffs, warm, residuals, uint(qlpcShift)), nil
}

// DecodeVerbatim decodes and returns a slice of samples. The samples are stored
// unencoded.
//
// ref: http://flac.sourceforge.net/format.html#subframe_verbatim
func (h *Header) DecodeVerbatim(br *bit.Reader, bps uint) (samples []Sample, err error) {
	// Read unencoded samples.
	samples = make([]Sample, h.SampleCount)
	for i := range samples {
		x, err := br.Read(bps)
		if err != nil {
			return nil, err
		}
		sample := Sample(signExtend(x, bps))
		dbg.Println("Verbatim sample:", sample)
		samples[i] = sample
	}

	return samples, nil
}

// DecodeResidual decodes and returns a slice of residuals.
//
// TODO(u): Add more detailed documentation.
//
// ref: http://flac.sourceforge.net/format.html#residual
func (h *Header) DecodeResidual(br *bit.Reader, predOrder int) (residuals []int32, err error) {
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
	return nil, fmt.Errorf("frame.Header.DecodeResidual: invalid residual coding method; reserved bit pattern: %02b", method)
}

// DecodeRice decodes and returns a slice of residuals. The residual coding
// method used is partitioned Rice coding with a 4-bit Rice parameter.
//
// ref: http://flac.sourceforge.net/format.html#partitioned_rice
func (h *Header) DecodeRice(br *bit.Reader, predOrder int) (residuals []int32, err error) {
	// Partition order.
	partOrder, err := br.Read(4)
	if err != nil {
		return nil, err
	}

	// Rice partitions.
	partCount := int(math.Pow(2, float64(partOrder)))
	for partNum := 0; partNum < partCount; partNum++ {
		partSampleCount := int(h.SampleCount) / partCount

		// Encoding parameter.
		riceParam, err := br.Read(4)
		if err != nil {
			return nil, err
		}
		if riceParam == 0xF {
			// 1111: Escape code, meaning the partition is in unencoded binary form
			// using n bits per sample; n follows as a 5-bit number.
			n, err := br.Read(5)
			if err != nil {
				return nil, err
			}
			panic("not yet implemented: unencoded samples.")
			// TODO(u): Check; the loop below is a best effort attempt at
			// understanding the spec, it may very well be inaccurate.
			for i := 0; i < partSampleCount; i++ {
				sample, err := br.Read(uint(n))
				if err != nil {
					return nil, err
				}
				// TODO(u): Figure out if we should change to API to return the
				// unencoded samples.
				dbg.Println("sample:", sample)
			}
		}
		dbg.Println("riceParam:", riceParam)

		// Encoded residual.
		if partOrder == 0 {
			partSampleCount = int(h.SampleCount) - predOrder
		} else if partNum != 0 {
			partSampleCount = int(h.SampleCount) / int(math.Pow(2, float64(partOrder)))
		} else {
			partSampleCount = int(h.SampleCount)/int(math.Pow(2, float64(partOrder))) - predOrder
		}
		dbg.Println("partSampleCount:", partSampleCount)

		// Decode rice partition residuals.
		partResiduals, err := riceDecode(br, uint(riceParam), partSampleCount)
		if err != nil {
			return nil, err
		}
		residuals = append(residuals, partResiduals...)
	}

	return residuals, nil
}

// riceDecode decodes the residual signals of a partition encoded using Rice
// coding.
func riceDecode(br *bit.Reader, k uint, n int) (residuals []int32, err error) {
	residuals = make([]int32, n)
	for i := 0; i < n; i++ {
		// Read unary encoded most significant bits.
		high, err := bitutil.DecodeUnary(br)
		if err != nil {
			return nil, err
		}

		// Read binary encoded least significant bits.
		low, err := br.Read(k)
		if err != nil {
			return nil, err
		}
		residual := int32(high<<k | low)

		// ZigZag decode.
		residual = int32(bitutil.DecodeZigZag(int(residual)))

		residuals[i] = residual
	}
	return residuals, nil
}

// DecodeRice2 decodes and returns a slice of residuals. The residual coding
// method used is partitioned Rice coding with a 5-bit Rice parameter.
//
// ref: http://flac.sourceforge.net/format.html#partitioned_rice2
func (h *Header) DecodeRice2(br *bit.Reader, predOrder int) (residuals []int32, err error) {
	// TODO(u): not yet implemented.
	return nil, errors.New("frame.Header.DecodeRice: not yet implemented; rice coding method 1")
}
