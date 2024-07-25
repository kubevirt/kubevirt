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
(*virConnectListAllNetworksFuncType)(virConnectPtr conn,
                                     virNetworkPtr ** nets,
                                     unsigned int flags);

int
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllNetworksFuncType virConnectListAllNetworksSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllNetworks",
                       (void**)&virConnectListAllNetworksSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllNetworksSymbol(conn,
                                          nets,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDefinedNetworksFuncType)(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames);

int
virConnectListDefinedNetworksWrapper(virConnectPtr conn,
                                     char ** const names,
                                     int maxnames,
                                     virErrorPtr err)
{
    int ret = -1;
    static virConnectListDefinedNetworksFuncType virConnectListDefinedNetworksSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListDefinedNetworks",
                       (void**)&virConnectListDefinedNetworksSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListDefinedNetworksSymbol(conn,
                                              names,
                                              maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListNetworksFuncType)(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames);

int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char ** const names,
                              int maxnames,
                              virErrorPtr err)
{
    int ret = -1;
    static virConnectListNetworksFuncType virConnectListNetworksSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListNetworks",
                       (void**)&virConnectListNetworksSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListNetworksSymbol(conn,
                                       names,
                                       maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNetworkEventDeregisterAnyFuncType)(virConnectPtr conn,
                                               int callbackID);

int
virConnectNetworkEventDeregisterAnyWrapper(virConnectPtr conn,
                                           int callbackID,
                                           virErrorPtr err)
{
    int ret = -1;
    static virConnectNetworkEventDeregisterAnyFuncType virConnectNetworkEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNetworkEventDeregisterAny",
                       (void**)&virConnectNetworkEventDeregisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNetworkEventDeregisterAnySymbol(conn,
                                                    callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNetworkEventRegisterAnyFuncType)(virConnectPtr conn,
                                             virNetworkPtr net,
                                             int eventID,
                                             virConnectNetworkEventGenericCallback cb,
                                             void * opaque,
                                             virFreeCallback freecb);

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
    static virConnectNetworkEventRegisterAnyFuncType virConnectNetworkEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNetworkEventRegisterAny",
                       (void**)&virConnectNetworkEventRegisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNetworkEventRegisterAnySymbol(conn,
                                                  net,
                                                  eventID,
                                                  cb,
                                                  opaque,
                                                  freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDefinedNetworksFuncType)(virConnectPtr conn);

int
virConnectNumOfDefinedNetworksWrapper(virConnectPtr conn,
                                      virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDefinedNetworksFuncType virConnectNumOfDefinedNetworksSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfDefinedNetworks",
                       (void**)&virConnectNumOfDefinedNetworksSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfDefinedNetworksSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfNetworksFuncType)(virConnectPtr conn);

int
virConnectNumOfNetworksWrapper(virConnectPtr conn,
                               virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfNetworksFuncType virConnectNumOfNetworksSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfNetworks",
                       (void**)&virConnectNumOfNetworksSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfNetworksSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkCreateFuncType)(virNetworkPtr network);

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err)
{
    int ret = -1;
    static virNetworkCreateFuncType virNetworkCreateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkCreate",
                       (void**)&virNetworkCreateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkCreateSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkCreateXMLFuncType)(virConnectPtr conn,
                               const char * xmlDesc);

virNetworkPtr
virNetworkCreateXMLWrapper(virConnectPtr conn,
                           const char * xmlDesc,
                           virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkCreateXMLFuncType virNetworkCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkCreateXML",
                       (void**)&virNetworkCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkCreateXMLSymbol(conn,
                                    xmlDesc);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkCreateXMLFlagsFuncType)(virConnectPtr conn,
                                    const char * xmlDesc,
                                    unsigned int flags);

virNetworkPtr
virNetworkCreateXMLFlagsWrapper(virConnectPtr conn,
                                const char * xmlDesc,
                                unsigned int flags,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkCreateXMLFlagsFuncType virNetworkCreateXMLFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkCreateXMLFlags",
                       (void**)&virNetworkCreateXMLFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkCreateXMLFlagsSymbol(conn,
                                         xmlDesc,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virNetworkDHCPLeaseFreeFuncType)(virNetworkDHCPLeasePtr lease);

void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease)
{

    static virNetworkDHCPLeaseFreeFuncType virNetworkDHCPLeaseFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkDHCPLeaseFree",
                       (void**)&virNetworkDHCPLeaseFreeSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virNetworkDHCPLeaseFreeSymbol(lease);
}

typedef virNetworkPtr
(*virNetworkDefineXMLFuncType)(virConnectPtr conn,
                               const char * xml);

virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char * xml,
                           virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkDefineXMLFuncType virNetworkDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkDefineXML",
                       (void**)&virNetworkDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkDefineXMLSymbol(conn,
                                    xml);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkDefineXMLFlagsFuncType)(virConnectPtr conn,
                                    const char * xml,
                                    unsigned int flags);

