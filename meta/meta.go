// Package meta contains functions for parsing FLAC metadata.
package meta

import "bytes"
import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "strings"

// Formatted error messages.
const (
	ErrInvalidBlockLen            = "invalid block length; expected %d, got %d."
	ErrInvalidMaxBlockSize        = "invalid block size; expected >= 16 and <= 65535, got %d."
	ErrInvalidMinBlockSize        = "invalid block size; expected >= 16, got %d."
	ErrInvalidNumTracksForCompact = "invalid number of tracks for a compact disc; expected <= 100, got %d."
	ErrInvalidPictureType         = "the picture type is invalid (must be <=20): %d"
	ErrInvalidSampleRate          = "invalid sample rate; expected > 0 and <= 655350, got %d."
	ErrMalformedVorbisComment     = "malformed vorbis comment: %s"
	ErrUnregisterdAppID           = "unregistered application id: %s."
)

// Error messages.
var (
	ErrInvalidBlockType    = errors.New("invalid block type.")
	ErrInvalidSeekTableLen = errors.New("invalid block size; seek table not divisible by 18.")
	ErrInvalidTrackNum     = errors.New("invalid track number; value 0 isn't allowed.")
	ErrMissingLeadOutTrack = errors.New("cuesheet requires a lead out track.")
	ErrReserved            = errors.New("reserved value.")
	ErrReservedNotZero     = errors.New("all reserved bits are not 0.")
)

/// Might trigger unnesccesary errors

// isAllZero returns true if the value of each byte in the provided slice is 0,
// and false otherwise.
func isAllZero(buf []byte) bool {
	for _, b := range buf {
		if b != 0 {
			return false
		}
	}
	return true
}

// Type is used to identify the metadata block type.
type Type uint8

// Metadata block types.
const (
	TypeStreamInfo Type = iota
	TypePadding
	TypeApplication
	TypeSeekTable
	TypeVorbisComment
	TypeCueSheet
	TypePicture
)

func (t Type) String() string {
	m := map[Type]string{
		TypeStreamInfo:    "stream info",
		TypePadding:       "padding",
		TypeApplication:   "application",
		TypeSeekTable:     "seek table",
		TypeVorbisComment: "vorbis comment",
		TypeCueSheet:      "cue sheet",
		TypePicture:       "picture",
	}
	return m[t]
}

// A BlockHeader contains type and length about a metadata block.
type BlockHeader struct {
	// IsLast is true if this block is the last metadata block before the audio
	// blocks, and false otherwise.
	IsLast bool
	// Block types:
	//    0: Streaminfo
	//    1: Padding
	//    2: Application
	//    3: Seektable
	//    4: Vorbis_comment
	//    5: Cuesheet
	//    6: Picture
	//    7-126: reserved
	//    127: invalid, to avoid confusion with a frame sync code
	BlockType Type
	// Length (in bytes) of metadata to follow (does not include the size of the
	// BlockHeader).
	Length int
}

// NewBlockHeader parses and returns a new metadata block header. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Block header format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_header
//
//    type METADATA_BLOCK_HEADER struct {
//       var is_last    bool
//       var block_type uint7
//       var length     uint24
//    }
func NewBlockHeader(r io.Reader) (h *BlockHeader, err error) {
	const (
		LastBlockMask = 0x80000000 // 1 bit
		TypeMask      = 0x7F000000 // 7 bits
		LengthMask    = 0x00FFFFFF // 24 bits
	)
	var bits uint32
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Is last.
	h = new(BlockHeader)
	if bits&LastBlockMask != 0 {
		h.IsLast = true
	}

	// Block type.
	h.BlockType = Type(bits & TypeMask >> 24)
	if h.BlockType >= 7 && h.BlockType <= 126 {
		// block type 7-126: reserved.
		return nil, errors.New("meta.NewBlockHeader: Reserved block type.")
	} else if h.BlockType == 127 {
		// block type 127: invalid.
		return nil, errors.New("meta.NewBlockHeader: Invalid block type.")
	}

	// Length.
	h.Length = int(bits & LengthMask) // won't overflow, since max is 0x00FFFFFF.

	return h, nil
}

