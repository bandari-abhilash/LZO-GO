// Copyright (C) 2026 Abhilash
//
// This program is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation; either version 2 of the License, or (at your option)
// any later version. See the LICENSE file at the repository root.

package lzo1z

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Vectors in testdata were produced by liblzo2's own lzo1z_999_compress; see
// tools/genvectors to regenerate them. Each .lzo1z file must decompress to the
// matching .orig file.
func TestDecompressVectors(t *testing.T) {
	dir := "testdata"
	matches, err := filepath.Glob(filepath.Join(dir, "*.lzo1z"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("no test vectors found in %s (err=%v)", dir, err)
	}
	for _, comp := range matches {
		name := filepath.Base(comp)
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(comp)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(comp[:len(comp)-len(".lzo1z")] + ".orig")
			if err != nil {
				t.Fatal(err)
			}
			dst := make([]byte, 65536)
			n, err := Decompress(src, dst)
			if err != nil {
				t.Fatalf("Decompress: %v (wrote %d bytes)", err, n)
			}
			if n != len(want) {
				t.Fatalf("length mismatch: got %d want %d", n, len(want))
			}
			if !bytes.Equal(dst[:n], want) {
				t.Fatalf("content mismatch")
			}
		})
	}
}

func TestDecompressTruncatedInput(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "text.lzo1z"))
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 65536)
	for cut := 1; cut < len(src); cut++ {
		if _, err := Decompress(src[:cut], dst); err == nil {
			t.Fatalf("truncated input (%d/%d bytes) did not error", cut, len(src))
		}
	}
}

func TestDecompressSmallOutputBuffer(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "zeros.lzo1z"))
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 100) // real output is 16384
	if _, err := Decompress(src, dst); err != ErrOutputOverrun {
		t.Fatalf("want ErrOutputOverrun, got %v", err)
	}
}

func BenchmarkDecompress(b *testing.B) {
	for _, name := range []string{"text", "records", "large", "random"} {
		src, err := os.ReadFile(filepath.Join("testdata", name+".lzo1z"))
		if err != nil {
			b.Fatal(err)
		}
		orig, err := os.ReadFile(filepath.Join("testdata", name+".orig"))
		if err != nil {
			b.Fatal(err)
		}
		dst := make([]byte, len(orig))
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(orig)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Decompress(src, dst); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
