// Package frame contains functions for parsing FLAC encoded audio data.
package frame

import dbg "fmt"
import "encoding/binary"
import "encoding/hex"
import "errors"
import "fmt"
import "io"
import "log"
import "math"
import "os"

import "github.com/mewkiz/pkg/hashutil/crc8"
import "github.com/mewkiz/pkg/readerutil"

type Frame struct {
	Header    *Header
	SubFrames []SubFrame
	Footer    FrameFooter
}

// A Header is a frame header, which contains information about the frame like
// the block size, sample rate, number of channels, etc, and an 8-bit CRC.
type Header struct {
	// Blocking strategy:
	//    false: fixed-blocksize stream.
	//    true:  variable-blocksize stream.
	HasVariableBlockSize bool
	// Block size in inter-channel samples.
	BlockSize uint16
	// Sample rate.
	SampleRate uint32
	// Channel order specifies the order in which channels are stored in the
	// frame.
	ChannelOrder ChannelOrder
	// Sample size in bits. Get from StreamInfo metadata block if set to 0.
	SampleSize uint8
	// Sample number is the frame's starting sample number, used by
	// variable-blocksize streams.
	SampleNum uint64
	// Frame number, used by fixed-blocksize streams. The frame's starting sample
	// number will be the frame number times the blocksize.
	FrameNum uint32
}

type SubFrame struct {
	Header *SubFrameHeader
	Block  interface{}
}

type SubFrameConstant struct {
	Value []byte
}

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

type FrameFooter struct {
	CRC uint16
}

/**
f.Footer.CRC = binary.BigEndian.Uint16(buf.Next(2))

const (
	zeroPaddingMask  = 0x80
	subFrameTypeMask = 0x7E
)

subFrame := new(SubFrame)

c, err := buf.ReadByte()
if err != nil {
	return err
}

//Zero bit padding, to prevent sync-fooling string of 1s
if c&zeroPaddingMask != 0 {
	return nil, ErrIsNotNil
}

// Subframe type:
// 000000 : SUBFRAME_CONSTANT
// 000001 : SUBFRAME_VERBATIM
// 00001x : reserved
// 0001xx : reserved
// 001xxx : if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
// 01xxxx : reserved
// 1xxxxx : SUBFRAME_LPC, xxxxx=order-1

subFrame.Header.subFrameType = c & subFrameTypeMask
*/

// Sync code for frame headers. Bit representation: 11111111111110.
const SyncCode = 0x3FFE

// ChannelOrder specifies the order in which channels are stored.
type ChannelOrder uint8

// Channel assignment. The following abbreviations are used:
//    L:   left
//    R:   right
//    C:   center
//    Lfe: low-frequency effects
//    Ls:  left surround
//    Rs:  right surround
//
// The first 6 channel constants follow the SMPTE/ITU-R channel order:
//    L R C Lfe Ls Rs
const (
	ChannelMono       ChannelOrder = iota // 1 channel:  mono.
	ChannelLR                             // 2 channels: left, right
	ChannelLRC                            // 3 channels: left, right, center
	ChannelLRLsRs                         // 4 channels: left, right, left surround, right surround
	ChannelLRCLsRs                        // 5 channels: left, right, center, left surround, right surround
	ChannelLRCLfeLsRs                     // 6 channels: left, right, center, low-frequency effects, left surround, right surround
	Channel7                              // 7 channels: not defined
	Channel8                              // 8 channels: not defined
	ChannelLSide                          // left/side stereo:  left, side (difference)
	ChannelSideR                          // side/right stereo: side (difference), right
	ChannelMidSide                        // mid/side stereo:   mid (average), side (difference)
)

