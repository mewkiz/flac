//Package meta contains functions for parsing flac metadata
package meta

import "bytes"
import "encoding/binary"
import "fmt"
import "strings"

//Formatted error messages
const (
	ErrInvalidBlockLen            = "invalid block length; must be: %d, function took: %d"
	ErrInvalidBlockType           = "invalid block type"
	ErrInvalidMaxBlockSize        = "invalid block size; %d should be < 65535 and > 16"
	ErrInvalidMinBlockSize        = "invalid block size - %d should be >= 16"
	ErrInvalidSampleRate          = "invalid sample rate - %d should be > 655350 and != 0"
	ErrMalformedVorbisComment     = "malformed vorbis comment: %s"
	ErrReserved                   = "reserved value"
	ErrUnregisterdAppSignature    = "unregistered application signature: %s"
	ErrMissingLeadOutTrack        = "cuesheet needs a lead out track"
	ErrInvalidNumTracksForCompact = "invalid number of tracks for a compact disc, can't be more than 100: %d"
	ErrInvalidTrackNum            = "invalid track number value 0 isn't allowed"
	ErrIsNotNil                   = "the reserved bits are not all 0"
	ErrInvalidPictureType         = "the picture type is invalid (must be <=20): %d"
	ErrInvalidNumSeekPoints       = "the number of seek points must be divisible by 18: %d"
	ErrInvalidSyncCode            = "sync code is invalid (must be 11111111111110 or 16382 decimal): %d"
)

//Application blocks which IDs are registered (http://flac.sourceforge.net/id.html)
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

type DataHeader struct {
	IsLast    bool
	BlockType uint8
	Length    uint32
}

// This block has information about the whole stream. It must be present as the first metadata block in the stream.
type StreamInfo struct {
	MinBlockSize  uint16
	MaxBlockSize  uint16
	MinFrameSize  uint32
	MaxFrameSize  uint32
	SampleRate    uint32
	NumChannels   uint8
	BitsPerSample uint8
	NumSamples    uint64
	MD5           []byte
}

//This block is for use by third-party applications. The only mandatory field is a 32-bit identifier. This ID is granted upon request to an application by the FLAC maintainers. The remainder is of the block is defined by the registered application.
type Application struct {
	Signature string
	Data      []byte ///interface{} type instead?
}

//This is an optional block for storing seek points. It is possible to seek to any given sample in a FLAC stream without a seek table, but the delay can be unpredictable since the bitrate may vary widely within a stream. By adding seek points to a stream, this delay can be significantly reduced. There can be only one SEEKTABLE in a stream, but the table can have any number of seek points. There is also a special 'placeholder' seekpoint which will be ignored by decoders but which can be used to reserve space for future seek point insertion.
type SeekTable struct {
	Points []SeekPoint
}

type SeekPoint struct {
	SampleNumber uint64
	Offset       uint64
	NumSamples   uint16
}

//This block is for storing a list of human-readable name/value pairs. Values are encoded using UTF-8. It is an implementation of the Vorbis comment specification (without the framing bit). This is the only officially supported tagging mechanism in FLAC. There may be only one VORBIS_COMMENT block in a stream. In some external documentation, Vorbis comments are called FLAC tags to lessen confusion.
type VorbisComment struct {
	Vendor  string
	Entries []VorbisEntry
}

type VorbisEntry struct {
	Name  string
	Value string
}

//This block is for storing various information that can be used in a cue sheet. It supports track and index points, compatible with Red Book CD digital audio discs, as well as other CD-DA metadata such as media catalog number and track ISRCs. The CUESHEET block is especially useful for backing up CD-DA discs, but it can be used as a general purpose cueing mechanism for playback.
type CueSheet struct {
	CatalogNum       []byte
	NumLeadInSamples uint64
	IsCompactDisc    bool
	NumTracks        uint8
	Tracks           []CueSheetTrack
}

type CueSheetTrack struct {
	Offset              uint64
	TrackNum            uint8
	ISRC                []byte
	IsAudio             bool
	HasPreEmphasis      bool
	NumTrackIndexPoints uint8
	TrackIndexes        []CueSheetTrackIndex
}

