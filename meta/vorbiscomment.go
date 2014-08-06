package meta

// VorbisComment contains a list of name-value pairs.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_vorbis_comment
type VorbisComment struct {
	// Vendor name.
	Vendor string
	// A list of tags, each represented by a name-value pair.
	Tags [][2]string
}

// parseVorbisComment reads and parses the body of an VorbisComment metadata block.
func (block *Block) parseVorbisComment() error {
	panic("not yet implemented.")
}
