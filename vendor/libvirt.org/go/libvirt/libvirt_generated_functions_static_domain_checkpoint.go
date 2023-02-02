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


virDomainCheckpointPtr
virDomainCheckpointCreateXMLWrapper(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointCreateXML not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointCreateXML(domain,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCheckpointDeleteWrapper(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointDelete not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointDelete(checkpoint,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCheckpointFreeWrapper(virDomainCheckpointPtr checkpoint,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointFree not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointFree(checkpoint);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virDomainCheckpointGetConnectWrapper(virDomainCheckpointPtr checkpoint,
                                     virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetConnect not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointGetConnect(checkpoint);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainCheckpointGetDomainWrapper(virDomainCheckpointPtr checkpoint,
                                    virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetDomain not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointGetDomain(checkpoint);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virDomainCheckpointGetNameWrapper(virDomainCheckpointPtr checkpoint,
                                  virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetName not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointGetName(checkpoint);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainCheckpointPtr
virDomainCheckpointGetParentWrapper(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetParent not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointGetParent(checkpoint,
                                       flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainCheckpointGetXMLDescWrapper(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetXMLDesc not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointGetXMLDesc(checkpoint,
                                        flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCheckpointListAllChildrenWrapper(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointListAllChildren not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointListAllChildren(checkpoint,
                                             children,
                                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainCheckpointPtr
virDomainCheckpointLookupByNameWrapper(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointLookupByName not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointLookupByName(domain,
                                          name,
                                          flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCheckpointRefWrapper(virDomainCheckpointPtr checkpoint,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointRef not available prior to libvirt version 5.6.0");
#else
    ret = virDomainCheckpointRef(checkpoint);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainListAllCheckpointsWrapper(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainListAllCheckpoints not available prior to libvirt version 5.6.0");
#else
    ret = virDomainListAllCheckpoints(domain,
                                      checkpoints,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
