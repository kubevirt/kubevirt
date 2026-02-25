/* NBD client library in userspace
 * WARNING: THIS FILE IS GENERATED FROM
 * generator/generator
 * ANY CHANGES YOU MAKE TO THIS FILE WILL BE LOST.
 *
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

#include <stdlib.h>

#include "libnbd.h"
#include "wrappers.h"
*/
import "C"

import "unsafe"

/* Closures. */

func copy_uint32_array(entries *C.uint32_t, count C.size_t) []uint32 {
	ret := make([]uint32, count)
	s := unsafe.Slice(entries, count)
	for i, item := range s {
		ret[i] = uint32(item)
	}
	return ret
}

func copy_extent_array(entries *C.nbd_extent, count C.size_t) []LibnbdExtent {
	ret := make([]LibnbdExtent, count)
	s := unsafe.Slice(entries, count)
	for i, item := range s {
		ret[i].Length = uint64(item.length)
		ret[i].Flags = uint64(item.flags)
	}
	return ret
}

type ChunkCallback func(subbuf []byte, offset uint64, status uint, error *int) int

//export chunk_callback
func chunk_callback(callbackid *C.long, subbuf unsafe.Pointer, count C.size_t, offset C.uint64_t, status C.uint, error *C.int) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(ChunkCallback)
	if !ok {
		panic("inappropriate callback type")
	}
	go_error := int(*error)
	ret := callback(C.GoBytes(subbuf, C.int(count)), uint64(offset), uint(status), &go_error)
	*error = C.int(go_error)
	return C.int(ret)
}

type CompletionCallback func(error *int) int

//export completion_callback
func completion_callback(callbackid *C.long, error *C.int) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(CompletionCallback)
	if !ok {
		panic("inappropriate callback type")
	}
	go_error := int(*error)
	ret := callback(&go_error)
	*error = C.int(go_error)
	return C.int(ret)
}

type DebugCallback func(context string, msg string) int

//export debug_callback
func debug_callback(callbackid *C.long, context *C.char, msg *C.char) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(DebugCallback)
	if !ok {
		panic("inappropriate callback type")
	}
	ret := callback(C.GoString(context), C.GoString(msg))
	return C.int(ret)
}

type ExtentCallback func(metacontext string, offset uint64, entries []uint32, error *int) int

//export extent_callback
func extent_callback(callbackid *C.long, metacontext *C.char, offset C.uint64_t, entries *C.uint32_t, nr_entries C.size_t, error *C.int) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(ExtentCallback)
	if !ok {
		panic("inappropriate callback type")
	}
	go_error := int(*error)
	ret := callback(C.GoString(metacontext), uint64(offset), copy_uint32_array(entries, nr_entries), &go_error)
	*error = C.int(go_error)
	return C.int(ret)
}

type Extent64Callback func(metacontext string, offset uint64, entries []LibnbdExtent, error *int) int

//export extent64_callback
func extent64_callback(callbackid *C.long, metacontext *C.char, offset C.uint64_t, entries *C.nbd_extent, nr_entries C.size_t, error *C.int) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(Extent64Callback)
	if !ok {
		panic("inappropriate callback type")
	}
	go_error := int(*error)
	ret := callback(C.GoString(metacontext), uint64(offset), copy_extent_array(entries, nr_entries), &go_error)
	*error = C.int(go_error)
	return C.int(ret)
}

type ListCallback func(name string, description string) int

//export list_callback
func list_callback(callbackid *C.long, name *C.char, description *C.char) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(ListCallback)
	if !ok {
		panic("inappropriate callback type")
	}
	ret := callback(C.GoString(name), C.GoString(description))
	return C.int(ret)
}

type ContextCallback func(name string) int

//export context_callback
func context_callback(callbackid *C.long, name *C.char) C.int {
	callbackFunc := getCallbackId(int(*callbackid))
	callback, ok := callbackFunc.(ContextCallback)
	if !ok {
		panic("inappropriate callback type")
	}
	ret := callback(C.GoString(name))
	return C.int(ret)
}
