// Copyright 2022 Klaus Post.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compress

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bufbuild/connect-go"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"
)

// Level provides 3 predefined compression levels.
type Level int

const (
	// LevelFastest will choose the least cpu intensive cpu method.
	LevelFastest Level = iota

	// LevelBalanced provides balanced compression.
	// Typical cpu usage will be around 200% of the fastest setting.
	LevelBalanced

	// LevelSmallest will use the strongest and most resource intensive
	// compression method.
	// This is generally not recommended.
	LevelSmallest
)

const (
	// Gzip provides faster compression methods than the standard library
	// built-in to go-connect.
	// Expected performance is ~200MB/s on JSON streams.
	// Size reduction is ~85% on JSON stream.
	Gzip = "gzip"

	// Zstandard uses Zstandard compression,
	// but with limited window sizes.
	// Generally Zstandard compresses better and is faster than gzip.
	// Expected performance is ~300MB/s on JSON streams.
	// Size ~35% smaller than gzip on JSON stream.
	Zstandard = "zstd"

	// Snappy uses Google snappy format.
	// Expected performance is ~550MB/s on JSON streams.
	// Size ~50% bigger than gzip on JSON stream.
	Snappy = "snappy"

	// S2 provides better compression than Snappy at similar or better speeds.
	// Expected performance is ~750MB/s on JSON streams.
	// Size ~2% bigger than gzip on JSON stream.
	S2 = "s2"
)

// Opts provides options
type Opts uint32

func (o Opts) contains(x Opts) bool {
	return o&x == x
}

// maxLimitedWindow is the window limit.
const maxLimitedWindow = 64 << 10

const (
	// OptStatelessGzip will force gzip compression to be stateless.
	// Since each Write call will compress input this will affect compression ratio
	// and should only be used when Write calls are controlled.
	// Typically, this means a buffer should be inserted on the writer.
	// See https://github.com/klauspost/compress#stateless-compression
	// Compression level will be ignored.
	OptStatelessGzip Opts = 1 << iota

	// OptAllowMultithreadedCompression will allow some compression modes to use multiple goroutines.
	OptAllowMultithreadedCompression

	// OptSmallWindow will limit the compression window to 64KB.
	// This will reduce memory usage of running operations,
	// but also make compression worse.
	OptSmallWindow

	// internal snappy option
	optSnappy
)

type compressorOption struct {
	connect.ClientOption
	connect.HandlerOption
}

// WithAll returns the client and handler option for all compression methods.
// Order of preference is S2, Snappy, Zstandard, Gzip.
func WithAll(level Level, options ...Opts) connect.Option {
	var opts []connect.Option

	for _, name := range []string{Gzip, Zstandard, Snappy, S2} {
		opts = append(opts, WithNew(name, level, options...))
	}
	return connect.WithOptions(opts...)
}

// WithNew returns client and handler options for a single compression method.
// Name must be one of the predefined in this package.
func WithNew(name string, level Level, options ...Opts) connect.Option {
	var o Opts
	for _, opt := range options {
		o = o | opt
	}
	var d func() connect.Decompressor
	var c func() connect.Compressor
	switch name {
	case Gzip:
		d, c = gzComp(level, o)
	case Zstandard:
		d, c = zstdComp(level, o)
	case Snappy:
		o |= optSnappy
		d, c = s2Comp(level, o)
	case S2:
		d, c = s2Comp(level, o)
	default:
		panic(fmt.Errorf("unknown compression name: %s", name))
	}
	return &compressorOption{
		ClientOption:  connect.WithAcceptCompression(name, d, c),
		HandlerOption: connect.WithCompression(name, d, c),
	}
}

func gzComp(level Level, o Opts) (d func() connect.Decompressor, c func() connect.Compressor) {
	return func() connect.Decompressor {
			return &gzip.Reader{}
		}, func() connect.Compressor {
			if o.contains(OptStatelessGzip) {
				gz, _ := gzip.NewWriterLevel(ioutil.Discard, gzip.StatelessCompression)
				return gz
			}
			switch level {
			case LevelFastest:
				gz, _ := gzip.NewWriterLevel(ioutil.Discard, 1)
				return gz
			case LevelSmallest:
				gz, _ := gzip.NewWriterLevel(ioutil.Discard, 9)
				return gz
			}
			return gzip.NewWriter(ioutil.Discard)
		}
}

