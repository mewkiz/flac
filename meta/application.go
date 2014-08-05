package meta

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
