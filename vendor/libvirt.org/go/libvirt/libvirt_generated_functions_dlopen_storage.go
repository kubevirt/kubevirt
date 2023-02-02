//go:build libvirt_dlopen
// +build libvirt_dlopen

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
#cgo LDFLAGS: -ldl
#cgo CFLAGS: -DLIBVIRT_DLOPEN
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_generated_dlopen.h"
#include "error_helper.h"


typedef char *
(*virConnectFindStoragePoolSourcesType)(virConnectPtr conn,
                                        const char * type,
                                        const char * srcSpec,
                                        unsigned int flags);

char *
virConnectFindStoragePoolSourcesWrapper(virConnectPtr conn,
                                        const char * type,
                                        const char * srcSpec,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    char * ret = NULL;
    static virConnectFindStoragePoolSourcesType virConnectFindStoragePoolSourcesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectFindStoragePoolSources",
                       (void**)&virConnectFindStoragePoolSourcesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectFindStoragePoolSourcesSymbol(conn,
                                                 type,
                                                 srcSpec,
                                                 flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetStoragePoolCapabilitiesType)(virConnectPtr conn,
                                            unsigned int flags);

char *
virConnectGetStoragePoolCapabilitiesWrapper(virConnectPtr conn,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    char * ret = NULL;
    static virConnectGetStoragePoolCapabilitiesType virConnectGetStoragePoolCapabilitiesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetStoragePoolCapabilities",
                       (void**)&virConnectGetStoragePoolCapabilitiesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetStoragePoolCapabilitiesSymbol(conn,
                                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListAllStoragePoolsType)(virConnectPtr conn,
                                     virStoragePoolPtr ** pools,
                                     unsigned int flags);

int
virConnectListAllStoragePoolsWrapper(virConnectPtr conn,
                                     virStoragePoolPtr ** pools,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllStoragePoolsType virConnectListAllStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllStoragePools",
                       (void**)&virConnectListAllStoragePoolsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllStoragePoolsSymbol(conn,
                                              pools,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDefinedStoragePoolsType)(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames);

int
virConnectListDefinedStoragePoolsWrapper(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames,
                                         virErrorPtr err)
{
    int ret = -1;
    static virConnectListDefinedStoragePoolsType virConnectListDefinedStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListDefinedStoragePools",
                       (void**)&virConnectListDefinedStoragePoolsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListDefinedStoragePoolsSymbol(conn,
                                                  names,
                                                  maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListStoragePoolsType)(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames);

int
virConnectListStoragePoolsWrapper(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames,
                                  virErrorPtr err)
{
    int ret = -1;
    static virConnectListStoragePoolsType virConnectListStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListStoragePools",
                       (void**)&virConnectListStoragePoolsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListStoragePoolsSymbol(conn,
                                           names,
                                           maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDefinedStoragePoolsType)(virConnectPtr conn);

int
virConnectNumOfDefinedStoragePoolsWrapper(virConnectPtr conn,
                                          virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDefinedStoragePoolsType virConnectNumOfDefinedStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfDefinedStoragePools",
                       (void**)&virConnectNumOfDefinedStoragePoolsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfDefinedStoragePoolsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfStoragePoolsType)(virConnectPtr conn);

int
virConnectNumOfStoragePoolsWrapper(virConnectPtr conn,
                                   virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfStoragePoolsType virConnectNumOfStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfStoragePools",
                       (void**)&virConnectNumOfStoragePoolsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfStoragePoolsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectStoragePoolEventDeregisterAnyType)(virConnectPtr conn,
                                               int callbackID);

int
virConnectStoragePoolEventDeregisterAnyWrapper(virConnectPtr conn,
                                               int callbackID,
                                               virErrorPtr err)
{
    int ret = -1;
    static virConnectStoragePoolEventDeregisterAnyType virConnectStoragePoolEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectStoragePoolEventDeregisterAny",
                       (void**)&virConnectStoragePoolEventDeregisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectStoragePoolEventDeregisterAnySymbol(conn,
                                                        callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectStoragePoolEventRegisterAnyType)(virConnectPtr conn,
                                             virStoragePoolPtr pool,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             void * opaque,
                                             virFreeCallback freecb);

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
    static virConnectStoragePoolEventRegisterAnyType virConnectStoragePoolEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectStoragePoolEventRegisterAny",
                       (void**)&virConnectStoragePoolEventRegisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectStoragePoolEventRegisterAnySymbol(conn,
                                                      pool,
                                                      eventID,
                                                      cb,
                                                      opaque,
                                                      freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolBuildType)(virStoragePoolPtr pool,
                           unsigned int flags);

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolBuildType virStoragePoolBuildSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolBuild",
                       (void**)&virStoragePoolBuildSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolBuildSymbol(pool,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolCreateType)(virStoragePoolPtr pool,
                            unsigned int flags);

int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolCreateType virStoragePoolCreateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolCreate",
                       (void**)&virStoragePoolCreateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolCreateSymbol(pool,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolCreateXMLType)(virConnectPtr conn,
                               const char * xmlDesc,
                               unsigned int flags);

virStoragePoolPtr
virStoragePoolCreateXMLWrapper(virConnectPtr conn,
                               const char * xmlDesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolCreateXMLType virStoragePoolCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolCreateXML",
                       (void**)&virStoragePoolCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolCreateXMLSymbol(conn,
                                        xmlDesc,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolDefineXMLType)(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags);

virStoragePoolPtr
virStoragePoolDefineXMLWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolDefineXMLType virStoragePoolDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolDefineXML",
                       (void**)&virStoragePoolDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolDefineXMLSymbol(conn,
                                        xml,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolDeleteType)(virStoragePoolPtr pool,
                            unsigned int flags);

int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolDeleteType virStoragePoolDeleteSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolDelete",
                       (void**)&virStoragePoolDeleteSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolDeleteSymbol(pool,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolDestroyType)(virStoragePoolPtr pool);

int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolDestroyType virStoragePoolDestroySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolDestroy",
                       (void**)&virStoragePoolDestroySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolDestroySymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolFreeType)(virStoragePoolPtr pool);

int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolFreeType virStoragePoolFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolFree",
                       (void**)&virStoragePoolFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolFreeSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetAutostartType)(virStoragePoolPtr pool,
                                  int * autostart);

int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int * autostart,
                                  virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolGetAutostartType virStoragePoolGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetAutostart",
                       (void**)&virStoragePoolGetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetAutostartSymbol(pool,
                                           autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virStoragePoolGetConnectType)(virStoragePoolPtr pool);

virConnectPtr
virStoragePoolGetConnectWrapper(virStoragePoolPtr pool,
                                virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virStoragePoolGetConnectType virStoragePoolGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetConnect",
                       (void**)&virStoragePoolGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetConnectSymbol(pool);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetInfoType)(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info);

int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolGetInfoType virStoragePoolGetInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetInfo",
                       (void**)&virStoragePoolGetInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetInfoSymbol(pool,
                                      info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virStoragePoolGetNameType)(virStoragePoolPtr pool);

const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    const char * ret = NULL;
    static virStoragePoolGetNameType virStoragePoolGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetName",
                       (void**)&virStoragePoolGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetNameSymbol(pool);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetUUIDType)(virStoragePoolPtr pool,
                             unsigned char * uuid);

int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolGetUUIDType virStoragePoolGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetUUID",
                       (void**)&virStoragePoolGetUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetUUIDSymbol(pool,
                                      uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetUUIDStringType)(virStoragePoolPtr pool,
                                   char * buf);

int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char * buf,
                                   virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolGetUUIDStringType virStoragePoolGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetUUIDString",
                       (void**)&virStoragePoolGetUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetUUIDStringSymbol(pool,
                                            buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virStoragePoolGetXMLDescType)(virStoragePoolPtr pool,
                                unsigned int flags);

char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = NULL;
    static virStoragePoolGetXMLDescType virStoragePoolGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolGetXMLDesc",
                       (void**)&virStoragePoolGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolGetXMLDescSymbol(pool,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolIsActiveType)(virStoragePoolPtr pool);

int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolIsActiveType virStoragePoolIsActiveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolIsActive",
                       (void**)&virStoragePoolIsActiveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolIsActiveSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolIsPersistentType)(virStoragePoolPtr pool);

int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolIsPersistentType virStoragePoolIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolIsPersistent",
                       (void**)&virStoragePoolIsPersistentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolIsPersistentSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolListAllVolumesType)(virStoragePoolPtr pool,
                                    virStorageVolPtr ** vols,
                                    unsigned int flags);

int
virStoragePoolListAllVolumesWrapper(virStoragePoolPtr pool,
                                    virStorageVolPtr ** vols,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolListAllVolumesType virStoragePoolListAllVolumesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolListAllVolumes",
                       (void**)&virStoragePoolListAllVolumesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolListAllVolumesSymbol(pool,
                                             vols,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolListVolumesType)(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames);

int
virStoragePoolListVolumesWrapper(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames,
                                 virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolListVolumesType virStoragePoolListVolumesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolListVolumes",
                       (void**)&virStoragePoolListVolumesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolListVolumesSymbol(pool,
                                          names,
                                          maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByNameType)(virConnectPtr conn,
                                  const char * name);

virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char * name,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolLookupByNameType virStoragePoolLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolLookupByName",
                       (void**)&virStoragePoolLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolLookupByNameSymbol(conn,
                                           name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByTargetPathType)(virConnectPtr conn,
                                        const char * path);

virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char * path,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolLookupByTargetPathType virStoragePoolLookupByTargetPathSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolLookupByTargetPath",
                       (void**)&virStoragePoolLookupByTargetPathSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolLookupByTargetPathSymbol(conn,
                                                 path);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByUUIDType)(virConnectPtr conn,
                                  const unsigned char * uuid);

virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolLookupByUUIDType virStoragePoolLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolLookupByUUID",
                       (void**)&virStoragePoolLookupByUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolLookupByUUIDSymbol(conn,
                                           uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByUUIDStringType)(virConnectPtr conn,
                                        const char * uuidstr);

virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolLookupByUUIDStringType virStoragePoolLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolLookupByUUIDString",
                       (void**)&virStoragePoolLookupByUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolLookupByUUIDStringSymbol(conn,
                                                 uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByVolumeType)(virStorageVolPtr vol);

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
    static virStoragePoolLookupByVolumeType virStoragePoolLookupByVolumeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolLookupByVolume",
                       (void**)&virStoragePoolLookupByVolumeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolLookupByVolumeSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolNumOfVolumesType)(virStoragePoolPtr pool);

int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolNumOfVolumesType virStoragePoolNumOfVolumesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolNumOfVolumes",
                       (void**)&virStoragePoolNumOfVolumesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolNumOfVolumesSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolRefType)(virStoragePoolPtr pool);

int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolRefType virStoragePoolRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolRef",
                       (void**)&virStoragePoolRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolRefSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolRefreshType)(virStoragePoolPtr pool,
                             unsigned int flags);

