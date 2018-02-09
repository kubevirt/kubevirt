// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package appstats

import (
	"fmt"
)

type byteSize float64

const (
	_            = iota
	_KB byteSize = 1 << (10 * iota)
	_MB
	_GB
	_TB
	_PB
	_EB
	_ZB
	_YB
)

func (b byteSize) String() string {
	switch {
	case b >= _YB:
		return fmt.Sprintf("%.2fYB", b/_YB)
	case b >= _ZB:
		return fmt.Sprintf("%.2fZB", b/_ZB)
	case b >= _EB:
		return fmt.Sprintf("%.2fEB", b/_EB)
	case b >= _PB:
		return fmt.Sprintf("%.2fPB", b/_PB)
	case b >= _TB:
		return fmt.Sprintf("%.2fTB", b/_TB)
	case b >= _GB:
		return fmt.Sprintf("%.2fGB", b/_GB)
	case b >= _MB:
		return fmt.Sprintf("%.2fMB", b/_MB)
	case b >= _KB:
		return fmt.Sprintf("%.2fKB", b/_KB)
	}
	return fmt.Sprintf("%.2fB", b)
}
