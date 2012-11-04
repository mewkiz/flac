// Package meta contains functions for parsing FLAC metadata.
package meta

import "bytes"
import "encoding/binary"
import "errors"
import "fmt"
import "strings"

// Formatted error messages.
const (
	ErrInvalidBlockLen            = "invalid block length; expected %d, got %d."
	ErrInvalidMaxBlockSize        = "invalid block size; expected >= 16 and <= 65535, got %d."
	ErrInvalidMinBlockSize        = "invalid block size; expected >= 16, got %d."
	ErrInvalidNumTracksForCompact = "invalid number of tracks for a compact disc, can't be more than 100: %d"
	ErrInvalidPictureType         = "the picture type is invalid (must be <=20): %d"
	ErrInvalidSampleRate          = "invalid sample rate; expected > 0 and <= 655350, got %d."
	///ErrInvalidSyncCode            = "sync code is invalid (must be 11111111111110 or 16382 decimal): %d"
	ErrMalformedVorbisComment = "malformed vorbis comment: %s"
	ErrUnregisterdAppID       = "unregistered application id: %s."
)

// Error messages.
var (
	ErrInvalidBlockType    = errors.New("invalid block type.")
	ErrInvalidSeekTableLen = errors.New("invalid block size; seek table not divisible by 18.")
	ErrInvalidTrackNum     = errors.New("invalid track number value 0 isn't allowed.")
	ErrMissingLeadOutTrack = errors.New("cuesheet needs a lead out track.")
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

// NewBlockHeader parses and returns a new metadata block header.
func NewBlockHeader(buf []byte) (h *BlockHeader, err error) {
	const (
		LastBlockMask = 0x80000000
		TypeMask      = 0x7F000000
		LengthMask    = 0x00FFFFFF
	)

	if len(buf) != 4 {
		return nil, fmt.Errorf(ErrInvalidBlockLen, 4, len(buf))
	}

	h = new(BlockHeader)
	bits := binary.BigEndian.Uint32(buf)

	// Check if this is the last metadata block.
	if bits&LastBlockMask != 0 {
		h.IsLast = true
	}

	h.BlockType = Type(bits & TypeMask >> 24)
	h.Length = int(bits & LengthMask) // won't overflow, since max is 0x00FFFFFF.

	if h.BlockType >= 7 && h.BlockType <= 126 {
		// block type 7-126: reserved
		return nil, ErrReserved
	} else if h.BlockType == 127 {
		// block type 127: invalid, to avoid confusion with a frame sync code
		return nil, ErrInvalidBlockType
	}

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
	// (number of channels)-1. FLAC supports from 1 to 8 channels.
	NumChannels uint8
	// (bits per sample)-1. FLAC supports from 4 to 32 bits per sample. Currently
	// the reference encoder and decoders only support up to 24 bits per sample.
	BitsPerSample uint8
	// Total samples in stream. 'Samples' means inter-channel sample, i.e. one
	// second of 44.1Khz audio will have 44100 samples regardless of the number
	// of channels. A value of zero here means the number of total samples is
	// unknown.
	NumSamples uint64
	// MD5 signature of the unencoded audio data. This allows the decoder to
	// determine if an error exists in the audio data even when the error does
	// not result in an invalid bitstream.
	MD5 []byte
}

// NewStreamInfo parses and returns a new StreamInfo metadata block.
func NewStreamInfo(buf []byte) (si *StreamInfo, err error) {
	// A StreamInfo block is always 34 bytes.
	if len(buf) != 34 {
		return nil, fmt.Errorf(ErrInvalidBlockLen, 34, len(buf))
	}

	si = new(StreamInfo)
	b := bytes.NewBuffer(buf)

	// Minimum block size (size: 2 bytes).
	si.MinBlockSize = binary.BigEndian.Uint16(b.Next(2))
	if si.MinBlockSize < 16 {
		return nil, fmt.Errorf(ErrInvalidMinBlockSize, si.MinBlockSize)
	}

	const (
		MaxBlockSizeMask = 0xFFFF000000000000 // 16 bits
		MinFrameSizeMask = 0x0000FFFFFF000000 // 24 bits
		MaxFrameSizeMask = 0x0000000000FFFFFF // 24 bits
	)

	// In order to keep everything on powers-of-2 boundaries, reads from the
	// block are grouped thus:
	// MaxBlockSize (16 bits) + MinFrameSize (24 bits) + MaxFrameSize (24 bits) =
	// 64 bits
	bits := binary.BigEndian.Uint64(b.Next(8))

	si.MaxBlockSize = uint16(bits & MaxBlockSizeMask >> 48)
	if si.MaxBlockSize < 16 || si.MaxBlockSize > 65535 {
		return nil, fmt.Errorf(ErrInvalidMaxBlockSize, si.MaxBlockSize)
	}

	si.MinFrameSize = uint32(bits & MinFrameSizeMask >> 24)
	si.MaxFrameSize = uint32(bits & MaxFrameSizeMask)

	const (
		SampleRateMask    = 0xFFFFF00000000000 // 20 bits
		NumChannelsMask   = 0x00000E0000000000 // 3 bits
		BitsPerSampleMask = 0x000001F000000000 // 5 bits
		NumSamplesMask    = 0x0000000FFFFFFFFF // 36 bits
	)

	// In order to keep everything on powers-of-2 boundaries, reads from the
	// block are grouped thus:
	// SampleRate (20 bits) + NumChannels (3 bits) + BitsPerSample (5 bits) +
	// NumSamples (36 bits) = 64 bits
	bits = binary.BigEndian.Uint64(b.Next(8))

	si.SampleRate = uint32(bits & SampleRateMask >> 44)
	if si.SampleRate > 655350 || si.SampleRate == 0 {
		return nil, fmt.Errorf(ErrInvalidSampleRate, si.SampleRate)
	}

	// Both NumChannels and BitsPerSample are specified to be subtracted by 1 in
	// the specification:
	// http://flac.sourceforge.net/format.html#metadata_block_streaminfo
	si.NumChannels = uint8(bits&NumChannelsMask>>41) + 1
	si.BitsPerSample = uint8(bits&BitsPerSampleMask>>36) + 1

	si.NumSamples = bits & NumSamplesMask

	// MD5 signature of unencoded audio data (size: 16 bytes).
	si.MD5 = b.Next(16)

	return si, nil
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

// NewApplication parses and returns a new Application metadata block.
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
	SampleNumber uint64
	// Offset (in bytes) from the first byte of the first frame header to the
	// first byte of the target frame's header.
	Offset uint64
	// Number of samples in the target frame.
	NumSamples uint16
}

// PlaceholderPoint is the sample number used for placeholder points.
const PlaceholderPoint = 0xFFFFFFFFFFFFFFFF

// NewSeekTable parses and returns a new SeekTable metadata block.
func NewSeekTable(buf []byte) (st *SeekTable, err error) {
	if len(buf)%18 != 0 {
		return nil, ErrInvalidSeekTableLen
	}

	st = new(SeekTable)
	b := bytes.NewBuffer(buf)
	numSeekPoints := len(buf) / 18

	// - For placeholder points, the second and third field values are undefined.
	// - Seek points within a table must be sorted in ascending order by sample
	//   number.
	// - Seek points within a table must be unique by sample number, with the
	//   exception of placeholder points.
	// - The previous two notes imply that there may be any number of placeholder
	//   points, but they must all occur at the end of the table.
	var prevSampleNumber uint64
	for i := 0; i < numSeekPoints; i++ {
		point := SeekPoint{
			SampleNumber: binary.BigEndian.Uint64(b.Next(8)), // Sample number (size: 8 bytes).
			Offset:       binary.BigEndian.Uint64(b.Next(8)), // Offset  (size: 8 bytes).
			NumSamples:   binary.BigEndian.Uint16(b.Next(2)), // Number of samples (size: 2 bytes).
		}
		if prevSampleNumber >= point.SampleNumber && i != 0 {
			if point.SampleNumber != PlaceholderPoint {
				return nil, fmt.Errorf("invalid seek point; sample number (%d) not in ascending order.", point.SampleNumber)
			}
		}
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

// NewVorbisComment parses and returns a new VorbisComment metadata block.
func NewVorbisComment(buf []byte) (vc *VorbisComment, err error) {
	b := bytes.NewBuffer(buf)

	vc = new(VorbisComment)

	//Vendor string (size: determined by previous 4 bytes)
	vc.Vendor = string(b.Next(int(binary.LittleEndian.Uint32(b.Next(4)))))

	//Number of comments (size: 4 bytes)
	userCommentListLength := binary.LittleEndian.Uint32(b.Next(4))

	for i := 0; i < int(userCommentListLength); i++ {
		///This might fail on `=a` strings or simply `=` strings

		//The `TYPE=Value` string (size: determined by previous 4 bytes)
		comment := string(b.Next(int(binary.LittleEndian.Uint32(b.Next(4)))))

		if !strings.Contains(comment, `=`) {
			return nil, fmt.Errorf(ErrMalformedVorbisComment, comment)
		}

		//Split at first occurence of `=`
		nameAndValue := strings.SplitN(comment, "=", 2)

		vc.Entries = append(vc.Entries, VorbisEntry{Name: nameAndValue[0], Value: nameAndValue[1]})
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
	CatalogNum       []byte
	NumLeadInSamples uint64
	IsCompactDisc    bool
	NumTracks        uint8
	Tracks           []CueSheetTrack
}

// A CueSheetTrack contains information about a track within a CueSheet.
type CueSheetTrack struct {
	Offset              uint64
	TrackNum            uint8
	ISRC                []byte
	IsAudio             bool
	HasPreEmphasis      bool
	NumTrackIndexPoints uint8
	TrackIndexes        []CueSheetTrackIndex
}

// A CueSheetTrackIndex contains information about an index point in a track.
type CueSheetTrackIndex struct {
	Offset        uint64
	IndexPointNum uint8
}

// NewCueSheet parses and returns a new CueSheet metadata block.
func NewCueSheet(buf []byte) (cs *CueSheet, err error) {

	const (
		//CueSheet
		IsCompactDiscMask    = 0x80
		CueSheetReservedMask = 0x7F

		//CueSheetTrack
		IsAudioMask               = 0x80
		HasPreEmphasisMask        = 0x40
		CueSheetTrackReservedMask = 0x3F
	)

	cs = new(CueSheet)
	b := bytes.NewBuffer(buf)

	//Media catalog number (size: 128 bytes)
	cs.CatalogNum = b.Next(128)

	//The number of lead-in samples (size: 8 bytes)
	cs.NumLeadInSamples = binary.BigEndian.Uint64(b.Next(8))

	//1 bit for IsCompactDisk boolean and 7 bits are reserved.
	bits := uint8(b.Next(1)[0])

	if bits&IsCompactDiscMask != 0 {
		cs.IsCompactDisc = true
	}

	//Reserved
	if bits&CueSheetReservedMask != 0 {
		return nil, ErrReservedNotZero
	}

	if !isAllZero(b.Next(258)) {
		return nil, ErrReservedNotZero
	}

	//The number of tracks (size: 1 byte)
	cs.NumTracks = uint8(b.Next(1)[0])
	if cs.NumTracks < 1 {
		return nil, ErrMissingLeadOutTrack
	} else if cs.NumTracks > 100 && cs.IsCompactDisc {
		return nil, fmt.Errorf(ErrInvalidNumTracksForCompact, cs.NumTracks)
	}

	for i := 0; i < int(cs.NumTracks); i++ {
		ct := new(CueSheetTrack)

		//Track offset in samples (size: 8 bytes)
		ct.Offset = binary.BigEndian.Uint64(b.Next(8))

		//Track number (size: 1 byte)
		ct.TrackNum = uint8(b.Next(1)[0])

		if ct.TrackNum == 0 {
			return nil, ErrInvalidTrackNum
		}

		//Track ISRC (size: 12 bytes)
		ct.ISRC = b.Next(12)

		bits := uint8(b.Next(1)[0])

		//Is track audio (size: 1 bit)
		if bits&IsAudioMask != 0 {
			ct.IsAudio = true
		}

		//Has pre emphasis (size: 1 bit)
		if bits&HasPreEmphasisMask != 0 {
			ct.HasPreEmphasis = true
		}

		if bits&CueSheetTrackReservedMask != 0 {
			return nil, ErrReservedNotZero
		}

		//Reserved (size: 13 bytes + 6 bits from last byte)
		if !isAllZero(b.Next(13)) {
			return nil, ErrReservedNotZero
		}

		///Must be at least 1 on regular but must be 0 at lead out
		//Number of track index points (size: 1 byte)
		ct.NumTrackIndexPoints = uint8(b.Next(1)[0])

		for i := 0; i < int(ct.NumTrackIndexPoints); i++ {
			ct.TrackIndexes = append(ct.TrackIndexes, CueSheetTrackIndex{
				Offset:        binary.BigEndian.Uint64(b.Next(8)), //Offset in samples (size: 8 bytes)
				IndexPointNum: uint8(b.Next(1)[0]),                //The index point number (size: 1 byte) ///Help with uint8
			})

			///All bits must be zero
			//Reserved (size: 3 bytes)
			if !isAllZero(b.Next(3)) {
				return nil, ErrReservedNotZero
			}
		}
	}

	return cs, nil
}

// A Picture metadata block is for storing pictures associated with the file,
// most commonly cover art from CDs. There may be more than one PICTURE block in
// a file.
type Picture struct {
	Type       uint32
	MIME       string
	PicDesc    string
	Width      uint32
	Height     uint32
	ColorDepth uint32
	NumColors  uint32
	Data       []byte
}

// NewPicture parses and returns a new Picture metadata block.
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
	p.NumColors = binary.BigEndian.Uint32(b.Next(4))

	//Length of the Picture data (size: 4 bytes), Picture data (size: depends on length)
	p.Data = b.Next(int(binary.BigEndian.Uint32(b.Next(4))))

	return p, nil
}
