// Package frame contains functions for parsing FLAC encoded audio data.
package frame

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/eaburns/bit"
	"github.com/mewkiz/pkg/hashutil"
	"github.com/mewkiz/pkg/hashutil/crc16"
)

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
func NewFrame(r io.Reader) (frame *Frame, err error) {
	// Create a new hash reader which adds the data from all read operations to a
	// running hash.
	hr := hashutil.NewHashReader(r, crc16.NewIBM())

	// Frame header.
	frame = new(Frame)
	frame.Header, err = NewHeader(hr)
	if err != nil {
		return nil, err
	}

	// Subframes.
	br := bit.NewReader(hr)
	h := frame.Header
	for i := 0; i < h.ChannelOrder.ChannelCount(); i++ {
		subframe, err := h.NewSubFrame(br)
		if err != nil {
			return nil, err
		}
		frame.SubFrames = append(frame.SubFrames, subframe)
	}

	// Padding.
	// ignore bits up to byte boundery.
	br = bit.NewReader(hr)
	/// ### [ TODO ] ###
	///    - verify paddings
	/// ### [/ TODO ] ###
	///bitOff, err := br.Seek(0, bit.SeekCur)
	///if err != nil {
	///	return nil, err
	///}
	///padBitCount := bitOff % 8
	///if padBitCount != 0 {
	///	pad, err := br.Read(int(padBitCount))
	///	if err != nil {
	///		return nil, err
	///	}
	///	if pad.Uint64() != 0 {
	///		return nil, errors.New("frame.NewFrame: invalid padding; must be 0")
	///	}
	///}

	// Frame footer.

	// Verify the CRC-16.
	got := hr.Sum16()
	// Disable hashing on hr.
	hr.Hash = nil

	var want uint16
	err = binary.Read(hr, binary.BigEndian, &want)
	if err != nil {
		return nil, err
	}
	if got != want {
		return nil, fmt.Errorf("frame.NewFrame: checksum mismatch; expected 0x%04X, got 0x%04X", want, got)
	}

	return frame, nil
}
