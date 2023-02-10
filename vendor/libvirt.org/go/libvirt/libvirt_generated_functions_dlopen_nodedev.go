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


typedef int
(*virConnectListAllNodeDevicesType)(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags);

int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllNodeDevicesType virConnectListAllNodeDevicesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllNodeDevices",
                       (void**)&virConnectListAllNodeDevicesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllNodeDevicesSymbol(conn,
                                             devices,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNodeDeviceEventDeregisterAnyType)(virConnectPtr conn,
                                              int callbackID);

int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err)
{
    int ret = -1;
    static virConnectNodeDeviceEventDeregisterAnyType virConnectNodeDeviceEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNodeDeviceEventDeregisterAny",
                       (void**)&virConnectNodeDeviceEventDeregisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNodeDeviceEventDeregisterAnySymbol(conn,
                                                       callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNodeDeviceEventRegisterAnyType)(virConnectPtr conn,
                                            virNodeDevicePtr dev,
                                            int eventID,
                                            virConnectNodeDeviceEventGenericCallback cb,
                                            void * opaque,
                                            virFreeCallback freecb);

int
virConnectNodeDeviceEventRegisterAnyWrapper(virConnectPtr conn,
                                            virNodeDevicePtr dev,
                                            int eventID,
                                            virConnectNodeDeviceEventGenericCallback cb,
                                            void * opaque,
                                            virFreeCallback freecb,
                                            virErrorPtr err)
{
    int ret = -1;
    static virConnectNodeDeviceEventRegisterAnyType virConnectNodeDeviceEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNodeDeviceEventRegisterAny",
                       (void**)&virConnectNodeDeviceEventRegisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNodeDeviceEventRegisterAnySymbol(conn,
                                                     dev,
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
(*virNodeDeviceCreateType)(virNodeDevicePtr dev,
                           unsigned int flags);

int
virNodeDeviceCreateWrapper(virNodeDevicePtr dev,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceCreateType virNodeDeviceCreateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceCreate",
                       (void**)&virNodeDeviceCreateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceCreateSymbol(dev,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceCreateXMLType)(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags);

virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceCreateXMLType virNodeDeviceCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceCreateXML",
                       (void**)&virNodeDeviceCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceCreateXMLSymbol(conn,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceDefineXMLType)(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags);

virNodeDevicePtr
virNodeDeviceDefineXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceDefineXMLType virNodeDeviceDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceDefineXML",
                       (void**)&virNodeDeviceDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceDefineXMLSymbol(conn,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceDestroyType)(virNodeDevicePtr dev);

int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceDestroyType virNodeDeviceDestroySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceDestroy",
                       (void**)&virNodeDeviceDestroySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceDestroySymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceDetachFlagsType)(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags);

int
virNodeDeviceDetachFlagsWrapper(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceDetachFlagsType virNodeDeviceDetachFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceDetachFlags",
                       (void**)&virNodeDeviceDetachFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceDetachFlagsSymbol(dev,
                                         driverName,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceDettachType)(virNodeDevicePtr dev);

int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceDettachType virNodeDeviceDettachSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceDettach",
                       (void**)&virNodeDeviceDettachSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceDettachSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceFreeType)(virNodeDevicePtr dev);

int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceFreeType virNodeDeviceFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceFree",
                       (void**)&virNodeDeviceFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceFreeSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceGetAutostartType)(virNodeDevicePtr dev,
                                 int * autostart);

int
virNodeDeviceGetAutostartWrapper(virNodeDevicePtr dev,
                                 int * autostart,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceGetAutostartType virNodeDeviceGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceGetAutostart",
                       (void**)&virNodeDeviceGetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceGetAutostartSymbol(dev,
                                          autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNodeDeviceGetNameType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    const char * ret = NULL;
    static virNodeDeviceGetNameType virNodeDeviceGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceGetName",
                       (void**)&virNodeDeviceGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceGetNameSymbol(dev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNodeDeviceGetParentType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    const char * ret = NULL;
    static virNodeDeviceGetParentType virNodeDeviceGetParentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceGetParent",
                       (void**)&virNodeDeviceGetParentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceGetParentSymbol(dev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNodeDeviceGetXMLDescType)(virNodeDevicePtr dev,
                               unsigned int flags);

char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
    static virNodeDeviceGetXMLDescType virNodeDeviceGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceGetXMLDesc",
                       (void**)&virNodeDeviceGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceGetXMLDescSymbol(dev,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceIsActiveType)(virNodeDevicePtr dev);

int
virNodeDeviceIsActiveWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceIsActiveType virNodeDeviceIsActiveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceIsActive",
                       (void**)&virNodeDeviceIsActiveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceIsActiveSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceIsPersistentType)(virNodeDevicePtr dev);

int
virNodeDeviceIsPersistentWrapper(virNodeDevicePtr dev,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceIsPersistentType virNodeDeviceIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceIsPersistent",
                       (void**)&virNodeDeviceIsPersistentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceIsPersistentSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceListCapsType)(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames);

int
virNodeDeviceListCapsWrapper(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceListCapsType virNodeDeviceListCapsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceListCaps",
                       (void**)&virNodeDeviceListCapsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceListCapsSymbol(dev,
                                      names,
                                      maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceLookupByNameType)(virConnectPtr conn,
                                 const char * name);

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char * name,
                                 virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceLookupByNameType virNodeDeviceLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceLookupByName",
                       (void**)&virNodeDeviceLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceLookupByNameSymbol(conn,
                                          name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceLookupSCSIHostByWWNType)(virConnectPtr conn,
                                        const char * wwnn,
                                        const char * wwpn,
                                        unsigned int flags);

virNodeDevicePtr
virNodeDeviceLookupSCSIHostByWWNWrapper(virConnectPtr conn,
                                        const char * wwnn,
                                        const char * wwpn,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceLookupSCSIHostByWWNType virNodeDeviceLookupSCSIHostByWWNSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceLookupSCSIHostByWWN",
                       (void**)&virNodeDeviceLookupSCSIHostByWWNSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceLookupSCSIHostByWWNSymbol(conn,
                                                 wwnn,
                                                 wwpn,
                                                 flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceNumOfCapsType)(virNodeDevicePtr dev);

int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceNumOfCapsType virNodeDeviceNumOfCapsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceNumOfCaps",
                       (void**)&virNodeDeviceNumOfCapsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceNumOfCapsSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceReAttachType)(virNodeDevicePtr dev);

int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceReAttachType virNodeDeviceReAttachSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceReAttach",
                       (void**)&virNodeDeviceReAttachSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceReAttachSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceRefType)(virNodeDevicePtr dev);

int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceRefType virNodeDeviceRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceRef",
                       (void**)&virNodeDeviceRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceRefSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceResetType)(virNodeDevicePtr dev);

int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceResetType virNodeDeviceResetSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceReset",
                       (void**)&virNodeDeviceResetSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceResetSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceSetAutostartType)(virNodeDevicePtr dev,
                                 int autostart);

int
virNodeDeviceSetAutostartWrapper(virNodeDevicePtr dev,
                                 int autostart,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceSetAutostartType virNodeDeviceSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceSetAutostart",
                       (void**)&virNodeDeviceSetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceSetAutostartSymbol(dev,
                                          autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceUndefineType)(virNodeDevicePtr dev,
                             unsigned int flags);

int
virNodeDeviceUndefineWrapper(virNodeDevicePtr dev,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceUndefineType virNodeDeviceUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceUndefine",
                       (void**)&virNodeDeviceUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceUndefineSymbol(dev,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeListDevicesType)(virConnectPtr conn,
                          const char * cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags);

int
virNodeListDevicesWrapper(virConnectPtr conn,
                          const char * cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virNodeListDevicesType virNodeListDevicesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeListDevices",
                       (void**)&virNodeListDevicesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeListDevicesSymbol(conn,
                                   cap,
                                   names,
                                   maxnames,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeNumOfDevicesType)(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags);

int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virNodeNumOfDevicesType virNodeNumOfDevicesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeNumOfDevices",
                       (void**)&virNodeNumOfDevicesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeNumOfDevicesSymbol(conn,
                                    cap,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
