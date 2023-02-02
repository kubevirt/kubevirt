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


int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllNodeDevices not available prior to libvirt version 0.10.2");
#else
    ret = virConnectListAllNodeDevices(conn,
                                       devices,
                                       flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 2, 0)
    setVirError(err, "Function virConnectNodeDeviceEventDeregisterAny not available prior to libvirt version 2.2.0");
#else
    ret = virConnectNodeDeviceEventDeregisterAny(conn,
                                                 callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(2, 2, 0)
    setVirError(err, "Function virConnectNodeDeviceEventRegisterAny not available prior to libvirt version 2.2.0");
#else
    ret = virConnectNodeDeviceEventRegisterAny(conn,
                                               dev,
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
virNodeDeviceCreateWrapper(virNodeDevicePtr dev,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 3, 0)
    setVirError(err, "Function virNodeDeviceCreate not available prior to libvirt version 7.3.0");
#else
    ret = virNodeDeviceCreate(dev,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 3)
    setVirError(err, "Function virNodeDeviceCreateXML not available prior to libvirt version 0.6.3");
#else
    ret = virNodeDeviceCreateXML(conn,
                                 xmlDesc,
                                 flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNodeDevicePtr
virNodeDeviceDefineXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(7, 3, 0)
    setVirError(err, "Function virNodeDeviceDefineXML not available prior to libvirt version 7.3.0");
#else
    ret = virNodeDeviceDefineXML(conn,
                                 xmlDesc,
                                 flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 3)
    setVirError(err, "Function virNodeDeviceDestroy not available prior to libvirt version 0.6.3");
#else
    ret = virNodeDeviceDestroy(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceDetachFlagsWrapper(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 5)
    setVirError(err, "Function virNodeDeviceDetachFlags not available prior to libvirt version 1.0.5");
#else
    ret = virNodeDeviceDetachFlags(dev,
                                   driverName,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeDeviceDettach not available prior to libvirt version 0.6.1");
#else
    ret = virNodeDeviceDettach(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceFree not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceFree(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceGetAutostartWrapper(virNodeDevicePtr dev,
                                 int * autostart,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceGetAutostart not available prior to libvirt version 7.8.0");
#else
    ret = virNodeDeviceGetAutostart(dev,
                                    autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceGetName not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceGetName(dev);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceGetParent not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceGetParent(dev);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceGetXMLDesc not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceGetXMLDesc(dev,
                                  flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceIsActiveWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceIsActive not available prior to libvirt version 7.8.0");
#else
    ret = virNodeDeviceIsActive(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceIsPersistentWrapper(virNodeDevicePtr dev,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceIsPersistent not available prior to libvirt version 7.8.0");
#else
    ret = virNodeDeviceIsPersistent(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceListCapsWrapper(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceListCaps not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceListCaps(dev,
                                names,
                                maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char * name,
                                 virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceLookupByName not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceLookupByName(conn,
                                    name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNodeDevicePtr
virNodeDeviceLookupSCSIHostByWWNWrapper(virConnectPtr conn,
                                        const char * wwnn,
                                        const char * wwpn,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virNodeDeviceLookupSCSIHostByWWN not available prior to libvirt version 1.0.3");
#else
    ret = virNodeDeviceLookupSCSIHostByWWN(conn,
                                           wwnn,
                                           wwpn,
                                           flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceNumOfCaps not available prior to libvirt version 0.5.0");
#else
    ret = virNodeDeviceNumOfCaps(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeDeviceReAttach not available prior to libvirt version 0.6.1");
#else
    ret = virNodeDeviceReAttach(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virNodeDeviceRef not available prior to libvirt version 0.6.0");
#else
    ret = virNodeDeviceRef(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeDeviceReset not available prior to libvirt version 0.6.1");
#else
    ret = virNodeDeviceReset(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceSetAutostartWrapper(virNodeDevicePtr dev,
                                 int autostart,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceSetAutostart not available prior to libvirt version 7.8.0");
#else
    ret = virNodeDeviceSetAutostart(dev,
                                    autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeDeviceUndefineWrapper(virNodeDevicePtr dev,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 3, 0)
    setVirError(err, "Function virNodeDeviceUndefine not available prior to libvirt version 7.3.0");
#else
    ret = virNodeDeviceUndefine(dev,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeListDevicesWrapper(virConnectPtr conn,
                          const char * cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeListDevices not available prior to libvirt version 0.5.0");
#else
    ret = virNodeListDevices(conn,
                             cap,
                             names,
                             maxnames,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeNumOfDevices not available prior to libvirt version 0.5.0");
#else
    ret = virNodeNumOfDevices(conn,
                              cap,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
