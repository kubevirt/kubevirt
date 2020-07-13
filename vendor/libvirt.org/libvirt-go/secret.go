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
#include "secret_wrapper.h"
*/
import "C"

import (
	"unsafe"
)

type SecretUsageType int

const (
	SECRET_USAGE_TYPE_NONE   = SecretUsageType(C.VIR_SECRET_USAGE_TYPE_NONE)
	SECRET_USAGE_TYPE_VOLUME = SecretUsageType(C.VIR_SECRET_USAGE_TYPE_VOLUME)
	SECRET_USAGE_TYPE_CEPH   = SecretUsageType(C.VIR_SECRET_USAGE_TYPE_CEPH)
	SECRET_USAGE_TYPE_ISCSI  = SecretUsageType(C.VIR_SECRET_USAGE_TYPE_ISCSI)
	SECRET_USAGE_TYPE_TLS    = SecretUsageType(C.VIR_SECRET_USAGE_TYPE_TLS)
	SECRET_USAGE_TYPE_VTPM   = SecretUsageType(C.VIR_SECRET_USAGE_TYPE_VTPM)
)

type SecretEventLifecycleType int

const (
	SECRET_EVENT_DEFINED   = SecretEventLifecycleType(C.VIR_SECRET_EVENT_DEFINED)
	SECRET_EVENT_UNDEFINED = SecretEventLifecycleType(C.VIR_SECRET_EVENT_UNDEFINED)
)

type SecretEventID int

const (
	SECRET_EVENT_ID_LIFECYCLE     = SecretEventID(C.VIR_SECRET_EVENT_ID_LIFECYCLE)
	SECRET_EVENT_ID_VALUE_CHANGED = SecretEventID(C.VIR_SECRET_EVENT_ID_VALUE_CHANGED)
)

type Secret struct {
	ptr C.virSecretPtr
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretFree
func (s *Secret) Free() error {
	var err C.virError
	ret := C.virSecretFreeWrapper(s.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretRef
func (c *Secret) Ref() error {
	var err C.virError
	ret := C.virSecretRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretUndefine
func (s *Secret) Undefine() error {
	var err C.virError
	result := C.virSecretUndefineWrapper(s.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretGetUUID
func (s *Secret) GetUUID() ([]byte, error) {
	var cUuid [C.VIR_UUID_BUFLEN](byte)
	cuidPtr := unsafe.Pointer(&cUuid)
	var err C.virError
	result := C.virSecretGetUUIDWrapper(s.ptr, (*C.uchar)(cuidPtr), &err)
	if result != 0 {
		return []byte{}, makeError(&err)
	}
	return C.GoBytes(cuidPtr, C.VIR_UUID_BUFLEN), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretGetUUIDString
func (s *Secret) GetUUIDString() (string, error) {
	var cUuid [C.VIR_UUID_STRING_BUFLEN](C.char)
	cuidPtr := unsafe.Pointer(&cUuid)
	var err C.virError
	result := C.virSecretGetUUIDStringWrapper(s.ptr, (*C.char)(cuidPtr), &err)
	if result != 0 {
		return "", makeError(&err)
	}
	return C.GoString((*C.char)(cuidPtr)), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretGetUsageID
func (s *Secret) GetUsageID() (string, error) {
	var err C.virError
	result := C.virSecretGetUsageIDWrapper(s.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	return C.GoString(result), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretGetUsageType
func (s *Secret) GetUsageType() (SecretUsageType, error) {
	var err C.virError
	result := SecretUsageType(C.virSecretGetUsageTypeWrapper(s.ptr, &err))
	if result == -1 {
		return 0, makeError(&err)
	}
	return result, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretGetXMLDesc
func (s *Secret) GetXMLDesc(flags uint32) (string, error) {
	var err C.virError
	result := C.virSecretGetXMLDescWrapper(s.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretGetValue
func (s *Secret) GetValue(flags uint32) ([]byte, error) {
	var cvalue_size C.size_t

	var err C.virError
	cvalue := C.virSecretGetValueWrapper(s.ptr, &cvalue_size, C.uint(flags), &err)
	if cvalue == nil {
		return nil, makeError(&err)
	}
	defer C.free(unsafe.Pointer(cvalue))
	ret := C.GoBytes(unsafe.Pointer(cvalue), C.int(cvalue_size))
	return ret, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-secret.html#virSecretSetValue
func (s *Secret) SetValue(value []byte, flags uint32) error {
	cvalue := make([]C.uchar, len(value))

	for i := 0; i < len(value); i++ {
		cvalue[i] = C.uchar(value[i])
	}

	var err C.virError
	result := C.virSecretSetValueWrapper(s.ptr, &cvalue[0], C.size_t(len(value)), C.uint(flags), &err)

	if result == -1 {
		return makeError(&err)
	}

	return nil
}
