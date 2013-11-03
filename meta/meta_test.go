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
				Body:   &meta.VorbisComment{Vendor: "reference libFLAC 1.2.1 20070917", Entries: []meta.VorbisEntry{}},
			},
		},
	},
}

func TestParseBlocks(t *testing.T) {
	for _, g := range golden {
		s, err := flac.Open(g.name)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		err = s.ParseBlocks(meta.TypeAllStrict)
		if err != nil {
			t.Fatal(err)
		}

		for i, got := range s.MetaBlocks {
			want := g.blocks[i]
			if !reflect.DeepEqual(want.Header, got.Header) {
				t.Errorf("i=%d: metadata block headers differ; expected %#v, got %#v.", i, want.Header, got.Header)
			}
			if !reflect.DeepEqual(want.Body, got.Body) {
				t.Errorf("i=%d: metadata block bodies differ; expected %#v, got %#v.", i, want.Body, got.Body)
			}
		}
	}
}
