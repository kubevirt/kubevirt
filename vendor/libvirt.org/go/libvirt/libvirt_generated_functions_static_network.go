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
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllNetworks not available prior to libvirt version 0.10.2");
#else
    ret = virConnectListAllNetworks(conn,
                                    nets,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListDefinedNetworksWrapper(virConnectPtr conn,
                                     char ** const names,
                                     int maxnames,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectListDefinedNetworks not available prior to libvirt version 0.2.0");
#else
    ret = virConnectListDefinedNetworks(conn,
                                        names,
                                        maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char ** const names,
                              int maxnames,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectListNetworks not available prior to libvirt version 0.2.0");
#else
    ret = virConnectListNetworks(conn,
                                 names,
                                 maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNetworkEventDeregisterAnyWrapper(virConnectPtr conn,
                                           int callbackID,
                                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 1)
    setVirError(err, "Function virConnectNetworkEventDeregisterAny not available prior to libvirt version 1.2.1");
#else
    ret = virConnectNetworkEventDeregisterAny(conn,
                                              callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNetworkEventRegisterAnyWrapper(virConnectPtr conn,
                                         virNetworkPtr net,
                                         int eventID,
                                         virConnectNetworkEventGenericCallback cb,
                                         void * opaque,
                                         virFreeCallback freecb,
                                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 1)
    setVirError(err, "Function virConnectNetworkEventRegisterAny not available prior to libvirt version 1.2.1");
#else
    ret = virConnectNetworkEventRegisterAny(conn,
                                            net,
                                            eventID,
                                            cb,
                                            opaque,
                                            freecb);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfDefinedNetworksWrapper(virConnectPtr conn,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectNumOfDefinedNetworks not available prior to libvirt version 0.2.0");
#else
    ret = virConnectNumOfDefinedNetworks(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfNetworksWrapper(virConnectPtr conn,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectNumOfNetworks not available prior to libvirt version 0.2.0");
#else
    ret = virConnectNumOfNetworks(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkCreate not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkCreate(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkCreateXMLWrapper(virConnectPtr conn,
                           const char * xmlDesc,
                           virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkCreateXML not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkCreateXML(conn,
                              xmlDesc);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkCreateXMLFlagsWrapper(virConnectPtr conn,
                                const char * xmlDesc,
                                unsigned int flags,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNetworkCreateXMLFlags not available prior to libvirt version 7.8.0");
#else
    ret = virNetworkCreateXMLFlags(conn,
                                   xmlDesc,
                                   flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease)
{

#if !LIBVIR_CHECK_VERSION(1, 2, 6)
    setVirError(NULL, "Function virNetworkDHCPLeaseFree not available prior to libvirt version 1.2.6");
#else
    virNetworkDHCPLeaseFree(lease);
#endif
    return;
}

virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char * xml,
                           virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkDefineXML not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkDefineXML(conn,
                              xml);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkDefineXMLFlagsWrapper(virConnectPtr conn,
                                const char * xml,
                                unsigned int flags,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(7, 7, 0)
    setVirError(err, "Function virNetworkDefineXMLFlags not available prior to libvirt version 7.7.0");
#else
    ret = virNetworkDefineXMLFlags(conn,
                                   xml,
                                   flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkDestroy not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkDestroy(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkFree not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkFree(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkGetAutostartWrapper(virNetworkPtr network,
                              int * autostart,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virNetworkGetAutostart not available prior to libvirt version 0.2.1");
#else
    ret = virNetworkGetAutostart(network,
                                 autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetBridgeName not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkGetBridgeName(network);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virNetworkGetConnect not available prior to libvirt version 0.3.0");
#else
    ret = virNetworkGetConnect(net);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkGetDHCPLeasesWrapper(virNetworkPtr network,
                               const char * mac,
                               virNetworkDHCPLeasePtr ** leases,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 6)
    setVirError(err, "Function virNetworkGetDHCPLeases not available prior to libvirt version 1.2.6");
#else
    ret = virNetworkGetDHCPLeases(network,
                                  mac,
                                  leases,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNetworkGetMetadataWrapper(virNetworkPtr network,
                             int type,
                             const char * uri,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(9, 7, 0)
    setVirError(err, "Function virNetworkGetMetadata not available prior to libvirt version 9.7.0");
#else
    ret = virNetworkGetMetadata(network,
                                type,
                                uri,
                                flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetName not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkGetName(network);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkGetUUIDWrapper(virNetworkPtr network,
                         unsigned char * uuid,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetUUID not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkGetUUID(network,
                            uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkGetUUIDStringWrapper(virNetworkPtr network,
                               char * buf,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetUUIDString not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkGetUUIDString(network,
                                  buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNetworkGetXMLDescWrapper(virNetworkPtr network,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetXMLDesc not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkGetXMLDesc(network,
                               flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virNetworkIsActive not available prior to libvirt version 0.7.3");
#else
    ret = virNetworkIsActive(net);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virNetworkIsPersistent not available prior to libvirt version 0.7.3");
#else
    ret = virNetworkIsPersistent(net);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkListAllPortsWrapper(virNetworkPtr network,
                              virNetworkPortPtr ** ports,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkListAllPorts not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkListAllPorts(network,
                                 ports,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkLookupByNameWrapper(virConnectPtr conn,
                              const char * name,
                              virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkLookupByName not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkLookupByName(conn,
                                 name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkLookupByUUIDWrapper(virConnectPtr conn,
                              const unsigned char * uuid,
                              virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkLookupByUUID not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkLookupByUUID(conn,
                                 uuid);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkLookupByUUIDStringWrapper(virConnectPtr conn,
                                    const char * uuidstr,
                                    virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkLookupByUUIDString not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkLookupByUUIDString(conn,
                                       uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPortPtr
virNetworkPortCreateXMLWrapper(virNetworkPtr net,
                               const char * xmldesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortCreateXML not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortCreateXML(net,
                                  xmldesc,
                                  flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortDeleteWrapper(virNetworkPortPtr port,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortDelete not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortDelete(port,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortFreeWrapper(virNetworkPortPtr port,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortFree not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortFree(port);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPtr
virNetworkPortGetNetworkWrapper(virNetworkPortPtr port,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetNetwork not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortGetNetwork(port);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortGetParametersWrapper(virNetworkPortPtr port,
                                   virTypedParameterPtr * params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetParameters not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortGetParameters(port,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortGetUUIDWrapper(virNetworkPortPtr port,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetUUID not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortGetUUID(port,
                                uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortGetUUIDStringWrapper(virNetworkPortPtr port,
                                   char * buf,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetUUIDString not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortGetUUIDString(port,
                                      buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNetworkPortGetXMLDescWrapper(virNetworkPortPtr port,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetXMLDesc not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortGetXMLDesc(port,
                                   flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPortPtr
virNetworkPortLookupByUUIDWrapper(virNetworkPtr net,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortLookupByUUID not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortLookupByUUID(net,
                                     uuid);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNetworkPortPtr
virNetworkPortLookupByUUIDStringWrapper(virNetworkPtr net,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortLookupByUUIDString not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortLookupByUUIDString(net,
                                           uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortRefWrapper(virNetworkPortPtr port,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortRef not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortRef(port);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkPortSetParametersWrapper(virNetworkPortPtr port,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortSetParameters not available prior to libvirt version 5.5.0");
#else
    ret = virNetworkPortSetParameters(port,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virNetworkRef not available prior to libvirt version 0.6.0");
#else
    ret = virNetworkRef(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkSetAutostartWrapper(virNetworkPtr network,
                              int autostart,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virNetworkSetAutostart not available prior to libvirt version 0.2.1");
#else
    ret = virNetworkSetAutostart(network,
                                 autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkSetMetadataWrapper(virNetworkPtr network,
                             int type,
                             const char * metadata,
                             const char * key,
                             const char * uri,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(9, 7, 0)
    setVirError(err, "Function virNetworkSetMetadata not available prior to libvirt version 9.7.0");
#else
    ret = virNetworkSetMetadata(network,
                                type,
                                metadata,
                                key,
                                uri,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkUndefine not available prior to libvirt version 0.2.0");
#else
    ret = virNetworkUndefine(network);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNetworkUpdateWrapper(virNetworkPtr network,
                        unsigned int command,
                        unsigned int section,
                        int parentIndex,
                        const char * xml,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virNetworkUpdate not available prior to libvirt version 0.10.2");
#else
    ret = virNetworkUpdate(network,
                           command,
                           section,
                           parentIndex,
                           xml,
                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
