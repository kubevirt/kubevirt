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


typedef char *
(*virConnectBaselineCPUType)(virConnectPtr conn,
                             const char ** xmlCPUs,
                             unsigned int ncpus,
                             unsigned int flags);

char *
virConnectBaselineCPUWrapper(virConnectPtr conn,
                             const char ** xmlCPUs,
                             unsigned int ncpus,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = NULL;
    static virConnectBaselineCPUType virConnectBaselineCPUSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectBaselineCPU",
                       (void**)&virConnectBaselineCPUSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectBaselineCPUSymbol(conn,
                                      xmlCPUs,
                                      ncpus,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectBaselineHypervisorCPUType)(virConnectPtr conn,
                                       const char * emulator,
                                       const char * arch,
                                       const char * machine,
                                       const char * virttype,
                                       const char ** xmlCPUs,
                                       unsigned int ncpus,
                                       unsigned int flags);

char *
virConnectBaselineHypervisorCPUWrapper(virConnectPtr conn,
                                       const char * emulator,
                                       const char * arch,
                                       const char * machine,
                                       const char * virttype,
                                       const char ** xmlCPUs,
                                       unsigned int ncpus,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    char * ret = NULL;
    static virConnectBaselineHypervisorCPUType virConnectBaselineHypervisorCPUSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectBaselineHypervisorCPU",
                       (void**)&virConnectBaselineHypervisorCPUSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectBaselineHypervisorCPUSymbol(conn,
                                                emulator,
                                                arch,
                                                machine,
                                                virttype,
                                                xmlCPUs,
                                                ncpus,
                                                flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectCloseType)(virConnectPtr conn);

int
virConnectCloseWrapper(virConnectPtr conn,
                       virErrorPtr err)
{
    int ret = -1;
    static virConnectCloseType virConnectCloseSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectClose",
                       (void**)&virConnectCloseSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectCloseSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectCompareCPUType)(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags);

int
virConnectCompareCPUWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virConnectCompareCPUType virConnectCompareCPUSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectCompareCPU",
                       (void**)&virConnectCompareCPUSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectCompareCPUSymbol(conn,
                                     xmlDesc,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectCompareHypervisorCPUType)(virConnectPtr conn,
                                      const char * emulator,
                                      const char * arch,
                                      const char * machine,
                                      const char * virttype,
                                      const char * xmlCPU,
                                      unsigned int flags);

int
virConnectCompareHypervisorCPUWrapper(virConnectPtr conn,
                                      const char * emulator,
                                      const char * arch,
                                      const char * machine,
                                      const char * virttype,
                                      const char * xmlCPU,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virConnectCompareHypervisorCPUType virConnectCompareHypervisorCPUSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectCompareHypervisorCPU",
                       (void**)&virConnectCompareHypervisorCPUSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectCompareHypervisorCPUSymbol(conn,
                                               emulator,
                                               arch,
                                               machine,
                                               virttype,
                                               xmlCPU,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetCPUModelNamesType)(virConnectPtr conn,
                                  const char * arch,
                                  char *** models,
                                  unsigned int flags);

int
virConnectGetCPUModelNamesWrapper(virConnectPtr conn,
                                  const char * arch,
                                  char *** models,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virConnectGetCPUModelNamesType virConnectGetCPUModelNamesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetCPUModelNames",
                       (void**)&virConnectGetCPUModelNamesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetCPUModelNamesSymbol(conn,
                                           arch,
                                           models,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetCapabilitiesType)(virConnectPtr conn);

char *
virConnectGetCapabilitiesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    char * ret = NULL;
    static virConnectGetCapabilitiesType virConnectGetCapabilitiesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetCapabilities",
                       (void**)&virConnectGetCapabilitiesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetCapabilitiesSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetHostnameType)(virConnectPtr conn);

char *
virConnectGetHostnameWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    char * ret = NULL;
    static virConnectGetHostnameType virConnectGetHostnameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetHostname",
                       (void**)&virConnectGetHostnameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetHostnameSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetLibVersionType)(virConnectPtr conn,
                               unsigned long * libVer);

int
virConnectGetLibVersionWrapper(virConnectPtr conn,
                               unsigned long * libVer,
                               virErrorPtr err)
{
    int ret = -1;
    static virConnectGetLibVersionType virConnectGetLibVersionSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetLibVersion",
                       (void**)&virConnectGetLibVersionSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetLibVersionSymbol(conn,
                                        libVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetMaxVcpusType)(virConnectPtr conn,
                             const char * type);

int
virConnectGetMaxVcpusWrapper(virConnectPtr conn,
                             const char * type,
                             virErrorPtr err)
{
    int ret = -1;
    static virConnectGetMaxVcpusType virConnectGetMaxVcpusSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetMaxVcpus",
                       (void**)&virConnectGetMaxVcpusSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetMaxVcpusSymbol(conn,
                                      type);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetSysinfoType)(virConnectPtr conn,
                            unsigned int flags);

char *
virConnectGetSysinfoWrapper(virConnectPtr conn,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virConnectGetSysinfoType virConnectGetSysinfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetSysinfo",
                       (void**)&virConnectGetSysinfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetSysinfoSymbol(conn,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virConnectGetTypeType)(virConnectPtr conn);

const char *
virConnectGetTypeWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    const char * ret = NULL;
    static virConnectGetTypeType virConnectGetTypeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetType",
                       (void**)&virConnectGetTypeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetTypeSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetURIType)(virConnectPtr conn);

char *
virConnectGetURIWrapper(virConnectPtr conn,
                        virErrorPtr err)
{
    char * ret = NULL;
    static virConnectGetURIType virConnectGetURISymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetURI",
                       (void**)&virConnectGetURISymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetURISymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetVersionType)(virConnectPtr conn,
                            unsigned long * hvVer);

int
virConnectGetVersionWrapper(virConnectPtr conn,
                            unsigned long * hvVer,
                            virErrorPtr err)
{
    int ret = -1;
    static virConnectGetVersionType virConnectGetVersionSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetVersion",
                       (void**)&virConnectGetVersionSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetVersionSymbol(conn,
                                     hvVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectIsAliveType)(virConnectPtr conn);

int
virConnectIsAliveWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    int ret = -1;
    static virConnectIsAliveType virConnectIsAliveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectIsAlive",
                       (void**)&virConnectIsAliveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectIsAliveSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectIsEncryptedType)(virConnectPtr conn);

int
virConnectIsEncryptedWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    int ret = -1;
    static virConnectIsEncryptedType virConnectIsEncryptedSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectIsEncrypted",
                       (void**)&virConnectIsEncryptedSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectIsEncryptedSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectIsSecureType)(virConnectPtr conn);

int
virConnectIsSecureWrapper(virConnectPtr conn,
                          virErrorPtr err)
{
    int ret = -1;
    static virConnectIsSecureType virConnectIsSecureSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectIsSecure",
                       (void**)&virConnectIsSecureSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectIsSecureSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virConnectOpenType)(const char * name);

virConnectPtr
virConnectOpenWrapper(const char * name,
                      virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virConnectOpenType virConnectOpenSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectOpen",
                       (void**)&virConnectOpenSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectOpenSymbol(name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virConnectOpenAuthType)(const char * name,
                          virConnectAuthPtr auth,
                          unsigned int flags);

virConnectPtr
virConnectOpenAuthWrapper(const char * name,
                          virConnectAuthPtr auth,
                          unsigned int flags,
                          virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virConnectOpenAuthType virConnectOpenAuthSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectOpenAuth",
                       (void**)&virConnectOpenAuthSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectOpenAuthSymbol(name,
                                   auth,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virConnectOpenReadOnlyType)(const char * name);

virConnectPtr
virConnectOpenReadOnlyWrapper(const char * name,
                              virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virConnectOpenReadOnlyType virConnectOpenReadOnlySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectOpenReadOnly",
                       (void**)&virConnectOpenReadOnlySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectOpenReadOnlySymbol(name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectRefType)(virConnectPtr conn);

int
virConnectRefWrapper(virConnectPtr conn,
                     virErrorPtr err)
{
    int ret = -1;
    static virConnectRefType virConnectRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectRef",
                       (void**)&virConnectRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectRefSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectRegisterCloseCallbackType)(virConnectPtr conn,
                                       virConnectCloseFunc cb,
                                       void * opaque,
                                       virFreeCallback freecb);

int
virConnectRegisterCloseCallbackWrapper(virConnectPtr conn,
                                       virConnectCloseFunc cb,
                                       void * opaque,
                                       virFreeCallback freecb,
                                       virErrorPtr err)
{
    int ret = -1;
    static virConnectRegisterCloseCallbackType virConnectRegisterCloseCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectRegisterCloseCallback",
                       (void**)&virConnectRegisterCloseCallbackSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectRegisterCloseCallbackSymbol(conn,
                                                cb,
                                                opaque,
                                                freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectSetIdentityType)(virConnectPtr conn,
                             virTypedParameterPtr params,
                             int nparams,
                             unsigned int flags);

int
virConnectSetIdentityWrapper(virConnectPtr conn,
                             virTypedParameterPtr params,
                             int nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virConnectSetIdentityType virConnectSetIdentitySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectSetIdentity",
                       (void**)&virConnectSetIdentitySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectSetIdentitySymbol(conn,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectSetKeepAliveType)(virConnectPtr conn,
                              int interval,
                              unsigned int count);

int
virConnectSetKeepAliveWrapper(virConnectPtr conn,
                              int interval,
                              unsigned int count,
                              virErrorPtr err)
{
    int ret = -1;
    static virConnectSetKeepAliveType virConnectSetKeepAliveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectSetKeepAlive",
                       (void**)&virConnectSetKeepAliveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectSetKeepAliveSymbol(conn,
                                       interval,
                                       count);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectUnregisterCloseCallbackType)(virConnectPtr conn,
                                         virConnectCloseFunc cb);

int
virConnectUnregisterCloseCallbackWrapper(virConnectPtr conn,
                                         virConnectCloseFunc cb,
                                         virErrorPtr err)
{
    int ret = -1;
    static virConnectUnregisterCloseCallbackType virConnectUnregisterCloseCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectUnregisterCloseCallback",
                       (void**)&virConnectUnregisterCloseCallbackSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectUnregisterCloseCallbackSymbol(conn,
                                                  cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virGetVersionType)(unsigned long * libVer,
                     const char * type,
                     unsigned long * typeVer);

int
virGetVersionWrapper(unsigned long * libVer,
                     const char * type,
                     unsigned long * typeVer,
                     virErrorPtr err)
{
    int ret = -1;
    static virGetVersionType virGetVersionSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virGetVersion",
                       (void**)&virGetVersionSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virGetVersionSymbol(libVer,
                              type,
                              typeVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInitializeType)(void);

int
virInitializeWrapper(virErrorPtr err)
{
    int ret = -1;
    static virInitializeType virInitializeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virInitialize",
                       (void**)&virInitializeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virInitializeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeAllocPagesType)(virConnectPtr conn,
                         unsigned int npages,
                         unsigned int * pageSizes,
                         unsigned long long * pageCounts,
                         int startCell,
                         unsigned int cellCount,
                         unsigned int flags);

int
virNodeAllocPagesWrapper(virConnectPtr conn,
                         unsigned int npages,
                         unsigned int * pageSizes,
                         unsigned long long * pageCounts,
                         int startCell,
                         unsigned int cellCount,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virNodeAllocPagesType virNodeAllocPagesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeAllocPages",
                       (void**)&virNodeAllocPagesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeAllocPagesSymbol(conn,
                                  npages,
                                  pageSizes,
                                  pageCounts,
                                  startCell,
                                  cellCount,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetCPUMapType)(virConnectPtr conn,
                        unsigned char ** cpumap,
                        unsigned int * online,
                        unsigned int flags);

int
virNodeGetCPUMapWrapper(virConnectPtr conn,
                        unsigned char ** cpumap,
                        unsigned int * online,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
    static virNodeGetCPUMapType virNodeGetCPUMapSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetCPUMap",
                       (void**)&virNodeGetCPUMapSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetCPUMapSymbol(conn,
                                 cpumap,
                                 online,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetCPUStatsType)(virConnectPtr conn,
                          int cpuNum,
                          virNodeCPUStatsPtr params,
                          int * nparams,
                          unsigned int flags);

int
virNodeGetCPUStatsWrapper(virConnectPtr conn,
                          int cpuNum,
                          virNodeCPUStatsPtr params,
                          int * nparams,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virNodeGetCPUStatsType virNodeGetCPUStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetCPUStats",
                       (void**)&virNodeGetCPUStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetCPUStatsSymbol(conn,
                                   cpuNum,
                                   params,
                                   nparams,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetCellsFreeMemoryType)(virConnectPtr conn,
                                 unsigned long long * freeMems,
                                 int startCell,
                                 int maxCells);

int
virNodeGetCellsFreeMemoryWrapper(virConnectPtr conn,
                                 unsigned long long * freeMems,
                                 int startCell,
                                 int maxCells,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeGetCellsFreeMemoryType virNodeGetCellsFreeMemorySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetCellsFreeMemory",
                       (void**)&virNodeGetCellsFreeMemorySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetCellsFreeMemorySymbol(conn,
                                          freeMems,
                                          startCell,
                                          maxCells);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned long long
(*virNodeGetFreeMemoryType)(virConnectPtr conn);

unsigned long long
virNodeGetFreeMemoryWrapper(virConnectPtr conn,
                            virErrorPtr err)
{
    unsigned long long ret = 0;
    static virNodeGetFreeMemoryType virNodeGetFreeMemorySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetFreeMemory",
                       (void**)&virNodeGetFreeMemorySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetFreeMemorySymbol(conn);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetFreePagesType)(virConnectPtr conn,
                           unsigned int npages,
                           unsigned int * pages,
                           int startCell,
                           unsigned int cellCount,
                           unsigned long long * counts,
                           unsigned int flags);

int
virNodeGetFreePagesWrapper(virConnectPtr conn,
                           unsigned int npages,
                           unsigned int * pages,
                           int startCell,
                           unsigned int cellCount,
                           unsigned long long * counts,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virNodeGetFreePagesType virNodeGetFreePagesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetFreePages",
                       (void**)&virNodeGetFreePagesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetFreePagesSymbol(conn,
                                    npages,
                                    pages,
                                    startCell,
                                    cellCount,
                                    counts,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetInfoType)(virConnectPtr conn,
                      virNodeInfoPtr info);

int
virNodeGetInfoWrapper(virConnectPtr conn,
                      virNodeInfoPtr info,
                      virErrorPtr err)
{
    int ret = -1;
    static virNodeGetInfoType virNodeGetInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetInfo",
                       (void**)&virNodeGetInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetInfoSymbol(conn,
                               info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetMemoryParametersType)(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags);

int
virNodeGetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virNodeGetMemoryParametersType virNodeGetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetMemoryParameters",
                       (void**)&virNodeGetMemoryParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetMemoryParametersSymbol(conn,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetMemoryStatsType)(virConnectPtr conn,
                             int cellNum,
                             virNodeMemoryStatsPtr params,
                             int * nparams,
                             unsigned int flags);

int
virNodeGetMemoryStatsWrapper(virConnectPtr conn,
                             int cellNum,
                             virNodeMemoryStatsPtr params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virNodeGetMemoryStatsType virNodeGetMemoryStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetMemoryStats",
                       (void**)&virNodeGetMemoryStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetMemoryStatsSymbol(conn,
                                      cellNum,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetSEVInfoType)(virConnectPtr conn,
                         virTypedParameterPtr * params,
                         int * nparams,
                         unsigned int flags);

int
virNodeGetSEVInfoWrapper(virConnectPtr conn,
                         virTypedParameterPtr * params,
                         int * nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virNodeGetSEVInfoType virNodeGetSEVInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetSEVInfo",
                       (void**)&virNodeGetSEVInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetSEVInfoSymbol(conn,
                                  params,
                                  nparams,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeGetSecurityModelType)(virConnectPtr conn,
                               virSecurityModelPtr secmodel);

int
virNodeGetSecurityModelWrapper(virConnectPtr conn,
                               virSecurityModelPtr secmodel,
                               virErrorPtr err)
{
    int ret = -1;
    static virNodeGetSecurityModelType virNodeGetSecurityModelSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeGetSecurityModel",
                       (void**)&virNodeGetSecurityModelSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeGetSecurityModelSymbol(conn,
                                        secmodel);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeSetMemoryParametersType)(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags);

int
virNodeSetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virNodeSetMemoryParametersType virNodeSetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeSetMemoryParameters",
                       (void**)&virNodeSetMemoryParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeSetMemoryParametersSymbol(conn,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeSuspendForDurationType)(virConnectPtr conn,
                                 unsigned int target,
                                 unsigned long long duration,
                                 unsigned int flags);

int
virNodeSuspendForDurationWrapper(virConnectPtr conn,
                                 unsigned int target,
                                 unsigned long long duration,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virNodeSuspendForDurationType virNodeSuspendForDurationSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virNodeSuspendForDuration",
                       (void**)&virNodeSuspendForDurationSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virNodeSuspendForDurationSymbol(conn,
                                          target,
                                          duration,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
