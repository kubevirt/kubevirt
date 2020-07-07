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
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <stdlib.h>
#include "interface_wrapper.h"
*/
import "C"

import (
	"unsafe"
)

type InterfaceXMLFlags int

const (
	INTERFACE_XML_INACTIVE = InterfaceXMLFlags(C.VIR_INTERFACE_XML_INACTIVE)
)

type Interface struct {
	ptr C.virInterfacePtr
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceCreate
func (n *Interface) Create(flags uint32) error {
	var err C.virError
	result := C.virInterfaceCreateWrapper(n.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceDestroy
func (n *Interface) Destroy(flags uint32) error {
	var err C.virError
	result := C.virInterfaceDestroyWrapper(n.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceIsActive
func (n *Interface) IsActive() (bool, error) {
	var err C.virError
	result := C.virInterfaceIsActiveWrapper(n.ptr, &err)
	if result == -1 {
		return false, makeError(&err)
	}
	if result == 1 {
		return true, nil
	}
	return false, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceGetMACString
func (n *Interface) GetMACString() (string, error) {
	var err C.virError
	result := C.virInterfaceGetMACStringWrapper(n.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	mac := C.GoString(result)
	return mac, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceGetName
func (n *Interface) GetName() (string, error) {
	var err C.virError
	result := C.virInterfaceGetNameWrapper(n.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	name := C.GoString(result)
	return name, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceGetXMLDesc
func (n *Interface) GetXMLDesc(flags InterfaceXMLFlags) (string, error) {
	var err C.virError
	result := C.virInterfaceGetXMLDescWrapper(n.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceUndefine
func (n *Interface) Undefine() error {
	var err C.virError
	result := C.virInterfaceUndefineWrapper(n.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceFree
func (n *Interface) Free() error {
	var err C.virError
	ret := C.virInterfaceFreeWrapper(n.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-interface.html#virInterfaceRef
func (c *Interface) Ref() error {
	var err C.virError
	ret := C.virInterfaceRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}
