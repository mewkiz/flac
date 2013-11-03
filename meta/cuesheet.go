package meta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/mewkiz/pkg/readerutil"
)

// A CueSheet metadata block is for storing various information that can be used
// in a cue sheet. It supports track and index points, compatible with Red Book
// CD digital audio discs, as well as other CD-DA metadata such as media catalog
// number and track ISRCs. The CUESHEET block is especially useful for backing
// up CD-DA discs, but it can be used as a general purpose cueing mechanism for
// playback.
type CueSheet struct {
	// Media catalog number, in ASCII printable characters 0x20-0x7e. In general,
	// the media catalog number may be 0 to 128 bytes long; any unused characters
	// should be right-padded with NUL characters. For CD-DA, this is a thirteen
	// digit number, followed by 115 NUL bytes.
	MCN string
	// The number of lead-in samples. This field has meaning only for CD-DA
	// cuesheets; for other uses it should be 0. For CD-DA, the lead-in is the
	// TRACK 00 area where the table of contents is stored; more precisely, it is
	// the number of samples from the first sample of the media to the first
	// sample of the first index point of the first track. According to the Red
	// Book, the lead-in must be silence and CD grabbing software does not
	// usually store it; additionally, the lead-in must be at least two seconds
	// but may be longer. For these reasons the lead-in length is stored here so
	// that the absolute position of the first track can be computed. Note that
	// the lead-in stored here is the number of samples up to the first index
	// point of the first track, not necessarily to INDEX 01 of the first track;
	// even the first track may have INDEX 00 data.
	LeadInSampleCount uint64
	// true if the CUESHEET corresponds to a Compact Disc, else false.
	IsCompactDisc bool
	// The number of tracks. Must be at least 1 (because of the requisite
	// lead-out track). For CD-DA, this number must be no more than 100 (99
	// regular tracks and one lead-out track).
	TrackCount uint8
	// One or more tracks. A CUESHEET block is required to have a lead-out track;
	// it is always the last track in the CUESHEET. For CD-DA, the lead-out track
	// number must be 170 as specified by the Red Book, otherwise is must be 255.
	Tracks []CueSheetTrack
}

// A CueSheetTrack contains information about a track within a CueSheet.
type CueSheetTrack struct {
	// Track offset in samples, relative to the beginning of the FLAC audio
	// stream. It is the offset to the first index point of the track. (Note how
	// this differs from CD-DA, where the track's offset in the TOC is that of
	// the track's INDEX 01 even if there is an INDEX 00.) For CD-DA, the offset
	// must be evenly divisible by 588 samples (588 samples = 44100 samples/sec *
	// 1/75th of a sec).
	Offset uint64
	// Track number. A track number of 0 is not allowed to avoid conflicting with
	// the CD-DA spec, which reserves this for the lead-in. For CD-DA the number
	// must be 1-99, or 170 for the lead-out; for non-CD-DA, the track number
	// must for 255 for the lead-out. It is not required but encouraged to start
	// with track 1 and increase sequentially. Track numbers must be unique
	// within a CUESHEET.
	TrackNum uint8
	// Track ISRC. This is a 12-digit alphanumeric code. A value of 12 ASCII NUL
	// characters may be used to denote absence of an ISRC.
	ISRC string
	// The track type: true for audio, false for non-audio.
	IsAudio bool
	// The pre-emphasis flag: false for no pre-emphasis, true for pre-emphasis.
	// This corresponds to the CD-DA Q-channel control bit 5.
	HasPreEmphasis bool
	// The number of track index points. There must be at least one index in
	// every track in a CUESHEET except for the lead-out track, which must have
	// zero. For CD-DA, this number may be no more than 100.
	TrackIndexCount uint8
	// For all tracks except the lead-out track, one or more track index points.
	TrackIndexes []CueSheetTrackIndex
}

// A CueSheetTrackIndex contains information about an index point in a track.
type CueSheetTrackIndex struct {
	// Offset in samples, relative to the track offset, of the index point. For
	// CD-DA, the offset must be evenly divisible by 588 samples (588 samples =
	// 44100 samples/sec * 1/75th of a sec). Note that the offset is from the
	// beginning of the track, not the beginning of the audio data.
	Offset uint64
	// The index point number. For CD-DA, an index number of 0 corresponds to the
	// track pre-gap. The first index in a track must have a number of 0 or 1,
	// and subsequently, index numbers must increase by 1. Index numbers must be
	// unique within a track.
	IndexPointNum uint8
}

