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
#include "storage_pool_wrapper.h"
*/
import "C"

import (
	"reflect"
	"unsafe"
)

type StoragePoolState int

const (
	STORAGE_POOL_INACTIVE     = StoragePoolState(C.VIR_STORAGE_POOL_INACTIVE)     // Not running
	STORAGE_POOL_BUILDING     = StoragePoolState(C.VIR_STORAGE_POOL_BUILDING)     // Initializing pool,not available
	STORAGE_POOL_RUNNING      = StoragePoolState(C.VIR_STORAGE_POOL_RUNNING)      // Running normally
	STORAGE_POOL_DEGRADED     = StoragePoolState(C.VIR_STORAGE_POOL_DEGRADED)     // Running degraded
	STORAGE_POOL_INACCESSIBLE = StoragePoolState(C.VIR_STORAGE_POOL_INACCESSIBLE) // Running,but not accessible
)

type StoragePoolBuildFlags int

const (
	STORAGE_POOL_BUILD_NEW          = StoragePoolBuildFlags(C.VIR_STORAGE_POOL_BUILD_NEW)          // Regular build from scratch
	STORAGE_POOL_BUILD_REPAIR       = StoragePoolBuildFlags(C.VIR_STORAGE_POOL_BUILD_REPAIR)       // Repair / reinitialize
	STORAGE_POOL_BUILD_RESIZE       = StoragePoolBuildFlags(C.VIR_STORAGE_POOL_BUILD_RESIZE)       // Extend existing pool
	STORAGE_POOL_BUILD_NO_OVERWRITE = StoragePoolBuildFlags(C.VIR_STORAGE_POOL_BUILD_NO_OVERWRITE) // Do not overwrite existing pool
	STORAGE_POOL_BUILD_OVERWRITE    = StoragePoolBuildFlags(C.VIR_STORAGE_POOL_BUILD_OVERWRITE)    // Overwrite data
)

type StoragePoolCreateFlags int

const (
	STORAGE_POOL_CREATE_NORMAL                  = StoragePoolCreateFlags(C.VIR_STORAGE_POOL_CREATE_NORMAL)
	STORAGE_POOL_CREATE_WITH_BUILD              = StoragePoolCreateFlags(C.VIR_STORAGE_POOL_CREATE_WITH_BUILD)
	STORAGE_POOL_CREATE_WITH_BUILD_OVERWRITE    = StoragePoolCreateFlags(C.VIR_STORAGE_POOL_CREATE_WITH_BUILD_OVERWRITE)
	STORAGE_POOL_CREATE_WITH_BUILD_NO_OVERWRITE = StoragePoolCreateFlags(C.VIR_STORAGE_POOL_CREATE_WITH_BUILD_NO_OVERWRITE)
)

type StoragePoolDeleteFlags int

const (
	STORAGE_POOL_DELETE_NORMAL = StoragePoolDeleteFlags(C.VIR_STORAGE_POOL_DELETE_NORMAL)
	STORAGE_POOL_DELETE_ZEROED = StoragePoolDeleteFlags(C.VIR_STORAGE_POOL_DELETE_ZEROED)
)

type StoragePoolEventID int

const (
	STORAGE_POOL_EVENT_ID_LIFECYCLE = StoragePoolEventID(C.VIR_STORAGE_POOL_EVENT_ID_LIFECYCLE)
	STORAGE_POOL_EVENT_ID_REFRESH   = StoragePoolEventID(C.VIR_STORAGE_POOL_EVENT_ID_REFRESH)
)

type StoragePoolEventLifecycleType int

const (
	STORAGE_POOL_EVENT_DEFINED   = StoragePoolEventLifecycleType(C.VIR_STORAGE_POOL_EVENT_DEFINED)
	STORAGE_POOL_EVENT_UNDEFINED = StoragePoolEventLifecycleType(C.VIR_STORAGE_POOL_EVENT_UNDEFINED)
	STORAGE_POOL_EVENT_STARTED   = StoragePoolEventLifecycleType(C.VIR_STORAGE_POOL_EVENT_STARTED)
	STORAGE_POOL_EVENT_STOPPED   = StoragePoolEventLifecycleType(C.VIR_STORAGE_POOL_EVENT_STOPPED)
	STORAGE_POOL_EVENT_CREATED   = StoragePoolEventLifecycleType(C.VIR_STORAGE_POOL_EVENT_CREATED)
	STORAGE_POOL_EVENT_DELETED   = StoragePoolEventLifecycleType(C.VIR_STORAGE_POOL_EVENT_DELETED)
)

