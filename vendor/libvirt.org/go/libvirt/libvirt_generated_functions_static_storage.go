//go:build !libvirt_dlopen
// +build !libvirt_dlopen

/*
 * This file is part of the libvirt-go-module project
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
 * Copyright (C) 2022 Red Hat, Inc.
 *
 */
/****************************************************************************
 * THIS CODE HAS BEEN GENERATED. DO NOT CHANGE IT DIRECTLY                  *
 ****************************************************************************/

package libvirt

/*
#cgo pkg-config: libvirt
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_generated.h"
#include "error_helper.h"


char *
virConnectFindStoragePoolSourcesWrapper(virConnectPtr conn,
                                        const char * type,
                                        const char * srcSpec,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 5)
    setVirError(err, "Function virConnectFindStoragePoolSources not available prior to libvirt version 0.4.5");
#else
    ret = virConnectFindStoragePoolSources(conn,
                                           type,
                                           srcSpec,
                                           flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectGetStoragePoolCapabilitiesWrapper(virConnectPtr conn,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 2, 0)
    setVirError(err, "Function virConnectGetStoragePoolCapabilities not available prior to libvirt version 5.2.0");
#else
    ret = virConnectGetStoragePoolCapabilities(conn,
                                               flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListAllStoragePoolsWrapper(virConnectPtr conn,
                                     virStoragePoolPtr ** pools,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllStoragePools not available prior to libvirt version 0.10.2");
#else
    ret = virConnectListAllStoragePools(conn,
                                        pools,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListDefinedStoragePoolsWrapper(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames,
                                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectListDefinedStoragePools not available prior to libvirt version 0.4.1");
#else
    ret = virConnectListDefinedStoragePools(conn,
                                            names,
                                            maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListStoragePoolsWrapper(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectListStoragePools not available prior to libvirt version 0.4.1");
#else
    ret = virConnectListStoragePools(conn,
                                     names,
                                     maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfDefinedStoragePoolsWrapper(virConnectPtr conn,
                                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectNumOfDefinedStoragePools not available prior to libvirt version 0.4.1");
#else
    ret = virConnectNumOfDefinedStoragePools(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfStoragePoolsWrapper(virConnectPtr conn,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectNumOfStoragePools not available prior to libvirt version 0.4.1");
#else
    ret = virConnectNumOfStoragePools(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectStoragePoolEventDeregisterAnyWrapper(virConnectPtr conn,
                                               int callbackID,
                                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virConnectStoragePoolEventDeregisterAny not available prior to libvirt version 2.0.0");
#else
    ret = virConnectStoragePoolEventDeregisterAny(conn,
                                                  callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectStoragePoolEventRegisterAnyWrapper(virConnectPtr conn,
                                             virStoragePoolPtr pool,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             void * opaque,
                                             virFreeCallback freecb,
                                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virConnectStoragePoolEventRegisterAny not available prior to libvirt version 2.0.0");
#else
    ret = virConnectStoragePoolEventRegisterAny(conn,
                                                pool,
                                                eventID,
                                                cb,
                                                opaque,
                                                freecb);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolBuild not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolBuild(pool,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolCreate not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolCreate(pool,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolCreateXMLWrapper(virConnectPtr conn,
                               const char * xmlDesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolCreateXML not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolCreateXML(conn,
                                  xmlDesc,
                                  flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolDefineXMLWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolDefineXML not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolDefineXML(conn,
                                  xml,
                                  flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolDelete not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolDelete(pool,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolDestroy not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolDestroy(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolFree not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolFree(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int * autostart,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetAutostart not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetAutostart(pool,
                                     autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virStoragePoolGetConnectWrapper(virStoragePoolPtr pool,
                                virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetConnect not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetConnect(pool);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetInfo not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetInfo(pool,
                                info);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetName not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetName(pool);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetUUID not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetUUID(pool,
                                uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char * buf,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetUUIDString not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetUUIDString(pool,
                                      buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetXMLDesc not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolGetXMLDesc(pool,
                                   flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virStoragePoolIsActive not available prior to libvirt version 0.7.3");
#else
    ret = virStoragePoolIsActive(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virStoragePoolIsPersistent not available prior to libvirt version 0.7.3");
#else
    ret = virStoragePoolIsPersistent(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolListAllVolumesWrapper(virStoragePoolPtr pool,
                                    virStorageVolPtr ** vols,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virStoragePoolListAllVolumes not available prior to libvirt version 0.10.2");
#else
    ret = virStoragePoolListAllVolumes(pool,
                                       vols,
                                       flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolListVolumesWrapper(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolListVolumes not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolListVolumes(pool,
                                    names,
                                    maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char * name,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByName not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolLookupByName(conn,
                                     name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char * path,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(4, 1, 0)
    setVirError(err, "Function virStoragePoolLookupByTargetPath not available prior to libvirt version 4.1.0");
#else
    ret = virStoragePoolLookupByTargetPath(conn,
                                           path);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByUUID not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolLookupByUUID(conn,
                                     uuid);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByUUIDString not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolLookupByUUIDString(conn,
                                           uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByVolume not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolLookupByVolume(vol);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolNumOfVolumes not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolNumOfVolumes(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virStoragePoolRef not available prior to libvirt version 0.6.0");
#else
    ret = virStoragePoolRef(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolRefresh not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolRefresh(pool,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolSetAutostart not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolSetAutostart(pool,
                                     autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolUndefine not available prior to libvirt version 0.4.1");
#else
    ret = virStoragePoolUndefine(pool);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStorageVolPtr
virStorageVolCreateXMLWrapper(virStoragePoolPtr pool,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolCreateXML not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolCreateXML(pool,
                                 xmlDesc,
                                 flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStorageVolPtr
virStorageVolCreateXMLFromWrapper(virStoragePoolPtr pool,
                                  const char * xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virStorageVolCreateXMLFrom not available prior to libvirt version 0.6.4");
#else
    ret = virStorageVolCreateXMLFrom(pool,
                                     xmlDesc,
                                     clonevol,
                                     flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolDelete not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolDelete(vol,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
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
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virStorageVolDownload not available prior to libvirt version 0.9.0");
#else
    ret = virStorageVolDownload(vol,
                                stream,
                                offset,
                                length,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolFree not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolFree(vol);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetConnect not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolGetConnect(vol);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetInfo not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolGetInfo(vol,
                               info);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolGetInfoFlagsWrapper(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virStorageVolGetInfoFlags not available prior to libvirt version 3.0.0");
#else
    ret = virStorageVolGetInfoFlags(vol,
                                    info,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetKey not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolGetKey(vol);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetName not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolGetName(vol);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetPath not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolGetPath(vol);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetXMLDesc not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolGetXMLDesc(vol,
                                  flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char * key,
                                virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolLookupByKey not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolLookupByKey(conn,
                                   key);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char * name,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolLookupByName not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolLookupByName(pool,
                                    name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char * path,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolLookupByPath not available prior to libvirt version 0.4.1");
#else
    ret = virStorageVolLookupByPath(conn,
                                    path);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virStorageVolRef not available prior to libvirt version 0.6.0");
#else
    ret = virStorageVolRef(vol);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolResizeWrapper(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virStorageVolResize not available prior to libvirt version 0.9.10");
#else
    ret = virStorageVolResize(vol,
                              capacity,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
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
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virStorageVolUpload not available prior to libvirt version 0.9.0");
#else
    ret = virStorageVolUpload(vol,
                              stream,
                              offset,
                              length,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virStorageVolWipe not available prior to libvirt version 0.8.0");
#else
    ret = virStorageVolWipe(vol,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStorageVolWipePatternWrapper(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virStorageVolWipePattern not available prior to libvirt version 0.9.10");
#else
    ret = virStorageVolWipePattern(vol,
                                   algorithm,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
