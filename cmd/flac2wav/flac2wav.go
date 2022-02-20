// The flac2wav tool converts FLAC files to WAV files.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/pkg/osutil"
	"github.com/mewkiz/pkg/pathutil"
	"github.com/pkg/errors"
)

func usage() {
	const use = `
Usage: flac2wav [OPTION]... FILE.flac...`
	fmt.Fprintln(os.Stderr, use[1:])
	flag.PrintDefaults()
}

func main() {
	// Parse command line arguments.
	var (
		// force overwrite WAV file if present already.
		force bool
	)
	flag.BoolVar(&force, "f", false, "force overwrite")
	flag.Usage = usage
	flag.Parse()
	for _, path := range flag.Args() {
		err := flac2wav(path, force)
		if err != nil {
			log.Fatalf("%+v", err)
		}
	}
}

// flac2wav converts the provided FLAC file to a WAV file.
func flac2wav(path string, force bool) error {
	// Open FLAC file.
	stream, err := flac.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer stream.Close()

	// Create WAV file.
	wavPath := pathutil.TrimExt(path) + ".wav"
	if !force {
		if osutil.Exists(wavPath) {
			return errors.Errorf("WAV file %q already present; use the -f flag to force overwrite", wavPath)
		}
	}
	fw, err := os.Create(wavPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer fw.Close()

	// Create WAV encoder.
	wavAudioFormat := 1 // PCM
	enc := wav.NewEncoder(fw, int(stream.Info.SampleRate), int(stream.Info.BitsPerSample), int(stream.Info.NChannels), wavAudioFormat)
	defer enc.Close()
	var data []int
	for {
		// Decode FLAC audio samples.
		frame, err := stream.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.WithStack(err)
		}

		// Encode WAV audio samples.
		data = data[:0]
		for i := 0; i < frame.Subframes[0].NSamples; i++ {
			for _, subframe := range frame.Subframes {
				sample := int(subframe.Samples[i])
				if frame.BitsPerSample == 8 {
					// WAV files with 8 bit-per-sample are stored with unsigned
					// values, WAV files with more than 8 bit-per-sample are stored
					// as signed values (ref page 59-60 of [1]).
					//
					// [1]: http://www-mmsp.ece.mcgill.ca/Documents/AudioFormats/WAVE/Docs/riffmci.pdf
					// ref: https://github.com/mewkiz/flac/issues/51#issuecomment-1046183409
					const midpointValue = 0x80
					sample += midpointValue
				}
				data = append(data, sample)
			}
		}
		buf := &audio.IntBuffer{
			Format: &audio.Format{
				NumChannels: int(stream.Info.NChannels),
				SampleRate:  int(stream.Info.SampleRate),
			},
			Data:           data,
			SourceBitDepth: int(stream.Info.BitsPerSample),
		}
		if err := enc.Write(buf); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
