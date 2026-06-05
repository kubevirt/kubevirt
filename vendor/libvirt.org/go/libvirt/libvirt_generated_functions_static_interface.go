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
virConnectListAllInterfacesWrapper(virConnectPtr conn,
                                   virInterfacePtr ** ifaces,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllInterfaces not available prior to libvirt version 0.10.2");
#else
    ret = virConnectListAllInterfaces(conn,
                                      ifaces,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListDefinedInterfacesWrapper(virConnectPtr conn,
                                       char ** const names,
                                       int maxnames,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 0)
    setVirError(err, "Function virConnectListDefinedInterfaces not available prior to libvirt version 0.7.0");
#else
    ret = virConnectListDefinedInterfaces(conn,
                                          names,
                                          maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListInterfacesWrapper(virConnectPtr conn,
                                char ** const names,
                                int maxnames,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectListInterfaces not available prior to libvirt version 0.6.4");
#else
    ret = virConnectListInterfaces(conn,
                                   names,
                                   maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfDefinedInterfacesWrapper(virConnectPtr conn,
                                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 0)
    setVirError(err, "Function virConnectNumOfDefinedInterfaces not available prior to libvirt version 0.7.0");
#else
    ret = virConnectNumOfDefinedInterfaces(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfInterfacesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectNumOfInterfaces not available prior to libvirt version 0.6.4");
#else
    ret = virConnectNumOfInterfaces(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceChangeBeginWrapper(virConnectPtr conn,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virInterfaceChangeBegin not available prior to libvirt version 0.9.2");
#else
    ret = virInterfaceChangeBegin(conn,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceChangeCommitWrapper(virConnectPtr conn,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virInterfaceChangeCommit not available prior to libvirt version 0.9.2");
#else
    ret = virInterfaceChangeCommit(conn,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceChangeRollbackWrapper(virConnectPtr conn,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virInterfaceChangeRollback not available prior to libvirt version 0.9.2");
#else
    ret = virInterfaceChangeRollback(conn,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceCreateWrapper(virInterfacePtr iface,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceCreate not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceCreate(iface,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virInterfacePtr
virInterfaceDefineXMLWrapper(virConnectPtr conn,
                             const char * xml,
                             unsigned int flags,
                             virErrorPtr err)
{
    virInterfacePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceDefineXML not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceDefineXML(conn,
                                xml,
                                flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceDestroyWrapper(virInterfacePtr iface,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceDestroy not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceDestroy(iface,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceFreeWrapper(virInterfacePtr iface,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceFree not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceFree(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virInterfaceGetConnectWrapper(virInterfacePtr iface,
                              virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetConnect not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceGetConnect(iface);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virInterfaceGetMACStringWrapper(virInterfacePtr iface,
                                virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetMACString not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceGetMACString(iface);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virInterfaceGetNameWrapper(virInterfacePtr iface,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetName not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceGetName(iface);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virInterfaceGetXMLDescWrapper(virInterfacePtr iface,
                              unsigned int flags,
                              virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetXMLDesc not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceGetXMLDesc(iface,
                                 flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceIsActiveWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virInterfaceIsActive not available prior to libvirt version 0.7.3");
#else
    ret = virInterfaceIsActive(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char * macstr,
                                     virErrorPtr err)
{
    virInterfacePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceLookupByMACString not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceLookupByMACString(conn,
                                        macstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char * name,
                                virErrorPtr err)
{
    virInterfacePtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceLookupByName not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceLookupByName(conn,
                                   name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceRefWrapper(virInterfacePtr iface,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceRef not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceRef(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInterfaceUndefineWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceUndefine not available prior to libvirt version 0.6.4");
#else
    ret = virInterfaceUndefine(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
