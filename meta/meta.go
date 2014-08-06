// Package meta implements access to FLAC metadata.
package meta

import "io"

// A Block contains the header and body of a metadata block.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block
type Block struct {
	// Metadata block header.
	Header
	// Metadata block body of type *StreamInfo, *Application, ... etc. It is
	// initially nil, and gets populated by a call to Parse.
	Body interface{}
	// Underlying io.Reader.
	r io.Reader
}

// New creates a new Block for accessing the metadata of r. It reads and parses
// a metadata block header. Call Block.Parse to parse the metadata block body,
// and call Block.Skip to ignore it.
func New(r io.Reader) (block *Block, err error) {
	block = &Block{r: r}
	err = block.parseHeader()
	if err != nil {
		return nil, err
	}
	panic("not yet implemented.")
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

// Parse reads and parses the metadata block body.
func (block *Block) Parse() error {
	panic("not yet implemented.")
}

// Skip ignores the contents of the metadata block body.
func (block *Block) Skip() error {
	panic("not yet implemented.")
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
func (block *Block) parseHeader() error {
	panic("not yet implemented.")
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
