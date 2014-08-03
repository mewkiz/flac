package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/mewkiz/pkg/bit"
	"github.com/mewkiz/pkg/dbg"
	"github.com/mewkiz/pkg/hashutil/crc8"
)

// A Header is a frame header, which contains information about the frame like
// the block size, sample rate, number of channels, etc, and an 8-bit CRC.
type Header struct {
	// Blocking strategy:
	//    false: fixed-sample count stream.
	//    true:  variable-sample count stream.
	HasVariableSampleCount bool
	// Sample count is the number of samples in any of a block's subblocks.
	SampleCount uint16
	// Sample rate. Get from StreamInfo metadata block if set to 0.
	SampleRate uint32
	// Channel order specifies the order in which channels are stored in the
	// frame.
	ChannelOrder ChannelOrder
	// Sample size in bits-per-sample. Get from StreamInfo metadata block if set
	// to 0.
	BitsPerSample uint8
	// Sample number is the frame's starting sample number, used by
	// variable-sample count streams.
	SampleNum uint64
	// Frame number, used by fixed-sample count streams. The frame's starting
	// sample number will be the frame number times the sample count.
	FrameNum uint32
}

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
	ChannelLeftSide                       // left/side stereo:  left, side (difference)
	ChannelRightSide                      // side/right stereo: side (difference), right
	ChannelMidSide                        // mid/side stereo:   mid (average), side (difference)
)

// channelCount maps from a channel assignment to its number of channels.
var channelCount = map[ChannelOrder]int{
	ChannelMono:       1,
	ChannelLR:         2,
	ChannelLRC:        3,
	ChannelLRLsRs:     4,
	ChannelLRCLsRs:    5,
	ChannelLRCLfeLsRs: 6,
	Channel7:          7,
	Channel8:          8,
	ChannelLeftSide:   2,
	ChannelRightSide:  2,
	ChannelMidSide:    2,
}

// ChannelCount returns the number of channels used by the provided channel
// order.
func (order ChannelOrder) ChannelCount() int {
	return channelCount[order]
}

