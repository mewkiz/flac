// Package frame contains functions for parsing FLAC encoded audio data.
package frame

type Frame struct {
	Header    *Header
	SubFrames []SubFrame
	Footer    FrameFooter
}

type FrameFooter struct {
	CRC uint16
}

/**
f.Footer.CRC = binary.BigEndian.Uint16(buf.Next(2))
*/
