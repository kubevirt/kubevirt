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
 * Copyright (C) 2021 Red Hat, Inc.
 *
 */

/****************************************************************************
 * THIS CODE HAS BEEN GENERATED. DO NOT CHANGE IT DIRECTLY                  *
 ****************************************************************************/

#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <dlfcn.h>
#include "module-generated.h"

/* dlopen's handlers */
static void *libvirt;
static void *qemu;
static void *lxc;

/* Exported variables */
virConnectAuthPtr virConnectAuthPtrDefaultVar;

static void *
libvirtSymbol(void *handle, const char *name, bool *success)
{
    void *symbol = NULL;
    bool ok = true;
    if (handle == NULL) {
        ok = false;
        goto end_symbol;
    }

    /* Documentation of dlsym says we should use dlerror() to check for failure
     * in dlsym() as a NULL might be the right address for a given symbol. This
     * is also the reason form @success argument.
     */
    symbol = dlsym(handle, name);
    char *err = dlerror();
    if (err != NULL) {
        ok = false;
        fprintf(stderr, "dlsym %s err: %s\n", err);
    }

end_symbol:
    if (success) {
        *success = ok;
    }
    return symbol;
}

static void
libvirtLoadLibvirtVariables(void)
{
    assert(libvirt != NULL);
    virConnectAuthPtrDefaultVar = libvirtSymbol(libvirt, "virConnectAuthPtrDefault", NULL);
}

static void
libvirtLoadOnce(void)
{
    static bool once;
    if (once) {
        return;
    }
    once = true;

    /* Note that we need to use soname */
    libvirt = dlopen("libvirt.so.0", RTLD_NOW|RTLD_LOCAL);
    if (libvirt == NULL) {
        fprintf(stderr, "dlopen libvirt.so.0 err: %s\n", dlerror());
        return;
    }
    libvirtLoadLibvirtVariables();

    /* The application might not need libvirt-qemu nor libvirt-lxc libraries so
     * we don't treat it as an error here if we can't load them, only when
     * trying to load symbols from * those libraries
     */
    qemu = dlopen("libvirt-qemu.so.0", RTLD_NOW|RTLD_LOCAL);
    lxc = dlopen("libvirt-lxc.so.0", RTLD_NOW|RTLD_LOCAL);
}
typedef int
(*virCopyLastErrorType)(virErrorPtr to);

int
virCopyLastErrorWrapper(virErrorPtr to) {
    static virCopyLastErrorType virCopyLastErrorSymbol;
    static bool once;
    static bool success;
    if (!once) {
        once = true;
        libvirtLoadOnce();
        virCopyLastErrorSymbol = libvirtSymbol(libvirt, "virCopyLastError", &success);
    }
    if (!success) {
        return -1;
    }
    return virCopyLastErrorSymbol(to);
}
typedef int
(*virConnCopyLastErrorType)(virConnectPtr conn,
                            virErrorPtr to);

int
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err)
{
    static virConnCopyLastErrorType virConnCopyLastErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnCopyLastErrorSymbol = libvirtSymbol(libvirt, "virConnCopyLastError", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnCopyLastError",
                libvirt);
        return ret;
    }

    ret = virConnCopyLastErrorSymbol(conn,
                                     to);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virErrorPtr
(*virConnGetLastErrorType)(virConnectPtr conn);

virErrorPtr
virConnGetLastErrorWrapper(virConnectPtr conn,
                           virErrorPtr err)
{
    static virConnGetLastErrorType virConnGetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnGetLastErrorSymbol = libvirtSymbol(libvirt, "virConnGetLastError", &success);
    }

    virErrorPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnGetLastError",
                libvirt);
        return ret;
    }

    ret = virConnGetLastErrorSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virConnResetLastErrorType)(virConnectPtr conn);

void
virConnResetLastErrorWrapper(virConnectPtr conn)
{
    static virConnResetLastErrorType virConnResetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnResetLastErrorSymbol = libvirtSymbol(libvirt, "virConnResetLastError", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virConnResetLastError",
                libvirt);
        return;
    }

    virConnResetLastErrorSymbol(conn);
}

typedef void
(*virConnSetErrorFuncType)(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler);

void
virConnSetErrorFuncWrapper(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler)
{
    static virConnSetErrorFuncType virConnSetErrorFuncSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnSetErrorFuncSymbol = libvirtSymbol(libvirt, "virConnSetErrorFunc", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virConnSetErrorFunc",
                libvirt);
        return;
    }

    virConnSetErrorFuncSymbol(conn,
                              userData,
                              handler);
}

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
    static virConnectBaselineCPUType virConnectBaselineCPUSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectBaselineCPUSymbol = libvirtSymbol(libvirt, "virConnectBaselineCPU", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectBaselineCPU",
                libvirt);
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
    static virConnectBaselineHypervisorCPUType virConnectBaselineHypervisorCPUSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectBaselineHypervisorCPUSymbol = libvirtSymbol(libvirt, "virConnectBaselineHypervisorCPU", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectBaselineHypervisorCPU",
                libvirt);
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
    static virConnectCloseType virConnectCloseSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectCloseSymbol = libvirtSymbol(libvirt, "virConnectClose", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectClose",
                libvirt);
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
    static virConnectCompareCPUType virConnectCompareCPUSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectCompareCPUSymbol = libvirtSymbol(libvirt, "virConnectCompareCPU", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectCompareCPU",
                libvirt);
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
    static virConnectCompareHypervisorCPUType virConnectCompareHypervisorCPUSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectCompareHypervisorCPUSymbol = libvirtSymbol(libvirt, "virConnectCompareHypervisorCPU", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectCompareHypervisorCPU",
                libvirt);
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
(*virConnectDomainEventDeregisterType)(virConnectPtr conn,
                                       virConnectDomainEventCallback cb);

int
virConnectDomainEventDeregisterWrapper(virConnectPtr conn,
                                       virConnectDomainEventCallback cb,
                                       virErrorPtr err)
{
    static virConnectDomainEventDeregisterType virConnectDomainEventDeregisterSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainEventDeregisterSymbol = libvirtSymbol(libvirt, "virConnectDomainEventDeregister", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainEventDeregister",
                libvirt);
        return ret;
    }

    ret = virConnectDomainEventDeregisterSymbol(conn,
                                                cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainEventDeregisterAnyType)(virConnectPtr conn,
                                          int callbackID);

int
virConnectDomainEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    static virConnectDomainEventDeregisterAnyType virConnectDomainEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectDomainEventDeregisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainEventDeregisterAny",
                libvirt);
        return ret;
    }

    ret = virConnectDomainEventDeregisterAnySymbol(conn,
                                                   callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainEventRegisterType)(virConnectPtr conn,
                                     virConnectDomainEventCallback cb,
                                     void * opaque,
                                     virFreeCallback freecb);

int
virConnectDomainEventRegisterWrapper(virConnectPtr conn,
                                     virConnectDomainEventCallback cb,
                                     void * opaque,
                                     virFreeCallback freecb,
                                     virErrorPtr err)
{
    static virConnectDomainEventRegisterType virConnectDomainEventRegisterSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainEventRegisterSymbol = libvirtSymbol(libvirt, "virConnectDomainEventRegister", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainEventRegister",
                libvirt);
        return ret;
    }

    ret = virConnectDomainEventRegisterSymbol(conn,
                                              cb,
                                              opaque,
                                              freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainEventRegisterAnyType)(virConnectPtr conn,
                                        virDomainPtr dom,
                                        int eventID,
                                        virConnectDomainEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb);

int
virConnectDomainEventRegisterAnyWrapper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        int eventID,
                                        virConnectDomainEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb,
                                        virErrorPtr err)
{
    static virConnectDomainEventRegisterAnyType virConnectDomainEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectDomainEventRegisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainEventRegisterAny",
                libvirt);
        return ret;
    }

    ret = virConnectDomainEventRegisterAnySymbol(conn,
                                                 dom,
                                                 eventID,
                                                 cb,
                                                 opaque,
                                                 freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectDomainXMLFromNativeType)(virConnectPtr conn,
                                     const char * nativeFormat,
                                     const char * nativeConfig,
                                     unsigned int flags);

char *
virConnectDomainXMLFromNativeWrapper(virConnectPtr conn,
                                     const char * nativeFormat,
                                     const char * nativeConfig,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virConnectDomainXMLFromNativeType virConnectDomainXMLFromNativeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainXMLFromNativeSymbol = libvirtSymbol(libvirt, "virConnectDomainXMLFromNative", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainXMLFromNative",
                libvirt);
        return ret;
    }

    ret = virConnectDomainXMLFromNativeSymbol(conn,
                                              nativeFormat,
                                              nativeConfig,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectDomainXMLToNativeType)(virConnectPtr conn,
                                   const char * nativeFormat,
                                   const char * domainXml,
                                   unsigned int flags);

char *
virConnectDomainXMLToNativeWrapper(virConnectPtr conn,
                                   const char * nativeFormat,
                                   const char * domainXml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virConnectDomainXMLToNativeType virConnectDomainXMLToNativeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainXMLToNativeSymbol = libvirtSymbol(libvirt, "virConnectDomainXMLToNative", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainXMLToNative",
                libvirt);
        return ret;
    }

    ret = virConnectDomainXMLToNativeSymbol(conn,
                                            nativeFormat,
                                            domainXml,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectFindStoragePoolSourcesType)(virConnectPtr conn,
                                        const char * type,
                                        const char * srcSpec,
                                        unsigned int flags);

char *
virConnectFindStoragePoolSourcesWrapper(virConnectPtr conn,
                                        const char * type,
                                        const char * srcSpec,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    static virConnectFindStoragePoolSourcesType virConnectFindStoragePoolSourcesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectFindStoragePoolSourcesSymbol = libvirtSymbol(libvirt, "virConnectFindStoragePoolSources", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectFindStoragePoolSources",
                libvirt);
        return ret;
    }

    ret = virConnectFindStoragePoolSourcesSymbol(conn,
                                                 type,
                                                 srcSpec,
                                                 flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetAllDomainStatsType)(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr ** retStats,
                                   unsigned int flags);

int
virConnectGetAllDomainStatsWrapper(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr ** retStats,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virConnectGetAllDomainStatsType virConnectGetAllDomainStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetAllDomainStatsSymbol = libvirtSymbol(libvirt, "virConnectGetAllDomainStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetAllDomainStats",
                libvirt);
        return ret;
    }

    ret = virConnectGetAllDomainStatsSymbol(conn,
                                            stats,
                                            retStats,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetCPUModelNamesType)(virConnectPtr conn,
                                  const char * arch,
                                  char ** * models,
                                  unsigned int flags);

int
virConnectGetCPUModelNamesWrapper(virConnectPtr conn,
                                  const char * arch,
                                  char ** * models,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virConnectGetCPUModelNamesType virConnectGetCPUModelNamesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetCPUModelNamesSymbol = libvirtSymbol(libvirt, "virConnectGetCPUModelNames", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetCPUModelNames",
                libvirt);
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
    static virConnectGetCapabilitiesType virConnectGetCapabilitiesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetCapabilitiesSymbol = libvirtSymbol(libvirt, "virConnectGetCapabilities", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetCapabilities",
                libvirt);
        return ret;
    }

    ret = virConnectGetCapabilitiesSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetDomainCapabilitiesType)(virConnectPtr conn,
                                       const char * emulatorbin,
                                       const char * arch,
                                       const char * machine,
                                       const char * virttype,
                                       unsigned int flags);

char *
virConnectGetDomainCapabilitiesWrapper(virConnectPtr conn,
                                       const char * emulatorbin,
                                       const char * arch,
                                       const char * machine,
                                       const char * virttype,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    static virConnectGetDomainCapabilitiesType virConnectGetDomainCapabilitiesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetDomainCapabilitiesSymbol = libvirtSymbol(libvirt, "virConnectGetDomainCapabilities", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetDomainCapabilities",
                libvirt);
        return ret;
    }

    ret = virConnectGetDomainCapabilitiesSymbol(conn,
                                                emulatorbin,
                                                arch,
                                                machine,
                                                virttype,
                                                flags);
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
    static virConnectGetHostnameType virConnectGetHostnameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetHostnameSymbol = libvirtSymbol(libvirt, "virConnectGetHostname", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetHostname",
                libvirt);
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
    static virConnectGetLibVersionType virConnectGetLibVersionSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetLibVersionSymbol = libvirtSymbol(libvirt, "virConnectGetLibVersion", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetLibVersion",
                libvirt);
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
    static virConnectGetMaxVcpusType virConnectGetMaxVcpusSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetMaxVcpusSymbol = libvirtSymbol(libvirt, "virConnectGetMaxVcpus", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetMaxVcpus",
                libvirt);
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
(*virConnectGetStoragePoolCapabilitiesType)(virConnectPtr conn,
                                            unsigned int flags);

char *
virConnectGetStoragePoolCapabilitiesWrapper(virConnectPtr conn,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    static virConnectGetStoragePoolCapabilitiesType virConnectGetStoragePoolCapabilitiesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetStoragePoolCapabilitiesSymbol = libvirtSymbol(libvirt, "virConnectGetStoragePoolCapabilities", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetStoragePoolCapabilities",
                libvirt);
        return ret;
    }

    ret = virConnectGetStoragePoolCapabilitiesSymbol(conn,
                                                     flags);
    if (!ret) {
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
    static virConnectGetSysinfoType virConnectGetSysinfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetSysinfoSymbol = libvirtSymbol(libvirt, "virConnectGetSysinfo", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetSysinfo",
                libvirt);
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
    static virConnectGetTypeType virConnectGetTypeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetTypeSymbol = libvirtSymbol(libvirt, "virConnectGetType", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetType",
                libvirt);
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
    static virConnectGetURIType virConnectGetURISymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetURISymbol = libvirtSymbol(libvirt, "virConnectGetURI", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetURI",
                libvirt);
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
    static virConnectGetVersionType virConnectGetVersionSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectGetVersionSymbol = libvirtSymbol(libvirt, "virConnectGetVersion", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectGetVersion",
                libvirt);
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
    static virConnectIsAliveType virConnectIsAliveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectIsAliveSymbol = libvirtSymbol(libvirt, "virConnectIsAlive", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectIsAlive",
                libvirt);
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
    static virConnectIsEncryptedType virConnectIsEncryptedSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectIsEncryptedSymbol = libvirtSymbol(libvirt, "virConnectIsEncrypted", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectIsEncrypted",
                libvirt);
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
    static virConnectIsSecureType virConnectIsSecureSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectIsSecureSymbol = libvirtSymbol(libvirt, "virConnectIsSecure", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectIsSecure",
                libvirt);
        return ret;
    }

    ret = virConnectIsSecureSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListAllDomainsType)(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags);

int
virConnectListAllDomainsWrapper(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virConnectListAllDomainsType virConnectListAllDomainsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllDomainsSymbol = libvirtSymbol(libvirt, "virConnectListAllDomains", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllDomains",
                libvirt);
        return ret;
    }

    ret = virConnectListAllDomainsSymbol(conn,
                                         domains,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListAllInterfacesType)(virConnectPtr conn,
                                   virInterfacePtr ** ifaces,
                                   unsigned int flags);

int
virConnectListAllInterfacesWrapper(virConnectPtr conn,
                                   virInterfacePtr ** ifaces,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virConnectListAllInterfacesType virConnectListAllInterfacesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllInterfacesSymbol = libvirtSymbol(libvirt, "virConnectListAllInterfaces", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllInterfaces",
                libvirt);
        return ret;
    }

    ret = virConnectListAllInterfacesSymbol(conn,
                                            ifaces,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

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
    static virConnectListAllNWFilterBindingsType virConnectListAllNWFilterBindingsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllNWFilterBindingsSymbol = libvirtSymbol(libvirt, "virConnectListAllNWFilterBindings", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllNWFilterBindings",
                libvirt);
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
    static virConnectListAllNWFiltersType virConnectListAllNWFiltersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllNWFiltersSymbol = libvirtSymbol(libvirt, "virConnectListAllNWFilters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllNWFilters",
                libvirt);
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
(*virConnectListAllNetworksType)(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags);

int
virConnectListAllNetworksWrapper(virConnectPtr conn,
                                 virNetworkPtr ** nets,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virConnectListAllNetworksType virConnectListAllNetworksSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllNetworksSymbol = libvirtSymbol(libvirt, "virConnectListAllNetworks", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllNetworks",
                libvirt);
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
(*virConnectListAllNodeDevicesType)(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags);

int
virConnectListAllNodeDevicesWrapper(virConnectPtr conn,
                                    virNodeDevicePtr ** devices,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virConnectListAllNodeDevicesType virConnectListAllNodeDevicesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllNodeDevicesSymbol = libvirtSymbol(libvirt, "virConnectListAllNodeDevices", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllNodeDevices",
                libvirt);
        return ret;
    }

    ret = virConnectListAllNodeDevicesSymbol(conn,
                                             devices,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

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
    static virConnectListAllSecretsType virConnectListAllSecretsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllSecretsSymbol = libvirtSymbol(libvirt, "virConnectListAllSecrets", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllSecrets",
                libvirt);
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
(*virConnectListAllStoragePoolsType)(virConnectPtr conn,
                                     virStoragePoolPtr ** pools,
                                     unsigned int flags);

int
virConnectListAllStoragePoolsWrapper(virConnectPtr conn,
                                     virStoragePoolPtr ** pools,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virConnectListAllStoragePoolsType virConnectListAllStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListAllStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectListAllStoragePools", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListAllStoragePools",
                libvirt);
        return ret;
    }

    ret = virConnectListAllStoragePoolsSymbol(conn,
                                              pools,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDefinedDomainsType)(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames);

int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames,
                                    virErrorPtr err)
{
    static virConnectListDefinedDomainsType virConnectListDefinedDomainsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListDefinedDomainsSymbol = libvirtSymbol(libvirt, "virConnectListDefinedDomains", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListDefinedDomains",
                libvirt);
        return ret;
    }

    ret = virConnectListDefinedDomainsSymbol(conn,
                                             names,
                                             maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDefinedInterfacesType)(virConnectPtr conn,
                                       char ** const names,
                                       int maxnames);

int
virConnectListDefinedInterfacesWrapper(virConnectPtr conn,
                                       char ** const names,
                                       int maxnames,
                                       virErrorPtr err)
{
    static virConnectListDefinedInterfacesType virConnectListDefinedInterfacesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListDefinedInterfacesSymbol = libvirtSymbol(libvirt, "virConnectListDefinedInterfaces", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListDefinedInterfaces",
                libvirt);
        return ret;
    }

    ret = virConnectListDefinedInterfacesSymbol(conn,
                                                names,
                                                maxnames);
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
    static virConnectListDefinedNetworksType virConnectListDefinedNetworksSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListDefinedNetworksSymbol = libvirtSymbol(libvirt, "virConnectListDefinedNetworks", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListDefinedNetworks",
                libvirt);
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
(*virConnectListDefinedStoragePoolsType)(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames);

int
virConnectListDefinedStoragePoolsWrapper(virConnectPtr conn,
                                         char ** const names,
                                         int maxnames,
                                         virErrorPtr err)
{
    static virConnectListDefinedStoragePoolsType virConnectListDefinedStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListDefinedStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectListDefinedStoragePools", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListDefinedStoragePools",
                libvirt);
        return ret;
    }

    ret = virConnectListDefinedStoragePoolsSymbol(conn,
                                                  names,
                                                  maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDomainsType)(virConnectPtr conn,
                             int * ids,
                             int maxids);

int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int * ids,
                             int maxids,
                             virErrorPtr err)
{
    static virConnectListDomainsType virConnectListDomainsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListDomainsSymbol = libvirtSymbol(libvirt, "virConnectListDomains", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListDomains",
                libvirt);
        return ret;
    }

    ret = virConnectListDomainsSymbol(conn,
                                      ids,
                                      maxids);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListInterfacesType)(virConnectPtr conn,
                                char ** const names,
                                int maxnames);

int
virConnectListInterfacesWrapper(virConnectPtr conn,
                                char ** const names,
                                int maxnames,
                                virErrorPtr err)
{
    static virConnectListInterfacesType virConnectListInterfacesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListInterfacesSymbol = libvirtSymbol(libvirt, "virConnectListInterfaces", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListInterfaces",
                libvirt);
        return ret;
    }

    ret = virConnectListInterfacesSymbol(conn,
                                         names,
                                         maxnames);
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
    static virConnectListNWFiltersType virConnectListNWFiltersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListNWFiltersSymbol = libvirtSymbol(libvirt, "virConnectListNWFilters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListNWFilters",
                libvirt);
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
(*virConnectListNetworksType)(virConnectPtr conn,
                              char ** const names,
                              int maxnames);

int
virConnectListNetworksWrapper(virConnectPtr conn,
                              char ** const names,
                              int maxnames,
                              virErrorPtr err)
{
    static virConnectListNetworksType virConnectListNetworksSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListNetworksSymbol = libvirtSymbol(libvirt, "virConnectListNetworks", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListNetworks",
                libvirt);
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
(*virConnectListSecretsType)(virConnectPtr conn,
                             char ** uuids,
                             int maxuuids);

int
virConnectListSecretsWrapper(virConnectPtr conn,
                             char ** uuids,
                             int maxuuids,
                             virErrorPtr err)
{
    static virConnectListSecretsType virConnectListSecretsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListSecretsSymbol = libvirtSymbol(libvirt, "virConnectListSecrets", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListSecrets",
                libvirt);
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
(*virConnectListStoragePoolsType)(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames);

int
virConnectListStoragePoolsWrapper(virConnectPtr conn,
                                  char ** const names,
                                  int maxnames,
                                  virErrorPtr err)
{
    static virConnectListStoragePoolsType virConnectListStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectListStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectListStoragePools", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectListStoragePools",
                libvirt);
        return ret;
    }

    ret = virConnectListStoragePoolsSymbol(conn,
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
    static virConnectNetworkEventDeregisterAnyType virConnectNetworkEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNetworkEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectNetworkEventDeregisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNetworkEventDeregisterAny",
                libvirt);
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
    static virConnectNetworkEventRegisterAnyType virConnectNetworkEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNetworkEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectNetworkEventRegisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNetworkEventRegisterAny",
                libvirt);
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
(*virConnectNodeDeviceEventDeregisterAnyType)(virConnectPtr conn,
                                              int callbackID);

int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err)
{
    static virConnectNodeDeviceEventDeregisterAnyType virConnectNodeDeviceEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNodeDeviceEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectNodeDeviceEventDeregisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNodeDeviceEventDeregisterAny",
                libvirt);
        return ret;
    }

    ret = virConnectNodeDeviceEventDeregisterAnySymbol(conn,
                                                       callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNodeDeviceEventRegisterAnyType)(virConnectPtr conn,
                                            virNodeDevicePtr dev,
                                            int eventID,
                                            virConnectNodeDeviceEventGenericCallback cb,
                                            void * opaque,
                                            virFreeCallback freecb);

int
virConnectNodeDeviceEventRegisterAnyWrapper(virConnectPtr conn,
                                            virNodeDevicePtr dev,
                                            int eventID,
                                            virConnectNodeDeviceEventGenericCallback cb,
                                            void * opaque,
                                            virFreeCallback freecb,
                                            virErrorPtr err)
{
    static virConnectNodeDeviceEventRegisterAnyType virConnectNodeDeviceEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNodeDeviceEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectNodeDeviceEventRegisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNodeDeviceEventRegisterAny",
                libvirt);
        return ret;
    }

    ret = virConnectNodeDeviceEventRegisterAnySymbol(conn,
                                                     dev,
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
(*virConnectNumOfDefinedDomainsType)(virConnectPtr conn);

int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err)
{
    static virConnectNumOfDefinedDomainsType virConnectNumOfDefinedDomainsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfDefinedDomainsSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedDomains", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfDefinedDomains",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfDefinedDomainsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDefinedInterfacesType)(virConnectPtr conn);

int
virConnectNumOfDefinedInterfacesWrapper(virConnectPtr conn,
                                        virErrorPtr err)
{
    static virConnectNumOfDefinedInterfacesType virConnectNumOfDefinedInterfacesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfDefinedInterfacesSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedInterfaces", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfDefinedInterfaces",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfDefinedInterfacesSymbol(conn);
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
    static virConnectNumOfDefinedNetworksType virConnectNumOfDefinedNetworksSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfDefinedNetworksSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedNetworks", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfDefinedNetworks",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfDefinedNetworksSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDefinedStoragePoolsType)(virConnectPtr conn);

int
virConnectNumOfDefinedStoragePoolsWrapper(virConnectPtr conn,
                                          virErrorPtr err)
{
    static virConnectNumOfDefinedStoragePoolsType virConnectNumOfDefinedStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfDefinedStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedStoragePools", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfDefinedStoragePools",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfDefinedStoragePoolsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDomainsType)(virConnectPtr conn);

