/* libnbd golang AIO buffer.
 * Copyright Red Hat
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
 */

package libnbd

/*
#cgo pkg-config: libnbd
#cgo CFLAGS: -D_GNU_SOURCE=1

#include <stdio.h>
#include <stdlib.h>

#include "libnbd.h"
#include "wrappers.h"

*/
import "C"

import "unsafe"

/* Asynchronous I/O buffer. */
type AioBuffer struct {
	P    unsafe.Pointer
	Size uint
}

// MakeAioBuffer makes a new buffer backed by an uninitialized C allocated
// array.
func MakeAioBuffer(size uint) AioBuffer {
	return AioBuffer{C.malloc(C.ulong(size)), size}
}

// MakeAioBuffer makes a new buffer backed by a C allocated array. The
// underlying buffer is set to zero.
func MakeAioBufferZero(size uint) AioBuffer {
	return AioBuffer{C.calloc(C.ulong(1), C.ulong(size)), size}
}

// FromBytes makes a new buffer backed by a C allocated array, initialized by
// copying the given Go slice.
func FromBytes(buf []byte) AioBuffer {
	ret := MakeAioBuffer(uint(len(buf)))
	copy(ret.Slice(), buf)
	return ret
}

// Free deallocates the underlying C allocated array. Using the buffer after
// Free() will panic.
func (b *AioBuffer) Free() {
	if b.P != nil {
		C.free(b.P)
		b.P = nil
	}
}

// Bytes copies the underlying C array to Go allocated memory and return a
// slice. Modifying the returned slice does not modify the underlying buffer
// backing array.
func (b *AioBuffer) Bytes() []byte {
	if b.P == nil {
		panic("Using AioBuffer after Free()")
	}
	return C.GoBytes(b.P, C.int(b.Size))
}

// Slice creates a slice backed by the underlying C array. The slice can be
// used to access or modify the contents of the underlying array. The slice
// must not be used after caling Free().
func (b *AioBuffer) Slice() []byte {
	if b.P == nil {
		panic("Using AioBuffer after Free()")
	}
	return unsafe.Slice((*byte)(b.P), b.Size)
}

// Get returns a pointer to a byte in the underlying C array. The pointer can
// be used to modify the underlying array. The pointer must not be used after
// calling Free().
func (b *AioBuffer) Get(i uint) *byte {
	if b.P == nil {
		panic("Using AioBuffer after Free()")
	}
	return (*byte)(unsafe.Pointer(uintptr(b.P) + uintptr(i)))
}
