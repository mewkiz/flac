package meta

import (
	"encoding/binary"
	"fmt"
	"io"
)

// A StreamInfo metadata block has information about the entire stream. The
// first metadata block in a FLAC stream must be a StreamInfo metadata block.
type StreamInfo struct {
	// The minimum block size (in samples) used in the stream.
	BlockSizeMin uint16
	// The maximum block size (in samples) used in the stream.
	// (BlockSizeMin == BlockSizeMax) implies a fixed-blocksize stream.
	BlockSizeMax uint16
	// The minimum frame size (in bytes) used in the stream. A value of 0 implies
	// that the minimum frame size is not known.
	FrameSizeMin uint32
	// The maximum frame size (in bytes) used in the stream. A value of 0 implies
	// that the maximum frame size is not known.
	FrameSizeMax uint32
	// Sample rate in Hz. Though 20 bits are available, the maximum sample rate
	// is limited by the structure of frame headers to 655350Hz. A value of 0 is
	// invalid.
	SampleRate uint32
	// Number of channels. FLAC supports from 1 to 8 channels.
	ChannelCount uint8
	// Bits per sample. FLAC supports from 4 to 32 bits per sample. Currently the
	// reference encoder and decoders only support up to 24 bits per sample.
	BitsPerSample uint8
	// Total number of samples in stream. This refers to inter-channel samples,
	// i.e. one second of 44.1Khz audio will have 44100 samples regardless of the
	// number of channels. A value of 0 implies that the number is channels is
	// not known.
	SampleCount uint64
	// MD5 signature of the unencoded audio data. This allows the decoder to
	// determine if an error exists in the audio data even when the error does
	// not result in an invalid bitstream.
	MD5sum [16]byte
}

// NewStreamInfo parses and returns a new StreamInfo metadata block. The
// provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
//
// Stream info format (pseudo code):
//
//    type METADATA_BLOCK_STREAMINFO struct {
//       block_size_min  uint16
//       block_size_max  uint16
//       frame_size_min  uint24
//       frame_size_max  uint24
//       sample_rate     uint20
//       channel_count   uint3 // (number of channels)-1.
//       bits_per_sample uint5 // (bits per sample)-1.
//       sample_count    uint36
//       md5sum          [16]byte
//    }
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_streaminfo
func NewStreamInfo(r io.Reader) (si *StreamInfo, err error) {
	// Minimum block size.
	si = new(StreamInfo)
	err = binary.Read(r, binary.BigEndian, &si.BlockSizeMin)
	if err != nil {
		return nil, err
	}
	if si.BlockSizeMin < 16 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid min block size; expected >= 16, got %d", si.BlockSizeMin)
	}

	const (
		blockSizeMaxMask = 0xFFFF000000000000 // 16 bits
		frameSizeMinMask = 0x0000FFFFFF000000 // 24 bits
		frameSizeMaxMask = 0x0000000000FFFFFF // 24 bits
	)
	// In order to keep everything on powers-of-2 boundaries, reads from the
	// block are grouped accordingly:
	// BlockSizeMax (16 bits) + FrameSizeMin (24 bits) + FrameSizeMax (24 bits) =
	// 64 bits
	var bits uint64
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Max block size.
	si.BlockSizeMax = uint16(bits & blockSizeMaxMask >> 48)
	if si.BlockSizeMax < 16 || si.BlockSizeMax > 65535 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid min block size; expected >= 16 and <= 65535, got %d", si.BlockSizeMax)
	}

	// Min frame size.
	si.FrameSizeMin = uint32(bits & frameSizeMinMask >> 24)

	// Max frame size.
	si.FrameSizeMax = uint32(bits & frameSizeMaxMask)

	const (
		sampleRateMask    = 0xFFFFF00000000000 // 20 bits
		channelCountMask  = 0x00000E0000000000 // 3 bits
		bitsPerSampleMask = 0x000001F000000000 // 5 bits
		sampleCountMask   = 0x0000000FFFFFFFFF // 36 bits
	)
	// In order to keep everything on powers-of-2 boundaries, reads from the
	// block are grouped accordingly:
	// SampleRate (20 bits) + ChannelCount (3 bits) + BitsPerSample (5 bits) +
	// SampleCount (36 bits) = 64 bits
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Sample rate.
	si.SampleRate = uint32(bits & sampleRateMask >> 44)
	if si.SampleRate > 655350 || si.SampleRate == 0 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid sample rate; expected > 0 and <= 655350, got %d", si.SampleRate)
	}

	// Both ChannelCount and BitsPerSample are specified to be subtracted by 1 in
	// the specification:
	// http://flac.sourceforge.net/format.html#metadata_block_streaminfo

	// Channel count.
	si.ChannelCount = uint8(bits&channelCountMask>>41) + 1
	if si.ChannelCount < 1 || si.ChannelCount > 8 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid number of channels; expected >= 1 and <= 8, got %d", si.ChannelCount)
	}

	// Bits per sample.
	si.BitsPerSample = uint8(bits&bitsPerSampleMask>>36) + 1
	if si.BitsPerSample < 4 || si.BitsPerSample > 32 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid number of bits per sample; expected >= 4 and <= 32, got %d", si.BitsPerSample)
	}

	// Sample count.
	si.SampleCount = bits & sampleCountMask

	// Md5sum MD5 signature of unencoded audio data.
	_, err = io.ReadFull(r, si.MD5sum[:])
	if err != nil {
		return nil, err
	}
	return si, nil
}
