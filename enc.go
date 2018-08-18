package flac

import (
	"encoding/binary"
	"io"
	"log"

	"github.com/icza/bitio"
	"github.com/mewkiz/flac/meta"
	"github.com/mewkiz/pkg/errutil"
)

// An Encoder represents a FLAC encoder.
type Encoder struct {
	// FLAC stream of encoder.
	stream *Stream
	// Underlying io.Writer to the output stream.
	w io.Writer
	// io.Closer to flush bit writer.
	c io.Closer
	// Current frame number if block size is fixed, and the first sample number
	// of the current frame otherwise.
	curNum uint64
}

// NewEncoder returns a new FLAC encoder for the given sample rate in Hz, number
// of channels and bits-per-sample.
func NewEncoder(w io.Writer, sampleRate, nchannels, bps int) (*Encoder, error) {
	// Store FLAC signature.
	enc := &Encoder{
		w: w,
	}
	bw := bitio.NewWriter(w)
	enc.c = bw
	if _, err := bw.Write(flacSignature); err != nil {
		return nil, errutil.Err(err)
	}

	// Store the StreamInfo metadata block.
	stream := &Stream{
		Info: &meta.StreamInfo{
			BlockSizeMin: 16, // TODO: set closer limit of BlockSize min and max, based on encoded frames.
			BlockSizeMax: 65535,
			//FrameSizeMin:  0, // TODO: set closer limit of FrameSize min and max, based on encoded frames.
			//FrameSizeMax:  0,
			SampleRate:    uint32(sampleRate),
			NChannels:     uint8(nchannels),
			BitsPerSample: uint8(bps),
			//NSamples:      0, // TODO: set NSamples, based on unencoded audio samples.
			//MD5sum:        0, // TODO: set MD5sum, based on unencoded audio samples.
		},
	}
	enc.stream = stream
	infoHdr := meta.Header{
		Type: meta.TypeStreamInfo,
		//Length: 0, // Length is calculated in writeStreamInfo
		IsLast: true,
	}
	if err := enc.writeStreamInfo(bw, infoHdr, stream.Info); err != nil {
		return nil, errutil.Err(err)
	}
	// Flush pending bit writes.
	if _, err := bw.Align(); err != nil {
		return nil, errutil.Err(err)
	}
	// Return encoder to be used for encoding audio samples.
	return enc, nil
}

// Close closes the underlying io.Writer of the encoder and flushes any pending
// writes.
func (enc *Encoder) Close() error {
	return enc.c.Close()
}

// TODO: Remove Encode in favour of using NewEncoder.

