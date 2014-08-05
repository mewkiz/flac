package meta

// SeekTable contains one or more precalculated audio frame seek points.
//
// ref: https://www.xiph.org/flac/format.html#metadata_block_seektable
type SeekTable struct {
	// One or more seek points.
	Points []SeekPoint
}

// A SeekPoint specifies the byte offset and initial sample number of a given
// target frame.
//
// ref: https://www.xiph.org/flac/format.html#seekpoint
type SeekPoint struct {
	// Sample number of the first sample in the target frame, or
	// 0xFFFFFFFFFFFFFFFF for a placeholder point.
	SampleNum uint64
	// Offset in bytes from the first byte of the first frame header to the first
	// byte of the target frame's header.
	Offset uint64
	// Number of samples in the target frame.
	NSamples uint16
}
