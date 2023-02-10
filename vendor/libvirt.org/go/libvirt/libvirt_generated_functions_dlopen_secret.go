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
(*virConnectListAllSecretsType)(virConnectPtr conn,
                                virSecretPtr ** secrets,
                                unsigned int flags);

int
virConnectListAllSecretsWrapper(virConnectPtr conn,
                                virSecretPtr ** secrets,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllSecretsType virConnectListAllSecretsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllSecrets",
                       (void**)&virConnectListAllSecretsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllSecretsSymbol(conn,
                                         secrets,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListSecretsType)(virConnectPtr conn,
                             char ** uuids,
                             int maxuuids);

int
virConnectListSecretsWrapper(virConnectPtr conn,
                             char ** uuids,
                             int maxuuids,
                             virErrorPtr err)
{
    int ret = -1;
    static virConnectListSecretsType virConnectListSecretsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListSecrets",
                       (void**)&virConnectListSecretsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListSecretsSymbol(conn,
                                      uuids,
                                      maxuuids);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfSecretsType)(virConnectPtr conn);

int
virConnectNumOfSecretsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfSecretsType virConnectNumOfSecretsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfSecrets",
                       (void**)&virConnectNumOfSecretsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfSecretsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectSecretEventDeregisterAnyType)(virConnectPtr conn,
                                          int callbackID);

int
virConnectSecretEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    int ret = -1;
    static virConnectSecretEventDeregisterAnyType virConnectSecretEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectSecretEventDeregisterAny",
                       (void**)&virConnectSecretEventDeregisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectSecretEventDeregisterAnySymbol(conn,
                                                   callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectSecretEventRegisterAnyType)(virConnectPtr conn,
                                        virSecretPtr secret,
                                        int eventID,
                                        virConnectSecretEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb);

