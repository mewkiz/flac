package meta

import (
	"fmt"
	"strings"

	"github.com/mewkiz/pkg/bit"
)

// VorbisComment contains a list of name-value pairs.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_vorbis_comment
type VorbisComment struct {
	// Vendor name.
	Vendor string
	// A list of tags, each represented by a name-value pair.
	Tags [][2]string
}

// parseVorbisComment reads and parses the body of an VorbisComment metadata
// block.
func (block *Block) parseVorbisComment() error {
	// 32 bits: vendor length.
	br := bit.NewReader(block.lr)
	x, err := br.Read(32)
	if err != nil {
		return err
	}

	// (vendor length) bits: Vendor.
	buf, err := readBytes(block.lr, int(x))
	if err != nil {
		return err
	}
	comment := new(VorbisComment)
	block.Body = comment
	comment.Vendor = string(buf)

	// 32 bits: number of tags.
	x, err = br.Read(32)
	if err != nil {
		return err
	}
	comment.Tags = make([][2]string, x)

	for i := range comment.Tags {
		// 32 bits: vector length
		x, err = br.Read(32)
		if err != nil {
			return err
		}

		// (vector length): vector.
		buf, err := readBytes(block.lr, int(x))
		if err != nil {
			return err
		}
		vector := string(buf)

		// Parse tag, which has the following format:
		//    NAME=VALUE
		pos := strings.Index(vector, "=")
		if pos == -1 {
			return fmt.Errorf("meta.Block.parseVorbisComment: unable to locate '=' in vector %q", vector)
		}
		comment.Tags[i][0] = vector[:pos]
		comment.Tags[i][1] = vector[pos+1:]
	}

	return nil
}
