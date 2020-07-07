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
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <assert.h>
#include "network_wrapper.h"

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err)
{
    int ret = virNetworkCreate(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease)
{
#if LIBVIR_VERSION_NUMBER < 1002006
    assert(0); // Caller should have checked version
#else
    virNetworkDHCPLeaseFree(lease);
#endif
}


int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    int ret = virNetworkDestroy(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err)
{
    int ret = virNetworkFree(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkGetAutostartWrapper(virNetworkPtr network,
                              int *autostart,
                              virErrorPtr err)
{
    int ret = virNetworkGetAutostart(network, autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err)
{
    char * ret = virNetworkGetBridgeName(network);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err)
{
    virConnectPtr ret = virNetworkGetConnect(net);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkGetDHCPLeasesWrapper(virNetworkPtr network,
                               const char *mac,
                               virNetworkDHCPLeasePtr **leases,
                               unsigned int flags,
                               virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002006
    assert(0); // Caller should have checked version
#else
    int ret = virNetworkGetDHCPLeases(network, mac, leases, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    const char * ret = virNetworkGetName(network);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkGetUUIDWrapper(virNetworkPtr network,
                         unsigned char *uuid,
                         virErrorPtr err)
{
    int ret = virNetworkGetUUID(network, uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkGetUUIDStringWrapper(virNetworkPtr network,
                               char *buf,
                               virErrorPtr err)
{
    int ret = virNetworkGetUUIDString(network, buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virNetworkGetXMLDescWrapper(virNetworkPtr network,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = virNetworkGetXMLDesc(network, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err)
{
    int ret = virNetworkIsActive(net);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err)
{
    int ret = virNetworkIsPersistent(net);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err)
{
    int ret = virNetworkRef(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkSetAutostartWrapper(virNetworkPtr network,
                              int autostart,
                              virErrorPtr err)
{
    int ret = virNetworkSetAutostart(network, autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err)
{
    int ret = virNetworkUndefine(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNetworkUpdateWrapper(virNetworkPtr network,
                        unsigned int command,
                        unsigned int section,
                        int parentIndex,
                        const char *xml,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = virNetworkUpdate(network, command, section, parentIndex, xml, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