int
virConnectSecretEventRegisterAnyWrapper(virConnectPtr conn,
                                        virSecretPtr secret,
                                        int eventID,
                                        virConnectSecretEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb,
                                        virErrorPtr err)
{
    int ret = -1;
    static virConnectSecretEventRegisterAnyType virConnectSecretEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectSecretEventRegisterAny",
                       (void**)&virConnectSecretEventRegisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectSecretEventRegisterAnySymbol(conn,
                                                 secret,
                                                 eventID,
                                                 cb,
                                                 opaque,
                                                 freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virSecretPtr
(*virSecretDefineXMLType)(virConnectPtr conn,
                          const char * xml,
                          unsigned int flags);

virSecretPtr
virSecretDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          unsigned int flags,
                          virErrorPtr err)
{
    virSecretPtr ret = NULL;
    static virSecretDefineXMLType virSecretDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretDefineXML",
                       (void**)&virSecretDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretDefineXMLSymbol(conn,
                                   xml,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretFreeType)(virSecretPtr secret);

int
virSecretFreeWrapper(virSecretPtr secret,
                     virErrorPtr err)
{
    int ret = -1;
    static virSecretFreeType virSecretFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretFree",
                       (void**)&virSecretFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretFreeSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virSecretGetConnectType)(virSecretPtr secret);

virConnectPtr
virSecretGetConnectWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virSecretGetConnectType virSecretGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetConnect",
                       (void**)&virSecretGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetConnectSymbol(secret);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretGetUUIDType)(virSecretPtr secret,
                        unsigned char * uuid);

int
virSecretGetUUIDWrapper(virSecretPtr secret,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    int ret = -1;
    static virSecretGetUUIDType virSecretGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetUUID",
                       (void**)&virSecretGetUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetUUIDSymbol(secret,
                                 uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretGetUUIDStringType)(virSecretPtr secret,
                              char * buf);

int
virSecretGetUUIDStringWrapper(virSecretPtr secret,
                              char * buf,
                              virErrorPtr err)
{
    int ret = -1;
    static virSecretGetUUIDStringType virSecretGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetUUIDString",
                       (void**)&virSecretGetUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetUUIDStringSymbol(secret,
                                       buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virSecretGetUsageIDType)(virSecretPtr secret);

const char *
virSecretGetUsageIDWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    const char * ret = NULL;
    static virSecretGetUsageIDType virSecretGetUsageIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetUsageID",
                       (void**)&virSecretGetUsageIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetUsageIDSymbol(secret);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretGetUsageTypeType)(virSecretPtr secret);

int
virSecretGetUsageTypeWrapper(virSecretPtr secret,
                             virErrorPtr err)
{
    int ret = -1;
    static virSecretGetUsageTypeType virSecretGetUsageTypeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetUsageType",
                       (void**)&virSecretGetUsageTypeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetUsageTypeSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned char *
(*virSecretGetValueType)(virSecretPtr secret,
                         size_t * value_size,
                         unsigned int flags);

unsigned char *
virSecretGetValueWrapper(virSecretPtr secret,
                         size_t * value_size,
                         unsigned int flags,
                         virErrorPtr err)
{
    unsigned char * ret = NULL;
    static virSecretGetValueType virSecretGetValueSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetValue",
                       (void**)&virSecretGetValueSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetValueSymbol(secret,
                                  value_size,
                                  flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virSecretGetXMLDescType)(virSecretPtr secret,
                           unsigned int flags);

char *
virSecretGetXMLDescWrapper(virSecretPtr secret,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
    static virSecretGetXMLDescType virSecretGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretGetXMLDesc",
                       (void**)&virSecretGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretGetXMLDescSymbol(secret,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virSecretPtr
(*virSecretLookupByUUIDType)(virConnectPtr conn,
                             const unsigned char * uuid);

virSecretPtr
virSecretLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    virSecretPtr ret = NULL;
    static virSecretLookupByUUIDType virSecretLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretLookupByUUID",
                       (void**)&virSecretLookupByUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretLookupByUUIDSymbol(conn,
                                      uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virSecretPtr
(*virSecretLookupByUUIDStringType)(virConnectPtr conn,
                                   const char * uuidstr);

virSecretPtr
virSecretLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    virSecretPtr ret = NULL;
    static virSecretLookupByUUIDStringType virSecretLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretLookupByUUIDString",
                       (void**)&virSecretLookupByUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretLookupByUUIDStringSymbol(conn,
                                            uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virSecretPtr
(*virSecretLookupByUsageType)(virConnectPtr conn,
                              int usageType,
                              const char * usageID);

virSecretPtr
virSecretLookupByUsageWrapper(virConnectPtr conn,
                              int usageType,
                              const char * usageID,
                              virErrorPtr err)
{
    virSecretPtr ret = NULL;
    static virSecretLookupByUsageType virSecretLookupByUsageSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretLookupByUsage",
                       (void**)&virSecretLookupByUsageSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretLookupByUsageSymbol(conn,
                                       usageType,
                                       usageID);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretRefType)(virSecretPtr secret);

int
virSecretRefWrapper(virSecretPtr secret,
                    virErrorPtr err)
{
    int ret = -1;
    static virSecretRefType virSecretRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretRef",
                       (void**)&virSecretRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretRefSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretSetValueType)(virSecretPtr secret,
                         const unsigned char * value,
                         size_t value_size,
                         unsigned int flags);

int
virSecretSetValueWrapper(virSecretPtr secret,
                         const unsigned char * value,
                         size_t value_size,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virSecretSetValueType virSecretSetValueSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretSetValue",
                       (void**)&virSecretSetValueSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretSetValueSymbol(secret,
                                  value,
                                  value_size,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virSecretUndefineType)(virSecretPtr secret);

int
virSecretUndefineWrapper(virSecretPtr secret,
                         virErrorPtr err)
{
    int ret = -1;
    static virSecretUndefineType virSecretUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSecretUndefine",
                       (void**)&virSecretUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSecretUndefineSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