// Encode writes the FLAC audio stream to w.
func Encode(w io.Writer, stream *Stream) error {
	// Create a bit writer to the output stream.
	bw := bitio.NewWriter(w)
	enc := &Encoder{
		w: w,
		c: bw,
	}

	// Store FLAC signature.
	if _, err := bw.Write(flacSignature); err != nil {
		return errutil.Err(err)
	}

	// Store the StreamInfo metadata block.
	infoHdr := meta.Header{
		IsLast: len(stream.Blocks) == 0,
		Type:   meta.TypeStreamInfo,
	}
	if err := enc.writeStreamInfo(bw, infoHdr, stream.Info); err != nil {
		return errutil.Err(err)
	}

	// Store metadata blocks.
	for i, block := range stream.Blocks {
		if block.Type > meta.TypePicture {
			log.Printf("ignoring metadata block of unknown block type %d", block.Type)
			continue
		}

		// Store metadata block body.
		var err error
		hdr := block.Header
		hdr.IsLast = i == len(stream.Blocks)-1
		switch body := block.Body.(type) {
		case *meta.Application:
			err = enc.writeApplication(bw, hdr, body)
		case *meta.SeekTable:
			err = enc.writeSeekTable(bw, hdr, body)
		case *meta.VorbisComment:
			err = enc.writeVorbisComment(bw, hdr, body)
		case *meta.CueSheet:
			err = enc.writeCueSheet(bw, hdr, body)
		case *meta.Picture:
			err = enc.writePicture(bw, hdr, body)
		default:
			err = enc.writePadding(bw, hdr)
		}
		if err != nil {
			return errutil.Err(err)
		}
	}

	// Flush pending bit writes.
	if _, err := bw.Align(); err != nil {
		return errutil.Err(err)
	}

	// TODO: Implement proper encoding support for audio samples. For now, copy
	// the audio sample stream verbatim from the source file.
	if _, err := io.Copy(w, stream.r); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writeBlockHeader writes the header of a metadata block.
func (enc *Encoder) writeBlockHeader(bw bitio.Writer, hdr meta.Header) error {
	// 1 bit: IsLast.
	x := uint64(0)
	if hdr.IsLast {
		x = 1
	}
	if err := bw.WriteBits(x, 1); err != nil {
		return errutil.Err(err)
	}

	// 7 bits: Type.
	if err := bw.WriteBits(uint64(hdr.Type), 7); err != nil {
		return errutil.Err(err)
	}

	// 24 bits: Length.
	if err := bw.WriteBits(uint64(hdr.Length), 24); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writeStreamInfo stores the body of a StreamInfo metadata block.
func (enc *Encoder) writeStreamInfo(bw bitio.Writer, hdr meta.Header, si *meta.StreamInfo) error {
	// Store metadata block header.
	const (
		BlockSizeMinBits  = 16
		BlockSizeMaxBits  = 16
		FrameSizeMinBits  = 24
		FrameSizeMaxBits  = 24
		SampleRateBits    = 20
		NChannelsBits     = 3
		BitsPerSampleBits = 5
		NSamplesBits      = 36
		MD5sumBits        = 8 * 16
	)
	nbits := int64(BlockSizeMinBits + BlockSizeMaxBits + FrameSizeMinBits +
		FrameSizeMaxBits + SampleRateBits + NChannelsBits + BitsPerSampleBits +
		NSamplesBits + MD5sumBits)
	hdr.Length = nbits / 8
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	// 16 bits: BlockSizeMin.
	if err := bw.WriteBits(uint64(si.BlockSizeMin), 16); err != nil {
		return errutil.Err(err)
	}

	// 16 bits: BlockSizeMax.
	if err := bw.WriteBits(uint64(si.BlockSizeMax), 16); err != nil {
		return errutil.Err(err)
	}

	// 24 bits: FrameSizeMin.
	if err := bw.WriteBits(uint64(si.FrameSizeMin), 24); err != nil {
		return errutil.Err(err)
	}

	// 24 bits: FrameSizeMax.
	if err := bw.WriteBits(uint64(si.FrameSizeMax), 24); err != nil {
		return errutil.Err(err)
	}

	// 20 bits: SampleRate.
	if err := bw.WriteBits(uint64(si.SampleRate), 20); err != nil {
		return errutil.Err(err)
	}

	// 3 bits: NChannels; stored as (number of channels) - 1.
	if err := bw.WriteBits(uint64(si.NChannels-1), 3); err != nil {
		return errutil.Err(err)
	}

	// 5 bits: BitsPerSample; stored as (bits-per-sample) - 1.
	if err := bw.WriteBits(uint64(si.BitsPerSample-1), 5); err != nil {
		return errutil.Err(err)
	}

	// 36 bits: NSamples.
	if err := bw.WriteBits(si.NSamples, 36); err != nil {
		return errutil.Err(err)
	}

	// 16 bytes: MD5sum.
	if _, err := bw.Write(si.MD5sum[:]); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writePadding writes the body of a Padding metadata block.
func (enc *Encoder) writePadding(bw bitio.Writer, hdr meta.Header) error {
	// Store metadata block header.
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	for i := 0; i < int(hdr.Length); i++ {
		if err := bw.WriteByte(0); err != nil {
			return errutil.Err(err)
		}
	}
	return nil
}

// writeApplication writes the body of an Application metadata block.
func (enc *Encoder) writeApplication(bw bitio.Writer, hdr meta.Header, app *meta.Application) error {
	// Store metadata block header.
	const (
		IDBits = 32
	)
	nbits := int64(IDBits + 8*len(app.Data))
	hdr.Length = nbits / 8
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	// 32 bits: ID.
	if err := bw.WriteBits(uint64(app.ID), 32); err != nil {
		return errutil.Err(err)
	}

	// Check if the Application block only contains an ID.
	if _, err := bw.Write(app.Data); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writeSeekTable writes the body of a SeekTable metadata block.
func (enc *Encoder) writeSeekTable(bw bitio.Writer, hdr meta.Header, table *meta.SeekTable) error {
	// Store metadata block header.
	const (
		SampleNumBits = 64
		OffsetBits    = 64
		NSamplesBits  = 16
		PointBits     = SampleNumBits + OffsetBits + NSamplesBits
	)
	nbits := int64(PointBits * len(table.Points))
	hdr.Length = nbits / 8
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	for _, point := range table.Points {
		if err := binary.Write(bw, binary.BigEndian, point); err != nil {
			return errutil.Err(err)
		}
	}
	return nil
}

// writeVorbisComment writes the body of a VorbisComment metadata block.
func (enc *Encoder) writeVorbisComment(bw bitio.Writer, hdr meta.Header, comment *meta.VorbisComment) error {
	// Store metadata block header.
	const (
		VendorLenBits = 32
		NTagsBits     = 32
	)
	nbits := int64(VendorLenBits + 8*len(comment.Vendor) + NTagsBits)
	for _, tag := range comment.Tags {
		const (
			VectorLenBits = 32
			EqualBits     = 8 * 1
		)
		nbits += int64(VectorLenBits + 8*len(tag[0]) + EqualBits + 8*len(tag[1]))
	}
	hdr.Length = nbits / 8
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	// 32 bits: vendor length.
	x := uint32(len(comment.Vendor))
	if err := binary.Write(bw, binary.LittleEndian, x); err != nil {
		return errutil.Err(err)
	}

	// (vendor length) bits: Vendor.
	if _, err := bw.Write([]byte(comment.Vendor)); err != nil {
		return errutil.Err(err)
	}

	// Store tags.
	// 32 bits: number of tags.
	x = uint32(len(comment.Tags))
	if err := binary.Write(bw, binary.LittleEndian, x); err != nil {
		return errutil.Err(err)
	}
	for _, tag := range comment.Tags {
		// Store tag, which has the following format:
		//    NAME=VALUE
		buf := []byte(tag[0] + "=" + tag[1])

		// 32 bits: vector length
		x = uint32(len(buf))
		if err := binary.Write(bw, binary.LittleEndian, x); err != nil {
			return errutil.Err(err)
		}

		// (vector length): vector.
		if _, err := bw.Write(buf); err != nil {
			return errutil.Err(err)
		}
	}

	return nil
}

// writeCueSheet writes the body of a CueSheet metadata block.
func (enc *Encoder) writeCueSheet(bw bitio.Writer, hdr meta.Header, cs *meta.CueSheet) error {
	// Store metadata block header.
	const (
		MCNBits            = 8 * 128
		NLeadInSamplesBits = 64
		IsCompactDiscBits  = 1
		Reserved1Bits      = 7 + 8*258
		NTracksBits        = 8
	)
	nbits := int64(MCNBits + NLeadInSamplesBits + IsCompactDiscBits +
		Reserved1Bits + NTracksBits)
	for _, track := range cs.Tracks {
		const (
			OffsetBits         = 64
			NumBits            = 8
			ISRCBits           = 8 * 12
			IsAudioBits        = 1
			HasPreEmphasisBits = 1
			Reserved2Bits      = 6 + 8*13
			NIndicesBits       = 8
		)
		nbits += OffsetBits + NumBits + ISRCBits + IsAudioBits + HasPreEmphasisBits + Reserved2Bits + NIndicesBits
		for range track.Indicies {
			const (
				OffsetBits    = 64
				NumBits       = 8
				Reserved3Bits = 8 * 3
			)
			nbits += OffsetBits + NumBits + Reserved3Bits
		}
	}
	hdr.Length = nbits / 8
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	// Parse cue sheet.
	// 128 bytes: MCN.
	mcn := make([]byte, 128)
	copy(mcn, cs.MCN)
	if _, err := bw.Write(mcn); err != nil {
		return errutil.Err(err)
	}

	// 64 bits: NLeadInSamples.
	if err := bw.WriteBits(cs.NLeadInSamples, 64); err != nil {
		return errutil.Err(err)
	}

	// 1 bit: IsCompactDisc.
	x := uint64(0)
	if cs.IsCompactDisc {
		x = 1
	}
	if err := bw.WriteBits(x, 1); err != nil {
		return errutil.Err(err)
	}

	// 7 bits and 258 bytes: reserved.
	if err := bw.WriteBits(0, 7); err != nil {
		return errutil.Err(err)
	}
	// TODO: Remove unnecessary allocation.
	padding := make([]byte, 258)
	if _, err := bw.Write(padding); err != nil {
		return errutil.Err(err)
	}

	// Parse cue sheet tracks.
	// 8 bits: (number of tracks)
	x = uint64(len(cs.Tracks))
	if err := bw.WriteBits(x, 8); err != nil {
		return errutil.Err(err)
	}
	for _, track := range cs.Tracks {
		// 64 bits: Offset.
		if err := bw.WriteBits(track.Offset, 64); err != nil {
			return errutil.Err(err)
		}

		// 8 bits: Num.
		if err := bw.WriteBits(uint64(track.Num), 8); err != nil {
			return errutil.Err(err)
		}

		// 12 bytes: ISRC.
		isrc := make([]byte, 12)
		copy(isrc, track.ISRC)
		if _, err := bw.Write(isrc); err != nil {
			return errutil.Err(err)
		}

		// 1 bit: IsAudio.
		x := uint64(0)
		if !track.IsAudio {
			x = 1
		}
		if err := bw.WriteBits(x, 1); err != nil {
			return errutil.Err(err)
		}

		// 1 bit: HasPreEmphasis.
		// mask = 01000000
		x = 0
		if track.HasPreEmphasis {
			x = 1
		}
		if err := bw.WriteBits(x, 1); err != nil {
			return errutil.Err(err)
		}

		// 6 bits and 13 bytes: reserved.
		// mask = 00111111
		if err := bw.WriteBits(0, 6); err != nil {
			return errutil.Err(err)
		}
		// TODO: Remove unnecessary allocation.
		padding := make([]byte, 13)
		if _, err := bw.Write(padding); err != nil {
			return errutil.Err(err)
		}

		// Parse indicies.
		// 8 bits: (number of indicies)
		x = uint64(len(track.Indicies))
		if err := bw.WriteBits(x, 8); err != nil {
			return errutil.Err(err)
		}
		for _, index := range track.Indicies {
			// 64 bits: Offset.
			if err := bw.WriteBits(index.Offset, 64); err != nil {
				return errutil.Err(err)
			}

			// 8 bits: Num.
			if err := bw.WriteBits(uint64(index.Num), 8); err != nil {
				return errutil.Err(err)
			}

			// 3 bytes: reserved.
			// TODO: Remove unnecessary allocation.
			padding := make([]byte, 3)
			if _, err := bw.Write(padding); err != nil {
				return errutil.Err(err)
			}
		}
	}

	return nil
}

// writePicture writes the body of a Picture metadata block.
func (enc *Encoder) writePicture(bw bitio.Writer, hdr meta.Header, pic *meta.Picture) error {
	// Store metadata block header.
	const (
		TypeBits       = 32
		MIMELenBits    = 32
		DescLenBits    = 32
		WidthBits      = 32
		HeightBits     = 32
		DepthBits      = 32
		NPalColorsBits = 32
		DataLenBits    = 32
	)
	nbits := int64(TypeBits + MIMELenBits + 8*len(pic.MIME) + DescLenBits +
		8*len(pic.Desc) + WidthBits + HeightBits + DepthBits + NPalColorsBits +
		DataLenBits + 8*len(pic.Data))
	hdr.Length = nbits / 8
	if err := enc.writeBlockHeader(bw, hdr); err != nil {
		return errutil.Err(err)
	}

	// Store metadata block body.
	// 32 bits: Type.
	if err := bw.WriteBits(uint64(pic.Type), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: (MIME type length).
	x := uint64(len(pic.MIME))
	if err := bw.WriteBits(x, 32); err != nil {
		return errutil.Err(err)
	}

	// (MIME type length) bytes: MIME.
	if _, err := bw.Write([]byte(pic.MIME)); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: (description length).
	x = uint64(len(pic.Desc))
	if err := bw.WriteBits(x, 32); err != nil {
		return errutil.Err(err)
	}

	// (description length) bytes: Desc.
	if _, err := bw.Write([]byte(pic.Desc)); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: Width.
	if err := bw.WriteBits(uint64(pic.Width), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: Height.
	if err := bw.WriteBits(uint64(pic.Height), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: Depth.
	if err := bw.WriteBits(uint64(pic.Depth), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: NPalColors.
	if err := bw.WriteBits(uint64(pic.NPalColors), 32); err != nil {
		return errutil.Err(err)
	}

	// 32 bits: (data length).
	x = uint64(len(pic.Data))
	if err := bw.WriteBits(x, 32); err != nil {
		return errutil.Err(err)
	}

	// (data length) bytes: Data.
	if _, err := bw.Write(pic.Data); err != nil {
		return errutil.Err(err)
	}

	return nil
}
