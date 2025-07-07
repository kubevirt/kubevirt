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
(*virConnectListAllInterfacesFuncType)(virConnectPtr conn,
                                       virInterfacePtr ** ifaces,
                                       unsigned int flags);

int
virConnectListAllInterfacesWrapper(virConnectPtr conn,
                                   virInterfacePtr ** ifaces,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllInterfacesFuncType virConnectListAllInterfacesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllInterfaces",
                       (void**)&virConnectListAllInterfacesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllInterfacesSymbol(conn,
                                            ifaces,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDefinedInterfacesFuncType)(virConnectPtr conn,
                                           char ** const names,
                                           int maxnames);

int
virConnectListDefinedInterfacesWrapper(virConnectPtr conn,
                                       char ** const names,
                                       int maxnames,
                                       virErrorPtr err)
{
    int ret = -1;
    static virConnectListDefinedInterfacesFuncType virConnectListDefinedInterfacesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListDefinedInterfaces",
                       (void**)&virConnectListDefinedInterfacesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListDefinedInterfacesSymbol(conn,
                                                names,
                                                maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListInterfacesFuncType)(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames);

int
virConnectListInterfacesWrapper(virConnectPtr conn,
                                char ** const names,
                                int maxnames,
                                virErrorPtr err)
{
    int ret = -1;
    static virConnectListInterfacesFuncType virConnectListInterfacesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListInterfaces",
                       (void**)&virConnectListInterfacesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListInterfacesSymbol(conn,
                                         names,
                                         maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDefinedInterfacesFuncType)(virConnectPtr conn);

int
virConnectNumOfDefinedInterfacesWrapper(virConnectPtr conn,
                                        virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDefinedInterfacesFuncType virConnectNumOfDefinedInterfacesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfDefinedInterfaces",
                       (void**)&virConnectNumOfDefinedInterfacesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfDefinedInterfacesSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfInterfacesFuncType)(virConnectPtr conn);

int
virConnectNumOfInterfacesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfInterfacesFuncType virConnectNumOfInterfacesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfInterfaces",
                       (void**)&virConnectNumOfInterfacesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfInterfacesSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceChangeBeginFuncType)(virConnectPtr conn,
                                   unsigned int flags);

int
virInterfaceChangeBeginWrapper(virConnectPtr conn,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virInterfaceChangeBeginFuncType virInterfaceChangeBeginSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceChangeBegin",
                       (void**)&virInterfaceChangeBeginSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceChangeBeginSymbol(conn,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceChangeCommitFuncType)(virConnectPtr conn,
                                    unsigned int flags);

int
virInterfaceChangeCommitWrapper(virConnectPtr conn,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virInterfaceChangeCommitFuncType virInterfaceChangeCommitSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceChangeCommit",
                       (void**)&virInterfaceChangeCommitSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceChangeCommitSymbol(conn,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceChangeRollbackFuncType)(virConnectPtr conn,
                                      unsigned int flags);

int
virInterfaceChangeRollbackWrapper(virConnectPtr conn,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virInterfaceChangeRollbackFuncType virInterfaceChangeRollbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceChangeRollback",
                       (void**)&virInterfaceChangeRollbackSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceChangeRollbackSymbol(conn,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceCreateFuncType)(virInterfacePtr iface,
                              unsigned int flags);

int
virInterfaceCreateWrapper(virInterfacePtr iface,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virInterfaceCreateFuncType virInterfaceCreateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceCreate",
                       (void**)&virInterfaceCreateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceCreateSymbol(iface,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virInterfacePtr
(*virInterfaceDefineXMLFuncType)(virConnectPtr conn,
                                 const char * xml,
                                 unsigned int flags);

virInterfacePtr
virInterfaceDefineXMLWrapper(virConnectPtr conn,
                             const char * xml,
                             unsigned int flags,
                             virErrorPtr err)
{
    virInterfacePtr ret = NULL;
    static virInterfaceDefineXMLFuncType virInterfaceDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceDefineXML",
                       (void**)&virInterfaceDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceDefineXMLSymbol(conn,
                                      xml,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceDestroyFuncType)(virInterfacePtr iface,
                               unsigned int flags);

int
virInterfaceDestroyWrapper(virInterfacePtr iface,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virInterfaceDestroyFuncType virInterfaceDestroySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceDestroy",
                       (void**)&virInterfaceDestroySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceDestroySymbol(iface,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceFreeFuncType)(virInterfacePtr iface);

int
virInterfaceFreeWrapper(virInterfacePtr iface,
                        virErrorPtr err)
{
    int ret = -1;
    static virInterfaceFreeFuncType virInterfaceFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceFree",
                       (void**)&virInterfaceFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceFreeSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virInterfaceGetConnectFuncType)(virInterfacePtr iface);

virConnectPtr
virInterfaceGetConnectWrapper(virInterfacePtr iface,
                              virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virInterfaceGetConnectFuncType virInterfaceGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceGetConnect",
                       (void**)&virInterfaceGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceGetConnectSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virInterfaceGetMACStringFuncType)(virInterfacePtr iface);

const char *
virInterfaceGetMACStringWrapper(virInterfacePtr iface,
                                virErrorPtr err)
{
    const char * ret = NULL;
    static virInterfaceGetMACStringFuncType virInterfaceGetMACStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceGetMACString",
                       (void**)&virInterfaceGetMACStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceGetMACStringSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virInterfaceGetNameFuncType)(virInterfacePtr iface);

const char *
virInterfaceGetNameWrapper(virInterfacePtr iface,
                           virErrorPtr err)
{
    const char * ret = NULL;
    static virInterfaceGetNameFuncType virInterfaceGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceGetName",
                       (void**)&virInterfaceGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceGetNameSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virInterfaceGetXMLDescFuncType)(virInterfacePtr iface,
                                  unsigned int flags);

char *
virInterfaceGetXMLDescWrapper(virInterfacePtr iface,
                              unsigned int flags,
                              virErrorPtr err)
{
    char * ret = NULL;
    static virInterfaceGetXMLDescFuncType virInterfaceGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceGetXMLDesc",
                       (void**)&virInterfaceGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceGetXMLDescSymbol(iface,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceIsActiveFuncType)(virInterfacePtr iface);

int
virInterfaceIsActiveWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = -1;
    static virInterfaceIsActiveFuncType virInterfaceIsActiveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceIsActive",
                       (void**)&virInterfaceIsActiveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceIsActiveSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virInterfacePtr
(*virInterfaceLookupByMACStringFuncType)(virConnectPtr conn,
                                         const char * macstr);

virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char * macstr,
                                     virErrorPtr err)
{
    virInterfacePtr ret = NULL;
    static virInterfaceLookupByMACStringFuncType virInterfaceLookupByMACStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceLookupByMACString",
                       (void**)&virInterfaceLookupByMACStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceLookupByMACStringSymbol(conn,
                                              macstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virInterfacePtr
(*virInterfaceLookupByNameFuncType)(virConnectPtr conn,
                                    const char * name);

virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char * name,
                                virErrorPtr err)
{
    virInterfacePtr ret = NULL;
    static virInterfaceLookupByNameFuncType virInterfaceLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceLookupByName",
                       (void**)&virInterfaceLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceLookupByNameSymbol(conn,
                                         name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceRefFuncType)(virInterfacePtr iface);

int
virInterfaceRefWrapper(virInterfacePtr iface,
                       virErrorPtr err)
{
    int ret = -1;
    static virInterfaceRefFuncType virInterfaceRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceRef",
                       (void**)&virInterfaceRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceRefSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceUndefineFuncType)(virInterfacePtr iface);

int
virInterfaceUndefineWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = -1;
    static virInterfaceUndefineFuncType virInterfaceUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInterfaceUndefine",
                       (void**)&virInterfaceUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInterfaceUndefineSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