// NewCueSheet parses and returns a new CueSheet metadata block. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Cue sheet format (pseudo code):
//
//    type METADATA_BLOCK_CUESHEET struct {
//       mcn                  [128]byte
//       lead_in_sample_count uint64
//       is_compact_disc      bool
//       _                    uint7
//       _                    [258]byte
//       track_count          uint8
//       tracks               [track_count]track
//    }
//
//    type track struct {
//       offset            uint64
//       track_num         uint8
//       isrc              [12]byte
//       is_audio          bool
//       has_pre_emphasis  bool
//       _                 uint6
//       _                 [13]byte
//       track_index_count uint8
//       track_indexes     [track_index_count]track_index
//    }
//
//    type track_index {
//       offset          uint64
//       index_point_num uint8
//       _               [3]byte
//    }
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_cuesheet
func NewCueSheet(r io.Reader) (cs *CueSheet, err error) {
	errReservedNotZero := errors.New("meta.NewCueSheet: all reserved bits must be 0")

	// Media catalog number (size: 128 bytes).
	buf, err := readBytes(r, 128)
	if err != nil {
		return nil, err
	}
	cs = new(CueSheet)
	cs.MCN = getStringFromSZ(buf)
	for _, r := range cs.MCN {
		if r < 0x20 || r > 0x7E {
			return nil, fmt.Errorf("meta.NewCueSheet: invalid character in media catalog number; expected >= 0x20 and <= 0x7E, got 0x%02X", r)
		}
	}

	// Lead-in sample count.
	err = binary.Read(r, binary.BigEndian, &cs.LeadInSampleCount)
	if err != nil {
		return nil, err
	}

	const (
		IsCompactDiscMask    = 0x80 // 1 bit
		CueSheetReservedMask = 0x7F // 7 bits
	)
	bits, err := readerutil.ReadByte(r)
	if err != nil {
		return nil, err
	}

	// Is compact disc.
	if bits&IsCompactDiscMask != 0 {
		cs.IsCompactDisc = true
	}

	// Reserved.
	if bits&CueSheetReservedMask != 0 {
		return nil, errReservedNotZero
	}
	buf, err = readBytes(r, 258) // 258 reserved bytes.
	if err != nil {
		return nil, err
	}
	if !isAllZero(buf) {
		return nil, errReservedNotZero
	}

	// Handle error checking of LeadInSampleCount here, since IsCompactDisc is
	// required.
	if !cs.IsCompactDisc && cs.LeadInSampleCount != 0 {
		return nil, fmt.Errorf("meta.NewCueSheet: invalid lead-in sample count for non CD-DA; expected 0, got %d", cs.LeadInSampleCount)
	}

	// Track count.
	err = binary.Read(r, binary.BigEndian, &cs.TrackCount)
	if err != nil {
		return nil, err
	}
	if cs.TrackCount < 1 {
		return nil, errors.New("meta.NewCueSheet: at least one track (the lead-out track) is required")
	}
	if cs.TrackCount > 100 && cs.IsCompactDisc {
		return nil, fmt.Errorf("meta.NewCueSheet: too many tracks for CD-DA cue sheet; expected <= 100, got %d", cs.TrackCount)
	}

	// Tracks.
	cs.Tracks = make([]CueSheetTrack, cs.TrackCount)
	for i := 0; i < len(cs.Tracks); i++ {
		// Track offset.
		track := &cs.Tracks[i]
		err = binary.Read(r, binary.BigEndian, &track.Offset)
		if err != nil {
			return nil, err
		}
		if cs.IsCompactDisc && track.Offset%588 != 0 {
			return nil, fmt.Errorf("meta.NewCueSheet: invalid track offset (%d) for CD-DA; must be evenly divisible by 588", track.Offset)
		}

		// Track number.
		err = binary.Read(r, binary.BigEndian, &track.TrackNum)
		if err != nil {
			return nil, err
		}
		if track.TrackNum == 0 {
			// A track number of 0 is not allowed to avoid conflicting with the
			// CD-DA spec, which reserves this for the lead-in.
			return nil, errors.New("meta.NewCueSheet: track number 0 not allowed")
		}
		if cs.IsCompactDisc {
			if i == len(cs.Tracks)-1 {
				if track.TrackNum != 170 {
					// The lead-out track number must be 170 for CD-DA.
					return nil, fmt.Errorf("meta.NewCueSheet: invalid lead-out track number for CD-DA; expected 170, got %d", track.TrackNum)
				}
			} else if track.TrackNum > 99 {
				return nil, fmt.Errorf("meta.NewCueSheet: invalid track number for CD-DA; expected <= 99, got %d", track.TrackNum)
			}
		} else {
			if i == len(cs.Tracks)-1 && track.TrackNum != 255 {
				// The lead-out track number must be 255 for non-CD-DA.
				return nil, fmt.Errorf("meta.NewCueSheet: invalid lead-out track number for non CD-DA; expected 255, got %d", track.TrackNum)
			}
		}

		// Track ISRC (size: 12 bytes).
		buf, err = readBytes(r, 12)
		if err != nil {
			return nil, err
		}
		track.ISRC = getStringFromSZ(buf)

		const (
			TrackTypeMask      = 0x80 // 1 bit
			HasPreEmphasisMask = 0x40 // 1 bit
			TrackReservedMask  = 0x3F // 6 bits
		)
		bits, err = readerutil.ReadByte(r)
		if err != nil {
			return nil, err
		}

		// Is audio.
		if bits&TrackTypeMask == 0 {
			// track type:
			//    0: audio.
			//    1: non-audio.
			track.IsAudio = true
		}

		// Has pre-emphasis.
		if bits&HasPreEmphasisMask != 0 {
			track.HasPreEmphasis = true
		}

		// Reserved.
		if bits&TrackReservedMask != 0 {
			return nil, errReservedNotZero
		}
		buf, err = readBytes(r, 13) // 13 reserved bytes.
		if err != nil {
			return nil, err
		}
		if !isAllZero(buf) {
			return nil, errReservedNotZero
		}

		// Track index point count.
		err = binary.Read(r, binary.BigEndian, &track.TrackIndexCount)
		if err != nil {
			return nil, err
		}
		if i == len(cs.Tracks)-1 {
			// Lead-out must have zero track index points.
			if track.TrackIndexCount != 0 {
				return nil, fmt.Errorf("meta.NewCueSheet: invalid number of track points for the lead-out track; expected 0, got %d", track.TrackIndexCount)
			}
		} else {
			if track.TrackIndexCount < 1 {
				// Every track, except for the lead-out track, must have at least
				// one track index point.
				return nil, fmt.Errorf("meta.NewCueSheet: invalid number of track points; expected >= 1, got %d", track.TrackIndexCount)
			}
			if cs.IsCompactDisc && track.TrackIndexCount > 100 {
				return nil, fmt.Errorf("meta.NewCueSheet: invalid number of track points for CD-DA; expected <= 100, got %d", track.TrackIndexCount)
			}
		}

		// Track indexes.
		track.TrackIndexes = make([]CueSheetTrackIndex, track.TrackIndexCount)
		for j := 0; j < len(track.TrackIndexes); j++ {
			// Track index point offset.
			trackIndex := &track.TrackIndexes[j]
			err = binary.Read(r, binary.BigEndian, &trackIndex.Offset)
			if err != nil {
				return nil, err
			}

			// Track index point num
			err = binary.Read(r, binary.BigEndian, &trackIndex.IndexPointNum)
			if err != nil {
				return nil, err
			}

			// Reserved.
			buf, err = readBytes(r, 3) // 3 reserved bytes.
			if err != nil {
				return nil, err
			}
			if !isAllZero(buf) {
				return nil, errReservedNotZero
			}
		}
	}

	return cs, nil
}

// getStringFromSZ converts the provided byte slice to a string after
// terminating it at the first occurance of a NULL character.
func getStringFromSZ(buf []byte) string {
	// Locate the first NULL character.
	posNull := bytes.IndexRune(buf, 0)
	if posNull != -1 {
		// Terminate the string at first occurance of a NULL character.
		buf = buf[:posNull]
	}
	return string(buf)
}
