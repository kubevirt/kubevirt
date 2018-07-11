/*
 * This file is part of the libvirt-go project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (C) 2018 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include <stdlib.h>
#include "nwfilter_binding_compat.h"
*/
import "C"

import (
	"unsafe"
)

type NWFilterBinding struct {
	ptr C.virNWFilterBindingPtr
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingFree
func (f *NWFilterBinding) Free() error {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return GetNotImplementedError("virNWFilterBindingFree")
	}
	ret := C.virNWFilterBindingFreeCompat(f.ptr)
	if ret == -1 {
		return GetLastError()
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingRef
func (c *NWFilterBinding) Ref() error {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return GetNotImplementedError("virNWFilterBindingRef")
	}
	ret := C.virNWFilterBindingRefCompat(c.ptr)
	if ret == -1 {
		return GetLastError()
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingDelete
func (f *NWFilterBinding) Delete() error {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return GetNotImplementedError("virNWFilterBindingDelete")
	}
	result := C.virNWFilterBindingDeleteCompat(f.ptr)
	if result == -1 {
		return GetLastError()
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingGetPortDev
func (f *NWFilterBinding) GetPortDev() (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return "", GetNotImplementedError("virNWFilterBindingGetPortDev")
	}
	result := C.virNWFilterBindingGetPortDevCompat(f.ptr)
	if result == nil {
		return "", GetLastError()
	}
	name := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return name, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingGetFilterName
func (f *NWFilterBinding) GetFilterName() (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return "", GetNotImplementedError("virNWFilterBindingGetFilterName")
	}
	result := C.virNWFilterBindingGetFilterNameCompat(f.ptr)
	if result == nil {
		return "", GetLastError()
	}
	name := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return name, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingGetXMLDesc
func (f *NWFilterBinding) GetXMLDesc(flags uint32) (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return "", GetNotImplementedError("virNWFilterBindingGetXMLDesc")
	}
	result := C.virNWFilterBindingGetXMLDescCompat(f.ptr, C.uint(flags))
	if result == nil {
		return "", GetLastError()
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}
