// Package meta contains functions for parsing FLAC metadata.
package meta

import "bytes"
import "encoding/binary"
import "errors"
import "fmt"
import "io"
import "io/ioutil"
import "strings"

import "github.com/mewkiz/pkg/readerutil"

// A Block is a metadata block, consisting of a block header and a body.
type Block struct {
	// Metadata block header.
	Header *BlockHeader
	// Metadata block body: StreamInfo, Application, SeekTable, etc.
	Body interface{}
}

// NewBlock parses and returns a new metadata block, which consists of a header
// and body.
func NewBlock(r io.Reader) (block *Block, err error) {
	// Read metadata block header.
	block = new(Block)
	block.Header, err = NewBlockHeader(r)
	if err != nil {
		return nil, err
	}

	// Read metadata block.
	lr := io.LimitReader(r, int64(block.Header.Length))
	switch block.Header.BlockType {
	case TypeStreamInfo:
		block.Body, err = NewStreamInfo(lr)
	case TypePadding:
		err = VerifyPadding(lr)
	case TypeApplication:
		block.Body, err = NewApplication(lr)
	case TypeSeekTable:
		block.Body, err = NewSeekTable(lr)
	case TypeVorbisComment:
		block.Body, err = NewVorbisComment(lr)
	case TypeCueSheet:
		block.Body, err = NewCueSheet(lr)
	case TypePicture:
		block.Body, err = NewPicture(lr)
	default:
		return nil, fmt.Errorf("meta.NewBlock: block type '%d' not yet supported.", block.Header.BlockType)
	}
	if err != nil {
		return nil, err
	}

	return block, nil
}

// BlockType is used to identify the metadata block type.
type BlockType uint8

// Metadata block types.
const (
	TypeStreamInfo BlockType = iota
	TypePadding
	TypeApplication
	TypeSeekTable
	TypeVorbisComment
	TypeCueSheet
	TypePicture
)

