package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mewkiz/flac"
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
	enc, err := flac.NewEncoder(w, sampleRate, nchannels, bps)
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

	samples := make([][]int32, nchannels)
	for i := range samples {
		samples[i] = make([]int32, nsamplesPerChannel)
	}
	for j := 0; !dec.EOF(); j++ {
		n, err := dec.PCMBuffer(buf)
		if err != nil {
			return errors.WithStack(err)
		}
		if n == 0 {
			break
		}
		for i, sample := range buf.Data {
			samples[i%nchannels][i/nchannels] = int32(sample)
		}
		fmt.Println("j:", j)
		//pretty.Println("samples:", samples)
		if err := enc.Write(samples); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