type CueSheetTrackIndex struct {
	Offset        uint64
	IndexPointNum uint8
}

//This block is for storing pictures associated with the file, most commonly cover art from CDs. There may be more than one PICTURE block in a file.
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

///Might trigger unnesccesary errors
func IsAllZero(buf []byte) (err error) {
	for _, b := range buf {
		if b != 0 {
			return fmt.Errorf(ErrIsNotNil)
		}
	}

	return nil
}

//Parse a metadata header
func (h *DataHeader) Parse(block []byte) (err error) {
	const (
		LastBlockMask = 0x80000000
		TypeMask      = 0x7F000000
		LengthMask    = 0x00FFFFFF
	)

	if len(block) != 4 {
		return fmt.Errorf(ErrInvalidBlockLen, len(block))
	}

	bits := binary.BigEndian.Uint32(block)

	//Check if this is the last metadata block
	if bits&LastBlockMask != 0 {
		h.IsLast = true
	}

	h.BlockType = uint8(bits & TypeMask >> 24)
	h.Length = bits & LengthMask

	// 0 : Streaminfo
	// 1 : Padding
	// 2 : Application
	// 3 : Seektable
	// 4 : Vorbis_comment
	// 5 : Cuesheet
	// 6 : Picture
	// 7-126 : reserved
	// 127 : invalid, to avoid confusion with a frame sync code
	if h.BlockType >= 7 && h.BlockType <= 126 {
		return fmt.Errorf(ErrReserved)
	} else if h.BlockType == 127 {
		return fmt.Errorf(ErrInvalidBlockType)
	}

	return nil
}

func (si *StreamInfo) Parse(block []byte) (err error) {

	const (
		MaxBlockSizeMask = 0xFFFF000000000000
		MinFrameSizeMask = 0x0000FFFFFF000000
		MaxFrameSizeMask = 0x0000000000FFFFFF

		SampleRateMask    = 0xFFFFF00000000000
		NumChannelsMask   = 0x00000E0000000000
		BitsPerSampleMask = 0x000001F000000000
		NumSamplesMask    = 0x0000000FFFFFFFFF
	)

	//A StreamInfo block is always 34 bytes
	if len(block) != 34 {
		return fmt.Errorf(ErrInvalidBlockLen, len(block))
	}

	buf := bytes.NewBuffer(block)

	//Minimum block size (size: 2 bytes)
	si.MinBlockSize = binary.BigEndian.Uint16(buf.Next(2))
	if si.MinBlockSize > 0 && si.MinBlockSize < 16 {
		return fmt.Errorf(ErrInvalidMinBlockSize, si.MinBlockSize)
	}

	//In order to keep everything on powers-of-2 boundaries, reads from the block are grouped thus:
	//MaxBlockSize (16 bits) + MinFrameSize (24 bits) + MaxFrameSize (24 bits) = 64 bits
	bits := binary.BigEndian.Uint64(buf.Next(8))

	si.MaxBlockSize = uint16((MaxBlockSizeMask & bits) >> 48)
	if si.MaxBlockSize > 65535 || (si.MaxBlockSize > 0 && si.MaxBlockSize < 16) {
		return fmt.Errorf(ErrInvalidMaxBlockSize, si.MaxBlockSize)
	}

	si.MinFrameSize = uint32((MinFrameSizeMask & bits) >> 32)
	si.MaxFrameSize = uint32((bits & MaxFrameSizeMask))

	//In order to keep everything on powers-of-2 boundaries, reads from the block are grouped thus:
	//SampleRate (20 bits) + NumChannels (3 bits) + BitsPerSample (5 bits) + NumSamples (36 bits) = 64 bits
	bits = binary.BigEndian.Uint64(buf.Next(8))

	si.SampleRate = uint32((SampleRateMask & bits) >> 44)
	if si.SampleRate > 655350 && si.SampleRate != 0 {
		return fmt.Errorf(ErrInvalidSampleRate, si.SampleRate)
	}

	//Both NumChannels and BitsPerSample are specified to be subtracted by 1 in the specification: http://flac.sourceforge.net/format.html#metadata_block_streaminfo
	si.NumChannels = uint8((NumChannelsMask&bits)>>41) + 1
	si.BitsPerSample = uint8((BitsPerSampleMask&bits)>>36) + 1

	si.NumSamples = NumSamplesMask & bits

	//MD5 signature of unencoded audio data (size: 16 bytes)
	si.MD5 = buf.Next(16)

	return nil
}