// A StreamInfo metadata block has information about the entire stream. It must
// be present as the first metadata block in the stream.
type StreamInfo struct {
	// The minimum block size (in samples) used in the stream.
	MinBlockSize uint16
	// The maximum block size (in samples) used in the stream.
	// (MinBlockSize == MaxBlockSize) implies a fixed-blocksize stream.
	MaxBlockSize uint16
	// The minimum frame size (in bytes) used in the stream. May be 0 to imply
	// the value is not known.
	MinFrameSize uint32
	// The maximum frame size (in bytes) used in the stream. May be 0 to imply
	// the value is not known.
	MaxFrameSize uint32
	// Sample rate in Hz. Though 20 bits are available, the maximum sample rate
	// is limited by the structure of frame headers to 655350Hz. Also, a value of
	// 0 is invalid.
	SampleRate uint32
	// Number of channels. FLAC supports from 1 to 8 channels.
	ChannelCount uint8
	// Bits per sample. FLAC supports from 4 to 32 bits per sample. Currently the
	// reference encoder and decoders only support up to 24 bits per sample.
	BitsPerSample uint8
	// Total samples in stream. 'Samples' means inter-channel sample, i.e. one
	// second of 44.1Khz audio will have 44100 samples regardless of the number
	// of channels. A value of zero here means the number of total samples is
	// unknown.
	SampleCount uint64
	// MD5 signature of the unencoded audio data. This allows the decoder to
	// determine if an error exists in the audio data even when the error does
	// not result in an invalid bitstream.
	MD5sum [16]byte
}

// NewStreamInfo parses and returns a new StreamInfo metadata block. The
// provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
//
// Stream info format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_streaminfo
//
//    type METADATA_BLOCK_STREAMINFO struct {
//       var min_block_size  uint16
//       var max_block_size  uint16
//       var min_frame_size  uint24
//       var max_frame_size  uint24
//       var sample_rate     uint20
//       var channel_count   uint3 // (number of channels)-1.
//       var bits_per_sample uint5 // (bits per sample)-1.
//       var sample_count    uint36
//       var md5sum          [16]byte
//    }
func NewStreamInfo(r io.Reader) (si *StreamInfo, err error) {
	// Minimum block size.
	si = new(StreamInfo)
	err = binary.Read(r, binary.BigEndian, &si.MinBlockSize)
	if err != nil {
		return nil, err
	}
	if si.MinBlockSize < 16 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid min block size; expected >= 16, got %d.", si.MinBlockSize)
	}

	const (
		MaxBlockSizeMask = 0xFFFF000000000000 // 16 bits
		MinFrameSizeMask = 0x0000FFFFFF000000 // 24 bits
		MaxFrameSizeMask = 0x0000000000FFFFFF // 24 bits
	)
	// In order to keep everything on powers-of-2 boundaries, reads from the
	// block are grouped accordingly:
	// MaxBlockSize (16 bits) + MinFrameSize (24 bits) + MaxFrameSize (24 bits) =
	// 64 bits
	var bits uint64
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Max block size.
	si.MaxBlockSize = uint16(bits & MaxBlockSizeMask >> 48)
	if si.MaxBlockSize < 16 || si.MaxBlockSize > 65535 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid min block size; expected >= 16 and <= 65535, got %d.", si.MaxBlockSize)
	}

	// Min frame size.
	si.MinFrameSize = uint32(bits & MinFrameSizeMask >> 24)

	// Max frame size.
	si.MaxFrameSize = uint32(bits & MaxFrameSizeMask)

	const (
		SampleRateMask    = 0xFFFFF00000000000 // 20 bits
		ChannelCountMask  = 0x00000E0000000000 // 3 bits
		BitsPerSampleMask = 0x000001F000000000 // 5 bits
		SampleCountMask   = 0x0000000FFFFFFFFF // 36 bits
	)
	// In order to keep everything on powers-of-2 boundaries, reads from the
	// block are grouped accordingly:
	// SampleRate (20 bits) + ChannelCount (3 bits) + BitsPerSample (5 bits) +
	// SampleCount (36 bits) = 64 bits
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Sample rate.
	si.SampleRate = uint32(bits & SampleRateMask >> 44)
	if si.SampleRate > 655350 || si.SampleRate == 0 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid sample rate; expected > 0 and <= 655350, got %d.", si.SampleRate)
	}

	// Both ChannelCount and BitsPerSample are specified to be subtracted by 1 in
	// the specification:
	// http://flac.sourceforge.net/format.html#metadata_block_streaminfo

	// Channel count.
	si.ChannelCount = uint8(bits&ChannelCountMask>>41) + 1
	if si.ChannelCount < 1 || si.ChannelCount > 8 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid number of channels; expected >= 1 and <= 8, got %d.", si.ChannelCount)
	}

	// Bits per sample.
	si.BitsPerSample = uint8(bits&BitsPerSampleMask>>36) + 1
	if si.BitsPerSample < 4 || si.BitsPerSample > 32 {
		return nil, fmt.Errorf("meta.NewStreamInfo: invalid number of bits per sample; expected >= 4 and <= 32, got %d.", si.BitsPerSample)
	}

	// Sample count.
	si.SampleCount = bits & SampleCountMask

	// Md5sum MD5 signature of unencoded audio data.
	_, err = io.ReadFull(r, si.MD5sum[:])
	if err != nil {
		return nil, err
	}
	return si, nil
}