int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolRefreshType virStoragePoolRefreshSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolRefresh",
                       (void**)&virStoragePoolRefreshSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolRefreshSymbol(pool,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolSetAutostartType)(virStoragePoolPtr pool,
                                  int autostart);

int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolSetAutostartType virStoragePoolSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolSetAutostart",
                       (void**)&virStoragePoolSetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolSetAutostartSymbol(pool,
                                           autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolUndefineType)(virStoragePoolPtr pool);

int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = -1;
    static virStoragePoolUndefineType virStoragePoolUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStoragePoolUndefine",
                       (void**)&virStoragePoolUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStoragePoolUndefineSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolCreateXMLType)(virStoragePoolPtr pool,
                              const char * xmlDesc,
                              unsigned int flags);

virStorageVolPtr
virStorageVolCreateXMLWrapper(virStoragePoolPtr pool,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
    static virStorageVolCreateXMLType virStorageVolCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolCreateXML",
                       (void**)&virStorageVolCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolCreateXMLSymbol(pool,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolCreateXMLFromType)(virStoragePoolPtr pool,
                                  const char * xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags);

virStorageVolPtr
virStorageVolCreateXMLFromWrapper(virStoragePoolPtr pool,
                                  const char * xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
    static virStorageVolCreateXMLFromType virStorageVolCreateXMLFromSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolCreateXMLFrom",
                       (void**)&virStorageVolCreateXMLFromSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolCreateXMLFromSymbol(pool,
                                           xmlDesc,
                                           clonevol,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolDeleteType)(virStorageVolPtr vol,
                           unsigned int flags);

int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virStorageVolDeleteType virStorageVolDeleteSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolDelete",
                       (void**)&virStorageVolDeleteSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolDeleteSymbol(vol,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolDownloadType)(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags);

int
virStorageVolDownloadWrapper(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virStorageVolDownloadType virStorageVolDownloadSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolDownload",
                       (void**)&virStorageVolDownloadSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolDownloadSymbol(vol,
                                      stream,
                                      offset,
                                      length,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolFreeType)(virStorageVolPtr vol);

int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err)
{
    int ret = -1;
    static virStorageVolFreeType virStorageVolFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolFree",
                       (void**)&virStorageVolFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolFreeSymbol(vol);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virStorageVolGetConnectType)(virStorageVolPtr vol);

virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virStorageVolGetConnectType virStorageVolGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetConnect",
                       (void**)&virStorageVolGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetConnectSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolGetInfoType)(virStorageVolPtr vol,
                            virStorageVolInfoPtr info);

int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err)
{
    int ret = -1;
    static virStorageVolGetInfoType virStorageVolGetInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetInfo",
                       (void**)&virStorageVolGetInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetInfoSymbol(vol,
                                     info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolGetInfoFlagsType)(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags);

int
virStorageVolGetInfoFlagsWrapper(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virStorageVolGetInfoFlagsType virStorageVolGetInfoFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetInfoFlags",
                       (void**)&virStorageVolGetInfoFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetInfoFlagsSymbol(vol,
                                          info,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virStorageVolGetKeyType)(virStorageVolPtr vol);

const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err)
{
    const char * ret = NULL;
    static virStorageVolGetKeyType virStorageVolGetKeySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetKey",
                       (void**)&virStorageVolGetKeySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetKeySymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virStorageVolGetNameType)(virStorageVolPtr vol);

const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    const char * ret = NULL;
    static virStorageVolGetNameType virStorageVolGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetName",
                       (void**)&virStorageVolGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetNameSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virStorageVolGetPathType)(virStorageVolPtr vol);