int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    static virConnectNumOfDomainsType virConnectNumOfDomainsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfDomainsSymbol = libvirtSymbol(libvirt, "virConnectNumOfDomains", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfDomains",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfDomainsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfInterfacesType)(virConnectPtr conn);

int
virConnectNumOfInterfacesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    static virConnectNumOfInterfacesType virConnectNumOfInterfacesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfInterfacesSymbol = libvirtSymbol(libvirt, "virConnectNumOfInterfaces", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfInterfaces",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfInterfacesSymbol(conn);
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
    static virConnectNumOfNWFiltersType virConnectNumOfNWFiltersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfNWFiltersSymbol = libvirtSymbol(libvirt, "virConnectNumOfNWFilters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfNWFilters",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfNWFiltersSymbol(conn);
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
    static virConnectNumOfNetworksType virConnectNumOfNetworksSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfNetworksSymbol = libvirtSymbol(libvirt, "virConnectNumOfNetworks", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfNetworks",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfNetworksSymbol(conn);
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
    static virConnectNumOfSecretsType virConnectNumOfSecretsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfSecretsSymbol = libvirtSymbol(libvirt, "virConnectNumOfSecrets", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfSecrets",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfSecretsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfStoragePoolsType)(virConnectPtr conn);

int
virConnectNumOfStoragePoolsWrapper(virConnectPtr conn,
                                   virErrorPtr err)
{
    static virConnectNumOfStoragePoolsType virConnectNumOfStoragePoolsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectNumOfStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectNumOfStoragePools", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectNumOfStoragePools",
                libvirt);
        return ret;
    }

    ret = virConnectNumOfStoragePoolsSymbol(conn);
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
    static virConnectOpenType virConnectOpenSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectOpenSymbol = libvirtSymbol(libvirt, "virConnectOpen", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectOpen",
                libvirt);
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
    static virConnectOpenAuthType virConnectOpenAuthSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectOpenAuthSymbol = libvirtSymbol(libvirt, "virConnectOpenAuth", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectOpenAuth",
                libvirt);
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
    static virConnectOpenReadOnlyType virConnectOpenReadOnlySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectOpenReadOnlySymbol = libvirtSymbol(libvirt, "virConnectOpenReadOnly", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectOpenReadOnly",
                libvirt);
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
    static virConnectRefType virConnectRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectRefSymbol = libvirtSymbol(libvirt, "virConnectRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectRef",
                libvirt);
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
    static virConnectRegisterCloseCallbackType virConnectRegisterCloseCallbackSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectRegisterCloseCallbackSymbol = libvirtSymbol(libvirt, "virConnectRegisterCloseCallback", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectRegisterCloseCallback",
                libvirt);
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
(*virConnectSecretEventDeregisterAnyType)(virConnectPtr conn,
                                          int callbackID);

int
virConnectSecretEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    static virConnectSecretEventDeregisterAnyType virConnectSecretEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectSecretEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectSecretEventDeregisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectSecretEventDeregisterAny",
                libvirt);
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
    static virConnectSecretEventRegisterAnyType virConnectSecretEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectSecretEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectSecretEventRegisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectSecretEventRegisterAny",
                libvirt);
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
    static virConnectSetIdentityType virConnectSetIdentitySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectSetIdentitySymbol = libvirtSymbol(libvirt, "virConnectSetIdentity", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectSetIdentity",
                libvirt);
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
    static virConnectSetKeepAliveType virConnectSetKeepAliveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectSetKeepAliveSymbol = libvirtSymbol(libvirt, "virConnectSetKeepAlive", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectSetKeepAlive",
                libvirt);
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
(*virConnectStoragePoolEventDeregisterAnyType)(virConnectPtr conn,
                                               int callbackID);

int
virConnectStoragePoolEventDeregisterAnyWrapper(virConnectPtr conn,
                                               int callbackID,
                                               virErrorPtr err)
{
    static virConnectStoragePoolEventDeregisterAnyType virConnectStoragePoolEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectStoragePoolEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectStoragePoolEventDeregisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectStoragePoolEventDeregisterAny",
                libvirt);
        return ret;
    }

    ret = virConnectStoragePoolEventDeregisterAnySymbol(conn,
                                                        callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectStoragePoolEventRegisterAnyType)(virConnectPtr conn,
                                             virStoragePoolPtr pool,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             void * opaque,
                                             virFreeCallback freecb);

int
virConnectStoragePoolEventRegisterAnyWrapper(virConnectPtr conn,
                                             virStoragePoolPtr pool,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             void * opaque,
                                             virFreeCallback freecb,
                                             virErrorPtr err)
{
    static virConnectStoragePoolEventRegisterAnyType virConnectStoragePoolEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectStoragePoolEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectStoragePoolEventRegisterAny", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectStoragePoolEventRegisterAny",
                libvirt);
        return ret;
    }

    ret = virConnectStoragePoolEventRegisterAnySymbol(conn,
                                                      pool,
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
(*virConnectUnregisterCloseCallbackType)(virConnectPtr conn,
                                         virConnectCloseFunc cb);

int
virConnectUnregisterCloseCallbackWrapper(virConnectPtr conn,
                                         virConnectCloseFunc cb,
                                         virErrorPtr err)
{
    static virConnectUnregisterCloseCallbackType virConnectUnregisterCloseCallbackSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectUnregisterCloseCallbackSymbol = libvirtSymbol(libvirt, "virConnectUnregisterCloseCallback", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectUnregisterCloseCallback",
                libvirt);
        return ret;
    }

    ret = virConnectUnregisterCloseCallbackSymbol(conn,
                                                  cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDefaultErrorFuncType)(virErrorPtr err);

void
virDefaultErrorFuncWrapper(virErrorPtr err)
{
    static virDefaultErrorFuncType virDefaultErrorFuncSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDefaultErrorFuncSymbol = libvirtSymbol(libvirt, "virDefaultErrorFunc", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virDefaultErrorFunc",
                libvirt);
        return;
    }

    virDefaultErrorFuncSymbol(err);
}

typedef int
(*virDomainAbortJobType)(virDomainPtr domain);

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    static virDomainAbortJobType virDomainAbortJobSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAbortJobSymbol = libvirtSymbol(libvirt, "virDomainAbortJob", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAbortJob",
                libvirt);
        return ret;
    }

    ret = virDomainAbortJobSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAddIOThreadType)(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags);

int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainAddIOThreadType virDomainAddIOThreadSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAddIOThreadSymbol = libvirtSymbol(libvirt, "virDomainAddIOThread", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAddIOThread",
                libvirt);
        return ret;
    }

    ret = virDomainAddIOThreadSymbol(domain,
                                     iothread_id,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAgentSetResponseTimeoutType)(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags);

int
virDomainAgentSetResponseTimeoutWrapper(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    static virDomainAgentSetResponseTimeoutType virDomainAgentSetResponseTimeoutSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAgentSetResponseTimeoutSymbol = libvirtSymbol(libvirt, "virDomainAgentSetResponseTimeout", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAgentSetResponseTimeout",
                libvirt);
        return ret;
    }

    ret = virDomainAgentSetResponseTimeoutSymbol(domain,
                                                 timeout,
                                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAttachDeviceType)(virDomainPtr domain,
                             const char * xml);

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    static virDomainAttachDeviceType virDomainAttachDeviceSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAttachDeviceSymbol = libvirtSymbol(libvirt, "virDomainAttachDevice", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAttachDevice",
                libvirt);
        return ret;
    }

    ret = virDomainAttachDeviceSymbol(domain,
                                      xml);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAttachDeviceFlagsType)(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags);

int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainAttachDeviceFlagsType virDomainAttachDeviceFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAttachDeviceFlagsSymbol = libvirtSymbol(libvirt, "virDomainAttachDeviceFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAttachDeviceFlags",
                libvirt);
        return ret;
    }

    ret = virDomainAttachDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAuthorizedSSHKeysGetType)(virDomainPtr domain,
                                     const char * user,
                                     char ** * keys,
                                     unsigned int flags);

int
virDomainAuthorizedSSHKeysGetWrapper(virDomainPtr domain,
                                     const char * user,
                                     char ** * keys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainAuthorizedSSHKeysGetType virDomainAuthorizedSSHKeysGetSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAuthorizedSSHKeysGetSymbol = libvirtSymbol(libvirt, "virDomainAuthorizedSSHKeysGet", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAuthorizedSSHKeysGet",
                libvirt);
        return ret;
    }

    ret = virDomainAuthorizedSSHKeysGetSymbol(domain,
                                              user,
                                              keys,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAuthorizedSSHKeysSetType)(virDomainPtr domain,
                                     const char * user,
                                     const char ** keys,
                                     unsigned int nkeys,
                                     unsigned int flags);

int
virDomainAuthorizedSSHKeysSetWrapper(virDomainPtr domain,
                                     const char * user,
                                     const char ** keys,
                                     unsigned int nkeys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainAuthorizedSSHKeysSetType virDomainAuthorizedSSHKeysSetSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainAuthorizedSSHKeysSetSymbol = libvirtSymbol(libvirt, "virDomainAuthorizedSSHKeysSet", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainAuthorizedSSHKeysSet",
                libvirt);
        return ret;
    }

    ret = virDomainAuthorizedSSHKeysSetSymbol(domain,
                                              user,
                                              keys,
                                              nkeys,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBackupBeginType)(virDomainPtr domain,
                            const char * backupXML,
                            const char * checkpointXML,
                            unsigned int flags);

int
virDomainBackupBeginWrapper(virDomainPtr domain,
                            const char * backupXML,
                            const char * checkpointXML,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainBackupBeginType virDomainBackupBeginSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBackupBeginSymbol = libvirtSymbol(libvirt, "virDomainBackupBegin", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBackupBegin",
                libvirt);
        return ret;
    }

    ret = virDomainBackupBeginSymbol(domain,
                                     backupXML,
                                     checkpointXML,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainBackupGetXMLDescType)(virDomainPtr domain,
                                 unsigned int flags);

char *
virDomainBackupGetXMLDescWrapper(virDomainPtr domain,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainBackupGetXMLDescType virDomainBackupGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBackupGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainBackupGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBackupGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virDomainBackupGetXMLDescSymbol(domain,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockCommitType)(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            const char * top,
                            unsigned long bandwidth,
                            unsigned int flags);

int
virDomainBlockCommitWrapper(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            const char * top,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainBlockCommitType virDomainBlockCommitSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockCommitSymbol = libvirtSymbol(libvirt, "virDomainBlockCommit", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockCommit",
                libvirt);
        return ret;
    }

    ret = virDomainBlockCommitSymbol(dom,
                                     disk,
                                     base,
                                     top,
                                     bandwidth,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockCopyType)(virDomainPtr dom,
                          const char * disk,
                          const char * destxml,
                          virTypedParameterPtr params,
                          int nparams,
                          unsigned int flags);

int
virDomainBlockCopyWrapper(virDomainPtr dom,
                          const char * disk,
                          const char * destxml,
                          virTypedParameterPtr params,
                          int nparams,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainBlockCopyType virDomainBlockCopySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockCopySymbol = libvirtSymbol(libvirt, "virDomainBlockCopy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockCopy",
                libvirt);
        return ret;
    }

    ret = virDomainBlockCopySymbol(dom,
                                   disk,
                                   destxml,
                                   params,
                                   nparams,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockJobAbortType)(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags);

int
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainBlockJobAbortType virDomainBlockJobAbortSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockJobAbortSymbol = libvirtSymbol(libvirt, "virDomainBlockJobAbort", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockJobAbort",
                libvirt);
        return ret;
    }

    ret = virDomainBlockJobAbortSymbol(dom,
                                       disk,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockJobSetSpeedType)(virDomainPtr dom,
                                 const char * disk,
                                 unsigned long bandwidth,
                                 unsigned int flags);

int
virDomainBlockJobSetSpeedWrapper(virDomainPtr dom,
                                 const char * disk,
                                 unsigned long bandwidth,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainBlockJobSetSpeedType virDomainBlockJobSetSpeedSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockJobSetSpeedSymbol = libvirtSymbol(libvirt, "virDomainBlockJobSetSpeed", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockJobSetSpeed",
                libvirt);
        return ret;
    }

    ret = virDomainBlockJobSetSpeedSymbol(dom,
                                          disk,
                                          bandwidth,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockPeekType)(virDomainPtr dom,
                          const char * disk,
                          unsigned long long offset,
                          size_t size,
                          void * buffer,
                          unsigned int flags);

int
virDomainBlockPeekWrapper(virDomainPtr dom,
                          const char * disk,
                          unsigned long long offset,
                          size_t size,
                          void * buffer,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainBlockPeekType virDomainBlockPeekSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockPeekSymbol = libvirtSymbol(libvirt, "virDomainBlockPeek", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockPeek",
                libvirt);
        return ret;
    }

    ret = virDomainBlockPeekSymbol(dom,
                                   disk,
                                   offset,
                                   size,
                                   buffer,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockPullType)(virDomainPtr dom,
                          const char * disk,
                          unsigned long bandwidth,
                          unsigned int flags);

int
virDomainBlockPullWrapper(virDomainPtr dom,
                          const char * disk,
                          unsigned long bandwidth,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainBlockPullType virDomainBlockPullSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockPullSymbol = libvirtSymbol(libvirt, "virDomainBlockPull", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockPull",
                libvirt);
        return ret;
    }

    ret = virDomainBlockPullSymbol(dom,
                                   disk,
                                   bandwidth,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockRebaseType)(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            unsigned long bandwidth,
                            unsigned int flags);

int
virDomainBlockRebaseWrapper(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainBlockRebaseType virDomainBlockRebaseSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockRebaseSymbol = libvirtSymbol(libvirt, "virDomainBlockRebase", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockRebase",
                libvirt);
        return ret;
    }

    ret = virDomainBlockRebaseSymbol(dom,
                                     disk,
                                     base,
                                     bandwidth,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockResizeType)(virDomainPtr dom,
                            const char * disk,
                            unsigned long long size,
                            unsigned int flags);

int
virDomainBlockResizeWrapper(virDomainPtr dom,
                            const char * disk,
                            unsigned long long size,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainBlockResizeType virDomainBlockResizeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockResizeSymbol = libvirtSymbol(libvirt, "virDomainBlockResize", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockResize",
                libvirt);
        return ret;
    }

    ret = virDomainBlockResizeSymbol(dom,
                                     disk,
                                     size,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockStatsType)(virDomainPtr dom,
                           const char * disk,
                           virDomainBlockStatsPtr stats,
                           size_t size);

int
virDomainBlockStatsWrapper(virDomainPtr dom,
                           const char * disk,
                           virDomainBlockStatsPtr stats,
                           size_t size,
                           virErrorPtr err)
{
    static virDomainBlockStatsType virDomainBlockStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockStatsSymbol = libvirtSymbol(libvirt, "virDomainBlockStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockStats",
                libvirt);
        return ret;
    }

    ret = virDomainBlockStatsSymbol(dom,
                                    disk,
                                    stats,
                                    size);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockStatsFlagsType)(virDomainPtr dom,
                                const char * disk,
                                virTypedParameterPtr params,
                                int * nparams,
                                unsigned int flags);

int
virDomainBlockStatsFlagsWrapper(virDomainPtr dom,
                                const char * disk,
                                virTypedParameterPtr params,
                                int * nparams,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainBlockStatsFlagsType virDomainBlockStatsFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainBlockStatsFlagsSymbol = libvirtSymbol(libvirt, "virDomainBlockStatsFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainBlockStatsFlags",
                libvirt);
        return ret;
    }

    ret = virDomainBlockStatsFlagsSymbol(dom,
                                         disk,
                                         params,
                                         nparams,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainCheckpointPtr
(*virDomainCheckpointCreateXMLType)(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointCreateXMLWrapper(virDomainPtr domain,
                                    const char * xmlDesc,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainCheckpointCreateXMLType virDomainCheckpointCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointCreateXMLSymbol = libvirtSymbol(libvirt, "virDomainCheckpointCreateXML", &success);
    }

    virDomainCheckpointPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointCreateXML",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointCreateXMLSymbol(domain,
                                             xmlDesc,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointDeleteType)(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags);

int
virDomainCheckpointDeleteWrapper(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainCheckpointDeleteType virDomainCheckpointDeleteSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointDeleteSymbol = libvirtSymbol(libvirt, "virDomainCheckpointDelete", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointDelete",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointDeleteSymbol(checkpoint,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointFreeType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointFreeWrapper(virDomainCheckpointPtr checkpoint,
                               virErrorPtr err)
{
    static virDomainCheckpointFreeType virDomainCheckpointFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointFreeSymbol = libvirtSymbol(libvirt, "virDomainCheckpointFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointFree",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointFreeSymbol(checkpoint);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virDomainCheckpointGetConnectType)(virDomainCheckpointPtr checkpoint);

virConnectPtr
virDomainCheckpointGetConnectWrapper(virDomainCheckpointPtr checkpoint,
                                     virErrorPtr err)
{
    static virDomainCheckpointGetConnectType virDomainCheckpointGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointGetConnectSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointGetConnect",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointGetConnectSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCheckpointGetDomainType)(virDomainCheckpointPtr checkpoint);

virDomainPtr
virDomainCheckpointGetDomainWrapper(virDomainCheckpointPtr checkpoint,
                                    virErrorPtr err)
{
    static virDomainCheckpointGetDomainType virDomainCheckpointGetDomainSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointGetDomainSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetDomain", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointGetDomain",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointGetDomainSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virDomainCheckpointGetNameType)(virDomainCheckpointPtr checkpoint);

const char *
virDomainCheckpointGetNameWrapper(virDomainCheckpointPtr checkpoint,
                                  virErrorPtr err)
{
    static virDomainCheckpointGetNameType virDomainCheckpointGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointGetNameSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointGetName",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointGetNameSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainCheckpointPtr
(*virDomainCheckpointGetParentType)(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointGetParentWrapper(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainCheckpointGetParentType virDomainCheckpointGetParentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointGetParentSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetParent", &success);
    }

    virDomainCheckpointPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointGetParent",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointGetParentSymbol(checkpoint,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainCheckpointGetXMLDescType)(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags);

char *
virDomainCheckpointGetXMLDescWrapper(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainCheckpointGetXMLDescType virDomainCheckpointGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointGetXMLDescSymbol(checkpoint,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointListAllChildrenType)(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags);

int
virDomainCheckpointListAllChildrenWrapper(virDomainCheckpointPtr checkpoint,
                                          virDomainCheckpointPtr ** children,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    static virDomainCheckpointListAllChildrenType virDomainCheckpointListAllChildrenSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointListAllChildrenSymbol = libvirtSymbol(libvirt, "virDomainCheckpointListAllChildren", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointListAllChildren",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointListAllChildrenSymbol(checkpoint,
                                                   children,
                                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainCheckpointPtr
(*virDomainCheckpointLookupByNameType)(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointLookupByNameWrapper(virDomainPtr domain,
                                       const char * name,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    static virDomainCheckpointLookupByNameType virDomainCheckpointLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointLookupByNameSymbol = libvirtSymbol(libvirt, "virDomainCheckpointLookupByName", &success);
    }

    virDomainCheckpointPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointLookupByName",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointLookupByNameSymbol(domain,
                                                name,
                                                flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCheckpointRefType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointRefWrapper(virDomainCheckpointPtr checkpoint,
                              virErrorPtr err)
{
    static virDomainCheckpointRefType virDomainCheckpointRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCheckpointRefSymbol = libvirtSymbol(libvirt, "virDomainCheckpointRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCheckpointRef",
                libvirt);
        return ret;
    }

    ret = virDomainCheckpointRefSymbol(checkpoint);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCoreDumpType)(virDomainPtr domain,
                         const char * to,
                         unsigned int flags);

int
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char * to,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virDomainCoreDumpType virDomainCoreDumpSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCoreDumpSymbol = libvirtSymbol(libvirt, "virDomainCoreDump", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCoreDump",
                libvirt);
        return ret;
    }

    ret = virDomainCoreDumpSymbol(domain,
                                  to,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCoreDumpWithFormatType)(virDomainPtr domain,
                                   const char * to,
                                   unsigned int dumpformat,
                                   unsigned int flags);

int
virDomainCoreDumpWithFormatWrapper(virDomainPtr domain,
                                   const char * to,
                                   unsigned int dumpformat,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainCoreDumpWithFormatType virDomainCoreDumpWithFormatSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCoreDumpWithFormatSymbol = libvirtSymbol(libvirt, "virDomainCoreDumpWithFormat", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCoreDumpWithFormat",
                libvirt);
        return ret;
    }

    ret = virDomainCoreDumpWithFormatSymbol(domain,
                                            to,
                                            dumpformat,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCreateType)(virDomainPtr domain);

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    static virDomainCreateType virDomainCreateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCreateSymbol = libvirtSymbol(libvirt, "virDomainCreate", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCreate",
                libvirt);
        return ret;
    }

    ret = virDomainCreateSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCreateLinuxType)(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags);

virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainCreateLinuxType virDomainCreateLinuxSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCreateLinuxSymbol = libvirtSymbol(libvirt, "virDomainCreateLinux", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCreateLinux",
                libvirt);
        return ret;
    }

    ret = virDomainCreateLinuxSymbol(conn,
                                     xmlDesc,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCreateWithFilesType)(virDomainPtr domain,
                                unsigned int nfiles,
                                int * files,
                                unsigned int flags);

int
virDomainCreateWithFilesWrapper(virDomainPtr domain,
                                unsigned int nfiles,
                                int * files,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainCreateWithFilesType virDomainCreateWithFilesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCreateWithFilesSymbol = libvirtSymbol(libvirt, "virDomainCreateWithFiles", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCreateWithFiles",
                libvirt);
        return ret;
    }

    ret = virDomainCreateWithFilesSymbol(domain,
                                         nfiles,
                                         files,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCreateWithFlagsType)(virDomainPtr domain,
                                unsigned int flags);

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainCreateWithFlagsType virDomainCreateWithFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCreateWithFlagsSymbol = libvirtSymbol(libvirt, "virDomainCreateWithFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCreateWithFlags",
                libvirt);
        return ret;
    }

    ret = virDomainCreateWithFlagsSymbol(domain,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCreateXMLType)(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags);

virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainCreateXMLType virDomainCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCreateXMLSymbol = libvirtSymbol(libvirt, "virDomainCreateXML", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCreateXML",
                libvirt);
        return ret;
    }

    ret = virDomainCreateXMLSymbol(conn,
                                   xmlDesc,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCreateXMLWithFilesType)(virConnectPtr conn,
                                   const char * xmlDesc,
                                   unsigned int nfiles,
                                   int * files,
                                   unsigned int flags);

virDomainPtr
virDomainCreateXMLWithFilesWrapper(virConnectPtr conn,
                                   const char * xmlDesc,
                                   unsigned int nfiles,
                                   int * files,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainCreateXMLWithFilesType virDomainCreateXMLWithFilesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainCreateXMLWithFilesSymbol = libvirtSymbol(libvirt, "virDomainCreateXMLWithFiles", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainCreateXMLWithFiles",
                libvirt);
        return ret;
    }

    ret = virDomainCreateXMLWithFilesSymbol(conn,
                                            xmlDesc,
                                            nfiles,
                                            files,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainDefineXMLType)(virConnectPtr conn,
                          const char * xml);

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          virErrorPtr err)
{
    static virDomainDefineXMLType virDomainDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDefineXMLSymbol = libvirtSymbol(libvirt, "virDomainDefineXML", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDefineXML",
                libvirt);
        return ret;
    }

    ret = virDomainDefineXMLSymbol(conn,
                                   xml);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainDefineXMLFlagsType)(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags);

virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainDefineXMLFlagsType virDomainDefineXMLFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDefineXMLFlagsSymbol = libvirtSymbol(libvirt, "virDomainDefineXMLFlags", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDefineXMLFlags",
                libvirt);
        return ret;
    }

    ret = virDomainDefineXMLFlagsSymbol(conn,
                                        xml,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDelIOThreadType)(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags);

int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainDelIOThreadType virDomainDelIOThreadSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDelIOThreadSymbol = libvirtSymbol(libvirt, "virDomainDelIOThread", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDelIOThread",
                libvirt);
        return ret;
    }

    ret = virDomainDelIOThreadSymbol(domain,
                                     iothread_id,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDestroyType)(virDomainPtr domain);

int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    static virDomainDestroyType virDomainDestroySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDestroySymbol = libvirtSymbol(libvirt, "virDomainDestroy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDestroy",
                libvirt);
        return ret;
    }

    ret = virDomainDestroySymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDestroyFlagsType)(virDomainPtr domain,
                             unsigned int flags);

int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainDestroyFlagsType virDomainDestroyFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDestroyFlagsSymbol = libvirtSymbol(libvirt, "virDomainDestroyFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDestroyFlags",
                libvirt);
        return ret;
    }

    ret = virDomainDestroyFlagsSymbol(domain,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDetachDeviceType)(virDomainPtr domain,
                             const char * xml);