virNetworkPtr
virNetworkDefineXMLFlagsWrapper(virConnectPtr conn,
                                const char * xml,
                                unsigned int flags,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkDefineXMLFlagsFuncType virNetworkDefineXMLFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkDefineXMLFlags",
                       (void**)&virNetworkDefineXMLFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkDefineXMLFlagsSymbol(conn,
                                         xml,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkDestroyFuncType)(virNetworkPtr network);

int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    int ret = -1;
    static virNetworkDestroyFuncType virNetworkDestroySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkDestroy",
                       (void**)&virNetworkDestroySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkDestroySymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkFreeFuncType)(virNetworkPtr network);

int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err)
{
    int ret = -1;
    static virNetworkFreeFuncType virNetworkFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkFree",
                       (void**)&virNetworkFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkFreeSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkGetAutostartFuncType)(virNetworkPtr network,
                                  int * autostart);

int
virNetworkGetAutostartWrapper(virNetworkPtr network,
                              int * autostart,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetAutostartFuncType virNetworkGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetAutostart",
                       (void**)&virNetworkGetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetAutostartSymbol(network,
                                       autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNetworkGetBridgeNameFuncType)(virNetworkPtr network);

char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkGetBridgeNameFuncType virNetworkGetBridgeNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetBridgeName",
                       (void**)&virNetworkGetBridgeNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetBridgeNameSymbol(network);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virNetworkGetConnectFuncType)(virNetworkPtr net);

virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virNetworkGetConnectFuncType virNetworkGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetConnect",
                       (void**)&virNetworkGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetConnectSymbol(net);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkGetDHCPLeasesFuncType)(virNetworkPtr network,
                                   const char * mac,
                                   virNetworkDHCPLeasePtr ** leases,
                                   unsigned int flags);

