package flac

import (
	"crypto/md5"
	"hash"
	"io"

	"github.com/icza/bitio"
	"github.com/mewkiz/flac/meta"
	"github.com/mewkiz/pkg/errutil"
)

// An Encoder represents a FLAC encoder.
type Encoder struct {
	// FLAC stream of encoder.
	*Stream
	// Underlying io.Writer to the output stream.
	w io.Writer
	// io.Closer to flush pending writes to output stream.
	c io.Closer
	// Current frame number if block size is fixed, and the first sample number
	// of the current frame otherwise.
	curNum uint64
	// MD5 running hash of unencoded audio samples.
	md5sum hash.Hash
}

// NewEncoder returns a new FLAC encoder for the given metadata StreamInfo block
// and optional metadata blocks.
func NewEncoder(w io.Writer, info *meta.StreamInfo, blocks ...*meta.Block) (*Encoder, error) {
	// Store FLAC signature.
	enc := &Encoder{
		Stream: &Stream{
			Info:   info,
			Blocks: blocks,
		},
		w:      w,
		md5sum: md5.New(),
	}
	if c, ok := w.(io.Closer); ok {
		enc.c = c
	}
	bw := bitio.NewWriter(w)
	if _, err := bw.Write(flacSignature); err != nil {
		return nil, errutil.Err(err)
	}
	// Encode metadata blocks.
	// TODO: consider using bufio.NewWriter.
	if err := encodeStreamInfo(bw, info, len(blocks) == 0); err != nil {
		return nil, errutil.Err(err)
	}
	for i, block := range blocks {
		if err := encodeBlock(bw, block.Body, i == len(blocks)-1); err != nil {
			return nil, errutil.Err(err)
		}
	}
	// Flush pending writes of metadata blocks.
	if _, err := bw.Align(); err != nil {
		return nil, errutil.Err(err)
	}
	// Return encoder to be used for encoding audio samples.
	return enc, nil
}

// Close closes the underlying io.Writer of the encoder and flushes any pending
// writes. If the io.Writer implements io.Seeker, the encoder will update the
// StreamInfo metadata block with the MD5 checksum of the unencoded audio data,
// the number of samples, and the minimum and maximum frame size and block size.
func (enc *Encoder) Close() error {
	// TODO: check if we should flush bit writer before seeking on enc.w.
	// Update StreamInfo metadata block.
	if ws, ok := enc.w.(io.WriteSeeker); ok {
		if _, err := ws.Seek(int64(len(flacSignature)), io.SeekStart); err != nil {
			return errutil.Err(err)
		}
		// Update MD5 checksum of unencoded audio samples.
		sum := enc.md5sum.Sum(nil)
		for i := range sum {
			enc.Info.MD5sum[i] = sum[i]
		}
		// TODO: update number of samples, and the minimum and maximum frame size
		// and block size.
		bw := bitio.NewWriter(ws)
		// Write updated StreamInfo metadata block to output stream.
		if err := encodeStreamInfo(bw, enc.Info, len(enc.Blocks) == 0); err != nil {
			return errutil.Err(err)
		}
		if _, err := bw.Align(); err != nil {
			return errutil.Err(err)
		}
	}
	if enc.c != nil {
		return enc.c.Close()
	}
	return nil
}
