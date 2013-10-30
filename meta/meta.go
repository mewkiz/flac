// Package meta contains functions for parsing FLAC metadata.
package meta

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/eaburns/bit"
)

// A Block is a metadata block, consisting of a block header and a block body.
type Block struct {
	// The underlying reader of the block.
	r io.ReadSeeker
	// Metadata block header.
	Header *BlockHeader
	// Metadata block body: *StreamInfo, *Application, *SeekTable, etc.
	Body interface{}
}

// ParseBlock reads from the provided io.ReadSeeker and returns a parsed
// metadata block. It parses both the header and the body of the metadata block.
// Use NewBlock instead for more granularity.
func ParseBlock(r io.ReadSeeker) (block *Block, err error) {
	block, err = NewBlock(r)
	if err != nil {
		return nil, err
	}

	err = block.Parse()
	if err != nil {
		return nil, err
	}

	return block, nil
}

// NewBlock reads and parses a metadata block header from the provided
// io.ReadSeeker and returns a handle to the metadata block. Call Parse to parse
// the metadata block body and Skip to ignore it.
func NewBlock(r io.ReadSeeker) (block *Block, err error) {
	// Read metadata block header.
	block = &Block{r: r}
	block.Header, err = NewBlockHeader(r)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// Parse reads and parses the metadata block body.
func (block *Block) Parse() (err error) {
	// Read metadata block.
	lr := io.LimitReader(block.r, int64(block.Header.Length))
	switch block.Header.BlockType {
	case TypeStreamInfo:
		block.Body, err = NewStreamInfo(lr)
	case TypePadding:
		err = VerifyPadding(lr)
	case TypeApplication:
		block.Body, err = NewApplication(lr)
	case TypeSeekTable:
		block.Body, err = NewSeekTable(lr)
	case TypeVorbisComment:
		block.Body, err = NewVorbisComment(lr)
	case TypeCueSheet:
		block.Body, err = NewCueSheet(lr)
	case TypePicture:
		block.Body, err = NewPicture(lr)
	default:
		return fmt.Errorf("meta.NewBlock: block type '%d' not yet supported", block.Header.BlockType)
	}
	if err != nil {
		return err
	}

	return nil
}

// Skip ignores the contents of the metadata block body.
func (block *Block) Skip() (err error) {
	_, err = block.r.Seek(int64(block.Header.Length), os.SEEK_CUR)
	if err != nil {
		return err
	}
	return nil
}

// BlockType is used to identify the metadata block type.
type BlockType uint8

// Metadata block types.
const (
	TypeStreamInfo BlockType = 1 << iota
	TypePadding
	TypeApplication
	TypeSeekTable
	TypeVorbisComment
	TypeCueSheet
	TypePicture

	// TypeAll is a bitmask of all block types.
	TypeAll = TypeStreamInfo | TypePadding | TypeApplication | TypeSeekTable | TypeVorbisComment | TypeCueSheet | TypePicture
)

// blockTypeName is a map from BlockType to name.
var blockTypeName = map[BlockType]string{
	TypeStreamInfo:    "stream info",
	TypePadding:       "padding",
	TypeApplication:   "application",
	TypeSeekTable:     "seek table",
	TypeVorbisComment: "vorbis comment",
	TypeCueSheet:      "cue sheet",
	TypePicture:       "picture",
}

func (t BlockType) String() string {
	return blockTypeName[t]
}

// A BlockHeader contains type and length information about a metadata block.
type BlockHeader struct {
	// IsLast is true if this block is the last metadata block before the audio
	// frames, and false otherwise.
	IsLast bool
	// Block type.
	BlockType BlockType
	// Length in bytes of the metadata body.
	Length int
}

// NewBlockHeader parses and returns a new metadata block header.
//
// Block header format (pseudo code):
//
//    type METADATA_BLOCK_HEADER struct {
//       is_last    bool
//       block_type uint7
//       length     uint24
//    }
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_header
func NewBlockHeader(r io.Reader) (h *BlockHeader, err error) {
	br := bit.NewReader(r)
	// is_last:    1 bit
	// block_type: 7 bits
	// length:     24 bits
	fields, err := br.ReadFields(1, 7, 24)
	if err != nil {
		return nil, err
	}

	// Is last.
	h = new(BlockHeader)
	if fields[0] != 0 {
		h.IsLast = true
	}

	// Block type.
	//    0:     Streaminfo
	//    1:     Padding
	//    2:     Application
	//    3:     Seektable
	//    4:     Vorbis_comment
	//    5:     Cuesheet
	//    6:     Picture
	//    7-126: reserved
	//    127:   invalid, to avoid confusion with a frame sync code
	blockType := fields[1]
	switch blockType {
	case 0:
		h.BlockType = TypeStreamInfo
	case 1:
		h.BlockType = TypePadding
	case 2:
		h.BlockType = TypeApplication
	case 3:
		h.BlockType = TypeSeekTable
	case 4:
		h.BlockType = TypeVorbisComment
	case 5:
		h.BlockType = TypeCueSheet
	case 6:
		h.BlockType = TypePicture
	default:
		if blockType >= 7 && blockType <= 126 {
			// block type 7-126: reserved.
			return nil, errors.New("meta.NewBlockHeader: reserved block type")
		} else if blockType == 127 {
			// block type 127: invalid.
			return nil, errors.New("meta.NewBlockHeader: invalid block type")
		}
	}

	// Length.
	// int won't overflow since the max value of Length is 0x00FFFFFF.
	h.Length = int(fields[2])
	return h, nil
}