// NewHeader parses and returns a new frame header.
//
// Frame header format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#frame_header
//
//    type FRAME_HEADER struct {
//       sync_code               uint14
//       _                       uint1
//       has_variable_block_size bool
//       block_size_spec         uint4
//       sample_rate_spec        uint4
//       channel_assignment      uint4
//       sample_size_spec        uint3
//       _                       uint1
//       if has_variable_block_size {
//          // "UTF-8" coded int, from 1 to 7 bytes.
//          sample_num           uint36
//       } else {
//          // "UTF-8" coded int, from 1 to 6 bytes.
//          frame_num            uint31
//       }
//       switch block_size_spec {
//       case 0110:
//          block_size           uint8  // block_size-1
//       case 0111:
//          block_size           uint16 // block_size-1
//       }
//       switch sample_rate_spec {
//       case 1100:
//          sample_rate          uint8  // sample rate in kHz.
//       case 1101:
//          sample_rate          uint16 // sample rate in Hz.
//       case 1110:
//          sample_rate          uint16 // sample rate in daHz (tens of Hz).
//       }
//       crc8                    uint8
//    }
func NewHeader(r io.ReadSeeker) (h *Header, err error) {
	// Record start offset, which is used when verifying the CRC-8 of the frame
	// header.
	start, err := r.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	// Read 32 bits which are arranged according to the following masks.
	const (
		SyncCodeMask          = 0xFFFC0000 // 14 bits   shift right: 18
		Reserved1Mask         = 0x00020000 // 1 bit     shift right: 17
		BlockingStrategyMask  = 0x00010000 // 1 bit     shift right: 16
		BlockSizeSpecMask     = 0x0000F000 // 4 bits    shift right: 12
		SampleRateSpecMask    = 0x00000F00 // 4 bits    shift right: 8
		ChannelAssignmentMask = 0x000000F0 // 4 bits    shift right: 4
		SampleSizeSpecMask    = 0x0000000E // 3 bits    shift right: 1
		Reserved2Mask         = 0x00000001 // 1 bit     shift right: 0
	)
	var bits uint32
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Sync code.
	syncCode := bits & SyncCodeMask >> 18
	if syncCode != SyncCode {
		return nil, fmt.Errorf("frame.NewHeader: invalid sync code; expected '%014b', got '%014b'.", SyncCode, syncCode)
	}

	// Reserved.
	if bits&Reserved1Mask != 0 {
		return nil, errors.New("frame.NewHeader: all reserved bits must be 0.")
	}

	// Blocking strategy.
	h = new(Header)
	if bits&BlockingStrategyMask != 0 {
		// blocking strategy:
		//    0: fixed-blocksize.
		//    1: variable-blocksize.
		h.HasVariableBlockSize = true
	}

	// Channel assignment.
	//    0000-0111: (number of independent channels)-1. Where defined, the
	//               channel order follows SMPTE/ITU-R recommendations. The
	//               assignments are as follows:
	//       1 channel: mono
	//       2 channels: left, right
	//       3 channels: left, right, center
	//       4 channels: left, right, left surround, right surround
	//       5 channels: left, right, center, left surround, right surround
	//       6 channels: left, right, center, low-frequency effects, left surround, right surround
	//       7 channels: not defined
	//       8 channels: not defined
	//    1000: left/side stereo:  left, side (difference)
	//    1001: side/right stereo: side (difference), right
	//    1010: mid/side stereo:   mid (average), side (difference)
	//    1011-1111: reserved
	n := bits & ChannelAssignmentMask >> 4
	switch {
	case n >= 0 && n <= 10:
		// 0000-0111: (number of independent channels)-1. Where defined, the
		//            channel order follows SMPTE/ITU-R recommendations. The
		//            assignments are as follows:
		//    1 channel: mono
		//    2 channels: left, right
		//    3 channels: left, right, center
		//    4 channels: left, right, left surround, right surround
		//    5 channels: left, right, center, left surround, right surround
		//    6 channels: left, right, center, low-frequency effects, left surround, right surround
		//    7 channels: not defined
		//    8 channels: not defined
		// 1000: left/side stereo:  left, side (difference)
		// 1001: side/right stereo: side (difference), right
		// 1010: mid/side stereo:   mid (average), side (difference)
		h.ChannelOrder = ChannelOrder(n)
	case n >= 11 && n <= 15:
		// 1011-1111: reserved
		return nil, fmt.Errorf("frame.NewHeader: invalid channel order; reserved bit pattern: %04b.", n)
	default:
		// should be unreachable.
		log.Fatalln(fmt.Errorf("frame.NewHeader: unhandled channel assignment bit pattern: %04b.", n))
	}

	// Sample size.
	//    000: get from STREAMINFO metadata block.
	//    001: 8 bits per sample.
	//    010: 12 bits per sample.
	//    011: reserved.
	//    100: 16 bits per sample.
	//    101: 20 bits per sample.
	//    110: 24 bits per sample.
	//    111: reserved.
	n = bits & SampleSizeSpecMask >> 1
	switch n {
	case 0:
		// 000: get from STREAMINFO metadata block.
		/// ### [ todo ] ###
		///    - Should we try to read StreamInfo from here? We won't always have
		///      access to it.
		/// ### [/ todo ] ###
		log.Println(fmt.Errorf("not yet implemented; sample size: %d.", n))
	case 1:
		// 001: 8 bits per sample.
		h.SampleSize = 8
	case 2:
		// 010: 12 bits per sample.
		h.SampleSize = 12
	case 3, 7:
		// 011: reserved.
		// 111: reserved.
		return nil, fmt.Errorf("frame.NewHeader: invalid sample size; reserved bit pattern: %03b.", n)
	case 4:
		// 100: 16 bits per sample.
		h.SampleSize = 16
	case 5:
		// 101: 20 bits per sample.
		h.SampleSize = 20
	case 6:
		// 110: 24 bits per sample.
		h.SampleSize = 24
	default:
		// should be unreachable.
		log.Fatalln(fmt.Errorf("frame.NewHeader: unhandled sample size bit pattern: %03b.", n))
	}

	// Reserved.
	if bits&Reserved2Mask != 0 {
		return nil, errors.New("frame.NewHeader: all reserved bits must be 0.")
	}

	// "UTF-8" coded sample number or frame number.
	if h.HasVariableBlockSize {
		// Sample number.
		h.SampleNum, err = decodeUTF8Int(r)
		if err != nil {
			return nil, err
		}
		dbg.Println("UTF-8 decoded sample number:", h.SampleNum)
	} else {
		// Frame number.
		frameNum, err := decodeUTF8Int(r)
		if err != nil {
			return nil, err
		}
		h.FrameNum = uint32(frameNum)
		dbg.Println("UTF-8 decoded frame number:", h.FrameNum)
	}

	// Block size.
	//    0000: reserved.
	//    0001: 192 samples.
	//    0010-0101: 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608.
	//    0110: get 8 bit (blocksize-1) from end of header.
	//    0111: get 16 bit (blocksize-1) from end of header.
	//    1000-1111: 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/
	//               16384/32768.
	n = bits & BlockSizeSpecMask >> 12
	switch {
	case n == 0:
		// 0000: reserved.
		return nil, errors.New("frame.NewHeader: invalid block size; reserved bit pattern.")
	case n == 1:
		// 0001: 192 samples.
		h.BlockSize = 192
	case n >= 2 && n <= 5:
		// 0010-0101: 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608.
		h.BlockSize = uint16(576 * math.Pow(2, float64(n-2)))
	case n == 6:
		// 0110: get 8 bit (blocksize-1) from end of header.
		var x uint8
		err = binary.Read(r, binary.BigEndian, &x)
		if err != nil {
			return nil, err
		}
		h.BlockSize = uint16(x) + 1
		dbg.Println("block size: %d (8 bits).", h.BlockSize)
	case n == 7:
		// 0111: get 16 bit (blocksize-1) from end of header.
		var x uint16
		err = binary.Read(r, binary.BigEndian, &x)
		if err != nil {
			return nil, err
		}
		h.BlockSize = x + 1
		dbg.Println("block size: %d (16 bits).", h.BlockSize)
	case n >= 8 && n <= 15:
		// 1000-1111: 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/
		//            16384/32768.
		h.BlockSize = uint16(256 * math.Pow(2, float64(n-8)))
	default:
		// should be unreachable.
		log.Fatalln(fmt.Errorf("frame.NewHeader: unhandled block size bit pattern: %04b.", n))
	}

	// Sample rate:
	//    0000: get from STREAMINFO metadata block.
	//    0001: 88.2kHz.
	//    0010: 176.4kHz.
	//    0011: 192kHz.
	//    0100: 8kHz.
	//    0101: 16kHz.
	//    0110: 22.05kHz.
	//    0111: 24kHz.
	//    1000: 32kHz.
	//    1001: 44.1kHz.
	//    1010: 48kHz.
	//    1011: 96kHz.
	//    1100: get 8 bit sample rate (in kHz) from end of header.
	//    1101: get 16 bit sample rate (in Hz) from end of header.
	//    1110: get 16 bit sample rate (in tens of Hz) from end of header.
	//    1111: invalid, to prevent sync-fooling string of 1s.
	n = bits & SampleRateSpecMask >> 8
	switch n {
	case 0:
		// 0000: get from STREAMINFO metadata block.
		/// ### [ todo ] ###
		///    - add flag to get from StreamInfo.
		/// ### [/ todo ] ###
		log.Println(fmt.Errorf("not yet implemented; sample rate: %d.", n))
	case 1:
		//0001: 88.2kHz.
		h.SampleRate = 88200
	case 2:
		//0010: 176.4kHz.
		h.SampleRate = 176400
	case 3:
		//0011: 192kHz.
		h.SampleRate = 192000
	case 4:
		//0100: 8kHz.
		h.SampleRate = 8000
	case 5:
		//0101: 16kHz.
		h.SampleRate = 16000
	case 6:
		//0110: 22.05kHz.
		h.SampleRate = 22050
	case 7:
		//0111: 24kHz.
		h.SampleRate = 24000
	case 8:
		//1000: 32kHz.
		h.SampleRate = 32000
	case 9:
		//1001: 44.1kHz.
		h.SampleRate = 44100
	case 10:
		//1010: 48kHz.
		h.SampleRate = 48000
	case 11:
		//1011: 96kHz.
		h.SampleRate = 96000
	case 12:
		//1100: get 8 bit sample rate (in kHz) from end of header.
		var sampleRate_kHz uint8
		err = binary.Read(r, binary.BigEndian, &sampleRate_kHz)
		if err != nil {
			return nil, err
		}
		dbg.Printf("sample rate: %d kHz.\n", sampleRate_kHz)
		h.SampleRate = uint32(sampleRate_kHz) * 1000
	case 13:
		//1101: get 16 bit sample rate (in Hz) from end of header.
		var sampleRate_Hz uint16
		err = binary.Read(r, binary.BigEndian, &sampleRate_Hz)
		if err != nil {
			return nil, err
		}
		dbg.Printf("sample rate: %d Hz.\n", sampleRate_Hz)
		h.SampleRate = uint32(sampleRate_Hz)
	case 14:
		//1110: get 16 bit sample rate (in tens of Hz) from end of header.
		var sampleRate_daHz uint16
		err = binary.Read(r, binary.BigEndian, &sampleRate_daHz)
		if err != nil {
			return nil, err
		}
		dbg.Printf("sample rate: %d daHz.\n", sampleRate_daHz)
		h.SampleRate = uint32(sampleRate_daHz) * 10
	case 15:
		//1111: invalid, to prevent sync-fooling string of 1s.
		return nil, fmt.Errorf("frame.NewHeader: invalid sample rate bit pattern: %04b.", n)
	default:
		// should be unreachable.
		log.Fatalln(fmt.Errorf("frame.NewHeader: unhandled sample rate bit pattern: %04b.", n))
	}

	// Read the frame header data and calculate the CRC-8.
	end, err := r.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}
	_, err = r.Seek(start, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	data := make([]byte, end-start)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}

	// Verify the CRC-8.
	var crc uint8
	err = binary.Read(r, binary.BigEndian, &crc)
	if err != nil {
		return nil, err
	}
	got := crc8.ChecksumATM(data)
	if crc != got {
		return nil, fmt.Errorf("frame.NewHeader: checksum mismatch; expected 0x%02X, got 0x%02X.", crc, got)
	}
	dbg.Println("crc:", crc)
	dbg.Println("got:", got)
	dbg.Println(hex.Dump(data))

	return h, nil
}