// VerifyPadding verifies that the padding metadata block only contains '0'
// bits.
func VerifyPadding(r io.Reader) (err error) {
	// Verify up to 4 kb of padding each iteration.
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !isAllZero(buf[:n]) {
			return errors.New("meta.VerifyPadding: invalid padding; must contain only zeroes.")
		}
	}
	return nil
}

// RegisteredApplications maps from a registered application ID to a
// description.
//
// ref: http://flac.sourceforge.net/id.html
var RegisteredApplications = map[string]string{
	"ATCH": "FlacFile",
	"BSOL": "beSolo",
	"BUGS": "Bugs Player",
	"Cues": "GoldWave cue points (specification)",
	"Fica": "CUE Splitter",
	"Ftol": "flac-tools",
	"MOTB": "MOTB MetaCzar",
	"MPSE": "MP3 Stream Editor",
	"MuML": "MusicML: Music Metadata Language",
	"RIFF": "Sound Devices RIFF chunk storage",
	"SFFL": "Sound Font FLAC",
	"SONY": "Sony Creative Software",
	"SQEZ": "flacsqueeze",
	"TtWv": "TwistedWave",
	"UITS": "UITS Embedding tools",
	"aiff": "FLAC AIFF chunk storage",
	"imag": "flac-image application for storing arbitrary files in APPLICATION metadata blocks",
	"peem": "Parseable Embedded Extensible Metadata (specification)",
	"qfst": "QFLAC Studio",
	"riff": "FLAC RIFF chunk storage",
	"tune": "TagTuner",
	"xbat": "XBAT",
	"xmcd": "xmcd",
}

// An Application metadata block is for use by third-party applications. The
// only mandatory field is a 32-bit identifier. This ID is granted upon request
// to an application by the FLAC maintainers. The remainder of the block is
// defined by the registered application.
type Application struct {
	// Registered application ID.
	ID string
	// Application data.
	Data []byte ///interface{} type instead?
}

