// Package frame implements access to FLAC audio frames.
package frame

import "io"

// A Frame contains the header and subframes of an audio frame. It holds the
// encoded samples from a block (a part) of the audio stream. Each subframe
// holding the samples from one of its channel.
//
// ref: https://www.xiph.org/flac/format.html#frame
type Frame struct {
	// Audio frame header.
	Header
	// One subframe per channel, containing encoded audio samples.
	Subframes []*Subframe
	// Underlying io.Reader.
	r io.Reader
}

// New creates a new Frame for accessing the audio samples of r. It reads and
// parses an audio frame header. Call Frame.Parse to parse the audio samples of
// its subframes, and Frame.Next to gain access to the next audio frame.
func New(r io.Reader) (frame *Frame, err error) {
	frame = &Frame{r: r}
	err = frame.parseHeader()
	if err != nil {
		return nil, err
	}
	panic("not yet implemented.")
}

// Next returns access to the next audio frame. It reads and parses an audio
// frame header. Call Frame.Parse to parse the audio samples of its subframes.
func (frame *Frame) Next() (next *Frame, err error) {
	panic("not yet implemented.")
}

// Parse reads and parses the audio samples from each subframe of the frame. If
// the samples are interchannel correlated between the subframes, it
// decorrelates them.
//
// ref: https://www.xiph.org/flac/format.html#interchannel
func (frame *Frame) Parse() error {
	panic("not yet implemented.")
}

// A Header contains the basic properties of an audio frame, such as its sample
// rate and channel count. To facilitate random access decoding each frame
// header starts with a sync-code. This allows the decoder to synchronize and
// locate the start of a frame header.
//
// ref: https://www.xiph.org/flac/format.html#frame_header
type Header struct {
	// Specifies if the block size is fixed or variable.
	HasFixedBlockSize bool
	// Block size in inter-channel samples, i.e. the number of audio samples in
	// each subframe.
	BlockSize uint16
	// Sample rate in Hz; a 0 value implies unknown, get sample rate from
	// StreamInfo.
	SampleRate uint32
	// Specifies the number of channels (subframes) that exist in the frame,
	// their order and possible interchannel correlation.
	Channels Channels
	// Sample size in bits-per-sample; a 0 value implies unknown, get sample size
	// from StreamInfo.
	BitsPerSample uint8
	// Specifies the frame number if the block size is fixed, and the first
	// sample number in the frame otherwise. When using fixed block size, the
	// first sample number in the frame can be derived by multiplying the frame
	// number with the block size (in samples).
	Num uint64
}

// parseHeader reads and parses the header of an audio frame.
func (frame *Frame) parseHeader() error {
	panic("not yet implemented.")
}

// Channels specifies the number of channels (subframes) that exist in a frame,
// their order and possible interchannel correlation.
type Channels uint8

// Channel assignments. The following abbreviations are used:
//    C:   center (directly in front)
//    R:   right (standard stereo)
//    Sr:  side right (directly to the right)
//    Rs:  right surround (back right)
//    Cs:  center surround (rear center)
//    Ls:  left surround (back left)
//    Sl:  side left (directly to the left)
//    L:   left (standard stereo)
//    Lfe: low-frequency effect (placed according to room acoustics)
//
// The first 6 channel constants follow the SMPTE/ITU-R channel order:
//    L R C Lfe Ls Rs
const (
	ChannelsMono           Channels = iota // 1 channel: mono
	ChannelsLR                             // 2 channels: left, right
	ChannelsLRC                            // 3 channels: left, right, center
	ChannelsLRLsRs                         // 4 channels: left, right, left surround, right surround
	ChannelsLRCLsRs                        // 5 channels: left, right, center, left surround, right surround
	ChannelsLRCLfeLsRs                     // 6 channels: left, right, center, LFE, left surround, right surround
	ChannelsLRCLfeCsSlSr                   // 7 channels: left, right, center, LFE, center surround, side left, side right
	ChannelsLRCLfeLsRsSlSr                 // 8 channels: left, right, center, LFE, left surround, right surround, side left, side right
	ChannelsLeftSide                       // 2 channels: left, side; using interchannel correlation
	ChannelsSideRight                      // 2 channels: side, right; using interchannel correlation
	ChannelsMidSide                        // 2 channels: mid, side; using interchannel correlation
)

// nChannels specifies the number of channels used by each channel assignment.
var nChannels = [...]int{
	ChannelsMono:           1,
	ChannelsLR:             2,
	ChannelsLRC:            3,
	ChannelsLRLsRs:         4,
	ChannelsLRCLsRs:        5,
	ChannelsLRCLfeLsRs:     6,
	ChannelsLRCLfeCsSlSr:   7,
	ChannelsLRCLfeLsRsSlSr: 8,
	ChannelsLeftSide:       2,
	ChannelsSideRight:      2,
	ChannelsMidSide:        2,
}

// Count returns the number of channels (subframes) used by the provided channel
// assignment.
func (channels Channels) Count() int {
	return nChannels[channels]
}
