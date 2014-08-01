package main

import (
	"flag"
	dbg "fmt"
	"log"

	"github.com/kylelemons/godebug/pretty"
	"github.com/mewkiz/flac"
)

func main() {
	flag.Parse()
	for _, filePath := range flag.Args() {
		err := play(filePath)
		if err != nil {
			log.Println(err)
		}
		dbg.Println()
	}
}

func play(filePath string) (err error) {
	dbg.Println("path:", filePath)
	s, err := flac.Open(filePath)
	if err != nil {
		return err
	}
	for _, metaBlock := range s.MetaBlocks {
		dbg.Println("meta block:")
		pretty.Print(metaBlock)
	}
	for _, frame := range s.Frames {
		dbg.Println("frame:")
		pretty.Print(frame)
	}
	return nil
}
