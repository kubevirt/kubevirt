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


char *
virConnectBaselineCPUWrapper(virConnectPtr conn,
                             const char ** xmlCPUs,
                             unsigned int ncpus,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virConnectBaselineCPU not available prior to libvirt version 0.7.7");
#else
    ret = virConnectBaselineCPU(conn,
                                xmlCPUs,
                                ncpus,
                                flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(4, 4, 0)
    setVirError(err, "Function virConnectBaselineHypervisorCPU not available prior to libvirt version 4.4.0");
#else
    ret = virConnectBaselineHypervisorCPU(conn,
                                          emulator,
                                          arch,
                                          machine,
                                          virttype,
                                          xmlCPUs,
                                          ncpus,
                                          flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectCloseWrapper(virConnectPtr conn,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectClose not available prior to libvirt version 0.0.3");
#else
    ret = virConnectClose(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectCompareCPUWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 5)
    setVirError(err, "Function virConnectCompareCPU not available prior to libvirt version 0.7.5");
#else
    ret = virConnectCompareCPU(conn,
                               xmlDesc,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(4, 4, 0)
    setVirError(err, "Function virConnectCompareHypervisorCPU not available prior to libvirt version 4.4.0");
#else
    ret = virConnectCompareHypervisorCPU(conn,
                                         emulator,
                                         arch,
                                         machine,
                                         virttype,
                                         xmlCPU,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectGetCPUModelNamesWrapper(virConnectPtr conn,
                                  const char * arch,
                                  char *** models,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 1, 3)
    setVirError(err, "Function virConnectGetCPUModelNames not available prior to libvirt version 1.1.3");
#else
    ret = virConnectGetCPUModelNames(conn,
                                     arch,
                                     models,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectGetCapabilitiesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virConnectGetCapabilities not available prior to libvirt version 0.2.1");
#else
    ret = virConnectGetCapabilities(conn);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectGetHostnameWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virConnectGetHostname not available prior to libvirt version 0.3.0");
#else
    ret = virConnectGetHostname(conn);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectGetLibVersionWrapper(virConnectPtr conn,
                               unsigned long * libVer,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virConnectGetLibVersion not available prior to libvirt version 0.7.3");
#else
    ret = virConnectGetLibVersion(conn,
                                  libVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectGetMaxVcpusWrapper(virConnectPtr conn,
                             const char * type,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virConnectGetMaxVcpus not available prior to libvirt version 0.2.1");
#else
    ret = virConnectGetMaxVcpus(conn,
                                type);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectGetSysinfoWrapper(virConnectPtr conn,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 8, 8)
    setVirError(err, "Function virConnectGetSysinfo not available prior to libvirt version 0.8.8");
#else
    ret = virConnectGetSysinfo(conn,
                               flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virConnectGetTypeWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectGetType not available prior to libvirt version 0.0.3");
#else
    ret = virConnectGetType(conn);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectGetURIWrapper(virConnectPtr conn,
                        virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virConnectGetURI not available prior to libvirt version 0.3.0");
#else
    ret = virConnectGetURI(conn);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectGetVersionWrapper(virConnectPtr conn,
                            unsigned long * hvVer,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectGetVersion not available prior to libvirt version 0.0.3");
#else
    ret = virConnectGetVersion(conn,
                               hvVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectIsAliveWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virConnectIsAlive not available prior to libvirt version 0.9.8");
#else
    ret = virConnectIsAlive(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectIsEncryptedWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virConnectIsEncrypted not available prior to libvirt version 0.7.3");
#else
    ret = virConnectIsEncrypted(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectIsSecureWrapper(virConnectPtr conn,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virConnectIsSecure not available prior to libvirt version 0.7.3");
#else
    ret = virConnectIsSecure(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virConnectOpenWrapper(const char * name,
                      virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectOpen not available prior to libvirt version 0.0.3");
#else
    ret = virConnectOpen(name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virConnectOpenAuthWrapper(const char * name,
                          virConnectAuthPtr auth,
                          unsigned int flags,
                          virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 4, 0)
    setVirError(err, "Function virConnectOpenAuth not available prior to libvirt version 0.4.0");
#else
    ret = virConnectOpenAuth(name,
                             auth,
                             flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virConnectOpenReadOnlyWrapper(const char * name,
                              virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectOpenReadOnly not available prior to libvirt version 0.0.3");
#else
    ret = virConnectOpenReadOnly(name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectRefWrapper(virConnectPtr conn,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virConnectRef not available prior to libvirt version 0.6.0");
#else
    ret = virConnectRef(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectRegisterCloseCallbackWrapper(virConnectPtr conn,
                                       virConnectCloseFunc cb,
                                       void * opaque,
                                       virFreeCallback freecb,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virConnectRegisterCloseCallback not available prior to libvirt version 0.10.0");
#else
    ret = virConnectRegisterCloseCallback(conn,
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
virConnectSetIdentityWrapper(virConnectPtr conn,
                             virTypedParameterPtr params,
                             int nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 8, 0)
    setVirError(err, "Function virConnectSetIdentity not available prior to libvirt version 5.8.0");
#else
    ret = virConnectSetIdentity(conn,
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
virConnectSetKeepAliveWrapper(virConnectPtr conn,
                              int interval,
                              unsigned int count,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virConnectSetKeepAlive not available prior to libvirt version 0.9.8");
#else
    ret = virConnectSetKeepAlive(conn,
                                 interval,
                                 count);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectUnregisterCloseCallbackWrapper(virConnectPtr conn,
                                         virConnectCloseFunc cb,
                                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virConnectUnregisterCloseCallback not available prior to libvirt version 0.10.0");
#else
    ret = virConnectUnregisterCloseCallback(conn,
                                            cb);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virGetVersionWrapper(unsigned long * libVer,
                     const char * type,
                     unsigned long * typeVer,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virGetVersion not available prior to libvirt version 0.0.3");
#else
    ret = virGetVersion(libVer,
                        type,
                        typeVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virInitializeWrapper(virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virInitialize not available prior to libvirt version 0.1.0");
#else
    ret = virInitialize();
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(1, 2, 9)
    setVirError(err, "Function virNodeAllocPages not available prior to libvirt version 1.2.9");
#else
    ret = virNodeAllocPages(conn,
                            npages,
                            pageSizes,
                            pageCounts,
                            startCell,
                            cellCount,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeGetCPUMapWrapper(virConnectPtr conn,
                        unsigned char ** cpumap,
                        unsigned int * online,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 0)
    setVirError(err, "Function virNodeGetCPUMap not available prior to libvirt version 1.0.0");
#else
    ret = virNodeGetCPUMap(conn,
                           cpumap,
                           online,
                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeGetCPUStatsWrapper(virConnectPtr conn,
                          int cpuNum,
                          virNodeCPUStatsPtr params,
                          int * nparams,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virNodeGetCPUStats not available prior to libvirt version 0.9.3");
#else
    ret = virNodeGetCPUStats(conn,
                             cpuNum,
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
virNodeGetCellsFreeMemoryWrapper(virConnectPtr conn,
                                 unsigned long long * freeMems,
                                 int startCell,
                                 int maxCells,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 3, 3)
    setVirError(err, "Function virNodeGetCellsFreeMemory not available prior to libvirt version 0.3.3");
#else
    ret = virNodeGetCellsFreeMemory(conn,
                                    freeMems,
                                    startCell,
                                    maxCells);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

unsigned long long
virNodeGetFreeMemoryWrapper(virConnectPtr conn,
                            virErrorPtr err)
{
    unsigned long long ret = 0;
#if !LIBVIR_CHECK_VERSION(0, 3, 3)
    setVirError(err, "Function virNodeGetFreeMemory not available prior to libvirt version 0.3.3");
#else
    ret = virNodeGetFreeMemory(conn);
    if (ret == 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(1, 2, 6)
    setVirError(err, "Function virNodeGetFreePages not available prior to libvirt version 1.2.6");
#else
    ret = virNodeGetFreePages(conn,
                              npages,
                              pages,
                              startCell,
                              cellCount,
                              counts,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeGetInfoWrapper(virConnectPtr conn,
                      virNodeInfoPtr info,
                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virNodeGetInfo not available prior to libvirt version 0.1.0");
#else
    ret = virNodeGetInfo(conn,
                         info);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeGetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virNodeGetMemoryParameters not available prior to libvirt version 0.10.2");
#else
    ret = virNodeGetMemoryParameters(conn,
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
virNodeGetMemoryStatsWrapper(virConnectPtr conn,
                             int cellNum,
                             virNodeMemoryStatsPtr params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virNodeGetMemoryStats not available prior to libvirt version 0.9.3");
#else
    ret = virNodeGetMemoryStats(conn,
                                cellNum,
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
virNodeGetSEVInfoWrapper(virConnectPtr conn,
                         virTypedParameterPtr * params,
                         int * nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNodeGetSEVInfo not available prior to libvirt version 4.5.0");
#else
    ret = virNodeGetSEVInfo(conn,
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
virNodeGetSecurityModelWrapper(virConnectPtr conn,
                               virSecurityModelPtr secmodel,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeGetSecurityModel not available prior to libvirt version 0.6.1");
#else
    ret = virNodeGetSecurityModel(conn,
                                  secmodel);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virNodeSetMemoryParametersWrapper(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virNodeSetMemoryParameters not available prior to libvirt version 0.10.2");
#else
    ret = virNodeSetMemoryParameters(conn,
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
virNodeSuspendForDurationWrapper(virConnectPtr conn,
                                 unsigned int target,
                                 unsigned long long duration,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virNodeSuspendForDuration not available prior to libvirt version 0.9.8");
#else
    ret = virNodeSuspendForDuration(conn,
                                    target,
                                    duration,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
