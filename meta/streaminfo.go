package meta

import "crypto/md5"

// StreamInfo contains the basic properties of a FLAC audio stream, such as its
// sample rate and channel count. It is the only mandatory metadata block and
// must be present as the first metadata block of a FLAC stream.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_streaminfo
type StreamInfo struct {
	// Minimum block size (in samples) used in the stream.
	BlockSizeMin uint16
	// Maximum block size (in samples) used in the stream.
	BlockSizeMax uint16
	// Minimum frame size in bytes; a 0 value implies unknown.
	FrameSizeMin uint32
	// Maximum frame size in bytes; a 0 value implies unknown.
	FrameSizeMax uint32
	// Sample rate in Hz; between 1 and 655350 Hz.
	SampleRate uint32
	// Number of channels; between 1 and 8 channels.
	NChannels uint8
	// Sample size in bits-per-sample; between 4 and 32 bits.
	BitsPerSample uint8
	// Total number of inter-channel samples in the stream. One second of 44.1
	// KHz audio will have 44100 samples regardless of the number of channels. A
	// 0 value implies unknown.
	NSamples uint64
	// MD5 checksum of the unencoded audio data.
	MD5sum [md5.BlockSize]uint8
}

// parseStreamInfo reads and parses the body of an StreamInfo metadata block.
func (block *Block) parseStreamInfo() error {
	panic("not yet implemented.")
}
