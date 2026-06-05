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
virConnectListAllSecretsWrapper(virConnectPtr conn,
                                virSecretPtr ** secrets,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllSecrets not available prior to libvirt version 0.10.2");
#else
    ret = virConnectListAllSecrets(conn,
                                   secrets,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListSecretsWrapper(virConnectPtr conn,
                             char ** uuids,
                             int maxuuids,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virConnectListSecrets not available prior to libvirt version 0.7.1");
#else
    ret = virConnectListSecrets(conn,
                                uuids,
                                maxuuids);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfSecretsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virConnectNumOfSecrets not available prior to libvirt version 0.7.1");
#else
    ret = virConnectNumOfSecrets(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectSecretEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virConnectSecretEventDeregisterAny not available prior to libvirt version 3.0.0");
#else
    ret = virConnectSecretEventDeregisterAny(conn,
                                             callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virConnectSecretEventRegisterAny not available prior to libvirt version 3.0.0");
#else
    ret = virConnectSecretEventRegisterAny(conn,
                                           secret,
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

virSecretPtr
virSecretDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          unsigned int flags,
                          virErrorPtr err)
{
    virSecretPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretDefineXML not available prior to libvirt version 0.7.1");
#else
    ret = virSecretDefineXML(conn,
                             xml,
                             flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretFreeWrapper(virSecretPtr secret,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretFree not available prior to libvirt version 0.7.1");
#else
    ret = virSecretFree(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virSecretGetConnectWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetConnect not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetConnect(secret);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretGetUUIDWrapper(virSecretPtr secret,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUUID not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetUUID(secret,
                           uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretGetUUIDStringWrapper(virSecretPtr secret,
                              char * buf,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUUIDString not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetUUIDString(secret,
                                 buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virSecretGetUsageIDWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUsageID not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetUsageID(secret);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretGetUsageTypeWrapper(virSecretPtr secret,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUsageType not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetUsageType(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

unsigned char *
virSecretGetValueWrapper(virSecretPtr secret,
                         size_t * value_size,
                         unsigned int flags,
                         virErrorPtr err)
{
    unsigned char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetValue not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetValue(secret,
                            value_size,
                            flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virSecretGetXMLDescWrapper(virSecretPtr secret,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetXMLDesc not available prior to libvirt version 0.7.1");
#else
    ret = virSecretGetXMLDesc(secret,
                              flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virSecretPtr
virSecretLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    virSecretPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretLookupByUUID not available prior to libvirt version 0.7.1");
#else
    ret = virSecretLookupByUUID(conn,
                                uuid);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virSecretPtr
virSecretLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    virSecretPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretLookupByUUIDString not available prior to libvirt version 0.7.1");
#else
    ret = virSecretLookupByUUIDString(conn,
                                      uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virSecretPtr
virSecretLookupByUsageWrapper(virConnectPtr conn,
                              int usageType,
                              const char * usageID,
                              virErrorPtr err)
{
    virSecretPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretLookupByUsage not available prior to libvirt version 0.7.1");
#else
    ret = virSecretLookupByUsage(conn,
                                 usageType,
                                 usageID);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretRefWrapper(virSecretPtr secret,
                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretRef not available prior to libvirt version 0.7.1");
#else
    ret = virSecretRef(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretSetValueWrapper(virSecretPtr secret,
                         const unsigned char * value,
                         size_t value_size,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretSetValue not available prior to libvirt version 0.7.1");
#else
    ret = virSecretSetValue(secret,
                            value,
                            value_size,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virSecretUndefineWrapper(virSecretPtr secret,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretUndefine not available prior to libvirt version 0.7.1");
#else
    ret = virSecretUndefine(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
