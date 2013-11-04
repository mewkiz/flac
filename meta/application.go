package meta

import (
	"fmt"
	"io"
	"io/ioutil"
)

// registeredApplications maps from a registered application ID to a
// description.
//
// ref: http://flac.sourceforge.net/id.html
var registeredApplications = map[ID]string{
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

// An ID is a 4 byte identifier of a registered application.
type ID string

func (id ID) String() string {
	s, ok := registeredApplications[id]
	if ok {
		return s
	}
	return fmt.Sprintf("<unregistered ID: %q>", string(id))
}

// An Application metadata block is used by third-party applications. The only
// mandatory field is a 32-bit identifier. This ID is granted upon request to an
// application by the FLAC maintainers. The remainder of the block is defined by
// the registered application.
type Application struct {
	// Registered application ID.
	ID ID
	// Application data.
	Data []byte
}

// ParseApplication parses and returns a new Application metadata block. The
// provided io.Reader should limit the amount of data that can be read to
// header.Length bytes.
//
// Application format (pseudo code):
//
//    type METADATA_BLOCK_APPLICATION struct {
//       ID   uint32
//       Data [header.Length-4]byte
//    }
//
// ref: http://flac.sourceforge.net/format.html#metadata_block_application
func ParseApplication(r io.Reader) (app *Application, err error) {
	// Application ID (size: 4 bytes).
	buf, err := readBytes(r, 4)
	if err != nil {
		return nil, err
	}
	app = &Application{ID: ID(buf)}
	_, ok := registeredApplications[app.ID]
	if !ok {
		return nil, fmt.Errorf("meta.ParseApplication: unregistered application ID %q", string(app.ID))
	}

	// Data.
	buf, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	app.Data = buf

	return app, nil
}
