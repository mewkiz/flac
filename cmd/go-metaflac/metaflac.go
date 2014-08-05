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

// flagList states if the list operation should be performed, which lists the
// content of one or more metadata blocks to stdout.
var flagList bool

// flagBlockNum contains an optional comma-separated list of block numbers to
// display, which can be used in conjunction with flagList.
var flagBlockNum string

func init() {
	flag.BoolVar(&flagList, "list", false, "List the contents of one or more metadata blocks to stdout.")
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
	for _, filePath := range flag.Args() {
		err := metaflac(filePath)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func metaflac(filePath string) (err error) {
	if flagList {
		err = list(filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func list(filePath string) (err error) {
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
	s, err := flac.Open(filePath)
	if err != nil {
		return err
	}
	err = s.ParseBlocks(meta.TypeAll)
	if err != nil {
		return err
	}

	if blockNums != nil {
		// Only list blocks specified in the "--block-number" command line flag.
		for _, blockNum := range blockNums {
			if blockNum < len(s.MetaBlocks) {
				listBlock(s.MetaBlocks[blockNum], blockNum)
			}
		}
	} else {
		// List all blocks.
		for blockNum, block := range s.MetaBlocks {
			listBlock(block, blockNum)
		}
	}

	return nil
}

func listBlock(block *meta.Block, blockNum int) {
	listHeader(block.Header, blockNum)
	switch body := block.Body.(type) {
	case *meta.StreamInfo:
		listStreamInfo(body)
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

// Example:
//    METADATA block #0
//      type: 0 (STREAMINFO)
//      is last: false
//      length: 34
func listHeader(header *meta.BlockHeader, blockNum int) {
	var blockTypeName = map[meta.BlockType]string{
		meta.TypeStreamInfo:    "STREAMINFO",
		meta.TypePadding:       "PADDING",
		meta.TypeApplication:   "APPLICATION",
		meta.TypeSeekTable:     "SEEKTABLE",
		meta.TypeVorbisComment: "VORBIS_COMMENT",
		meta.TypeCueSheet:      "CUESHEET",
		meta.TypePicture:       "PICTURE",
	}
	fmt.Printf("METADATA block #%d\n", blockNum)
	fmt.Printf("  type: %d (%s)\n", header.BlockType, blockTypeName[header.BlockType])
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
	fmt.Printf("  channels: %d\n", si.ChannelCount)
	fmt.Printf("  bits-per-sample: %d\n", si.BitsPerSample)
	fmt.Printf("  total samples: %d\n", si.SampleCount)
	fmt.Printf("  MD5 signature: %x\n", si.MD5sum)
}

// Example:
//      application ID: 46696361
//      data contents:
//    Medieval CUE Splitter (www.medieval.it)
func listApplication(app *meta.Application) {
	fmt.Printf("  application ID: %x\n", app.ID)
	fmt.Println("  data contents:")
	fmt.Println(string(app.Data))
}

// Example:
//      seek points: 17
//        point 0: sample_number=0, stream_offset=0, frame_samples=4608
//        point 1: sample_number=2419200, stream_offset=3733871, frame_samples=4608
//        ...
func listSeekTable(st *meta.SeekTable) {
	fmt.Printf("  seek points: %d\n", len(st.Points))
	for pointNum, point := range st.Points {
		fmt.Printf("    point %d: sample_number=%d, stream_offset=%d, frame_samples=%d\n", pointNum, point.SampleNum, point.Offset, point.SampleCount)
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
	fmt.Printf("  comments: %d\n", len(vc.Entries))
	for entryNum, entry := range vc.Entries {
		fmt.Printf("    comment[%d]: %s=%s\n", entryNum, entry.Name, entry.Value)
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
	fmt.Printf("  lead-in: %d\n", cs.LeadInSampleCount)
	fmt.Printf("  is CD: %t\n", cs.IsCompactDisc)
	fmt.Printf("  number of tracks: %d\n", cs.TrackCount)
	for trackNum, track := range cs.Tracks {
		fmt.Printf("    track[%d]\n", trackNum)
		fmt.Printf("      offset: %d\n", track.Offset)
		if trackNum == len(cs.Tracks)-1 {
			// Lead-out track.
			fmt.Printf("      number: %d (LEAD-OUT)\n", track.TrackNum)
			continue
		}
		fmt.Printf("      number: %d\n", track.TrackNum)
		fmt.Printf("      ISRC: %s\n", track.ISRC)
		var trackTypeName = map[bool]string{
			false: "DATA",
			true:  "AUDIO",
		}
		fmt.Printf("      type: %s\n", trackTypeName[track.IsAudio])
		fmt.Printf("      pre-emphasis: %t\n", track.HasPreEmphasis)
		fmt.Printf("      number of index points: %d\n", track.TrackIndexCount)
		for indexNum, index := range track.TrackIndexes {
			fmt.Printf("        index[%d]\n", indexNum)
			fmt.Printf("          offset: %d\n", index.Offset)
			fmt.Printf("          number: %d\n", index.IndexPointNum)
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
	fmt.Printf("  MIME type: %s", pic.MIME)
	fmt.Printf("  description: %s\n", pic.Desc)
	fmt.Printf("  width: %d\n", pic.Width)
	fmt.Printf("  height: %d\n", pic.Height)
	fmt.Printf("  depth: %d\n", pic.ColorDepth)
	fmt.Printf("  colors: %d", pic.ColorCount)
	if pic.ColorCount == 0 {
		fmt.Print("(unindexed)")
	}
	fmt.Println()
	fmt.Printf("  data length: %d\n", len(pic.Data))
	fmt.Printf("  data:\n")
	fmt.Print(hex.Dump(pic.Data))
}
