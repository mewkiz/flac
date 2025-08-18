package meta_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

var golden = []struct {
	path   string
	info   *meta.StreamInfo
	blocks []*meta.Block
}{
	{
		path: "../testdata/59996.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1000, BlockSizeMax: 0x1000, FrameSizeMin: 0x44c5, FrameSizeMax: 0x4588, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x18, NSamples: 0x2000, MD5sum: [16]uint8{0x95, 0xba, 0xe5, 0xe2, 0xc7, 0x45, 0xbb, 0x3c, 0xa9, 0x5c, 0xa3, 0xb1, 0x35, 0xc9, 0x43, 0xf4}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x4, Length: 202, IsLast: true},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917", Tags: [][2]string{{"Description", "Waving a bamboo staff"}, {"YEAR", "2008"}, {"ARTIST", "qubodup aka Iwan Gabovitch | qubodup@gmail.com"}, {"COMMENTS", "I release this file into the public domain"}}},
			},
		},
	},
	{
		path: "../testdata/172960.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1000, BlockSizeMax: 0x1000, FrameSizeMin: 0xb7c, FrameSizeMax: 0x256b, SampleRate: 0x17700, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0xaaa3, MD5sum: [16]uint8{0x76, 0x3d, 0xa8, 0xa5, 0xb7, 0x58, 0xe6, 0x2, 0x61, 0xb4, 0xd4, 0xc2, 0x88, 0x4d, 0x8e, 0xe}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x4, Length: 180, IsLast: true},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917", Tags: [][2]string{{"GENRE", "Sound Clip"}, {"ARTIST", "Iwan 'qubodup' Gabovitch"}, {"Artist Homepage", "http://qubodup.net"}, {"Artist Email", "qubodup@gmail.com"}, {"DATE", "2012"}}},
			},
		},
	},
	{
		path: "../testdata/189983.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0x94d, FrameSizeMax: 0x264a, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x50f4, MD5sum: [16]uint8{0x63, 0x28, 0xed, 0x6d, 0xd3, 0xe, 0x55, 0xfb, 0xa5, 0x73, 0x69, 0x2b, 0xb7, 0x35, 0x73, 0xb7}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x4, Length: 40, IsLast: true},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917", Tags: nil},
			},
		},
	},
	{
		path: "testdata/input-SCPAP.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x3, Length: 180, IsLast: false},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{{SampleNum: 0x0, Offset: 0x0, NSamples: 0x1200}, {SampleNum: 0x1200, Offset: 0xe, NSamples: 0x4f8}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}}},
			},
			{
				Header: meta.Header{Type: 0x5, Length: 540, IsLast: false},
				Body:   &meta.CueSheet{MCN: "1234567890123", NLeadInSamples: 0x15888, IsCompactDisc: true, Tracks: []meta.CueSheetTrack{{Offset: 0x0, Num: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}, {Offset: 0x24c, Num: 0x2}}}, {Offset: 0xb7c, Num: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}}}, {Offset: 0x16f8, Num: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: meta.Header{Type: 0x1, Length: 4, IsLast: false},
				Body:   nil,
			},
			{
				Header: meta.Header{Type: 0x2, Length: 4, IsLast: false},
				Body:   &meta.Application{ID: 0x66616b65, Data: nil},
			},
			{
				Header: meta.Header{Type: 0x1, Length: 3201, IsLast: true},
				Body:   nil,
			},
		},
	},
	{
		path: "testdata/input-SCVA.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x3, Length: 180, IsLast: false},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{{SampleNum: 0x0, Offset: 0x0, NSamples: 0x1200}, {SampleNum: 0x1200, Offset: 0xe, NSamples: 0x4f8}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}}},
			},
			{
				Header: meta.Header{Type: 0x5, Length: 540, IsLast: false},
				Body:   &meta.CueSheet{MCN: "1234567890123", NLeadInSamples: 0x15888, IsCompactDisc: true, Tracks: []meta.CueSheetTrack{{Offset: 0x0, Num: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}, {Offset: 0x24c, Num: 0x2}}}, {Offset: 0xb7c, Num: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}}}, {Offset: 0x16f8, Num: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: meta.Header{Type: 0x4, Length: 203, IsLast: false},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Tags: [][2]string{{"REPLAYGAIN_TRACK_PEAK", "0.99996948"}, {"REPLAYGAIN_TRACK_GAIN", "-7.89 dB"}, {"REPLAYGAIN_ALBUM_PEAK", "0.99996948"}, {"REPLAYGAIN_ALBUM_GAIN", "-7.89 dB"}, {"artist", "1"}, {"title", "2"}}},
			},
			{
				Header: meta.Header{Type: 0x2, Length: 4, IsLast: true},
				Body:   &meta.Application{ID: 0x66616b65, Data: nil},
			},
		},
	},
	{
		path: "testdata/input-SCVAUP.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x3, Length: 180, IsLast: false},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{{SampleNum: 0x0, Offset: 0x0, NSamples: 0x1200}, {SampleNum: 0x1200, Offset: 0xe, NSamples: 0x4f8}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}}},
			},
			{
				Header: meta.Header{Type: 0x5, Length: 540, IsLast: false},
				Body:   &meta.CueSheet{MCN: "1234567890123", NLeadInSamples: 0x15888, IsCompactDisc: true, Tracks: []meta.CueSheetTrack{{Offset: 0x0, Num: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}, {Offset: 0x24c, Num: 0x2}}}, {Offset: 0xb7c, Num: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}}}, {Offset: 0x16f8, Num: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: meta.Header{Type: 0x4, Length: 203, IsLast: false},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Tags: [][2]string{{"REPLAYGAIN_TRACK_PEAK", "0.99996948"}, {"REPLAYGAIN_TRACK_GAIN", "-7.89 dB"}, {"REPLAYGAIN_ALBUM_PEAK", "0.99996948"}, {"REPLAYGAIN_ALBUM_GAIN", "-7.89 dB"}, {"artist", "1"}, {"title", "2"}}},
			},
			{
				Header: meta.Header{Type: 0x2, Length: 4, IsLast: false},
				Body:   &meta.Application{ID: 0x66616b65, Data: nil},
			},
			{
				Header: meta.Header{Type: 0x7e, Length: 0, IsLast: false},
				Body:   nil,
			},
			{
				Header: meta.Header{Type: 0x1, Length: 3201, IsLast: true},
				Body:   nil,
			},
		},
	},
	{
		path: "testdata/input-SCVPAP.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x3, Length: 180, IsLast: false},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{{SampleNum: 0x0, Offset: 0x0, NSamples: 0x1200}, {SampleNum: 0x1200, Offset: 0xe, NSamples: 0x4f8}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}}},
			},
			{
				Header: meta.Header{Type: 0x5, Length: 540, IsLast: false},
				Body:   &meta.CueSheet{MCN: "1234567890123", NLeadInSamples: 0x15888, IsCompactDisc: true, Tracks: []meta.CueSheetTrack{{Offset: 0x0, Num: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}, {Offset: 0x24c, Num: 0x2}}}, {Offset: 0xb7c, Num: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex{{Offset: 0x0, Num: 0x1}}}, {Offset: 0x16f8, Num: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, Indicies: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: meta.Header{Type: 0x4, Length: 203, IsLast: false},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Tags: [][2]string{{"REPLAYGAIN_TRACK_PEAK", "0.99996948"}, {"REPLAYGAIN_TRACK_GAIN", "-7.89 dB"}, {"REPLAYGAIN_ALBUM_PEAK", "0.99996948"}, {"REPLAYGAIN_ALBUM_GAIN", "-7.89 dB"}, {"artist", "1"}, {"title", "2"}}},
			},
			{
				Header: meta.Header{Type: 0x1, Length: 4, IsLast: false},
				Body:   nil,
			},
			{
				Header: meta.Header{Type: 0x2, Length: 4, IsLast: false},
				Body:   &meta.Application{ID: 0x66616b65, Data: nil},
			},
			{
				Header: meta.Header{Type: 0x1, Length: 3201, IsLast: true},
				Body:   nil,
			},
		},
	},
	{
		path: "testdata/input-SVAUP.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x3, Length: 180, IsLast: false},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{{SampleNum: 0x0, Offset: 0x0, NSamples: 0x1200}, {SampleNum: 0x1200, Offset: 0xe, NSamples: 0x4f8}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}, {SampleNum: 0xffffffffffffffff, Offset: 0x0, NSamples: 0x0}}},
			},
			{
				Header: meta.Header{Type: 0x4, Length: 203, IsLast: false},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Tags: [][2]string{{"REPLAYGAIN_TRACK_PEAK", "0.99996948"}, {"REPLAYGAIN_TRACK_GAIN", "-7.89 dB"}, {"REPLAYGAIN_ALBUM_PEAK", "0.99996948"}, {"REPLAYGAIN_ALBUM_GAIN", "-7.89 dB"}, {"artist", "1"}, {"title", "2"}}},
			},
			{
				Header: meta.Header{Type: 0x2, Length: 4, IsLast: false},
				Body:   &meta.Application{ID: 0x66616b65, Data: nil},
			},
			{
				Header: meta.Header{Type: 0x7e, Length: 0, IsLast: false},
				Body:   nil,
			},
			{
				Header: meta.Header{Type: 0x1, Length: 3201, IsLast: true},
				Body:   nil,
			},
		},
	},
	{
		path: "testdata/input-VA.flac",
		info: &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, NChannels: 0x2, BitsPerSample: 0x10, NSamples: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
		blocks: []*meta.Block{
			{
				Header: meta.Header{Type: 0x4, Length: 203, IsLast: false},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Tags: [][2]string{{"REPLAYGAIN_TRACK_PEAK", "0.99996948"}, {"REPLAYGAIN_TRACK_GAIN", "-7.89 dB"}, {"REPLAYGAIN_ALBUM_PEAK", "0.99996948"}, {"REPLAYGAIN_ALBUM_GAIN", "-7.89 dB"}, {"artist", "1"}, {"title", "2"}}},
			},
			{
				Header: meta.Header{Type: 0x2, Length: 4, IsLast: true},
				Body:   &meta.Application{ID: 0x66616b65, Data: nil},
			},
		},
	},
}

