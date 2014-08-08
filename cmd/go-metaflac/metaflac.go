// TODO(u): Prefix lines with file name if flag.NArg() > 1. Example:
//    tone24bit.flac:METADATA block #2
//    tone24bit.flac:  type: 4 (VORBIS_COMMENT)
//    tone24bit.flac:  is last: false
//    tone24bit.flac:  length: 40
//    tone24bit.flac:  vendor string: reference libFLAC 1.1.4 20070213
//    tone24bit.flac:  comments: 0
//    tone24bit.flac:METADATA block #3
//    tone24bit.flac:  type: 1 (PADDING)
//    tone24bit.flac:  is last: true
//    tone24bit.flac:  length: 8192
//    zonophone-x28010-10407u.flac:METADATA block #0
//    zonophone-x28010-10407u.flac:  type: 0 (STREAMINFO)
//    zonophone-x28010-10407u.flac:  is last: false
//    zonophone-x28010-10407u.flac:  length: 34

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

// flagBlockNum contains an optional comma-separated list of block numbers to
// display.
var flagBlockNum string

func init() {
	flag.StringVar(&flagBlockNum, "block-number", "", "An optional comma-separated list of block numbers to display.")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: metaflac [OPTION]... FILE...")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	for _, path := range flag.Args() {
		err := metaflac(path)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func metaflac(path string) (err error) {
	err = list(path)
	if err != nil {
		return err
	}
	return nil
}

func list(path string) (err error) {
	var blockNums []int
	if flagBlockNum != "" {
		// Parse "--block-number" command line flag.
		rawBlockNums := strings.Split(flagBlockNum, ",")
		for _, rawBlockNum := range rawBlockNums {
			blockNum, err := strconv.Atoi(rawBlockNum)
			if err != nil {
				return err
			}
			blockNums = append(blockNums, blockNum)
		}
	}

	// Open FLAC stream.
	stream, err := flac.ParseFile(path)
	if err != nil {
		return err
	}

	if blockNums != nil {
		// Only list blocks specified in the "--block-number" command line flag.
		for _, blockNum := range blockNums {
			if blockNum == 0 {
				listStreamInfo(stream.Info)
			} else {
				// strea.Blocks doesn't contain StreamInfo, therefore the blockNum
				// is one less.
				blockNum--
			}
			if blockNum < len(stream.Blocks) {
				listBlock(stream.Blocks[blockNum], blockNum)
			}
		}
	} else {
		// List all blocks.
		var isLast bool
		if len(stream.Blocks) == 0 {
			isLast = true
		}
		listStreamInfoHeader(isLast)
		listStreamInfo(stream.Info)
		for blockNum, block := range stream.Blocks {
			// strea.Blocks doesn't contain StreamInfo, therefore the blockNum
			// is one less.
			blockNum--
			listBlock(block, blockNum)
		}
	}

	return nil
}

func listBlock(block *meta.Block, blockNum int) {
	listHeader(&block.Header, blockNum)
	switch body := block.Body.(type) {
	case *meta.Application:
		listApplication(body)
	case *meta.SeekTable:
		listSeekTable(body)
	case *meta.VorbisComment:
		listVorbisComment(body)
	case *meta.CueSheet:
		listCueSheet(body)
	case *meta.Picture:
		listPicture(body)
	}
}

// typeName maps from metadata block type to a string version of its name.
var typeName = map[meta.Type]string{
	meta.TypeStreamInfo:    "STREAMINFO",
	meta.TypePadding:       "PADDING",
	meta.TypeApplication:   "APPLICATION",
	meta.TypeSeekTable:     "SEEKTABLE",
	meta.TypeVorbisComment: "VORBIS_COMMENT",
	meta.TypeCueSheet:      "CUESHEET",
	meta.TypePicture:       "PICTURE",
}

// Each field of the StreamInfo header is constant, with the exception of
// is_last.
//
// Example:
//    METADATA block #0
//      type: 0 (STREAMINFO)
//      is last: false
//      length: 34
func listStreamInfoHeader(isLast bool) {
	fmt.Println("METADATA block #0")
	fmt.Println("  type: 0 (STREAMINFO)")
	fmt.Println("  is last:", isLast)
	fmt.Println("  length: 34")
}

// Example:
//    METADATA block #0
//      type: 0 (STREAMINFO)
//      is last: false
//      length: 34
func listHeader(header *meta.Header, blockNum int) {
	name, ok := typeName[header.Type]
	if !ok {
		name = "UNKNOWN"
	}
	fmt.Printf("METADATA block #%d\n", blockNum)
	fmt.Printf("  type: %d (%s)\n", header.Type, name)
	fmt.Printf("  is last: %t\n", header.IsLast)
	fmt.Printf("  length: %d\n", header.Length)
}

// Example:
//      minimum blocksize: 4608 samples
//      maximum blocksize: 4608 samples
//      minimum framesize: 0 bytes
//      maximum framesize: 19024 bytes
//      sample_rate: 44100 Hz
//      channels: 2
//      bits-per-sample: 16
//      total samples: 151007220
//      MD5 signature: 2e6238f5d9fe5c19f3ead628f750fd3d
func listStreamInfo(si *meta.StreamInfo) {
	fmt.Printf("  minimum blocksize: %d samples\n", si.BlockSizeMin)
	fmt.Printf("  maximum blocksize: %d samples\n", si.BlockSizeMax)
	fmt.Printf("  minimum framesize: %d bytes\n", si.FrameSizeMin)
	fmt.Printf("  maximum framesize: %d bytes\n", si.FrameSizeMax)
	fmt.Printf("  sample_rate: %d Hz\n", si.SampleRate)
	fmt.Printf("  channels: %d\n", si.NChannels)
	fmt.Printf("  bits-per-sample: %d\n", si.BitsPerSample)
	fmt.Printf("  total samples: %d\n", si.NSamples)
	fmt.Printf("  MD5 signature: %x\n", si.MD5sum)
}

// Example:
//      application ID: 46696361
//      data contents:
//    Medieval CUE Splitter (www.medieval.it)
func listApplication(app *meta.Application) {
	fmt.Printf("  application ID: %x\n", string(app.ID))
	fmt.Println("  data contents:")
	if len(app.Data) > 0 {
		fmt.Println(string(app.Data))
	}
}

// Example:
//      seek points: 17
//        point 0: sample_number=0, stream_offset=0, frame_samples=4608
//        point 1: sample_number=2419200, stream_offset=3733871, frame_samples=4608
//        ...
func listSeekTable(st *meta.SeekTable) {
	fmt.Printf("  seek points: %d\n", len(st.Points))
	for pointNum, point := range st.Points {
		if point.SampleNum == meta.PlaceholderPoint {
			fmt.Printf("    point %d: PLACEHOLDER\n", pointNum)
		} else {
			fmt.Printf("    point %d: sample_number=%d, stream_offset=%d, frame_samples=%d\n", pointNum, point.SampleNum, point.Offset, point.NSamples)
		}
	}
}

// Example:
//      vendor string: reference libFLAC 1.2.1 20070917
//      comments: 10
//        comment[0]: ALBUM=「sugar sweet nightmare」 & 「化物語」劇伴音楽集 其の壹
//        comment[1]: ARTIST=神前暁
//        ...
func listVorbisComment(vc *meta.VorbisComment) {
	fmt.Printf("  vendor string: %s\n", vc.Vendor)
	fmt.Printf("  comments: %d\n", len(vc.Tags))
	for tagNum, tag := range vc.Tags {
		fmt.Printf("    comment[%d]: %s=%s\n", tagNum, tag[0], tag[1])
	}
}

// Example:
//      media catalog number:
//      lead-in: 88200
//      is CD: true
//      number of tracks: 18
//        track[0]
//          offset: 0
//          number: 1
//          ISRC:
//          type: AUDIO
//          pre-emphasis: false
//          number of index points: 1
//            index[0]
//              offset: 0
//              number: 1
//        track[1]
//          offset: 2421384
//          number: 2
//          ISRC:
//          type: AUDIO
//          pre-emphasis: false
//          number of index points: 1
//            index[0]
//              offset: 0
//              number: 1
//        ...
//        track[17]
//          offset: 151007220
//          number: 170 (LEAD-OUT)
func listCueSheet(cs *meta.CueSheet) {
	fmt.Printf("  media catalog number: %s\n", cs.MCN)
	fmt.Printf("  lead-in: %d\n", cs.NLeadInSamples)
	fmt.Printf("  is CD: %t\n", cs.IsCompactDisc)
	fmt.Printf("  number of tracks: %d\n", len(cs.Tracks))
	for trackNum, track := range cs.Tracks {
		fmt.Printf("    track[%d]\n", trackNum)
		fmt.Printf("      offset: %d\n", track.Offset)
		if trackNum == len(cs.Tracks)-1 {
			// Lead-out track.
			fmt.Printf("      number: %d (LEAD-OUT)\n", track.Num)
			continue
		}
		fmt.Printf("      number: %d\n", track.Num)
		fmt.Printf("      ISRC: %s\n", track.ISRC)
		var trackTypeName = map[bool]string{
			false: "DATA",
			true:  "AUDIO",
		}
		fmt.Printf("      type: %s\n", trackTypeName[track.IsAudio])
		fmt.Printf("      pre-emphasis: %t\n", track.HasPreEmphasis)
		fmt.Printf("      number of index points: %d\n", len(track.Indicies))
		for indexNum, index := range track.Indicies {
			fmt.Printf("        index[%d]\n", indexNum)
			fmt.Printf("          offset: %d\n", index.Offset)
			fmt.Printf("          number: %d\n", index.Num)
		}
	}
}

// Example:
//      type: 3 (Cover (front))
//      MIME type: image/jpeg
//      description:
//      width: 0
//      height: 0
//      depth: 0
//      colors: 0 (unindexed)
//      data length: 234569
//      data:
//        00000000: FF D8 FF E0 00 10 4A 46 49 46 00 01 01 01 00 60 ......JFIF.....`
//        00000010: 00 60 00 00 FF DB 00 43 00 01 01 01 01 01 01 01 .`.....C........
func listPicture(pic *meta.Picture) {
	typeName := map[uint32]string{
		0:  "Other",
		1:  "32x32 pixels 'file icon' (PNG only)",
		2:  "Other file icon",
		3:  "Cover (front)",
		4:  "Cover (back)",
		5:  "Leaflet page",
		6:  "Media (e.g. label side of CD)",
		7:  "Lead artist/lead performer/soloist",
		8:  "Artist/performer",
		9:  "Conductor",
		10: "Band/Orchestra",
		11: "Composer",
		12: "Lyricist/text writer",
		13: "Recording Location",
		14: "During recording",
		15: "During performance",
		16: "Movie/video screen capture",
		17: "A bright coloured fish",
		18: "Illustration",
		19: "Band/artist logotype",
		20: "Publisher/Studio logotype",
	}
	fmt.Printf("  type: %d (%s)\n", pic.Type, typeName[pic.Type])
	fmt.Printf("  MIME type: %s\n", pic.MIME)
	fmt.Printf("  description: %s\n", pic.Desc)
	fmt.Printf("  width: %d\n", pic.Width)
	fmt.Printf("  height: %d\n", pic.Height)
	fmt.Printf("  depth: %d\n", pic.Depth)
	fmt.Printf("  colors: %d", pic.NPalColors)
	if pic.NPalColors == 0 {
		fmt.Print(" (unindexed)")
	}
	fmt.Println()
	fmt.Printf("  data length: %d\n", len(pic.Data))
	fmt.Printf("  data:\n")
	fmt.Print(hex.Dump(pic.Data))
}