char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virStorageVolGetPathType virStorageVolGetPathSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetPath",
                       (void**)&virStorageVolGetPathSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetPathSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virStorageVolGetXMLDescType)(virStorageVolPtr vol,
                               unsigned int flags);

char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
    static virStorageVolGetXMLDescType virStorageVolGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolGetXMLDesc",
                       (void**)&virStorageVolGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolGetXMLDescSymbol(vol,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolLookupByKeyType)(virConnectPtr conn,
                                const char * key);

virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char * key,
                                virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
    static virStorageVolLookupByKeyType virStorageVolLookupByKeySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolLookupByKey",
                       (void**)&virStorageVolLookupByKeySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolLookupByKeySymbol(conn,
                                         key);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolLookupByNameType)(virStoragePoolPtr pool,
                                 const char * name);

virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char * name,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
    static virStorageVolLookupByNameType virStorageVolLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolLookupByName",
                       (void**)&virStorageVolLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolLookupByNameSymbol(pool,
                                          name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolLookupByPathType)(virConnectPtr conn,
                                 const char * path);

virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char * path,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
    static virStorageVolLookupByPathType virStorageVolLookupByPathSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolLookupByPath",
                       (void**)&virStorageVolLookupByPathSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolLookupByPathSymbol(conn,
                                          path);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolRefType)(virStorageVolPtr vol);

int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err)
{
    int ret = -1;
    static virStorageVolRefType virStorageVolRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolRef",
                       (void**)&virStorageVolRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolRefSymbol(vol);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolResizeType)(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags);

int
virStorageVolResizeWrapper(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virStorageVolResizeType virStorageVolResizeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolResize",
                       (void**)&virStorageVolResizeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolResizeSymbol(vol,
                                    capacity,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolUploadType)(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags);

int
virStorageVolUploadWrapper(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virStorageVolUploadType virStorageVolUploadSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolUpload",
                       (void**)&virStorageVolUploadSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolUploadSymbol(vol,
                                    stream,
                                    offset,
                                    length,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolWipeType)(virStorageVolPtr vol,
                         unsigned int flags);

int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virStorageVolWipeType virStorageVolWipeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolWipe",
                       (void**)&virStorageVolWipeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolWipeSymbol(vol,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolWipePatternType)(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags);

int
virStorageVolWipePatternWrapper(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virStorageVolWipePatternType virStorageVolWipePatternSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStorageVolWipePattern",
                       (void**)&virStorageVolWipePatternSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStorageVolWipePatternSymbol(vol,
                                         algorithm,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
