package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/mewkiz/pkg/osutil"
	"github.com/mewkiz/pkg/pathutil"
	"github.com/pkg/errors"
)

func main() {
	// Parse command line arguments.
	var (
		// force overwrite FLAC file if already present.
		force bool
	)
	flag.BoolVar(&force, "f", false, "force overwrite")
	flag.Parse()
	for _, wavPath := range flag.Args() {
		if err := wav2flac(wavPath, force); err != nil {
			log.Fatalf("%+v", err)
		}
	}
}

func wav2flac(wavPath string, force bool) error {
	// Create WAV decoder.
	r, err := os.Open(wavPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer r.Close()
	dec := wav.NewDecoder(r)
	if !dec.IsValidFile() {
		return errors.Errorf("invalid WAV file %q", wavPath)
	}
	sampleRate, nchannels, bps := int(dec.SampleRate), int(dec.NumChans), int(dec.BitDepth)

	// Create FLAC encoder.
	flacPath := pathutil.TrimExt(wavPath) + ".flac"
	if !force && osutil.Exists(flacPath) {
		return errors.Errorf("FLAC file %q already present; use -f flag to force overwrite", flacPath)
	}
	w, err := os.Create(flacPath)
	if err != nil {
		return errors.WithStack(err)
	}
	info := &meta.StreamInfo{
		// Minimum block size (in samples) used in the stream; between 16 and
		// 65535 samples.
		BlockSizeMin: 16, // adjusted by encoder.
		// Maximum block size (in samples) used in the stream; between 16 and
		// 65535 samples.
		BlockSizeMax: 65535, // adjusted by encoder.
		// Minimum frame size in bytes; a 0 value implies unknown.
		//FrameSizeMin // set by encoder.
		// Maximum frame size in bytes; a 0 value implies unknown.
		//FrameSizeMax // set by encoder.
		// Sample rate in Hz; between 1 and 655350 Hz.
		SampleRate: uint32(sampleRate),
		// Number of channels; between 1 and 8 channels.
		NChannels: uint8(nchannels),
		// Sample size in bits-per-sample; between 4 and 32 bits.
		BitsPerSample: uint8(bps),
		// Total number of inter-channel samples in the stream. One second of
		// 44.1 KHz audio will have 44100 samples regardless of the number of
		// channels. A 0 value implies unknown.
		//NSamples // set by encoder.
		// MD5 checksum of the unencoded audio data.
		//MD5sum // set by encoder.
	}
	enc, err := flac.NewEncoder(w, info)
	if err != nil {
		return errors.WithStack(err)
	}
	defer enc.Close()

	// Encode samples.
	if err := dec.FwdToPCM(); err != nil {
		return errors.WithStack(err)
	}
	// Number of samples per channel and block.
	const nsamplesPerChannel = 16
	nsamplesPerBlock := nchannels * nsamplesPerChannel
	buf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: nchannels,
			SampleRate:  sampleRate,
		},
		Data:           make([]int, nsamplesPerBlock),
		SourceBitDepth: bps,
	}

	subframes := make([]*frame.Subframe, nchannels)
	for i := range subframes {
		subframe := &frame.Subframe{
			Samples: make([]int32, nsamplesPerChannel),
		}
		subframes[i] = subframe
	}
	for frameNum := 0; !dec.EOF(); frameNum++ {
		fmt.Println("frame number:", frameNum)
		// Decode WAV samples.
		n, err := dec.PCMBuffer(buf)
		if err != nil {
			return errors.WithStack(err)
		}
		if n == 0 {
			break
		}
		for _, subframe := range subframes {
			subHdr := frame.SubHeader{
				// Specifies the prediction method used to encode the audio sample of the
				// subframe.
				Pred: frame.PredVerbatim,
				// Prediction order used by fixed and FIR linear prediction decoding.
				Order: 0,
				// Wasted bits-per-sample.
				Wasted: 0,
			}
			subframe.SubHeader = subHdr
			subframe.NSamples = n / nchannels
			subframe.Samples = subframe.Samples[:subframe.NSamples]
		}
		for i, sample := range buf.Data {
			subframe := subframes[i%nchannels]
			subframe.Samples[i/nchannels] = int32(sample)
		}
		// Check if the subframe may be encoded as constant; when all samples are
		// the same.
		for _, subframe := range subframes {
			sample := subframe.Samples[0]
			constant := true
			for _, s := range subframe.Samples[1:] {
				if sample != s {
					constant = false
				}
			}
			if constant {
				fmt.Println("constant method")
				subframe.SubHeader.Pred = frame.PredConstant
			}
		}

		// Encode FLAC frame.
		channels, err := getChannels(nchannels)
		if err != nil {
			return errors.WithStack(err)
		}
		hdr := frame.Header{
			// Specifies if the block size is fixed or variable.
			HasFixedBlockSize: false,
			// Block size in inter-channel samples, i.e. the number of audio samples
			// in each subframe.
			BlockSize: uint16(nsamplesPerChannel),
			// Sample rate in Hz; a 0 value implies unknown, get sample rate from
			// StreamInfo.
			SampleRate: uint32(sampleRate),
			// Specifies the number of channels (subframes) that exist in the frame,
			// their order and possible inter-channel decorrelation.
			Channels: channels,
			// Sample size in bits-per-sample; a 0 value implies unknown, get sample
			// size from StreamInfo.
			BitsPerSample: uint8(bps),
			// Specifies the frame number if the block size is fixed, and the first
			// sample number in the frame otherwise. When using fixed block size, the
			// first sample number in the frame can be derived by multiplying the
			// frame number with the block size (in samples).
			//Num // set by encoder.
		}
		f := &frame.Frame{
			Header:    hdr,
			Subframes: subframes,
		}
		if err := enc.WriteFrame(f); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// getChannels returns the channels assignment matching the given number of
// channels.
func getChannels(nchannels int) (frame.Channels, error) {
	switch nchannels {
	case 1:
		// 1 channel: mono.
		return frame.ChannelsMono, nil
	case 2:
		// 2 channels: left, right.
		return frame.ChannelsLR, nil
		//return frame.ChannelsLeftSide, nil  // 2 channels: left, side; using inter-channel decorrelation.
		//return frame.ChannelsSideRight, nil // 2 channels: side, right; using inter-channel decorrelation.
		//return frame.ChannelsMidSide, nil   // 2 channels: mid, side; using inter-channel decorrelation.
	case 3:
		// 3 channels: left, right, center.
		return frame.ChannelsLRC, nil
	case 4:
		// 4 channels: left, right, left surround, right surround.
		return frame.ChannelsLRLsRs, nil
	case 5:
		// 5 channels: left, right, center, left surround, right surround.
		return frame.ChannelsLRCLsRs, nil
	case 6:
		// 6 channels: left, right, center, LFE, left surround, right surround.
		return frame.ChannelsLRCLfeLsRs, nil
	case 7:
		// 7 channels: left, right, center, LFE, center surround, side left, side right.
		return frame.ChannelsLRCLfeCsSlSr, nil
	case 8:
		// 8 channels: left, right, center, LFE, left surround, right surround, side left, side right.
		return frame.ChannelsLRCLfeLsRsSlSr, nil
	default:
		return 0, errors.Errorf("support for %d number of channels not yet implemented", nchannels)
	}
}
