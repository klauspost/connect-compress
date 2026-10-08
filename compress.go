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
	"runtime"
	"sync"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"
	"github.com/minio/minlz"
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

	// MinLZ provides better compression than Snappy/S2 at similar speeds.
	// See https://github.com/minio/minlz
	MinLZ = "minlz"
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

// WithAll returns the client and handler option for all compression methods.
// Order of preference is MinLZ, S2, Snappy, Zstandard, Gzip.
// Note that this replaces any previously configured compressors,
// including the default gzip.
func WithAll(level Level, options ...Opts) connecthttp.Option {
	var compressors []connect.Compressor

	for _, name := range []string{MinLZ, S2, Snappy, Zstandard, Gzip} {
		compressors = append(compressors, New(name, level, options...))
	}
	return connecthttp.WithCompressors(compressors...)
}

// New returns a compressor for a single compression method.
// Name must be one of the predefined in this package,
// otherwise New panics.
func New(name string, level Level, options ...Opts) connect.Compressor {
	var o Opts
	for _, opt := range options {
		o = o | opt
	}
	var a algorithm
	switch name {
	case Gzip:
		a = gzComp(level, o)
	case Zstandard:
		a = zstdComp(level, o)
	case Snappy:
		o |= optSnappy
		a = s2Comp(level, o)
	case S2:
		a = s2Comp(level, o)
	case MinLZ:
		a = mzComp(level, o)
	default:
		panic(fmt.Errorf("unknown compression name: %s", name))
	}
	return &compressor{name: name, algorithm: a}
}

// algorithm creates the reusable writers and readers of a compression method.
type algorithm interface {
	newWriter() resetWriter
	newReader() resetReader
}

type resetWriter interface {
	io.WriteCloser
	Reset(io.Writer)
}

type resetReader interface {
	io.ReadCloser
	Reset(io.Reader) error
}

// compressor implements connect.Compressor by pooling
// the writers and readers of an algorithm.
type compressor struct {
	algorithm
	name    string
	writers sync.Pool
	readers sync.Pool
}

func (c *compressor) Name() string {
	return c.name
}

func (c *compressor) Compress(dst io.Writer) (io.WriteCloser, error) {
	w, ok := c.writers.Get().(*compressWriter)
	if !ok {
		w = &compressWriter{resetWriter: c.newWriter(), pool: c}
	}
	w.Reset(dst)
	w.closed = false
	return w, nil
}

func (c *compressor) Decompress(src io.Reader) (io.ReadCloser, error) {
	r, ok := c.readers.Get().(*decompressReader)
	if !ok {
		r = &decompressReader{resetReader: c.newReader(), pool: c}
	}
	if err := r.Reset(src); err != nil {
		return nil, err
	}
	r.closed = false
	return r, nil
}

// compressWriter returns itself to the pool on Close.
type compressWriter struct {
	resetWriter
	pool   *compressor
	closed bool
}

func (w *compressWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.resetWriter.Close(); err != nil {
		return err
	}
	w.pool.writers.Put(w)
	return nil
}

// decompressReader returns itself to the pool on Close.
type decompressReader struct {
	resetReader
	pool   *compressor
	closed bool
}

func (r *decompressReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	if err := r.resetReader.Close(); err != nil {
		return err
	}
	r.pool.readers.Put(r)
	return nil
}

type gzipAlgorithm struct {
	level int
}

func gzComp(level Level, o Opts) algorithm {
	if o.contains(OptStatelessGzip) {
		return gzipAlgorithm{level: gzip.StatelessCompression}
	}
	switch level {
	case LevelFastest:
		return gzipAlgorithm{level: 1}
	case LevelBalanced:
		return gzipAlgorithm{level: 5}
	case LevelSmallest:
		return gzipAlgorithm{level: 9}
	}
	return gzipAlgorithm{level: gzip.DefaultCompression}
}

func (g gzipAlgorithm) newWriter() resetWriter {
	gz, _ := gzip.NewWriterLevel(io.Discard, g.level)
	return gz
}

func (g gzipAlgorithm) newReader() resetReader {
	return &gzip.Reader{}
}

type zstdAlgorithm struct {
	copts []zstd.EOption
	dopts []zstd.DOption
}

func zstdComp(level Level, o Opts) algorithm {
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
	return zstdAlgorithm{copts: copts, dopts: dopts}
}

func (z zstdAlgorithm) newWriter() resetWriter {
	zs, _ := zstd.NewWriter(nil, z.copts...)
	return zs
}

func (a zstdAlgorithm) newReader() resetReader {
	zs, _ := zstd.NewReader(nil, a.dopts...)
	z := &zstdWrapper{dec: zs}
	runtime.AddCleanup(z, func(dec *zstd.Decoder) {
		dec.Close()
	}, zs)
	return z
}

type zstdWrapper struct {
	dec *zstd.Decoder
}

func (z *zstdWrapper) Read(p []byte) (n int, err error) {
	return z.dec.Read(p)
}

func (z *zstdWrapper) Close() error {
	// Do not close so it can be reused, but de-ref the input.
	return z.dec.Reset(nil)
}

func (z *zstdWrapper) Reset(reader io.Reader) error {
	return z.dec.Reset(reader)
}

type s2Algorithm struct {
	wopts []s2.WriterOption
	ropts []s2.ReaderOption
}

func s2Comp(level Level, o Opts) algorithm {
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
	return s2Algorithm{wopts: wopts, ropts: ropts}
}

func (s s2Algorithm) newWriter() resetWriter {
	return s2.NewWriter(nil, s.wopts...)
}

func (s s2Algorithm) newReader() resetReader {
	dec := s2.NewReader(nil, s.ropts...)
	return &s2rWrapper{dec: dec}
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

type mzAlgorithm struct {
	wopts []minlz.WriterOption
	ropts []minlz.ReaderOption
}

func mzComp(level Level, o Opts) algorithm {
	var wopts []minlz.WriterOption
	var ropts []minlz.ReaderOption
	if o.contains(OptSmallWindow) {
		wopts = append(wopts, minlz.WriterBlockSize(maxLimitedWindow))
		ropts = append(ropts, minlz.ReaderMaxBlockSize(maxLimitedWindow))
	}

	if !o.contains(OptAllowMultithreadedCompression) {
		wopts = append(wopts, minlz.WriterConcurrency(1))
	}

	switch level {
	case LevelFastest:
		wopts = append(wopts, minlz.WriterLevel(minlz.LevelFastest))
	case LevelBalanced:
		wopts = append(wopts, minlz.WriterLevel(minlz.LevelBalanced))
	case LevelSmallest:
		wopts = append(wopts, minlz.WriterLevel(minlz.LevelSmallest))
		if !o.contains(OptSmallWindow) {
			wopts = append(wopts, minlz.WriterBlockSize(8<<20))
		}
	}
	return mzAlgorithm{wopts: wopts, ropts: ropts}
}

func (m mzAlgorithm) newWriter() resetWriter {
	return minlz.NewWriter(nil, m.wopts...)
}

func (m mzAlgorithm) newReader() resetReader {
	dec := minlz.NewReader(nil, m.ropts...)
	return &mzWrapper{dec: dec}
}

type mzWrapper struct {
	dec *minlz.Reader
}

func (s *mzWrapper) Read(p []byte) (n int, err error) {
	return s.dec.Read(p)
}

func (s *mzWrapper) Close() error {
	s.dec.Reset(nil)
	return nil
}

func (s *mzWrapper) Reset(reader io.Reader) error {
	s.dec.Reset(reader)
	return nil
}