func zstdComp(level Level, o Opts) (d func() connect.Decompressor, c func() connect.Compressor) {
	copts := []zstd.EOption{zstd.WithLowerEncoderMem(true)}
	dopts := []zstd.DOption{zstd.WithDecoderLowmem(true), zstd.WithDecoderConcurrency(1)}
	if o.contains(OptSmallWindow) {
		dopts = append(dopts, zstd.WithDecoderMaxWindow(64<<10))
	}

	if o.contains(OptAllowMultithreadedCompression) {
		// No need to go over board here.
		copts = append(copts, zstd.WithEncoderConcurrency(4))
	} else {
		copts = append(copts, zstd.WithEncoderConcurrency(1))
	}

	switch level {
	case LevelFastest:
		copts = append(copts, zstd.WithEncoderLevel(zstd.SpeedFastest))
		if o.contains(OptSmallWindow) {
			copts = append(copts, zstd.WithWindowSize(64<<10))
		} else {
			copts = append(copts, zstd.WithWindowSize(1<<20))
		}
	case LevelBalanced:
		copts = append(copts, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if o.contains(OptSmallWindow) {
			copts = append(copts, zstd.WithWindowSize(64<<10))
		} else {
			copts = append(copts, zstd.WithWindowSize(1<<20))
		}
	case LevelSmallest:
		copts = append(copts, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		if o.contains(OptSmallWindow) {
			copts = append(copts, zstd.WithWindowSize(64<<10))
		} else {
			copts = append(copts, zstd.WithWindowSize(4<<20))
		}
	}
	return func() connect.Decompressor {
			zs, _ := zstd.NewReader(nil, dopts...)
			return &zstdWrapper{ReadCloser: zs.IOReadCloser(), dec: zs}
		}, func() connect.Compressor {
			zs, _ := zstd.NewWriter(nil, copts...)
			return zs
		}
}

type zstdWrapper struct {
	io.ReadCloser
	dec *zstd.Decoder
}

func (z *zstdWrapper) Reset(reader io.Reader) error {
	return z.dec.Reset(reader)
}

func s2Comp(level Level, o Opts) (d func() connect.Decompressor, c func() connect.Compressor) {
	var wopts []s2.WriterOption
	var ropts []s2.ReaderOption
	if o.contains(optSnappy) {
		wopts = append(wopts, s2.WriterSnappyCompat())
		ropts = append(ropts, s2.ReaderMaxBlockSize(maxLimitedWindow), s2.ReaderAllocBlock(maxLimitedWindow))
	} else if o.contains(OptSmallWindow) {
		wopts = append(wopts, s2.WriterBlockSize(maxLimitedWindow))
		ropts = append(ropts, s2.ReaderMaxBlockSize(maxLimitedWindow), s2.ReaderAllocBlock(maxLimitedWindow))
	}

	if !o.contains(OptAllowMultithreadedCompression) {
		wopts = append(wopts, s2.WriterConcurrency(1))
	}

	switch level {
	case LevelFastest:
	case LevelBalanced:
		wopts = append(wopts, s2.WriterBetterCompression())
	case LevelSmallest:
		wopts = append(wopts, s2.WriterBestCompression())
		if !o.contains(OptSmallWindow) && !o.contains(optSnappy) {
			wopts = append(wopts, s2.WriterBlockSize(4<<20))
		}
	}

	return func() connect.Decompressor {
			dec := s2.NewReader(nil, ropts...)
			return &s2rWrapper{dec: dec}
		}, func() connect.Compressor {
			return s2.NewWriter(nil, wopts...)
		}
}

type s2rWrapper struct {
	dec *s2.Reader
}

func (s *s2rWrapper) Read(p []byte) (n int, err error) {
	return s.dec.Read(p)
}

func (s *s2rWrapper) Close() error {
	s.dec.Reset(nil)
	return nil
}

func (s *s2rWrapper) Reset(reader io.Reader) error {
	s.dec.Reset(reader)
	return nil
}
