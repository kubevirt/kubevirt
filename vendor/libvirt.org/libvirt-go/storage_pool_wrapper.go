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
#include <assert.h>
#include "storage_pool_wrapper.h"

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virStoragePoolBuild(pool, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virStoragePoolCreate(pool, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = virStoragePoolDelete(pool, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    int ret = virStoragePoolDestroy(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err)
{
    int ret = virStoragePoolFree(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int *autostart,
                                  virErrorPtr err)
{
    int ret = virStoragePoolGetAutostart(pool, autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err)
{
    int ret = virStoragePoolGetInfo(pool, info);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    const char * ret = virStoragePoolGetName(pool);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char *uuid,
                             virErrorPtr err)
{
    int ret = virStoragePoolGetUUID(pool, uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char *buf,
                                   virErrorPtr err)
{
    int ret = virStoragePoolGetUUIDString(pool, buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = virStoragePoolGetXMLDesc(pool, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = virStoragePoolIsActive(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = virStoragePoolIsPersistent(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolListAllVolumesWrapper(virStoragePoolPtr pool,
                                    virStorageVolPtr **vols,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virStoragePoolListAllVolumes(pool, vols, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolListVolumesWrapper(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames,
                                 virErrorPtr err)
{
    int ret = virStoragePoolListVolumes(pool, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = virStoragePoolNumOfVolumes(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err)
{
    int ret = virStoragePoolRef(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = virStoragePoolRefresh(pool, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err)
{
    int ret = virStoragePoolSetAutostart(pool, autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = virStoragePoolUndefine(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virStorageVolPtr
virStorageVolCreateXMLWrapper(virStoragePoolPtr pool,
                              const char *xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virStorageVolPtr ret = virStorageVolCreateXML(pool, xmlDesc, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStorageVolPtr
virStorageVolCreateXMLFromWrapper(virStoragePoolPtr pool,
                                  const char *xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virStorageVolPtr ret = virStorageVolCreateXMLFrom(pool, xmlDesc, clonevol, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char *name,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = virStorageVolLookupByName(pool, name);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}



*/
import "C"