func (ap *Application) Parse(block []byte) (err error) {

	const (
		AppSignatureLen = 32
	)

	buf := bytes.NewBuffer(block)

	ap.Signature = string(buf.Next(AppSignatureLen / 8))
	_, ok := RegisteredApplications[ap.Signature]
	if !ok {
		return fmt.Errorf(ErrUnregisterdAppSignature, ap.Signature)
	}

	///Make uber switch case for all applications
	// switch ap.Signature {

	// }

	return nil
}

func (st *SeekTable) Parse(block []byte) (err error) {

	///Wtf is placeholder point xD
	///Fix this
	// For placeholder points, the second and third field values are undefined.
	// Seek points within a table must be sorted in ascending order by sample number.
	// Seek points within a table must be unique by sample number, with the exception of placeholder points.
	// The previous two notes imply that there may be any number of placeholder points, but they must all occur at the end of the table.

	const (
		SampleNumberLen      = 64
		OffsetLen            = 64
		NumSamplesInFrameLen = 16
	)

	buf := bytes.NewBuffer(block)

	///Error check for fractions
	if len(block)%18 != 0 {
		return fmt.Errorf(ErrInvalidNumSeekPoints, len(block))
	}
	numSeekPoints := len(block) / 18

	for i := 0; i < numSeekPoints; i++ {
		st.Points = append(st.Points, SeekPoint{
			SampleNumber: binary.BigEndian.Uint64(buf.Next(8)), //Sample Number (size: 8 bytes)
			Offset:       binary.BigEndian.Uint64(buf.Next(8)), //Offset (in bytes) from the first byte of the first frame header to the first byte of the target frame's header. (size: 8 bytes)
			NumSamples:   binary.BigEndian.Uint16(buf.Next(2)), //Number of samples in the target frame.  (size: 2 bytes)
		})
	}

	return nil
}

func (vc *VorbisComment) Parse(block []byte) (err error) {
	buf := bytes.NewBuffer(block)

	//Vendor string (size: determined by previous 4 bytes)
	vc.Vendor = string(buf.Next(int(binary.LittleEndian.Uint32(buf.Next(4)))))

	//Number of comments (size: 4 bytes)
	userCommentListLength := binary.LittleEndian.Uint32(buf.Next(4))

	for i := 0; i < int(userCommentListLength); i++ {
		///This might fail on `=a` strings or simply `=` strings

		//The `TYPE=Value` string (size: determined by previous 4 bytes)
		comment := string(buf.Next(int(binary.LittleEndian.Uint32(buf.Next(4)))))

		if !strings.Contains(comment, `=`) {
			return fmt.Errorf(ErrMalformedVorbisComment)
		}

		//Split at first occurence of `=`
		nameAndValue := strings.SplitN(comment, "=", 2)

		vc.Entries = append(vc.Entries, VorbisEntry{Name: nameAndValue[0], Value: nameAndValue[1]})
	}

	return nil
}

