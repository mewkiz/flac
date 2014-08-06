package meta

import (
	"encoding/binary"
	"errors"
)

// SeekTable contains one or more pre-calculated audio frame seek points.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_seektable
type SeekTable struct {
	// One or more seek points.
	Points []SeekPoint
}

// parseSeekTable reads and parses the body of an SeekTable metadata block.
func (block *Block) parseSeekTable() error {
	// The number of seek points is derived from the header length, divided by
	// the size of a SeekPoint; which is 18 bytes.
	n := block.Length / 18
	if n < 1 {
		return errors.New("meta.Block.parseSeekTable: at least one seek point is required")
	}
	table := &SeekTable{Points: make([]SeekPoint, n)}
	block.Body = table
	for i := range table.Points {
		err := binary.Read(block.lr, binary.LittleEndian, &table.Points[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// A SeekPoint specifies the byte offset and initial sample number of a given
// target frame.
//
// ref: https://www.xiph.org/flac/format.html#seekpoint
type SeekPoint struct {
	// Sample number of the first sample in the target frame, or
	// 0xFFFFFFFFFFFFFFFF for a placeholder point.
	SampleNum uint64
	// Offset in bytes from the first byte of the first frame header to the first
	// byte of the target frame's header.
	Offset uint64
	// Number of samples in the target frame.
	NSamples uint16
}