type SubFrameHeader struct {
	SubFrameType uint8
	WastedBits   uint8
}

func NewSubFrameHeader(r io.Reader) (sh *SubFrameHeader, err error) {
	// Read 8 bits which are arranged according to the following masks.
	const (
		PaddingMask      = 0x80 // 1 bit    shift right: 7
		SubFrameTypeMask = 0x7E // 6 bits   shift right: 1
		WastedBitsMask   = 0x01 // 1 bit    shift right: 0
	)
	bits, err := readerutil.ReadByte(r)
	if err != nil {
		return nil, err
	}

	// Padding.
	if bits&PaddingMask != 0 {
		return nil, errors.New("frame.NewSubFrameHeader: invalid padding; must be 0.")
	}

	// Subframe type.
	//    000000: SUBFRAME_CONSTANT
	//    000001: SUBFRAME_VERBATIM
	//    00001x: reserved
	//    0001xx: reserved
	//    001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
	//    01xxxx: reserved
	//    1xxxxx: SUBFRAME_LPC, xxxxx=order-1
	n := bits & SubFrameTypeMask >> 1
	switch {
	case n == 0:
		// 000000: SUBFRAME_CONSTANT
		/// ### [ todo ] ###
		///    - handle subframe constant.
		/// ### [/ todo ] ###
		log.Println(fmt.Errorf("not yet implemented; subframe type: %d.", n))
	case n == 1:
		// 000001: SUBFRAME_VERBATIM
		/// ### [ todo ] ###
		///    - handle subframe verbatim.
		/// ### [/ todo ] ###
		log.Println(fmt.Errorf("not yet implemented; subframe type: %d.", n))
	case n < 8:
		// 00001x: reserved
		// 0001xx: reserved
		return nil, fmt.Errorf("frame.NewSubFrameHeader: invalid subframe type; reserved bit pattern: %06b.", n)
	case n < 16:
		// 001xxx: if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
		const orderMask = 0x07
		order := n & orderMask
		if order > 4 {
			return nil, fmt.Errorf("frame.NewSubFrameHeader: invalid subframe type; reserved bit pattern: %06b.", n)
		}
		dbg.Println("subframe fixed order:", order)
		/// ### [ todo ] ###
		///    - get subframe fixed order.
		/// ### [/ todo ] ###
		log.Println(fmt.Errorf("not yet implemented; subframe type: %d.", n))
	case n < 32:
		// 01xxxx: reserved
		return nil, fmt.Errorf("frame.NewSubFrameHeader: invalid subframe type; reserved bit pattern: %06b.", n)
	case n < 64:
		// 1xxxxx: SUBFRAME_LPC, xxxxx=order-1
		const orderMask = 0x1F
		order := n & orderMask
		dbg.Println("subframe LPC order:", order)
		/// ### [ todo ] ###
		///    - get subframe lpc order.
		/// ### [/ todo ] ###
		log.Println(fmt.Errorf("not yet implemented; subframe type: %d.", n))
	default:
		// should be unreachable.
		log.Fatalln(fmt.Errorf("frame.NewSubFrameHeader: unhandled subframe type bit pattern: %06b.", n))
	}

	// Wasted bits-per-sample.
	if bits&WastedBitsMask != 0 {
		/// ### [ todo ] ###
		///    - handle wasted bits-per-sample.
		/// ### [/ todo ] ###
		log.Println(errors.New("not yet implemented; wasted bits-per-sample."))
	}
	return sh, nil
}
