package meta

// verifyPadding verifies the body of a Padding metadata block. It should only
// contain zero-padding.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_padding
func (block *Block) verifyPadding() error {
	panic("not yet implemented.")
}