// NewApplication parses and returns a new Application metadata block. The
// provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
//
// Application format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_application
//
//    type METADATA_BLOCK_APPLICATION struct {
//       var ID   uint32
//       var Data [header.Length-4]byte
//    }
func NewApplication(buf []byte) (ap *Application, err error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("invalid block size; expected >= 4, got %d.", len(buf))
	}

	ap = new(Application)
	b := bytes.NewBuffer(buf)

	// Application ID (size: 4 bytes).
	ap.ID = string(b.Next(4))
	_, ok := RegisteredApplications[ap.ID]
	if !ok {
		return nil, fmt.Errorf(ErrUnregisterdAppID, ap.ID)
	}

	ap.Data = b.Bytes()

	///Make uber switch case for all applications
	// switch ap.ID {

	// }

	return ap, nil
}

// A SeekTable metadata block is an optional block for storing seek points. It
// is possible to seek to any given sample in a FLAC stream without a seek
// table, but the delay can be unpredictable since the bitrate may vary widely
// within a stream. By adding seek points to a stream, this delay can be
// significantly reduced. Each seek point takes 18 bytes, so 1% resolution
// within a stream adds less than 2k.
//
// There can be only one SEEKTABLE in a stream, but the table can have any
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
// placeholder points, the second and third field values are undefined.
const PlaceholderPoint = 0xFFFFFFFFFFFFFFFF

// NewSeekTable parses and returns a new SeekTable metadata block. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Seek table format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_seektable
//
//    type METADATA_BLOCK_SEEKTABLE struct {
//       // The number of seek points is implied by the metadata header 'length'
//       // field, i.e. equal to length / 18.
//       var points []point
//    }
//
//    type point struct {
//       var sample_num   uint64
//       var offset       uint64
//       var sample_count uint16
//    }
func NewSeekTable(r io.Reader) (st *SeekTable, err error) {
	st = new(SeekTable)
	var hasPrev bool
	var prevSampleNum uint64
	for {
		var point SeekPoint
		err = binary.Read(r, binary.BigEndian, &point)
		if err != nil {
			if err == io.EOF {
				return st, nil
			}
			return nil, err
		}
		if hasPrev && prevSampleNum >= point.SampleNum {
			// - Seek points within a table must be sorted in ascending order by
			//   sample number.
			// - Seek points within a table must be unique by sample number, with
			//   the exception of placeholder points.
			// - The previous two notes imply that there may be any number of
			//   placeholder points, but they must all occur at the end of the
			//   table.
			if point.SampleNum != PlaceholderPoint {
				return nil, fmt.Errorf("meta.NewSeekTable: invalid seek point; sample number (%d) not in ascending order.", point.SampleNum)
			}
		}
		prevSampleNum = point.SampleNum
		hasPrev = true
		st.Points = append(st.Points, point)
	}
	return st, nil
}

// A VorbisComment metadata block is for storing a list of human-readable
// name/value pairs. Values are encoded using UTF-8. It is an implementation of
// the Vorbis comment specification (without the framing bit). This is the only
// officially supported tagging mechanism in FLAC. There may be only one
// VORBIS_COMMENT block in a stream. In some external documentation, Vorbis
// comments are called FLAC tags to lessen confusion.
type VorbisComment struct {
	Vendor  string
	Entries []VorbisEntry
}

// A VorbisEntry is a name/value pair.
type VorbisEntry struct {
	Name  string
	Value string
}

