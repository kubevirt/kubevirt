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
#include <stdlib.h>
#include "nwfilter_binding_wrapper.h"
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
		return makeNotImplementedError("virNWFilterBindingFree")
	}
	var err C.virError
	ret := C.virNWFilterBindingFreeWrapper(f.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingRef
func (c *NWFilterBinding) Ref() error {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return makeNotImplementedError("virNWFilterBindingRef")
	}
	var err C.virError
	ret := C.virNWFilterBindingRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingDelete
func (f *NWFilterBinding) Delete() error {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return makeNotImplementedError("virNWFilterBindingDelete")
	}
	var err C.virError
	result := C.virNWFilterBindingDeleteWrapper(f.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingGetPortDev
func (f *NWFilterBinding) GetPortDev() (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return "", makeNotImplementedError("virNWFilterBindingGetPortDev")
	}
	var err C.virError
	result := C.virNWFilterBindingGetPortDevWrapper(f.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	name := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return name, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingGetFilterName
func (f *NWFilterBinding) GetFilterName() (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return "", makeNotImplementedError("virNWFilterBindingGetFilterName")
	}
	var err C.virError
	result := C.virNWFilterBindingGetFilterNameWrapper(f.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	name := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return name, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-nwfilter.html#virNWFilterBindingGetXMLDesc
func (f *NWFilterBinding) GetXMLDesc(flags uint32) (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 4005000 {
		return "", makeNotImplementedError("virNWFilterBindingGetXMLDesc")
	}
	var err C.virError
	result := C.virNWFilterBindingGetXMLDescWrapper(f.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}
