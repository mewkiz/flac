package meta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
)

// A CueSheet describes how tracks are laid out within a FLAC stream.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_cuesheet
type CueSheet struct {
	// Media catalog number.
	MCN string
	// Number of lead-in samples. This field only has meaning for CD-DA cue
	// sheets; for other uses it should be 0. Refer to the spec for additional
	// information.
	NLeadInSamples uint64
	// Specifies if the cue sheet corresponds to a Compact Disc.
	IsCompactDisc bool
	// One or more tracks. The last track of a cue sheet is always the lead-out
	// track.
	Tracks []CueSheetTrack
}

// parseCueSheet reads and parses the body of an CueSheet metadata block.
func (block *Block) parseCueSheet() error {
	// Parse cue sheet.
	// 128 bytes: MCN.
	buf, err := readBytes(block.lr, 128)
	if err != nil {
		return err
	}
	cs := new(CueSheet)
	block.Body = cs
	cs.MCN = stringFromSZ(buf)

	// 64 bits: NLeadInSamples.
	err = binary.Read(block.lr, binary.BigEndian, &cs.NLeadInSamples)
	if err != nil {
		return err
	}

	// 1 bit: IsCompactDisc.
	var x uint8
	err = binary.Read(block.lr, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	if x&1 != 0 {
		cs.IsCompactDisc = true
	}

	// 7 bits and 258 bytes: reserved.
	// mask = 11111110
	if x&0xFE != 0 {
		return ErrInvalidPadding
	}
	lr := io.LimitReader(block.lr, 258)
	zr := zeros{r: lr}
	_, err = io.Copy(ioutil.Discard, zr)
	if err != nil {
		return err
	}

	// Parse cue sheet tracks.
	// 8 bits: (number of tracks)
	err = binary.Read(block.lr, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	if x < 1 {
		return errors.New("meta.Block.parseCueSheet: at least one track required")
	}
	cs.Tracks = make([]CueSheetTrack, x)
	for i := range cs.Tracks {
		// 64 bits: Offset.
		track := &cs.Tracks[i]
		err = binary.Read(block.lr, binary.BigEndian, &track.Offset)
		if err != nil {
			return err
		}

		// 8 bits: Num.
		err = binary.Read(block.lr, binary.BigEndian, &track.Num)
		if err != nil {
			return err
		}

		// 12 bytes: ISRC.
		buf, err = readBytes(block.lr, 12)
		if err != nil {
			return err
		}
		track.ISRC = stringFromSZ(buf)

		// 1 bit: IsAudio.
		err = binary.Read(block.lr, binary.BigEndian, &x)
		if err != nil {
			return err
		}
		if x&1 == 0 {
			track.IsAudio = true
		}

		// 1 bit: HasPreEmphasis.
		if x&2 == 0 {
			track.HasPreEmphasis = true
		}

		// 6 bits and 13 bytes: reserved.
		// mask = 11111110
		if x&0xFC != 0 {
			return ErrInvalidPadding
		}
		lr = io.LimitReader(block.lr, 13)
		zr = zeros{r: lr}
		_, err = io.Copy(ioutil.Discard, zr)
		if err != nil {
			return err
		}

		// Parse indicies.
		// 8 bits: (number of indicies)
		err = binary.Read(block.lr, binary.BigEndian, &x)
		if err != nil {
			return err
		}
		track.Indicies = make([]CueSheetTrackIndex, x)
		for i := range track.Indicies {
			index := &track.Indicies[i]
			// 64 bits: Offset.
			err = binary.Read(block.lr, binary.BigEndian, &index.Offset)
			if err != nil {
				return err
			}

			// 8 bits: Num.
			err = binary.Read(block.lr, binary.BigEndian, &index.Num)
			if err != nil {
				return err
			}

			// 3 bytes: reserved.
			lr = io.LimitReader(block.lr, 3)
			zr = zeros{r: lr}
			_, err = io.Copy(ioutil.Discard, zr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// stringFromSZ converts the provided byte slice to a string after terminating
// it at the first occurance of a NULL character.
func stringFromSZ(buf []byte) string {
	pos := bytes.IndexByte(buf, 0)
	if pos == -1 {
		return string(buf)
	}
	return string(buf[:pos])
}

// CueSheetTrack contains the start offset of a track and other track specific
// metadata.
type CueSheetTrack struct {
	// Track offset in samples, relative to the beginning of the FLAC audio
	// stream.
	Offset uint64
	// Track number; never 0, always unique.
	Num uint8
	// International Standard Recording Code; empty string if not present.
	//
	// ref: http://isrc.ifpi.org/
	ISRC string
	// Specifies if the track contains audio or data.
	IsAudio bool
	// Specifies if the track has been recorded with pre-emphasis
	HasPreEmphasis bool
	// Every track has one or more track index points, except for the lead-out
	// track which has zero. Each index point specifies a position within the
	// track.
	Indicies []CueSheetTrackIndex
}

// A CueSheetTrackIndex specifies a position within a track.
type CueSheetTrackIndex struct {
	// Index point offset in samples, relative to the track offset.
	Offset uint64
	// Index point number; subsequently incrementing by 1 and always unique
	// within a track.
	Num uint8
}