// NewVorbisComment parses and returns a new VorbisComment metadata block. The
// provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
//
// Vorbis comment format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_vorbis_comment
//
//    type METADATA_BLOCK_VORBIS_COMMENT struct {
//       var vendor_length uint32
//       var vendor_string [vendor_length]byte
//       var comment_count uint32
//       var comments      [comment_count]comment
//    }
//
//    type comment struct {
//       var vector_length uint32
//       // vector_string is a name/value pair. Example: "NAME=value".
//       var vector_string [length]byte
//    }
func NewVorbisComment(r io.Reader) (vc *VorbisComment, err error) {
	// Vendor length.
	var vendorLen uint32
	err = binary.Read(r, binary.LittleEndian, &vendorLen)
	if err != nil {
		return nil, err
	}

	// Vendor string.
	buf := make([]byte, vendorLen)
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}
	vc = new(VorbisComment)
	vc.Vendor = string(buf)

	// Comment count.
	var commentCount uint32
	err = binary.Read(r, binary.LittleEndian, &commentCount)
	if err != nil {
		return nil, err
	}

	// Comments.
	vc.Entries = make([]VorbisEntry, commentCount)
	for i := 0; i < len(vc.Entries); i++ {
		// Vector length
		var vectorLen uint32
		err = binary.Read(r, binary.LittleEndian, &vectorLen)
		if err != nil {
			return nil, err
		}

		// Vector string.
		buf = make([]byte, vectorLen)
		_, err = r.Read(buf)
		if err != nil {
			return nil, err
		}
		vector := string(buf)
		pos := strings.Index(vector, "=")
		if pos == -1 {
			return nil, fmt.Errorf("meta.NewVorbisComment: invalid comment vector; no '=' present in: %s.", vector)
		}

		// Comment.
		entry := VorbisEntry{
			Name:  vector[:pos],
			Value: vector[pos+1:],
		}
		vc.Entries[i] = entry
	}
	return vc, nil
}

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
	MCN []byte
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
	ISRC []byte
	// The track type: false for audio, true for non-audio. This corresponds to
	// the CD-DA Q-channel control bit 3.
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
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_cuesheet
//
//    type METADATA_BLOCK_CUESHEET struct {
//       var mcn                  [128]byte
//       var lead_in_sample_count uint64
//       var is_compact_disc      bool
//       var _                    uint7
//       var _                    [258]byte
//       var track_count          uint8
//       var tracks               [track_count]track
//    }
//
//    type track struct {
//       var offset            uint64
//       var track_num         uint8
//       var isrc              [12]byte
//       var is_audio          bool
//       var has_pre_emphasis  bool
//       var _                 uint6
//       var _                 [13]byte
//       var track_index_count uint8
//       var track_indexes     [track_index_count]track_index
//    }
//
//    type track_index {
//       var offset          uint64
//       var index_point_num uint8
//       var _               [3]byte
//    }
func NewCueSheet(buf []byte) (cs *CueSheet, err error) {
	// Minimum valid size based on CueSheet with one lead-out track:
	// len(METADATA_BLOCK_CUESHEET) + len(CUESHEET_TRACK) = 432
	if len(buf) < 432 {
		return nil, fmt.Errorf("invalid block size; expected >= 432, got %d.", len(buf))
	}

	cs = new(CueSheet)
	b := bytes.NewBuffer(buf)

	// Media catalog number (size: 128 bytes).
	cs.MCN = b.Next(128)

	// The number of lead-in samples (size: 8 bytes).
	cs.LeadInSampleCount = binary.BigEndian.Uint64(b.Next(8))

	const (
		// CueSheet
		IsCompactDiscMask    = 0x80
		CueSheetReservedMask = 0x7F
	)

	// 1 bit for IsCompactDisk boolean and 7 bits are reserved.
	bits, err := b.ReadByte()
	if err != nil {
		return nil, err
	}

	if bits&IsCompactDiscMask != 0 {
		cs.IsCompactDisc = true
	}

	// Reserved
	if bits&CueSheetReservedMask != 0 {
		return nil, ErrReservedNotZero
	}

	if !isAllZero(b.Next(258)) {
		return nil, ErrReservedNotZero
	}

	// The number of tracks (size: 1 byte).
	cs.TrackCount, err = b.ReadByte()
	if err != nil {
		return nil, err
	}
	if cs.TrackCount < 1 {
		return nil, ErrMissingLeadOutTrack
	} else if cs.TrackCount > 100 && cs.IsCompactDisc {
		return nil, fmt.Errorf(ErrInvalidNumTracksForCompact, cs.TrackCount)
	}

	// Minimum valid size of Tracks:
	// len(CUESHEET_TRACK) + (TrackCount-1)*len(CUESHEET_TRACK_INDEX) =
	// 36 + (TrackCount-1)*12
	TracksMinSize := int(36 + (cs.TrackCount-1)*12)
	if b.Len() < TracksMinSize {
		return nil, fmt.Errorf("invalid block size; expected >= %d, got %d.", TracksMinSize, b.Len())
	}
	for trackNum := 0; trackNum < int(cs.TrackCount); trackNum++ {
		ct := new(CueSheetTrack)

		// Track offset in samples (size: 8 bytes).
		ct.Offset = binary.BigEndian.Uint64(b.Next(8))

		// Track number (size: 1 byte).
		ct.TrackNum, err = b.ReadByte()
		if err != nil {
			return nil, err
		}
		if ct.TrackNum == 0 {
			return nil, ErrInvalidTrackNum
		}

		// Track ISRC (size: 12 bytes)
		ct.ISRC = b.Next(12)

		const (
			// CueSheetTrack
			IsAudioMask               = 0x80
			HasPreEmphasisMask        = 0x40
			CueSheetTrackReservedMask = 0x3F
		)

		bits, err := b.ReadByte()
		if err != nil {
			return nil, err
		}

		if bits&IsAudioMask != 0 {
			ct.IsAudio = true
		}

		if bits&HasPreEmphasisMask != 0 {
			ct.HasPreEmphasis = true
		}

		if bits&CueSheetTrackReservedMask != 0 {
			return nil, ErrReservedNotZero
		}

		// Reserved (size: 13 bytes + 6 bits from last byte).
		if !isAllZero(b.Next(13)) {
			return nil, ErrReservedNotZero
		}

		// Number of track index points (size: 1 byte).
		ct.TrackIndexCount, err = b.ReadByte()
		if err != nil {
			return nil, err
		}
		if trackNum == int(cs.TrackCount-1) {
			// The lead-out track is always the last track in the CUESHEET. It must
			// have zero track index points.
			if ct.TrackIndexCount != 0 {
				return nil, fmt.Errorf("invalid number of track index points in cuesheet lead-out track; expected 0 got %d.", ct.TrackIndexCount)
			}
		} else {
			// There must be at least one index in every track in a CUESHEET except
			// for the lead-out track.
			if ct.TrackIndexCount < 1 {
				return nil, fmt.Errorf("invalid cuesheet track, too few index points; expected >= 1, got %d.", ct.TrackIndexCount)
			}
		}

		// Minimum valid size of TrackIndexes:
		// len(CUESHEET_TRACK_INDEX)*TrackIndexCount = 12*TrackIndexCount
		TrackIndexesMinSize := int(12 * ct.TrackIndexCount)
		if b.Len() < TrackIndexesMinSize {
			return nil, fmt.Errorf("invalid size of TrackIndexes; expected >= %d, got %d.", TrackIndexesMinSize, b.Len())
		}
		ct.TrackIndexes = make([]CueSheetTrackIndex, ct.TrackIndexCount)
		for i := 0; i < len(ct.TrackIndexes); i++ {
			trackIndex := CueSheetTrackIndex{
				Offset: binary.BigEndian.Uint64(b.Next(8)), // Offset in samples (size: 8 bytes)
			}
			// The index point number (size: 1 byte).
			trackIndex.IndexPointNum, err = b.ReadByte()
			if err != nil {
				return nil, err
			}
			// Reserved (size: 3 bytes).
			if !isAllZero(b.Next(3)) {
				return nil, ErrReservedNotZero
			}
			ct.TrackIndexes[i] = trackIndex
		}
	}

	return cs, nil
}

