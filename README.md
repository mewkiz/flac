# flac

[![Build Status](https://travis-ci.org/mewkiz/flac.svg?branch=master)](https://travis-ci.org/mewkiz/flac)
[![Coverage Status](https://img.shields.io/coveralls/mewkiz/flac.svg)](https://coveralls.io/r/mewkiz/flac?branch=master)
[![GoDoc](https://pkg.go.dev/badge/github.com/mewkiz/flac)](https://pkg.go.dev/github.com/mewkiz/flac)

This package provides access to [FLAC][1] (Free Lossless Audio Codec) streams.

[1]: http://flac.sourceforge.net/format.html

## Documentation

Documentation provided by GoDoc.

- [flac]: provides access to FLAC (Free Lossless Audio Codec) streams.
    - [frame][flac/frame]: implements access to FLAC audio frames.
    - [meta][flac/meta]: implements access to FLAC metadata blocks.

[flac]: http://pkg.go.dev/github.com/mewkiz/flac
[flac/frame]: http://pkg.go.dev/github.com/mewkiz/flac/frame
[flac/meta]: http://pkg.go.dev/github.com/mewkiz/flac/meta

## Usage


Pre-install:

```bash
$ go mod download
$ go get -u
```

To run tests, just run:

```bash
$ go test -v ./...
```

To build just run:

1. Build

```bash
$ go build
```

The `flac` binary is now available in the current directory. You may also wish to run it by following:

```bash
$ ./flac
```

2. Install

To install and use `flac`, simply run:

```bash
$ go install github.com/mewkiz/flac/...
```

The `flac` binary is now installed in your `$GOPATH`.  It has several options available
for generating waveform images:

```
$ flac -h
Usage:

flac2wav [OPTION]... FILE.flac...
  Convert FLAC files to WAV format.
  -f    Force overwrite of output files.

metaflac [OPTION]... FILE.flac...
  List metadata of FLAC files.

wav2flac [OPTION]... FILE.wav...
  Convert WAV files to FLAC format.
  -f    Force overwrite of output files.

Flags:
  -block-number string
        An optional comma-separated list of block numbers to display.
  -f    force overwrite
```

## Examples

### flac2wav

```bash
$ ./flac flac2wav -f testdata/1.flac
$ ./flac flac2wav -f testdata/1.flac testdata/2.flac
```


