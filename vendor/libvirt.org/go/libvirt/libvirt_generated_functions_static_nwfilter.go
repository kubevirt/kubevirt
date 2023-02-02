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
virConnectListAllNWFilterBindingsWrapper(virConnectPtr conn,
                                         virNWFilterBindingPtr ** bindings,
                                         unsigned int flags,
                                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virConnectListAllNWFilterBindings not available prior to libvirt version 4.5.0");
#else
    ret = virConnectListAllNWFilterBindings(conn,
                                            bindings,
                                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListAllNWFiltersWrapper(virConnectPtr conn,
                                  virNWFilterPtr ** filters,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllNWFilters not available prior to libvirt version 0.10.2");
#else
    ret = virConnectListAllNWFilters(conn,
                                     filters,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListNWFiltersWrapper(virConnectPtr conn,
                               char ** const names,
                               int maxnames,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectListNWFilters not available prior to libvirt version 0.8.0");
#else
    ret = virConnectListNWFilters(conn,
                                  names,
                                  maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfNWFiltersWrapper(virConnectPtr conn,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectNumOfNWFilters not available prior to libvirt version 0.8.0");
#else
    ret = virConnectNumOfNWFilters(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterBindingPtr
virNWFilterBindingCreateXMLWrapper(virConnectPtr conn,
                                   const char * xml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    virNWFilterBindingPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingCreateXML not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingCreateXML(conn,
                                      xml,
                                      flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterBindingDeleteWrapper(virNWFilterBindingPtr binding,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingDelete not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingDelete(binding);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterBindingFreeWrapper(virNWFilterBindingPtr binding,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingFree not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingFree(binding);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virNWFilterBindingGetFilterNameWrapper(virNWFilterBindingPtr binding,
                                       virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingGetFilterName not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingGetFilterName(binding);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virNWFilterBindingGetPortDevWrapper(virNWFilterBindingPtr binding,
                                    virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingGetPortDev not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingGetPortDev(binding);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNWFilterBindingGetXMLDescWrapper(virNWFilterBindingPtr binding,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingGetXMLDesc not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingGetXMLDesc(binding,
                                       flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterBindingPtr
virNWFilterBindingLookupByPortDevWrapper(virConnectPtr conn,
                                         const char * portdev,
                                         virErrorPtr err)
{
    virNWFilterBindingPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingLookupByPortDev not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingLookupByPortDev(conn,
                                            portdev);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterBindingRefWrapper(virNWFilterBindingPtr binding,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingRef not available prior to libvirt version 4.5.0");
#else
    ret = virNWFilterBindingRef(binding);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterPtr
virNWFilterDefineXMLWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterDefineXML not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterDefineXML(conn,
                               xmlDesc);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterPtr
virNWFilterDefineXMLFlagsWrapper(virConnectPtr conn,
                                 const char * xmlDesc,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(7, 7, 0)
    setVirError(err, "Function virNWFilterDefineXMLFlags not available prior to libvirt version 7.7.0");
#else
    ret = virNWFilterDefineXMLFlags(conn,
                                    xmlDesc,
                                    flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterFreeWrapper(virNWFilterPtr nwfilter,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterFree not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterFree(nwfilter);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virNWFilterGetNameWrapper(virNWFilterPtr nwfilter,
                          virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetName not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterGetName(nwfilter);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterGetUUIDWrapper(virNWFilterPtr nwfilter,
                          unsigned char * uuid,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetUUID not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterGetUUID(nwfilter,
                             uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterGetUUIDStringWrapper(virNWFilterPtr nwfilter,
                                char * buf,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetUUIDString not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterGetUUIDString(nwfilter,
                                   buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virNWFilterGetXMLDescWrapper(virNWFilterPtr nwfilter,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetXMLDesc not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterGetXMLDesc(nwfilter,
                                flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterPtr
virNWFilterLookupByNameWrapper(virConnectPtr conn,
                               const char * name,
                               virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterLookupByName not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterLookupByName(conn,
                                  name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterPtr
virNWFilterLookupByUUIDWrapper(virConnectPtr conn,
                               const unsigned char * uuid,
                               virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterLookupByUUID not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterLookupByUUID(conn,
                                  uuid);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virNWFilterPtr
virNWFilterLookupByUUIDStringWrapper(virConnectPtr conn,
                                     const char * uuidstr,
                                     virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterLookupByUUIDString not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterLookupByUUIDString(conn,
                                        uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterRefWrapper(virNWFilterPtr nwfilter,
                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterRef not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterRef(nwfilter);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNWFilterUndefineWrapper(virNWFilterPtr nwfilter,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterUndefine not available prior to libvirt version 0.8.0");
#else
    ret = virNWFilterUndefine(nwfilter);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
