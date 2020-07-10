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
#include <assert.h>
#include "storage_volume_wrapper.h"

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err)
{
    virStoragePoolPtr ret = virStoragePoolLookupByVolume(vol);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virStorageVolDelete(vol, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolDownloadWrapper(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virStorageVolDownload(vol, stream, offset, length, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err)
{
    int ret = virStorageVolFree(vol);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err)
{
    virConnectPtr ret = virStorageVolGetConnect(vol);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err)
{
    int ret = virStorageVolGetInfo(vol, info);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolGetInfoFlagsWrapper(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags,
                                 virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 3000000
    assert(0); // Caller should have checked version
#else
    int ret = virStorageVolGetInfoFlags(vol, info, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err)
{
    const char *ret = virStorageVolGetKey(vol);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    const char *ret = virStorageVolGetName(vol);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    char *ret = virStorageVolGetPath(vol);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err)
{
    char *ret = virStorageVolGetXMLDesc(vol, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err)
{
    int ret = virStorageVolRef(vol);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolResizeWrapper(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virStorageVolResize(vol, capacity, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolUploadWrapper(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virStorageVolUpload(vol, stream, offset, length, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = virStorageVolWipe(vol, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStorageVolWipePatternWrapper(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virStorageVolWipePattern(vol, algorithm, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