int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    static virDomainDetachDeviceType virDomainDetachDeviceSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDetachDeviceSymbol = libvirtSymbol(libvirt, "virDomainDetachDevice", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDetachDevice",
                libvirt);
        return ret;
    }

    ret = virDomainDetachDeviceSymbol(domain,
                                      xml);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDetachDeviceAliasType)(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags);

int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainDetachDeviceAliasType virDomainDetachDeviceAliasSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDetachDeviceAliasSymbol = libvirtSymbol(libvirt, "virDomainDetachDeviceAlias", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDetachDeviceAlias",
                libvirt);
        return ret;
    }

    ret = virDomainDetachDeviceAliasSymbol(domain,
                                           alias,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDetachDeviceFlagsType)(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags);

int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainDetachDeviceFlagsType virDomainDetachDeviceFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainDetachDeviceFlagsSymbol = libvirtSymbol(libvirt, "virDomainDetachDeviceFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainDetachDeviceFlags",
                libvirt);
        return ret;
    }

    ret = virDomainDetachDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFSFreezeType)(virDomainPtr dom,
                         const char ** mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags);

int
virDomainFSFreezeWrapper(virDomainPtr dom,
                         const char ** mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virDomainFSFreezeType virDomainFSFreezeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainFSFreezeSymbol = libvirtSymbol(libvirt, "virDomainFSFreeze", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainFSFreeze",
                libvirt);
        return ret;
    }

    ret = virDomainFSFreezeSymbol(dom,
                                  mountpoints,
                                  nmountpoints,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainFSInfoFreeType)(virDomainFSInfoPtr info);

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info)
{
    static virDomainFSInfoFreeType virDomainFSInfoFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainFSInfoFreeSymbol = libvirtSymbol(libvirt, "virDomainFSInfoFree", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainFSInfoFree",
                libvirt);
        return;
    }

    virDomainFSInfoFreeSymbol(info);
}

typedef int
(*virDomainFSThawType)(virDomainPtr dom,
                       const char ** mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags);

int
virDomainFSThawWrapper(virDomainPtr dom,
                       const char ** mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags,
                       virErrorPtr err)
{
    static virDomainFSThawType virDomainFSThawSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainFSThawSymbol = libvirtSymbol(libvirt, "virDomainFSThaw", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainFSThaw",
                libvirt);
        return ret;
    }

    ret = virDomainFSThawSymbol(dom,
                                mountpoints,
                                nmountpoints,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFSTrimType)(virDomainPtr dom,
                       const char * mountPoint,
                       unsigned long long minimum,
                       unsigned int flags);

int
virDomainFSTrimWrapper(virDomainPtr dom,
                       const char * mountPoint,
                       unsigned long long minimum,
                       unsigned int flags,
                       virErrorPtr err)
{
    static virDomainFSTrimType virDomainFSTrimSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainFSTrimSymbol = libvirtSymbol(libvirt, "virDomainFSTrim", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainFSTrim",
                libvirt);
        return ret;
    }

    ret = virDomainFSTrimSymbol(dom,
                                mountPoint,
                                minimum,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFreeType)(virDomainPtr domain);

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err)
{
    static virDomainFreeType virDomainFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainFreeSymbol = libvirtSymbol(libvirt, "virDomainFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainFree",
                libvirt);
        return ret;
    }

    ret = virDomainFreeSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetAutostartType)(virDomainPtr domain,
                             int * autostart);

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int * autostart,
                             virErrorPtr err)
{
    static virDomainGetAutostartType virDomainGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetAutostartSymbol = libvirtSymbol(libvirt, "virDomainGetAutostart", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetAutostart",
                libvirt);
        return ret;
    }

    ret = virDomainGetAutostartSymbol(domain,
                                      autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlkioParametersType)(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int * nparams,
                                   unsigned int flags);

int
virDomainGetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainGetBlkioParametersType virDomainGetBlkioParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetBlkioParametersSymbol = libvirtSymbol(libvirt, "virDomainGetBlkioParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetBlkioParameters",
                libvirt);
        return ret;
    }

    ret = virDomainGetBlkioParametersSymbol(domain,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlockInfoType)(virDomainPtr domain,
                             const char * disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags);

int
virDomainGetBlockInfoWrapper(virDomainPtr domain,
                             const char * disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainGetBlockInfoType virDomainGetBlockInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetBlockInfoSymbol = libvirtSymbol(libvirt, "virDomainGetBlockInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetBlockInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetBlockInfoSymbol(domain,
                                      disk,
                                      info,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlockIoTuneType)(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int * nparams,
                               unsigned int flags);

int
virDomainGetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int * nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainGetBlockIoTuneType virDomainGetBlockIoTuneSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetBlockIoTuneSymbol = libvirtSymbol(libvirt, "virDomainGetBlockIoTune", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetBlockIoTune",
                libvirt);
        return ret;
    }

    ret = virDomainGetBlockIoTuneSymbol(dom,
                                        disk,
                                        params,
                                        nparams,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlockJobInfoType)(virDomainPtr dom,
                                const char * disk,
                                virDomainBlockJobInfoPtr info,
                                unsigned int flags);

int
virDomainGetBlockJobInfoWrapper(virDomainPtr dom,
                                const char * disk,
                                virDomainBlockJobInfoPtr info,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainGetBlockJobInfoType virDomainGetBlockJobInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetBlockJobInfoSymbol = libvirtSymbol(libvirt, "virDomainGetBlockJobInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetBlockJobInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetBlockJobInfoSymbol(dom,
                                         disk,
                                         info,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetCPUStatsType)(virDomainPtr domain,
                            virTypedParameterPtr params,
                            unsigned int nparams,
                            int start_cpu,
                            unsigned int ncpus,
                            unsigned int flags);

int
virDomainGetCPUStatsWrapper(virDomainPtr domain,
                            virTypedParameterPtr params,
                            unsigned int nparams,
                            int start_cpu,
                            unsigned int ncpus,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainGetCPUStatsType virDomainGetCPUStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetCPUStatsSymbol = libvirtSymbol(libvirt, "virDomainGetCPUStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetCPUStats",
                libvirt);
        return ret;
    }

    ret = virDomainGetCPUStatsSymbol(domain,
                                     params,
                                     nparams,
                                     start_cpu,
                                     ncpus,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virDomainGetConnectType)(virDomainPtr dom);

virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err)
{
    static virDomainGetConnectType virDomainGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetConnectSymbol = libvirtSymbol(libvirt, "virDomainGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetConnect",
                libvirt);
        return ret;
    }

    ret = virDomainGetConnectSymbol(dom);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetControlInfoType)(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags);

int
virDomainGetControlInfoWrapper(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainGetControlInfoType virDomainGetControlInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetControlInfoSymbol = libvirtSymbol(libvirt, "virDomainGetControlInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetControlInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetControlInfoSymbol(domain,
                                        info,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetDiskErrorsType)(virDomainPtr dom,
                              virDomainDiskErrorPtr errors,
                              unsigned int maxerrors,
                              unsigned int flags);

int
virDomainGetDiskErrorsWrapper(virDomainPtr dom,
                              virDomainDiskErrorPtr errors,
                              unsigned int maxerrors,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainGetDiskErrorsType virDomainGetDiskErrorsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetDiskErrorsSymbol = libvirtSymbol(libvirt, "virDomainGetDiskErrors", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetDiskErrors",
                libvirt);
        return ret;
    }

    ret = virDomainGetDiskErrorsSymbol(dom,
                                       errors,
                                       maxerrors,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetEmulatorPinInfoType)(virDomainPtr domain,
                                   unsigned char * cpumap,
                                   int maplen,
                                   unsigned int flags);

int
virDomainGetEmulatorPinInfoWrapper(virDomainPtr domain,
                                   unsigned char * cpumap,
                                   int maplen,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainGetEmulatorPinInfoType virDomainGetEmulatorPinInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetEmulatorPinInfoSymbol = libvirtSymbol(libvirt, "virDomainGetEmulatorPinInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetEmulatorPinInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetEmulatorPinInfoSymbol(domain,
                                            cpumap,
                                            maplen,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetFSInfoType)(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags);

int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainGetFSInfoType virDomainGetFSInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetFSInfoSymbol = libvirtSymbol(libvirt, "virDomainGetFSInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetFSInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetFSInfoSymbol(dom,
                                   info,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetGuestInfoType)(virDomainPtr domain,
                             unsigned int types,
                             virTypedParameterPtr * params,
                             int * nparams,
                             unsigned int flags);

int
virDomainGetGuestInfoWrapper(virDomainPtr domain,
                             unsigned int types,
                             virTypedParameterPtr * params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainGetGuestInfoType virDomainGetGuestInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetGuestInfoSymbol = libvirtSymbol(libvirt, "virDomainGetGuestInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetGuestInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetGuestInfoSymbol(domain,
                                      types,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetGuestVcpusType)(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              unsigned int * nparams,
                              unsigned int flags);

int
virDomainGetGuestVcpusWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              unsigned int * nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainGetGuestVcpusType virDomainGetGuestVcpusSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetGuestVcpusSymbol = libvirtSymbol(libvirt, "virDomainGetGuestVcpus", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetGuestVcpus",
                libvirt);
        return ret;
    }

    ret = virDomainGetGuestVcpusSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetHostnameType)(virDomainPtr domain,
                            unsigned int flags);

char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainGetHostnameType virDomainGetHostnameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetHostnameSymbol = libvirtSymbol(libvirt, "virDomainGetHostname", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetHostname",
                libvirt);
        return ret;
    }

    ret = virDomainGetHostnameSymbol(domain,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned int
(*virDomainGetIDType)(virDomainPtr domain);

unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err)
{
    static virDomainGetIDType virDomainGetIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetIDSymbol = libvirtSymbol(libvirt, "virDomainGetID", &success);
    }

    unsigned int ret = 0;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetID",
                libvirt);
        return ret;
    }

    ret = virDomainGetIDSymbol(domain);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetIOThreadInfoType)(virDomainPtr dom,
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags);

int
virDomainGetIOThreadInfoWrapper(virDomainPtr dom,
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainGetIOThreadInfoType virDomainGetIOThreadInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetIOThreadInfoSymbol = libvirtSymbol(libvirt, "virDomainGetIOThreadInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetIOThreadInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetIOThreadInfoSymbol(dom,
                                         info,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetInfoType)(virDomainPtr domain,
                        virDomainInfoPtr info);

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err)
{
    static virDomainGetInfoType virDomainGetInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetInfoSymbol = libvirtSymbol(libvirt, "virDomainGetInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetInfoSymbol(domain,
                                 info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetInterfaceParametersType)(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       unsigned int flags);

int
virDomainGetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    static virDomainGetInterfaceParametersType virDomainGetInterfaceParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetInterfaceParametersSymbol = libvirtSymbol(libvirt, "virDomainGetInterfaceParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetInterfaceParameters",
                libvirt);
        return ret;
    }

    ret = virDomainGetInterfaceParametersSymbol(domain,
                                                device,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetJobInfoType)(virDomainPtr domain,
                           virDomainJobInfoPtr info);

int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err)
{
    static virDomainGetJobInfoType virDomainGetJobInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetJobInfoSymbol = libvirtSymbol(libvirt, "virDomainGetJobInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetJobInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetJobInfoSymbol(domain,
                                    info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetJobStatsType)(virDomainPtr domain,
                            int * type,
                            virTypedParameterPtr * params,
                            int * nparams,
                            unsigned int flags);

int
virDomainGetJobStatsWrapper(virDomainPtr domain,
                            int * type,
                            virTypedParameterPtr * params,
                            int * nparams,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainGetJobStatsType virDomainGetJobStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetJobStatsSymbol = libvirtSymbol(libvirt, "virDomainGetJobStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetJobStats",
                libvirt);
        return ret;
    }

    ret = virDomainGetJobStatsSymbol(domain,
                                     type,
                                     params,
                                     nparams,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetLaunchSecurityInfoType)(virDomainPtr domain,
                                      virTypedParameterPtr * params,
                                      int * nparams,
                                      unsigned int flags);

int
virDomainGetLaunchSecurityInfoWrapper(virDomainPtr domain,
                                      virTypedParameterPtr * params,
                                      int * nparams,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    static virDomainGetLaunchSecurityInfoType virDomainGetLaunchSecurityInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetLaunchSecurityInfoSymbol = libvirtSymbol(libvirt, "virDomainGetLaunchSecurityInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetLaunchSecurityInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetLaunchSecurityInfoSymbol(domain,
                                               params,
                                               nparams,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned long
(*virDomainGetMaxMemoryType)(virDomainPtr domain);

unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err)
{
    static virDomainGetMaxMemoryType virDomainGetMaxMemorySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetMaxMemorySymbol = libvirtSymbol(libvirt, "virDomainGetMaxMemory", &success);
    }

    unsigned long ret = 0;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetMaxMemory",
                libvirt);
        return ret;
    }

    ret = virDomainGetMaxMemorySymbol(domain);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetMaxVcpusType)(virDomainPtr domain);

int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err)
{
    static virDomainGetMaxVcpusType virDomainGetMaxVcpusSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetMaxVcpusSymbol = libvirtSymbol(libvirt, "virDomainGetMaxVcpus", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetMaxVcpus",
                libvirt);
        return ret;
    }

    ret = virDomainGetMaxVcpusSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetMemoryParametersType)(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int * nparams,
                                    unsigned int flags);

int
virDomainGetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int * nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainGetMemoryParametersType virDomainGetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetMemoryParametersSymbol = libvirtSymbol(libvirt, "virDomainGetMemoryParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetMemoryParameters",
                libvirt);
        return ret;
    }

    ret = virDomainGetMemoryParametersSymbol(domain,
                                             params,
                                             nparams,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetMessagesType)(virDomainPtr domain,
                            char ** * msgs,
                            unsigned int flags);

int
virDomainGetMessagesWrapper(virDomainPtr domain,
                            char ** * msgs,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainGetMessagesType virDomainGetMessagesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetMessagesSymbol = libvirtSymbol(libvirt, "virDomainGetMessages", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetMessages",
                libvirt);
        return ret;
    }

    ret = virDomainGetMessagesSymbol(domain,
                                     msgs,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetMetadataType)(virDomainPtr domain,
                            int type,
                            const char * uri,
                            unsigned int flags);

char *
virDomainGetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char * uri,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainGetMetadataType virDomainGetMetadataSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetMetadataSymbol = libvirtSymbol(libvirt, "virDomainGetMetadata", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetMetadata",
                libvirt);
        return ret;
    }

    ret = virDomainGetMetadataSymbol(domain,
                                     type,
                                     uri,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virDomainGetNameType)(virDomainPtr domain);

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    static virDomainGetNameType virDomainGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetNameSymbol = libvirtSymbol(libvirt, "virDomainGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetName",
                libvirt);
        return ret;
    }

    ret = virDomainGetNameSymbol(domain);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetNumaParametersType)(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags);

int
virDomainGetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainGetNumaParametersType virDomainGetNumaParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetNumaParametersSymbol = libvirtSymbol(libvirt, "virDomainGetNumaParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetNumaParameters",
                libvirt);
        return ret;
    }

    ret = virDomainGetNumaParametersSymbol(domain,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetOSTypeType)(virDomainPtr domain);

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err)
{
    static virDomainGetOSTypeType virDomainGetOSTypeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetOSTypeSymbol = libvirtSymbol(libvirt, "virDomainGetOSType", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetOSType",
                libvirt);
        return ret;
    }

    ret = virDomainGetOSTypeSymbol(domain);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetPerfEventsType)(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              int * nparams,
                              unsigned int flags);

int
virDomainGetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              int * nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainGetPerfEventsType virDomainGetPerfEventsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetPerfEventsSymbol = libvirtSymbol(libvirt, "virDomainGetPerfEvents", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetPerfEvents",
                libvirt);
        return ret;
    }

    ret = virDomainGetPerfEventsSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSchedulerParametersType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams);

int
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       virErrorPtr err)
{
    static virDomainGetSchedulerParametersType virDomainGetSchedulerParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetSchedulerParametersSymbol = libvirtSymbol(libvirt, "virDomainGetSchedulerParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetSchedulerParameters",
                libvirt);
        return ret;
    }

    ret = virDomainGetSchedulerParametersSymbol(domain,
                                                params,
                                                nparams);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSchedulerParametersFlagsType)(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int * nparams,
                                            unsigned int flags);

int
virDomainGetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int * nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    static virDomainGetSchedulerParametersFlagsType virDomainGetSchedulerParametersFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetSchedulerParametersFlagsSymbol = libvirtSymbol(libvirt, "virDomainGetSchedulerParametersFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetSchedulerParametersFlags",
                libvirt);
        return ret;
    }

    ret = virDomainGetSchedulerParametersFlagsSymbol(domain,
                                                     params,
                                                     nparams,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetSchedulerTypeType)(virDomainPtr domain,
                                 int * nparams);

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int * nparams,
                                 virErrorPtr err)
{
    static virDomainGetSchedulerTypeType virDomainGetSchedulerTypeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetSchedulerTypeSymbol = libvirtSymbol(libvirt, "virDomainGetSchedulerType", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetSchedulerType",
                libvirt);
        return ret;
    }

    ret = virDomainGetSchedulerTypeSymbol(domain,
                                          nparams);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSecurityLabelType)(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel);

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err)
{
    static virDomainGetSecurityLabelType virDomainGetSecurityLabelSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetSecurityLabelSymbol = libvirtSymbol(libvirt, "virDomainGetSecurityLabel", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetSecurityLabel",
                libvirt);
        return ret;
    }

    ret = virDomainGetSecurityLabelSymbol(domain,
                                          seclabel);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSecurityLabelListType)(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels);

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels,
                                     virErrorPtr err)
{
    static virDomainGetSecurityLabelListType virDomainGetSecurityLabelListSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetSecurityLabelListSymbol = libvirtSymbol(libvirt, "virDomainGetSecurityLabelList", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetSecurityLabelList",
                libvirt);
        return ret;
    }

    ret = virDomainGetSecurityLabelListSymbol(domain,
                                              seclabels);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetStateType)(virDomainPtr domain,
                         int * state,
                         int * reason,
                         unsigned int flags);

int
virDomainGetStateWrapper(virDomainPtr domain,
                         int * state,
                         int * reason,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virDomainGetStateType virDomainGetStateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetStateSymbol = libvirtSymbol(libvirt, "virDomainGetState", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetState",
                libvirt);
        return ret;
    }

    ret = virDomainGetStateSymbol(domain,
                                  state,
                                  reason,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetTimeType)(virDomainPtr dom,
                        long long * seconds,
                        unsigned int * nseconds,
                        unsigned int flags);

int
virDomainGetTimeWrapper(virDomainPtr dom,
                        long long * seconds,
                        unsigned int * nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
    static virDomainGetTimeType virDomainGetTimeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetTimeSymbol = libvirtSymbol(libvirt, "virDomainGetTime", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetTime",
                libvirt);
        return ret;
    }

    ret = virDomainGetTimeSymbol(dom,
                                 seconds,
                                 nseconds,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetUUIDType)(virDomainPtr domain,
                        unsigned char * uuid);

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    static virDomainGetUUIDType virDomainGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetUUIDSymbol = libvirtSymbol(libvirt, "virDomainGetUUID", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetUUID",
                libvirt);
        return ret;
    }

    ret = virDomainGetUUIDSymbol(domain,
                                 uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetUUIDStringType)(virDomainPtr domain,
                              char * buf);

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char * buf,
                              virErrorPtr err)
{
    static virDomainGetUUIDStringType virDomainGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetUUIDStringSymbol = libvirtSymbol(libvirt, "virDomainGetUUIDString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetUUIDString",
                libvirt);
        return ret;
    }

    ret = virDomainGetUUIDStringSymbol(domain,
                                       buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetVcpuPinInfoType)(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char * cpumaps,
                               int maplen,
                               unsigned int flags);

int
virDomainGetVcpuPinInfoWrapper(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char * cpumaps,
                               int maplen,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainGetVcpuPinInfoType virDomainGetVcpuPinInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetVcpuPinInfoSymbol = libvirtSymbol(libvirt, "virDomainGetVcpuPinInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetVcpuPinInfo",
                libvirt);
        return ret;
    }

    ret = virDomainGetVcpuPinInfoSymbol(domain,
                                        ncpumaps,
                                        cpumaps,
                                        maplen,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetVcpusType)(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char * cpumaps,
                         int maplen);

int
virDomainGetVcpusWrapper(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char * cpumaps,
                         int maplen,
                         virErrorPtr err)
{
    static virDomainGetVcpusType virDomainGetVcpusSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetVcpusSymbol = libvirtSymbol(libvirt, "virDomainGetVcpus", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetVcpus",
                libvirt);
        return ret;
    }

    ret = virDomainGetVcpusSymbol(domain,
                                  info,
                                  maxinfo,
                                  cpumaps,
                                  maplen);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetVcpusFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainGetVcpusFlagsType virDomainGetVcpusFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetVcpusFlagsSymbol = libvirtSymbol(libvirt, "virDomainGetVcpusFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetVcpusFlags",
                libvirt);
        return ret;
    }

    ret = virDomainGetVcpusFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetXMLDescType)(virDomainPtr domain,
                           unsigned int flags);

char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virDomainGetXMLDescType virDomainGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virDomainGetXMLDescSymbol(domain,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainHasCurrentSnapshotType)(virDomainPtr domain,
                                   unsigned int flags);

int
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainHasCurrentSnapshotType virDomainHasCurrentSnapshotSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainHasCurrentSnapshotSymbol = libvirtSymbol(libvirt, "virDomainHasCurrentSnapshot", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainHasCurrentSnapshot",
                libvirt);
        return ret;
    }

    ret = virDomainHasCurrentSnapshotSymbol(domain,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainHasManagedSaveImageType)(virDomainPtr dom,
                                    unsigned int flags);

int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainHasManagedSaveImageType virDomainHasManagedSaveImageSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainHasManagedSaveImageSymbol = libvirtSymbol(libvirt, "virDomainHasManagedSaveImage", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainHasManagedSaveImage",
                libvirt);
        return ret;
    }

    ret = virDomainHasManagedSaveImageSymbol(dom,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainIOThreadInfoFreeType)(virDomainIOThreadInfoPtr info);

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info)
{
    static virDomainIOThreadInfoFreeType virDomainIOThreadInfoFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainIOThreadInfoFreeSymbol = libvirtSymbol(libvirt, "virDomainIOThreadInfoFree", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainIOThreadInfoFree",
                libvirt);
        return;
    }

    virDomainIOThreadInfoFreeSymbol(info);
}

typedef int
(*virDomainInjectNMIType)(virDomainPtr domain,
                          unsigned int flags);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainInjectNMIType virDomainInjectNMISymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainInjectNMISymbol = libvirtSymbol(libvirt, "virDomainInjectNMI", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainInjectNMI",
                libvirt);
        return ret;
    }

    ret = virDomainInjectNMISymbol(domain,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainInterfaceAddressesType)(virDomainPtr dom,
                                   virDomainInterfacePtr ** ifaces,
                                   unsigned int source,
                                   unsigned int flags);

int
virDomainInterfaceAddressesWrapper(virDomainPtr dom,
                                   virDomainInterfacePtr ** ifaces,
                                   unsigned int source,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainInterfaceAddressesType virDomainInterfaceAddressesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainInterfaceAddressesSymbol = libvirtSymbol(libvirt, "virDomainInterfaceAddresses", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainInterfaceAddresses",
                libvirt);
        return ret;
    }

    ret = virDomainInterfaceAddressesSymbol(dom,
                                            ifaces,
                                            source,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainInterfaceFreeType)(virDomainInterfacePtr iface);

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface)
{
    static virDomainInterfaceFreeType virDomainInterfaceFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainInterfaceFreeSymbol = libvirtSymbol(libvirt, "virDomainInterfaceFree", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainInterfaceFree",
                libvirt);
        return;
    }

    virDomainInterfaceFreeSymbol(iface);
}

typedef int
(*virDomainInterfaceStatsType)(virDomainPtr dom,
                               const char * device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size);

int
virDomainInterfaceStatsWrapper(virDomainPtr dom,
                               const char * device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size,
                               virErrorPtr err)
{
    static virDomainInterfaceStatsType virDomainInterfaceStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainInterfaceStatsSymbol = libvirtSymbol(libvirt, "virDomainInterfaceStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainInterfaceStats",
                libvirt);
        return ret;
    }

    ret = virDomainInterfaceStatsSymbol(dom,
                                        device,
                                        stats,
                                        size);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainIsActiveType)(virDomainPtr dom);

int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err)
{
    static virDomainIsActiveType virDomainIsActiveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainIsActiveSymbol = libvirtSymbol(libvirt, "virDomainIsActive", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainIsActive",
                libvirt);
        return ret;
    }

    ret = virDomainIsActiveSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainIsPersistentType)(virDomainPtr dom);

int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err)
{
    static virDomainIsPersistentType virDomainIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainIsPersistentSymbol = libvirtSymbol(libvirt, "virDomainIsPersistent", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainIsPersistent",
                libvirt);
        return ret;
    }

    ret = virDomainIsPersistentSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainIsUpdatedType)(virDomainPtr dom);

