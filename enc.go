package flac

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"

	"github.com/icza/bitio"
	"github.com/mewkiz/flac/meta"
	"github.com/mewkiz/pkg/errutil"
)

// Encode writes the FLAC audio stream to w.
func Encode(w io.Writer, stream *Stream) error {
	// Create a bit writer to the output stream.

	// TODO: Remove buf when me manage to find a way to flush bits without
	// closing the underlying writer.

	// Use a temporary buffer to avoid closing the underlying writer when calling
	// `Close` on the bit writer to flushing pending bits.
	buf := new(bytes.Buffer)
	enc := &encoder{bw: bitio.NewWriter(buf)}

	// Store FLAC signature.
	if _, err := enc.bw.Write(signature); err != nil {
		return errutil.Err(err)
	}

	// Store StreamInfo metadata block header.
	infoHdr := meta.Header{
		IsLast: len(stream.Blocks) == 0,
		Type:   meta.TypeStreamInfo,
		// The StreamInfo metadata block body is 34 bytes in length.
		//
		//    34 = (16+16+24+24+20+3+5+36+8*16)/8
		Length: 34,
	}
	if err := enc.writeBlockHeader(infoHdr); err != nil {
		return errutil.Err(err)
	}

	// Store the StreamInfo metadata block.
	if err := enc.writeStreamInfo(stream.Info); err != nil {
		return errutil.Err(err)
	}

	// Store metadata blocks.
	for _, block := range stream.Blocks {
		if block.Type > meta.TypePicture {
			log.Printf("ignoring metadata block of unknown block type %d", block.Type)
			continue
		}

		if err := enc.writeBlockHeader(block.Header); err != nil {
			return errutil.Err(err)
		}
		var err error
		switch body := block.Body.(type) {
		case *meta.Application:
			err = enc.writeApplication(body)
		case *meta.SeekTable:
			err = enc.writeSeekTable(body)
		case *meta.VorbisComment:
			err = enc.writeVorbisComment(body)
		case *meta.CueSheet:
			err = enc.writeCueSheet(body)
		case *meta.Picture:
			err = enc.writePicture(body)
		default:
			err = enc.writePadding(int(block.Length))
		}
		if err != nil {
			return errutil.Err(err)
		}
	}

	// Flush pending bit writes.
	if err := enc.bw.Close(); err != nil {
		return errutil.Err(err)
	}

	// Copy buffer to output stream.
	if _, err := io.Copy(w, buf); err != nil {
		return errutil.Err(err)
	}

	// TODO: Implement proper encoding support for audio samples. For now, copy
	// the audio sample stream verbatim from the source file.
	if _, err := io.Copy(w, stream.r); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// An encoder represents a FLAC encoder.
type encoder struct {
	// Bit writer to the output stream.
	bw bitio.Writer
}

// writeBlockHeader writes the header of a metadata block.
func (enc *encoder) writeBlockHeader(hdr meta.Header) error {
	// 1 bit: IsLast.
	x := uint64(0)
	if hdr.IsLast {
		x = 1
	}
	if err := enc.bw.WriteBits(x, 1); err != nil {
		return errutil.Err(err)
	}

	// 7 bits: Type.
	if err := enc.bw.WriteBits(uint64(hdr.Type), 7); err != nil {
		return errutil.Err(err)
	}

	// 24 bits: Length.
	if err := enc.bw.WriteBits(uint64(hdr.Length), 24); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writeStreamInfo stores the body of a StreamInfo metadata block.
func (enc *encoder) writeStreamInfo(si *meta.StreamInfo) error {
	// 16 bits: BlockSizeMin.
	if err := enc.bw.WriteBits(uint64(si.BlockSizeMin), 16); err != nil {
		return errutil.Err(err)
	}

	// 16 bits: BlockSizeMax.
	if err := enc.bw.WriteBits(uint64(si.BlockSizeMax), 16); err != nil {
		return errutil.Err(err)
	}

	// 24 bits: FrameSizeMin.
	if err := enc.bw.WriteBits(uint64(si.FrameSizeMin), 24); err != nil {
		return errutil.Err(err)
	}

	// 24 bits: FrameSizeMax.
	if err := enc.bw.WriteBits(uint64(si.FrameSizeMax), 24); err != nil {
		return errutil.Err(err)
	}

	// 20 bits: SampleRate.
	if err := enc.bw.WriteBits(uint64(si.SampleRate), 20); err != nil {
		return errutil.Err(err)
	}

	// 3 bits: NChannels; stored as (number of channels) - 1.
	if err := enc.bw.WriteBits(uint64(si.NChannels-1), 3); err != nil {
		return errutil.Err(err)
	}

	// 5 bits: BitsPerSample; stored as (bits-per-sample) - 1.
	if err := enc.bw.WriteBits(uint64(si.BitsPerSample-1), 5); err != nil {
		return errutil.Err(err)
	}

	// 36 bits: NSamples.
	if err := enc.bw.WriteBits(uint64(si.NSamples), 36); err != nil {
		return errutil.Err(err)
	}

	// 16 bytes: MD5sum.
	if _, err := enc.bw.Write(si.MD5sum[:]); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writePadding writes the body of a Padding metadata block.
func (enc *encoder) writePadding(n int) error {
	for i := 0; i < n; i++ {
		if err := enc.bw.WriteByte(0); err != nil {
			return errutil.Err(err)
		}
	}
	return nil
}

// writeApplication writes the body of an Application metadata block.
func (enc *encoder) writeApplication(app *meta.Application) error {
	// 32 bits: ID.
	if err := enc.bw.WriteBits(uint64(app.ID), 32); err != nil {
		return errutil.Err(err)
	}

	// Check if the Application block only contains an ID.
	if _, err := enc.bw.Write(app.Data); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writeSeekTable writes the body of a SeekTable metadata block.
func (enc *encoder) writeSeekTable(table *meta.SeekTable) error {
	for _, point := range table.Points {
		if err := binary.Write(enc.bw, binary.BigEndian, point); err != nil {
			return errutil.Err(err)
		}
	}
	return nil
}

// writeVorbisComment writes the body of a VorbisComment metadata block.
func (enc *encoder) writeVorbisComment(comment *meta.VorbisComment) error {
	// 32 bits: vendor length.
	x := uint32(len(comment.Vendor))
	if err := binary.Write(enc.bw, binary.LittleEndian, x); err != nil {
		return errutil.Err(err)
	}

	// (vendor length) bits: Vendor.
	if _, err := enc.bw.Write([]byte(comment.Vendor)); err != nil {
		return errutil.Err(err)
	}

	// Store tags.
	// 32 bits: number of tags.
	x = uint32(len(comment.Tags))
	if err := binary.Write(enc.bw, binary.LittleEndian, x); err != nil {
		return errutil.Err(err)
	}
	for _, tag := range comment.Tags {
		// Store tag, which has the following format:
		//    NAME=VALUE
		buf := []byte(tag[0] + "=" + tag[1])

		// 32 bits: vector length
		x = uint32(len(buf))
		if err := binary.Write(enc.bw, binary.LittleEndian, x); err != nil {
			return errutil.Err(err)
		}

		// (vector length): vector.
		if _, err := enc.bw.Write(buf); err != nil {
			return errutil.Err(err)
		}
	}

	return nil
}

// writeCueSheet writes the body of a CueSheet metadata block.
func (enc *encoder) writeCueSheet(cs *meta.CueSheet) error {
	// Parse cue sheet.
	// 128 bytes: MCN.
	mcn := make([]byte, 128)
	copy(mcn, cs.MCN)
	if _, err := enc.bw.Write(mcn); err != nil {
		return errutil.Err(err)
	}

	// 64 bits: NLeadInSamples.
	if err := enc.bw.WriteBits(cs.NLeadInSamples, 64); err != nil {
		return errutil.Err(err)
	}

	// 1 bit: IsCompactDisc.
	x := uint64(0)
	if cs.IsCompactDisc {
		x = 1
	}
	if err := enc.bw.WriteBits(x, 1); err != nil {
		return errutil.Err(err)
	}

	// 7 bits and 258 bytes: reserved.
	if err := enc.bw.WriteBits(0, 7); err != nil {
		return errutil.Err(err)
	}
	// TODO: Remove unnecessary allocation.
	padding := make([]byte, 258)
	if _, err := enc.bw.Write(padding); err != nil {
		return errutil.Err(err)
	}

	// Parse cue sheet tracks.
	// 8 bits: (number of tracks)
	x = uint64(len(cs.Tracks))
	if err := enc.bw.WriteBits(x, 8); err != nil {
		return errutil.Err(err)
	}
	for _, track := range cs.Tracks {
		// 64 bits: Offset.
		if err := enc.bw.WriteBits(track.Offset, 64); err != nil {
			return errutil.Err(err)
		}

		// 8 bits: Num.
		if err := enc.bw.WriteBits(uint64(track.Num), 8); err != nil {
			return errutil.Err(err)
		}

		// 12 bytes: ISRC.
		isrc := make([]byte, 12)
		copy(isrc, track.ISRC)
		if _, err := enc.bw.Write(isrc); err != nil {
			return errutil.Err(err)
		}

		// 1 bit: IsAudio.
		x := uint64(0)
		if !track.IsAudio {
			x = 1
		}
		if err := enc.bw.WriteBits(x, 1); err != nil {
			return errutil.Err(err)
		}

		// 1 bit: HasPreEmphasis.
		// mask = 01000000
		x = 0
		if track.HasPreEmphasis {
			x = 1
		}
		if err := enc.bw.WriteBits(x, 1); err != nil {
			return errutil.Err(err)
		}

		// 6 bits and 13 bytes: reserved.
		// mask = 00111111
		if err := enc.bw.WriteBits(0, 6); err != nil {
			return errutil.Err(err)
		}
		// TODO: Remove unnecessary allocation.
		padding := make([]byte, 13)
		if _, err := enc.bw.Write(padding); err != nil {
			return errutil.Err(err)
		}

		// Parse indicies.
		// 8 bits: (number of indicies)
		x = uint64(len(track.Indicies))
		if err := enc.bw.WriteBits(x, 8); err != nil {
			return errutil.Err(err)
		}
		for _, index := range track.Indicies {
			// 64 bits: Offset.
			if err := enc.bw.WriteBits(index.Offset, 64); err != nil {
				return errutil.Err(err)
			}

			// 8 bits: Num.
			if err := enc.bw.WriteBits(uint64(index.Num), 8); err != nil {
				return errutil.Err(err)
			}

			// 3 bytes: reserved.
			// TODO: Remove unnecessary allocation.
			padding := make([]byte, 3)
			if _, err := enc.bw.Write(padding); err != nil {
				return errutil.Err(err)
			}
		}
	}

	return nil
}

// writePicture writes the body of a Picture metadata block.
func (enc *encoder) writePicture(pic *meta.Picture) error {
	// 32 bits: Type.
	if err := enc.bw.WriteBits(uint64(pic.Type), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: (MIME type length).
	x := uint64(len(pic.MIME))
	if err := enc.bw.WriteBits(x, 32); err != nil {
		return errutil.Err(err)
	}

	// (MIME type length) bytes: MIME.
	if _, err := enc.bw.Write([]byte(pic.MIME)); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: (description length).
	x = uint64(len(pic.Desc))
	if err := enc.bw.WriteBits(x, 32); err != nil {
		return errutil.Err(err)
	}

	// (description length) bytes: Desc.
	if _, err := enc.bw.Write([]byte(pic.Desc)); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: Width.
	if err := enc.bw.WriteBits(uint64(pic.Width), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: Height.
	if err := enc.bw.WriteBits(uint64(pic.Height), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: Depth.
	if err := enc.bw.WriteBits(uint64(pic.Depth), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: NPalColors.
	if err := enc.bw.WriteBits(uint64(pic.NPalColors), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: (data length).
	x = uint64(len(pic.Data))
	if err := enc.bw.WriteBits(x, 32); err != nil {
		return errutil.Err(err)
	}

	// (data length) bytes: Data.
	if _, err := enc.bw.Write([]byte(pic.Data)); err != nil {
		return errutil.Err(err)
	}

	return nil
}
