package meta_test

import (
	"reflect"
	"testing"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

var golden = []struct {
	name   string
	blocks []*meta.Block
}{
	// i=0
	{
		name: "../testdata/59996.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1000, BlockSizeMax: 0x1000, FrameSizeMin: 0x44c5, FrameSizeMax: 0x4588, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x18, SampleCount: 0x2000, MD5sum: [16]uint8{0x95, 0xba, 0xe5, 0xe2, 0xc7, 0x45, 0xbb, 0x3c, 0xa9, 0x5c, 0xa3, 0xb1, 0x35, 0xc9, 0x43, 0xf4}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x10, Length: 202},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "Description", Value: "Waving a bamboo staff"}, meta.VorbisEntry{Name: "YEAR", Value: "2008"}, meta.VorbisEntry{Name: "ARTIST", Value: "qubodup aka Iwan Gabovitch | qubodup@gmail.com"}, meta.VorbisEntry{Name: "COMMENTS", Value: "I release this file into the public domain"}}},
			},
		},
	},

	// i=1
	{
		name: "../testdata/172960.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1000, BlockSizeMax: 0x1000, FrameSizeMin: 0xb7c, FrameSizeMax: 0x256b, SampleRate: 0x17700, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0xaaa3, MD5sum: [16]uint8{0x76, 0x3d, 0xa8, 0xa5, 0xb7, 0x58, 0xe6, 0x2, 0x61, 0xb4, 0xd4, 0xc2, 0x88, 0x4d, 0x8e, 0xe}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x10, Length: 180},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "GENRE", Value: "Sound Clip"}, meta.VorbisEntry{Name: "ARTIST", Value: "Iwan 'qubodup' Gabovitch"}, meta.VorbisEntry{Name: "Artist Homepage", Value: "http://qubodup.net"}, meta.VorbisEntry{Name: "Artist Email", Value: "qubodup@gmail.com"}, meta.VorbisEntry{Name: "DATE", Value: "2012"}}},
			},
		},
	},

	// i=2
	{
		name: "../testdata/189983.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0x94d, FrameSizeMax: 0x264a, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x50f4, MD5sum: [16]uint8{0x63, 0x28, 0xed, 0x6d, 0xd3, 0xe, 0x55, 0xfb, 0xa5, 0x73, 0x69, 0x2b, 0xb7, 0x35, 0x73, 0xb7}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x10, Length: 40},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917"},
			},
		},
	},

	// i=3
	{
		name: "testdata/input-SCPAP.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x8, Length: 180},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{meta.SeekPoint{SampleNum: 0x0, Offset: 0x0, SampleCount: 0x1200}, meta.SeekPoint{SampleNum: 0x1200, Offset: 0xe, SampleCount: 0x4f8}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x20, Length: 540},
				Body:   &meta.CueSheet{MCN: "1234567890123", LeadInSampleCount: 0x15888, IsCompactDisc: true, TrackCount: 0x3, Tracks: []meta.CueSheetTrack{meta.CueSheetTrack{Offset: 0x0, TrackNum: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x2, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}, meta.CueSheetTrackIndex{Offset: 0x24c, IndexPointNum: 0x2}}}, meta.CueSheetTrack{Offset: 0xb7c, TrackNum: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x1, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}}}, meta.CueSheetTrack{Offset: 0x16f8, TrackNum: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x0, TrackIndexes: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x2, Length: 4},
				Body:   nil,
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x4, Length: 4},
				Body:   &meta.Application{ID: "fake", Data: []uint8{}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x2, Length: 3201},
				Body:   nil,
			},
		},
	},

	// i=4
	{
		name: "testdata/input-SCVA.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x8, Length: 180},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{meta.SeekPoint{SampleNum: 0x0, Offset: 0x0, SampleCount: 0x1200}, meta.SeekPoint{SampleNum: 0x1200, Offset: 0xe, SampleCount: 0x4f8}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x20, Length: 540},
				Body:   &meta.CueSheet{MCN: "1234567890123", LeadInSampleCount: 0x15888, IsCompactDisc: true, TrackCount: 0x3, Tracks: []meta.CueSheetTrack{meta.CueSheetTrack{Offset: 0x0, TrackNum: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x2, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}, meta.CueSheetTrackIndex{Offset: 0x24c, IndexPointNum: 0x2}}}, meta.CueSheetTrack{Offset: 0xb7c, TrackNum: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x1, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}}}, meta.CueSheetTrack{Offset: 0x16f8, TrackNum: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x0, TrackIndexes: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x10, Length: 203},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "artist", Value: "1"}, meta.VorbisEntry{Name: "title", Value: "2"}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x4, Length: 4},
				Body:   &meta.Application{ID: "fake", Data: []uint8{}},
			},
		},
	},

	// i=5
	{
		name: "testdata/input-SCVAUP.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x8, Length: 180},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{meta.SeekPoint{SampleNum: 0x0, Offset: 0x0, SampleCount: 0x1200}, meta.SeekPoint{SampleNum: 0x1200, Offset: 0xe, SampleCount: 0x4f8}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x20, Length: 540},
				Body:   &meta.CueSheet{MCN: "1234567890123", LeadInSampleCount: 0x15888, IsCompactDisc: true, TrackCount: 0x3, Tracks: []meta.CueSheetTrack{meta.CueSheetTrack{Offset: 0x0, TrackNum: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x2, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}, meta.CueSheetTrackIndex{Offset: 0x24c, IndexPointNum: 0x2}}}, meta.CueSheetTrack{Offset: 0xb7c, TrackNum: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x1, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}}}, meta.CueSheetTrack{Offset: 0x16f8, TrackNum: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x0, TrackIndexes: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x10, Length: 203},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "artist", Value: "1"}, meta.VorbisEntry{Name: "title", Value: "2"}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x4, Length: 4},
				Body:   &meta.Application{ID: "fake", Data: []uint8{}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x80, Length: 0},
				Body:   nil,
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x2, Length: 3201},
				Body:   nil,
			},
		},
	},
	// i=6
	{
		name: "testdata/input-SCVPAP.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x8, Length: 180},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{meta.SeekPoint{SampleNum: 0x0, Offset: 0x0, SampleCount: 0x1200}, meta.SeekPoint{SampleNum: 0x1200, Offset: 0xe, SampleCount: 0x4f8}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x20, Length: 540},
				Body:   &meta.CueSheet{MCN: "1234567890123", LeadInSampleCount: 0x15888, IsCompactDisc: true, TrackCount: 0x3, Tracks: []meta.CueSheetTrack{meta.CueSheetTrack{Offset: 0x0, TrackNum: 0x1, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x2, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}, meta.CueSheetTrackIndex{Offset: 0x24c, IndexPointNum: 0x2}}}, meta.CueSheetTrack{Offset: 0xb7c, TrackNum: 0x2, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x1, TrackIndexes: []meta.CueSheetTrackIndex{meta.CueSheetTrackIndex{Offset: 0x0, IndexPointNum: 0x1}}}, meta.CueSheetTrack{Offset: 0x16f8, TrackNum: 0xaa, ISRC: "", IsAudio: true, HasPreEmphasis: false, TrackIndexCount: 0x0, TrackIndexes: []meta.CueSheetTrackIndex(nil)}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x10, Length: 203},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "artist", Value: "1"}, meta.VorbisEntry{Name: "title", Value: "2"}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x2, Length: 4},
				Body:   nil,
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x4, Length: 4},
				Body:   &meta.Application{ID: "fake", Data: []uint8{}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x2, Length: 3201},
				Body:   nil,
			},
		},
	},

	// i=7
	{
		name: "testdata/input-SVAUP.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x8, Length: 180},
				Body:   &meta.SeekTable{Points: []meta.SeekPoint{meta.SeekPoint{SampleNum: 0x0, Offset: 0x0, SampleCount: 0x1200}, meta.SeekPoint{SampleNum: 0x1200, Offset: 0xe, SampleCount: 0x4f8}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}, meta.SeekPoint{SampleNum: 0xffffffffffffffff, Offset: 0x0, SampleCount: 0x0}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x10, Length: 203},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "artist", Value: "1"}, meta.VorbisEntry{Name: "title", Value: "2"}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x4, Length: 4},
				Body:   &meta.Application{ID: "fake", Data: []uint8{}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x80, Length: 0},
				Body:   nil,
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x2, Length: 3201},
				Body:   nil,
			},
		},
	},

	// i=8
	{
		name: "testdata/input-VA.flac",
		blocks: []*meta.Block{
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x1, Length: 34},
				Body:   &meta.StreamInfo{BlockSizeMin: 0x1200, BlockSizeMax: 0x1200, FrameSizeMin: 0xe, FrameSizeMax: 0x10, SampleRate: 0xac44, ChannelCount: 0x2, BitsPerSample: 0x10, SampleCount: 0x16f8, MD5sum: [16]uint8{0x74, 0xff, 0xd4, 0x73, 0x7e, 0xb5, 0x48, 0x8d, 0x51, 0x2b, 0xe4, 0xaf, 0x58, 0x94, 0x33, 0x62}},
			},
			{
				Header: &meta.BlockHeader{IsLast: false, BlockType: 0x10, Length: 203},
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.1.3 20060805", Entries: []meta.VorbisEntry{meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_TRACK_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_PEAK", Value: "0.99996948"}, meta.VorbisEntry{Name: "REPLAYGAIN_ALBUM_GAIN", Value: "-7.89 dB"}, meta.VorbisEntry{Name: "artist", Value: "1"}, meta.VorbisEntry{Name: "title", Value: "2"}}},
			},
			{
				Header: &meta.BlockHeader{IsLast: true, BlockType: 0x4, Length: 4},
				Body:   &meta.Application{ID: "fake", Data: []uint8{}},
			},
		},
	},
}

func TestParseBlocks(t *testing.T) {
	for i, g := range golden {
		s, err := flac.Open(g.name)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		err = s.ParseBlocks(meta.TypeAllStrict)
		if err != nil {
			t.Fatal(err)
		}

		if len(g.blocks) != len(s.MetaBlocks) {
			t.Errorf("i=%d: invalid number of metadata blocks; expected %d, got %d.", i, len(g.blocks), len(s.MetaBlocks))
			continue
		}

		for j, got := range s.MetaBlocks {
			want := g.blocks[j]
			if !reflect.DeepEqual(want.Header, got.Header) {
				t.Errorf("i=%d, j=%d: metadata block headers differ; expected %#v, got %#v.", i, j, want.Header, got.Header)
			}
			if !reflect.DeepEqual(want.Body, got.Body) {
				t.Errorf("i=%d, j=%d: metadata block bodies differ; expected %#v, got %#v.", i, j, want.Body, got.Body)
			}
		}
	}
}