func (t BlockType) String() string {
	m := map[BlockType]string{
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
	BlockType BlockType
	// Length (in bytes) of metadata to follow (does not include the size of the
	// BlockHeader).
	Length int
}

// NewBlockHeader parses and returns a new metadata block header.
//
// Block header format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_header
//
//    type METADATA_BLOCK_HEADER struct {
//       is_last    bool
//       block_type uint7
//       length     uint24
//    }
func NewBlockHeader(r io.Reader) (h *BlockHeader, err error) {
	const (
		IsLastMask = 0x80000000 // 1 bit
		TypeMask   = 0x7F000000 // 7 bits
		LengthMask = 0x00FFFFFF // 24 bits
	)
	var bits uint32
	err = binary.Read(r, binary.BigEndian, &bits)
	if err != nil {
		return nil, err
	}

	// Is last.
	h = new(BlockHeader)
	if bits&IsLastMask != 0 {
		h.IsLast = true
	}

	// Block type.
	h.BlockType = BlockType(bits & TypeMask >> 24)
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
//       min_block_size  uint16
//       max_block_size  uint16
//       min_frame_size  uint24
//       max_frame_size  uint24
//       sample_rate     uint20
//       channel_count   uint3 // (number of channels)-1.
//       bits_per_sample uint5 // (bits per sample)-1.
//       sample_count    uint36
//       md5sum          [16]byte
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

// VerifyPadding verifies that the padding metadata block only contains 0 bits.
// The provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
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

/// ### [ note ] ###
///    - Might trigger unnecessary errors.
/// ### [/ note ] ###

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
	Data []byte
}

// NewApplication parses and returns a new Application metadata block. The
// provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
//
// Application format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_application
//
//    type METADATA_BLOCK_APPLICATION struct {
//       ID   uint32
//       Data [header.Length-4]byte
//    }
func NewApplication(r io.Reader) (app *Application, err error) {
	// Application ID (size: 4 bytes).
	buf := make([]byte, 4)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	app = new(Application)
	app.ID = string(buf)
	_, ok := RegisteredApplications[app.ID]
	if !ok {
		return nil, fmt.Errorf("meta.NewApplication: unregistered application ID '%s'.", app.ID)
	}

	// Data.
	buf, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	app.Data = buf

	return app, nil
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
// placeholder points, the second and third field values in the SeekPoint
// structure are undefined.
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
//       points []point
//    }
//
//    type point struct {
//       sample_num   uint64
//       offset       uint64
//       sample_count uint16
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
//       vendor_length uint32
//       vendor_string [vendor_length]byte
//       comment_count uint32
//       comments      [comment_count]comment
//    }
//
//    type comment struct {
//       vector_length uint32
//       // vector_string is a name/value pair. Example: "NAME=value".
//       vector_string [length]byte
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
		_, err = io.ReadFull(r, buf)
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

// NewCueSheet parses and returns a new CueSheet metadata block. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Cue sheet format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_cuesheet
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
func NewCueSheet(r io.Reader) (cs *CueSheet, err error) {
	errReservedNotZero := errors.New("meta.NewCueSheet: all reserved bits must be 0.")

	// Media catalog number (size: 128 bytes).
	buf := make([]byte, 128)
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}
	cs = new(CueSheet)
	cs.MCN = getStringFromSZ(buf)
	for _, r := range cs.MCN {
		if r < 0x20 || r > 0x7E {
			return nil, fmt.Errorf("meta.NewCueSheet: invalid character in media catalog number; expected >= 0x20 and <= 0x7E, got 0x%02X.", r)
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
	buf = make([]byte, 258)
	_, err = r.Read(buf) // 258 reserved bytes.
	if err != nil {
		return nil, err
	}
	if !isAllZero(buf) {
		return nil, errReservedNotZero
	}

	// Handle error checking of LeadInSampleCount here, since IsCompactDisc is
	// required.
	if !cs.IsCompactDisc && cs.LeadInSampleCount != 0 {
		return nil, fmt.Errorf("meta.NewCueSheet: invalid lead-in sample count for non CD-DA; expected 0, got %d.", cs.LeadInSampleCount)
	}

	// Track count.
	err = binary.Read(r, binary.BigEndian, &cs.TrackCount)
	if err != nil {
		return nil, err
	}
	if cs.TrackCount < 1 {
		return nil, errors.New("meta.NewCueSheet: at least one track (the lead-out track) is required.")
	}
	if cs.TrackCount > 100 && cs.IsCompactDisc {
		return nil, fmt.Errorf("meta.NewCueSheet: too many tracks for CD-DA cue sheet; expected <= 100, got %d.", cs.TrackCount)
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
			return nil, fmt.Errorf("meta.NewCueSheet: invalid track offset (%d) for CD-DA; must be evenly divisible by 588.", track.Offset)
		}

		// Track number.
		err = binary.Read(r, binary.BigEndian, &track.TrackNum)
		if err != nil {
			return nil, err
		}
		if track.TrackNum == 0 {
			// A track number of 0 is not allowed to avoid conflicting with the
			// CD-DA spec, which reserves this for the lead-in.
			return nil, errors.New("meta.NewCueSheet: track number 0 not allowed.")
		}
		if cs.IsCompactDisc {
			if i == len(cs.Tracks)-1 {
				if track.TrackNum != 170 {
					// The lead-out track number must be 170 for CD-DA.
					return nil, fmt.Errorf("meta.NewCueSheet: invalid lead-out track number for CD-DA; expected 170, got %d.", track.TrackNum)
				}
			} else if track.TrackNum > 99 {
				return nil, fmt.Errorf("meta.NewCueSheet: invalid track number for CD-DA; expected <= 99, got %d.", track.TrackNum)
			}
		} else {
			if i == len(cs.Tracks)-1 && track.TrackNum != 255 {
				// The lead-out track number must be 255 for non-CD-DA.
				return nil, fmt.Errorf("meta.NewCueSheet: invalid lead-out track number for non CD-DA; expected 255, got %d.", track.TrackNum)
			}
		}

		// Track ISRC (size: 12 bytes).
		buf = make([]byte, 12)
		_, err = r.Read(buf)
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
		buf = make([]byte, 13)
		_, err = r.Read(buf) // 13 reserved bytes.
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
				return nil, fmt.Errorf("meta.NewCueSheet: invalid number of track points for the lead-out track; expected 0, got %d.", track.TrackIndexCount)
			}
		} else {
			if track.TrackIndexCount < 1 {
				// Every track, except for the lead-out track, must have at least
				// one track index point.
				return nil, fmt.Errorf("meta.NewCueSheet: invalid number of track points; expected >= 1, got %d.", track.TrackIndexCount)
			}
			if cs.IsCompactDisc && track.TrackIndexCount > 100 {
				return nil, fmt.Errorf("meta.NewCueSheet: invalid number of track points for CD-DA; expected <= 100, got %d.", track.TrackIndexCount)
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
			buf = make([]byte, 3)
			_, err = io.ReadFull(r, buf) // 3 reserved bytes.
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
	Type uint32
	// The MIME type string, in printable ASCII characters 0x20-0x7e. The MIME
	// type may also be --> to signify that the data part is a URL of the picture
	// instead of the picture data itself.
	MIME string
	// The description of the picture, in UTF-8.
	Desc string
	// The width of the picture in pixels.
	Width uint32
	// The height of the picture in pixels.
	Height uint32
	// The color depth of the picture in bits-per-pixel.
	ColorDepth uint32
	// For indexed-color pictures (e.g. GIF), the number of colors used, or 0 for
	// non-indexed pictures.
	ColorCount uint32
	// The binary picture data.
	Data []byte
}

// NewPicture parses and returns a new Picture metadata block. The provided
// io.Reader should limit the amount of data that can be read to header.Length
// bytes.
//
// Picture format (pseudo code):
//    // ref: http://flac.sourceforge.net/format.html#metadata_block_picture
//
//    type METADATA_BLOCK_PICTURE struct {
//       type        uint32
//       mime_length uint32
//       mime_string [mime_length]byte
//       desc_length uint32
//       desc_string [desc_length]byte
//       width       uint32
//       height      uint32
//       color_depth uint32
//       color_count uint32
//       data_length uint32
//       data        [data_length]byte
//    }
func NewPicture(r io.Reader) (pic *Picture, err error) {
	// Type.
	pic = new(Picture)
	err = binary.Read(r, binary.BigEndian, &pic.Type)
	if err != nil {
		return nil, err
	}
	if pic.Type > 20 {
		return nil, fmt.Errorf("meta.NewPicture: reserved picture type: %d.", pic.Type)
	}

	// Mime length.
	var mimeLen uint32
	err = binary.Read(r, binary.BigEndian, &mimeLen)
	if err != nil {
		return nil, err
	}

	// Mime string.
	buf := make([]byte, mimeLen)
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}
	pic.MIME = getStringFromSZ(buf)
	for _, r := range pic.MIME {
		if r < 0x20 || r > 0x7E {
			return nil, fmt.Errorf("meta.NewPicture: invalid character in MIME type; expected >= 0x20 and <= 0x7E, got 0x%02X.", r)
		}
	}

	// Desc length.
	var descLen uint32
	err = binary.Read(r, binary.BigEndian, &descLen)
	if err != nil {
		return nil, err
	}

	// Desc string.
	buf = make([]byte, descLen)
	_, err = r.Read(buf)
	if err != nil {
		return nil, err
	}
	pic.Desc = getStringFromSZ(buf)

	// Width.
	err = binary.Read(r, binary.BigEndian, &pic.Width)
	if err != nil {
		return nil, err
	}

	// Height.
	err = binary.Read(r, binary.BigEndian, &pic.Height)
	if err != nil {
		return nil, err
	}

	// ColorDepth.
	err = binary.Read(r, binary.BigEndian, &pic.ColorDepth)
	if err != nil {
		return nil, err
	}

	// ColorCount.
	err = binary.Read(r, binary.BigEndian, &pic.ColorCount)
	if err != nil {
		return nil, err
	}

	// Data length.
	var dataLen uint32
	err = binary.Read(r, binary.BigEndian, &dataLen)
	if err != nil {
		return nil, err
	}

	// Data.
	pic.Data, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(pic.Data) != int(dataLen) {
		return nil, fmt.Errorf("meta.NewPicture: invalid data length; expected %d, got %d.", dataLen, len(pic.Data))
	}

	return pic, nil
}