func TestParseBlocks(t *testing.T) {
	for _, g := range golden {
		stream, err := flac.ParseFile(g.path)
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		blocks := stream.Blocks

		if len(blocks) != len(g.blocks) {
			t.Errorf("path=%q: invalid number of metadata blocks; expected %d, got %d", g.path, len(g.blocks), len(blocks))
			continue
		}

		got := stream.Info
		want := g.info
		if !reflect.DeepEqual(got, want) {
			t.Errorf("path=%q: metadata StreamInfo block bodies differ; expected %#v, got %#v", g.path, want, got)
		}

		for blockNum, got := range blocks {
			want := g.blocks[blockNum]
			if !reflect.DeepEqual(got.Header, want.Header) {
				t.Errorf("path=%q, blockNum=%d: metadata block headers differ; expected %#v, got %#v", g.path, blockNum, want.Header, got.Header)
			}
			if !reflect.DeepEqual(got.Body, want.Body) {
				t.Errorf("path=%q, blockNum=%d: metadata block bodies differ; expected %#v, got %#v", g.path, blockNum, want.Body, got.Body)
			}
		}
	}
}

func TestParsePicture(t *testing.T) {
	stream, err := flac.ParseFile("testdata/silence.flac")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	want, err := ioutil.ReadFile("testdata/silence.jpg")
	if err != nil {
		t.Fatal(err)
	}

	for _, block := range stream.Blocks {
		if block.Type == meta.TypePicture {
			pic := block.Body.(*meta.Picture)
			got := pic.Data
			if !bytes.Equal(got, want) {
				t.Errorf("picture data differ; expected %v, got %v", want, got)
			}
			break
		}
	}
}

