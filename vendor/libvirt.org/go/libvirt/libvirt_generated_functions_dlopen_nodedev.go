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
(*virConnectListAllNodeDevicesFuncType)(virConnectPtr conn,
                                        virNodeDevicePtr ** devices,
                                        unsigned int flags);

int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllNodeDevicesFuncType virConnectListAllNodeDevicesSymbol;
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
(*virConnectNodeDeviceEventDeregisterAnyFuncType)(virConnectPtr conn,
                                                  int callbackID);

int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err)
{
    int ret = -1;
    static virConnectNodeDeviceEventDeregisterAnyFuncType virConnectNodeDeviceEventDeregisterAnySymbol;
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
(*virConnectNodeDeviceEventRegisterAnyFuncType)(virConnectPtr conn,
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
    static virConnectNodeDeviceEventRegisterAnyFuncType virConnectNodeDeviceEventRegisterAnySymbol;
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
(*virNodeDeviceCreateFuncType)(virNodeDevicePtr dev,
                               unsigned int flags);

int
virNodeDeviceCreateWrapper(virNodeDevicePtr dev,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceCreateFuncType virNodeDeviceCreateSymbol;
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
(*virNodeDeviceCreateXMLFuncType)(virConnectPtr conn,
                                  const char * xmlDesc,
                                  unsigned int flags);

virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceCreateXMLFuncType virNodeDeviceCreateXMLSymbol;
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
(*virNodeDeviceDefineXMLFuncType)(virConnectPtr conn,
                                  const char * xmlDesc,
                                  unsigned int flags);

virNodeDevicePtr
virNodeDeviceDefineXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceDefineXMLFuncType virNodeDeviceDefineXMLSymbol;
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
(*virNodeDeviceDestroyFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceDestroyFuncType virNodeDeviceDestroySymbol;
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
(*virNodeDeviceDetachFlagsFuncType)(virNodeDevicePtr dev,
                                    const char * driverName,
                                    unsigned int flags);

int
virNodeDeviceDetachFlagsWrapper(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceDetachFlagsFuncType virNodeDeviceDetachFlagsSymbol;
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
(*virNodeDeviceDettachFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceDettachFuncType virNodeDeviceDettachSymbol;
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
(*virNodeDeviceFreeFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceFreeFuncType virNodeDeviceFreeSymbol;
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
(*virNodeDeviceGetAutostartFuncType)(virNodeDevicePtr dev,
                                     int * autostart);

int
virNodeDeviceGetAutostartWrapper(virNodeDevicePtr dev,
                                 int * autostart,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceGetAutostartFuncType virNodeDeviceGetAutostartSymbol;
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
(*virNodeDeviceGetNameFuncType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    const char * ret = NULL;
    static virNodeDeviceGetNameFuncType virNodeDeviceGetNameSymbol;
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
(*virNodeDeviceGetParentFuncType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    const char * ret = NULL;
    static virNodeDeviceGetParentFuncType virNodeDeviceGetParentSymbol;
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
(*virNodeDeviceGetXMLDescFuncType)(virNodeDevicePtr dev,
                                   unsigned int flags);

char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
    static virNodeDeviceGetXMLDescFuncType virNodeDeviceGetXMLDescSymbol;
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
(*virNodeDeviceIsActiveFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceIsActiveWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceIsActiveFuncType virNodeDeviceIsActiveSymbol;
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
(*virNodeDeviceIsPersistentFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceIsPersistentWrapper(virNodeDevicePtr dev,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceIsPersistentFuncType virNodeDeviceIsPersistentSymbol;
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
(*virNodeDeviceListCapsFuncType)(virNodeDevicePtr dev,
                                 char ** const names,
                                 int maxnames);

int
virNodeDeviceListCapsWrapper(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceListCapsFuncType virNodeDeviceListCapsSymbol;
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
(*virNodeDeviceLookupByNameFuncType)(virConnectPtr conn,
                                     const char * name);

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char * name,
                                 virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
    static virNodeDeviceLookupByNameFuncType virNodeDeviceLookupByNameSymbol;
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
(*virNodeDeviceLookupSCSIHostByWWNFuncType)(virConnectPtr conn,
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
    static virNodeDeviceLookupSCSIHostByWWNFuncType virNodeDeviceLookupSCSIHostByWWNSymbol;
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
(*virNodeDeviceNumOfCapsFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceNumOfCapsFuncType virNodeDeviceNumOfCapsSymbol;
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
(*virNodeDeviceReAttachFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceReAttachFuncType virNodeDeviceReAttachSymbol;
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
(*virNodeDeviceRefFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceRefFuncType virNodeDeviceRefSymbol;
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
(*virNodeDeviceResetFuncType)(virNodeDevicePtr dev);

int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceResetFuncType virNodeDeviceResetSymbol;
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
(*virNodeDeviceSetAutostartFuncType)(virNodeDevicePtr dev,
                                     int autostart);

int
virNodeDeviceSetAutostartWrapper(virNodeDevicePtr dev,
                                 int autostart,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceSetAutostartFuncType virNodeDeviceSetAutostartSymbol;
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
(*virNodeDeviceUndefineFuncType)(virNodeDevicePtr dev,
                                 unsigned int flags);

int
virNodeDeviceUndefineWrapper(virNodeDevicePtr dev,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceUndefineFuncType virNodeDeviceUndefineSymbol;
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
(*virNodeDeviceUpdateFuncType)(virNodeDevicePtr dev,
                               const char * xmlDesc,
                               unsigned int flags);

int
virNodeDeviceUpdateWrapper(virNodeDevicePtr dev,
                           const char * xmlDesc,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virNodeDeviceUpdateFuncType virNodeDeviceUpdateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeDeviceUpdate",
                       (void**)&virNodeDeviceUpdateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeDeviceUpdateSymbol(dev,
                                    xmlDesc,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeListDevicesFuncType)(virConnectPtr conn,
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
    static virNodeListDevicesFuncType virNodeListDevicesSymbol;
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
(*virNodeNumOfDevicesFuncType)(virConnectPtr conn,
                               const char * cap,
                               unsigned int flags);

int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virNodeNumOfDevicesFuncType virNodeNumOfDevicesSymbol;
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
