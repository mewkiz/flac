package meta

// VorbisComment contains a list of name/value-pairs.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_vorbis_comment
type VorbisComment struct {
	// Vendor name.
	Vendor string
	// Name-value pair tags.
	Tags []VorbisTag
}

// A VorbisTag represents a name/value-pair.
type VorbisTag struct {
	// Tag name.
	Name string
	// Tag value.
	Val string
}
