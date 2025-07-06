package flac

import (
	"bytes"
	"math"
	"testing"

	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// BenchmarkEncodeSyntheticAudio measures the performance of encoding synthetic audio data.
// It creates a simple sine wave pattern to avoid dependency on external files.
func BenchmarkEncodeSyntheticAudio(b *testing.B) {
	// Create synthetic audio data (1 second of 44.1kHz stereo audio)
	sampleRate := 44100
	channels := 2
	bitsPerSample := 16
	numSamples := sampleRate

	// Create StreamInfo
	info := &meta.StreamInfo{
		BlockSizeMin:  4096,
		BlockSizeMax:  4096,
		FrameSizeMin:  0,
		FrameSizeMax:  0,
		SampleRate:    uint32(sampleRate),
		NChannels:     uint8(channels),
		BitsPerSample: uint8(bitsPerSample),
		NSamples:      uint64(numSamples),
	}

	// Generate synthetic audio data (sine wave)
	samples := make([]int32, numSamples*channels)
	freq := 440.0 // A4 note
	for i := 0; i < numSamples; i++ {
		// Generate a sine wave
		sample := int32(math.Sin(2*math.Pi*freq*float64(i)/float64(sampleRate)) * 32767)
		// Fill both channels with the same data
		samples[i*2] = sample
		samples[i*2+1] = sample
	}

	// Reset the timer before the actual benchmark
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Create a buffer to write the encoded data
		buf := &bytes.Buffer{}

		// Create encoder
		enc, err := NewEncoder(buf, info)
		if err != nil {
			b.Fatal(err)
		}

		// Process samples in blocks
		for offset := 0; offset < numSamples; offset += 4096 {
			blockSize := 4096
			if offset+blockSize > numSamples {
				blockSize = numSamples - offset
			}

			// Create frame
			fr := &frame.Frame{
				Header: frame.Header{
					HasFixedBlockSize: true,
					BlockSize:         uint16(blockSize),
					SampleRate:        uint32(sampleRate),
					Channels:          frame.ChannelsLR,
					BitsPerSample:     uint8(bitsPerSample),
				},
			}

			// Create subframes
			fr.Subframes = make([]*frame.Subframe, channels)
			for ch := 0; ch < channels; ch++ {
				// Extract samples for this channel
				channelSamples := make([]int32, blockSize)
				for j := 0; j < blockSize; j++ {
					channelSamples[j] = samples[(offset+j)*channels+ch]
				}

				// Create verbatim subframe since we're just testing encoding speed
				fr.Subframes[ch] = &frame.Subframe{
					SubHeader: frame.SubHeader{
						Pred: frame.PredVerbatim,
					},
					Samples:  channelSamples,
					NSamples: blockSize,
				}
			}

			// Encode frame
			if err := enc.WriteFrame(fr); err != nil {
				b.Fatal(err)
			}
		}

		// Close encoder
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
