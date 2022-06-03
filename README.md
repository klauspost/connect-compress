# connect-compress

[![GoDoc](https://pkg.go.dev/badge/github.com/klauspost/connect-compress.svg)](https://pkg.go.dev/github.com/klauspost/connect-compress)

This package provides improved compression schemes for [Buf Connect](github.com/bufbuild/connect-go).

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

## Gzip

Gzip provides faster compression and decompression methods than the standard library built-in to go-connect.

Expected performance is ~200MB/s on JSON streams. Size reduction is ~85% on JSON stream.

## Zstandard

[Zstandard](https://github.com/facebook/zstd) uses Zstandard compression, but with limited window sizes. Generally
Zstandard compresses better and is faster than gzip.

Expected performance is ~300MB/s on JSON streams. Size ~35% smaller than gzip on JSON stream.

With `OptAllowMultithreadedCompression` typically 2 goroutines will be used.

## Snappy

[Snappy](https://github.com/google/snappy) uses Google snappy format.

Expected performance is ~550MB/s on JSON streams. Size ~50% bigger than gzip on JSON stream.

With `OptAllowMultithreadedCompression` all cores can used for a roughly linear speed improvement.

## S2

[S2](https://github.com/klauspost/compress/tree/master/s2#s2-compression) provides better compression than Snappy at
similar or better speeds.

Expected performance is ~750MB/s on JSON streams. Size ~2% bigger than gzip on JSON stream.

With `OptAllowMultithreadedCompression` all cores can used for a roughly linear speed improvement.

# License

This package is made available with the Apache License Version 2.0

See LICENSE for more information.