type StoragePool struct {
	ptr C.virStoragePoolPtr
}

type StoragePoolInfo struct {
	State      StoragePoolState
	Capacity   uint64
	Allocation uint64
	Available  uint64
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolBuild
func (p *StoragePool) Build(flags StoragePoolBuildFlags) error {
	var err C.virError
	result := C.virStoragePoolBuildWrapper(p.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolCreate
func (p *StoragePool) Create(flags StoragePoolCreateFlags) error {
	var err C.virError
	result := C.virStoragePoolCreateWrapper(p.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolDelete
func (p *StoragePool) Delete(flags StoragePoolDeleteFlags) error {
	var err C.virError
	result := C.virStoragePoolDeleteWrapper(p.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolDestroy
func (p *StoragePool) Destroy() error {
	var err C.virError
	result := C.virStoragePoolDestroyWrapper(p.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolFree
func (p *StoragePool) Free() error {
	var err C.virError
	ret := C.virStoragePoolFreeWrapper(p.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolRef
func (c *StoragePool) Ref() error {
	var err C.virError
	ret := C.virStoragePoolRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolGetAutostart
func (p *StoragePool) GetAutostart() (bool, error) {
	var out C.int
	var err C.virError
	result := C.virStoragePoolGetAutostartWrapper(p.ptr, (*C.int)(unsafe.Pointer(&out)), &err)
	if result == -1 {
		return false, makeError(&err)
	}
	switch out {
	case 1:
		return true, nil
	default:
		return false, nil
	}
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolGetInfo
func (p *StoragePool) GetInfo() (*StoragePoolInfo, error) {
	var cinfo C.virStoragePoolInfo
	var err C.virError
	result := C.virStoragePoolGetInfoWrapper(p.ptr, &cinfo, &err)
	if result == -1 {
		return nil, makeError(&err)
	}
	return &StoragePoolInfo{
		State:      StoragePoolState(cinfo.state),
		Capacity:   uint64(cinfo.capacity),
		Allocation: uint64(cinfo.allocation),
		Available:  uint64(cinfo.available),
	}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolGetName
func (p *StoragePool) GetName() (string, error) {
	var err C.virError
	name := C.virStoragePoolGetNameWrapper(p.ptr, &err)
	if name == nil {
		return "", makeError(&err)
	}
	return C.GoString(name), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolGetUUID
func (p *StoragePool) GetUUID() ([]byte, error) {
	var cUuid [C.VIR_UUID_BUFLEN](byte)
	cuidPtr := unsafe.Pointer(&cUuid)
	var err C.virError
	result := C.virStoragePoolGetUUIDWrapper(p.ptr, (*C.uchar)(cuidPtr), &err)
	if result != 0 {
		return []byte{}, makeError(&err)
	}
	return C.GoBytes(cuidPtr, C.VIR_UUID_BUFLEN), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolGetUUIDString
func (p *StoragePool) GetUUIDString() (string, error) {
	var cUuid [C.VIR_UUID_STRING_BUFLEN](C.char)
	cuidPtr := unsafe.Pointer(&cUuid)
	var err C.virError
	result := C.virStoragePoolGetUUIDStringWrapper(p.ptr, (*C.char)(cuidPtr), &err)
	if result != 0 {
		return "", makeError(&err)
	}
	return C.GoString((*C.char)(cuidPtr)), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolGetXMLDesc
func (p *StoragePool) GetXMLDesc(flags StorageXMLFlags) (string, error) {
	var err C.virError
	result := C.virStoragePoolGetXMLDescWrapper(p.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolIsActive
func (p *StoragePool) IsActive() (bool, error) {
	var err C.virError
	result := C.virStoragePoolIsActiveWrapper(p.ptr, &err)
	if result == -1 {
		return false, makeError(&err)
	}
	if result == 1 {
		return true, nil
	}
	return false, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolIsPersistent
func (p *StoragePool) IsPersistent() (bool, error) {
	var err C.virError
	result := C.virStoragePoolIsPersistentWrapper(p.ptr, &err)
	if result == -1 {
		return false, makeError(&err)
	}
	if result == 1 {
		return true, nil
	}
	return false, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolSetAutostart
func (p *StoragePool) SetAutostart(autostart bool) error {
	var cAutostart C.int
	switch autostart {
	case true:
		cAutostart = 1
	default:
		cAutostart = 0
	}
	var err C.virError
	result := C.virStoragePoolSetAutostartWrapper(p.ptr, cAutostart, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolRefresh
func (p *StoragePool) Refresh(flags uint32) error {
	var err C.virError
	result := C.virStoragePoolRefreshWrapper(p.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolUndefine
func (p *StoragePool) Undefine() error {
	var err C.virError
	result := C.virStoragePoolUndefineWrapper(p.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolCreateXML
func (p *StoragePool) StorageVolCreateXML(xmlConfig string, flags StorageVolCreateFlags) (*StorageVol, error) {
	cXml := C.CString(string(xmlConfig))
	defer C.free(unsafe.Pointer(cXml))
	var err C.virError
	ptr := C.virStorageVolCreateXMLWrapper(p.ptr, cXml, C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &StorageVol{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolCreateXMLFrom
func (p *StoragePool) StorageVolCreateXMLFrom(xmlConfig string, clonevol *StorageVol, flags StorageVolCreateFlags) (*StorageVol, error) {
	cXml := C.CString(string(xmlConfig))
	defer C.free(unsafe.Pointer(cXml))
	var err C.virError
	ptr := C.virStorageVolCreateXMLFromWrapper(p.ptr, cXml, clonevol.ptr, C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &StorageVol{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolLookupByName
func (p *StoragePool) LookupStorageVolByName(name string) (*StorageVol, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	var err C.virError
	ptr := C.virStorageVolLookupByNameWrapper(p.ptr, cName, &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &StorageVol{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolNumOfVolumes
func (p *StoragePool) NumOfStorageVolumes() (int, error) {
	var err C.virError
	result := int(C.virStoragePoolNumOfVolumesWrapper(p.ptr, &err))
	if result == -1 {
		return 0, makeError(&err)
	}
	return result, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolListVolumes
func (p *StoragePool) ListStorageVolumes() ([]string, error) {
	const maxVols = 1024
	var names [maxVols](*C.char)
	namesPtr := unsafe.Pointer(&names)
	var err C.virError
	numStorageVols := C.virStoragePoolListVolumesWrapper(
		p.ptr,
		(**C.char)(namesPtr),
		maxVols, &err)
	if numStorageVols == -1 {
		return nil, makeError(&err)
	}
	goNames := make([]string, numStorageVols)
	for k := 0; k < int(numStorageVols); k++ {
		goNames[k] = C.GoString(names[k])
		C.free(unsafe.Pointer(names[k]))
	}
	return goNames, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolListAllVolumes
func (p *StoragePool) ListAllStorageVolumes(flags uint32) ([]StorageVol, error) {
	var cList *C.virStorageVolPtr
	var err C.virError
	numVols := C.virStoragePoolListAllVolumesWrapper(p.ptr, (**C.virStorageVolPtr)(&cList), C.uint(flags), &err)
	if numVols == -1 {
		return nil, makeError(&err)
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cList)),
		Len:  int(numVols),
		Cap:  int(numVols),
	}
	var pools []StorageVol
	slice := *(*[]C.virStorageVolPtr)(unsafe.Pointer(&hdr))
	for _, ptr := range slice {
		pools = append(pools, StorageVol{ptr})
	}
	C.free(unsafe.Pointer(cList))
	return pools, nil
}
