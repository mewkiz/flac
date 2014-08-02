package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/mewkiz/flac"
)

func main() {
	f, err := os.Create("flac-frame.pprof")
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	err = pprof.StartCPUProfile(f)
	if err != nil {
		log.Println(err)
	}
	defer pprof.StopCPUProfile()

	flag.Parse()
	for _, filePath := range flag.Args() {
		err := flacFrame(filePath)
		if err != nil {
			log.Println(err)
		}
	}
}

func flacFrame(filePath string) (err error) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	br := bufio.NewReader(f)

	s, err := flac.NewStream(br)
	if err != nil {
		return err
	}
	err = s.ParseBlocks(0)
	if err != nil {
		return err
	}
	err = s.ParseFrames()
	if err != nil {
		return err
	}

	return nil
}