// A Picture metadata block is for storing pictures associated with the file,
// most commonly cover art from CDs. There may be more than one PICTURE block in
// a file.
type Picture struct {
	// The picture type according to the ID3v2 APIC frame:
	//    0 - Other
	//    1 - 32x32 pixels 'file icon' (PNG only)
	//    2 - Other file icon
	//    3 - Cover (front)
	//    4 - Cover (back)
	//    5 - Leaflet page
	//    6 - Media (e.g. label side of CD)
	//    7 - Lead artist/lead performer/soloist
	//    8 - Artist/performer
	//    9 - Conductor
	//    10 - Band/Orchestra
	//    11 - Composer
	//    12 - Lyricist/text writer
	//    13 - Recording Location
	//    14 - During recording
	//    15 - During performance
	//    16 - Movie/video screen capture
	//    17 - A bright coloured fish
	//    18 - Illustration
	//    19 - Band/artist logotype
	//    20 - Publisher/Studio logotype
	//
	// Others are reserved and should not be used. There may only be one each of
	// picture type 1 and 2 in a file.
	Type       uint32
	// The MIME type string, in printable ASCII characters 0x20-0x7e. The MIME
	// type may also be --> to signify that the data part is a URL of the picture
	// instead of the picture data itself.
	MIME       string
	// The description of the picture, in UTF-8.
	PicDesc    string
	// The width of the picture in pixels.
	Width      uint32
	// The height of the picture in pixels.
	Height     uint32
	// The color depth of the picture in bits-per-pixel.
	ColorDepth uint32
	// For indexed-color pictures (e.g. GIF), the number of colors used, or 0 for
	// non-indexed pictures.
	ColorCount uint32
	// The binary picture data.
	Data       []byte
}

