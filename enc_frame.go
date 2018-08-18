package flac

import (
	"io"

	"github.com/icza/bitio"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/internal/hashutil/crc8"
	"github.com/mewkiz/pkg/errutil"
)

// Write encodes the audio samples to FLAC frames.
func (enc *Encoder) Write(samples []int32) error {
	// Encode frame header.
	nsamples := len(samples) / int(enc.stream.Info.NChannels)
	hdr := &frame.Header{
		// Specifies if the block size is fixed or variable.
		HasFixedBlockSize: false,
		// Block size in inter-channel samples, i.e. the number of audio samples in
		// each subframe.
		BlockSize: uint16(nsamples),
		// Sample rate in Hz; a 0 value implies unknown, get sample rate from
		// StreamInfo.
		SampleRate: enc.stream.Info.SampleRate,
		// Specifies the number of channels (subframes) that exist in the frame,
		// their order and possible inter-channel decorrelation.
		Channels: frame.ChannelsMono,
		// Sample size in bits-per-sample; a 0 value implies unknown, get sample size
		// from StreamInfo.
		BitsPerSample: enc.stream.Info.BitsPerSample,
		// Specifies the frame number if the block size is fixed, and the first
		// sample number in the frame otherwise. When using fixed block size, the
		// first sample number in the frame can be derived by multiplying the frame
		// number with the block size (in samples).
		Num: uint64(enc.curNum),
	}
	if hdr.HasFixedBlockSize {
		enc.curNum++
	} else {
		enc.curNum += uint64(nsamples)
	}
	if err := enc.encodeFrameHeader(hdr); err != nil {
		return errutil.Err(err)
	}
	// TODO: Encode samples.
	return nil
}

