// Package meta contains functions for parsing FLAC metadata.
package meta

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// A Block is a metadata block, consisting of a block header and a block body.
type Block struct {
	// Metadata block header.
	Header *BlockHeader
	// Metadata block body: *StreamInfo, *Application, *SeekTable, etc.
	Body interface{}
}

// NewBlock parses and returns a new metadata block, which consists of a block
// header and a block body.
func NewBlock(r io.Reader) (block *Block, err error) {
	// Read metadata block header.
	block = new(Block)
	block.Header, err = NewBlockHeader(r)
	if err != nil {
		return nil, err
	}

	// Read metadata block.
	lr := io.LimitReader(r, int64(block.Header.Length))
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
		return nil, fmt.Errorf("meta.NewBlock: block type '%d' not yet supported", block.Header.BlockType)
	}
	if err != nil {
		return nil, err
	}

	return block, nil
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
	const (
		isLastMask = 0x80000000 // 1 bit
		typeMask   = 0x7F000000 // 7 bits
		lengthMask = 0x00FFFFFF // 24 bits
	)
	var bits uint32
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Is last.
	h = new(BlockHeader)
	if bits&isLastMask != 0 {
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
	blockType := bits & typeMask >> 24
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
	h.Length = int(bits & lengthMask) // won't overflow, since max is 0x00FFFFFF.

	return h, nil
}
