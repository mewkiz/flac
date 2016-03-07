// NOTE: This example is longer than needs to be when using Azul3d. The reason
// for this is to make the FLAC decoding explicit to showcase the low-level API,
// rather than using the front-end decoder implemented for Azul3d. An equivalent
// example using the Azul3d audio decoder interface for FLAC decoding may be
// viewed at github.com/azul3d/examples/azul3d_flac2wav.

// flac2wav is a tool which converts FLAC files to WAV files.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"azul3d.org/engine/audio"
	"azul3d.org/engine/audio/wav"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/pkg/osutil"
	"github.com/mewkiz/pkg/pathutil"
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
			log.Fatal(err)
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
			return fmt.Errorf("the file %q exists already", wavPath)
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
		samples := make(audio.Int16, 1)
		for i := 0; i < int(frame.BlockSize); i++ {
			for _, subframe := range frame.Subframes {
				samples[0] = int16(subframe.Samples[i])
				_, err = enc.Write(samples)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
