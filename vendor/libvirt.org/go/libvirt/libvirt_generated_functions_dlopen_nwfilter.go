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
(*virConnectListAllNWFilterBindingsType)(virConnectPtr conn,
                                         virNWFilterBindingPtr ** bindings,
                                         unsigned int flags);

int
virConnectListAllNWFilterBindingsWrapper(virConnectPtr conn,
                                         virNWFilterBindingPtr ** bindings,
                                         unsigned int flags,
                                         virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllNWFilterBindingsType virConnectListAllNWFilterBindingsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllNWFilterBindings",
                       (void**)&virConnectListAllNWFilterBindingsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllNWFilterBindingsSymbol(conn,
                                                  bindings,
                                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListAllNWFiltersType)(virConnectPtr conn,
                                  virNWFilterPtr ** filters,
                                  unsigned int flags);

int
virConnectListAllNWFiltersWrapper(virConnectPtr conn,
                                  virNWFilterPtr ** filters,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllNWFiltersType virConnectListAllNWFiltersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllNWFilters",
                       (void**)&virConnectListAllNWFiltersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllNWFiltersSymbol(conn,
                                           filters,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListNWFiltersType)(virConnectPtr conn,
                               char ** const names,
                               int maxnames);

int
virConnectListNWFiltersWrapper(virConnectPtr conn,
                               char ** const names,
                               int maxnames,
                               virErrorPtr err)
{
    int ret = -1;
    static virConnectListNWFiltersType virConnectListNWFiltersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListNWFilters",
                       (void**)&virConnectListNWFiltersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListNWFiltersSymbol(conn,
                                        names,
                                        maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfNWFiltersType)(virConnectPtr conn);

int
virConnectNumOfNWFiltersWrapper(virConnectPtr conn,
                                virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfNWFiltersType virConnectNumOfNWFiltersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfNWFilters",
                       (void**)&virConnectNumOfNWFiltersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfNWFiltersSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterBindingPtr
(*virNWFilterBindingCreateXMLType)(virConnectPtr conn,
                                   const char * xml,
                                   unsigned int flags);

virNWFilterBindingPtr
virNWFilterBindingCreateXMLWrapper(virConnectPtr conn,
                                   const char * xml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    virNWFilterBindingPtr ret = NULL;
    static virNWFilterBindingCreateXMLType virNWFilterBindingCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingCreateXML",
                       (void**)&virNWFilterBindingCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingCreateXMLSymbol(conn,
                                            xml,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterBindingDeleteType)(virNWFilterBindingPtr binding);

int
virNWFilterBindingDeleteWrapper(virNWFilterBindingPtr binding,
                                virErrorPtr err)
{
    int ret = -1;
    static virNWFilterBindingDeleteType virNWFilterBindingDeleteSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingDelete",
                       (void**)&virNWFilterBindingDeleteSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingDeleteSymbol(binding);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterBindingFreeType)(virNWFilterBindingPtr binding);

int
virNWFilterBindingFreeWrapper(virNWFilterBindingPtr binding,
                              virErrorPtr err)
{
    int ret = -1;
    static virNWFilterBindingFreeType virNWFilterBindingFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingFree",
                       (void**)&virNWFilterBindingFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingFreeSymbol(binding);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNWFilterBindingGetFilterNameType)(virNWFilterBindingPtr binding);

const char *
virNWFilterBindingGetFilterNameWrapper(virNWFilterBindingPtr binding,
                                       virErrorPtr err)
{
    const char * ret = NULL;
    static virNWFilterBindingGetFilterNameType virNWFilterBindingGetFilterNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingGetFilterName",
                       (void**)&virNWFilterBindingGetFilterNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingGetFilterNameSymbol(binding);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNWFilterBindingGetPortDevType)(virNWFilterBindingPtr binding);

const char *
virNWFilterBindingGetPortDevWrapper(virNWFilterBindingPtr binding,
                                    virErrorPtr err)
{
    const char * ret = NULL;
    static virNWFilterBindingGetPortDevType virNWFilterBindingGetPortDevSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingGetPortDev",
                       (void**)&virNWFilterBindingGetPortDevSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingGetPortDevSymbol(binding);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNWFilterBindingGetXMLDescType)(virNWFilterBindingPtr binding,
                                    unsigned int flags);

char *
virNWFilterBindingGetXMLDescWrapper(virNWFilterBindingPtr binding,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    char * ret = NULL;
    static virNWFilterBindingGetXMLDescType virNWFilterBindingGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingGetXMLDesc",
                       (void**)&virNWFilterBindingGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingGetXMLDescSymbol(binding,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterBindingPtr
(*virNWFilterBindingLookupByPortDevType)(virConnectPtr conn,
                                         const char * portdev);

virNWFilterBindingPtr
virNWFilterBindingLookupByPortDevWrapper(virConnectPtr conn,
                                         const char * portdev,
                                         virErrorPtr err)
{
    virNWFilterBindingPtr ret = NULL;
    static virNWFilterBindingLookupByPortDevType virNWFilterBindingLookupByPortDevSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingLookupByPortDev",
                       (void**)&virNWFilterBindingLookupByPortDevSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingLookupByPortDevSymbol(conn,
                                                  portdev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterBindingRefType)(virNWFilterBindingPtr binding);

int
virNWFilterBindingRefWrapper(virNWFilterBindingPtr binding,
                             virErrorPtr err)
{
    int ret = -1;
    static virNWFilterBindingRefType virNWFilterBindingRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterBindingRef",
                       (void**)&virNWFilterBindingRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterBindingRefSymbol(binding);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterPtr
(*virNWFilterDefineXMLType)(virConnectPtr conn,
                            const char * xmlDesc);

virNWFilterPtr
virNWFilterDefineXMLWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
    static virNWFilterDefineXMLType virNWFilterDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterDefineXML",
                       (void**)&virNWFilterDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterDefineXMLSymbol(conn,
                                     xmlDesc);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterPtr
(*virNWFilterDefineXMLFlagsType)(virConnectPtr conn,
                                 const char * xmlDesc,
                                 unsigned int flags);

virNWFilterPtr
virNWFilterDefineXMLFlagsWrapper(virConnectPtr conn,
                                 const char * xmlDesc,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
    static virNWFilterDefineXMLFlagsType virNWFilterDefineXMLFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterDefineXMLFlags",
                       (void**)&virNWFilterDefineXMLFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterDefineXMLFlagsSymbol(conn,
                                          xmlDesc,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterFreeType)(virNWFilterPtr nwfilter);

int
virNWFilterFreeWrapper(virNWFilterPtr nwfilter,
                       virErrorPtr err)
{
    int ret = -1;
    static virNWFilterFreeType virNWFilterFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterFree",
                       (void**)&virNWFilterFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterFreeSymbol(nwfilter);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNWFilterGetNameType)(virNWFilterPtr nwfilter);

const char *
virNWFilterGetNameWrapper(virNWFilterPtr nwfilter,
                          virErrorPtr err)
{
    const char * ret = NULL;
    static virNWFilterGetNameType virNWFilterGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterGetName",
                       (void**)&virNWFilterGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterGetNameSymbol(nwfilter);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterGetUUIDType)(virNWFilterPtr nwfilter,
                          unsigned char * uuid);

int
virNWFilterGetUUIDWrapper(virNWFilterPtr nwfilter,
                          unsigned char * uuid,
                          virErrorPtr err)
{
    int ret = -1;
    static virNWFilterGetUUIDType virNWFilterGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterGetUUID",
                       (void**)&virNWFilterGetUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterGetUUIDSymbol(nwfilter,
                                   uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterGetUUIDStringType)(virNWFilterPtr nwfilter,
                                char * buf);

int
virNWFilterGetUUIDStringWrapper(virNWFilterPtr nwfilter,
                                char * buf,
                                virErrorPtr err)
{
    int ret = -1;
    static virNWFilterGetUUIDStringType virNWFilterGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterGetUUIDString",
                       (void**)&virNWFilterGetUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterGetUUIDStringSymbol(nwfilter,
                                         buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNWFilterGetXMLDescType)(virNWFilterPtr nwfilter,
                             unsigned int flags);

char *
virNWFilterGetXMLDescWrapper(virNWFilterPtr nwfilter,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = NULL;
    static virNWFilterGetXMLDescType virNWFilterGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterGetXMLDesc",
                       (void**)&virNWFilterGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterGetXMLDescSymbol(nwfilter,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterPtr
(*virNWFilterLookupByNameType)(virConnectPtr conn,
                               const char * name);

virNWFilterPtr
virNWFilterLookupByNameWrapper(virConnectPtr conn,
                               const char * name,
                               virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
    static virNWFilterLookupByNameType virNWFilterLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterLookupByName",
                       (void**)&virNWFilterLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterLookupByNameSymbol(conn,
                                        name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterPtr
(*virNWFilterLookupByUUIDType)(virConnectPtr conn,
                               const unsigned char * uuid);

virNWFilterPtr
virNWFilterLookupByUUIDWrapper(virConnectPtr conn,
                               const unsigned char * uuid,
                               virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
    static virNWFilterLookupByUUIDType virNWFilterLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterLookupByUUID",
                       (void**)&virNWFilterLookupByUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterLookupByUUIDSymbol(conn,
                                        uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNWFilterPtr
(*virNWFilterLookupByUUIDStringType)(virConnectPtr conn,
                                     const char * uuidstr);

virNWFilterPtr
virNWFilterLookupByUUIDStringWrapper(virConnectPtr conn,
                                     const char * uuidstr,
                                     virErrorPtr err)
{
    virNWFilterPtr ret = NULL;
    static virNWFilterLookupByUUIDStringType virNWFilterLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterLookupByUUIDString",
                       (void**)&virNWFilterLookupByUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterLookupByUUIDStringSymbol(conn,
                                              uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterRefType)(virNWFilterPtr nwfilter);

int
virNWFilterRefWrapper(virNWFilterPtr nwfilter,
                      virErrorPtr err)
{
    int ret = -1;
    static virNWFilterRefType virNWFilterRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterRef",
                       (void**)&virNWFilterRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterRefSymbol(nwfilter);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNWFilterUndefineType)(virNWFilterPtr nwfilter);

int
virNWFilterUndefineWrapper(virNWFilterPtr nwfilter,
                           virErrorPtr err)
{
    int ret = -1;
    static virNWFilterUndefineType virNWFilterUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNWFilterUndefine",
                       (void**)&virNWFilterUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNWFilterUndefineSymbol(nwfilter);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
