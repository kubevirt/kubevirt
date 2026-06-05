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
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainHasCurrentSnapshot not available prior to libvirt version 0.8.0");
#else
    ret = virDomainHasCurrentSnapshot(domain,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainListAllSnapshots not available prior to libvirt version 0.9.13");
#else
    ret = virDomainListAllSnapshots(domain,
                                    snaps,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainRevertToSnapshot not available prior to libvirt version 0.8.0");
#else
    ret = virDomainRevertToSnapshot(snapshot,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainSnapshotPtr
virDomainSnapshotCreateXMLWrapper(virDomainPtr domain,
                                  const char * xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotCreateXML not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotCreateXML(domain,
                                     xmlDesc,
                                     flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotCurrent not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotCurrent(domain,
                                   flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotDelete not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotDelete(snapshot,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotFree not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotFree(snapshot);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainSnapshotGetConnect not available prior to libvirt version 0.9.5");
#else
    ret = virDomainSnapshotGetConnect(snapshot);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainSnapshotGetDomain not available prior to libvirt version 0.9.5");
#else
    ret = virDomainSnapshotGetDomain(snapshot);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainSnapshotGetName not available prior to libvirt version 0.9.5");
#else
    ret = virDomainSnapshotGetName(snapshot);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainSnapshotGetParent not available prior to libvirt version 0.9.7");
#else
    ret = virDomainSnapshotGetParent(snapshot,
                                     flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotGetXMLDesc not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotGetXMLDesc(snapshot,
                                      flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotHasMetadata not available prior to libvirt version 0.9.13");
#else
    ret = virDomainSnapshotHasMetadata(snapshot,
                                       flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotIsCurrent not available prior to libvirt version 0.9.13");
#else
    ret = virDomainSnapshotIsCurrent(snapshot,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotListAllChildrenWrapper(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotListAllChildren not available prior to libvirt version 0.9.13");
#else
    ret = virDomainSnapshotListAllChildren(snapshot,
                                           snaps,
                                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotListChildrenNamesWrapper(virDomainSnapshotPtr snapshot,
                                          char ** names,
                                          int nameslen,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainSnapshotListChildrenNames not available prior to libvirt version 0.9.7");
#else
    ret = virDomainSnapshotListChildrenNames(snapshot,
                                             names,
                                             nameslen,
                                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotListNamesWrapper(virDomainPtr domain,
                                  char ** names,
                                  int nameslen,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotListNames not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotListNames(domain,
                                     names,
                                     nameslen,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotLookupByName not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotLookupByName(domain,
                                        name,
                                        flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotNum not available prior to libvirt version 0.8.0");
#else
    ret = virDomainSnapshotNum(domain,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainSnapshotNumChildren not available prior to libvirt version 0.9.7");
#else
    ret = virDomainSnapshotNumChildren(snapshot,
                                       flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotRef not available prior to libvirt version 0.9.13");
#else
    ret = virDomainSnapshotRef(snapshot);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
