package meta

import (
	"encoding/binary"
	"fmt"
	"io"
)

// A SeekTable metadata block is an optional block for storing seek points. It
// is possible to seek to any given sample in a FLAC stream without a seek
// table, but the delay can be unpredictable since the bitrate may vary widely
// within a stream. By adding seek points to a stream, this delay can be
// significantly reduced. Each seek point takes 18 bytes, so 1% resolution
// within a stream adds less than 2k.
//
// There can be only one SeekTable in a stream, but the table can have any
// number of seek points. There is also a special 'placeholder' seekpoint which
// will be ignored by decoders but which can be used to reserve space for future
// seek point insertion.
type SeekTable struct {
	// One or more seek points.
	Points []SeekPoint
}

// A SeekPoint specifies the offset of a sample.
type SeekPoint struct {
	// Sample number of first sample in the target frame, or 0xFFFFFFFFFFFFFFFF
	// for a placeholder point.
	SampleNum uint64
	// Offset (in bytes) from the first byte of the first frame header to the
	// first byte of the target frame's header.
	Offset uint64
	// Number of samples in the target frame.
	SampleCount uint16
}

// PlaceholderPoint is the sample number used for placeholder points. For
// placeholder points, the second and third field values in the SeekPoint
// structure are undefined.
const PlaceholderPoint = 0xFFFFFFFFFFFFFFFF

// NewSeekTable parses and returns a new SeekTable metadata block. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Seek table format (pseudo code):
//
//    type METADATA_BLOCK_SEEKTABLE struct {
//       // The number of seek points is implied by the metadata header 'length'
//       // field, i.e. equal to length / 18.
//       points []point
//    }
//
//    type point struct {
//       sample_num   uint64
//       offset       uint64
//       sample_count uint16
//    }
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_seektable
func NewSeekTable(r io.Reader) (st *SeekTable, err error) {
	st = new(SeekTable)
	var hasPrev bool
	var prevSampleNum uint64
	for {
		var point SeekPoint
		err = binary.Read(r, binary.BigEndian, &point)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if hasPrev && point.SampleNum != PlaceholderPoint {
			// - Seek points within a table must be sorted in ascending order by
			//   sample number.
			// - Seek points within a table must be unique by sample number, with
			//   the exception of placeholder points.
			// - The previous two notes imply that there may be any number of
			//   placeholder points, but they must all occur at the end of the
			//   table.
			if prevSampleNum == point.SampleNum {
				return nil, fmt.Errorf("meta.NewSeekTable: invalid seek point; sample number (%d) is not unique", point.SampleNum)
			} else if prevSampleNum > point.SampleNum {
				return nil, fmt.Errorf("meta.NewSeekTable: invalid seek point; sample number (%d) is not in ascending order", point.SampleNum)
			}
		}
		prevSampleNum = point.SampleNum
		hasPrev = true
		st.Points = append(st.Points, point)
	}
	return st, nil
}