int
virNetworkGetDHCPLeasesWrapper(virNetworkPtr network,
                               const char * mac,
                               virNetworkDHCPLeasePtr ** leases,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetDHCPLeasesFuncType virNetworkGetDHCPLeasesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetDHCPLeases",
                       (void**)&virNetworkGetDHCPLeasesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetDHCPLeasesSymbol(network,
                                        mac,
                                        leases,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNetworkGetMetadataFuncType)(virNetworkPtr network,
                                 int type,
                                 const char * uri,
                                 unsigned int flags);

char *
virNetworkGetMetadataWrapper(virNetworkPtr network,
                             int type,
                             const char * uri,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkGetMetadataFuncType virNetworkGetMetadataSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetMetadata",
                       (void**)&virNetworkGetMetadataSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetMetadataSymbol(network,
                                      type,
                                      uri,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNetworkGetNameFuncType)(virNetworkPtr network);

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    const char * ret = NULL;
    static virNetworkGetNameFuncType virNetworkGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetName",
                       (void**)&virNetworkGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetNameSymbol(network);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkGetUUIDFuncType)(virNetworkPtr network,
                             unsigned char * uuid);

int
virNetworkGetUUIDWrapper(virNetworkPtr network,
                         unsigned char * uuid,
                         virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetUUIDFuncType virNetworkGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetUUID",
                       (void**)&virNetworkGetUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetUUIDSymbol(network,
                                  uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkGetUUIDStringFuncType)(virNetworkPtr network,
                                   char * buf);

int
virNetworkGetUUIDStringWrapper(virNetworkPtr network,
                               char * buf,
                               virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetUUIDStringFuncType virNetworkGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetUUIDString",
                       (void**)&virNetworkGetUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetUUIDStringSymbol(network,
                                        buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNetworkGetXMLDescFuncType)(virNetworkPtr network,
                                unsigned int flags);

char *
virNetworkGetXMLDescWrapper(virNetworkPtr network,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkGetXMLDescFuncType virNetworkGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkGetXMLDesc",
                       (void**)&virNetworkGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkGetXMLDescSymbol(network,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkIsActiveFuncType)(virNetworkPtr net);

int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err)
{
    int ret = -1;
    static virNetworkIsActiveFuncType virNetworkIsActiveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkIsActive",
                       (void**)&virNetworkIsActiveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkIsActiveSymbol(net);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkIsPersistentFuncType)(virNetworkPtr net);

int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkIsPersistentFuncType virNetworkIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkIsPersistent",
                       (void**)&virNetworkIsPersistentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkIsPersistentSymbol(net);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkListAllPortsFuncType)(virNetworkPtr network,
                                  virNetworkPortPtr ** ports,
                                  unsigned int flags);

int
virNetworkListAllPortsWrapper(virNetworkPtr network,
                              virNetworkPortPtr ** ports,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkListAllPortsFuncType virNetworkListAllPortsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkListAllPorts",
                       (void**)&virNetworkListAllPortsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkListAllPortsSymbol(network,
                                       ports,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkLookupByNameFuncType)(virConnectPtr conn,
                                  const char * name);

virNetworkPtr
virNetworkLookupByNameWrapper(virConnectPtr conn,
                              const char * name,
                              virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkLookupByNameFuncType virNetworkLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkLookupByName",
                       (void**)&virNetworkLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkLookupByNameSymbol(conn,
                                       name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkLookupByUUIDFuncType)(virConnectPtr conn,
                                  const unsigned char * uuid);

virNetworkPtr
virNetworkLookupByUUIDWrapper(virConnectPtr conn,
                              const unsigned char * uuid,
                              virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkLookupByUUIDFuncType virNetworkLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkLookupByUUID",
                       (void**)&virNetworkLookupByUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkLookupByUUIDSymbol(conn,
                                       uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkLookupByUUIDStringFuncType)(virConnectPtr conn,
                                        const char * uuidstr);

virNetworkPtr
virNetworkLookupByUUIDStringWrapper(virConnectPtr conn,
                                    const char * uuidstr,
                                    virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkLookupByUUIDStringFuncType virNetworkLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkLookupByUUIDString",
                       (void**)&virNetworkLookupByUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkLookupByUUIDStringSymbol(conn,
                                             uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPortPtr
(*virNetworkPortCreateXMLFuncType)(virNetworkPtr net,
                                   const char * xmldesc,
                                   unsigned int flags);

virNetworkPortPtr
virNetworkPortCreateXMLWrapper(virNetworkPtr net,
                               const char * xmldesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
    static virNetworkPortCreateXMLFuncType virNetworkPortCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortCreateXML",
                       (void**)&virNetworkPortCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortCreateXMLSymbol(net,
                                        xmldesc,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortDeleteFuncType)(virNetworkPortPtr port,
                                unsigned int flags);

int
virNetworkPortDeleteWrapper(virNetworkPortPtr port,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortDeleteFuncType virNetworkPortDeleteSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortDelete",
                       (void**)&virNetworkPortDeleteSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortDeleteSymbol(port,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortFreeFuncType)(virNetworkPortPtr port);

int
virNetworkPortFreeWrapper(virNetworkPortPtr port,
                          virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortFreeFuncType virNetworkPortFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortFree",
                       (void**)&virNetworkPortFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortFreeSymbol(port);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPtr
(*virNetworkPortGetNetworkFuncType)(virNetworkPortPtr port);

virNetworkPtr
virNetworkPortGetNetworkWrapper(virNetworkPortPtr port,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkPortGetNetworkFuncType virNetworkPortGetNetworkSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortGetNetwork",
                       (void**)&virNetworkPortGetNetworkSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortGetNetworkSymbol(port);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortGetParametersFuncType)(virNetworkPortPtr port,
                                       virTypedParameterPtr * params,
                                       int * nparams,
                                       unsigned int flags);

int
virNetworkPortGetParametersWrapper(virNetworkPortPtr port,
                                   virTypedParameterPtr * params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortGetParametersFuncType virNetworkPortGetParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortGetParameters",
                       (void**)&virNetworkPortGetParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortGetParametersSymbol(port,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortGetUUIDFuncType)(virNetworkPortPtr port,
                                 unsigned char * uuid);

int
virNetworkPortGetUUIDWrapper(virNetworkPortPtr port,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortGetUUIDFuncType virNetworkPortGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortGetUUID",
                       (void**)&virNetworkPortGetUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortGetUUIDSymbol(port,
                                      uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortGetUUIDStringFuncType)(virNetworkPortPtr port,
                                       char * buf);

int
virNetworkPortGetUUIDStringWrapper(virNetworkPortPtr port,
                                   char * buf,
                                   virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortGetUUIDStringFuncType virNetworkPortGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortGetUUIDString",
                       (void**)&virNetworkPortGetUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortGetUUIDStringSymbol(port,
                                            buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNetworkPortGetXMLDescFuncType)(virNetworkPortPtr port,
                                    unsigned int flags);

char *
virNetworkPortGetXMLDescWrapper(virNetworkPortPtr port,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkPortGetXMLDescFuncType virNetworkPortGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortGetXMLDesc",
                       (void**)&virNetworkPortGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortGetXMLDescSymbol(port,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPortPtr
(*virNetworkPortLookupByUUIDFuncType)(virNetworkPtr net,
                                      const unsigned char * uuid);

virNetworkPortPtr
virNetworkPortLookupByUUIDWrapper(virNetworkPtr net,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
    static virNetworkPortLookupByUUIDFuncType virNetworkPortLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortLookupByUUID",
                       (void**)&virNetworkPortLookupByUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortLookupByUUIDSymbol(net,
                                           uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNetworkPortPtr
(*virNetworkPortLookupByUUIDStringFuncType)(virNetworkPtr net,
                                            const char * uuidstr);

virNetworkPortPtr
virNetworkPortLookupByUUIDStringWrapper(virNetworkPtr net,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
    static virNetworkPortLookupByUUIDStringFuncType virNetworkPortLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortLookupByUUIDString",
                       (void**)&virNetworkPortLookupByUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortLookupByUUIDStringSymbol(net,
                                                 uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortRefFuncType)(virNetworkPortPtr port);

int
virNetworkPortRefWrapper(virNetworkPortPtr port,
                         virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortRefFuncType virNetworkPortRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortRef",
                       (void**)&virNetworkPortRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortRefSymbol(port);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkPortSetParametersFuncType)(virNetworkPortPtr port,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags);

int
virNetworkPortSetParametersWrapper(virNetworkPortPtr port,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortSetParametersFuncType virNetworkPortSetParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkPortSetParameters",
                       (void**)&virNetworkPortSetParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkPortSetParametersSymbol(port,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkRefFuncType)(virNetworkPtr network);

int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err)
{
    int ret = -1;
    static virNetworkRefFuncType virNetworkRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkRef",
                       (void**)&virNetworkRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkRefSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkSetAutostartFuncType)(virNetworkPtr network,
                                  int autostart);

int
virNetworkSetAutostartWrapper(virNetworkPtr network,
                              int autostart,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkSetAutostartFuncType virNetworkSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkSetAutostart",
                       (void**)&virNetworkSetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkSetAutostartSymbol(network,
                                       autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkSetMetadataFuncType)(virNetworkPtr network,
                                 int type,
                                 const char * metadata,
                                 const char * key,
                                 const char * uri,
                                 unsigned int flags);

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
    static virNetworkSetMetadataFuncType virNetworkSetMetadataSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkSetMetadata",
                       (void**)&virNetworkSetMetadataSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkSetMetadataSymbol(network,
                                      type,
                                      metadata,
                                      key,
                                      uri,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkUndefineFuncType)(virNetworkPtr network);

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err)
{
    int ret = -1;
    static virNetworkUndefineFuncType virNetworkUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkUndefine",
                       (void**)&virNetworkUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkUndefineSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNetworkUpdateFuncType)(virNetworkPtr network,
                            unsigned int command,
                            unsigned int section,
                            int parentIndex,
                            const char * xml,
                            unsigned int flags);

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
    static virNetworkUpdateFuncType virNetworkUpdateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNetworkUpdate",
                       (void**)&virNetworkUpdateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNetworkUpdateSymbol(network,
                                 command,
                                 section,
                                 parentIndex,
                                 xml,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