// encodeFrameHeader encodes the given frame header to the output stream.
func (enc *Encoder) encodeFrameHeader(hdr *frame.Header) error {
	// Create a new CRC-8 hash writer which adds the data from all write
	// operations to a running hash.
	h := crc8.NewATM()
	hw := io.MultiWriter(h, enc.w)
	bw := bitio.NewWriter(hw)
	enc.c = bw

	//  Sync code: 11111111111110
	if err := bw.WriteBits(0x3FFE, 14); err != nil {
		return errutil.Err(err)
	}

	// Reserved: 0
	if err := bw.WriteBits(0x0, 1); err != nil {
		return errutil.Err(err)
	}

	// Blocking strategy:
	//    0 : fixed-blocksize stream; frame header encodes the frame number
	//    1 : variable-blocksize stream; frame header encodes the sample number
	if err := bw.WriteBool(!hdr.HasFixedBlockSize); err != nil {
		return errutil.Err(err)
	}

	// Block size in inter-channel samples:
	//    0000 : reserved
	//    0001 : 192 samples
	//    0010-0101 : 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608
	//    0110 : get 8 bit (blocksize-1) from end of header
	//    0111 : get 16 bit (blocksize-1) from end of header
	//    1000-1111 : 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/16384/32768
	var (
		bits uint64
		// number of bits used to store block size after the frame header.
		nblockSizeSuffixBits byte
	)
	switch hdr.BlockSize {
	case 192:
		// 0001
		bits = 0x1
	case 576, 1152, 2304, 4608:
		// 0010-0101 : 576 * (2^(n-2)) samples, i.e. 576/1152/2304/4608
		bits = 0x2 + uint64(hdr.BlockSize/576) - 1
	case 256, 512, 1024, 2048, 4096, 8192, 16384, 32768:
		// 1000-1111 : 256 * (2^(n-8)) samples, i.e. 256/512/1024/2048/4096/8192/16384/32768
		bits = 0x8 + uint64(hdr.BlockSize/256) - 1
	default:
		if hdr.BlockSize <= 256 {
			// 0110 : get 8 bit (blocksize-1) from end of header
			bits = 0x6
			nblockSizeSuffixBits = 8
		} else {
			// 0111 : get 16 bit (blocksize-1) from end of header
			bits = 0x7
			nblockSizeSuffixBits = 16
		}
	}
	if err := bw.WriteBits(bits, 4); err != nil {
		return errutil.Err(err)
	}

	// Sample rate:
	//    0000 : get from STREAMINFO metadata block
	//    0001 : 88.2kHz
	//    0010 : 176.4kHz
	//    0011 : 192kHz
	//    0100 : 8kHz
	//    0101 : 16kHz
	//    0110 : 22.05kHz
	//    0111 : 24kHz
	//    1000 : 32kHz
	//    1001 : 44.1kHz
	//    1010 : 48kHz
	//    1011 : 96kHz
	//    1100 : get 8 bit sample rate (in kHz) from end of header
	//    1101 : get 16 bit sample rate (in Hz) from end of header
	//    1110 : get 16 bit sample rate (in tens of Hz) from end of header
	//    1111 : invalid, to prevent sync-fooling string of 1s
	var (
		// bits used to store sample rate after the frame header.
		sampleRateSuffixBits uint64
		// number of bits used to store sample rate after the frame header.
		nsampleRateSuffixBits byte
	)
	switch hdr.SampleRate {
	case 0:
		// 0000 : get from STREAMINFO metadata block
		bits = 0
	case 88200:
		// 0001 : 88.2kHz
		bits = 0x1
	case 176400:
		// 0010 : 176.4kHz
		bits = 0x2
	case 192000:
		// 0011 : 192kHz
		bits = 0x3
	case 8000:
		// 0100 : 8kHz
		bits = 0x4
	case 16000:
		// 0101 : 16kHz
		bits = 0x5
	case 22050:
		// 0110 : 22.05kHz
		bits = 0x6
	case 24000:
		// 0111 : 24kHz
		bits = 0x7
	case 32000:
		// 1000 : 32kHz
		bits = 0x8
	case 44100:
		// 1001 : 44.1kHz
		bits = 0x9
	case 48000:
		// 1010 : 48kHz
		bits = 0xA
	case 96000:
		// 1011 : 96kHz
		bits = 0xB
	default:
		switch {
		case hdr.SampleRate <= 255000 && hdr.SampleRate%1000 == 0:
			// 1100 : get 8 bit sample rate (in kHz) from end of header
			bits = 0xC
			sampleRateSuffixBits = uint64(hdr.SampleRate / 1000)
			nsampleRateSuffixBits = 8
		case hdr.SampleRate <= 65535:
			// 1101 : get 16 bit sample rate (in Hz) from end of header
			bits = 0xD
			sampleRateSuffixBits = uint64(hdr.SampleRate)
			nsampleRateSuffixBits = 16
		case hdr.SampleRate <= 655350 && hdr.SampleRate%10 == 0:
			// 1110 : get 16 bit sample rate (in tens of Hz) from end of header
			bits = 0xE
			sampleRateSuffixBits = uint64(hdr.SampleRate / 10)
			nsampleRateSuffixBits = 16
		default:
			return errutil.Newf("unable to encode sample rate %v", hdr.SampleRate)
		}
	}
	if err := bw.WriteBits(bits, 4); err != nil {
		return errutil.Err(err)
	}

	// Channel assignment.
	//    0000-0111 : (number of independent channels)-1. Where defined, the channel order follows SMPTE/ITU-R recommendations. The assignments are as follows:
	//        1 channel: mono
	//        2 channels: left, right
	//        3 channels: left, right, center
	//        4 channels: front left, front right, back left, back right
	//        5 channels: front left, front right, front center, back/surround left, back/surround right
	//        6 channels: front left, front right, front center, LFE, back/surround left, back/surround right
	//        7 channels: front left, front right, front center, LFE, back center, side left, side right
	//        8 channels: front left, front right, front center, LFE, back left, back right, side left, side right
	//    1000 : left/side stereo: channel 0 is the left channel, channel 1 is the side(difference) channel
	//    1001 : right/side stereo: channel 0 is the side(difference) channel, channel 1 is the right channel
	//    1010 : mid/side stereo: channel 0 is the mid(average) channel, channel 1 is the side(difference) channel
	//    1011-1111 : reserved
	switch hdr.Channels {
	case frame.ChannelsMono, frame.ChannelsLR, frame.ChannelsLRC, frame.ChannelsLRLsRs, frame.ChannelsLRCLsRs, frame.ChannelsLRCLfeLsRs, frame.ChannelsLRCLfeCsSlSr, frame.ChannelsLRCLfeLsRsSlSr:
		// 1 channel: mono.
		// 2 channels: left, right.
		// 3 channels: left, right, center.
		// 4 channels: left, right, left surround, right surround.
		// 5 channels: left, right, center, left surround, right surround.
		// 6 channels: left, right, center, LFE, left surround, right surround.
		// 7 channels: left, right, center, LFE, center surround, side left, side right.
		// 8 channels: left, right, center, LFE, left surround, right surround, side left, side right.
		bits = uint64(hdr.Channels.Count() - 1)
	case frame.ChannelsLeftSide:
		// 2 channels: left, side; using inter-channel decorrelation.
		// 1000 : left/side stereo: channel 0 is the left channel, channel 1 is the side(difference) channel
		bits = 0x8
	case frame.ChannelsSideRight:
		// 2 channels: side, right; using inter-channel decorrelation.
		// 1001 : right/side stereo: channel 0 is the side(difference) channel, channel 1 is the right channel
		bits = 0x9
	case frame.ChannelsMidSide:
		// 2 channels: mid, side; using inter-channel decorrelation.
		// 1010 : mid/side stereo: channel 0 is the mid(average) channel, channel 1 is the side(difference) channel
		bits = 0xA
	default:
		return errutil.Newf("support for channel assignment %v not yet implemented", hdr.Channels)
	}
	if err := bw.WriteBits(bits, 4); err != nil {
		return errutil.Err(err)
	}

	// Sample size in bits:
	//    000 : get from STREAMINFO metadata block
	//    001 : 8 bits per sample
	//    010 : 12 bits per sample
	//    011 : reserved
	//    100 : 16 bits per sample
	//    101 : 20 bits per sample
	//    110 : 24 bits per sample
	//    111 : reserved
	switch hdr.BitsPerSample {
	case 0:
		// 000 : get from STREAMINFO metadata block
		bits = 0x0
	case 8:
		// 001 : 8 bits per sample
		bits = 0x1
	case 12:
		// 010 : 12 bits per sample
		bits = 0x2
	case 16:
		// 100 : 16 bits per sample
		bits = 0x4
	case 20:
		// 101 : 20 bits per sample
		bits = 0x5
	case 24:
		// 110 : 24 bits per sample
		bits = 0x6
	default:
		return errutil.Newf("support for sample size %v not yet implemented", hdr.BitsPerSample)
	}
	if err := bw.WriteBits(bits, 3); err != nil {
		return errutil.Err(err)
	}

	// Reserved: 0
	if err := bw.WriteBits(0x0, 1); err != nil {
		return errutil.Err(err)
	}

	//    if (variable blocksize)
	//       <8-56>:"UTF-8" coded sample number (decoded number is 36 bits)
	//    else
	//       <8-48>:"UTF-8" coded frame number (decoded number is 31 bits)
	if err := encodeUTF8(bw, hdr.Num); err != nil {
		return errutil.Err(err)
	}

	// Write block size after the frame header (used for uncommon block sizes).
	if nblockSizeSuffixBits > 0 {
		// 0110 : get 8 bit (blocksize-1) from end of header
		// 0111 : get 16 bit (blocksize-1) from end of header
		if err := bw.WriteBits(uint64(hdr.BlockSize-1), nblockSizeSuffixBits); err != nil {
			return errutil.Err(err)
		}
	}

	// Write sample rate after the frame header (used for uncommon sample rates).
	if nsampleRateSuffixBits > 0 {
		if err := bw.WriteBits(sampleRateSuffixBits, nsampleRateSuffixBits); err != nil {
			return errutil.Err(err)
		}
	}

	// Flush pending writes.
	if _, err := bw.Align(); err != nil {
		return errutil.Err(err)
	}

	// CRC-8 (polynomial = x^8 + x^2 + x^1 + x^0, initialized with 0) of
	// everything before the crc, including the sync code.
	crc := h.Sum8()
	if err := writeByte(enc.w, crc); err != nil {
		return errutil.Err(err)
	}

	return nil
}

// writeByte writes the given byte to w.
func writeByte(w io.Writer, b byte) error {
	var buf [1]byte
	buf[0] = b
	n, err := w.Write(buf[:])
	if err != nil {
		return errutil.Err(err)
	}
	if n != 1 {
		return errutil.Newf("unable to write byte 0x%02X to io.Writer", b)
	}
	return nil
}
