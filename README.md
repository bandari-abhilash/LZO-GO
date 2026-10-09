# lzo1z

[![CI](https://github.com/bandari-abhilash/LZO-GO/actions/workflows/ci.yml/badge.svg)](https://github.com/bandari-abhilash/LZO-GO/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/bandari-abhilash/LZO-GO.svg)](https://pkg.go.dev/github.com/bandari-abhilash/LZO-GO)

**LZO1Z decompression in pure Go.** No cgo, no `liblzo2` at runtime, nothing
outside the standard library. Builds on Go 1.21+.

```
go get github.com/bandari-abhilash/LZO-GO
```

> **License: GPL-2.0-or-later.** This is a derivative of Markus Oberhumer's
> GPL-licensed LZO library, so it cannot be anything more permissive. Go links
> statically, so importing this package makes your whole binary GPL when you
> distribute it. See [Licensing](#licensing) before you depend on it.

## LZO1Z is not LZO1X

This matters more than anything else in this README. LZO1Z shares LZO1X's
block format but uses an M2 match offset of `0x0700` instead of `0x0800`. The
two are **not** interchangeable — decoding LZO1X data with this package gives
you either an error or silently wrong bytes.

If your producer calls `lzo1x_*`, you want an LZO1X decoder. This package is
for data produced by `lzo1z_*`, which shows up in a handful of exchange market
data feeds and older embedded protocols.

Decompression only. There's no compressor here; use `liblzo2` to produce data.

## Usage

```go
import lzo1z "github.com/bandari-abhilash/LZO-GO"

dst := make([]byte, maxDecompressedSize)

n, err := lzo1z.Decompress(src, dst)
if err != nil {
    return err
}
use(dst[:n])
```

The caller sizes `dst`. LZO carries no length header, so you must know the
decompressed bound out of band — from your protocol's framing, or a
conservative ceiling (64 KiB is typical for feed blocks).

`Decompress` never grows `dst` and never writes past it. An undersized buffer
is a clean `ErrOutputOverrun`, not a corruption:

| error | meaning |
|---|---|
| `ErrInputOverrun` | input ended mid-token — truncated or corrupt |
| `ErrOutputOverrun` | `dst` too small; nothing was written past the end |
| `ErrLookbehind` | match offset points before the start of output |
| `ErrNotConsumed` | trailing bytes after the end-of-stream marker |

Every malformed-input path is bounds-checked. `TestDecompressTruncatedInput`
feeds every possible truncation of a real vector and requires an error from
each one.

## Performance

`go test -bench .` on an M-series Mac, single core:

```
BenchmarkDecompress/text-10        3954 ns/op    2020 MB/s    0 allocs/op
BenchmarkDecompress/records-10    14889 ns/op    1558 MB/s    0 allocs/op
BenchmarkDecompress/large-10      28471 ns/op    2107 MB/s    0 allocs/op
```

Zero allocations — the decoder writes only into the `dst` you hand it.

### TODO: performance work

The decoder is a direct port and has not been tuned yet. On an Apple M5,
liblzo2 (C) decodes the `records` vector about 3.8x as fast and lzo-java about
1.5x as fast as this package. A CPU profile of `BenchmarkDecompress` attributes
about 70% of the time to the byte-by-byte match copy in `copyMatch`
(`for ; t > 0; t-- { dst[op] = dst[mPos] ... }`).

- [ ] **Chunked match copy.** When the match distance `op - mPos` is at least 8,
      copy 8 bytes at a time (`binary.LittleEndian.Uint64`/`PutUint64`, or
      `copy()` on non-overlapping pieces). Keep the byte loop only for short
      distances, where source and destination overlap (e.g. repeated-byte runs).
      liblzo2 does the same with its 4- and 8-byte copy paths. Expected to be
      the largest win, especially on `large`.
- [ ] **Bounds-check elimination.** `go build -gcflags=-d=ssa/check_bce/debug=1`
      reports 34 bounds checks in `Decompress`. Reslice once per copy (for
      example `d := dst[mPos : op+t]`) so the compiler can prove the indexes in
      range and drop per-byte checks.
- [ ] **Trailing-literal copy.** The 1–3 byte loop in `matchNext` could become a
      fixed-size copy when enough input and output space remains.
- [ ] **Restructure the `goto` state machine** into a loop with a `switch`, if
      profiling shows registers being spilled across labels.
- [ ] Re-run the cross-language harnesses in
      [the benchmark write-up](https://bandariabhilash.com/blog/benchmarks/lzo1z/README.md)
      and update the numbers above and in the article.

Every change must keep `go test ./...` passing against the liblzo2-generated
vectors, including the malformed-input and overrun cases.

## Test vectors

`testdata/` holds `.orig`/`.lzo1z` pairs covering empty-ish inputs (1, 4 and
18 bytes), long-match text, a 16 KiB zero run, incompressible random data,
fixed-width records, mixed-entropy runs, and a 60 KB buffer that exercises
long match distances.

They are **reproducible**. `tools/genvectors/genvectors.c` regenerates the
whole corpus with liblzo2's own `lzo1z_999_compress`:

```sh
cd tools/genvectors
cc -O2 -o genvectors genvectors.c -llzo2      # brew install lzo / apt install liblzo2-dev
./genvectors ../../testdata
go test ./...
```

Corpora come from a fixed LCG seed, so the `.orig` files are byte-identical on
every machine; the `.lzo1z` files depend on your liblzo2 version, which is
fine — the tests assert round-trip correctness, not specific compressed bytes.

## Provenance

A direct port of `lzo1z_decompress` from LZO 2.10 (`lzo1x_d.ch` with the LZO1Z
branches, `M2_MAX_OFFSET = 0x0700`), validated against vectors generated by
liblzo2 2.10 itself.

## Licensing

- **This package: GPL-2.0-or-later.** See [LICENSE](LICENSE).
- **Original LZO:** Copyright © 1996-2017 Markus Franz Xaver Johannes
  Oberhumer, <http://www.oberhumer.com/opensource/lzo/>.

Because Go produces statically linked binaries, there is no LGPL-style
"link dynamically and stay proprietary" option. If you import this package
into a program and **distribute** that program, the whole program must be
released under the GPL with source.

Two things that are *not* triggers: running it on your own servers without
shipping binaries (GPL-2 has no network clause), and using it internally.

If you need LZO1Z in a proprietary product, the original author sells
commercial LZO licenses — contact him rather than using this port.
