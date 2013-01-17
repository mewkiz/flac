// Package frame contains functions for parsing FLAC encoded audio data.
package frame

import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "os"

import "github.com/mewkiz/pkg/bit"
import "github.com/mewkiz/pkg/hashutil/crc16"

// A Frame is an audio frame, consisting of a frame header and one subframe per
// channel.
type Frame struct {
	// Audio frame header.
	Header *Header
	// Audio subframes, one per channel.
	SubFrames []*SubFrame
}

// NewFrame parses and returns a new frame, which consists of a frame header and
// one subframe per channel.
//
// Frame format (pseudo code):
//
//    type FRAME struct {
//       header    FRAME_HEADER
//       subframes []SUBFRAME
//       _         uint0 to uint7 // zero-padding to byte alignment.
//       footer    uint16 // CRC-16 of the entire frame, excluding the footer.
//    }
//
// ref: http://flac.sourceforge.net/format.html#frame
func NewFrame(r io.ReadSeeker) (frame *Frame, err error) {
	// Record start offset, which is used when verifying the CRC-16 of the frame.
	start, err := r.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	// Frame header.
	frame = new(Frame)
	frame.Header, err = NewHeader(r)
	if err != nil {
		return nil, err
	}

	// Subframes.
	br := bit.NewReader(r)
	h := frame.Header
	for i := 0; i < h.ChannelOrder.ChannelCount(); i++ {
		subframe, err := h.NewSubFrame(br)
		if err != nil {
			return nil, err
		}
		frame.SubFrames = append(frame.SubFrames, subframe)
	}

	// Padding.
	bitOff, err := br.Seek(0, bit.SeekCur)
	if err != nil {
		return nil, err
	}
	padBitCount := bitOff % 8
	if padBitCount != 0 {
		pad, err := br.Read(int(padBitCount))
		if err != nil {
			return nil, err
		}
		if pad.Uint64() != 0 {
			return nil, errors.New("frame.NewFrame: invalid padding; must be 0.")
		}
	}

	// Frame footer.

	// Read the frame data and calculate the CRC-16.
	end, err := r.Seek(0, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}
	_, err = r.Seek(start, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	data := make([]byte, end-start)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}

	// Verify the CRC-16.
	var crc uint16
	err = binary.Read(r, binary.BigEndian, &crc)
	if err != nil {
		return nil, err
	}
	got := crc16.ChecksumIBM(data)
	if crc != got {
		return nil, fmt.Errorf("frame.NewFrame: checksum mismatch; expected 0x%04X, got 0x%04X.", crc, got)
	}

	return frame, nil
}
