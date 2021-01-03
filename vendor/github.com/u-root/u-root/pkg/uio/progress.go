// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uio

import (
	"io"
	"strings"
)

// ProgressReader implements io.Reader and prints Symbol to W after every
// Interval bytes passes through R.
type ProgressReader struct {
	R io.Reader

	Symbol   string
	Interval int
	W        io.Writer

	counter int
}

// Read implements io.Reader for ProgressReader.
func (r *ProgressReader) Read(p []byte) (n int, err error) {
	defer func() {
		//log.Print("r.Counter %d, r.Interval %d, n
		numSymbols := (r.counter%r.Interval + n) / r.Interval
		r.W.Write([]byte(strings.Repeat(r.Symbol, numSymbols)))
		r.counter += n
	}()
	return r.R.Read(p)
}