// NewHeader parses and returns a new frame header.
//
// Frame header format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#frame_header
//
//    type FRAME_HEADER struct {
//       sync_code                 uint14
//       _                         uint1
//       has_variable_sample_count bool   // referred to as "variable blocksize" in the spec.
//       sample_count_spec         uint4  // referred to as "blocksize" in the spec.
//       sample_rate_spec          uint4
//       channel_assignment        uint4
//       sample_size_spec          uint3
//       _                         uint1
//       if has_variable_sample_count {
//          // "UTF-8" coded int, from 1 to 7 bytes.
//          sample_num             uint36
//       } else {
//          // "UTF-8" coded int, from 1 to 6 bytes.
//          frame_num              uint31
//       }
//       switch sample_count_spec {
//       case 0110:
//          sample_count           uint8  // sample_count-1
//       case 0111:
//          sample_count           uint16 // sample_count-1
//       }
//       switch sample_rate_spec {
//       case 1100:
//          sample_rate            uint8  // sample rate in kHz.
//       case 1101:
//          sample_rate            uint16 // sample rate in Hz.
//       case 1110:
//          sample_rate            uint16 // sample rate in daHz (tens of Hz).
//       }
//       crc8                      uint8
//    }
func NewHeader(r io.Reader) (hdr *Header, err error) {
	// Create a new hash reader which adds the data from all read operations to a
	// running hash.
	h := crc8.NewATM()
	hr := io.TeeReader(r, h)

	br := bit.NewReader(hr)
	// field 0: sync_code                 (14 bits)
	// field 1: reserved                  (1 bit)
	// field 2: has_variable_sample_count (1 bit)
	// field 3: sample_count_spec         (4 bits)
	// field 4: sample_rate_spec          (4 bits)
	// field 5: channel_assignment        (4 bits)
	// field 6: sample_size_spec          (3 bits)
	// field 7: reserved                  (1 bit)
	fields, err := br.ReadFields(14, 1, 1, 4, 4, 4, 3, 1)
	if err != nil {
		return nil, err
	}

	// Sync code.
	// field 0: sync_code (14 bits)
	syncCode := fields[0]
	if syncCode != SyncCode {
		return nil, fmt.Errorf("frame.NewHeader: invalid sync code; expected '%014b', got '%014b'", SyncCode, syncCode)
	}

	// Reserved.
	// field 1: reserved (1 bit)
	if fields[1] != 0 {
		return nil, errors.New("frame.NewHeader: all reserved bits must be 0")
	}

	// Blocking strategy.
	hdr = new(Header)
	// field 2: has_variable_sample_count (1 bit)
	if fields[2] != 0 {
		// blocking strategy:
		//    0: fixed-sample count.
		//    1: variable-sample count.
		hdr.HasVariableSampleCount = true
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
	// field 5: channel_assignment (4 bits)
	n := fields[5]
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
		hdr.ChannelOrder = ChannelOrder(n)
	case n >= 11 && n <= 15:
		// 1011-1111: reserved
		return nil, fmt.Errorf("frame.NewHeader: invalid channel order; reserved bit pattern: %04b", n)
	default:
		// should be unreachable.
		panic(fmt.Errorf("frame.NewHeader: unhandled channel assignment bit pattern: %04b", n))
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
	// field 6: sample_size_spec (3 bits)
	n = fields[6]
	switch n {
	case 0:
		// 000: get from STREAMINFO metadata block.
		// TODO(u): Should we try to read StreamInfo from here? We won't always
		// have access to it.
		panic("not yet implemented; bits-per-sample 0")
	case 1:
		// 001: 8 bits per sample.
		hdr.BitsPerSample = 8
	case 2:
		// 010: 12 bits per sample.
		hdr.BitsPerSample = 12
	case 3, 7:
		// 011: reserved.
		// 111: reserved.
		return nil, fmt.Errorf("frame.NewHeader: invalid sample size; reserved bit pattern: %03b", n)
	case 4:
		// 100: 16 bits per sample.
		hdr.BitsPerSample = 16
	case 5:
		// 101: 20 bits per sample.
		hdr.BitsPerSample = 20
	case 6:
		// 110: 24 bits per sample.
		hdr.BitsPerSample = 24
	default:
		// should be unreachable.
		panic(fmt.Errorf("frame.NewHeader: unhandled sample size bit pattern: %03b", n))
	}

	// Reserved.
	// field 7: reserved (1 bit)
	if fields[7] != 0 {
		return nil, errors.New("frame.NewHeader: all reserved bits must be 0")
	}

	// "UTF-8" coded sample number or frame number.
	if hdr.HasVariableSampleCount {
		// Sample number.
		hdr.SampleNum, err = decodeUTF8Int(hr)
		if err != nil {
			return nil, err
		}
		dbg.Println("UTF-8 decoded sample number:", hdr.SampleNum)
	} else {
		// Frame number.
		frameNum, err := decodeUTF8Int(hr)
		if err != nil {
			return nil, err
		}
		hdr.FrameNum = uint32(frameNum)
		dbg.Println("UTF-8 decoded frame number:", hdr.FrameNum)
	}

	// Block size.
	//    0000: reserved.
	//    0001: 192 samples.
	//    0010-0101: 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608.
	//    0110: get 8 bit (sampleCount-1) from end of header.
	//    0111: get 16 bit (sampleCount-1) from end of header.
	//    1000-1111: 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/
	//               16384/32768.
	// field 3: sample_count_spec (4 bits)
	n = fields[3]
	switch {
	case n == 0:
		// 0000: reserved.
		return nil, errors.New("frame.NewHeader: invalid block size; reserved bit pattern")
	case n == 1:
		// 0001: 192 samples.
		hdr.SampleCount = 192
	case n >= 2 && n <= 5:
		// 0010-0101: 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608.
		hdr.SampleCount = uint16(576 * math.Pow(2, float64(n-2)))
	case n == 6:
		// 0110: get 8 bit (sampleCount-1) from end of header.
		var x uint8
		err = binary.Read(hr, binary.BigEndian, &x)
		if err != nil {
			return nil, err
		}
		hdr.SampleCount = uint16(x) + 1
	case n == 7:
		// 0111: get 16 bit (sampleCount-1) from end of header.
		var x uint16
		err = binary.Read(hr, binary.BigEndian, &x)
		if err != nil {
			return nil, err
		}
		hdr.SampleCount = x + 1
	case n >= 8 && n <= 15:
		// 1000-1111: 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/
		//            16384/32768.
		hdr.SampleCount = uint16(256 * math.Pow(2, float64(n-8)))
	default:
		// should be unreachable.
		panic(fmt.Errorf("frame.NewHeader: unhandled block size bit pattern: %04b", n))
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
	// field 4: sample_rate_spec (4 bits)
	n = fields[4]
	switch n {
	case 0:
		// 0000: get from STREAMINFO metadata block.
		// TODO(u): Add flag to get from StreamInfo?
		panic("not yet implemented; sample rate 0")
	case 1:
		//0001: 88.2kHz.
		hdr.SampleRate = 88200
	case 2:
		//0010: 176.4kHz.
		hdr.SampleRate = 176400
	case 3:
		//0011: 192kHz.
		hdr.SampleRate = 192000
	case 4:
		//0100: 8kHz.
		hdr.SampleRate = 8000
	case 5:
		//0101: 16kHz.
		hdr.SampleRate = 16000
	case 6:
		//0110: 22.05kHz.
		hdr.SampleRate = 22050
	case 7:
		//0111: 24kHz.
		hdr.SampleRate = 24000
	case 8:
		//1000: 32kHz.
		hdr.SampleRate = 32000
	case 9:
		//1001: 44.1kHz.
		hdr.SampleRate = 44100
	case 10:
		//1010: 48kHz.
		hdr.SampleRate = 48000
	case 11:
		//1011: 96kHz.
		hdr.SampleRate = 96000
	case 12:
		//1100: get 8 bit sample rate (in kHz) from end of header.
		var sampleRate_kHz uint8
		err = binary.Read(hr, binary.BigEndian, &sampleRate_kHz)
		if err != nil {
			return nil, err
		}
		hdr.SampleRate = uint32(sampleRate_kHz) * 1000
	case 13:
		//1101: get 16 bit sample rate (in Hz) from end of header.
		var sampleRate_Hz uint16
		err = binary.Read(hr, binary.BigEndian, &sampleRate_Hz)
		if err != nil {
			return nil, err
		}
		hdr.SampleRate = uint32(sampleRate_Hz)
	case 14:
		//1110: get 16 bit sample rate (in tens of Hz) from end of header.
		var sampleRate_daHz uint16
		err = binary.Read(hr, binary.BigEndian, &sampleRate_daHz)
		if err != nil {
			return nil, err
		}
		hdr.SampleRate = uint32(sampleRate_daHz) * 10
	case 15:
		//1111: invalid, to prevent sync-fooling string of 1s.
		return nil, fmt.Errorf("frame.NewHeader: invalid sample rate bit pattern: %04b", n)
	default:
		// should be unreachable.
		panic(fmt.Errorf("frame.NewHeader: unhandled sample rate bit pattern: %04b", n))
	}

	// Verify the CRC-8.
	got := h.Sum8()

	var want uint8
	err = binary.Read(r, binary.BigEndian, &want)
	if err != nil {
		return nil, err
	}
	if got != want {
		return nil, fmt.Errorf("frame.NewHeader: checksum mismatch; expected 0x%02X, got 0x%02X", want, got)
	}

	return hdr, nil
}
