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
#include "storage_volume_wrapper.h"
*/
import "C"

import (
	"unsafe"
)

type StorageVolCreateFlags int

const (
	STORAGE_VOL_CREATE_PREALLOC_METADATA = StorageVolCreateFlags(C.VIR_STORAGE_VOL_CREATE_PREALLOC_METADATA)
	STORAGE_VOL_CREATE_REFLINK           = StorageVolCreateFlags(C.VIR_STORAGE_VOL_CREATE_REFLINK)
)

type StorageVolDeleteFlags int

const (
	STORAGE_VOL_DELETE_NORMAL         = StorageVolDeleteFlags(C.VIR_STORAGE_VOL_DELETE_NORMAL)         // Delete metadata only (fast)
	STORAGE_VOL_DELETE_ZEROED         = StorageVolDeleteFlags(C.VIR_STORAGE_VOL_DELETE_ZEROED)         // Clear all data to zeros (slow)
	STORAGE_VOL_DELETE_WITH_SNAPSHOTS = StorageVolDeleteFlags(C.VIR_STORAGE_VOL_DELETE_WITH_SNAPSHOTS) // Force removal of volume, even if in use
)

type StorageVolResizeFlags int

const (
	STORAGE_VOL_RESIZE_ALLOCATE = StorageVolResizeFlags(C.VIR_STORAGE_VOL_RESIZE_ALLOCATE) // force allocation of new size
	STORAGE_VOL_RESIZE_DELTA    = StorageVolResizeFlags(C.VIR_STORAGE_VOL_RESIZE_DELTA)    // size is relative to current
	STORAGE_VOL_RESIZE_SHRINK   = StorageVolResizeFlags(C.VIR_STORAGE_VOL_RESIZE_SHRINK)   // allow decrease in capacity
)

type StorageVolType int

const (
	STORAGE_VOL_FILE    = StorageVolType(C.VIR_STORAGE_VOL_FILE)    // Regular file based volumes
	STORAGE_VOL_BLOCK   = StorageVolType(C.VIR_STORAGE_VOL_BLOCK)   // Block based volumes
	STORAGE_VOL_DIR     = StorageVolType(C.VIR_STORAGE_VOL_DIR)     // Directory-passthrough based volume
	STORAGE_VOL_NETWORK = StorageVolType(C.VIR_STORAGE_VOL_NETWORK) //Network volumes like RBD (RADOS Block Device)
	STORAGE_VOL_NETDIR  = StorageVolType(C.VIR_STORAGE_VOL_NETDIR)  // Network accessible directory that can contain other network volumes
	STORAGE_VOL_PLOOP   = StorageVolType(C.VIR_STORAGE_VOL_PLOOP)   // Ploop directory based volumes
)

type StorageVolWipeAlgorithm int

const (
	STORAGE_VOL_WIPE_ALG_ZERO       = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_ZERO)       // 1-pass, all zeroes
	STORAGE_VOL_WIPE_ALG_NNSA       = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_NNSA)       // 4-pass NNSA Policy Letter NAP-14.1-C (XVI-8)
	STORAGE_VOL_WIPE_ALG_DOD        = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_DOD)        // 4-pass DoD 5220.22-M section 8-306 procedure
	STORAGE_VOL_WIPE_ALG_BSI        = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_BSI)        // 9-pass method recommended by the German Center of Security in Information Technologies
	STORAGE_VOL_WIPE_ALG_GUTMANN    = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_GUTMANN)    // The canonical 35-pass sequence
	STORAGE_VOL_WIPE_ALG_SCHNEIER   = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_SCHNEIER)   // 7-pass method described by Bruce Schneier in "Applied Cryptography" (1996)
	STORAGE_VOL_WIPE_ALG_PFITZNER7  = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_PFITZNER7)  // 7-pass random
	STORAGE_VOL_WIPE_ALG_PFITZNER33 = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_PFITZNER33) // 33-pass random
	STORAGE_VOL_WIPE_ALG_RANDOM     = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_RANDOM)     // 1-pass random
	STORAGE_VOL_WIPE_ALG_TRIM       = StorageVolWipeAlgorithm(C.VIR_STORAGE_VOL_WIPE_ALG_TRIM)       // Trim the underlying storage
)

type StorageXMLFlags int

const (
	STORAGE_XML_INACTIVE = StorageXMLFlags(C.VIR_STORAGE_XML_INACTIVE)
)

type StorageVolInfoFlags int

const (
	STORAGE_VOL_USE_ALLOCATION = StorageVolInfoFlags(C.VIR_STORAGE_VOL_USE_ALLOCATION)
	STORAGE_VOL_GET_PHYSICAL   = StorageVolInfoFlags(C.VIR_STORAGE_VOL_GET_PHYSICAL)
)

type StorageVolUploadFlags int

