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


typedef virDomainCheckpointPtr
(*virDomainCheckpointCreateXMLType)(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointCreateXMLWrapper(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
    static virDomainCheckpointCreateXMLType virDomainCheckpointCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointCreateXML",
                       (void**)&virDomainCheckpointCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointCreateXMLSymbol(domain,
                                             xmlDesc,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointDeleteType)(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags);

int
virDomainCheckpointDeleteWrapper(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointDeleteType virDomainCheckpointDeleteSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointDelete",
                       (void**)&virDomainCheckpointDeleteSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointDeleteSymbol(checkpoint,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointFreeType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointFreeWrapper(virDomainCheckpointPtr checkpoint,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointFreeType virDomainCheckpointFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointFree",
                       (void**)&virDomainCheckpointFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointFreeSymbol(checkpoint);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virDomainCheckpointGetConnectType)(virDomainCheckpointPtr checkpoint);

virConnectPtr
virDomainCheckpointGetConnectWrapper(virDomainCheckpointPtr checkpoint,
                                     virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virDomainCheckpointGetConnectType virDomainCheckpointGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointGetConnect",
                       (void**)&virDomainCheckpointGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointGetConnectSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCheckpointGetDomainType)(virDomainCheckpointPtr checkpoint);

virDomainPtr
virDomainCheckpointGetDomainWrapper(virDomainCheckpointPtr checkpoint,
                                    virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCheckpointGetDomainType virDomainCheckpointGetDomainSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointGetDomain",
                       (void**)&virDomainCheckpointGetDomainSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointGetDomainSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virDomainCheckpointGetNameType)(virDomainCheckpointPtr checkpoint);

const char *
virDomainCheckpointGetNameWrapper(virDomainCheckpointPtr checkpoint,
                                  virErrorPtr err)
{
    const char * ret = NULL;
    static virDomainCheckpointGetNameType virDomainCheckpointGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointGetName",
                       (void**)&virDomainCheckpointGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointGetNameSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainCheckpointPtr
(*virDomainCheckpointGetParentType)(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointGetParentWrapper(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
    static virDomainCheckpointGetParentType virDomainCheckpointGetParentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointGetParent",
                       (void**)&virDomainCheckpointGetParentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointGetParentSymbol(checkpoint,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainCheckpointGetXMLDescType)(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags);

char *
virDomainCheckpointGetXMLDescWrapper(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = NULL;
    static virDomainCheckpointGetXMLDescType virDomainCheckpointGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointGetXMLDesc",
                       (void**)&virDomainCheckpointGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointGetXMLDescSymbol(checkpoint,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointListAllChildrenType)(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags);

int
virDomainCheckpointListAllChildrenWrapper(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointListAllChildrenType virDomainCheckpointListAllChildrenSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointListAllChildren",
                       (void**)&virDomainCheckpointListAllChildrenSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointListAllChildrenSymbol(checkpoint,
                                                   children,
                                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainCheckpointPtr
(*virDomainCheckpointLookupByNameType)(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointLookupByNameWrapper(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
    static virDomainCheckpointLookupByNameType virDomainCheckpointLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointLookupByName",
                       (void**)&virDomainCheckpointLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointLookupByNameSymbol(domain,
                                                name,
                                                flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointRefType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointRefWrapper(virDomainCheckpointPtr checkpoint,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointRefType virDomainCheckpointRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCheckpointRef",
                       (void**)&virDomainCheckpointRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCheckpointRefSymbol(checkpoint);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainListAllCheckpointsType)(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags);

int
virDomainListAllCheckpointsWrapper(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainListAllCheckpointsType virDomainListAllCheckpointsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainListAllCheckpoints",
                       (void**)&virDomainListAllCheckpointsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainListAllCheckpointsSymbol(domain,
                                            checkpoints,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
