// Copyright (C) 2026 Abhilash
//
// This file is derived from the LZO real-time data compression library:
// Copyright (C) 1996-2017 Markus Franz Xaver Johannes Oberhumer
// <markus@oberhumer.com>, http://www.oberhumer.com/opensource/lzo/
//
// This program is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation; either version 2 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <https://www.gnu.org/licenses/>.

// Package lzo1z implements LZO1Z decompression in pure Go, with no cgo and no
// dependencies outside the standard library.
//
// LZO1Z is one variant of Markus Oberhumer's LZO family. It shares LZO1X's
// block format but uses a 0x0700 M2 match offset instead of 0x0800, so the two
// are NOT interchangeable: feeding LZO1X data to this decoder yields either an
// error or silent corruption. If your producer calls lzo1x_* you want an LZO1X
// decoder, not this one.
//
// Decompression only. There is no compressor here; use liblzo2 if you need to
// produce LZO1Z data.
//
//	dst := make([]byte, maxDecompressedSize)
//	n, err := lzo1z.Decompress(src, dst)
//	if err != nil {
//	    return err
//	}
//	use(dst[:n])
//
// The caller sizes dst. Decompress never grows it and returns ErrOutputOverrun
// rather than writing past the end, so an undersized buffer is a clean error
// and not a corruption. Every malformed-input path is checked: truncated
// input, output overrun, and a match offset pointing before the start of the
// output all return an error instead of reading or writing out of bounds.
//
// # Provenance
//
// This is a direct port of lzo1z_decompress from LZO 2.10 (lzo1x_d.ch with the
// LZO1Z branches, M2_MAX_OFFSET = 0x0700). Because it is a derivative of GPL
// code it is distributed under the GNU General Public License v2 or later; see
// the LICENSE file. Commercial licensing of LZO is available separately from
// the original author.
//
// Correctness is pinned against vectors produced by liblzo2's own
// lzo1z_999_compress; see tools/genvectors to regenerate them.
package lzo1z

import "errors"

var (
	ErrInputOverrun  = errors.New("lzo1z: input overrun")
	ErrOutputOverrun = errors.New("lzo1z: output overrun")
	ErrLookbehind    = errors.New("lzo1z: lookbehind overrun")
	ErrNotConsumed   = errors.New("lzo1z: input not fully consumed")
)

const m2MaxOffset = 0x0700