func (cs *CueSheet) Parse(block []byte) (err error) {

	const (
		//CueSheet
		IsCompactDiscMask    = 0x80
		CueSheetReservedMask = 0x7F

		//CueSheetTrack
		IsAudioMask               = 0x80
		HasPreEmphasisMask        = 0x40
		CueSheetTrackReservedMask = 0x3F
	)

	buf := bytes.NewBuffer(block)

	//Media catalog number (size: 128 bytes)
	cs.CatalogNum = buf.Next(128)

	//The number of lead-in samples (size: 8 bytes)
	cs.NumLeadInSamples = binary.BigEndian.Uint64(buf.Next(8))

	//1 bit for IsCompactDisk boolean and 7 bits are reserved.
	bits := uint8(buf.Next(1)[0])

	if bits&IsCompactDiscMask != 0 {
		cs.IsCompactDisc = true
	}

	//Reserved
	if bits&CueSheetReservedMask != 0 {
		return fmt.Errorf(ErrIsNotNil)
	}

	err = IsAllZero(buf.Next(258))
	if err != nil {
		return err
	}

	//The number of tracks (size: 1 byte)
	cs.NumTracks = uint8(buf.Next(1)[0])
	if cs.NumTracks < 1 {
		return fmt.Errorf(ErrMissingLeadOutTrack)
	} else if cs.NumTracks > 100 && cs.IsCompactDisc {
		return fmt.Errorf(ErrInvalidNumTracksForCompact, cs.NumTracks)
	}

	for i := 0; i < int(cs.NumTracks); i++ {
		ct := new(CueSheetTrack)

		//Track offset in samples (size: 8 bytes)
		ct.Offset = binary.BigEndian.Uint64(buf.Next(8))

		//Track number (size: 1 byte)
		ct.TrackNum = uint8(buf.Next(1)[0])

		if ct.TrackNum == 0 {
			return fmt.Errorf(ErrInvalidTrackNum)
		}

		//Track ISRC (size: 12 bytes)
		ct.ISRC = buf.Next(12)

		bits := uint8(buf.Next(1)[0])

		//Is track audio (size: 1 bit)
		if bits&IsAudioMask != 0 {
			ct.IsAudio = true
		}

		//Has pre emphasis (size: 1 bit)
		if bits&HasPreEmphasisMask != 0 {
			ct.HasPreEmphasis = true
		}

		if bits&CueSheetTrackReservedMask != 0 {
			return fmt.Errorf(ErrIsNotNil)
		}

		//Reserved (size: 13 bytes + 6 bits from last byte)
		err = IsAllZero(buf.Next(13))
		if err != nil {
			return err
		}

		///Must be at least 1 on regular but must be 0 at lead out
		//Number of track index points (size: 1 byte)
		ct.NumTrackIndexPoints = uint8(buf.Next(1)[0])

		for i := 0; i < int(ct.NumTrackIndexPoints); i++ {
			ct.TrackIndexes = append(ct.TrackIndexes, CueSheetTrackIndex{
				Offset:        binary.BigEndian.Uint64(buf.Next(8)), //Offset in samples (size: 8 bytes)
				IndexPointNum: uint8(buf.Next(1)[0]),                //The index point number (size: 1 byte) ///Help with uint8
			})

			///All bits must be zero
			//Reserved (size: 3 bytes)
			err = IsAllZero(buf.Next(3))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Picture) Parse(block []byte) (err error) {
	buf := bytes.NewBuffer(block)

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
	p.Type = binary.BigEndian.Uint32(buf.Next(4))
	if p.Type > 20 {
		return fmt.Errorf(ErrInvalidPictureType, p.Type)
	}

	//Length of the mime type (size: 4 bytes), Mime type string (size: depends on length)
	p.MIME = string(buf.Next(int(binary.BigEndian.Uint32(buf.Next(4)))))

	//Length of the Picture description (size: 4 bytes), Description string (size: depends on length)
	p.PicDesc = string(buf.Next(int(binary.BigEndian.Uint32(buf.Next(4)))))

	p.Width = binary.BigEndian.Uint32(buf.Next(4))
	p.Height = binary.BigEndian.Uint32(buf.Next(4))
	p.ColorDepth = binary.BigEndian.Uint32(buf.Next(4))
	p.NumColors = binary.BigEndian.Uint32(buf.Next(4))

	//Length of the Picture data (size: 4 bytes), Picture data (size: depends on length)
	p.Data = buf.Next(int(binary.BigEndian.Uint32(buf.Next(4))))

	return nil
}
