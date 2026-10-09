// Copyright (C) 2026 Abhilash
//
// This program is free software; you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation; either version 2 of the License, or (at your option)
// any later version. See the LICENSE file at the repository root.

package lzo1z_test

import (
	"errors"
	"fmt"

	lzo1z "github.com/bandari-abhilash/LZO-GO"
)

func ExampleDecompress() {
	// "ABCDEFGHIJKLMNOPQR" as compressed by liblzo2's lzo1z_999_compress.
	src := []byte{
		0x23, 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
		'N', 'O', 'P', 'Q', 'R', 0x11, 0x00, 0x00,
	}

	dst := make([]byte, 64*1024)
	n, err := lzo1z.Decompress(src, dst)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s\n", dst[:n])

	// An undersized dst is a clean error, never an out-of-bounds write.
	_, err = lzo1z.Decompress(src, make([]byte, 4))
	fmt.Println(errors.Is(err, lzo1z.ErrOutputOverrun))
	// Output:
	// ABCDEFGHIJKLMNOPQR
	// true
}
