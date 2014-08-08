package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
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
	for _, path := range flag.Args() {
		err := flacFrame(path)
		if err != nil {
			log.Println(err)
		}
	}
}

func flacFrame(path string) error {
	stream, err := flac.ParseFile(path)
	if err != nil {
		return err
	}

	md5sum := md5.New()
	for {
		frame, err := stream.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		frame.Hash(md5sum)
	}
	fmt.Printf("original MD5: %032x\n", stream.Info.MD5sum[:])
	fmt.Printf("decoded MD5:  %032x\n", md5sum.Sum(nil))

	return nil
}
