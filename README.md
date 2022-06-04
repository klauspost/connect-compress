# connect-compress

[![GoDoc](https://pkg.go.dev/badge/github.com/klauspost/connect-compress.svg)](https://pkg.go.dev/github.com/klauspost/connect-compress)

This package provides improved compression schemes for [Buf Connect](https://github.com/bufbuild/connect-go).

Compression is provided from the [github.com/klauspost/compress](https://github.com/klauspost/compress) package.

# Usage

The `compress.All` function will return options that allow both client and servers to compress and decompress all
formats.

```Go
    // Get client and server options for all compressors...
clientOpts, serverOpts := compress.All(compress.LevelBalanced)
```

To enable client compression and force a specific method use `connect.WithSendCompression(...)`
with one of the 4 provided compression options.

For more details and options see the [documentation](https://pkg.go.dev/github.com/klauspost/connect-compress).

# Supported Formats

For deeper information about the specific implementations
see [github.com/klauspost/compress](https://github.com/klauspost/compress) project.

All implementations in this package provides very fast handling of *incompressible* data. That means that it will impose
only a minor penalty when sending pre-compressed data.

## Gzip

Gzip provides faster compression and decompression methods than the standard library built-in to go-connect.

Expected performance is ~200MB/s on JSON streams. Size reduction is ~85% on JSON stream.

Approximate speeds on different data types. Compression only, single thread:

|                    | JSON   | Binary | Objects | Incompressible |
|--------------------|--------|--------|---------|----------------|
|      Fastest, MB/s | 338.17 | 263.55 |  373.15 | 6460.57
| Fastest, Reduction | 82.14% | 75.40% |  81.21% | -0.01%
|     Balanced, MB/s | 206.04 | 148.05 |  215.19 | 5535.14
| Balanced, Reduction| 84.97% | 76.64% |  83.96% | -0.01%
|     Smallest, MB/s | 59.09  |  18.21 |   46.44 | 119.81
| Smallest, Reduction| 85.70% | 76.60% |  85.47% | -0.02%

Note that Gzip decompression speed can often be below 200MB/s, so this will impose the practical limit.

Gzip, stdlib as available in `go-connect` for reference:

|                    | JSON   | Binary | Objects | Incompressible |
|--------------------|--------|--------|---------|----------------|
|               MB/s | 96.05  |  45.61 |   89.93 | 62.74          |
|          Reduction | 85.61% | 76.62% |  85.01% | -0.03%         |

## Zstandard

[Zstandard](https://github.com/facebook/zstd) uses Zstandard compression, but with limited window sizes. Generally
Zstandard compresses better and is faster than gzip.

Expected performance is ~300MB/s on JSON streams. Size ~35% smaller than gzip on JSON stream.

Approximate speeds on different data types. Compression only, single thread:

|                     | JSON   | Binary | Objects | Incompressible |
|---------------------|--------|--------|---------|----------------|
|       Fastest, MB/s | 611.23 | 395.09 |  564.35 | 2836.57
|  Fastest, Reduction | 88.88% | 75.62% |  87.76% | 0.00%
|      Balanced, MB/s | 322.50 | 155.23 |  364.79 | 2066.60
| Balanced, Reduction | 90.26% | 77.56% |  89.33% | 0.00%
|      Smallest, MB/s | 36.18  |  14.42 |   38.33 | 172.43
| Smallest, Reduction | 92.59% | 79.40% |  91.51% | 0.00%

With `OptAllowMultithreadedCompression` typically 2 goroutines will be used.

Generally decompression should be able to keep up with compression speed.

## Snappy

[Snappy](https://github.com/google/snappy) uses Google snappy format.

Expected performance is ~600MB/s on JSON streams. Size ~50% bigger than gzip on JSON stream.

Approximate speeds on different data types. Compression only, single thread:

|                    | JSON   | Binary | Objects | Incompressible  |
|--------------------|---------|--------|---------|----------------|
|      Fastest, MB/s | 1004.25 | 959.14 | 1041.04 | 4511.01
| Fastest, Reduction | 76.47%  | 65.79% |  75.77% | -0.01%
|     Balanced, MB/s | 599.05  | 536.50 |  595.51 | 2298.54
| Balanced, Reduction| 77.68%  | 68.81% |  77.46% | -0.01%
|     Smallest, MB/s | 68.50   |  57.75 |   70.55 | 216.17
| Smallest, Reduction| 78.93%  | 69.09% |  78.58% | -0.01%

With `OptAllowMultithreadedCompression` all cores can used for a roughly linear speed improvement.

Decompression will usually be limited at around 1000MB/s.

## S2

[S2](https://github.com/klauspost/compress/tree/master/s2#s2-compression) provides better compression than Snappy at
similar or better speeds.

Expected performance is ~750MB/s on JSON streams. Size ~2% bigger than gzip on JSON stream.

Approximate speeds on different data types. Compression only, single thread:

|                    | JSON    | Binary | Objects | Incompressible |
|--------------------|---------|--------|---------|----------------|
|      Fastest, MB/s | 1145.57 | 902.15 | 1054.89 | 5520.22
| Fastest, Reduction | 83.40%  | 66.51% |  81.81% | 0.00%
|     Balanced, MB/s | 773.94  | 509.71 |  721.64 | 4413.79
| Balanced, Reduction| 84.79%  | 69.74% |  83.85% | 0.00%
|     Smallest, MB/s | 59.66   |  29.87 |   58.59 | 630.93
| Smallest, Reduction| 86.75%  | 70.24% |  86.30% | 0.00%

With `OptAllowMultithreadedCompression` all cores can used for a roughly linear speed improvement.

# License

This package is made available with the Apache License Version 2.0

See LICENSE for more information.
