package meta

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

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
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_picture
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
