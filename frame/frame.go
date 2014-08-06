// Package frame implements access to FLAC audio frames.
package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/mewkiz/pkg/bit"
	"github.com/mewkiz/pkg/hashutil"
	"github.com/mewkiz/pkg/hashutil/crc16"
	"github.com/mewkiz/pkg/hashutil/crc8"
)

// A Frame contains the header and subframes of an audio frame. It holds the
// encoded samples from a block (a part) of the audio stream. Each subframe
// holding the samples from one of its channel.
//
// ref: https://www.xiph.org/flac/format.html#frame
type Frame struct {
	// Audio frame header.
	Header
	// One subframe per channel, containing encoded audio samples.
	Subframes []*Subframe
	// CRC-16 hash sum, calculated by read operations on hr.
	crc hashutil.Hash16
	// A CRC-16 hash reader, wrapping read operations to r.
	hr io.Reader
	// Underlying io.Reader.
	r io.Reader
}

// New creates a new Frame for accessing the audio samples of r. It reads and
// parses an audio frame header. Call Frame.Parse to parse the audio samples of
// its subframes.
func New(r io.Reader) (frame *Frame, err error) {
	// Create a new CRC-16 hash reader which adds the data from all read
	// operations to a running hash.
	crc := crc16.NewIBM()
	hr := io.TeeReader(r, crc)

	// Parse frame header.
	frame = &Frame{crc: crc, hr: hr, r: r}
	err = frame.parseHeader()
	return frame, err
}

// Parse reads and parses the header, and the audio samples from each subframe
// of a frame. If the samples are interchannel correlated between the subframes,
// it decorrelates them.
//
// ref: https://www.xiph.org/flac/format.html#interchannel
func Parse(r io.Reader) (frame *Frame, err error) {
	// Parse frame header.
	frame, err = New(r)
	if err != nil {
		return frame, err
	}

	// Parse subframes.
	err = frame.Parse()
	return frame, err
}

// Parse reads and parses the audio samples from each subframe of the frame. If
// the samples are interchannel correlated between the subframes, it
// decorrelates them.
//
// ref: https://www.xiph.org/flac/format.html#interchannel
func (frame *Frame) Parse() error {
	// Parse subframes.
	frame.Subframes = make([]*Subframe, frame.Channels.Count())
	var err error
	for i := range frame.Subframes {
		frame.Subframes[i], err = frame.parseSubframe()
		if err != nil {
			return err
		}
	}

	// Decorrelate subframe samples.
	// TODO(u): Implement interchannel decorrelation of samples.

	// 2 bytes: CRC-16 checksum.
	var want uint16
	err = binary.Read(frame.r, binary.BigEndian, &want)
	if err != nil {
		return err
	}
	got := frame.crc.Sum16()
	if got != want {
		return fmt.Errorf("frame.Frame.Parse: CRC-16 checksum mismatch; expected %v, got %v", want, got)
	}

	return nil
}

// A Header contains the basic properties of an audio frame, such as its sample
// rate and channel count. To facilitate random access decoding each frame
// header starts with a sync-code. This allows the decoder to synchronize and
// locate the start of a frame header.
//
// ref: https://www.xiph.org/flac/format.html#frame_header
type Header struct {
	// Specifies if the block size is fixed or variable.
	HasFixedBlockSize bool
	// Block size in inter-channel samples, i.e. the number of audio samples in
	// each subframe.
	BlockSize uint16
	// Sample rate in Hz; a 0 value implies unknown, get sample rate from
	// StreamInfo.
	SampleRate uint32
	// Specifies the number of channels (subframes) that exist in the frame,
	// their order and possible interchannel correlation.
	Channels Channels
	// Sample size in bits-per-sample; a 0 value implies unknown, get sample size
	// from StreamInfo.
	BitsPerSample uint8
	// Specifies the frame number if the block size is fixed, and the first
	// sample number in the frame otherwise. When using fixed block size, the
	// first sample number in the frame can be derived by multiplying the frame
	// number with the block size (in samples).
	Num uint64
}

// Errors returned by Frame.parseHeader.
var (
	ErrInvalidSync = errors.New("frame.Frame.parseHeader: invalid sync-code")
)

