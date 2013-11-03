package meta

import (
	"fmt"
	"io"

	"github.com/eaburns/bit"
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
	br := bit.NewReader(r)
	// field 0: block_size_min  (16 bits)
	// field 1: block_size_max  (16 bits)
	// field 2: frame_size_min  (24 bits)
	// field 3: frame_size_max  (24 bits)
	// field 4: sample_rate     (20 bits)
	// field 5: channel_count   (3 bits)
	// field 6: bits_per_sample (5 bits)
	// field 7: sample_count    (36 bits)
	fields, err := br.ReadFields(16, 16, 24, 24, 20, 3, 5, 36)
	if err != nil {
		return nil, err
	}

	// Minimum block size.
	si = new(StreamInfo)
	si.BlockSizeMin = uint16(fields[0])
	if si.BlockSizeMin < 16 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid min block size; expected >= 16, got %d", si.BlockSizeMin)
	}

	// Maximum block size.
	si.BlockSizeMax = uint16(fields[1])
	if si.BlockSizeMax < 16 || si.BlockSizeMax > 65535 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid min block size; expected >= 16 and <= 65535, got %d", si.BlockSizeMax)
	}

	// Minimum frame size.
	si.FrameSizeMin = uint32(fields[2])

	// Maximum frame size.
	si.FrameSizeMax = uint32(fields[3])

	// Sample rate.
	si.SampleRate = uint32(fields[4])
	if si.SampleRate > 655350 || si.SampleRate == 0 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid sample rate; expected > 0 and <= 655350, got %d", si.SampleRate)
	}

	// According to the specification 1 should be added to both ChannelCount and
	// BitsPerSample:
	//
	// ref: http://flac.sourceforge.net/format.html#metadata_block_streaminfo

	// Channel count.
	si.ChannelCount = uint8(fields[5]) + 1
	if si.ChannelCount < 1 || si.ChannelCount > 8 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid number of channels; expected >= 1 and <= 8, got %d", si.ChannelCount)
	}

	// Bits per sample.
	si.BitsPerSample = uint8(fields[6]) + 1
	if si.BitsPerSample < 4 || si.BitsPerSample > 32 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid number of bits per sample; expected >= 4 and <= 32, got %d", si.BitsPerSample)
	}

	// Sample count.
	si.SampleCount = fields[7]

	// MD5 signature of the unencoded audio data.
	_, err = io.ReadFull(r, si.MD5sum[:])
	if err != nil {
		return nil, err
	}
	return si, nil
}
