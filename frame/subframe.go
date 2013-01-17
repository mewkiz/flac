package frame

import dbg "fmt"
import "errors"
import "fmt"

import "github.com/mewkiz/pkg/bit"

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
func (h *Header) NewSubFrame(br bit.Reader) (subframe *SubFrame, err error) {
	// Parse subframe header.
	subframe = new(SubFrame)
	subframe.Header, err = h.NewSubHeader(br)
	if err != nil {
		return nil, err
	}

	// Decode samples.
	sh := subframe.Header
	switch sh.EncType {
	case EncConstant:
		subframe.Samples, err = h.DecodeConstant(br)
	case EncFixed:
		subframe.Samples, err = h.DecodeFixed(br, int(sh.Order))
	case EncLPC:
		subframe.Samples, err = h.DecodeLPC(br, int(sh.Order))
	case EncVerbatim:
		subframe.Samples, err = h.DecodeVerbatim(br)
	default:
		return nil, fmt.Errorf("Header.NewSubFrame: unknown subframe encoding type: %d.", sh.EncType)
	}
	if err != nil {
		return nil, err
	}

	return subframe, nil
}

// A SubHeader is a subframe header, which contains information about how the
// subframe audio samples are encoded.
type SubHeader struct {
	// EncType is the subframe encoding type.
	EncType EncType
	// WastedBitCount is the number of wasted bits per sample.
	WastedBitCount int8
	// Order is used when decoding fixed and LPC-encoded subframes accordingly:
	//    Fixed: Predictor order.
	//    LPC:   LPC order.
	Order int8
}

// EncType specifies the subframe encoding type.
type EncType int8

// Subframe encoding types.
const (
	EncConstant EncType = iota
	EncFixed
	EncLPC
	EncVerbatim
)

// NewSubHeader parses and returns a new subframe header.
//
// Subframe header format (pseudo code):
//    type SUBFRAME_HEADER struct {
//       _                uint1 // zero-padding, to prevent sync-fooling string of 1s.
//       type             uint6
//       // 0: no wasted bits-per-sample in source subblock, k = 0.
//       // 1: k wasted bits-per-sample in source subblock, k-1 follows, unary
//       // coded; e.g. k=3 => 001 follows, k=7 => 0000001 follows.
//       wasted_bit_count uint1+k
//    }
//
// ref: http://flac.sourceforge.net/format.html#subframe_header
func (h *Header) NewSubHeader(br bit.Reader) (sh *SubHeader, err error) {
	// Padding, 1 bit.
	pad, err := br.Read(1)
	if err != nil {
		return nil, err
	}
	if pad.Uint64() != 0 {
		return nil, errors.New("Header.NewSubHeader: invalid padding; must be 0.")
	}

	// Subframe type, 6 bits.
	typ, err := br.Read(6)
	if err != nil {
		return nil, err
	}
	// Subframe type.
	//    000000: SUBFRAME_CONSTANT
	//    000001: SUBFRAME_VERBATIM
	//    00001x: reserved
	//    0001xx: reserved
	//    001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
	//    01xxxx: reserved
	//    1xxxxx: SUBFRAME_LPC, xxxxx=order-1
	sh = new(SubHeader)
	n := typ.Uint64()
	switch {
	case n == 0:
		// 000000: SUBFRAME_CONSTANT
		sh.EncType = EncConstant
	case n == 1:
		// 000001: SUBFRAME_VERBATIM
		sh.EncType = EncVerbatim
	case n < 8:
		// 00001x: reserved
		// 0001xx: reserved
		return nil, fmt.Errorf("Header.NewSubHeader: invalid subframe type; reserved bit pattern: %06b.", n)
	case n < 16:
		// 001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		const orderMask = 0x07
		sh.Order = int8(n) & orderMask
		if sh.Order > 4 {
			return nil, fmt.Errorf("Header.NewSubHeader: invalid subframe type; reserved bit pattern: %06b.", n)
		}
		sh.EncType = EncFixed
	case n < 32:
		// 01xxxx: reserved
		return nil, fmt.Errorf("Header.NewSubHeader: invalid subframe type; reserved bit pattern: %06b.", n)
	case n < 64:
		// 1xxxxx: SUBFRAME_LPC, xxxxx=order-1
		const orderMask = 0x1F
		sh.Order = int8(n)&orderMask + 1
		sh.EncType = EncLPC
	default:
		// should be unreachable.
		return nil, fmt.Errorf("Header.NewSubHeader: unhandled subframe type; bit pattern: %06b.", n)
	}

	// Wasted bits-per-sample, 1+k bits.
	bits, err := br.Read(1)
	if err != nil {
		return nil, err
	}
	if bits.Uint64() != 0 {
		// k wasted bits-per-sample in source subblock, k-1 follows, unary coded;
		// e.g. k=3 => 001 follows, k=7 => 0000001 follows.
		/// ### [ todo ] ###
		///    - verify.
		/// ### [/ todo ] ###
		for {
			sh.WastedBitCount++
			bits, err := br.Read(1)
			if err != nil {
				return nil, err
			}
			if bits.Uint64() == 1 {
				break
			}
		}
	}

	return sh, nil
}

