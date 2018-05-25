package flac_test

import (
	"testing"

	"github.com/mewkiz/flac"
)

func TestSkipID3v2(t *testing.T) {
	if _, err := flac.ParseFile("testdata/id3.flac"); err != nil {
		t.Fatal(err)
	}
}
