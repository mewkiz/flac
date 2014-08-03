// Package frame contains functions for parsing FLAC encoded audio data.
package frame

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/mewkiz/pkg/bit"
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
	h := crc16.NewIBM()
	hr := io.TeeReader(r, h)

	// Frame header.
	frame = new(Frame)
	frame.Header, err = NewHeader(hr)
	if err != nil {
		return nil, err
	}

	// Subframes.
	br := bit.NewReader(hr)
	hdr := frame.Header
	for subFrameNum := 0; subFrameNum < hdr.ChannelOrder.ChannelCount(); subFrameNum++ {
		// NOTE: This piece of code is based on https://github.com/eaburns/flac/blob/master/decode.go#L437
		// It is governed by a MIT license: https://github.com/eaburns/flac/blob/master/LICENSE
		bps := uint(hdr.BitsPerSample)
		switch hdr.ChannelOrder {
		case ChannelLeftSide, ChannelMidSide:
			if subFrameNum == 1 {
				bps++
			}
		case ChannelRightSide:
			if subFrameNum == 0 {
				bps++
			}
		}

		subframe, err := hdr.NewSubFrame(br, bps)
		if err != nil {
			return nil, err
		}
		frame.SubFrames = append(frame.SubFrames, subframe)
	}

	// Padding.
	// TODO(u): Verify paddings.
	// ignore bits up to byte boundery.
	br = bit.NewReader(hr)

	// Frame footer.
	// Verify the CRC-16.
	got := h.Sum16()

	var want uint16
	err = binary.Read(r, binary.BigEndian, &want)
	if err != nil {
		return nil, err
	}
	if got != want {
		return nil, fmt.Errorf("frame.NewFrame: checksum mismatch; expected 0x%04X, got 0x%04X", want, got)
	}

	// Decorrelate the left and right channels from each other.
	decorrelate(frame)

	return frame, nil
}

// decorrelate decorrelates the left and right channels from each other.
//
// ref: https://www.xiph.org/flac/format.html#interchannel
func decorrelate(frame *Frame) {
	// NOTE: This piece of code is based on https://github.com/eaburns/flac/blob/master/decode.go#L341
	// It is governed by a MIT license: https://github.com/eaburns/flac/blob/master/LICENSE
	// TODO(u): Verify that the channel mapping is correct (left, right, mid, leftSample, ...)
	switch frame.Header.ChannelOrder {
	case ChannelLeftSide:
		left := frame.SubFrames[0].Samples
		right := frame.SubFrames[1].Samples
		for i, leftSample := range left {
			right[i] = leftSample - right[i]
		}
	case ChannelRightSide:
		right := frame.SubFrames[0].Samples
		left := frame.SubFrames[1].Samples
		for i, leftSample := range left {
			right[i] += leftSample
		}
	case ChannelMidSide:
		mid := frame.SubFrames[0].Samples
		side := frame.SubFrames[1].Samples
		for i, midSample := range mid {
			sideSample := side[i]
			midSample *= 2
			midSample |= (sideSample & 1) // if side is odd
			mid[i] = (midSample + sideSample) / 2
			side[i] = (midSample - sideSample) / 2
		}
	}
}