int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err)
{
    static virDomainIsUpdatedType virDomainIsUpdatedSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainIsUpdatedSymbol = libvirtSymbol(libvirt, "virDomainIsUpdated", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainIsUpdated",
                libvirt);
        return ret;
    }

    ret = virDomainIsUpdatedSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainListAllCheckpointsType)(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags);

int
virDomainListAllCheckpointsWrapper(virDomainPtr domain,
                                   virDomainCheckpointPtr ** checkpoints,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainListAllCheckpointsType virDomainListAllCheckpointsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainListAllCheckpointsSymbol = libvirtSymbol(libvirt, "virDomainListAllCheckpoints", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainListAllCheckpoints",
                libvirt);
        return ret;
    }

    ret = virDomainListAllCheckpointsSymbol(domain,
                                            checkpoints,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainListAllSnapshotsType)(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags);

int
virDomainListAllSnapshotsWrapper(virDomainPtr domain,
                                 virDomainSnapshotPtr ** snaps,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainListAllSnapshotsType virDomainListAllSnapshotsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainListAllSnapshotsSymbol = libvirtSymbol(libvirt, "virDomainListAllSnapshots", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainListAllSnapshots",
                libvirt);
        return ret;
    }

    ret = virDomainListAllSnapshotsSymbol(domain,
                                          snaps,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainListGetStatsType)(virDomainPtr * doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr ** retStats,
                             unsigned int flags);

int
virDomainListGetStatsWrapper(virDomainPtr * doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr ** retStats,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainListGetStatsType virDomainListGetStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainListGetStatsSymbol = libvirtSymbol(libvirt, "virDomainListGetStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainListGetStats",
                libvirt);
        return ret;
    }

    ret = virDomainListGetStatsSymbol(doms,
                                      stats,
                                      retStats,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByIDType)(virConnectPtr conn,
                           int id);

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err)
{
    static virDomainLookupByIDType virDomainLookupByIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLookupByIDSymbol = libvirtSymbol(libvirt, "virDomainLookupByID", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLookupByID",
                libvirt);
        return ret;
    }

    ret = virDomainLookupByIDSymbol(conn,
                                    id);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByNameType)(virConnectPtr conn,
                             const char * name);

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char * name,
                             virErrorPtr err)
{
    static virDomainLookupByNameType virDomainLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLookupByNameSymbol = libvirtSymbol(libvirt, "virDomainLookupByName", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLookupByName",
                libvirt);
        return ret;
    }

    ret = virDomainLookupByNameSymbol(conn,
                                      name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByUUIDType)(virConnectPtr conn,
                             const unsigned char * uuid);

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    static virDomainLookupByUUIDType virDomainLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLookupByUUIDSymbol = libvirtSymbol(libvirt, "virDomainLookupByUUID", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLookupByUUID",
                libvirt);
        return ret;
    }

    ret = virDomainLookupByUUIDSymbol(conn,
                                      uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByUUIDStringType)(virConnectPtr conn,
                                   const char * uuidstr);

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    static virDomainLookupByUUIDStringType virDomainLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virDomainLookupByUUIDString", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLookupByUUIDString",
                libvirt);
        return ret;
    }

    ret = virDomainLookupByUUIDStringSymbol(conn,
                                            uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainManagedSaveType)(virDomainPtr dom,
                            unsigned int flags);

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainManagedSaveType virDomainManagedSaveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainManagedSaveSymbol = libvirtSymbol(libvirt, "virDomainManagedSave", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainManagedSave",
                libvirt);
        return ret;
    }

    ret = virDomainManagedSaveSymbol(dom,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainManagedSaveDefineXMLType)(virDomainPtr domain,
                                     const char * dxml,
                                     unsigned int flags);

int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char * dxml,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainManagedSaveDefineXMLType virDomainManagedSaveDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainManagedSaveDefineXMLSymbol = libvirtSymbol(libvirt, "virDomainManagedSaveDefineXML", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainManagedSaveDefineXML",
                libvirt);
        return ret;
    }

    ret = virDomainManagedSaveDefineXMLSymbol(domain,
                                              dxml,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainManagedSaveGetXMLDescType)(virDomainPtr domain,
                                      unsigned int flags);

char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    static virDomainManagedSaveGetXMLDescType virDomainManagedSaveGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainManagedSaveGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainManagedSaveGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainManagedSaveGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virDomainManagedSaveGetXMLDescSymbol(domain,
                                               flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainManagedSaveRemoveType)(virDomainPtr dom,
                                  unsigned int flags);

int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainManagedSaveRemoveType virDomainManagedSaveRemoveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainManagedSaveRemoveSymbol = libvirtSymbol(libvirt, "virDomainManagedSaveRemove", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainManagedSaveRemove",
                libvirt);
        return ret;
    }

    ret = virDomainManagedSaveRemoveSymbol(dom,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMemoryPeekType)(virDomainPtr dom,
                           unsigned long long start,
                           size_t size,
                           void * buffer,
                           unsigned int flags);

int
virDomainMemoryPeekWrapper(virDomainPtr dom,
                           unsigned long long start,
                           size_t size,
                           void * buffer,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virDomainMemoryPeekType virDomainMemoryPeekSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMemoryPeekSymbol = libvirtSymbol(libvirt, "virDomainMemoryPeek", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMemoryPeek",
                libvirt);
        return ret;
    }

    ret = virDomainMemoryPeekSymbol(dom,
                                    start,
                                    size,
                                    buffer,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMemoryStatsType)(virDomainPtr dom,
                            virDomainMemoryStatPtr stats,
                            unsigned int nr_stats,
                            unsigned int flags);

int
virDomainMemoryStatsWrapper(virDomainPtr dom,
                            virDomainMemoryStatPtr stats,
                            unsigned int nr_stats,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainMemoryStatsType virDomainMemoryStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMemoryStatsSymbol = libvirtSymbol(libvirt, "virDomainMemoryStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMemoryStats",
                libvirt);
        return ret;
    }

    ret = virDomainMemoryStatsSymbol(dom,
                                     stats,
                                     nr_stats,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainMigrateType)(virDomainPtr domain,
                        virConnectPtr dconn,
                        unsigned long flags,
                        const char * dname,
                        const char * uri,
                        unsigned long bandwidth);

virDomainPtr
virDomainMigrateWrapper(virDomainPtr domain,
                        virConnectPtr dconn,
                        unsigned long flags,
                        const char * dname,
                        const char * uri,
                        unsigned long bandwidth,
                        virErrorPtr err)
{
    static virDomainMigrateType virDomainMigrateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateSymbol = libvirtSymbol(libvirt, "virDomainMigrate", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrate",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateSymbol(domain,
                                 dconn,
                                 flags,
                                 dname,
                                 uri,
                                 bandwidth);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainMigrate2Type)(virDomainPtr domain,
                         virConnectPtr dconn,
                         const char * dxml,
                         unsigned long flags,
                         const char * dname,
                         const char * uri,
                         unsigned long bandwidth);

virDomainPtr
virDomainMigrate2Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         const char * dxml,
                         unsigned long flags,
                         const char * dname,
                         const char * uri,
                         unsigned long bandwidth,
                         virErrorPtr err)
{
    static virDomainMigrate2Type virDomainMigrate2Symbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrate2Symbol = libvirtSymbol(libvirt, "virDomainMigrate2", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrate2",
                libvirt);
        return ret;
    }

    ret = virDomainMigrate2Symbol(domain,
                                  dconn,
                                  dxml,
                                  flags,
                                  dname,
                                  uri,
                                  bandwidth);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainMigrate3Type)(virDomainPtr domain,
                         virConnectPtr dconn,
                         virTypedParameterPtr params,
                         unsigned int nparams,
                         unsigned int flags);

virDomainPtr
virDomainMigrate3Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         virTypedParameterPtr params,
                         unsigned int nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virDomainMigrate3Type virDomainMigrate3Symbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrate3Symbol = libvirtSymbol(libvirt, "virDomainMigrate3", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrate3",
                libvirt);
        return ret;
    }

    ret = virDomainMigrate3Symbol(domain,
                                  dconn,
                                  params,
                                  nparams,
                                  flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateGetCompressionCacheType)(virDomainPtr domain,
                                           unsigned long long * cacheSize,
                                           unsigned int flags);

int
virDomainMigrateGetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long * cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    static virDomainMigrateGetCompressionCacheType virDomainMigrateGetCompressionCacheSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateGetCompressionCacheSymbol = libvirtSymbol(libvirt, "virDomainMigrateGetCompressionCache", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateGetCompressionCache",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateGetCompressionCacheSymbol(domain,
                                                    cacheSize,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateGetMaxDowntimeType)(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags);

int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    static virDomainMigrateGetMaxDowntimeType virDomainMigrateGetMaxDowntimeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateGetMaxDowntimeSymbol = libvirtSymbol(libvirt, "virDomainMigrateGetMaxDowntime", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateGetMaxDowntime",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateGetMaxDowntimeSymbol(domain,
                                               downtime,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateGetMaxSpeedType)(virDomainPtr domain,
                                   unsigned long * bandwidth,
                                   unsigned int flags);

int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long * bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainMigrateGetMaxSpeedType virDomainMigrateGetMaxSpeedSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateGetMaxSpeedSymbol = libvirtSymbol(libvirt, "virDomainMigrateGetMaxSpeed", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateGetMaxSpeed",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateGetMaxSpeedSymbol(domain,
                                            bandwidth,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateSetCompressionCacheType)(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags);

int
virDomainMigrateSetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    static virDomainMigrateSetCompressionCacheType virDomainMigrateSetCompressionCacheSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateSetCompressionCacheSymbol = libvirtSymbol(libvirt, "virDomainMigrateSetCompressionCache", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateSetCompressionCache",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateSetCompressionCacheSymbol(domain,
                                                    cacheSize,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateSetMaxDowntimeType)(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags);

int
virDomainMigrateSetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    static virDomainMigrateSetMaxDowntimeType virDomainMigrateSetMaxDowntimeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateSetMaxDowntimeSymbol = libvirtSymbol(libvirt, "virDomainMigrateSetMaxDowntime", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateSetMaxDowntime",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateSetMaxDowntimeSymbol(domain,
                                               downtime,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateSetMaxSpeedType)(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags);

int
virDomainMigrateSetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainMigrateSetMaxSpeedType virDomainMigrateSetMaxSpeedSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateSetMaxSpeedSymbol = libvirtSymbol(libvirt, "virDomainMigrateSetMaxSpeed", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateSetMaxSpeed",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateSetMaxSpeedSymbol(domain,
                                            bandwidth,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateStartPostCopyType)(virDomainPtr domain,
                                     unsigned int flags);

int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainMigrateStartPostCopyType virDomainMigrateStartPostCopySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateStartPostCopySymbol = libvirtSymbol(libvirt, "virDomainMigrateStartPostCopy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateStartPostCopy",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateStartPostCopySymbol(domain,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateToURIType)(virDomainPtr domain,
                             const char * duri,
                             unsigned long flags,
                             const char * dname,
                             unsigned long bandwidth);

int
virDomainMigrateToURIWrapper(virDomainPtr domain,
                             const char * duri,
                             unsigned long flags,
                             const char * dname,
                             unsigned long bandwidth,
                             virErrorPtr err)
{
    static virDomainMigrateToURIType virDomainMigrateToURISymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateToURISymbol = libvirtSymbol(libvirt, "virDomainMigrateToURI", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateToURI",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateToURISymbol(domain,
                                      duri,
                                      flags,
                                      dname,
                                      bandwidth);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateToURI2Type)(virDomainPtr domain,
                              const char * dconnuri,
                              const char * miguri,
                              const char * dxml,
                              unsigned long flags,
                              const char * dname,
                              unsigned long bandwidth);

int
virDomainMigrateToURI2Wrapper(virDomainPtr domain,
                              const char * dconnuri,
                              const char * miguri,
                              const char * dxml,
                              unsigned long flags,
                              const char * dname,
                              unsigned long bandwidth,
                              virErrorPtr err)
{
    static virDomainMigrateToURI2Type virDomainMigrateToURI2Symbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateToURI2Symbol = libvirtSymbol(libvirt, "virDomainMigrateToURI2", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateToURI2",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateToURI2Symbol(domain,
                                       dconnuri,
                                       miguri,
                                       dxml,
                                       flags,
                                       dname,
                                       bandwidth);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateToURI3Type)(virDomainPtr domain,
                              const char * dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags);

int
virDomainMigrateToURI3Wrapper(virDomainPtr domain,
                              const char * dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainMigrateToURI3Type virDomainMigrateToURI3Symbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainMigrateToURI3Symbol = libvirtSymbol(libvirt, "virDomainMigrateToURI3", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainMigrateToURI3",
                libvirt);
        return ret;
    }

    ret = virDomainMigrateToURI3Symbol(domain,
                                       dconnuri,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenChannelType)(virDomainPtr dom,
                            const char * name,
                            virStreamPtr st,
                            unsigned int flags);

int
virDomainOpenChannelWrapper(virDomainPtr dom,
                            const char * name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainOpenChannelType virDomainOpenChannelSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainOpenChannelSymbol = libvirtSymbol(libvirt, "virDomainOpenChannel", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainOpenChannel",
                libvirt);
        return ret;
    }

    ret = virDomainOpenChannelSymbol(dom,
                                     name,
                                     st,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenConsoleType)(virDomainPtr dom,
                            const char * dev_name,
                            virStreamPtr st,
                            unsigned int flags);

int
virDomainOpenConsoleWrapper(virDomainPtr dom,
                            const char * dev_name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainOpenConsoleType virDomainOpenConsoleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainOpenConsoleSymbol = libvirtSymbol(libvirt, "virDomainOpenConsole", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainOpenConsole",
                libvirt);
        return ret;
    }

    ret = virDomainOpenConsoleSymbol(dom,
                                     dev_name,
                                     st,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenGraphicsType)(virDomainPtr dom,
                             unsigned int idx,
                             int fd,
                             unsigned int flags);

int
virDomainOpenGraphicsWrapper(virDomainPtr dom,
                             unsigned int idx,
                             int fd,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainOpenGraphicsType virDomainOpenGraphicsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainOpenGraphicsSymbol = libvirtSymbol(libvirt, "virDomainOpenGraphics", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainOpenGraphics",
                libvirt);
        return ret;
    }

    ret = virDomainOpenGraphicsSymbol(dom,
                                      idx,
                                      fd,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenGraphicsFDType)(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags);

int
virDomainOpenGraphicsFDWrapper(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainOpenGraphicsFDType virDomainOpenGraphicsFDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainOpenGraphicsFDSymbol = libvirtSymbol(libvirt, "virDomainOpenGraphicsFD", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainOpenGraphicsFD",
                libvirt);
        return ret;
    }

    ret = virDomainOpenGraphicsFDSymbol(dom,
                                        idx,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPMSuspendForDurationType)(virDomainPtr dom,
                                     unsigned int target,
                                     unsigned long long duration,
                                     unsigned int flags);

int
virDomainPMSuspendForDurationWrapper(virDomainPtr dom,
                                     unsigned int target,
                                     unsigned long long duration,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainPMSuspendForDurationType virDomainPMSuspendForDurationSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainPMSuspendForDurationSymbol = libvirtSymbol(libvirt, "virDomainPMSuspendForDuration", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainPMSuspendForDuration",
                libvirt);
        return ret;
    }

    ret = virDomainPMSuspendForDurationSymbol(dom,
                                              target,
                                              duration,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPMWakeupType)(virDomainPtr dom,
                         unsigned int flags);

int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virDomainPMWakeupType virDomainPMWakeupSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainPMWakeupSymbol = libvirtSymbol(libvirt, "virDomainPMWakeup", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainPMWakeup",
                libvirt);
        return ret;
    }

    ret = virDomainPMWakeupSymbol(dom,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinEmulatorType)(virDomainPtr domain,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags);

int
virDomainPinEmulatorWrapper(virDomainPtr domain,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainPinEmulatorType virDomainPinEmulatorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainPinEmulatorSymbol = libvirtSymbol(libvirt, "virDomainPinEmulator", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainPinEmulator",
                libvirt);
        return ret;
    }

    ret = virDomainPinEmulatorSymbol(domain,
                                     cpumap,
                                     maplen,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinIOThreadType)(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags);

int
virDomainPinIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainPinIOThreadType virDomainPinIOThreadSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainPinIOThreadSymbol = libvirtSymbol(libvirt, "virDomainPinIOThread", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainPinIOThread",
                libvirt);
        return ret;
    }

    ret = virDomainPinIOThreadSymbol(domain,
                                     iothread_id,
                                     cpumap,
                                     maplen,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinVcpuType)(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char * cpumap,
                        int maplen);

int
virDomainPinVcpuWrapper(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char * cpumap,
                        int maplen,
                        virErrorPtr err)
{
    static virDomainPinVcpuType virDomainPinVcpuSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainPinVcpuSymbol = libvirtSymbol(libvirt, "virDomainPinVcpu", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainPinVcpu",
                libvirt);
        return ret;
    }

    ret = virDomainPinVcpuSymbol(domain,
                                 vcpu,
                                 cpumap,
                                 maplen);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinVcpuFlagsType)(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char * cpumap,
                             int maplen,
                             unsigned int flags);

int
virDomainPinVcpuFlagsWrapper(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char * cpumap,
                             int maplen,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainPinVcpuFlagsType virDomainPinVcpuFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainPinVcpuFlagsSymbol = libvirtSymbol(libvirt, "virDomainPinVcpuFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainPinVcpuFlags",
                libvirt);
        return ret;
    }

    ret = virDomainPinVcpuFlagsSymbol(domain,
                                      vcpu,
                                      cpumap,
                                      maplen,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRebootType)(virDomainPtr domain,
                       unsigned int flags);

int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err)
{
    static virDomainRebootType virDomainRebootSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainRebootSymbol = libvirtSymbol(libvirt, "virDomainReboot", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainReboot",
                libvirt);
        return ret;
    }

    ret = virDomainRebootSymbol(domain,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRefType)(virDomainPtr domain);

int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err)
{
    static virDomainRefType virDomainRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainRefSymbol = libvirtSymbol(libvirt, "virDomainRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainRef",
                libvirt);
        return ret;
    }

    ret = virDomainRefSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRenameType)(virDomainPtr dom,
                       const char * new_name,
                       unsigned int flags);

int
virDomainRenameWrapper(virDomainPtr dom,
                       const char * new_name,
                       unsigned int flags,
                       virErrorPtr err)
{
    static virDomainRenameType virDomainRenameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainRenameSymbol = libvirtSymbol(libvirt, "virDomainRename", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainRename",
                libvirt);
        return ret;
    }

    ret = virDomainRenameSymbol(dom,
                                new_name,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainResetType)(virDomainPtr domain,
                      unsigned int flags);

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err)
{
    static virDomainResetType virDomainResetSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainResetSymbol = libvirtSymbol(libvirt, "virDomainReset", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainReset",
                libvirt);
        return ret;
    }

    ret = virDomainResetSymbol(domain,
                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRestoreType)(virConnectPtr conn,
                        const char * from);

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char * from,
                        virErrorPtr err)
{
    static virDomainRestoreType virDomainRestoreSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainRestoreSymbol = libvirtSymbol(libvirt, "virDomainRestore", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainRestore",
                libvirt);
        return ret;
    }

    ret = virDomainRestoreSymbol(conn,
                                 from);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRestoreFlagsType)(virConnectPtr conn,
                             const char * from,
                             const char * dxml,
                             unsigned int flags);

int
virDomainRestoreFlagsWrapper(virConnectPtr conn,
                             const char * from,
                             const char * dxml,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virDomainRestoreFlagsType virDomainRestoreFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainRestoreFlagsSymbol = libvirtSymbol(libvirt, "virDomainRestoreFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainRestoreFlags",
                libvirt);
        return ret;
    }

    ret = virDomainRestoreFlagsSymbol(conn,
                                      from,
                                      dxml,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainResumeType)(virDomainPtr domain);

int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    static virDomainResumeType virDomainResumeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainResumeSymbol = libvirtSymbol(libvirt, "virDomainResume", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainResume",
                libvirt);
        return ret;
    }

    ret = virDomainResumeSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRevertToSnapshotType)(virDomainSnapshotPtr snapshot,
                                 unsigned int flags);

int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainRevertToSnapshotType virDomainRevertToSnapshotSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainRevertToSnapshotSymbol = libvirtSymbol(libvirt, "virDomainRevertToSnapshot", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainRevertToSnapshot",
                libvirt);
        return ret;
    }

    ret = virDomainRevertToSnapshotSymbol(snapshot,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveType)(virDomainPtr domain,
                     const char * to);

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char * to,
                     virErrorPtr err)
{
    static virDomainSaveType virDomainSaveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSaveSymbol = libvirtSymbol(libvirt, "virDomainSave", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSave",
                libvirt);
        return ret;
    }

    ret = virDomainSaveSymbol(domain,
                              to);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveFlagsType)(virDomainPtr domain,
                          const char * to,
                          const char * dxml,
                          unsigned int flags);

int
virDomainSaveFlagsWrapper(virDomainPtr domain,
                          const char * to,
                          const char * dxml,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virDomainSaveFlagsType virDomainSaveFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSaveFlagsSymbol = libvirtSymbol(libvirt, "virDomainSaveFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSaveFlags",
                libvirt);
        return ret;
    }

    ret = virDomainSaveFlagsSymbol(domain,
                                   to,
                                   dxml,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveImageDefineXMLType)(virConnectPtr conn,
                                   const char * file,
                                   const char * dxml,
                                   unsigned int flags);

int
virDomainSaveImageDefineXMLWrapper(virConnectPtr conn,
                                   const char * file,
                                   const char * dxml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainSaveImageDefineXMLType virDomainSaveImageDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSaveImageDefineXMLSymbol = libvirtSymbol(libvirt, "virDomainSaveImageDefineXML", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSaveImageDefineXML",
                libvirt);
        return ret;
    }

    ret = virDomainSaveImageDefineXMLSymbol(conn,
                                            file,
                                            dxml,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainSaveImageGetXMLDescType)(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags);

char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainSaveImageGetXMLDescType virDomainSaveImageGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSaveImageGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainSaveImageGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSaveImageGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virDomainSaveImageGetXMLDescSymbol(conn,
                                             file,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainScreenshotType)(virDomainPtr domain,
                           virStreamPtr stream,
                           unsigned int screen,
                           unsigned int flags);

char *
virDomainScreenshotWrapper(virDomainPtr domain,
                           virStreamPtr stream,
                           unsigned int screen,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virDomainScreenshotType virDomainScreenshotSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainScreenshotSymbol = libvirtSymbol(libvirt, "virDomainScreenshot", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainScreenshot",
                libvirt);
        return ret;
    }

    ret = virDomainScreenshotSymbol(domain,
                                    stream,
                                    screen,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSendKeyType)(virDomainPtr domain,
                        unsigned int codeset,
                        unsigned int holdtime,
                        unsigned int * keycodes,
                        int nkeycodes,
                        unsigned int flags);

int
virDomainSendKeyWrapper(virDomainPtr domain,
                        unsigned int codeset,
                        unsigned int holdtime,
                        unsigned int * keycodes,
                        int nkeycodes,
                        unsigned int flags,
                        virErrorPtr err)
{
    static virDomainSendKeyType virDomainSendKeySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSendKeySymbol = libvirtSymbol(libvirt, "virDomainSendKey", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSendKey",
                libvirt);
        return ret;
    }

    ret = virDomainSendKeySymbol(domain,
                                 codeset,
                                 holdtime,
                                 keycodes,
                                 nkeycodes,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSendProcessSignalType)(virDomainPtr domain,
                                  long long pid_value,
                                  unsigned int signum,
                                  unsigned int flags);

int
virDomainSendProcessSignalWrapper(virDomainPtr domain,
                                  long long pid_value,
                                  unsigned int signum,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSendProcessSignalType virDomainSendProcessSignalSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSendProcessSignalSymbol = libvirtSymbol(libvirt, "virDomainSendProcessSignal", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSendProcessSignal",
                libvirt);
        return ret;
    }

    ret = virDomainSendProcessSignalSymbol(domain,
                                           pid_value,
                                           signum,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetAutostartType)(virDomainPtr domain,
                             int autostart);

int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err)
{
    static virDomainSetAutostartType virDomainSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetAutostartSymbol = libvirtSymbol(libvirt, "virDomainSetAutostart", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetAutostart",
                libvirt);
        return ret;
    }

    ret = virDomainSetAutostartSymbol(domain,
                                      autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetBlkioParametersType)(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags);

