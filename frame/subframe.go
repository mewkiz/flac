package frame

import dbg "fmt"
import "errors"
import "fmt"

import "github.com/mewkiz/pkg/bit"

type SubFrame struct {
	Header  *SubHeader
	Samples []Sample
}

// Sample is an audio sample. The size of each sample is between 4 and 32 bits.
type Sample uint32

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
		sample, err := h.DecodeConstant(br)
		if err != nil {
			return nil, err
		}
		for i := uint16(0); i < h.SampleCount; i++ {
			subframe.Samples = append(subframe.Samples, sample)
		}
	case EncFixed:
		subframe.Samples, err = h.DecodeFixed(br, int(sh.Order))
		if err != nil {
			return nil, err
		}
	default:
		/// ### [ todo ] ###
		///   - not yet implemented.
		/// ### [/ todo ] ###
		return nil, fmt.Errorf("not yet implemented; subframe encoding type: %d.", sh.EncType)
	}

	return subframe, nil
}

// A SubHeader is a subframe header, which contains information about how the
// subframe samples are encoded.
type SubHeader struct {
	// Subframe encoding type:
	EncType    EncType
	WastedBits int8
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
//    // ref: http://flac.sourceforge.net/format.html#subframe_header
//
//    type SUBFRAME_HEADER struct {
//       _          uint1
//       type       uint6
//       // 0: no wasted bits-per-sample in source subblock, k = 0.
//       // 1: k wasted bits-per-sample in source subblock, k-1 follows, unary
//       // coded; e.g. k=3 => 001 follows, k=7 => 0000001 follows.
//       wastedBits uint1+k
//    }
func (h *Header) NewSubHeader(br bit.Reader) (sh *SubHeader, err error) {
	// Padding, 1 bit.
	pad, err := br.Read(1)
	if err != nil {
		return nil, err
	}
	if pad.Uint64() != 0 {
		return nil, errors.New("frame.NewSubHeader: invalid padding; must be 0.")
	}

	// Subframe type, 6 bits.
	sh = new(SubHeader)
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
		return nil, fmt.Errorf("frame.NewSubHeader: invalid subframe type; reserved bit pattern: %06b.", n)
	case n < 16:
		// 001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		const orderMask = 0x07
		sh.Order = int8(n) & orderMask
		if sh.Order > 4 {
			return nil, fmt.Errorf("frame.NewSubHeader: invalid subframe type; reserved bit pattern: %06b.", n)
		}
		sh.EncType = EncFixed
	case n < 32:
		// 01xxxx: reserved
		return nil, fmt.Errorf("frame.NewSubHeader: invalid subframe type; reserved bit pattern: %06b.", n)
	case n < 64:
		// 1xxxxx: SUBFRAME_LPC, xxxxx=order-1
		const orderMask = 0x1F
		sh.Order = int8(n)&orderMask + 1
		sh.EncType = EncLPC
	default:
		// should be unreachable.
		return nil, fmt.Errorf("frame.NewSubHeader: unhandled subframe type bit pattern: %06b.", n)
	}

	// Wasted bits-per-sample, 1+k bits.
	wastedBits, err := br.Read(1)
	if err != nil {
		return nil, err
	}
	if wastedBits.Uint64() != 0 {
		/// ### [ todo ] ###
		///    - handle wasted bits-per-sample.
		/// ### [/ todo ] ###
		return nil, errors.New("not yet implemented; wasted bits-per-sample.")
	}

	return sh, nil
}

// DecodeConstant decodes and returns a sample that is constant throughout the
// entire subframe.
func (h *Header) DecodeConstant(br bit.Reader) (sample Sample, err error) {
	bits, err := br.Read(int(h.SampleSize))
	if err != nil {
		return 0, err
	}
	sample = Sample(bits.Uint64())
	dbg.Println("constant sample:", sample)
	return sample, nil
}

func (h *Header) DecodeFixed(br bit.Reader, order int) (samples []Sample, err error) {
	// Unencoded warm-up samples:
	//    n bits = frame's bits-per-sample * predictor order
	for i := 0; i < order; i++ {
		bits, err := br.Read(int(h.SampleSize))
		if err != nil {
			return nil, err
		}
		sample := Sample(bits.Uint64())
		samples = append(samples, sample)
	}
	for _, sample := range samples {
		dbg.Println("warm-up sample:", sample)
	}

	bits, err := br.Read(2)
	if err != nil {
		return nil, err
	}
	rice := bits.Uint64()
	switch rice {
	case 0:
		// 00: partitioned Rice coding with 4-bit Rice parameter;
		//     RESIDUAL_CODING_METHOD_PARTITIONED_RICE follows
		return h.DecodeRice0(br, order)
	case 1:
		// 01: partitioned Rice coding with 5-bit Rice parameter;
		//     RESIDUAL_CODING_METHOD_PARTITIONED_RICE2 follows
		return h.DecodeRice1(br, order)
	}
	// 1x: reserved
	return nil, fmt.Errorf("frame.DecodeFixed: invalid rice coding method; reserved bit pattern: %02b.", rice)
}

// ref: http://flac.sourceforge.net/format.html#partitioned_rice
func (h *Header) DecodeRice0(br bit.Reader, order int) (samples []Sample, err error) {
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
func (h *Header) DecodeRice1(br bit.Reader, order int) (samples []Sample, err error) {
	/// ### [ todo ] ###
	///    - not yet implemented.
	/// ### [/ todo ] ###
	return nil, fmt.Errorf("not yet implemented; rice coding method: 1.")
}

/**
type SubFrameFixed struct {
	WarmUpSamples []byte
	Residual      []Residual
}

type SubFrameLpc struct {
	WarmUpSamples         []byte
	Precision             uint8
	ShiftNeeded           uint8
	PredictorCoefficients []byte
}

type SubFrameVerbatim struct {
	UnencodedSubblock []byte
}

type Residual struct {
	UsesRice2 bool
}

type Rice struct {
	PartitionOrder uint8
	Partitions     []RicePartition
}

type Rice2 struct {
	PartitionOrder uint8
	Partitions     []Rice2Partition
}

type RicePartition struct {
	EncodingParameter uint16
}

type Rice2Partition struct{}
*/
