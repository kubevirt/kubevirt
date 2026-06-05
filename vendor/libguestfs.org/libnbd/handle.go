/* libnbd golang handle.
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

static struct nbd_handle *
_nbd_create_wrapper (struct error *err)
{
  struct nbd_handle *ret;

  ret = nbd_create ();
  if (ret == NULL)
    save_error (err);
  return ret;
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

/* Handle. */
type Libnbd struct {
	h *C.struct_nbd_handle
}

/* Convert handle to string (just for debugging). */
func (h *Libnbd) String() string {
	return "&Libnbd{}"
}

/* Used for block status callback. */
type LibnbdExtent struct {
	Length uint64 // length of the extent
	Flags  uint64 // flags describing properties of the extent
}

/* All functions (except Close) return ([result,] LibnbdError). */
type LibnbdError struct {
	Op     string        // operation which failed
	Errmsg string        // string (nbd_get_error)
	Errno  syscall.Errno // errno (nbd_get_errno)
}

func (e *LibnbdError) String() string {
	if e.Errno != 0 {
		return fmt.Sprintf("%s: %s", e.Op, e.Errmsg)
	} else {
		return fmt.Sprintf("%s: %s: %s", e.Op, e.Errmsg, e.Errno)
	}
}

/* Implement the error interface */
func (e *LibnbdError) Error() string {
	return e.String()
}

func get_error(op string, c_err C.struct_error) *LibnbdError {
	errmsg := C.GoString(c_err.error)
	errno := syscall.Errno(c_err.errnum)
	return &LibnbdError{Op: op, Errmsg: errmsg, Errno: errno}
}

func closed_handle_error(op string) *LibnbdError {
	return &LibnbdError{Op: op, Errmsg: "handle is closed",
		Errno: syscall.Errno(0)}
}

/* Create a new handle. */
func Create() (*Libnbd, error) {
	var c_err C.struct_error
	c_h := C._nbd_create_wrapper(&c_err)
	if c_h == nil {
		err := get_error("create", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	h := &Libnbd{h: c_h}
	// Finalizers aren't guaranteed to run, but try having one anyway ...
	runtime.SetFinalizer(h, (*Libnbd).Close)
	return h, nil
}

/* Close the handle. */
func (h *Libnbd) Close() *LibnbdError {
	if h.h == nil {
		return closed_handle_error("close")
	}
	C.nbd_close(h.h)
	h.h = nil
	return nil
}

/* Functions for translating between NULL-terminated lists of
 * C strings and golang []string.
 */
func arg_string_list(xs []string) []*C.char {
	r := make([]*C.char, 1+len(xs))
	for i, x := range xs {
		r[i] = C.CString(x)
	}
	r[len(xs)] = nil
	return r
}

func free_string_list(argv []*C.char) {
	for i := 0; argv[i] != nil; i++ {
		C.free(unsafe.Pointer(argv[i]))
	}
}