int
virDomainSetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainSetBlkioParametersType virDomainSetBlkioParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetBlkioParametersSymbol = libvirtSymbol(libvirt, "virDomainSetBlkioParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetBlkioParameters",
                libvirt);
        return ret;
    }

    ret = virDomainSetBlkioParametersSymbol(domain,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetBlockIoTuneType)(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags);

int
virDomainSetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainSetBlockIoTuneType virDomainSetBlockIoTuneSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetBlockIoTuneSymbol = libvirtSymbol(libvirt, "virDomainSetBlockIoTune", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetBlockIoTune",
                libvirt);
        return ret;
    }

    ret = virDomainSetBlockIoTuneSymbol(dom,
                                        disk,
                                        params,
                                        nparams,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetBlockThresholdType)(virDomainPtr domain,
                                  const char * dev,
                                  unsigned long long threshold,
                                  unsigned int flags);

int
virDomainSetBlockThresholdWrapper(virDomainPtr domain,
                                  const char * dev,
                                  unsigned long long threshold,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSetBlockThresholdType virDomainSetBlockThresholdSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetBlockThresholdSymbol = libvirtSymbol(libvirt, "virDomainSetBlockThreshold", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetBlockThreshold",
                libvirt);
        return ret;
    }

    ret = virDomainSetBlockThresholdSymbol(domain,
                                           dev,
                                           threshold,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetGuestVcpusType)(virDomainPtr domain,
                              const char * cpumap,
                              int state,
                              unsigned int flags);

int
virDomainSetGuestVcpusWrapper(virDomainPtr domain,
                              const char * cpumap,
                              int state,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainSetGuestVcpusType virDomainSetGuestVcpusSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetGuestVcpusSymbol = libvirtSymbol(libvirt, "virDomainSetGuestVcpus", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetGuestVcpus",
                libvirt);
        return ret;
    }

    ret = virDomainSetGuestVcpusSymbol(domain,
                                       cpumap,
                                       state,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetIOThreadParamsType)(virDomainPtr domain,
                                  unsigned int iothread_id,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags);

int
virDomainSetIOThreadParamsWrapper(virDomainPtr domain,
                                  unsigned int iothread_id,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSetIOThreadParamsType virDomainSetIOThreadParamsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetIOThreadParamsSymbol = libvirtSymbol(libvirt, "virDomainSetIOThreadParams", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetIOThreadParams",
                libvirt);
        return ret;
    }

    ret = virDomainSetIOThreadParamsSymbol(domain,
                                           iothread_id,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetInterfaceParametersType)(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags);

int
virDomainSetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    static virDomainSetInterfaceParametersType virDomainSetInterfaceParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetInterfaceParametersSymbol = libvirtSymbol(libvirt, "virDomainSetInterfaceParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetInterfaceParameters",
                libvirt);
        return ret;
    }

    ret = virDomainSetInterfaceParametersSymbol(domain,
                                                device,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetLifecycleActionType)(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int action,
                                   unsigned int flags);

int
virDomainSetLifecycleActionWrapper(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int action,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainSetLifecycleActionType virDomainSetLifecycleActionSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetLifecycleActionSymbol = libvirtSymbol(libvirt, "virDomainSetLifecycleAction", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetLifecycleAction",
                libvirt);
        return ret;
    }

    ret = virDomainSetLifecycleActionSymbol(domain,
                                            type,
                                            action,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMaxMemoryType)(virDomainPtr domain,
                             unsigned long memory);

int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err)
{
    static virDomainSetMaxMemoryType virDomainSetMaxMemorySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetMaxMemorySymbol = libvirtSymbol(libvirt, "virDomainSetMaxMemory", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetMaxMemory",
                libvirt);
        return ret;
    }

    ret = virDomainSetMaxMemorySymbol(domain,
                                      memory);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryType)(virDomainPtr domain,
                          unsigned long memory);

int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err)
{
    static virDomainSetMemoryType virDomainSetMemorySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetMemorySymbol = libvirtSymbol(libvirt, "virDomainSetMemory", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetMemory",
                libvirt);
        return ret;
    }

    ret = virDomainSetMemorySymbol(domain,
                                   memory);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryFlagsType)(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags);

int
virDomainSetMemoryFlagsWrapper(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainSetMemoryFlagsType virDomainSetMemoryFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetMemoryFlagsSymbol = libvirtSymbol(libvirt, "virDomainSetMemoryFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetMemoryFlags",
                libvirt);
        return ret;
    }

    ret = virDomainSetMemoryFlagsSymbol(domain,
                                        memory,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryParametersType)(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int nparams,
                                    unsigned int flags);

int
virDomainSetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainSetMemoryParametersType virDomainSetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetMemoryParametersSymbol = libvirtSymbol(libvirt, "virDomainSetMemoryParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetMemoryParameters",
                libvirt);
        return ret;
    }

    ret = virDomainSetMemoryParametersSymbol(domain,
                                             params,
                                             nparams,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryStatsPeriodType)(virDomainPtr domain,
                                     int period,
                                     unsigned int flags);

int
virDomainSetMemoryStatsPeriodWrapper(virDomainPtr domain,
                                     int period,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainSetMemoryStatsPeriodType virDomainSetMemoryStatsPeriodSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetMemoryStatsPeriodSymbol = libvirtSymbol(libvirt, "virDomainSetMemoryStatsPeriod", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetMemoryStatsPeriod",
                libvirt);
        return ret;
    }

    ret = virDomainSetMemoryStatsPeriodSymbol(domain,
                                              period,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMetadataType)(virDomainPtr domain,
                            int type,
                            const char * metadata,
                            const char * key,
                            const char * uri,
                            unsigned int flags);

int
virDomainSetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char * metadata,
                            const char * key,
                            const char * uri,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainSetMetadataType virDomainSetMetadataSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetMetadataSymbol = libvirtSymbol(libvirt, "virDomainSetMetadata", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetMetadata",
                libvirt);
        return ret;
    }

    ret = virDomainSetMetadataSymbol(domain,
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
(*virDomainSetNumaParametersType)(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags);

int
virDomainSetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSetNumaParametersType virDomainSetNumaParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetNumaParametersSymbol = libvirtSymbol(libvirt, "virDomainSetNumaParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetNumaParameters",
                libvirt);
        return ret;
    }

    ret = virDomainSetNumaParametersSymbol(domain,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetPerfEventsType)(virDomainPtr domain,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags);

int
virDomainSetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainSetPerfEventsType virDomainSetPerfEventsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetPerfEventsSymbol = libvirtSymbol(libvirt, "virDomainSetPerfEvents", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetPerfEvents",
                libvirt);
        return ret;
    }

    ret = virDomainSetPerfEventsSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetSchedulerParametersType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams);

int
virDomainSetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       virErrorPtr err)
{
    static virDomainSetSchedulerParametersType virDomainSetSchedulerParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetSchedulerParametersSymbol = libvirtSymbol(libvirt, "virDomainSetSchedulerParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetSchedulerParameters",
                libvirt);
        return ret;
    }

    ret = virDomainSetSchedulerParametersSymbol(domain,
                                                params,
                                                nparams);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetSchedulerParametersFlagsType)(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int nparams,
                                            unsigned int flags);

int
virDomainSetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    static virDomainSetSchedulerParametersFlagsType virDomainSetSchedulerParametersFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetSchedulerParametersFlagsSymbol = libvirtSymbol(libvirt, "virDomainSetSchedulerParametersFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetSchedulerParametersFlags",
                libvirt);
        return ret;
    }

    ret = virDomainSetSchedulerParametersFlagsSymbol(domain,
                                                     params,
                                                     nparams,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetTimeType)(virDomainPtr dom,
                        long long seconds,
                        unsigned int nseconds,
                        unsigned int flags);

int
virDomainSetTimeWrapper(virDomainPtr dom,
                        long long seconds,
                        unsigned int nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
    static virDomainSetTimeType virDomainSetTimeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetTimeSymbol = libvirtSymbol(libvirt, "virDomainSetTime", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetTime",
                libvirt);
        return ret;
    }

    ret = virDomainSetTimeSymbol(dom,
                                 seconds,
                                 nseconds,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetUserPasswordType)(virDomainPtr dom,
                                const char * user,
                                const char * password,
                                unsigned int flags);

int
virDomainSetUserPasswordWrapper(virDomainPtr dom,
                                const char * user,
                                const char * password,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainSetUserPasswordType virDomainSetUserPasswordSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetUserPasswordSymbol = libvirtSymbol(libvirt, "virDomainSetUserPassword", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetUserPassword",
                libvirt);
        return ret;
    }

    ret = virDomainSetUserPasswordSymbol(dom,
                                         user,
                                         password,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetVcpuType)(virDomainPtr domain,
                        const char * vcpumap,
                        int state,
                        unsigned int flags);

int
virDomainSetVcpuWrapper(virDomainPtr domain,
                        const char * vcpumap,
                        int state,
                        unsigned int flags,
                        virErrorPtr err)
{
    static virDomainSetVcpuType virDomainSetVcpuSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetVcpuSymbol = libvirtSymbol(libvirt, "virDomainSetVcpu", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetVcpu",
                libvirt);
        return ret;
    }

    ret = virDomainSetVcpuSymbol(domain,
                                 vcpumap,
                                 state,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetVcpusType)(virDomainPtr domain,
                         unsigned int nvcpus);

int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err)
{
    static virDomainSetVcpusType virDomainSetVcpusSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetVcpusSymbol = libvirtSymbol(libvirt, "virDomainSetVcpus", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetVcpus",
                libvirt);
        return ret;
    }

    ret = virDomainSetVcpusSymbol(domain,
                                  nvcpus);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetVcpusFlagsType)(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags);

int
virDomainSetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainSetVcpusFlagsType virDomainSetVcpusFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSetVcpusFlagsSymbol = libvirtSymbol(libvirt, "virDomainSetVcpusFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSetVcpusFlags",
                libvirt);
        return ret;
    }

    ret = virDomainSetVcpusFlagsSymbol(domain,
                                       nvcpus,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainShutdownType)(virDomainPtr domain);

int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    static virDomainShutdownType virDomainShutdownSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainShutdownSymbol = libvirtSymbol(libvirt, "virDomainShutdown", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainShutdown",
                libvirt);
        return ret;
    }

    ret = virDomainShutdownSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainShutdownFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainShutdownFlagsType virDomainShutdownFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainShutdownFlagsSymbol = libvirtSymbol(libvirt, "virDomainShutdownFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainShutdownFlags",
                libvirt);
        return ret;
    }

    ret = virDomainShutdownFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotCreateXMLType)(virDomainPtr domain,
                                  const char * xmlDesc,
                                  unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCreateXMLWrapper(virDomainPtr domain,
                                  const char * xmlDesc,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSnapshotCreateXMLType virDomainSnapshotCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotCreateXMLSymbol = libvirtSymbol(libvirt, "virDomainSnapshotCreateXML", &success);
    }

    virDomainSnapshotPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotCreateXML",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotCreateXMLSymbol(domain,
                                           xmlDesc,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotCurrentType)(virDomainPtr domain,
                                unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virDomainSnapshotCurrentType virDomainSnapshotCurrentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotCurrentSymbol = libvirtSymbol(libvirt, "virDomainSnapshotCurrent", &success);
    }

    virDomainSnapshotPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotCurrent",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotCurrentSymbol(domain,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotDeleteType)(virDomainSnapshotPtr snapshot,
                               unsigned int flags);

int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainSnapshotDeleteType virDomainSnapshotDeleteSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotDeleteSymbol = libvirtSymbol(libvirt, "virDomainSnapshotDelete", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotDelete",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotDeleteSymbol(snapshot,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotFreeType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err)
{
    static virDomainSnapshotFreeType virDomainSnapshotFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotFreeSymbol = libvirtSymbol(libvirt, "virDomainSnapshotFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotFree",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotFreeSymbol(snapshot);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virDomainSnapshotGetConnectType)(virDomainSnapshotPtr snapshot);

virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err)
{
    static virDomainSnapshotGetConnectType virDomainSnapshotGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotGetConnectSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotGetConnect",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotGetConnectSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainSnapshotGetDomainType)(virDomainSnapshotPtr snapshot);

virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err)
{
    static virDomainSnapshotGetDomainType virDomainSnapshotGetDomainSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotGetDomainSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetDomain", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotGetDomain",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotGetDomainSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virDomainSnapshotGetNameType)(virDomainSnapshotPtr snapshot);

const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err)
{
    static virDomainSnapshotGetNameType virDomainSnapshotGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotGetNameSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotGetName",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotGetNameSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotGetParentType)(virDomainSnapshotPtr snapshot,
                                  unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSnapshotGetParentType virDomainSnapshotGetParentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotGetParentSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetParent", &success);
    }

    virDomainSnapshotPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotGetParent",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotGetParentSymbol(snapshot,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainSnapshotGetXMLDescType)(virDomainSnapshotPtr snapshot,
                                   unsigned int flags);

char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainSnapshotGetXMLDescType virDomainSnapshotGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotGetXMLDescSymbol(snapshot,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotHasMetadataType)(virDomainSnapshotPtr snapshot,
                                    unsigned int flags);

int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainSnapshotHasMetadataType virDomainSnapshotHasMetadataSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotHasMetadataSymbol = libvirtSymbol(libvirt, "virDomainSnapshotHasMetadata", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotHasMetadata",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotHasMetadataSymbol(snapshot,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotIsCurrentType)(virDomainSnapshotPtr snapshot,
                                  unsigned int flags);

int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSnapshotIsCurrentType virDomainSnapshotIsCurrentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotIsCurrentSymbol = libvirtSymbol(libvirt, "virDomainSnapshotIsCurrent", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotIsCurrent",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotIsCurrentSymbol(snapshot,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotListAllChildrenType)(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags);

int
virDomainSnapshotListAllChildrenWrapper(virDomainSnapshotPtr snapshot,
                                        virDomainSnapshotPtr ** snaps,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    static virDomainSnapshotListAllChildrenType virDomainSnapshotListAllChildrenSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotListAllChildrenSymbol = libvirtSymbol(libvirt, "virDomainSnapshotListAllChildren", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotListAllChildren",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotListAllChildrenSymbol(snapshot,
                                                 snaps,
                                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotListChildrenNamesType)(virDomainSnapshotPtr snapshot,
                                          char ** names,
                                          int nameslen,
                                          unsigned int flags);

int
virDomainSnapshotListChildrenNamesWrapper(virDomainSnapshotPtr snapshot,
                                          char ** names,
                                          int nameslen,
                                          unsigned int flags,
                                          virErrorPtr err)
{
    static virDomainSnapshotListChildrenNamesType virDomainSnapshotListChildrenNamesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotListChildrenNamesSymbol = libvirtSymbol(libvirt, "virDomainSnapshotListChildrenNames", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotListChildrenNames",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotListChildrenNamesSymbol(snapshot,
                                                   names,
                                                   nameslen,
                                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotListNamesType)(virDomainPtr domain,
                                  char ** names,
                                  int nameslen,
                                  unsigned int flags);

int
virDomainSnapshotListNamesWrapper(virDomainPtr domain,
                                  char ** names,
                                  int nameslen,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainSnapshotListNamesType virDomainSnapshotListNamesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotListNamesSymbol = libvirtSymbol(libvirt, "virDomainSnapshotListNames", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotListNames",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotListNamesSymbol(domain,
                                           names,
                                           nameslen,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotLookupByNameType)(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotLookupByNameWrapper(virDomainPtr domain,
                                     const char * name,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    static virDomainSnapshotLookupByNameType virDomainSnapshotLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotLookupByNameSymbol = libvirtSymbol(libvirt, "virDomainSnapshotLookupByName", &success);
    }

    virDomainSnapshotPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotLookupByName",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotLookupByNameSymbol(domain,
                                              name,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotNumType)(virDomainPtr domain,
                            unsigned int flags);

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virDomainSnapshotNumType virDomainSnapshotNumSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotNumSymbol = libvirtSymbol(libvirt, "virDomainSnapshotNum", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotNum",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotNumSymbol(domain,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotNumChildrenType)(virDomainSnapshotPtr snapshot,
                                    unsigned int flags);

int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virDomainSnapshotNumChildrenType virDomainSnapshotNumChildrenSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotNumChildrenSymbol = libvirtSymbol(libvirt, "virDomainSnapshotNumChildren", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotNumChildren",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotNumChildrenSymbol(snapshot,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSnapshotRefType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err)
{
    static virDomainSnapshotRefType virDomainSnapshotRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSnapshotRefSymbol = libvirtSymbol(libvirt, "virDomainSnapshotRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSnapshotRef",
                libvirt);
        return ret;
    }

    ret = virDomainSnapshotRefSymbol(snapshot);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainStartDirtyRateCalcType)(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags);

int
virDomainStartDirtyRateCalcWrapper(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainStartDirtyRateCalcType virDomainStartDirtyRateCalcSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainStartDirtyRateCalcSymbol = libvirtSymbol(libvirt, "virDomainStartDirtyRateCalc", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainStartDirtyRateCalc",
                libvirt);
        return ret;
    }

    ret = virDomainStartDirtyRateCalcSymbol(domain,
                                            seconds,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainStatsRecordListFreeType)(virDomainStatsRecordPtr * stats);

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats)
{
    static virDomainStatsRecordListFreeType virDomainStatsRecordListFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainStatsRecordListFreeSymbol = libvirtSymbol(libvirt, "virDomainStatsRecordListFree", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainStatsRecordListFree",
                libvirt);
        return;
    }

    virDomainStatsRecordListFreeSymbol(stats);
}

typedef int
(*virDomainSuspendType)(virDomainPtr domain);

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    static virDomainSuspendType virDomainSuspendSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainSuspendSymbol = libvirtSymbol(libvirt, "virDomainSuspend", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainSuspend",
                libvirt);
        return ret;
    }

    ret = virDomainSuspendSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainUndefineType)(virDomainPtr domain);

int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    static virDomainUndefineType virDomainUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainUndefineSymbol = libvirtSymbol(libvirt, "virDomainUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainUndefine",
                libvirt);
        return ret;
    }

    ret = virDomainUndefineSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainUndefineFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virDomainUndefineFlagsType virDomainUndefineFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainUndefineFlagsSymbol = libvirtSymbol(libvirt, "virDomainUndefineFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainUndefineFlags",
                libvirt);
        return ret;
    }

    ret = virDomainUndefineFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainUpdateDeviceFlagsType)(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags);

int
virDomainUpdateDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainUpdateDeviceFlagsType virDomainUpdateDeviceFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainUpdateDeviceFlagsSymbol = libvirtSymbol(libvirt, "virDomainUpdateDeviceFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainUpdateDeviceFlags",
                libvirt);
        return ret;
    }

    ret = virDomainUpdateDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventAddHandleType)(int fd,
                         int events,
                         virEventHandleCallback cb,
                         void * opaque,
                         virFreeCallback ff);

int
virEventAddHandleWrapper(int fd,
                         int events,
                         virEventHandleCallback cb,
                         void * opaque,
                         virFreeCallback ff,
                         virErrorPtr err)
{
    static virEventAddHandleType virEventAddHandleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventAddHandleSymbol = libvirtSymbol(libvirt, "virEventAddHandle", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virEventAddHandle",
                libvirt);
        return ret;
    }

    ret = virEventAddHandleSymbol(fd,
                                  events,
                                  cb,
                                  opaque,
                                  ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventAddTimeoutType)(int timeout,
                          virEventTimeoutCallback cb,
                          void * opaque,
                          virFreeCallback ff);

int
virEventAddTimeoutWrapper(int timeout,
                          virEventTimeoutCallback cb,
                          void * opaque,
                          virFreeCallback ff,
                          virErrorPtr err)
{
    static virEventAddTimeoutType virEventAddTimeoutSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventAddTimeoutSymbol = libvirtSymbol(libvirt, "virEventAddTimeout", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virEventAddTimeout",
                libvirt);
        return ret;
    }

    ret = virEventAddTimeoutSymbol(timeout,
                                   cb,
                                   opaque,
                                   ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventRegisterDefaultImplType)(void);

int
virEventRegisterDefaultImplWrapper(virErrorPtr err)
{
    static virEventRegisterDefaultImplType virEventRegisterDefaultImplSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventRegisterDefaultImplSymbol = libvirtSymbol(libvirt, "virEventRegisterDefaultImpl", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virEventRegisterDefaultImpl",
                libvirt);
        return ret;
    }

    ret = virEventRegisterDefaultImplSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virEventRegisterImplType)(virEventAddHandleFunc addHandle,
                            virEventUpdateHandleFunc updateHandle,
                            virEventRemoveHandleFunc removeHandle,
                            virEventAddTimeoutFunc addTimeout,
                            virEventUpdateTimeoutFunc updateTimeout,
                            virEventRemoveTimeoutFunc removeTimeout);

void
virEventRegisterImplWrapper(virEventAddHandleFunc addHandle,
                            virEventUpdateHandleFunc updateHandle,
                            virEventRemoveHandleFunc removeHandle,
                            virEventAddTimeoutFunc addTimeout,
                            virEventUpdateTimeoutFunc updateTimeout,
                            virEventRemoveTimeoutFunc removeTimeout)
{
    static virEventRegisterImplType virEventRegisterImplSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventRegisterImplSymbol = libvirtSymbol(libvirt, "virEventRegisterImpl", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virEventRegisterImpl",
                libvirt);
        return;
    }

    virEventRegisterImplSymbol(addHandle,
                               updateHandle,
                               removeHandle,
                               addTimeout,
                               updateTimeout,
                               removeTimeout);
}

typedef int
(*virEventRemoveHandleType)(int watch);

int
virEventRemoveHandleWrapper(int watch,
                            virErrorPtr err)
{
    static virEventRemoveHandleType virEventRemoveHandleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventRemoveHandleSymbol = libvirtSymbol(libvirt, "virEventRemoveHandle", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virEventRemoveHandle",
                libvirt);
        return ret;
    }

    ret = virEventRemoveHandleSymbol(watch);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventRemoveTimeoutType)(int timer);

int
virEventRemoveTimeoutWrapper(int timer,
                             virErrorPtr err)
{
    static virEventRemoveTimeoutType virEventRemoveTimeoutSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventRemoveTimeoutSymbol = libvirtSymbol(libvirt, "virEventRemoveTimeout", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virEventRemoveTimeout",
                libvirt);
        return ret;
    }

    ret = virEventRemoveTimeoutSymbol(timer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventRunDefaultImplType)(void);

int
virEventRunDefaultImplWrapper(virErrorPtr err)
{
    static virEventRunDefaultImplType virEventRunDefaultImplSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventRunDefaultImplSymbol = libvirtSymbol(libvirt, "virEventRunDefaultImpl", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virEventRunDefaultImpl",
                libvirt);
        return ret;
    }

    ret = virEventRunDefaultImplSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virEventUpdateHandleType)(int watch,
                            int events);

void
virEventUpdateHandleWrapper(int watch,
                            int events)
{
    static virEventUpdateHandleType virEventUpdateHandleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventUpdateHandleSymbol = libvirtSymbol(libvirt, "virEventUpdateHandle", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virEventUpdateHandle",
                libvirt);
        return;
    }

    virEventUpdateHandleSymbol(watch,
                               events);
}

typedef void
(*virEventUpdateTimeoutType)(int timer,
                             int timeout);

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout)
{
    static virEventUpdateTimeoutType virEventUpdateTimeoutSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virEventUpdateTimeoutSymbol = libvirtSymbol(libvirt, "virEventUpdateTimeout", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virEventUpdateTimeout",
                libvirt);
        return;
    }

    virEventUpdateTimeoutSymbol(timer,
                                timeout);
}

typedef void
(*virFreeErrorType)(virErrorPtr err);

void
virFreeErrorWrapper(virErrorPtr err)
{
    static virFreeErrorType virFreeErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virFreeErrorSymbol = libvirtSymbol(libvirt, "virFreeError", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virFreeError",
                libvirt);
        return;
    }

    virFreeErrorSymbol(err);
}

typedef virErrorPtr
(*virGetLastErrorType)(void);

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err)
{
    static virGetLastErrorType virGetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virGetLastErrorSymbol = libvirtSymbol(libvirt, "virGetLastError", &success);
    }

    virErrorPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virGetLastError",
                libvirt);
        return ret;
    }

    ret = virGetLastErrorSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virGetLastErrorCodeType)(void);

int
virGetLastErrorCodeWrapper(virErrorPtr err)
{
    static virGetLastErrorCodeType virGetLastErrorCodeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virGetLastErrorCodeSymbol = libvirtSymbol(libvirt, "virGetLastErrorCode", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virGetLastErrorCode",
                libvirt);
        return ret;
    }

    ret = virGetLastErrorCodeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virGetLastErrorDomainType)(void);

int
virGetLastErrorDomainWrapper(virErrorPtr err)
{
    static virGetLastErrorDomainType virGetLastErrorDomainSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virGetLastErrorDomainSymbol = libvirtSymbol(libvirt, "virGetLastErrorDomain", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virGetLastErrorDomain",
                libvirt);
        return ret;
    }

    ret = virGetLastErrorDomainSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virGetLastErrorMessageType)(void);

