// flac2wav is a tool which converts FLAC files to WAV files.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"azul3d.org/audio.v1"
	"github.com/mewkiz/audio/wav"
	"github.com/mewkiz/pkg/osutil"
	"github.com/mewkiz/pkg/pathutil"
	"gopkg.in/mewkiz/flac.v1"
)

// flagForce specifies if file overwriting should be forced, when a WAV file of
// the same name already exists.
var flagForce bool

func init() {
	flag.BoolVar(&flagForce, "f", false, "Force overwrite.")
}

func main() {
	flag.Parse()
	for _, path := range flag.Args() {
		err := flac2wav(path)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

// flac2wav converts the provided FLAC file to a WAV file.
func flac2wav(path string) error {
	// Open FLAC file.
	stream, err := flac.Open(path)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Create WAV file.
	wavPath := pathutil.TrimExt(path) + ".wav"
	if !flagForce {
		exists, err := osutil.Exists(wavPath)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("the file %q exists already.", wavPath)
		}
	}
	fw, err := os.Create(wavPath)
	if err != nil {
		return err
	}
	defer fw.Close()

	// Create WAV encoder.
	conf := audio.Config{
		Channels:   int(stream.Info.NChannels),
		SampleRate: int(stream.Info.SampleRate),
	}
	enc, err := wav.NewEncoder(fw, conf)
	if err != nil {
		return err
	}
	defer enc.Close()

	for {
		// Decode FLAC audio samples.
		frame, err := stream.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Encode WAV audio samples.
		samples := make(audio.PCM16Samples, 1)
		for i := 0; i < int(frame.BlockSize); i++ {
			for _, subframe := range frame.Subframes {
				samples[0] = audio.PCM16(subframe.Samples[i])
				_, err = enc.Write(samples)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