// Decompress decompresses one LZO1Z block from src into dst and returns the
// number of bytes written.
//
// dst must be large enough to hold the entire decompressed block; LZO carries
// no length header, so the caller must know the bound out of band. Too small a
// dst returns ErrOutputOverrun with no partial success. Any trailing bytes in
// src after the end-of-stream marker return ErrNotConsumed.
func Decompress(src, dst []byte) (int, error) {
	var (
		ip, op, t, mPos, lastMOff, off int
	)
	inLen := len(src)
	outLen := len(dst)

	if inLen < 1 {
		return 0, ErrInputOverrun
	}
	if src[ip] > 17 {
		t = int(src[ip]) - 17
		ip++
		if t < 4 {
			goto matchNext
		}
		if op+t > outLen {
			return op, ErrOutputOverrun
		}
		if ip+t > inLen {
			return op, ErrInputOverrun
		}
		copy(dst[op:op+t], src[ip:ip+t])
		op += t
		ip += t
		goto firstLiteralRun
	}

begin:
	if ip >= inLen {
		return op, ErrInputOverrun
	}
	t = int(src[ip])
	ip++
	if t >= 16 {
		goto match
	}
	// a literal run
	if t == 0 {
		for ip < inLen && src[ip] == 0 {
			t += 255
			ip++
		}
		if ip >= inLen {
			return op, ErrInputOverrun
		}
		t += 15 + int(src[ip])
		ip++
	}
	// copy t+3 literals
	t += 3
	if op+t > outLen {
		return op, ErrOutputOverrun
	}
	if ip+t > inLen {
		return op, ErrInputOverrun
	}
	copy(dst[op:op+t], src[ip:ip+t])
	op += t
	ip += t

firstLiteralRun:
	if ip >= inLen {
		return op, ErrInputOverrun
	}
	t = int(src[ip])
	ip++
	if t >= 16 {
		goto match
	}
	// M1 match after a literal run (LZO1Z offset encoding)
	if ip >= inLen {
		return op, ErrInputOverrun
	}
	off = (1 + m2MaxOffset) + (t << 6) + (int(src[ip]) >> 2)
	ip++
	mPos = op - off
	lastMOff = off
	if mPos < 0 {
		return op, ErrLookbehind
	}
	if op+3 > outLen {
		return op, ErrOutputOverrun
	}
	dst[op] = dst[mPos]
	dst[op+1] = dst[mPos+1]
	dst[op+2] = dst[mPos+2]
	op += 3
	goto matchDone

match:
	if t >= 64 { // M2 match
		off = t & 0x1f
		if off >= 0x1c {
			if lastMOff <= 0 {
				return op, ErrLookbehind
			}
			mPos = op - lastMOff
		} else {
			if ip >= inLen {
				return op, ErrInputOverrun
			}
			off = 1 + (off << 6) + (int(src[ip]) >> 2)
			ip++
			mPos = op - off
			lastMOff = off
		}
		t = (t >> 5) - 1
		goto copyMatch
	}
	if t >= 32 { // M3 match
		t &= 31
		if t == 0 {
			for ip < inLen && src[ip] == 0 {
				t += 255
				ip++
			}
			if ip >= inLen {
				return op, ErrInputOverrun
			}
			t += 31 + int(src[ip])
			ip++
		}
		if ip+2 > inLen {
			return op, ErrInputOverrun
		}
		off = 1 + (int(src[ip]) << 6) + (int(src[ip+1]) >> 2)
		mPos = op - off
		lastMOff = off
		ip += 2
		goto copyMatch
	}
	if t >= 16 { // M4 match
		mPos = op - ((t & 8) << 11)
		t &= 7
		if t == 0 {
			for ip < inLen && src[ip] == 0 {
				t += 255
				ip++
			}
			if ip >= inLen {
				return op, ErrInputOverrun
			}
			t += 7 + int(src[ip])
			ip++
		}
		if ip+2 > inLen {
			return op, ErrInputOverrun
		}
		mPos -= (int(src[ip]) << 6) + (int(src[ip+1]) >> 2)
		ip += 2
		if mPos == op {
			goto eofFound
		}
		mPos -= 0x4000
		lastMOff = op - mPos
		goto copyMatch
	}
	// M1 match
	if ip >= inLen {
		return op, ErrInputOverrun
	}
	off = 1 + (t << 6) + (int(src[ip]) >> 2)
	ip++
	mPos = op - off
	lastMOff = off
	if mPos < 0 {
		return op, ErrLookbehind
	}
	if op+2 > outLen {
		return op, ErrOutputOverrun
	}
	dst[op] = dst[mPos]
	dst[op+1] = dst[mPos+1]
	op += 2
	goto matchDone

copyMatch:
	// copy t+2 bytes from mPos; must be byte-by-byte (regions may overlap)
	if mPos < 0 {
		return op, ErrLookbehind
	}
	t += 2
	if op+t > outLen {
		return op, ErrOutputOverrun
	}
	for ; t > 0; t-- {
		dst[op] = dst[mPos]
		op++
		mPos++
	}

matchDone:
	t = int(src[ip-1]) & 3 // LZO1Z keeps the state bits in the last offset byte
	if t == 0 {
		goto begin
	}

matchNext:
	// copy 1-3 trailing literals, then the next token is a match token
	if op+t > outLen {
		return op, ErrOutputOverrun
	}
	if ip+t >= inLen {
		return op, ErrInputOverrun
	}
	for ; t > 0; t-- {
		dst[op] = src[ip]
		op++
		ip++
	}
	t = int(src[ip])
	ip++
	goto match

eofFound:
	if ip < inLen {
		return op, ErrNotConsumed
	}
	if ip > inLen {
		return op, ErrInputOverrun
	}
	return op, nil
}
