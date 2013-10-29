package meta

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

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
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_vorbis_comment
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
			return nil, fmt.Errorf("meta.NewVorbisComment: invalid comment vector; no '=' present in: %s", vector)
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
