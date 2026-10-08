// Copyright 2026 Klaus Post.
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
	"bytes"
	"io"
	"testing"

	"connectrpc.com/connect/v2"
)

func makeTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i * 7)
	}
	return data
}

func roundtrip(t *testing.T, comp connect.Compressor, data []byte) {
	t.Helper()

	var compressed bytes.Buffer
	w, err := comp.Compress(&compressed)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	_, err = w.Write(data)
	if err != nil {
		t.Fatalf("compress write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("compress close: %v", err)
	}

	r, err := comp.Decompress(&compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("decomp read: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("decomp close: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Fatalf("roundtrip mismatch: got %d bytes, want %d bytes", len(decompressed), len(data))
	}
}

func TestRoundtrip(t *testing.T) {
	sizes := []int{0, 1, 100, 1024, 64 * 1024, 1024 * 1024}

	type compressorConfig struct {
		name string
		opts []Opts
	}

	configs := []compressorConfig{
		{name: Gzip, opts: []Opts{0, OptSmallWindow, OptAllowMultithreadedCompression, OptStatelessGzip}},
		{name: Zstandard, opts: []Opts{0, OptSmallWindow, OptAllowMultithreadedCompression}},
		{name: Snappy, opts: []Opts{0, OptAllowMultithreadedCompression}},
		{name: S2, opts: []Opts{0, OptSmallWindow, OptAllowMultithreadedCompression}},
		{name: MinLZ, opts: []Opts{0, OptSmallWindow, OptAllowMultithreadedCompression}},
	}

	levels := []Level{LevelFastest, LevelBalanced, LevelSmallest}

	for _, cfg := range configs {
		for _, level := range levels {
			for _, opt := range cfg.opts {
				name := cfg.name
				if level == LevelFastest {
					name += "/fastest"
				} else if level == LevelBalanced {
					name += "/balanced"
				} else {
					name += "/smallest"
				}
				if opt == 0 {
					name += "/default"
				}
				if opt&OptSmallWindow != 0 {
					name += "/smallwin"
				}
				if opt&OptAllowMultithreadedCompression != 0 {
					name += "/mt"
				}
				if opt&OptStatelessGzip != 0 {
					name += "/stateless"
				}

				t.Run(name, func(t *testing.T) {
					comp := New(cfg.name, level, opt)
					if comp.Name() != cfg.name {
						t.Fatalf("name: got %q, want %q", comp.Name(), cfg.name)
					}

					for _, size := range sizes {
						data := makeTestData(size)
						roundtrip(t, comp, data)
					}

					// Test re-use with different data pattern
					for _, size := range sizes {
						data := makeTestData(size)
						for i := range data {
							data[i] ^= 0xFF
						}
						roundtrip(t, comp, data)
					}
				})
			}
		}
	}
}