// parseHeader reads and parses the header of an audio frame.
func (frame *Frame) parseHeader() error {
	// Create a new CRC-8 hash reader which adds the data from all read
	// operations to a running hash.
	h := crc8.NewATM()
	hr := io.TeeReader(frame.hr, h)

	// 14 bits: sync-code (11111111111110)
	br := bit.NewReader(hr)
	x, err := br.Read(14)
	if err != nil {
		return err
	}
	if x != 0x3FFE {
		return ErrInvalidSync
	}

	// 1 bit: reserved.
	x, err = br.Read(1)
	if err != nil {
		return err
	}
	if x != 0 {
		return errors.New("frame.Frame.parseHeader: non-zero reserved value")
	}

	// 1 bit: HasFixedBlockSize.
	x, err = br.Read(1)
	if err != nil {
		return err
	}
	if x != 0 {
		frame.HasFixedBlockSize = true
	}

	// 4 bits: BlockSize. The block size parsing is simplified by deferring it to
	// the end of the header.
	blockSize, err := br.Read(4)
	if err != nil {
		return err
	}

	// 4 bits: SampleRate. The sample rate parsing is simplified by deferring it
	// to the end of the header.
	sampleRate, err := br.Read(4)
	if err != nil {
		return err
	}

	// 4 bits: Channels.
	//
	// The 4 bits are used to specify the channels as follows:
	//    0000: (1 channel) mono
	//    0001: (2 channels) left, right
	//    0010: (3 channels) left, right, center
	//    0011: (4 channels) left, right, left surround, right surround
	//    0100: (5 channels) left, right, center, left surround, right surround
	//    0101: (6 channels) left, right, center, LFE, left surround, right surround
	//    0110: (7 channels) left, right, center, LFE, center surround, side left, side right
	//    0111: (8 channels) left, right, center, LFE, left surround, right surround, side left, side right
	//    1000: (2 channels) left, side; using interchannel correlation
	//    1001: (2 channels) side, right; using interchannel correlation
	//    1010: (2 channels) mid, side; using interchannel correlation
	//    1011: reserved.
	//    1100: reserved.
	//    1101: reserved.
	//    1111: reserved.
	x, err = br.Read(4)
	if err != nil {
		return err
	}
	if x >= 0xB {
		return fmt.Errorf("frame.Frame.parseHeader: reserved channels bit pattern (%04b)", x)
	}
	frame.Channels = Channels(x)

	// 3 bits: BitsPerSample.
	x, err = br.Read(3)
	if err != nil {
		return err
	}
	// The 3 bits are used to specify the sample size as follows:
	//    000: unknown sample size; get from StreamInfo.
	//    001: 8 bits-per-sample.
	//    010: 12 bits-per-sample.
	//    011: reserved.
	//    100: 16 bits-per-sample.
	//    101: 20 bits-per-sample.
	//    110: 24 bits-per-sample.
	//    111: reserved.
	switch x {
	case 0x0:
		// 000: unknown bits-per-sample; get from StreamInfo.
	case 0x1:
		// 001: 8 bits-per-sample.
		frame.BitsPerSample = 8
	case 0x2:
		// 010: 12 bits-per-sample.
		frame.BitsPerSample = 12
	case 0x4:
		// 100: 16 bits-per-sample.
		frame.BitsPerSample = 16
	case 0x5:
		// 101: 20 bits-per-sample.
		frame.BitsPerSample = 20
	case 0x6:
		// 110: 24 bits-per-sample.
		frame.BitsPerSample = 24
	default:
		// 011: reserved.
		// 111: reserved.
		return fmt.Errorf("frame.Frame.parseHeader: reserved sample size bit pattern (%03b)", x)
	}

	// 1 bit: reserved.
	x, err = br.Read(1)
	if err != nil {
		return err
	}
	if x != 0 {
		return errors.New("frame.Frame.parseHeader: non-zero reserved value")
	}

	// if (fixed block size)
	//    1-6 bytes: UTF-8 encoded frame number.
	// else
	//    1-7 bytes: UTF-8 encoded sample number.
	frame.Num, err = decodeUTF8Int(hr)
	if err != nil {
		return err
	}

	// Parse block size.
	//
	// The 4 bits of n are used to specify the block size as follows:
	//    0000: reserved.
	//    0001: 192 samples.
	//    0010-0101: 576 * 2^(n-2) samples.
	//    0110: get 8 bit (block size)-1 from the end of the header.
	//    0111: get 16 bit (block size)-1 from the end of the header.
	//    1000-1111: 256 * 2^(n-8) samples.
	n := blockSize
	switch {
	case n == 0x0:
		// 0000: reserved.
		return errors.New("frame.Frame.parseHeader: reserved block size bit pattern (0000)")
	case n == 0x1:
		// 0001: 192 samples.
		frame.BlockSize = 192
	case n >= 0x2 && n <= 0x5:
		// 0010-0101: 576 * 2^(n-2) samples.
		frame.BlockSize = 576 * (1 << (n - 2))
	case n == 0x6:
		// 0110: get 8 bit (block size)-1 from the end of the header.
		x, err = br.Read(8)
		if err != nil {
			return err
		}
		frame.BlockSize = uint16(x + 1)
	case n == 0x7:
		// 0111: get 16 bit (block size)-1 from the end of the header.
		x, err = br.Read(16)
		if err != nil {
			return err
		}
		frame.BlockSize = uint16(x + 1)
	default:
		//    1000-1111: 256 * 2^(n-8) samples.
		frame.BlockSize = 256 * (1 << (n - 8))
	}

	// Parse sample rate.
	//
	// The 4 bits are used to specify the sample rate as follows:
	//    0000: unknown sample rate; get from StreamInfo.
	//    0001: 88.2 kHz.
	//    0010: 176.4 kHz.
	//    0011: 192 kHz.
	//    0100: 8 kHz.
	//    0101: 16 kHz.
	//    0110: 22.05 kHz.
	//    0111: 24 kHz.
	//    1000: 32 kHz.
	//    1001: 44.1 kHz.
	//    1010: 48 kHz.
	//    1011: 96 kHz.
	//    1100: get 8 bit sample rate (in kHz) from the end of the header.
	//    1101: get 16 bit sample rate (in Hz) from the end of the header.
	//    1110: get 16 bit sample rate (in daHz) from the end of the header.
	//    1111: invalid.
	switch sampleRate {
	case 0x0:
		// 0000: unknown sample rate; get from StreamInfo.
	case 0x1:
		// 0001: 88.2 kHz.
		frame.SampleRate = 88200
	case 0x2:
		// 0010: 176.4 kHz.
		frame.SampleRate = 176400
	case 0x3:
		// 0011: 192 kHz.
		frame.SampleRate = 192000
	case 0x4:
		// 0100: 8 kHz.
		frame.SampleRate = 8000
	case 0x5:
		// 0101: 16 kHz.
		frame.SampleRate = 16000
	case 0x6:
		// 0110: 22.05 kHz.
		frame.SampleRate = 22050
	case 0x7:
		// 0111: 24 kHz.
		frame.SampleRate = 24000
	case 0x8:
		// 1000: 32 kHz.
		frame.SampleRate = 32000
	case 0x9:
		// 1001: 44.1 kHz.
		frame.SampleRate = 44100
	case 0xA:
		// 1010: 48 kHz.
		frame.SampleRate = 48000
	case 0xB:
		// 1011: 96 kHz.
		frame.SampleRate = 96000
	case 0xC:
		// 1100: get 8 bit sample rate (in kHz) from the end of the header.
		x, err = br.Read(8)
		if err != nil {
			return err
		}
		frame.SampleRate = uint32(x * 1000)
	case 0xD:
		// 1101: get 16 bit sample rate (in Hz) from the end of the header.
		x, err = br.Read(16)
		if err != nil {
			return err
		}
		frame.SampleRate = uint32(x)
	case 0xE:
		// 1110: get 16 bit sample rate (in daHz) from the end of the header.
		x, err = br.Read(16)
		if err != nil {
			return err
		}
		frame.SampleRate = uint32(x * 10)
	default:
		// 1111: invalid.
		return errors.New("frame.Frame.parseHeader: invalid sample rate bit pattern (1111)")
	}

	// 1 byte: CRC-8 checksum.
	var want uint8
	err = binary.Read(frame.hr, binary.BigEndian, &want)
	if err != nil {
		return err
	}
	got := h.Sum8()
	if got != want {
		return fmt.Errorf("frame.Frame.parseHeader: CRC-8 checksum mismatch; expected %v, got %v", want, got)
	}

	return nil
}

