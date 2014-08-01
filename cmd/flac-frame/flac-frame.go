package main

import (
	"flag"
	"log"

	"github.com/mewkiz/flac"
)

func main() {
	flag.Parse()
	for _, filePath := range flag.Args() {
		err := flacFrame(filePath)
		if err != nil {
			log.Println(err)
		}
	}
}

func flacFrame(filePath string) (err error) {
	_, err = flac.Parse(filePath)
	if err != nil {
		return err
	}
	return nil
}
