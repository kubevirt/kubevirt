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
(*virDomainHasCurrentSnapshotType)(virDomainPtr domain,
                                   unsigned int flags);

int
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainHasCurrentSnapshotType virDomainHasCurrentSnapshotSymbol;
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
(*virDomainListAllSnapshotsType)(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags);

int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainListAllSnapshotsType virDomainListAllSnapshotsSymbol;
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
(*virDomainRevertToSnapshotType)(virDomainSnapshotPtr snapshot,
                                 unsigned int flags);

int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainRevertToSnapshotType virDomainRevertToSnapshotSymbol;
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
(*virDomainSnapshotCreateXMLType)(virDomainPtr domain,
                                  const char * xmlDesc,
                                  unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCreateXMLWrapper(virDomainPtr domain,
                                  const char * xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotCreateXMLType virDomainSnapshotCreateXMLSymbol;
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
(*virDomainSnapshotCurrentType)(virDomainPtr domain,
                                unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotCurrentType virDomainSnapshotCurrentSymbol;
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
(*virDomainSnapshotDeleteType)(virDomainSnapshotPtr snapshot,
                               unsigned int flags);

int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotDeleteType virDomainSnapshotDeleteSymbol;
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
(*virDomainSnapshotFreeType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotFreeType virDomainSnapshotFreeSymbol;
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
(*virDomainSnapshotGetConnectType)(virDomainSnapshotPtr snapshot);

virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virDomainSnapshotGetConnectType virDomainSnapshotGetConnectSymbol;
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
(*virDomainSnapshotGetDomainType)(virDomainSnapshotPtr snapshot);

virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainSnapshotGetDomainType virDomainSnapshotGetDomainSymbol;
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
(*virDomainSnapshotGetNameType)(virDomainSnapshotPtr snapshot);

const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err)
{
    const char * ret = NULL;
    static virDomainSnapshotGetNameType virDomainSnapshotGetNameSymbol;
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
(*virDomainSnapshotGetParentType)(virDomainSnapshotPtr snapshot,
                                  unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotGetParentType virDomainSnapshotGetParentSymbol;
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
(*virDomainSnapshotGetXMLDescType)(virDomainSnapshotPtr snapshot,
                                   unsigned int flags);

char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = NULL;
    static virDomainSnapshotGetXMLDescType virDomainSnapshotGetXMLDescSymbol;
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
(*virDomainSnapshotHasMetadataType)(virDomainSnapshotPtr snapshot,
                                    unsigned int flags);

int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotHasMetadataType virDomainSnapshotHasMetadataSymbol;
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
(*virDomainSnapshotIsCurrentType)(virDomainSnapshotPtr snapshot,
                                  unsigned int flags);

int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotIsCurrentType virDomainSnapshotIsCurrentSymbol;
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
(*virDomainSnapshotListAllChildrenType)(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags);

int
virDomainSnapshotListAllChildrenWrapper(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotListAllChildrenType virDomainSnapshotListAllChildrenSymbol;
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
(*virDomainSnapshotListChildrenNamesType)(virDomainSnapshotPtr snapshot,
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
    static virDomainSnapshotListChildrenNamesType virDomainSnapshotListChildrenNamesSymbol;
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
(*virDomainSnapshotListNamesType)(virDomainPtr domain,
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
    static virDomainSnapshotListNamesType virDomainSnapshotListNamesSymbol;
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
(*virDomainSnapshotLookupByNameType)(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
    static virDomainSnapshotLookupByNameType virDomainSnapshotLookupByNameSymbol;
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
(*virDomainSnapshotNumType)(virDomainPtr domain,
                            unsigned int flags);

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotNumType virDomainSnapshotNumSymbol;
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
(*virDomainSnapshotNumChildrenType)(virDomainSnapshotPtr snapshot,
                                    unsigned int flags);

int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotNumChildrenType virDomainSnapshotNumChildrenSymbol;
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
(*virDomainSnapshotRefType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainSnapshotRefType virDomainSnapshotRefSymbol;
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
