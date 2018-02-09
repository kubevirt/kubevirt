// Copyright (c) 2013, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the README file.
// Source code and contact info at http://github.com/streadway/handy

package atomic

import (
	"strconv"
	"sync/atomic"
)

// Int implements atomic 64bit int operations
type Int int64

// Add increments i by n
func (i *Int) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

// Get returns the value without races.
func (i *Int) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

// String returns the base 10 formatted value.
func (i *Int) String() string {
	return strconv.FormatInt(i.Get(), 10)
}