// NewPicture parses and returns a new Picture metadata block. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Picture format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_picture
//
//    type METADATA_BLOCK_PICTURE struct {
//       var type        uint32
//       var mime_length uint32
//       var mime_string [mime_length]byte
//       var desc_length uint32
//       var desc_string [desc_length]byte
//       var width       uint32
//       var height      uint32
//       var color_depth uint32
//       var color_count uint32
//       var data_length uint32
//       var data        [data_length]byte
//    }
func NewPicture(buf []byte) (p *Picture, err error) {
	p = new(Picture)
	b := bytes.NewBuffer(buf)

	///Check for multiple pictures of the same type

	//A list of allowed picture types
	// 0 - Other
	// 1 - 32x32 pixels 'file icon' (PNG only)
	// 2 - Other file icon
	// 3 - Cover (front)
	// 4 - Cover (back)
	// 5 - Leaflet page
	// 6 - Media (e.g. label side of CD)
	// 7 - Lead artist/lead performer/soloist
	// 8 - Artist/performer
	// 9 - Conductor
	// 10 - Band/Orchestra
	// 11 - Composer
	// 12 - Lyricist/text writer
	// 13 - Recording Location
	// 14 - During recording
	// 15 - During performance
	// 16 - Movie/video screen capture
	// 17 - A bright coloured fish
	// 18 - Illustration
	// 19 - Band/artist logotype
	// 20 - Publisher/Studio logotype

	//Picture type (size: 4 bytes)
	p.Type = binary.BigEndian.Uint32(b.Next(4))
	if p.Type > 20 {
		return nil, fmt.Errorf(ErrInvalidPictureType, p.Type)
	}

	//Length of the mime type (size: 4 bytes), Mime type string (size: depends on length)
	p.MIME = string(b.Next(int(binary.BigEndian.Uint32(b.Next(4)))))

	//Length of the Picture description (size: 4 bytes), Description string (size: depends on length)
	p.PicDesc = string(b.Next(int(binary.BigEndian.Uint32(b.Next(4)))))

	p.Width = binary.BigEndian.Uint32(b.Next(4))
	p.Height = binary.BigEndian.Uint32(b.Next(4))
	p.ColorDepth = binary.BigEndian.Uint32(b.Next(4))
	p.ColorCount = binary.BigEndian.Uint32(b.Next(4))

	//Length of the Picture data (size: 4 bytes), Picture data (size: depends on length)
	p.Data = b.Next(int(binary.BigEndian.Uint32(b.Next(4))))

	return p, nil
}