const char *
virGetLastErrorMessageWrapper(virErrorPtr err)
{
    static virGetLastErrorMessageType virGetLastErrorMessageSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virGetLastErrorMessageSymbol = libvirtSymbol(libvirt, "virGetLastErrorMessage", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virGetLastErrorMessage",
                libvirt);
        return ret;
    }

    ret = virGetLastErrorMessageSymbol();
    if (!ret) {
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
    static virGetVersionType virGetVersionSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virGetVersionSymbol = libvirtSymbol(libvirt, "virGetVersion", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virGetVersion",
                libvirt);
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
    static virInitializeType virInitializeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInitializeSymbol = libvirtSymbol(libvirt, "virInitialize", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInitialize",
                libvirt);
        return ret;
    }

    ret = virInitializeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceChangeBeginType)(virConnectPtr conn,
                               unsigned int flags);

int
virInterfaceChangeBeginWrapper(virConnectPtr conn,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virInterfaceChangeBeginType virInterfaceChangeBeginSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceChangeBeginSymbol = libvirtSymbol(libvirt, "virInterfaceChangeBegin", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceChangeBegin",
                libvirt);
        return ret;
    }

    ret = virInterfaceChangeBeginSymbol(conn,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceChangeCommitType)(virConnectPtr conn,
                                unsigned int flags);

int
virInterfaceChangeCommitWrapper(virConnectPtr conn,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virInterfaceChangeCommitType virInterfaceChangeCommitSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceChangeCommitSymbol = libvirtSymbol(libvirt, "virInterfaceChangeCommit", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceChangeCommit",
                libvirt);
        return ret;
    }

    ret = virInterfaceChangeCommitSymbol(conn,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceChangeRollbackType)(virConnectPtr conn,
                                  unsigned int flags);

int
virInterfaceChangeRollbackWrapper(virConnectPtr conn,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virInterfaceChangeRollbackType virInterfaceChangeRollbackSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceChangeRollbackSymbol = libvirtSymbol(libvirt, "virInterfaceChangeRollback", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceChangeRollback",
                libvirt);
        return ret;
    }

    ret = virInterfaceChangeRollbackSymbol(conn,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceCreateType)(virInterfacePtr iface,
                          unsigned int flags);

int
virInterfaceCreateWrapper(virInterfacePtr iface,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virInterfaceCreateType virInterfaceCreateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceCreateSymbol = libvirtSymbol(libvirt, "virInterfaceCreate", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceCreate",
                libvirt);
        return ret;
    }

    ret = virInterfaceCreateSymbol(iface,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virInterfacePtr
(*virInterfaceDefineXMLType)(virConnectPtr conn,
                             const char * xml,
                             unsigned int flags);

virInterfacePtr
virInterfaceDefineXMLWrapper(virConnectPtr conn,
                             const char * xml,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virInterfaceDefineXMLType virInterfaceDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceDefineXMLSymbol = libvirtSymbol(libvirt, "virInterfaceDefineXML", &success);
    }

    virInterfacePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceDefineXML",
                libvirt);
        return ret;
    }

    ret = virInterfaceDefineXMLSymbol(conn,
                                      xml,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceDestroyType)(virInterfacePtr iface,
                           unsigned int flags);

int
virInterfaceDestroyWrapper(virInterfacePtr iface,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virInterfaceDestroyType virInterfaceDestroySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceDestroySymbol = libvirtSymbol(libvirt, "virInterfaceDestroy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceDestroy",
                libvirt);
        return ret;
    }

    ret = virInterfaceDestroySymbol(iface,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceFreeType)(virInterfacePtr iface);

int
virInterfaceFreeWrapper(virInterfacePtr iface,
                        virErrorPtr err)
{
    static virInterfaceFreeType virInterfaceFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceFreeSymbol = libvirtSymbol(libvirt, "virInterfaceFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceFree",
                libvirt);
        return ret;
    }

    ret = virInterfaceFreeSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virInterfaceGetConnectType)(virInterfacePtr iface);

virConnectPtr
virInterfaceGetConnectWrapper(virInterfacePtr iface,
                              virErrorPtr err)
{
    static virInterfaceGetConnectType virInterfaceGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceGetConnectSymbol = libvirtSymbol(libvirt, "virInterfaceGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceGetConnect",
                libvirt);
        return ret;
    }

    ret = virInterfaceGetConnectSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virInterfaceGetMACStringType)(virInterfacePtr iface);

const char *
virInterfaceGetMACStringWrapper(virInterfacePtr iface,
                                virErrorPtr err)
{
    static virInterfaceGetMACStringType virInterfaceGetMACStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceGetMACStringSymbol = libvirtSymbol(libvirt, "virInterfaceGetMACString", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceGetMACString",
                libvirt);
        return ret;
    }

    ret = virInterfaceGetMACStringSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virInterfaceGetNameType)(virInterfacePtr iface);

const char *
virInterfaceGetNameWrapper(virInterfacePtr iface,
                           virErrorPtr err)
{
    static virInterfaceGetNameType virInterfaceGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceGetNameSymbol = libvirtSymbol(libvirt, "virInterfaceGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceGetName",
                libvirt);
        return ret;
    }

    ret = virInterfaceGetNameSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virInterfaceGetXMLDescType)(virInterfacePtr iface,
                              unsigned int flags);

char *
virInterfaceGetXMLDescWrapper(virInterfacePtr iface,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virInterfaceGetXMLDescType virInterfaceGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceGetXMLDescSymbol = libvirtSymbol(libvirt, "virInterfaceGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virInterfaceGetXMLDescSymbol(iface,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceIsActiveType)(virInterfacePtr iface);

int
virInterfaceIsActiveWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    static virInterfaceIsActiveType virInterfaceIsActiveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceIsActiveSymbol = libvirtSymbol(libvirt, "virInterfaceIsActive", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceIsActive",
                libvirt);
        return ret;
    }

    ret = virInterfaceIsActiveSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virInterfacePtr
(*virInterfaceLookupByMACStringType)(virConnectPtr conn,
                                     const char * macstr);

virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char * macstr,
                                     virErrorPtr err)
{
    static virInterfaceLookupByMACStringType virInterfaceLookupByMACStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceLookupByMACStringSymbol = libvirtSymbol(libvirt, "virInterfaceLookupByMACString", &success);
    }

    virInterfacePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceLookupByMACString",
                libvirt);
        return ret;
    }

    ret = virInterfaceLookupByMACStringSymbol(conn,
                                              macstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virInterfacePtr
(*virInterfaceLookupByNameType)(virConnectPtr conn,
                                const char * name);

virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char * name,
                                virErrorPtr err)
{
    static virInterfaceLookupByNameType virInterfaceLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceLookupByNameSymbol = libvirtSymbol(libvirt, "virInterfaceLookupByName", &success);
    }

    virInterfacePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceLookupByName",
                libvirt);
        return ret;
    }

    ret = virInterfaceLookupByNameSymbol(conn,
                                         name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceRefType)(virInterfacePtr iface);

int
virInterfaceRefWrapper(virInterfacePtr iface,
                       virErrorPtr err)
{
    static virInterfaceRefType virInterfaceRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceRefSymbol = libvirtSymbol(libvirt, "virInterfaceRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceRef",
                libvirt);
        return ret;
    }

    ret = virInterfaceRefSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virInterfaceUndefineType)(virInterfacePtr iface);

int
virInterfaceUndefineWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    static virInterfaceUndefineType virInterfaceUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virInterfaceUndefineSymbol = libvirtSymbol(libvirt, "virInterfaceUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virInterfaceUndefine",
                libvirt);
        return ret;
    }

    ret = virInterfaceUndefineSymbol(iface);
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
    static virNWFilterBindingCreateXMLType virNWFilterBindingCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingCreateXMLSymbol = libvirtSymbol(libvirt, "virNWFilterBindingCreateXML", &success);
    }

    virNWFilterBindingPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingCreateXML",
                libvirt);
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
    static virNWFilterBindingDeleteType virNWFilterBindingDeleteSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingDeleteSymbol = libvirtSymbol(libvirt, "virNWFilterBindingDelete", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingDelete",
                libvirt);
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
    static virNWFilterBindingFreeType virNWFilterBindingFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingFreeSymbol = libvirtSymbol(libvirt, "virNWFilterBindingFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingFree",
                libvirt);
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
    static virNWFilterBindingGetFilterNameType virNWFilterBindingGetFilterNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingGetFilterNameSymbol = libvirtSymbol(libvirt, "virNWFilterBindingGetFilterName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingGetFilterName",
                libvirt);
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
    static virNWFilterBindingGetPortDevType virNWFilterBindingGetPortDevSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingGetPortDevSymbol = libvirtSymbol(libvirt, "virNWFilterBindingGetPortDev", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingGetPortDev",
                libvirt);
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
    static virNWFilterBindingGetXMLDescType virNWFilterBindingGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingGetXMLDescSymbol = libvirtSymbol(libvirt, "virNWFilterBindingGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingGetXMLDesc",
                libvirt);
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
    static virNWFilterBindingLookupByPortDevType virNWFilterBindingLookupByPortDevSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingLookupByPortDevSymbol = libvirtSymbol(libvirt, "virNWFilterBindingLookupByPortDev", &success);
    }

    virNWFilterBindingPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingLookupByPortDev",
                libvirt);
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
    static virNWFilterBindingRefType virNWFilterBindingRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterBindingRefSymbol = libvirtSymbol(libvirt, "virNWFilterBindingRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterBindingRef",
                libvirt);
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
    static virNWFilterDefineXMLType virNWFilterDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterDefineXMLSymbol = libvirtSymbol(libvirt, "virNWFilterDefineXML", &success);
    }

    virNWFilterPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterDefineXML",
                libvirt);
        return ret;
    }

    ret = virNWFilterDefineXMLSymbol(conn,
                                     xmlDesc);
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
    static virNWFilterFreeType virNWFilterFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterFreeSymbol = libvirtSymbol(libvirt, "virNWFilterFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterFree",
                libvirt);
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
    static virNWFilterGetNameType virNWFilterGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterGetNameSymbol = libvirtSymbol(libvirt, "virNWFilterGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterGetName",
                libvirt);
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
    static virNWFilterGetUUIDType virNWFilterGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterGetUUIDSymbol = libvirtSymbol(libvirt, "virNWFilterGetUUID", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterGetUUID",
                libvirt);
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
    static virNWFilterGetUUIDStringType virNWFilterGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterGetUUIDStringSymbol = libvirtSymbol(libvirt, "virNWFilterGetUUIDString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterGetUUIDString",
                libvirt);
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
    static virNWFilterGetXMLDescType virNWFilterGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterGetXMLDescSymbol = libvirtSymbol(libvirt, "virNWFilterGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterGetXMLDesc",
                libvirt);
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
    static virNWFilterLookupByNameType virNWFilterLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterLookupByNameSymbol = libvirtSymbol(libvirt, "virNWFilterLookupByName", &success);
    }

    virNWFilterPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterLookupByName",
                libvirt);
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
    static virNWFilterLookupByUUIDType virNWFilterLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterLookupByUUIDSymbol = libvirtSymbol(libvirt, "virNWFilterLookupByUUID", &success);
    }

    virNWFilterPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterLookupByUUID",
                libvirt);
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
    static virNWFilterLookupByUUIDStringType virNWFilterLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virNWFilterLookupByUUIDString", &success);
    }

    virNWFilterPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterLookupByUUIDString",
                libvirt);
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
    static virNWFilterRefType virNWFilterRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterRefSymbol = libvirtSymbol(libvirt, "virNWFilterRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterRef",
                libvirt);
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
    static virNWFilterUndefineType virNWFilterUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNWFilterUndefineSymbol = libvirtSymbol(libvirt, "virNWFilterUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNWFilterUndefine",
                libvirt);
        return ret;
    }

    ret = virNWFilterUndefineSymbol(nwfilter);
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
    static virNetworkCreateType virNetworkCreateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkCreateSymbol = libvirtSymbol(libvirt, "virNetworkCreate", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkCreate",
                libvirt);
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
    static virNetworkCreateXMLType virNetworkCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkCreateXMLSymbol = libvirtSymbol(libvirt, "virNetworkCreateXML", &success);
    }

    virNetworkPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkCreateXML",
                libvirt);
        return ret;
    }

    ret = virNetworkCreateXMLSymbol(conn,
                                    xmlDesc);
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

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkDHCPLeaseFreeSymbol = libvirtSymbol(libvirt, "virNetworkDHCPLeaseFree", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkDHCPLeaseFree",
                libvirt);
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
    static virNetworkDefineXMLType virNetworkDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkDefineXMLSymbol = libvirtSymbol(libvirt, "virNetworkDefineXML", &success);
    }

    virNetworkPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkDefineXML",
                libvirt);
        return ret;
    }

    ret = virNetworkDefineXMLSymbol(conn,
                                    xml);
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
    static virNetworkDestroyType virNetworkDestroySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkDestroySymbol = libvirtSymbol(libvirt, "virNetworkDestroy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkDestroy",
                libvirt);
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
    static virNetworkFreeType virNetworkFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkFreeSymbol = libvirtSymbol(libvirt, "virNetworkFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkFree",
                libvirt);
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
    static virNetworkGetAutostartType virNetworkGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetAutostartSymbol = libvirtSymbol(libvirt, "virNetworkGetAutostart", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetAutostart",
                libvirt);
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
    static virNetworkGetBridgeNameType virNetworkGetBridgeNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetBridgeNameSymbol = libvirtSymbol(libvirt, "virNetworkGetBridgeName", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetBridgeName",
                libvirt);
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
    static virNetworkGetConnectType virNetworkGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetConnectSymbol = libvirtSymbol(libvirt, "virNetworkGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetConnect",
                libvirt);
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
    static virNetworkGetDHCPLeasesType virNetworkGetDHCPLeasesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetDHCPLeasesSymbol = libvirtSymbol(libvirt, "virNetworkGetDHCPLeases", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetDHCPLeases",
                libvirt);
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

typedef const char *
(*virNetworkGetNameType)(virNetworkPtr network);

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    static virNetworkGetNameType virNetworkGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetNameSymbol = libvirtSymbol(libvirt, "virNetworkGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetName",
                libvirt);
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
    static virNetworkGetUUIDType virNetworkGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetUUIDSymbol = libvirtSymbol(libvirt, "virNetworkGetUUID", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetUUID",
                libvirt);
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
    static virNetworkGetUUIDStringType virNetworkGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkGetUUIDString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetUUIDString",
                libvirt);
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
    static virNetworkGetXMLDescType virNetworkGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkGetXMLDescSymbol = libvirtSymbol(libvirt, "virNetworkGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkGetXMLDesc",
                libvirt);
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
    static virNetworkIsActiveType virNetworkIsActiveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkIsActiveSymbol = libvirtSymbol(libvirt, "virNetworkIsActive", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkIsActive",
                libvirt);
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
    static virNetworkIsPersistentType virNetworkIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkIsPersistentSymbol = libvirtSymbol(libvirt, "virNetworkIsPersistent", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkIsPersistent",
                libvirt);
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
    static virNetworkListAllPortsType virNetworkListAllPortsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkListAllPortsSymbol = libvirtSymbol(libvirt, "virNetworkListAllPorts", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkListAllPorts",
                libvirt);
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
    static virNetworkLookupByNameType virNetworkLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkLookupByNameSymbol = libvirtSymbol(libvirt, "virNetworkLookupByName", &success);
    }

    virNetworkPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkLookupByName",
                libvirt);
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
    static virNetworkLookupByUUIDType virNetworkLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkLookupByUUIDSymbol = libvirtSymbol(libvirt, "virNetworkLookupByUUID", &success);
    }

    virNetworkPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkLookupByUUID",
                libvirt);
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
    static virNetworkLookupByUUIDStringType virNetworkLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkLookupByUUIDString", &success);
    }

    virNetworkPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkLookupByUUIDString",
                libvirt);
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
    static virNetworkPortCreateXMLType virNetworkPortCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortCreateXMLSymbol = libvirtSymbol(libvirt, "virNetworkPortCreateXML", &success);
    }

    virNetworkPortPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortCreateXML",
                libvirt);
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
    static virNetworkPortDeleteType virNetworkPortDeleteSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortDeleteSymbol = libvirtSymbol(libvirt, "virNetworkPortDelete", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortDelete",
                libvirt);
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
    static virNetworkPortFreeType virNetworkPortFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortFreeSymbol = libvirtSymbol(libvirt, "virNetworkPortFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortFree",
                libvirt);
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
    static virNetworkPortGetNetworkType virNetworkPortGetNetworkSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortGetNetworkSymbol = libvirtSymbol(libvirt, "virNetworkPortGetNetwork", &success);
    }

    virNetworkPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortGetNetwork",
                libvirt);
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
    static virNetworkPortGetParametersType virNetworkPortGetParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortGetParametersSymbol = libvirtSymbol(libvirt, "virNetworkPortGetParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortGetParameters",
                libvirt);
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
    static virNetworkPortGetUUIDType virNetworkPortGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortGetUUIDSymbol = libvirtSymbol(libvirt, "virNetworkPortGetUUID", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortGetUUID",
                libvirt);
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
    static virNetworkPortGetUUIDStringType virNetworkPortGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortGetUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkPortGetUUIDString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortGetUUIDString",
                libvirt);
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
    static virNetworkPortGetXMLDescType virNetworkPortGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortGetXMLDescSymbol = libvirtSymbol(libvirt, "virNetworkPortGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortGetXMLDesc",
                libvirt);
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
    static virNetworkPortLookupByUUIDType virNetworkPortLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortLookupByUUIDSymbol = libvirtSymbol(libvirt, "virNetworkPortLookupByUUID", &success);
    }

    virNetworkPortPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortLookupByUUID",
                libvirt);
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
    static virNetworkPortLookupByUUIDStringType virNetworkPortLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkPortLookupByUUIDString", &success);
    }

    virNetworkPortPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortLookupByUUIDString",
                libvirt);
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
    static virNetworkPortRefType virNetworkPortRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortRefSymbol = libvirtSymbol(libvirt, "virNetworkPortRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortRef",
                libvirt);
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
    static virNetworkPortSetParametersType virNetworkPortSetParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkPortSetParametersSymbol = libvirtSymbol(libvirt, "virNetworkPortSetParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkPortSetParameters",
                libvirt);
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
    static virNetworkRefType virNetworkRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkRefSymbol = libvirtSymbol(libvirt, "virNetworkRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkRef",
                libvirt);
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
    static virNetworkSetAutostartType virNetworkSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkSetAutostartSymbol = libvirtSymbol(libvirt, "virNetworkSetAutostart", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkSetAutostart",
                libvirt);
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
(*virNetworkUndefineType)(virNetworkPtr network);

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err)
{
    static virNetworkUndefineType virNetworkUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkUndefineSymbol = libvirtSymbol(libvirt, "virNetworkUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkUndefine",
                libvirt);
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
    static virNetworkUpdateType virNetworkUpdateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNetworkUpdateSymbol = libvirtSymbol(libvirt, "virNetworkUpdate", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNetworkUpdate",
                libvirt);
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
    static virNodeAllocPagesType virNodeAllocPagesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeAllocPagesSymbol = libvirtSymbol(libvirt, "virNodeAllocPages", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeAllocPages",
                libvirt);
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
(*virNodeDeviceCreateType)(virNodeDevicePtr dev,
                           unsigned int flags);

int
virNodeDeviceCreateWrapper(virNodeDevicePtr dev,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virNodeDeviceCreateType virNodeDeviceCreateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceCreateSymbol = libvirtSymbol(libvirt, "virNodeDeviceCreate", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceCreate",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceCreateSymbol(dev,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceCreateXMLType)(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags);

virNodeDevicePtr
virNodeDeviceCreateXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virNodeDeviceCreateXMLType virNodeDeviceCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceCreateXMLSymbol = libvirtSymbol(libvirt, "virNodeDeviceCreateXML", &success);
    }

    virNodeDevicePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceCreateXML",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceCreateXMLSymbol(conn,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceDefineXMLType)(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags);

virNodeDevicePtr
virNodeDeviceDefineXMLWrapper(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virNodeDeviceDefineXMLType virNodeDeviceDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceDefineXMLSymbol = libvirtSymbol(libvirt, "virNodeDeviceDefineXML", &success);
    }

    virNodeDevicePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceDefineXML",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceDefineXMLSymbol(conn,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceDestroyType)(virNodeDevicePtr dev);

int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    static virNodeDeviceDestroyType virNodeDeviceDestroySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceDestroySymbol = libvirtSymbol(libvirt, "virNodeDeviceDestroy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceDestroy",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceDestroySymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceDetachFlagsType)(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags);

int
virNodeDeviceDetachFlagsWrapper(virNodeDevicePtr dev,
                                const char * driverName,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virNodeDeviceDetachFlagsType virNodeDeviceDetachFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceDetachFlagsSymbol = libvirtSymbol(libvirt, "virNodeDeviceDetachFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceDetachFlags",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceDetachFlagsSymbol(dev,
                                         driverName,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceDettachType)(virNodeDevicePtr dev);

int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    static virNodeDeviceDettachType virNodeDeviceDettachSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceDettachSymbol = libvirtSymbol(libvirt, "virNodeDeviceDettach", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceDettach",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceDettachSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceFreeType)(virNodeDevicePtr dev);

int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err)
{
    static virNodeDeviceFreeType virNodeDeviceFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceFreeSymbol = libvirtSymbol(libvirt, "virNodeDeviceFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceFree",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceFreeSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNodeDeviceGetNameType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    static virNodeDeviceGetNameType virNodeDeviceGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceGetNameSymbol = libvirtSymbol(libvirt, "virNodeDeviceGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceGetName",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceGetNameSymbol(dev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virNodeDeviceGetParentType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    static virNodeDeviceGetParentType virNodeDeviceGetParentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceGetParentSymbol = libvirtSymbol(libvirt, "virNodeDeviceGetParent", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceGetParent",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceGetParentSymbol(dev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virNodeDeviceGetXMLDescType)(virNodeDevicePtr dev,
                               unsigned int flags);

char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virNodeDeviceGetXMLDescType virNodeDeviceGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceGetXMLDescSymbol = libvirtSymbol(libvirt, "virNodeDeviceGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceGetXMLDescSymbol(dev,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceListCapsType)(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames);

int
virNodeDeviceListCapsWrapper(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames,
                             virErrorPtr err)
{
    static virNodeDeviceListCapsType virNodeDeviceListCapsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceListCapsSymbol = libvirtSymbol(libvirt, "virNodeDeviceListCaps", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceListCaps",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceListCapsSymbol(dev,
                                      names,
                                      maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceLookupByNameType)(virConnectPtr conn,
                                 const char * name);

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char * name,
                                 virErrorPtr err)
{
    static virNodeDeviceLookupByNameType virNodeDeviceLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceLookupByNameSymbol = libvirtSymbol(libvirt, "virNodeDeviceLookupByName", &success);
    }

    virNodeDevicePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceLookupByName",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceLookupByNameSymbol(conn,
                                          name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virNodeDevicePtr
(*virNodeDeviceLookupSCSIHostByWWNType)(virConnectPtr conn,
                                        const char * wwnn,
                                        const char * wwpn,
                                        unsigned int flags);

virNodeDevicePtr
virNodeDeviceLookupSCSIHostByWWNWrapper(virConnectPtr conn,
                                        const char * wwnn,
                                        const char * wwpn,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    static virNodeDeviceLookupSCSIHostByWWNType virNodeDeviceLookupSCSIHostByWWNSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceLookupSCSIHostByWWNSymbol = libvirtSymbol(libvirt, "virNodeDeviceLookupSCSIHostByWWN", &success);
    }

    virNodeDevicePtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceLookupSCSIHostByWWN",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceLookupSCSIHostByWWNSymbol(conn,
                                                 wwnn,
                                                 wwpn,
                                                 flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceNumOfCapsType)(virNodeDevicePtr dev);

int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    static virNodeDeviceNumOfCapsType virNodeDeviceNumOfCapsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceNumOfCapsSymbol = libvirtSymbol(libvirt, "virNodeDeviceNumOfCaps", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceNumOfCaps",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceNumOfCapsSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceReAttachType)(virNodeDevicePtr dev);

int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    static virNodeDeviceReAttachType virNodeDeviceReAttachSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceReAttachSymbol = libvirtSymbol(libvirt, "virNodeDeviceReAttach", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceReAttach",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceReAttachSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceRefType)(virNodeDevicePtr dev);

int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err)
{
    static virNodeDeviceRefType virNodeDeviceRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceRefSymbol = libvirtSymbol(libvirt, "virNodeDeviceRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceRef",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceRefSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceResetType)(virNodeDevicePtr dev);

int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err)
{
    static virNodeDeviceResetType virNodeDeviceResetSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceResetSymbol = libvirtSymbol(libvirt, "virNodeDeviceReset", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceReset",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceResetSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeDeviceUndefineType)(virNodeDevicePtr dev,
                             unsigned int flags);

int
virNodeDeviceUndefineWrapper(virNodeDevicePtr dev,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virNodeDeviceUndefineType virNodeDeviceUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeDeviceUndefineSymbol = libvirtSymbol(libvirt, "virNodeDeviceUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeDeviceUndefine",
                libvirt);
        return ret;
    }

    ret = virNodeDeviceUndefineSymbol(dev,
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
    static virNodeGetCPUMapType virNodeGetCPUMapSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetCPUMapSymbol = libvirtSymbol(libvirt, "virNodeGetCPUMap", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetCPUMap",
                libvirt);
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
    static virNodeGetCPUStatsType virNodeGetCPUStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetCPUStatsSymbol = libvirtSymbol(libvirt, "virNodeGetCPUStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetCPUStats",
                libvirt);
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
    static virNodeGetCellsFreeMemoryType virNodeGetCellsFreeMemorySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetCellsFreeMemorySymbol = libvirtSymbol(libvirt, "virNodeGetCellsFreeMemory", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetCellsFreeMemory",
                libvirt);
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
    static virNodeGetFreeMemoryType virNodeGetFreeMemorySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetFreeMemorySymbol = libvirtSymbol(libvirt, "virNodeGetFreeMemory", &success);
    }

    unsigned long long ret = 0;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetFreeMemory",
                libvirt);
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
    static virNodeGetFreePagesType virNodeGetFreePagesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetFreePagesSymbol = libvirtSymbol(libvirt, "virNodeGetFreePages", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetFreePages",
                libvirt);
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
    static virNodeGetInfoType virNodeGetInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetInfoSymbol = libvirtSymbol(libvirt, "virNodeGetInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetInfo",
                libvirt);
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
    static virNodeGetMemoryParametersType virNodeGetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetMemoryParametersSymbol = libvirtSymbol(libvirt, "virNodeGetMemoryParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetMemoryParameters",
                libvirt);
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
    static virNodeGetMemoryStatsType virNodeGetMemoryStatsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetMemoryStatsSymbol = libvirtSymbol(libvirt, "virNodeGetMemoryStats", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetMemoryStats",
                libvirt);
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
    static virNodeGetSEVInfoType virNodeGetSEVInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetSEVInfoSymbol = libvirtSymbol(libvirt, "virNodeGetSEVInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetSEVInfo",
                libvirt);
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
    static virNodeGetSecurityModelType virNodeGetSecurityModelSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeGetSecurityModelSymbol = libvirtSymbol(libvirt, "virNodeGetSecurityModel", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeGetSecurityModel",
                libvirt);
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
(*virNodeListDevicesType)(virConnectPtr conn,
                          const char * cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags);

int
virNodeListDevicesWrapper(virConnectPtr conn,
                          const char * cap,
                          char ** const names,
                          int maxnames,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virNodeListDevicesType virNodeListDevicesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeListDevicesSymbol = libvirtSymbol(libvirt, "virNodeListDevices", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeListDevices",
                libvirt);
        return ret;
    }

    ret = virNodeListDevicesSymbol(conn,
                                   cap,
                                   names,
                                   maxnames,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virNodeNumOfDevicesType)(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags);

int
virNodeNumOfDevicesWrapper(virConnectPtr conn,
                           const char * cap,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virNodeNumOfDevicesType virNodeNumOfDevicesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeNumOfDevicesSymbol = libvirtSymbol(libvirt, "virNodeNumOfDevices", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeNumOfDevices",
                libvirt);
        return ret;
    }

    ret = virNodeNumOfDevicesSymbol(conn,
                                    cap,
                                    flags);
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
    static virNodeSetMemoryParametersType virNodeSetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeSetMemoryParametersSymbol = libvirtSymbol(libvirt, "virNodeSetMemoryParameters", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeSetMemoryParameters",
                libvirt);
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
    static virNodeSuspendForDurationType virNodeSuspendForDurationSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virNodeSuspendForDurationSymbol = libvirtSymbol(libvirt, "virNodeSuspendForDuration", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virNodeSuspendForDuration",
                libvirt);
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

typedef void
(*virResetErrorType)(virErrorPtr err);

void
virResetErrorWrapper(virErrorPtr err)
{
    static virResetErrorType virResetErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virResetErrorSymbol = libvirtSymbol(libvirt, "virResetError", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virResetError",
                libvirt);
        return;
    }

    virResetErrorSymbol(err);
}

typedef void
(*virResetLastErrorType)(void);

void
virResetLastErrorWrapper(void)
{
    static virResetLastErrorType virResetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virResetLastErrorSymbol = libvirtSymbol(libvirt, "virResetLastError", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virResetLastError",
                libvirt);
        return;
    }

    virResetLastErrorSymbol();
}

typedef virErrorPtr
(*virSaveLastErrorType)(void);

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err)
{
    static virSaveLastErrorType virSaveLastErrorSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSaveLastErrorSymbol = libvirtSymbol(libvirt, "virSaveLastError", &success);
    }

    virErrorPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSaveLastError",
                libvirt);
        return ret;
    }

    ret = virSaveLastErrorSymbol();
    if (!ret) {
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
    static virSecretDefineXMLType virSecretDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretDefineXMLSymbol = libvirtSymbol(libvirt, "virSecretDefineXML", &success);
    }

    virSecretPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretDefineXML",
                libvirt);
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
    static virSecretFreeType virSecretFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretFreeSymbol = libvirtSymbol(libvirt, "virSecretFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretFree",
                libvirt);
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
    static virSecretGetConnectType virSecretGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetConnectSymbol = libvirtSymbol(libvirt, "virSecretGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetConnect",
                libvirt);
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
    static virSecretGetUUIDType virSecretGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetUUIDSymbol = libvirtSymbol(libvirt, "virSecretGetUUID", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetUUID",
                libvirt);
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
    static virSecretGetUUIDStringType virSecretGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetUUIDStringSymbol = libvirtSymbol(libvirt, "virSecretGetUUIDString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetUUIDString",
                libvirt);
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
    static virSecretGetUsageIDType virSecretGetUsageIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetUsageIDSymbol = libvirtSymbol(libvirt, "virSecretGetUsageID", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetUsageID",
                libvirt);
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
    static virSecretGetUsageTypeType virSecretGetUsageTypeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetUsageTypeSymbol = libvirtSymbol(libvirt, "virSecretGetUsageType", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetUsageType",
                libvirt);
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
    static virSecretGetValueType virSecretGetValueSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetValueSymbol = libvirtSymbol(libvirt, "virSecretGetValue", &success);
    }

    unsigned char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetValue",
                libvirt);
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
    static virSecretGetXMLDescType virSecretGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretGetXMLDescSymbol = libvirtSymbol(libvirt, "virSecretGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretGetXMLDesc",
                libvirt);
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
    static virSecretLookupByUUIDType virSecretLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretLookupByUUIDSymbol = libvirtSymbol(libvirt, "virSecretLookupByUUID", &success);
    }

    virSecretPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretLookupByUUID",
                libvirt);
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
    static virSecretLookupByUUIDStringType virSecretLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virSecretLookupByUUIDString", &success);
    }

    virSecretPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretLookupByUUIDString",
                libvirt);
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
    static virSecretLookupByUsageType virSecretLookupByUsageSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretLookupByUsageSymbol = libvirtSymbol(libvirt, "virSecretLookupByUsage", &success);
    }

    virSecretPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretLookupByUsage",
                libvirt);
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
    static virSecretRefType virSecretRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretRefSymbol = libvirtSymbol(libvirt, "virSecretRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretRef",
                libvirt);
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
    static virSecretSetValueType virSecretSetValueSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretSetValueSymbol = libvirtSymbol(libvirt, "virSecretSetValue", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretSetValue",
                libvirt);
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
    static virSecretUndefineType virSecretUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSecretUndefineSymbol = libvirtSymbol(libvirt, "virSecretUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virSecretUndefine",
                libvirt);
        return ret;
    }

    ret = virSecretUndefineSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virSetErrorFuncType)(void * userData,
                       virErrorFunc handler);

