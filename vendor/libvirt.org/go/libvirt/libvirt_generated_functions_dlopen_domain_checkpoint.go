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
(*virDomainCheckpointCreateXMLFuncType)(virDomainPtr domain,
                                        const char * xmlDesc,
                                        unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointCreateXMLWrapper(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
    static virDomainCheckpointCreateXMLFuncType virDomainCheckpointCreateXMLSymbol;
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
(*virDomainCheckpointDeleteFuncType)(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags);

int
virDomainCheckpointDeleteWrapper(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointDeleteFuncType virDomainCheckpointDeleteSymbol;
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
(*virDomainCheckpointFreeFuncType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointFreeWrapper(virDomainCheckpointPtr checkpoint,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointFreeFuncType virDomainCheckpointFreeSymbol;
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
(*virDomainCheckpointGetConnectFuncType)(virDomainCheckpointPtr checkpoint);

virConnectPtr
virDomainCheckpointGetConnectWrapper(virDomainCheckpointPtr checkpoint,
                                     virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virDomainCheckpointGetConnectFuncType virDomainCheckpointGetConnectSymbol;
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
(*virDomainCheckpointGetDomainFuncType)(virDomainCheckpointPtr checkpoint);

virDomainPtr
virDomainCheckpointGetDomainWrapper(virDomainCheckpointPtr checkpoint,
                                    virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCheckpointGetDomainFuncType virDomainCheckpointGetDomainSymbol;
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
(*virDomainCheckpointGetNameFuncType)(virDomainCheckpointPtr checkpoint);

const char *
virDomainCheckpointGetNameWrapper(virDomainCheckpointPtr checkpoint,
                                  virErrorPtr err)
{
    const char * ret = NULL;
    static virDomainCheckpointGetNameFuncType virDomainCheckpointGetNameSymbol;
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
(*virDomainCheckpointGetParentFuncType)(virDomainCheckpointPtr checkpoint,
                                        unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointGetParentWrapper(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
    static virDomainCheckpointGetParentFuncType virDomainCheckpointGetParentSymbol;
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
(*virDomainCheckpointGetXMLDescFuncType)(virDomainCheckpointPtr checkpoint,
                                         unsigned int flags);

char *
virDomainCheckpointGetXMLDescWrapper(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = NULL;
    static virDomainCheckpointGetXMLDescFuncType virDomainCheckpointGetXMLDescSymbol;
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
(*virDomainCheckpointListAllChildrenFuncType)(virDomainCheckpointPtr checkpoint,
                                              virDomainCheckpointPtr ** children,
                                              unsigned int flags);

int
virDomainCheckpointListAllChildrenWrapper(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointListAllChildrenFuncType virDomainCheckpointListAllChildrenSymbol;
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
(*virDomainCheckpointLookupByNameFuncType)(virDomainPtr domain,
                                           const char * name,
                                           unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointLookupByNameWrapper(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
    static virDomainCheckpointLookupByNameFuncType virDomainCheckpointLookupByNameSymbol;
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
(*virDomainCheckpointRefFuncType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointRefWrapper(virDomainCheckpointPtr checkpoint,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainCheckpointRefFuncType virDomainCheckpointRefSymbol;
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
(*virDomainListAllCheckpointsFuncType)(virDomainPtr domain,
                                       virDomainCheckpointPtr ** checkpoints,
                                       unsigned int flags);

int
virDomainListAllCheckpointsWrapper(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainListAllCheckpointsFuncType virDomainListAllCheckpointsSymbol;
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
