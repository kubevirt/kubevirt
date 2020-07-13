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
 * Copyright (C) 2016-2019 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <stdlib.h>
#include "domain_checkpoint_wrapper.h"
*/
import "C"

import (
	"reflect"
	"unsafe"
)

type DomainCheckpointCreateFlags int

const (
	DOMAIN_CHECKPOINT_CREATE_REDEFINE = DomainCheckpointCreateFlags(C.VIR_DOMAIN_CHECKPOINT_CREATE_REDEFINE)
	DOMAIN_CHECKPOINT_CREATE_QUIESCE  = DomainCheckpointCreateFlags(C.VIR_DOMAIN_CHECKPOINT_CREATE_QUIESCE)
)

type DomainCheckpointListFlags int

const (
	DOMAIN_CHECKPOINT_LIST_ROOTS       = DomainCheckpointListFlags(C.VIR_DOMAIN_CHECKPOINT_LIST_ROOTS)
	DOMAIN_CHECKPOINT_LIST_DESCENDANTS = DomainCheckpointListFlags(C.VIR_DOMAIN_CHECKPOINT_LIST_DESCENDANTS)
	DOMAIN_CHECKPOINT_LIST_LEAVES      = DomainCheckpointListFlags(C.VIR_DOMAIN_CHECKPOINT_LIST_LEAVES)
	DOMAIN_CHECKPOINT_LIST_NO_LEAVES   = DomainCheckpointListFlags(C.VIR_DOMAIN_CHECKPOINT_LIST_NO_LEAVES)
	DOMAIN_CHECKPOINT_LIST_TOPOLOGICAL = DomainCheckpointListFlags(C.VIR_DOMAIN_CHECKPOINT_LIST_TOPOLOGICAL)
)

type DomainCheckpointDeleteFlags int

const (
	DOMAIN_CHECKPOINT_DELETE_CHILDREN      = DomainCheckpointDeleteFlags(C.VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN)
	DOMAIN_CHECKPOINT_DELETE_METADATA_ONLY = DomainCheckpointDeleteFlags(C.VIR_DOMAIN_CHECKPOINT_DELETE_METADATA_ONLY)
	DOMAIN_CHECKPOINT_DELETE_CHILDREN_ONLY = DomainCheckpointDeleteFlags(C.VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN_ONLY)
)

type DomainCheckpointXMLFlags int

const (
	DOMAIN_CHECKPOINT_XML_SECURE    = DomainCheckpointXMLFlags(C.VIR_DOMAIN_CHECKPOINT_XML_SECURE)
	DOMAIN_CHECKPOINT_XML_NO_DOMAIN = DomainCheckpointXMLFlags(C.VIR_DOMAIN_CHECKPOINT_XML_NO_DOMAIN)
	DOMAIN_CHECKPOINT_XML_SIZE      = DomainCheckpointXMLFlags(C.VIR_DOMAIN_CHECKPOINT_XML_SIZE)
)

type DomainCheckpoint struct {
	ptr C.virDomainCheckpointPtr
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointFree
func (s *DomainCheckpoint) Free() error {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return makeNotImplementedError("virDomainCheckpointFree")
	}

	var err C.virError
	ret := C.virDomainCheckpointFreeWrapper(s.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointRef
func (c *DomainCheckpoint) Ref() error {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return makeNotImplementedError("virDomainCheckpointRef")
	}

	var err C.virError
	ret := C.virDomainCheckpointRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointDelete
func (s *DomainCheckpoint) Delete(flags DomainCheckpointDeleteFlags) error {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return makeNotImplementedError("virDomainCheckpointDelete")
	}

	var err C.virError
	result := C.virDomainCheckpointDeleteWrapper(s.ptr, C.uint(flags), &err)
	if result != 0 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointGetXMLDesc
func (s *DomainCheckpoint) GetXMLDesc(flags DomainCheckpointXMLFlags) (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return "", makeNotImplementedError("virDomainCheckpointGetXMLDesc")
	}

	var err C.virError
	result := C.virDomainCheckpointGetXMLDescWrapper(s.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointGetName
func (s *DomainCheckpoint) GetName() (string, error) {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return "", makeNotImplementedError("virDomainCheckpointGetName")
	}

	var err C.virError
	name := C.virDomainCheckpointGetNameWrapper(s.ptr, &err)
	if name == nil {
		return "", makeError(&err)
	}
	return C.GoString(name), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointGetParent
func (s *DomainCheckpoint) GetParent(flags uint32) (*DomainCheckpoint, error) {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return nil, makeNotImplementedError("virDomainCheckpointGetParent")
	}

	var err C.virError
	ptr := C.virDomainCheckpointGetParentWrapper(s.ptr, C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &DomainCheckpoint{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-domain-checkpoint.html#virDomainCheckpointListAllChildren
func (d *DomainCheckpoint) ListAllChildren(flags DomainCheckpointListFlags) ([]DomainCheckpoint, error) {
	if C.LIBVIR_VERSION_NUMBER < 5006000 {
		return []DomainCheckpoint{}, makeNotImplementedError("virDomainCheckpointListAllChildren")
	}

	var cList *C.virDomainCheckpointPtr
	var err C.virError
	numVols := C.virDomainCheckpointListAllChildrenWrapper(d.ptr, (**C.virDomainCheckpointPtr)(&cList), C.uint(flags), &err)
	if numVols == -1 {
		return nil, makeError(&err)
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cList)),
		Len:  int(numVols),
		Cap:  int(numVols),
	}
	var pools []DomainCheckpoint
	slice := *(*[]C.virDomainCheckpointPtr)(unsafe.Pointer(&hdr))
	for _, ptr := range slice {
		pools = append(pools, DomainCheckpoint{ptr})
	}
	C.free(unsafe.Pointer(cList))
	return pools, nil
}