void
virSetErrorFuncWrapper(void * userData,
                       virErrorFunc handler)
{
    static virSetErrorFuncType virSetErrorFuncSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virSetErrorFuncSymbol = libvirtSymbol(libvirt, "virSetErrorFunc", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virSetErrorFunc",
                libvirt);
        return;
    }

    virSetErrorFuncSymbol(userData,
                          handler);
}

typedef int
(*virStoragePoolBuildType)(virStoragePoolPtr pool,
                           unsigned int flags);

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virStoragePoolBuildType virStoragePoolBuildSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolBuildSymbol = libvirtSymbol(libvirt, "virStoragePoolBuild", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolBuild",
                libvirt);
        return ret;
    }

    ret = virStoragePoolBuildSymbol(pool,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolCreateType)(virStoragePoolPtr pool,
                            unsigned int flags);

int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virStoragePoolCreateType virStoragePoolCreateSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolCreateSymbol = libvirtSymbol(libvirt, "virStoragePoolCreate", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolCreate",
                libvirt);
        return ret;
    }

    ret = virStoragePoolCreateSymbol(pool,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolCreateXMLType)(virConnectPtr conn,
                               const char * xmlDesc,
                               unsigned int flags);

virStoragePoolPtr
virStoragePoolCreateXMLWrapper(virConnectPtr conn,
                               const char * xmlDesc,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virStoragePoolCreateXMLType virStoragePoolCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolCreateXMLSymbol = libvirtSymbol(libvirt, "virStoragePoolCreateXML", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolCreateXML",
                libvirt);
        return ret;
    }

    ret = virStoragePoolCreateXMLSymbol(conn,
                                        xmlDesc,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolDefineXMLType)(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags);

virStoragePoolPtr
virStoragePoolDefineXMLWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virStoragePoolDefineXMLType virStoragePoolDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolDefineXMLSymbol = libvirtSymbol(libvirt, "virStoragePoolDefineXML", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolDefineXML",
                libvirt);
        return ret;
    }

    ret = virStoragePoolDefineXMLSymbol(conn,
                                        xml,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolDeleteType)(virStoragePoolPtr pool,
                            unsigned int flags);

int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    static virStoragePoolDeleteType virStoragePoolDeleteSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolDeleteSymbol = libvirtSymbol(libvirt, "virStoragePoolDelete", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolDelete",
                libvirt);
        return ret;
    }

    ret = virStoragePoolDeleteSymbol(pool,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolDestroyType)(virStoragePoolPtr pool);

int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    static virStoragePoolDestroyType virStoragePoolDestroySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolDestroySymbol = libvirtSymbol(libvirt, "virStoragePoolDestroy", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolDestroy",
                libvirt);
        return ret;
    }

    ret = virStoragePoolDestroySymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolFreeType)(virStoragePoolPtr pool);

int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err)
{
    static virStoragePoolFreeType virStoragePoolFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolFreeSymbol = libvirtSymbol(libvirt, "virStoragePoolFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolFree",
                libvirt);
        return ret;
    }

    ret = virStoragePoolFreeSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetAutostartType)(virStoragePoolPtr pool,
                                  int * autostart);

int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int * autostart,
                                  virErrorPtr err)
{
    static virStoragePoolGetAutostartType virStoragePoolGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetAutostartSymbol = libvirtSymbol(libvirt, "virStoragePoolGetAutostart", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetAutostart",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetAutostartSymbol(pool,
                                           autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virStoragePoolGetConnectType)(virStoragePoolPtr pool);

virConnectPtr
virStoragePoolGetConnectWrapper(virStoragePoolPtr pool,
                                virErrorPtr err)
{
    static virStoragePoolGetConnectType virStoragePoolGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetConnectSymbol = libvirtSymbol(libvirt, "virStoragePoolGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetConnect",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetConnectSymbol(pool);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetInfoType)(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info);

int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err)
{
    static virStoragePoolGetInfoType virStoragePoolGetInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetInfoSymbol = libvirtSymbol(libvirt, "virStoragePoolGetInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetInfo",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetInfoSymbol(pool,
                                      info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virStoragePoolGetNameType)(virStoragePoolPtr pool);

const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    static virStoragePoolGetNameType virStoragePoolGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetNameSymbol = libvirtSymbol(libvirt, "virStoragePoolGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetName",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetNameSymbol(pool);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetUUIDType)(virStoragePoolPtr pool,
                             unsigned char * uuid);

int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    static virStoragePoolGetUUIDType virStoragePoolGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetUUIDSymbol = libvirtSymbol(libvirt, "virStoragePoolGetUUID", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetUUID",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetUUIDSymbol(pool,
                                      uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolGetUUIDStringType)(virStoragePoolPtr pool,
                                   char * buf);

int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char * buf,
                                   virErrorPtr err)
{
    static virStoragePoolGetUUIDStringType virStoragePoolGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetUUIDStringSymbol = libvirtSymbol(libvirt, "virStoragePoolGetUUIDString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetUUIDString",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetUUIDStringSymbol(pool,
                                            buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virStoragePoolGetXMLDescType)(virStoragePoolPtr pool,
                                unsigned int flags);

char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virStoragePoolGetXMLDescType virStoragePoolGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolGetXMLDescSymbol = libvirtSymbol(libvirt, "virStoragePoolGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virStoragePoolGetXMLDescSymbol(pool,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolIsActiveType)(virStoragePoolPtr pool);

int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    static virStoragePoolIsActiveType virStoragePoolIsActiveSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolIsActiveSymbol = libvirtSymbol(libvirt, "virStoragePoolIsActive", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolIsActive",
                libvirt);
        return ret;
    }

    ret = virStoragePoolIsActiveSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolIsPersistentType)(virStoragePoolPtr pool);

int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    static virStoragePoolIsPersistentType virStoragePoolIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolIsPersistentSymbol = libvirtSymbol(libvirt, "virStoragePoolIsPersistent", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolIsPersistent",
                libvirt);
        return ret;
    }

    ret = virStoragePoolIsPersistentSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolListAllVolumesType)(virStoragePoolPtr pool,
                                    virStorageVolPtr ** vols,
                                    unsigned int flags);

int
virStoragePoolListAllVolumesWrapper(virStoragePoolPtr pool,
                                    virStorageVolPtr ** vols,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    static virStoragePoolListAllVolumesType virStoragePoolListAllVolumesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolListAllVolumesSymbol = libvirtSymbol(libvirt, "virStoragePoolListAllVolumes", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolListAllVolumes",
                libvirt);
        return ret;
    }

    ret = virStoragePoolListAllVolumesSymbol(pool,
                                             vols,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolListVolumesType)(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames);

int
virStoragePoolListVolumesWrapper(virStoragePoolPtr pool,
                                 char ** const names,
                                 int maxnames,
                                 virErrorPtr err)
{
    static virStoragePoolListVolumesType virStoragePoolListVolumesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolListVolumesSymbol = libvirtSymbol(libvirt, "virStoragePoolListVolumes", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolListVolumes",
                libvirt);
        return ret;
    }

    ret = virStoragePoolListVolumesSymbol(pool,
                                          names,
                                          maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByNameType)(virConnectPtr conn,
                                  const char * name);

virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char * name,
                                  virErrorPtr err)
{
    static virStoragePoolLookupByNameType virStoragePoolLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolLookupByNameSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByName", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolLookupByName",
                libvirt);
        return ret;
    }

    ret = virStoragePoolLookupByNameSymbol(conn,
                                           name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByTargetPathType)(virConnectPtr conn,
                                        const char * path);

virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char * path,
                                        virErrorPtr err)
{
    static virStoragePoolLookupByTargetPathType virStoragePoolLookupByTargetPathSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolLookupByTargetPathSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByTargetPath", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolLookupByTargetPath",
                libvirt);
        return ret;
    }

    ret = virStoragePoolLookupByTargetPathSymbol(conn,
                                                 path);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByUUIDType)(virConnectPtr conn,
                                  const unsigned char * uuid);

virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    static virStoragePoolLookupByUUIDType virStoragePoolLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolLookupByUUIDSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByUUID", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolLookupByUUID",
                libvirt);
        return ret;
    }

    ret = virStoragePoolLookupByUUIDSymbol(conn,
                                           uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByUUIDStringType)(virConnectPtr conn,
                                        const char * uuidstr);

virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    static virStoragePoolLookupByUUIDStringType virStoragePoolLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByUUIDString", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolLookupByUUIDString",
                libvirt);
        return ret;
    }

    ret = virStoragePoolLookupByUUIDStringSymbol(conn,
                                                 uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByVolumeType)(virStorageVolPtr vol);

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err)
{
    static virStoragePoolLookupByVolumeType virStoragePoolLookupByVolumeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolLookupByVolumeSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByVolume", &success);
    }

    virStoragePoolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolLookupByVolume",
                libvirt);
        return ret;
    }

    ret = virStoragePoolLookupByVolumeSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolNumOfVolumesType)(virStoragePoolPtr pool);

int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    static virStoragePoolNumOfVolumesType virStoragePoolNumOfVolumesSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolNumOfVolumesSymbol = libvirtSymbol(libvirt, "virStoragePoolNumOfVolumes", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolNumOfVolumes",
                libvirt);
        return ret;
    }

    ret = virStoragePoolNumOfVolumesSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolRefType)(virStoragePoolPtr pool);

int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err)
{
    static virStoragePoolRefType virStoragePoolRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolRefSymbol = libvirtSymbol(libvirt, "virStoragePoolRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolRef",
                libvirt);
        return ret;
    }

    ret = virStoragePoolRefSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolRefreshType)(virStoragePoolPtr pool,
                             unsigned int flags);

int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virStoragePoolRefreshType virStoragePoolRefreshSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolRefreshSymbol = libvirtSymbol(libvirt, "virStoragePoolRefresh", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolRefresh",
                libvirt);
        return ret;
    }

    ret = virStoragePoolRefreshSymbol(pool,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolSetAutostartType)(virStoragePoolPtr pool,
                                  int autostart);

int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err)
{
    static virStoragePoolSetAutostartType virStoragePoolSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolSetAutostartSymbol = libvirtSymbol(libvirt, "virStoragePoolSetAutostart", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolSetAutostart",
                libvirt);
        return ret;
    }

    ret = virStoragePoolSetAutostartSymbol(pool,
                                           autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStoragePoolUndefineType)(virStoragePoolPtr pool);

int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    static virStoragePoolUndefineType virStoragePoolUndefineSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStoragePoolUndefineSymbol = libvirtSymbol(libvirt, "virStoragePoolUndefine", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStoragePoolUndefine",
                libvirt);
        return ret;
    }

    ret = virStoragePoolUndefineSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolCreateXMLType)(virStoragePoolPtr pool,
                              const char * xmlDesc,
                              unsigned int flags);

virStorageVolPtr
virStorageVolCreateXMLWrapper(virStoragePoolPtr pool,
                              const char * xmlDesc,
                              unsigned int flags,
                              virErrorPtr err)
{
    static virStorageVolCreateXMLType virStorageVolCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolCreateXMLSymbol = libvirtSymbol(libvirt, "virStorageVolCreateXML", &success);
    }

    virStorageVolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolCreateXML",
                libvirt);
        return ret;
    }

    ret = virStorageVolCreateXMLSymbol(pool,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolCreateXMLFromType)(virStoragePoolPtr pool,
                                  const char * xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags);

virStorageVolPtr
virStorageVolCreateXMLFromWrapper(virStoragePoolPtr pool,
                                  const char * xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virStorageVolCreateXMLFromType virStorageVolCreateXMLFromSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolCreateXMLFromSymbol = libvirtSymbol(libvirt, "virStorageVolCreateXMLFrom", &success);
    }

    virStorageVolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolCreateXMLFrom",
                libvirt);
        return ret;
    }

    ret = virStorageVolCreateXMLFromSymbol(pool,
                                           xmlDesc,
                                           clonevol,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolDeleteType)(virStorageVolPtr vol,
                           unsigned int flags);

int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virStorageVolDeleteType virStorageVolDeleteSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolDeleteSymbol = libvirtSymbol(libvirt, "virStorageVolDelete", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolDelete",
                libvirt);
        return ret;
    }

    ret = virStorageVolDeleteSymbol(vol,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolDownloadType)(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags);

int
virStorageVolDownloadWrapper(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags,
                             virErrorPtr err)
{
    static virStorageVolDownloadType virStorageVolDownloadSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolDownloadSymbol = libvirtSymbol(libvirt, "virStorageVolDownload", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolDownload",
                libvirt);
        return ret;
    }

    ret = virStorageVolDownloadSymbol(vol,
                                      stream,
                                      offset,
                                      length,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolFreeType)(virStorageVolPtr vol);

int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err)
{
    static virStorageVolFreeType virStorageVolFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolFreeSymbol = libvirtSymbol(libvirt, "virStorageVolFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolFree",
                libvirt);
        return ret;
    }

    ret = virStorageVolFreeSymbol(vol);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virStorageVolGetConnectType)(virStorageVolPtr vol);

virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err)
{
    static virStorageVolGetConnectType virStorageVolGetConnectSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetConnectSymbol = libvirtSymbol(libvirt, "virStorageVolGetConnect", &success);
    }

    virConnectPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetConnect",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetConnectSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolGetInfoType)(virStorageVolPtr vol,
                            virStorageVolInfoPtr info);

int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err)
{
    static virStorageVolGetInfoType virStorageVolGetInfoSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetInfoSymbol = libvirtSymbol(libvirt, "virStorageVolGetInfo", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetInfo",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetInfoSymbol(vol,
                                     info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolGetInfoFlagsType)(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags);

int
virStorageVolGetInfoFlagsWrapper(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virStorageVolGetInfoFlagsType virStorageVolGetInfoFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetInfoFlagsSymbol = libvirtSymbol(libvirt, "virStorageVolGetInfoFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetInfoFlags",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetInfoFlagsSymbol(vol,
                                          info,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virStorageVolGetKeyType)(virStorageVolPtr vol);

const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err)
{
    static virStorageVolGetKeyType virStorageVolGetKeySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetKeySymbol = libvirtSymbol(libvirt, "virStorageVolGetKey", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetKey",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetKeySymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virStorageVolGetNameType)(virStorageVolPtr vol);

const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    static virStorageVolGetNameType virStorageVolGetNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetNameSymbol = libvirtSymbol(libvirt, "virStorageVolGetName", &success);
    }

    const char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetName",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetNameSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virStorageVolGetPathType)(virStorageVolPtr vol);

char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    static virStorageVolGetPathType virStorageVolGetPathSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetPathSymbol = libvirtSymbol(libvirt, "virStorageVolGetPath", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetPath",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetPathSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virStorageVolGetXMLDescType)(virStorageVolPtr vol,
                               unsigned int flags);

char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virStorageVolGetXMLDescType virStorageVolGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolGetXMLDescSymbol = libvirtSymbol(libvirt, "virStorageVolGetXMLDesc", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolGetXMLDesc",
                libvirt);
        return ret;
    }

    ret = virStorageVolGetXMLDescSymbol(vol,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolLookupByKeyType)(virConnectPtr conn,
                                const char * key);

virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char * key,
                                virErrorPtr err)
{
    static virStorageVolLookupByKeyType virStorageVolLookupByKeySymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolLookupByKeySymbol = libvirtSymbol(libvirt, "virStorageVolLookupByKey", &success);
    }

    virStorageVolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolLookupByKey",
                libvirt);
        return ret;
    }

    ret = virStorageVolLookupByKeySymbol(conn,
                                         key);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolLookupByNameType)(virStoragePoolPtr pool,
                                 const char * name);

virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char * name,
                                 virErrorPtr err)
{
    static virStorageVolLookupByNameType virStorageVolLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolLookupByNameSymbol = libvirtSymbol(libvirt, "virStorageVolLookupByName", &success);
    }

    virStorageVolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolLookupByName",
                libvirt);
        return ret;
    }

    ret = virStorageVolLookupByNameSymbol(pool,
                                          name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStorageVolPtr
(*virStorageVolLookupByPathType)(virConnectPtr conn,
                                 const char * path);

virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char * path,
                                 virErrorPtr err)
{
    static virStorageVolLookupByPathType virStorageVolLookupByPathSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolLookupByPathSymbol = libvirtSymbol(libvirt, "virStorageVolLookupByPath", &success);
    }

    virStorageVolPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolLookupByPath",
                libvirt);
        return ret;
    }

    ret = virStorageVolLookupByPathSymbol(conn,
                                          path);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolRefType)(virStorageVolPtr vol);

int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err)
{
    static virStorageVolRefType virStorageVolRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolRefSymbol = libvirtSymbol(libvirt, "virStorageVolRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolRef",
                libvirt);
        return ret;
    }

    ret = virStorageVolRefSymbol(vol);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolResizeType)(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags);

int
virStorageVolResizeWrapper(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virStorageVolResizeType virStorageVolResizeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolResizeSymbol = libvirtSymbol(libvirt, "virStorageVolResize", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolResize",
                libvirt);
        return ret;
    }

    ret = virStorageVolResizeSymbol(vol,
                                    capacity,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolUploadType)(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags);

