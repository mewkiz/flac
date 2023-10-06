package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	cmd "github.com/mewkiz/flac/cmd"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: [flac2wav|metaflac|wav2flac] [OPTION]... FILE...")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "flac2wav [OPTION]... FILE.flac...")
	fmt.Fprintln(os.Stderr, "  Convert FLAC files to WAV format.")
	fmt.Fprintln(os.Stderr, "  -f    Force overwrite of output files.")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "metaflac [OPTION]... FILE.flac...")
	fmt.Fprintln(os.Stderr, "  List metadata of FLAC files.")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "wav2flac [OPTION]... FILE.wav...")
	fmt.Fprintln(os.Stderr, "  Convert WAV files to FLAC format.")
	fmt.Fprintln(os.Stderr, "  -f    Force overwrite of output files.")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
}

func checkArgs() {
	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	} else if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}
}

func main() {
	var (
		force bool
	)

	flag.BoolVar(&force, "f", false, "force overwrite")
	flag.Usage = usage
	flag.Parse()
	checkArgs()

	command := os.Args[1]

	if command == "" {
		log.Fatalln("No command specified")
	}

	os.Args = append(os.Args[:1], os.Args[2:]...)
	flag.CommandLine.Parse(os.Args[1:])

	switch command {
	case "flac2wav":
		for _, path := range flag.Args() {
			err := cmd.Flac2wav(path, force)
			if err != nil {
				log.Fatalf("%+v", err)
			}
		}

	case "metaflac":
		for _, path := range flag.Args() {
			err := cmd.List(path)
			if err != nil {
				log.Fatalln(err)
			}
		}

	case "wav2flac":
		for _, path := range flag.Args() {
			if err := cmd.Wav2flac(path, force); err != nil {
				log.Fatalf("%+v", err)
			}
		}

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