// TODO: better error verification than string-based comparisons.
func TestMissingValue(t *testing.T) {
	_, err := flac.ParseFile("testdata/missing-value.flac")
	if err.Error() != `meta.Block.parseVorbisComment: unable to locate '=' in vector "title 2"` {
		t.Fatal(err)
	}
}

var MaliciousTooManyTags = []byte{
	// "fLaC"
	0x66, 0x4C, 0x61, 0x43,
	// StreamInfo header: type=0, len=34 (0x22)
	0x00, 0x00, 0x00, 0x22,
	// StreamInfo body (34 bytes):
	// BlockSizeMin=16, BlockSizeMax=16
	0x00, 0x10, 0x00, 0x10,
	// FrameSizeMin=0, FrameSizeMax=0
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// 64-bit packed: sampleRate=1, channels=1, bitsPerSample=4, nSamples=0
	0x00, 0x00, 0x10, 0x30, 0x00, 0x00, 0x00, 0x00,
	// MD5 (16 zeros)
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	// VorbisComment header: isLast=1,type=4,len=9
	0x84, 0x00, 0x00, 0x09,
	// vendor length = 1 (little endian)
	0x01, 0x00, 0x00, 0x00,
	// vendor string: "x"
	0x78,
	// tags list length = 4278190080 (little endian)
	0x00, 0x00, 0x00, 0xff,
}

func TestVorbisCommentTooManyTags(t *testing.T) {
	_, err := flac.Parse(bytes.NewReader(MaliciousTooManyTags))
	if !errors.Is(err, meta.ErrDeclaredBlockTooBig) {
		t.Errorf("expected to detect malicious number of tags; actual error=%q", err)
	}
}

// TestVorbisCommentTooManyTagsOOM is designed to parse corrupt or malicious data that may lead to out-of-memory problems.
// It is skipped by default as it may cause instability during test runs.
func TestVorbisCommentTooManyTagsOOM(t *testing.T) {
	t.Skip()
	for i := 0; i < 255; i++ {
		// Parse full metadata stream
		s, err := flac.Parse(bytes.NewReader(MaliciousTooManyTags))
		if err != nil {
			continue
		}
		for {
			if _, err := s.ParseNext(); err != nil {
				break
			}
		}
	}
}