// Channels specifies the number of channels (subframes) that exist in a frame,
// their order and possible interchannel correlation.
type Channels uint8

// Channel assignments. The following abbreviations are used:
//    C:   center (directly in front)
//    R:   right (standard stereo)
//    Sr:  side right (directly to the right)
//    Rs:  right surround (back right)
//    Cs:  center surround (rear center)
//    Ls:  left surround (back left)
//    Sl:  side left (directly to the left)
//    L:   left (standard stereo)
//    Lfe: low-frequency effect (placed according to room acoustics)
//
// The first 6 channel constants follow the SMPTE/ITU-R channel order:
//    L R C Lfe Ls Rs
const (
	ChannelsMono           Channels = iota // 1 channel: mono
	ChannelsLR                             // 2 channels: left, right
	ChannelsLRC                            // 3 channels: left, right, center
	ChannelsLRLsRs                         // 4 channels: left, right, left surround, right surround
	ChannelsLRCLsRs                        // 5 channels: left, right, center, left surround, right surround
	ChannelsLRCLfeLsRs                     // 6 channels: left, right, center, LFE, left surround, right surround
	ChannelsLRCLfeCsSlSr                   // 7 channels: left, right, center, LFE, center surround, side left, side right
	ChannelsLRCLfeLsRsSlSr                 // 8 channels: left, right, center, LFE, left surround, right surround, side left, side right
	ChannelsLeftSide                       // 2 channels: left, side; using interchannel correlation
	ChannelsSideRight                      // 2 channels: side, right; using interchannel correlation
	ChannelsMidSide                        // 2 channels: mid, side; using interchannel correlation
)

// nChannels specifies the number of channels used by each channel assignment.
var nChannels = [...]int{
	ChannelsMono:           1,
	ChannelsLR:             2,
	ChannelsLRC:            3,
	ChannelsLRLsRs:         4,
	ChannelsLRCLsRs:        5,
	ChannelsLRCLfeLsRs:     6,
	ChannelsLRCLfeCsSlSr:   7,
	ChannelsLRCLfeLsRsSlSr: 8,
	ChannelsLeftSide:       2,
	ChannelsSideRight:      2,
	ChannelsMidSide:        2,
}

// Count returns the number of channels (subframes) used by the provided channel
// assignment.
func (channels Channels) Count() int {
	return nChannels[channels]
}
