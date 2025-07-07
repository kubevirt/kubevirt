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
(*virDomainHasCurrentSnapshotFuncType)(virDomainPtr domain,
                                       unsigned int flags);

int
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainHasCurrentSnapshotFuncType virDomainHasCurrentSnapshotSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainHasCurrentSnapshot",
                       (void**)&virDomainHasCurrentSnapshotSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainHasCurrentSnapshotSymbol(domain,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainListAllSnapshotsFuncType)(virDomainPtr domain,
                                     virDomainSnapshotPtr ** snaps,
                                     unsigned int flags);

int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainListAllSnapshotsFuncType virDomainListAllSnapshotsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainListAllSnapshots",
                       (void**)&virDomainListAllSnapshotsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainListAllSnapshotsSymbol(domain,
                                          snaps,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRevertToSnapshotFuncType)(virDomainSnapshotPtr snapshot,
                                     unsigned int flags);

int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainRevertToSnapshotFuncType virDomainRevertToSnapshotSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainRevertToSnapshot",
                       (void**)&virDomainRevertToSnapshotSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRevertToSnapshotSymbol(snapshot,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotCreateXMLFuncType)(virDomainPtr domain,
                                      const char * xmlDesc,
                                      unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCreateXMLWrapper(virDomainPtr domain,
                                  const char * xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotCreateXMLFuncType virDomainSnapshotCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotCreateXML",
                       (void**)&virDomainSnapshotCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotCreateXMLSymbol(domain,
                                           xmlDesc,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotCurrentFuncType)(virDomainPtr domain,
                                    unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotCurrentFuncType virDomainSnapshotCurrentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotCurrent",
                       (void**)&virDomainSnapshotCurrentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotCurrentSymbol(domain,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotDeleteFuncType)(virDomainSnapshotPtr snapshot,
                                   unsigned int flags);

int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotDeleteFuncType virDomainSnapshotDeleteSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotDelete",
                       (void**)&virDomainSnapshotDeleteSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotDeleteSymbol(snapshot,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotFreeFuncType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotFreeFuncType virDomainSnapshotFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotFree",
                       (void**)&virDomainSnapshotFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotFreeSymbol(snapshot);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virDomainSnapshotGetConnectFuncType)(virDomainSnapshotPtr snapshot);

virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virDomainSnapshotGetConnectFuncType virDomainSnapshotGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotGetConnect",
                       (void**)&virDomainSnapshotGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotGetConnectSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainSnapshotGetDomainFuncType)(virDomainSnapshotPtr snapshot);

virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainSnapshotGetDomainFuncType virDomainSnapshotGetDomainSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotGetDomain",
                       (void**)&virDomainSnapshotGetDomainSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotGetDomainSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virDomainSnapshotGetNameFuncType)(virDomainSnapshotPtr snapshot);

const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err)
{
    const char * ret = NULL;
    static virDomainSnapshotGetNameFuncType virDomainSnapshotGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotGetName",
                       (void**)&virDomainSnapshotGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotGetNameSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotGetParentFuncType)(virDomainSnapshotPtr snapshot,
                                      unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotGetParentFuncType virDomainSnapshotGetParentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotGetParent",
                       (void**)&virDomainSnapshotGetParentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotGetParentSymbol(snapshot,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainSnapshotGetXMLDescFuncType)(virDomainSnapshotPtr snapshot,
                                       unsigned int flags);

char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = NULL;
    static virDomainSnapshotGetXMLDescFuncType virDomainSnapshotGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotGetXMLDesc",
                       (void**)&virDomainSnapshotGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotGetXMLDescSymbol(snapshot,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotHasMetadataFuncType)(virDomainSnapshotPtr snapshot,
                                        unsigned int flags);

int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotHasMetadataFuncType virDomainSnapshotHasMetadataSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotHasMetadata",
                       (void**)&virDomainSnapshotHasMetadataSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotHasMetadataSymbol(snapshot,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotIsCurrentFuncType)(virDomainSnapshotPtr snapshot,
                                      unsigned int flags);

int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotIsCurrentFuncType virDomainSnapshotIsCurrentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotIsCurrent",
                       (void**)&virDomainSnapshotIsCurrentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotIsCurrentSymbol(snapshot,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotListAllChildrenFuncType)(virDomainSnapshotPtr snapshot,
                                            virDomainSnapshotPtr ** snaps,
                                            unsigned int flags);

int
virDomainSnapshotListAllChildrenWrapper(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotListAllChildrenFuncType virDomainSnapshotListAllChildrenSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotListAllChildren",
                       (void**)&virDomainSnapshotListAllChildrenSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotListAllChildrenSymbol(snapshot,
                                                 snaps,
                                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotListChildrenNamesFuncType)(virDomainSnapshotPtr snapshot,
                                              char ** names,
                                              int nameslen,
                                              unsigned int flags);

int
virDomainSnapshotListChildrenNamesWrapper(virDomainSnapshotPtr snapshot,
                                          char ** names,
                                          int nameslen,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotListChildrenNamesFuncType virDomainSnapshotListChildrenNamesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotListChildrenNames",
                       (void**)&virDomainSnapshotListChildrenNamesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotListChildrenNamesSymbol(snapshot,
                                                   names,
                                                   nameslen,
                                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotListNamesFuncType)(virDomainPtr domain,
                                      char ** names,
                                      int nameslen,
                                      unsigned int flags);

int
virDomainSnapshotListNamesWrapper(virDomainPtr domain,
                                  char ** names,
                                  int nameslen,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotListNamesFuncType virDomainSnapshotListNamesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotListNames",
                       (void**)&virDomainSnapshotListNamesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotListNamesSymbol(domain,
                                           names,
                                           nameslen,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotLookupByNameFuncType)(virDomainPtr domain,
                                         const char * name,
                                         unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotLookupByNameFuncType virDomainSnapshotLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotLookupByName",
                       (void**)&virDomainSnapshotLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotLookupByNameSymbol(domain,
                                              name,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotNumFuncType)(virDomainPtr domain,
                                unsigned int flags);

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotNumFuncType virDomainSnapshotNumSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotNum",
                       (void**)&virDomainSnapshotNumSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotNumSymbol(domain,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotNumChildrenFuncType)(virDomainSnapshotPtr snapshot,
                                        unsigned int flags);

int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotNumChildrenFuncType virDomainSnapshotNumChildrenSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotNumChildren",
                       (void**)&virDomainSnapshotNumChildrenSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotNumChildrenSymbol(snapshot,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotRefFuncType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotRefFuncType virDomainSnapshotRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSnapshotRef",
                       (void**)&virDomainSnapshotRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSnapshotRefSymbol(snapshot);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
