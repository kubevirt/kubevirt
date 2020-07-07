/*
 * This file is part of the libvirt-go project
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
 * Copyright (C) 2018 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <assert.h>
#include "domain_snapshot_wrapper.h"


int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = virDomainRevertToSnapshot(snapshot, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = virDomainSnapshotDelete(snapshot, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err)
{
    int ret = virDomainSnapshotFree(snapshot);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err)
{
    virConnectPtr ret = virDomainSnapshotGetConnect(snapshot);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err)
{
    virDomainPtr ret = virDomainSnapshotGetDomain(snapshot);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err)
{
    const char * ret = virDomainSnapshotGetName(snapshot);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = virDomainSnapshotGetParent(snapshot, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = virDomainSnapshotGetXMLDesc(snapshot, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virDomainSnapshotHasMetadata(snapshot, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainSnapshotIsCurrent(snapshot, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotListAllChildrenWrapper(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr **snaps,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = virDomainSnapshotListAllChildren(snapshot, snaps, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotListChildrenNamesWrapper(virDomainSnapshotPtr snapshot,
                                          char **names,
                                          int nameslen,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    int ret = virDomainSnapshotListChildrenNames(snapshot, names, nameslen, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = virDomainSnapshotNumChildren(snapshot, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err)
{
    int ret = virDomainSnapshotRef(snapshot);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
