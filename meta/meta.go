// Package meta implements access to FLAC metadata.
package meta

import (
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/mewkiz/pkg/bit"
)

// A Block contains the header and body of a metadata block.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block
type Block struct {
	// Metadata block header.
	Header
	// Metadata block body of type *StreamInfo, *Application, ... etc. Body is
	// initially nil, and gets populated by a call to Block.Parse.
	Body interface{}
	// Underlying io.Reader; limited by the length of the block body.
	lr io.Reader
}

// New creates a new Block for accessing the metadata of r. It reads and parses
// a metadata block header. Call Block.Parse to parse the metadata block body,
// and call Block.Skip to ignore it.
func New(r io.Reader) (block *Block, err error) {
	block = new(Block)
	err = block.parseHeader(r)
	if err != nil {
		return nil, err
	}
	block.lr = io.LimitReader(r, block.Length)
	return block, nil
}

// Parse reads and parses the header and body of a metadata block. Use New for
// additional granularity.
func Parse(r io.Reader) (block *Block, err error) {
	block, err = New(r)
	if err != nil {
		return nil, err
	}
	err = block.Parse()
	if err != nil {
		return nil, err
	}
	return block, nil
}

// Errors returned by Parse.
var (
	ErrReserved = errors.New("meta.Block.Parse: reserved block type")
	ErrInvalid  = errors.New("meta.Block.Parse: invalid block type")
)

// Parse reads and parses the metadata block body.
func (block *Block) Parse() error {
	switch block.Type {
	case TypeStreamInfo:
		return block.parseStreamInfo()
	case TypePadding:
		return block.verifyPadding()
	case TypeApplication:
		return block.parseApplication()
	case TypeSeekTable:
		return block.parseSeekTable()
	case TypeVorbisComment:
		return block.parseVorbisComment()
	case TypeCueSheet:
		return block.parseCueSheet()
	case TypePicture:
		return block.parsePicture()
	}
	if block.Type >= 7 && block.Type <= 126 {
		return ErrReserved
	}
	return ErrInvalid
}

// Skip ignores the contents of the metadata block body.
func (block *Block) Skip() error {
	if sr, ok := block.lr.(io.Seeker); ok {
		_, err := sr.Seek(0, os.SEEK_END)
		return err
	}
	_, err := io.Copy(ioutil.Discard, block.lr)
	return err
}

// A Header contains information about the type and length of a metadata block.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_header
type Header struct {
	// Metadata block body type.
	Type Type
	// Length of body data in bytes.
	Length int64
	// IsLast specifies if the block is the last metadata block.
	IsLast bool
}

// parseHeader reads and parses the header of a metadata block.
func (block *Block) parseHeader(r io.Reader) error {
	// 1 bit: IsLast.
	br := bit.NewReader(r)
	x, err := br.Read(1)
	if err != nil {
		return err
	}
	if x != 0 {
		block.IsLast = true
	}

	// 7 bits: Type.
	x, err = br.Read(1)
	if err != nil {
		return err
	}
	block.Type = Type(x)

	// 24 bits: Length.
	x, err = br.Read(1)
	if err != nil {
		return err
	}
	block.Length = int64(x)

	return nil
}

// Type represents the type of a metadata block body.
type Type uint8

// Metadata block body types.
const (
	TypeStreamInfo Type = iota
	TypePadding
	TypeApplication
	TypeSeekTable
	TypeVorbisComment
	TypeCueSheet
	TypePicture
)