const (
	STORAGE_VOL_UPLOAD_SPARSE_STREAM = StorageVolUploadFlags(C.VIR_STORAGE_VOL_UPLOAD_SPARSE_STREAM)
)

type StorageVolDownloadFlags int

const (
	STORAGE_VOL_DOWNLOAD_SPARSE_STREAM = StorageVolDownloadFlags(C.VIR_STORAGE_VOL_DOWNLOAD_SPARSE_STREAM)
)

type StorageVol struct {
	ptr C.virStorageVolPtr
}

type StorageVolInfo struct {
	Type       StorageVolType
	Capacity   uint64
	Allocation uint64
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolDelete
func (v *StorageVol) Delete(flags StorageVolDeleteFlags) error {
	var err C.virError
	result := C.virStorageVolDeleteWrapper(v.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolFree
func (v *StorageVol) Free() error {
	var err C.virError
	ret := C.virStorageVolFreeWrapper(v.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolRef
func (c *StorageVol) Ref() error {
	var err C.virError
	ret := C.virStorageVolRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolGetInfo
func (v *StorageVol) GetInfo() (*StorageVolInfo, error) {
	var cinfo C.virStorageVolInfo
	var err C.virError
	result := C.virStorageVolGetInfoWrapper(v.ptr, &cinfo, &err)
	if result == -1 {
		return nil, makeError(&err)
	}
	return &StorageVolInfo{
		Type:       StorageVolType(cinfo._type),
		Capacity:   uint64(cinfo.capacity),
		Allocation: uint64(cinfo.allocation),
	}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolGetInfoFlags
func (v *StorageVol) GetInfoFlags(flags StorageVolInfoFlags) (*StorageVolInfo, error) {
	if C.LIBVIR_VERSION_NUMBER < 3000000 {
		return nil, makeNotImplementedError("virStorageVolGetInfoFlags")
	}

	var cinfo C.virStorageVolInfo
	var err C.virError
	result := C.virStorageVolGetInfoFlagsWrapper(v.ptr, &cinfo, C.uint(flags), &err)
	if result == -1 {
		return nil, makeError(&err)
	}
	return &StorageVolInfo{
		Type:       StorageVolType(cinfo._type),
		Capacity:   uint64(cinfo.capacity),
		Allocation: uint64(cinfo.allocation),
	}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolGetKey
func (v *StorageVol) GetKey() (string, error) {
	var err C.virError
	key := C.virStorageVolGetKeyWrapper(v.ptr, &err)
	if key == nil {
		return "", makeError(&err)
	}
	return C.GoString(key), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolGetName
func (v *StorageVol) GetName() (string, error) {
	var err C.virError
	name := C.virStorageVolGetNameWrapper(v.ptr, &err)
	if name == nil {
		return "", makeError(&err)
	}
	return C.GoString(name), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolGetPath
func (v *StorageVol) GetPath() (string, error) {
	var err C.virError
	result := C.virStorageVolGetPathWrapper(v.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	path := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return path, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolGetXMLDesc
func (v *StorageVol) GetXMLDesc(flags uint32) (string, error) {
	var err C.virError
	result := C.virStorageVolGetXMLDescWrapper(v.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolResize
func (v *StorageVol) Resize(capacity uint64, flags StorageVolResizeFlags) error {
	var err C.virError
	result := C.virStorageVolResizeWrapper(v.ptr, C.ulonglong(capacity), C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolWipe
func (v *StorageVol) Wipe(flags uint32) error {
	var err C.virError
	result := C.virStorageVolWipeWrapper(v.ptr, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolWipePattern
func (v *StorageVol) WipePattern(algorithm StorageVolWipeAlgorithm, flags uint32) error {
	var err C.virError
	result := C.virStorageVolWipePatternWrapper(v.ptr, C.uint(algorithm), C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolUpload
func (v *StorageVol) Upload(stream *Stream, offset, length uint64, flags StorageVolUploadFlags) error {
	var err C.virError
	if C.virStorageVolUploadWrapper(v.ptr, stream.ptr, C.ulonglong(offset),
		C.ulonglong(length), C.uint(flags), &err) == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStorageVolDownload
func (v *StorageVol) Download(stream *Stream, offset, length uint64, flags StorageVolDownloadFlags) error {
	var err C.virError
	if C.virStorageVolDownloadWrapper(v.ptr, stream.ptr, C.ulonglong(offset),
		C.ulonglong(length), C.uint(flags), &err) == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-storage.html#virStoragePoolLookupByVolume
func (v *StorageVol) LookupPoolByVolume() (*StoragePool, error) {
	var err C.virError
	poolPtr := C.virStoragePoolLookupByVolumeWrapper(v.ptr, &err)
	if poolPtr == nil {
		return nil, makeError(&err)
	}
	return &StoragePool{ptr: poolPtr}, nil
}
