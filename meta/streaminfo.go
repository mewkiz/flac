package meta

// StreamInfo contains information about the FLAC audio stream. It must be
// present as the first metadata block of a FLAC stream.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_streaminfo
type StreamInfo struct{}