int
virStorageVolUploadWrapper(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virStorageVolUploadType virStorageVolUploadSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolUploadSymbol = libvirtSymbol(libvirt, "virStorageVolUpload", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolUpload",
                libvirt);
        return ret;
    }

    ret = virStorageVolUploadSymbol(vol,
                                    stream,
                                    offset,
                                    length,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolWipeType)(virStorageVolPtr vol,
                         unsigned int flags);

int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virStorageVolWipeType virStorageVolWipeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolWipeSymbol = libvirtSymbol(libvirt, "virStorageVolWipe", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolWipe",
                libvirt);
        return ret;
    }

    ret = virStorageVolWipeSymbol(vol,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStorageVolWipePatternType)(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags);

int
virStorageVolWipePatternWrapper(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags,
                                virErrorPtr err)
{
    static virStorageVolWipePatternType virStorageVolWipePatternSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStorageVolWipePatternSymbol = libvirtSymbol(libvirt, "virStorageVolWipePattern", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStorageVolWipePattern",
                libvirt);
        return ret;
    }

    ret = virStorageVolWipePatternSymbol(vol,
                                         algorithm,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamAbortType)(virStreamPtr stream);

int
virStreamAbortWrapper(virStreamPtr stream,
                      virErrorPtr err)
{
    static virStreamAbortType virStreamAbortSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamAbortSymbol = libvirtSymbol(libvirt, "virStreamAbort", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamAbort",
                libvirt);
        return ret;
    }

    ret = virStreamAbortSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamEventAddCallbackType)(virStreamPtr stream,
                                 int events,
                                 virStreamEventCallback cb,
                                 void * opaque,
                                 virFreeCallback ff);

int
virStreamEventAddCallbackWrapper(virStreamPtr stream,
                                 int events,
                                 virStreamEventCallback cb,
                                 void * opaque,
                                 virFreeCallback ff,
                                 virErrorPtr err)
{
    static virStreamEventAddCallbackType virStreamEventAddCallbackSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamEventAddCallbackSymbol = libvirtSymbol(libvirt, "virStreamEventAddCallback", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamEventAddCallback",
                libvirt);
        return ret;
    }

    ret = virStreamEventAddCallbackSymbol(stream,
                                          events,
                                          cb,
                                          opaque,
                                          ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamEventRemoveCallbackType)(virStreamPtr stream);

int
virStreamEventRemoveCallbackWrapper(virStreamPtr stream,
                                    virErrorPtr err)
{
    static virStreamEventRemoveCallbackType virStreamEventRemoveCallbackSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamEventRemoveCallbackSymbol = libvirtSymbol(libvirt, "virStreamEventRemoveCallback", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamEventRemoveCallback",
                libvirt);
        return ret;
    }

    ret = virStreamEventRemoveCallbackSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamEventUpdateCallbackType)(virStreamPtr stream,
                                    int events);

int
virStreamEventUpdateCallbackWrapper(virStreamPtr stream,
                                    int events,
                                    virErrorPtr err)
{
    static virStreamEventUpdateCallbackType virStreamEventUpdateCallbackSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamEventUpdateCallbackSymbol = libvirtSymbol(libvirt, "virStreamEventUpdateCallback", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamEventUpdateCallback",
                libvirt);
        return ret;
    }

    ret = virStreamEventUpdateCallbackSymbol(stream,
                                             events);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamFinishType)(virStreamPtr stream);

int
virStreamFinishWrapper(virStreamPtr stream,
                       virErrorPtr err)
{
    static virStreamFinishType virStreamFinishSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamFinishSymbol = libvirtSymbol(libvirt, "virStreamFinish", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamFinish",
                libvirt);
        return ret;
    }

    ret = virStreamFinishSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamFreeType)(virStreamPtr stream);

int
virStreamFreeWrapper(virStreamPtr stream,
                     virErrorPtr err)
{
    static virStreamFreeType virStreamFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamFreeSymbol = libvirtSymbol(libvirt, "virStreamFree", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamFree",
                libvirt);
        return ret;
    }

    ret = virStreamFreeSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStreamPtr
(*virStreamNewType)(virConnectPtr conn,
                    unsigned int flags);

virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err)
{
    static virStreamNewType virStreamNewSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamNewSymbol = libvirtSymbol(libvirt, "virStreamNew", &success);
    }

    virStreamPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamNew",
                libvirt);
        return ret;
    }

    ret = virStreamNewSymbol(conn,
                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvType)(virStreamPtr stream,
                     char * data,
                     size_t nbytes);

int
virStreamRecvWrapper(virStreamPtr stream,
                     char * data,
                     size_t nbytes,
                     virErrorPtr err)
{
    static virStreamRecvType virStreamRecvSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamRecvSymbol = libvirtSymbol(libvirt, "virStreamRecv", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamRecv",
                libvirt);
        return ret;
    }

    ret = virStreamRecvSymbol(stream,
                              data,
                              nbytes);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvAllType)(virStreamPtr stream,
                        virStreamSinkFunc handler,
                        void * opaque);

int
virStreamRecvAllWrapper(virStreamPtr stream,
                        virStreamSinkFunc handler,
                        void * opaque,
                        virErrorPtr err)
{
    static virStreamRecvAllType virStreamRecvAllSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamRecvAllSymbol = libvirtSymbol(libvirt, "virStreamRecvAll", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamRecvAll",
                libvirt);
        return ret;
    }

    ret = virStreamRecvAllSymbol(stream,
                                 handler,
                                 opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvFlagsType)(virStreamPtr stream,
                          char * data,
                          size_t nbytes,
                          unsigned int flags);

int
virStreamRecvFlagsWrapper(virStreamPtr stream,
                          char * data,
                          size_t nbytes,
                          unsigned int flags,
                          virErrorPtr err)
{
    static virStreamRecvFlagsType virStreamRecvFlagsSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamRecvFlagsSymbol = libvirtSymbol(libvirt, "virStreamRecvFlags", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamRecvFlags",
                libvirt);
        return ret;
    }

    ret = virStreamRecvFlagsSymbol(stream,
                                   data,
                                   nbytes,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvHoleType)(virStreamPtr stream,
                         long long * length,
                         unsigned int flags);

int
virStreamRecvHoleWrapper(virStreamPtr stream,
                         long long * length,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virStreamRecvHoleType virStreamRecvHoleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamRecvHoleSymbol = libvirtSymbol(libvirt, "virStreamRecvHole", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamRecvHole",
                libvirt);
        return ret;
    }

    ret = virStreamRecvHoleSymbol(stream,
                                  length,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRefType)(virStreamPtr stream);

int
virStreamRefWrapper(virStreamPtr stream,
                    virErrorPtr err)
{
    static virStreamRefType virStreamRefSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamRefSymbol = libvirtSymbol(libvirt, "virStreamRef", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamRef",
                libvirt);
        return ret;
    }

    ret = virStreamRefSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSendType)(virStreamPtr stream,
                     const char * data,
                     size_t nbytes);

int
virStreamSendWrapper(virStreamPtr stream,
                     const char * data,
                     size_t nbytes,
                     virErrorPtr err)
{
    static virStreamSendType virStreamSendSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamSendSymbol = libvirtSymbol(libvirt, "virStreamSend", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamSend",
                libvirt);
        return ret;
    }

    ret = virStreamSendSymbol(stream,
                              data,
                              nbytes);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSendAllType)(virStreamPtr stream,
                        virStreamSourceFunc handler,
                        void * opaque);

int
virStreamSendAllWrapper(virStreamPtr stream,
                        virStreamSourceFunc handler,
                        void * opaque,
                        virErrorPtr err)
{
    static virStreamSendAllType virStreamSendAllSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamSendAllSymbol = libvirtSymbol(libvirt, "virStreamSendAll", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamSendAll",
                libvirt);
        return ret;
    }

    ret = virStreamSendAllSymbol(stream,
                                 handler,
                                 opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSendHoleType)(virStreamPtr stream,
                         long long length,
                         unsigned int flags);

int
virStreamSendHoleWrapper(virStreamPtr stream,
                         long long length,
                         unsigned int flags,
                         virErrorPtr err)
{
    static virStreamSendHoleType virStreamSendHoleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamSendHoleSymbol = libvirtSymbol(libvirt, "virStreamSendHole", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamSendHole",
                libvirt);
        return ret;
    }

    ret = virStreamSendHoleSymbol(stream,
                                  length,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSparseRecvAllType)(virStreamPtr stream,
                              virStreamSinkFunc handler,
                              virStreamSinkHoleFunc holeHandler,
                              void * opaque);

int
virStreamSparseRecvAllWrapper(virStreamPtr stream,
                              virStreamSinkFunc handler,
                              virStreamSinkHoleFunc holeHandler,
                              void * opaque,
                              virErrorPtr err)
{
    static virStreamSparseRecvAllType virStreamSparseRecvAllSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamSparseRecvAllSymbol = libvirtSymbol(libvirt, "virStreamSparseRecvAll", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamSparseRecvAll",
                libvirt);
        return ret;
    }

    ret = virStreamSparseRecvAllSymbol(stream,
                                       handler,
                                       holeHandler,
                                       opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSparseSendAllType)(virStreamPtr stream,
                              virStreamSourceFunc handler,
                              virStreamSourceHoleFunc holeHandler,
                              virStreamSourceSkipFunc skipHandler,
                              void * opaque);

int
virStreamSparseSendAllWrapper(virStreamPtr stream,
                              virStreamSourceFunc handler,
                              virStreamSourceHoleFunc holeHandler,
                              virStreamSourceSkipFunc skipHandler,
                              void * opaque,
                              virErrorPtr err)
{
    static virStreamSparseSendAllType virStreamSparseSendAllSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virStreamSparseSendAllSymbol = libvirtSymbol(libvirt, "virStreamSparseSendAll", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virStreamSparseSendAll",
                libvirt);
        return ret;
    }

    ret = virStreamSparseSendAllSymbol(stream,
                                       handler,
                                       holeHandler,
                                       skipHandler,
                                       opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddBooleanType)(virTypedParameterPtr * params,
                                int * nparams,
                                int * maxparams,
                                const char * name,
                                int value);

int
virTypedParamsAddBooleanWrapper(virTypedParameterPtr * params,
                                int * nparams,
                                int * maxparams,
                                const char * name,
                                int value,
                                virErrorPtr err)
{
    static virTypedParamsAddBooleanType virTypedParamsAddBooleanSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddBooleanSymbol = libvirtSymbol(libvirt, "virTypedParamsAddBoolean", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddBoolean",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddBooleanSymbol(params,
                                         nparams,
                                         maxparams,
                                         name,
                                         value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddDoubleType)(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               double value);

int
virTypedParamsAddDoubleWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               double value,
                               virErrorPtr err)
{
    static virTypedParamsAddDoubleType virTypedParamsAddDoubleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddDoubleSymbol = libvirtSymbol(libvirt, "virTypedParamsAddDouble", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddDouble",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddDoubleSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddFromStringType)(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   int type,
                                   const char * value);

int
virTypedParamsAddFromStringWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   int type,
                                   const char * value,
                                   virErrorPtr err)
{
    static virTypedParamsAddFromStringType virTypedParamsAddFromStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddFromStringSymbol = libvirtSymbol(libvirt, "virTypedParamsAddFromString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddFromString",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddFromStringSymbol(params,
                                            nparams,
                                            maxparams,
                                            name,
                                            type,
                                            value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddIntType)(virTypedParameterPtr * params,
                            int * nparams,
                            int * maxparams,
                            const char * name,
                            int value);

int
virTypedParamsAddIntWrapper(virTypedParameterPtr * params,
                            int * nparams,
                            int * maxparams,
                            const char * name,
                            int value,
                            virErrorPtr err)
{
    static virTypedParamsAddIntType virTypedParamsAddIntSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddIntSymbol = libvirtSymbol(libvirt, "virTypedParamsAddInt", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddInt",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddIntSymbol(params,
                                     nparams,
                                     maxparams,
                                     name,
                                     value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddLLongType)(virTypedParameterPtr * params,
                              int * nparams,
                              int * maxparams,
                              const char * name,
                              long long value);

int
virTypedParamsAddLLongWrapper(virTypedParameterPtr * params,
                              int * nparams,
                              int * maxparams,
                              const char * name,
                              long long value,
                              virErrorPtr err)
{
    static virTypedParamsAddLLongType virTypedParamsAddLLongSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddLLongSymbol = libvirtSymbol(libvirt, "virTypedParamsAddLLong", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddLLong",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddLLongSymbol(params,
                                       nparams,
                                       maxparams,
                                       name,
                                       value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddStringType)(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               const char * value);

int
virTypedParamsAddStringWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               const char * value,
                               virErrorPtr err)
{
    static virTypedParamsAddStringType virTypedParamsAddStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddStringSymbol = libvirtSymbol(libvirt, "virTypedParamsAddString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddString",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddStringSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddStringListType)(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   const char ** values);

int
virTypedParamsAddStringListWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   const char ** values,
                                   virErrorPtr err)
{
    static virTypedParamsAddStringListType virTypedParamsAddStringListSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddStringListSymbol = libvirtSymbol(libvirt, "virTypedParamsAddStringList", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddStringList",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddStringListSymbol(params,
                                            nparams,
                                            maxparams,
                                            name,
                                            values);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddUIntType)(virTypedParameterPtr * params,
                             int * nparams,
                             int * maxparams,
                             const char * name,
                             unsigned int value);

int
virTypedParamsAddUIntWrapper(virTypedParameterPtr * params,
                             int * nparams,
                             int * maxparams,
                             const char * name,
                             unsigned int value,
                             virErrorPtr err)
{
    static virTypedParamsAddUIntType virTypedParamsAddUIntSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddUIntSymbol = libvirtSymbol(libvirt, "virTypedParamsAddUInt", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddUInt",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddUIntSymbol(params,
                                      nparams,
                                      maxparams,
                                      name,
                                      value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddULLongType)(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               unsigned long long value);

int
virTypedParamsAddULLongWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               unsigned long long value,
                               virErrorPtr err)
{
    static virTypedParamsAddULLongType virTypedParamsAddULLongSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsAddULLongSymbol = libvirtSymbol(libvirt, "virTypedParamsAddULLong", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsAddULLong",
                libvirt);
        return ret;
    }

    ret = virTypedParamsAddULLongSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virTypedParamsClearType)(virTypedParameterPtr params,
                           int nparams);

void
virTypedParamsClearWrapper(virTypedParameterPtr params,
                           int nparams)
{
    static virTypedParamsClearType virTypedParamsClearSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsClearSymbol = libvirtSymbol(libvirt, "virTypedParamsClear", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsClear",
                libvirt);
        return;
    }

    virTypedParamsClearSymbol(params,
                              nparams);
}

typedef void
(*virTypedParamsFreeType)(virTypedParameterPtr params,
                          int nparams);

void
virTypedParamsFreeWrapper(virTypedParameterPtr params,
                          int nparams)
{
    static virTypedParamsFreeType virTypedParamsFreeSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsFreeSymbol = libvirtSymbol(libvirt, "virTypedParamsFree", &success);
    }


    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsFree",
                libvirt);
        return;
    }

    virTypedParamsFreeSymbol(params,
                             nparams);
}

typedef virTypedParameterPtr
(*virTypedParamsGetType)(virTypedParameterPtr params,
                         int nparams,
                         const char * name);

virTypedParameterPtr
virTypedParamsGetWrapper(virTypedParameterPtr params,
                         int nparams,
                         const char * name,
                         virErrorPtr err)
{
    static virTypedParamsGetType virTypedParamsGetSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetSymbol = libvirtSymbol(libvirt, "virTypedParamsGet", &success);
    }

    virTypedParameterPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGet",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetSymbol(params,
                                  nparams,
                                  name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetBooleanType)(virTypedParameterPtr params,
                                int nparams,
                                const char * name,
                                int * value);

int
virTypedParamsGetBooleanWrapper(virTypedParameterPtr params,
                                int nparams,
                                const char * name,
                                int * value,
                                virErrorPtr err)
{
    static virTypedParamsGetBooleanType virTypedParamsGetBooleanSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetBooleanSymbol = libvirtSymbol(libvirt, "virTypedParamsGetBoolean", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetBoolean",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetBooleanSymbol(params,
                                         nparams,
                                         name,
                                         value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetDoubleType)(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               double * value);

int
virTypedParamsGetDoubleWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               double * value,
                               virErrorPtr err)
{
    static virTypedParamsGetDoubleType virTypedParamsGetDoubleSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetDoubleSymbol = libvirtSymbol(libvirt, "virTypedParamsGetDouble", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetDouble",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetDoubleSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetIntType)(virTypedParameterPtr params,
                            int nparams,
                            const char * name,
                            int * value);

int
virTypedParamsGetIntWrapper(virTypedParameterPtr params,
                            int nparams,
                            const char * name,
                            int * value,
                            virErrorPtr err)
{
    static virTypedParamsGetIntType virTypedParamsGetIntSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetIntSymbol = libvirtSymbol(libvirt, "virTypedParamsGetInt", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetInt",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetIntSymbol(params,
                                     nparams,
                                     name,
                                     value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetLLongType)(virTypedParameterPtr params,
                              int nparams,
                              const char * name,
                              long long * value);

int
virTypedParamsGetLLongWrapper(virTypedParameterPtr params,
                              int nparams,
                              const char * name,
                              long long * value,
                              virErrorPtr err)
{
    static virTypedParamsGetLLongType virTypedParamsGetLLongSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetLLongSymbol = libvirtSymbol(libvirt, "virTypedParamsGetLLong", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetLLong",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetLLongSymbol(params,
                                       nparams,
                                       name,
                                       value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetStringType)(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               const char ** value);

int
virTypedParamsGetStringWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               const char ** value,
                               virErrorPtr err)
{
    static virTypedParamsGetStringType virTypedParamsGetStringSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetStringSymbol = libvirtSymbol(libvirt, "virTypedParamsGetString", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetString",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetStringSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetUIntType)(virTypedParameterPtr params,
                             int nparams,
                             const char * name,
                             unsigned int * value);

int
virTypedParamsGetUIntWrapper(virTypedParameterPtr params,
                             int nparams,
                             const char * name,
                             unsigned int * value,
                             virErrorPtr err)
{
    static virTypedParamsGetUIntType virTypedParamsGetUIntSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetUIntSymbol = libvirtSymbol(libvirt, "virTypedParamsGetUInt", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetUInt",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetUIntSymbol(params,
                                      nparams,
                                      name,
                                      value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetULLongType)(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               unsigned long long * value);

int
virTypedParamsGetULLongWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               unsigned long long * value,
                               virErrorPtr err)
{
    static virTypedParamsGetULLongType virTypedParamsGetULLongSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virTypedParamsGetULLongSymbol = libvirtSymbol(libvirt, "virTypedParamsGetULLong", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virTypedParamsGetULLong",
                libvirt);
        return ret;
    }

    ret = virTypedParamsGetULLongSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcEnterCGroupType)(virDomainPtr domain,
                               unsigned int flags);

int
virDomainLxcEnterCGroupWrapper(virDomainPtr domain,
                               unsigned int flags,
                               virErrorPtr err)
{
    static virDomainLxcEnterCGroupType virDomainLxcEnterCGroupSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLxcEnterCGroupSymbol = libvirtSymbol(lxc, "virDomainLxcEnterCGroup", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLxcEnterCGroup",
                lxc);
        return ret;
    }

    ret = virDomainLxcEnterCGroupSymbol(domain,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcEnterNamespaceType)(virDomainPtr domain,
                                  unsigned int nfdlist,
                                  int * fdlist,
                                  unsigned int * noldfdlist,
                                  int ** oldfdlist,
                                  unsigned int flags);

int
virDomainLxcEnterNamespaceWrapper(virDomainPtr domain,
                                  unsigned int nfdlist,
                                  int * fdlist,
                                  unsigned int * noldfdlist,
                                  int ** oldfdlist,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    static virDomainLxcEnterNamespaceType virDomainLxcEnterNamespaceSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLxcEnterNamespaceSymbol = libvirtSymbol(lxc, "virDomainLxcEnterNamespace", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLxcEnterNamespace",
                lxc);
        return ret;
    }

    ret = virDomainLxcEnterNamespaceSymbol(domain,
                                           nfdlist,
                                           fdlist,
                                           noldfdlist,
                                           oldfdlist,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcEnterSecurityLabelType)(virSecurityModelPtr model,
                                      virSecurityLabelPtr label,
                                      virSecurityLabelPtr oldlabel,
                                      unsigned int flags);

int
virDomainLxcEnterSecurityLabelWrapper(virSecurityModelPtr model,
                                      virSecurityLabelPtr label,
                                      virSecurityLabelPtr oldlabel,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    static virDomainLxcEnterSecurityLabelType virDomainLxcEnterSecurityLabelSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLxcEnterSecurityLabelSymbol = libvirtSymbol(lxc, "virDomainLxcEnterSecurityLabel", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLxcEnterSecurityLabel",
                lxc);
        return ret;
    }

    ret = virDomainLxcEnterSecurityLabelSymbol(model,
                                               label,
                                               oldlabel,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcOpenNamespaceType)(virDomainPtr domain,
                                 int ** fdlist,
                                 unsigned int flags);

int
virDomainLxcOpenNamespaceWrapper(virDomainPtr domain,
                                 int ** fdlist,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainLxcOpenNamespaceType virDomainLxcOpenNamespaceSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainLxcOpenNamespaceSymbol = libvirtSymbol(lxc, "virDomainLxcOpenNamespace", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainLxcOpenNamespace",
                lxc);
        return ret;
    }

    ret = virDomainLxcOpenNamespaceSymbol(domain,
                                          fdlist,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainQemuMonitorEventDeregisterType)(virConnectPtr conn,
                                                  int callbackID);

int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err)
{
    static virConnectDomainQemuMonitorEventDeregisterType virConnectDomainQemuMonitorEventDeregisterSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainQemuMonitorEventDeregisterSymbol = libvirtSymbol(qemu, "virConnectDomainQemuMonitorEventDeregister", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainQemuMonitorEventDeregister",
                qemu);
        return ret;
    }

    ret = virConnectDomainQemuMonitorEventDeregisterSymbol(conn,
                                                           callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainQemuMonitorEventRegisterType)(virConnectPtr conn,
                                                virDomainPtr dom,
                                                const char * event,
                                                virConnectDomainQemuMonitorEventCallback cb,
                                                void * opaque,
                                                virFreeCallback freecb,
                                                unsigned int flags);

int
virConnectDomainQemuMonitorEventRegisterWrapper(virConnectPtr conn,
                                                virDomainPtr dom,
                                                const char * event,
                                                virConnectDomainQemuMonitorEventCallback cb,
                                                void * opaque,
                                                virFreeCallback freecb,
                                                unsigned int flags,
                                                virErrorPtr err)
{
    static virConnectDomainQemuMonitorEventRegisterType virConnectDomainQemuMonitorEventRegisterSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virConnectDomainQemuMonitorEventRegisterSymbol = libvirtSymbol(qemu, "virConnectDomainQemuMonitorEventRegister", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virConnectDomainQemuMonitorEventRegister",
                qemu);
        return ret;
    }

    ret = virConnectDomainQemuMonitorEventRegisterSymbol(conn,
                                                         dom,
                                                         event,
                                                         cb,
                                                         opaque,
                                                         freecb,
                                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainQemuAgentCommandType)(virDomainPtr domain,
                                 const char * cmd,
                                 int timeout,
                                 unsigned int flags);

char *
virDomainQemuAgentCommandWrapper(virDomainPtr domain,
                                 const char * cmd,
                                 int timeout,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    static virDomainQemuAgentCommandType virDomainQemuAgentCommandSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainQemuAgentCommandSymbol = libvirtSymbol(qemu, "virDomainQemuAgentCommand", &success);
    }

    char * ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainQemuAgentCommand",
                qemu);
        return ret;
    }

    ret = virDomainQemuAgentCommandSymbol(domain,
                                          cmd,
                                          timeout,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainQemuAttachType)(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags);

virDomainPtr
virDomainQemuAttachWrapper(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags,
                           virErrorPtr err)
{
    static virDomainQemuAttachType virDomainQemuAttachSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainQemuAttachSymbol = libvirtSymbol(qemu, "virDomainQemuAttach", &success);
    }

    virDomainPtr ret = NULL;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainQemuAttach",
                qemu);
        return ret;
    }

    ret = virDomainQemuAttachSymbol(conn,
                                    pid_value,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainQemuMonitorCommandType)(virDomainPtr domain,
                                   const char * cmd,
                                   char ** result,
                                   unsigned int flags);

int
virDomainQemuMonitorCommandWrapper(virDomainPtr domain,
                                   const char * cmd,
                                   char ** result,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    static virDomainQemuMonitorCommandType virDomainQemuMonitorCommandSymbol;
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        libvirtLoadOnce();
        virDomainQemuMonitorCommandSymbol = libvirtSymbol(qemu, "virDomainQemuMonitorCommand", &success);
    }

    int ret = -1;
    if (!success) {
        fprintf(stderr,
                "%p can't call virDomainQemuMonitorCommand",
                qemu);
        return ret;
    }

    ret = virDomainQemuMonitorCommandSymbol(domain,
                                            cmd,
                                            result,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}



