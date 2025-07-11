package flac

import (
	"bytes"
	"math"
	"testing"

	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// BenchmarkEncodeSyntheticAudio measures the performance of encoding synthetic
// audio data. It creates a simple sine wave pattern to avoid dependency on
// external files.
func BenchmarkEncodeSyntheticAudio(b *testing.B) {
	// Create synthetic audio data (1 second of 44.1kHz stereo audio)
	const (
		sampleRate    = 44100
		nchannels     = 2
		bitsPerSample = 16
		nsamples      = sampleRate
	)

	// Create StreamInfo
	info := &meta.StreamInfo{
		BlockSizeMin:  4096,
		BlockSizeMax:  4096,
		FrameSizeMin:  0,
		FrameSizeMax:  0,
		SampleRate:    sampleRate,
		NChannels:     nchannels,
		BitsPerSample: bitsPerSample,
		NSamples:      nsamples,
	}

	// Generate synthetic audio data (sine wave)
	samples := make([]int32, nsamples*nchannels)
	freq := 440.0 // A4 note
	for i := 0; i < nsamples; i++ {
		// Generate a sine wave
		sample := int32(math.Sin(2*math.Pi*freq*float64(i)/float64(sampleRate)) * 32767)
		// Fill both channels with the same data
		samples[i*2] = sample
		samples[i*2+1] = sample
	}

	// Reset the timer before the actual benchmark
	b.ResetTimer()

	// Run the benchmark
	for range b.N {
		// Create a buffer to write the encoded data
		buf := &bytes.Buffer{}

		// Create encoder
		enc, err := NewEncoder(buf, info)
		if err != nil {
			b.Fatal(err)
		}

		// Process samples in blocks
		for offset := 0; offset < nsamples; offset += 4096 {
			blockSize := 4096
			if offset+blockSize > nsamples {
				blockSize = nsamples - offset
			}

			// Create frame
			f := &frame.Frame{
				Header: frame.Header{
					HasFixedBlockSize: true,
					BlockSize:         uint16(blockSize),
					SampleRate:        sampleRate,
					Channels:          frame.ChannelsLR,
					BitsPerSample:     bitsPerSample,
				},
			}

			// Create subframes
			f.Subframes = make([]*frame.Subframe, nchannels)
			for channel := 0; channel < nchannels; channel++ {
				// Extract samples for this channel
				channelSamples := make([]int32, blockSize)
				for i := 0; i < blockSize; i++ {
					channelSamples[i] = samples[(offset+i)*nchannels+channel]
				}

				// Create verbatim subframe since we're just testing encoding speed
				f.Subframes[channel] = &frame.Subframe{
					SubHeader: frame.SubHeader{
						Pred: frame.PredVerbatim,
					},
					Samples:  channelSamples,
					NSamples: blockSize,
				}
			}

			// Encode frame
			if err := enc.WriteFrame(f); err != nil {
				b.Fatal(err)
			}
		}

		// Close encoder
		if err := enc.Close(); err != nil {
			b.Fatal(err)
		}
	}
}
