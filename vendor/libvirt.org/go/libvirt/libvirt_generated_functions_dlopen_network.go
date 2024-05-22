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
(*virConnectListAllNetworksType)(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags);

int
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllNetworksType virConnectListAllNetworksSymbol;
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
(*virConnectListDefinedNetworksType)(virConnectPtr conn,
                                     char ** const names,
                                     int maxnames);

int
virConnectListDefinedNetworksWrapper(virConnectPtr conn,
                                     char ** const names,
                                     int maxnames,
                                     virErrorPtr err)
{
    int ret = -1;
    static virConnectListDefinedNetworksType virConnectListDefinedNetworksSymbol;
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
(*virConnectListNetworksType)(virConnectPtr conn,
                              char ** const names,
                              int maxnames);

int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char ** const names,
                              int maxnames,
                              virErrorPtr err)
{
    int ret = -1;
    static virConnectListNetworksType virConnectListNetworksSymbol;
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
(*virConnectNetworkEventDeregisterAnyType)(virConnectPtr conn,
                                           int callbackID);

int
virConnectNetworkEventDeregisterAnyWrapper(virConnectPtr conn,
                                           int callbackID,
                                           virErrorPtr err)
{
    int ret = -1;
    static virConnectNetworkEventDeregisterAnyType virConnectNetworkEventDeregisterAnySymbol;
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
(*virConnectNetworkEventRegisterAnyType)(virConnectPtr conn,
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
    static virConnectNetworkEventRegisterAnyType virConnectNetworkEventRegisterAnySymbol;
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
(*virConnectNumOfDefinedNetworksType)(virConnectPtr conn);

int
virConnectNumOfDefinedNetworksWrapper(virConnectPtr conn,
                                      virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDefinedNetworksType virConnectNumOfDefinedNetworksSymbol;
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
(*virConnectNumOfNetworksType)(virConnectPtr conn);

int
virConnectNumOfNetworksWrapper(virConnectPtr conn,
                               virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfNetworksType virConnectNumOfNetworksSymbol;
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
(*virNetworkCreateType)(virNetworkPtr network);

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err)
{
    int ret = -1;
    static virNetworkCreateType virNetworkCreateSymbol;
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
(*virNetworkCreateXMLType)(virConnectPtr conn,
                           const char * xmlDesc);

virNetworkPtr
virNetworkCreateXMLWrapper(virConnectPtr conn,
                           const char * xmlDesc,
                           virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkCreateXMLType virNetworkCreateXMLSymbol;
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
(*virNetworkCreateXMLFlagsType)(virConnectPtr conn,
                                const char * xmlDesc,
                                unsigned int flags);

virNetworkPtr
virNetworkCreateXMLFlagsWrapper(virConnectPtr conn,
                                const char * xmlDesc,
                                unsigned int flags,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkCreateXMLFlagsType virNetworkCreateXMLFlagsSymbol;
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
(*virNetworkDHCPLeaseFreeType)(virNetworkDHCPLeasePtr lease);

void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease)
{

    static virNetworkDHCPLeaseFreeType virNetworkDHCPLeaseFreeSymbol;
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
(*virNetworkDefineXMLType)(virConnectPtr conn,
                           const char * xml);

virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char * xml,
                           virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkDefineXMLType virNetworkDefineXMLSymbol;
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
(*virNetworkDefineXMLFlagsType)(virConnectPtr conn,
                                const char * xml,
                                unsigned int flags);

virNetworkPtr
virNetworkDefineXMLFlagsWrapper(virConnectPtr conn,
                                const char * xml,
                                unsigned int flags,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkDefineXMLFlagsType virNetworkDefineXMLFlagsSymbol;
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
(*virNetworkDestroyType)(virNetworkPtr network);

int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    int ret = -1;
    static virNetworkDestroyType virNetworkDestroySymbol;
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
(*virNetworkFreeType)(virNetworkPtr network);

int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err)
{
    int ret = -1;
    static virNetworkFreeType virNetworkFreeSymbol;
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
(*virNetworkGetAutostartType)(virNetworkPtr network,
                              int * autostart);

int
virNetworkGetAutostartWrapper(virNetworkPtr network,
                              int * autostart,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetAutostartType virNetworkGetAutostartSymbol;
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
(*virNetworkGetBridgeNameType)(virNetworkPtr network);

char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkGetBridgeNameType virNetworkGetBridgeNameSymbol;
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
(*virNetworkGetConnectType)(virNetworkPtr net);

virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virNetworkGetConnectType virNetworkGetConnectSymbol;
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
(*virNetworkGetDHCPLeasesType)(virNetworkPtr network,
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
    static virNetworkGetDHCPLeasesType virNetworkGetDHCPLeasesSymbol;
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
(*virNetworkGetMetadataType)(virNetworkPtr network,
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
    static virNetworkGetMetadataType virNetworkGetMetadataSymbol;
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
(*virNetworkGetNameType)(virNetworkPtr network);

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    const char * ret = NULL;
    static virNetworkGetNameType virNetworkGetNameSymbol;
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
(*virNetworkGetUUIDType)(virNetworkPtr network,
                         unsigned char * uuid);

int
virNetworkGetUUIDWrapper(virNetworkPtr network,
                         unsigned char * uuid,
                         virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetUUIDType virNetworkGetUUIDSymbol;
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
(*virNetworkGetUUIDStringType)(virNetworkPtr network,
                               char * buf);

int
virNetworkGetUUIDStringWrapper(virNetworkPtr network,
                               char * buf,
                               virErrorPtr err)
{
    int ret = -1;
    static virNetworkGetUUIDStringType virNetworkGetUUIDStringSymbol;
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
(*virNetworkGetXMLDescType)(virNetworkPtr network,
                            unsigned int flags);

char *
virNetworkGetXMLDescWrapper(virNetworkPtr network,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkGetXMLDescType virNetworkGetXMLDescSymbol;
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
(*virNetworkIsActiveType)(virNetworkPtr net);

int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err)
{
    int ret = -1;
    static virNetworkIsActiveType virNetworkIsActiveSymbol;
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
(*virNetworkIsPersistentType)(virNetworkPtr net);

int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkIsPersistentType virNetworkIsPersistentSymbol;
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
(*virNetworkListAllPortsType)(virNetworkPtr network,
                              virNetworkPortPtr ** ports,
                              unsigned int flags);

int
virNetworkListAllPortsWrapper(virNetworkPtr network,
                              virNetworkPortPtr ** ports,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkListAllPortsType virNetworkListAllPortsSymbol;
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
(*virNetworkLookupByNameType)(virConnectPtr conn,
                              const char * name);

virNetworkPtr
virNetworkLookupByNameWrapper(virConnectPtr conn,
                              const char * name,
                              virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkLookupByNameType virNetworkLookupByNameSymbol;
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
(*virNetworkLookupByUUIDType)(virConnectPtr conn,
                              const unsigned char * uuid);

virNetworkPtr
virNetworkLookupByUUIDWrapper(virConnectPtr conn,
                              const unsigned char * uuid,
                              virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkLookupByUUIDType virNetworkLookupByUUIDSymbol;
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
(*virNetworkLookupByUUIDStringType)(virConnectPtr conn,
                                    const char * uuidstr);

virNetworkPtr
virNetworkLookupByUUIDStringWrapper(virConnectPtr conn,
                                    const char * uuidstr,
                                    virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkLookupByUUIDStringType virNetworkLookupByUUIDStringSymbol;
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
(*virNetworkPortCreateXMLType)(virNetworkPtr net,
                               const char * xmldesc,
                               unsigned int flags);

virNetworkPortPtr
virNetworkPortCreateXMLWrapper(virNetworkPtr net,
                               const char * xmldesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
    static virNetworkPortCreateXMLType virNetworkPortCreateXMLSymbol;
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
(*virNetworkPortDeleteType)(virNetworkPortPtr port,
                            unsigned int flags);

int
virNetworkPortDeleteWrapper(virNetworkPortPtr port,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortDeleteType virNetworkPortDeleteSymbol;
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
(*virNetworkPortFreeType)(virNetworkPortPtr port);

int
virNetworkPortFreeWrapper(virNetworkPortPtr port,
                          virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortFreeType virNetworkPortFreeSymbol;
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
(*virNetworkPortGetNetworkType)(virNetworkPortPtr port);

virNetworkPtr
virNetworkPortGetNetworkWrapper(virNetworkPortPtr port,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
    static virNetworkPortGetNetworkType virNetworkPortGetNetworkSymbol;
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
(*virNetworkPortGetParametersType)(virNetworkPortPtr port,
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
    static virNetworkPortGetParametersType virNetworkPortGetParametersSymbol;
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
(*virNetworkPortGetUUIDType)(virNetworkPortPtr port,
                             unsigned char * uuid);

int
virNetworkPortGetUUIDWrapper(virNetworkPortPtr port,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortGetUUIDType virNetworkPortGetUUIDSymbol;
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
(*virNetworkPortGetUUIDStringType)(virNetworkPortPtr port,
                                   char * buf);

int
virNetworkPortGetUUIDStringWrapper(virNetworkPortPtr port,
                                   char * buf,
                                   virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortGetUUIDStringType virNetworkPortGetUUIDStringSymbol;
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
(*virNetworkPortGetXMLDescType)(virNetworkPortPtr port,
                                unsigned int flags);

char *
virNetworkPortGetXMLDescWrapper(virNetworkPortPtr port,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = NULL;
    static virNetworkPortGetXMLDescType virNetworkPortGetXMLDescSymbol;
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
(*virNetworkPortLookupByUUIDType)(virNetworkPtr net,
                                  const unsigned char * uuid);

virNetworkPortPtr
virNetworkPortLookupByUUIDWrapper(virNetworkPtr net,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
    static virNetworkPortLookupByUUIDType virNetworkPortLookupByUUIDSymbol;
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
(*virNetworkPortLookupByUUIDStringType)(virNetworkPtr net,
                                        const char * uuidstr);

virNetworkPortPtr
virNetworkPortLookupByUUIDStringWrapper(virNetworkPtr net,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    virNetworkPortPtr ret = NULL;
    static virNetworkPortLookupByUUIDStringType virNetworkPortLookupByUUIDStringSymbol;
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
(*virNetworkPortRefType)(virNetworkPortPtr port);

int
virNetworkPortRefWrapper(virNetworkPortPtr port,
                         virErrorPtr err)
{
    int ret = -1;
    static virNetworkPortRefType virNetworkPortRefSymbol;
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
(*virNetworkPortSetParametersType)(virNetworkPortPtr port,
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
    static virNetworkPortSetParametersType virNetworkPortSetParametersSymbol;
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
(*virNetworkRefType)(virNetworkPtr network);

int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err)
{
    int ret = -1;
    static virNetworkRefType virNetworkRefSymbol;
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
(*virNetworkSetAutostartType)(virNetworkPtr network,
                              int autostart);

int
virNetworkSetAutostartWrapper(virNetworkPtr network,
                              int autostart,
                              virErrorPtr err)
{
    int ret = -1;
    static virNetworkSetAutostartType virNetworkSetAutostartSymbol;
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
(*virNetworkSetMetadataType)(virNetworkPtr network,
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
    static virNetworkSetMetadataType virNetworkSetMetadataSymbol;
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
(*virNetworkUndefineType)(virNetworkPtr network);

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err)
{
    int ret = -1;
    static virNetworkUndefineType virNetworkUndefineSymbol;
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
(*virNetworkUpdateType)(virNetworkPtr network,
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
    static virNetworkUpdateType virNetworkUpdateSymbol;
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
