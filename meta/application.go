package meta

import (
	"encoding/binary"
	"io/ioutil"
)

// Application contains third party application specific data.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_application
type Application struct {
	// Registered application ID.
	//
	// ref: https://www.xiph.org/flac/id.html
	ID uint32
	// Application data.
	Data []byte
}

// parseApplication reads and parses the body of an Application metadata block.
func (block *Block) parseApplication() error {
	app := new(Application)
	err := binary.Read(block.lr, binary.BigEndian, &app.ID)
	if err != nil {
		return err
	}
	app.Data, err = ioutil.ReadAll(block.lr)
	return err
}