// DecodeConstant decodes and returns a sample that is constant throughout the
// entire subframe.
//
// ref: http://flac.sourceforge.net/format.html#subframe_constant
func (h *Header) DecodeConstant(br bit.Reader) (samples []Sample, err error) {
	// Decode constant sample.
	bits, err := br.Read(int(h.SampleSize))
	if err != nil {
		return nil, err
	}
	sample := Sample(bits.Uint64())
	dbg.Println("constant sample:", sample)

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
func (h *Header) DecodeFixed(br bit.Reader, order int) (samples []Sample, err error) {
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * predictor order
	for i := 0; i < order; i++ {
		bits, err := br.Read(int(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits.Uint64())
		dbg.Println("Fixed warm-up sample:", sample)
		samples = append(samples, sample)
	}

	residuals, err := h.DecodeResidual(br, order)
	if err != nil {
		return nil, err
	}
	_ = residuals
	return nil, fmt.Errorf("not yet implemented; Fixed encoding.")
}

// DecodeLPC decodes and returns a slice of samples.
/// ### [ todo ] ###
///    - add more details.
/// ### [/ todo ] ###
//
// ref: http://flac.sourceforge.net/format.html#subframe_lpc
func (h *Header) DecodeLPC(br bit.Reader, order int) (samples []Sample, err error) {
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * lpc order
	for i := 0; i < order; i++ {
		bits, err := br.Read(int(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits.Uint64())
		dbg.Println("LPC warm-up sample:", sample)
		samples = append(samples, sample)
	}

	residuals, err := h.DecodeResidual(br, order)
	if err != nil {
		return nil, err
	}
	_ = residuals
	return nil, fmt.Errorf("not yet implemented; LPC encoding.")
}

// DecodeVerbatim decodes and returns a slice of samples, which were unencoded.
//
// ref: http://flac.sourceforge.net/format.html#subframe_verbatim
func (h *Header) DecodeVerbatim(br bit.Reader) (samples []Sample, err error) {
	// bits-per-sample
	for i := uint16(0); i < h.SampleCount; i++ {
		bits, err := br.Read(int(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits.Uint64())
		dbg.Println("Verbatim sample:", sample)
		samples = append(samples, sample)
	}
	return samples, nil
}

// ref: http://flac.sourceforge.net/format.html#residual
func (h *Header) DecodeResidual(br bit.Reader, order int) (residuals []int, err error) {
	bits, err := br.Read(2)
	if err != nil {
		return nil, err
	}
	rice := bits.Uint64()
	switch rice {
	case 0:
		// 00: partitioned Rice coding with 4-bit Rice parameter;
		//     RESIDUAL_CODING_METHOD_PARTITIONED_RICE follows
		return h.DecodeRice(br, order)
		if err != nil {
			return nil, err
		}
	case 1:
		// 01: partitioned Rice coding with 5-bit Rice parameter;
		//     RESIDUAL_CODING_METHOD_PARTITIONED_RICE2 follows
		return h.DecodeRice2(br, order)
	}
	// 1x: reserved
	return nil, fmt.Errorf("frame.DecodeFixed: invalid rice coding method; reserved bit pattern: %02b.", rice)
}

// ref: http://flac.sourceforge.net/format.html#partitioned_rice
func (h *Header) DecodeRice(br bit.Reader, order int) (residuals []int, err error) {
	// Encoding param.
	bits, err := br.Read(4)
	param := bits.Uint64()
	switch {
	case param == 0xF:
		// 1111: Escape code, meaning the partition is in unencoded binary form
		//       using n bits per sample; n follows as a 5-bit number.
		/// ### [ todo ] ###
		///   - not yet implemented.
		/// ### [/ todo ] ###
		return nil, fmt.Errorf("not yet implemented; rice coding method: 0, param escape code.")
	}
	// 0000-1110: Rice parameter.
	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, fmt.Errorf("not yet implemented; rice coding method: 0.")
}

// ref: http://flac.sourceforge.net/format.html#partitioned_rice2
func (h *Header) DecodeRice2(br bit.Reader, order int) (residuals []int, err error) {
	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, fmt.Errorf("not yet implemented; rice coding method: 1.")
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
