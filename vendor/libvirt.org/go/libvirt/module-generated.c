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

static void * libvirtSymbol(void *handle, const char *name);
static void libvirtLoadVariables(void)
{
    if (libvirt != NULL) {
        virConnectAuthPtrDefaultVar = libvirtSymbol(libvirt, "virConnectAuthPtrDefault");

    }
}

static void
libvirtLoad(void)
{
    if (libvirt == NULL) {
        libvirt = dlopen("libvirt.so", RTLD_NOW|RTLD_LOCAL);
        assert(libvirt != NULL);
        libvirtLoadVariables();
    }
    if (qemu == NULL) {
        qemu = dlopen("libvirt-qemu.so", RTLD_NOW|RTLD_LOCAL);
        assert(qemu != NULL);
    }
    if (lxc == NULL) {
        lxc = dlopen("libvirt-lxc.so", RTLD_NOW|RTLD_LOCAL);
        assert(lxc != NULL);
    }
}

static void *
libvirtSymbol(void *handle, const char *name)
{
    assert(handle != NULL);
    return dlsym(handle, name);
}

typedef int
(*virCopyLastErrorType)(virErrorPtr to);

int
virCopyLastErrorWrapper(virErrorPtr to) {
    static virCopyLastErrorType virCopyLastErrorSymbol;
    static bool virCopyLastErrorSymbolInit;
    int ret;
    if (!virCopyLastErrorSymbolInit) {
        libvirtLoad();
        virCopyLastErrorSymbol = libvirtSymbol(libvirt, "virCopyLastError");
        virCopyLastErrorSymbolInit = (virCopyLastErrorSymbol != NULL);
    }
    if (!virCopyLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    ret = virCopyLastErrorSymbol(to);
    return ret;
}
typedef int
(*virConnCopyLastErrorType)(virConnectPtr conn,
                            virErrorPtr to);

int
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err) {
    static virConnCopyLastErrorType virConnCopyLastErrorSymbol;
    static bool virConnCopyLastErrorSymbolInit;
    int ret;
    if (!virConnCopyLastErrorSymbolInit) {
        libvirtLoad();
        virConnCopyLastErrorSymbol = libvirtSymbol(libvirt, "virConnCopyLastError");
        virConnCopyLastErrorSymbolInit = (virConnCopyLastErrorSymbol != NULL);
    }
    if (!virConnCopyLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virConnGetLastErrorType virConnGetLastErrorSymbol;
    static bool virConnGetLastErrorSymbolInit;
    virErrorPtr ret;
    if (!virConnGetLastErrorSymbolInit) {
        libvirtLoad();
        virConnGetLastErrorSymbol = libvirtSymbol(libvirt, "virConnGetLastError");
        virConnGetLastErrorSymbolInit = (virConnGetLastErrorSymbol != NULL);
    }
    if (!virConnGetLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virConnResetLastErrorWrapper(virConnectPtr conn) {
    static virConnResetLastErrorType virConnResetLastErrorSymbol;
    static bool virConnResetLastErrorSymbolInit;

    if (!virConnResetLastErrorSymbolInit) {
        libvirtLoad();
        virConnResetLastErrorSymbol = libvirtSymbol(libvirt, "virConnResetLastError");
        virConnResetLastErrorSymbolInit = (virConnResetLastErrorSymbol != NULL);
    }
    if (!virConnResetLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorFunc handler) {
    static virConnSetErrorFuncType virConnSetErrorFuncSymbol;
    static bool virConnSetErrorFuncSymbolInit;

    if (!virConnSetErrorFuncSymbolInit) {
        libvirtLoad();
        virConnSetErrorFuncSymbol = libvirtSymbol(libvirt, "virConnSetErrorFunc");
        virConnSetErrorFuncSymbolInit = (virConnSetErrorFuncSymbol != NULL);
    }
    if (!virConnSetErrorFuncSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectBaselineCPUType virConnectBaselineCPUSymbol;
    static bool virConnectBaselineCPUSymbolInit;
    char * ret;
    if (!virConnectBaselineCPUSymbolInit) {
        libvirtLoad();
        virConnectBaselineCPUSymbol = libvirtSymbol(libvirt, "virConnectBaselineCPU");
        virConnectBaselineCPUSymbolInit = (virConnectBaselineCPUSymbol != NULL);
    }
    if (!virConnectBaselineCPUSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virConnectBaselineHypervisorCPUType virConnectBaselineHypervisorCPUSymbol;
    static bool virConnectBaselineHypervisorCPUSymbolInit;
    char * ret;
    if (!virConnectBaselineHypervisorCPUSymbolInit) {
        libvirtLoad();
        virConnectBaselineHypervisorCPUSymbol = libvirtSymbol(libvirt, "virConnectBaselineHypervisorCPU");
        virConnectBaselineHypervisorCPUSymbolInit = (virConnectBaselineHypervisorCPUSymbol != NULL);
    }
    if (!virConnectBaselineHypervisorCPUSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virConnectCloseType virConnectCloseSymbol;
    static bool virConnectCloseSymbolInit;
    int ret;
    if (!virConnectCloseSymbolInit) {
        libvirtLoad();
        virConnectCloseSymbol = libvirtSymbol(libvirt, "virConnectClose");
        virConnectCloseSymbolInit = (virConnectCloseSymbol != NULL);
    }
    if (!virConnectCloseSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virConnectCompareCPUType virConnectCompareCPUSymbol;
    static bool virConnectCompareCPUSymbolInit;
    int ret;
    if (!virConnectCompareCPUSymbolInit) {
        libvirtLoad();
        virConnectCompareCPUSymbol = libvirtSymbol(libvirt, "virConnectCompareCPU");
        virConnectCompareCPUSymbolInit = (virConnectCompareCPUSymbol != NULL);
    }
    if (!virConnectCompareCPUSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virConnectCompareHypervisorCPUType virConnectCompareHypervisorCPUSymbol;
    static bool virConnectCompareHypervisorCPUSymbolInit;
    int ret;
    if (!virConnectCompareHypervisorCPUSymbolInit) {
        libvirtLoad();
        virConnectCompareHypervisorCPUSymbol = libvirtSymbol(libvirt, "virConnectCompareHypervisorCPU");
        virConnectCompareHypervisorCPUSymbolInit = (virConnectCompareHypervisorCPUSymbol != NULL);
    }
    if (!virConnectCompareHypervisorCPUSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virConnectDomainEventDeregisterType virConnectDomainEventDeregisterSymbol;
    static bool virConnectDomainEventDeregisterSymbolInit;
    int ret;
    if (!virConnectDomainEventDeregisterSymbolInit) {
        libvirtLoad();
        virConnectDomainEventDeregisterSymbol = libvirtSymbol(libvirt, "virConnectDomainEventDeregister");
        virConnectDomainEventDeregisterSymbolInit = (virConnectDomainEventDeregisterSymbol != NULL);
    }
    if (!virConnectDomainEventDeregisterSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                          virErrorPtr err) {
    static virConnectDomainEventDeregisterAnyType virConnectDomainEventDeregisterAnySymbol;
    static bool virConnectDomainEventDeregisterAnySymbolInit;
    int ret;
    if (!virConnectDomainEventDeregisterAnySymbolInit) {
        libvirtLoad();
        virConnectDomainEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectDomainEventDeregisterAny");
        virConnectDomainEventDeregisterAnySymbolInit = (virConnectDomainEventDeregisterAnySymbol != NULL);
    }
    if (!virConnectDomainEventDeregisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virConnectDomainEventRegisterType virConnectDomainEventRegisterSymbol;
    static bool virConnectDomainEventRegisterSymbolInit;
    int ret;
    if (!virConnectDomainEventRegisterSymbolInit) {
        libvirtLoad();
        virConnectDomainEventRegisterSymbol = libvirtSymbol(libvirt, "virConnectDomainEventRegister");
        virConnectDomainEventRegisterSymbolInit = (virConnectDomainEventRegisterSymbol != NULL);
    }
    if (!virConnectDomainEventRegisterSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virConnectDomainEventRegisterAnyType virConnectDomainEventRegisterAnySymbol;
    static bool virConnectDomainEventRegisterAnySymbolInit;
    int ret;
    if (!virConnectDomainEventRegisterAnySymbolInit) {
        libvirtLoad();
        virConnectDomainEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectDomainEventRegisterAny");
        virConnectDomainEventRegisterAnySymbolInit = (virConnectDomainEventRegisterAnySymbol != NULL);
    }
    if (!virConnectDomainEventRegisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virConnectDomainXMLFromNativeType virConnectDomainXMLFromNativeSymbol;
    static bool virConnectDomainXMLFromNativeSymbolInit;
    char * ret;
    if (!virConnectDomainXMLFromNativeSymbolInit) {
        libvirtLoad();
        virConnectDomainXMLFromNativeSymbol = libvirtSymbol(libvirt, "virConnectDomainXMLFromNative");
        virConnectDomainXMLFromNativeSymbolInit = (virConnectDomainXMLFromNativeSymbol != NULL);
    }
    if (!virConnectDomainXMLFromNativeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virConnectDomainXMLToNativeType virConnectDomainXMLToNativeSymbol;
    static bool virConnectDomainXMLToNativeSymbolInit;
    char * ret;
    if (!virConnectDomainXMLToNativeSymbolInit) {
        libvirtLoad();
        virConnectDomainXMLToNativeSymbol = libvirtSymbol(libvirt, "virConnectDomainXMLToNative");
        virConnectDomainXMLToNativeSymbolInit = (virConnectDomainXMLToNativeSymbol != NULL);
    }
    if (!virConnectDomainXMLToNativeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virConnectFindStoragePoolSourcesType virConnectFindStoragePoolSourcesSymbol;
    static bool virConnectFindStoragePoolSourcesSymbolInit;
    char * ret;
    if (!virConnectFindStoragePoolSourcesSymbolInit) {
        libvirtLoad();
        virConnectFindStoragePoolSourcesSymbol = libvirtSymbol(libvirt, "virConnectFindStoragePoolSources");
        virConnectFindStoragePoolSourcesSymbolInit = (virConnectFindStoragePoolSourcesSymbol != NULL);
    }
    if (!virConnectFindStoragePoolSourcesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virConnectGetAllDomainStatsType virConnectGetAllDomainStatsSymbol;
    static bool virConnectGetAllDomainStatsSymbolInit;
    int ret;
    if (!virConnectGetAllDomainStatsSymbolInit) {
        libvirtLoad();
        virConnectGetAllDomainStatsSymbol = libvirtSymbol(libvirt, "virConnectGetAllDomainStats");
        virConnectGetAllDomainStatsSymbolInit = (virConnectGetAllDomainStatsSymbol != NULL);
    }
    if (!virConnectGetAllDomainStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virConnectGetCPUModelNamesType virConnectGetCPUModelNamesSymbol;
    static bool virConnectGetCPUModelNamesSymbolInit;
    int ret;
    if (!virConnectGetCPUModelNamesSymbolInit) {
        libvirtLoad();
        virConnectGetCPUModelNamesSymbol = libvirtSymbol(libvirt, "virConnectGetCPUModelNames");
        virConnectGetCPUModelNamesSymbolInit = (virConnectGetCPUModelNamesSymbol != NULL);
    }
    if (!virConnectGetCPUModelNamesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virConnectGetCapabilitiesType virConnectGetCapabilitiesSymbol;
    static bool virConnectGetCapabilitiesSymbolInit;
    char * ret;
    if (!virConnectGetCapabilitiesSymbolInit) {
        libvirtLoad();
        virConnectGetCapabilitiesSymbol = libvirtSymbol(libvirt, "virConnectGetCapabilities");
        virConnectGetCapabilitiesSymbolInit = (virConnectGetCapabilitiesSymbol != NULL);
    }
    if (!virConnectGetCapabilitiesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virConnectGetDomainCapabilitiesType virConnectGetDomainCapabilitiesSymbol;
    static bool virConnectGetDomainCapabilitiesSymbolInit;
    char * ret;
    if (!virConnectGetDomainCapabilitiesSymbolInit) {
        libvirtLoad();
        virConnectGetDomainCapabilitiesSymbol = libvirtSymbol(libvirt, "virConnectGetDomainCapabilities");
        virConnectGetDomainCapabilitiesSymbolInit = (virConnectGetDomainCapabilitiesSymbol != NULL);
    }
    if (!virConnectGetDomainCapabilitiesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectGetHostnameType virConnectGetHostnameSymbol;
    static bool virConnectGetHostnameSymbolInit;
    char * ret;
    if (!virConnectGetHostnameSymbolInit) {
        libvirtLoad();
        virConnectGetHostnameSymbol = libvirtSymbol(libvirt, "virConnectGetHostname");
        virConnectGetHostnameSymbolInit = (virConnectGetHostnameSymbol != NULL);
    }
    if (!virConnectGetHostnameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virConnectGetLibVersionType virConnectGetLibVersionSymbol;
    static bool virConnectGetLibVersionSymbolInit;
    int ret;
    if (!virConnectGetLibVersionSymbolInit) {
        libvirtLoad();
        virConnectGetLibVersionSymbol = libvirtSymbol(libvirt, "virConnectGetLibVersion");
        virConnectGetLibVersionSymbolInit = (virConnectGetLibVersionSymbol != NULL);
    }
    if (!virConnectGetLibVersionSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectGetMaxVcpusType virConnectGetMaxVcpusSymbol;
    static bool virConnectGetMaxVcpusSymbolInit;
    int ret;
    if (!virConnectGetMaxVcpusSymbolInit) {
        libvirtLoad();
        virConnectGetMaxVcpusSymbol = libvirtSymbol(libvirt, "virConnectGetMaxVcpus");
        virConnectGetMaxVcpusSymbolInit = (virConnectGetMaxVcpusSymbol != NULL);
    }
    if (!virConnectGetMaxVcpusSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                            virErrorPtr err) {
    static virConnectGetStoragePoolCapabilitiesType virConnectGetStoragePoolCapabilitiesSymbol;
    static bool virConnectGetStoragePoolCapabilitiesSymbolInit;
    char * ret;
    if (!virConnectGetStoragePoolCapabilitiesSymbolInit) {
        libvirtLoad();
        virConnectGetStoragePoolCapabilitiesSymbol = libvirtSymbol(libvirt, "virConnectGetStoragePoolCapabilities");
        virConnectGetStoragePoolCapabilitiesSymbolInit = (virConnectGetStoragePoolCapabilitiesSymbol != NULL);
    }
    if (!virConnectGetStoragePoolCapabilitiesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virConnectGetSysinfoType virConnectGetSysinfoSymbol;
    static bool virConnectGetSysinfoSymbolInit;
    char * ret;
    if (!virConnectGetSysinfoSymbolInit) {
        libvirtLoad();
        virConnectGetSysinfoSymbol = libvirtSymbol(libvirt, "virConnectGetSysinfo");
        virConnectGetSysinfoSymbolInit = (virConnectGetSysinfoSymbol != NULL);
    }
    if (!virConnectGetSysinfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virConnectGetTypeType virConnectGetTypeSymbol;
    static bool virConnectGetTypeSymbolInit;
    const char * ret;
    if (!virConnectGetTypeSymbolInit) {
        libvirtLoad();
        virConnectGetTypeSymbol = libvirtSymbol(libvirt, "virConnectGetType");
        virConnectGetTypeSymbolInit = (virConnectGetTypeSymbol != NULL);
    }
    if (!virConnectGetTypeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virConnectGetURIType virConnectGetURISymbol;
    static bool virConnectGetURISymbolInit;
    char * ret;
    if (!virConnectGetURISymbolInit) {
        libvirtLoad();
        virConnectGetURISymbol = libvirtSymbol(libvirt, "virConnectGetURI");
        virConnectGetURISymbolInit = (virConnectGetURISymbol != NULL);
    }
    if (!virConnectGetURISymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virConnectGetVersionType virConnectGetVersionSymbol;
    static bool virConnectGetVersionSymbolInit;
    int ret;
    if (!virConnectGetVersionSymbolInit) {
        libvirtLoad();
        virConnectGetVersionSymbol = libvirtSymbol(libvirt, "virConnectGetVersion");
        virConnectGetVersionSymbolInit = (virConnectGetVersionSymbol != NULL);
    }
    if (!virConnectGetVersionSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virConnectIsAliveType virConnectIsAliveSymbol;
    static bool virConnectIsAliveSymbolInit;
    int ret;
    if (!virConnectIsAliveSymbolInit) {
        libvirtLoad();
        virConnectIsAliveSymbol = libvirtSymbol(libvirt, "virConnectIsAlive");
        virConnectIsAliveSymbolInit = (virConnectIsAliveSymbol != NULL);
    }
    if (!virConnectIsAliveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectIsEncryptedType virConnectIsEncryptedSymbol;
    static bool virConnectIsEncryptedSymbolInit;
    int ret;
    if (!virConnectIsEncryptedSymbolInit) {
        libvirtLoad();
        virConnectIsEncryptedSymbol = libvirtSymbol(libvirt, "virConnectIsEncrypted");
        virConnectIsEncryptedSymbolInit = (virConnectIsEncryptedSymbol != NULL);
    }
    if (!virConnectIsEncryptedSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virConnectIsSecureType virConnectIsSecureSymbol;
    static bool virConnectIsSecureSymbolInit;
    int ret;
    if (!virConnectIsSecureSymbolInit) {
        libvirtLoad();
        virConnectIsSecureSymbol = libvirtSymbol(libvirt, "virConnectIsSecure");
        virConnectIsSecureSymbolInit = (virConnectIsSecureSymbol != NULL);
    }
    if (!virConnectIsSecureSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virConnectListAllDomainsType virConnectListAllDomainsSymbol;
    static bool virConnectListAllDomainsSymbolInit;
    int ret;
    if (!virConnectListAllDomainsSymbolInit) {
        libvirtLoad();
        virConnectListAllDomainsSymbol = libvirtSymbol(libvirt, "virConnectListAllDomains");
        virConnectListAllDomainsSymbolInit = (virConnectListAllDomainsSymbol != NULL);
    }
    if (!virConnectListAllDomainsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virConnectListAllInterfacesType virConnectListAllInterfacesSymbol;
    static bool virConnectListAllInterfacesSymbolInit;
    int ret;
    if (!virConnectListAllInterfacesSymbolInit) {
        libvirtLoad();
        virConnectListAllInterfacesSymbol = libvirtSymbol(libvirt, "virConnectListAllInterfaces");
        virConnectListAllInterfacesSymbolInit = (virConnectListAllInterfacesSymbol != NULL);
    }
    if (!virConnectListAllInterfacesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                         virErrorPtr err) {
    static virConnectListAllNWFilterBindingsType virConnectListAllNWFilterBindingsSymbol;
    static bool virConnectListAllNWFilterBindingsSymbolInit;
    int ret;
    if (!virConnectListAllNWFilterBindingsSymbolInit) {
        libvirtLoad();
        virConnectListAllNWFilterBindingsSymbol = libvirtSymbol(libvirt, "virConnectListAllNWFilterBindings");
        virConnectListAllNWFilterBindingsSymbolInit = (virConnectListAllNWFilterBindingsSymbol != NULL);
    }
    if (!virConnectListAllNWFilterBindingsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virConnectListAllNWFiltersType virConnectListAllNWFiltersSymbol;
    static bool virConnectListAllNWFiltersSymbolInit;
    int ret;
    if (!virConnectListAllNWFiltersSymbolInit) {
        libvirtLoad();
        virConnectListAllNWFiltersSymbol = libvirtSymbol(libvirt, "virConnectListAllNWFilters");
        virConnectListAllNWFiltersSymbolInit = (virConnectListAllNWFiltersSymbol != NULL);
    }
    if (!virConnectListAllNWFiltersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virConnectListAllNetworksType virConnectListAllNetworksSymbol;
    static bool virConnectListAllNetworksSymbolInit;
    int ret;
    if (!virConnectListAllNetworksSymbolInit) {
        libvirtLoad();
        virConnectListAllNetworksSymbol = libvirtSymbol(libvirt, "virConnectListAllNetworks");
        virConnectListAllNetworksSymbolInit = (virConnectListAllNetworksSymbol != NULL);
    }
    if (!virConnectListAllNetworksSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virConnectListAllNodeDevicesType virConnectListAllNodeDevicesSymbol;
    static bool virConnectListAllNodeDevicesSymbolInit;
    int ret;
    if (!virConnectListAllNodeDevicesSymbolInit) {
        libvirtLoad();
        virConnectListAllNodeDevicesSymbol = libvirtSymbol(libvirt, "virConnectListAllNodeDevices");
        virConnectListAllNodeDevicesSymbolInit = (virConnectListAllNodeDevicesSymbol != NULL);
    }
    if (!virConnectListAllNodeDevicesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virConnectListAllSecretsType virConnectListAllSecretsSymbol;
    static bool virConnectListAllSecretsSymbolInit;
    int ret;
    if (!virConnectListAllSecretsSymbolInit) {
        libvirtLoad();
        virConnectListAllSecretsSymbol = libvirtSymbol(libvirt, "virConnectListAllSecrets");
        virConnectListAllSecretsSymbolInit = (virConnectListAllSecretsSymbol != NULL);
    }
    if (!virConnectListAllSecretsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virConnectListAllStoragePoolsType virConnectListAllStoragePoolsSymbol;
    static bool virConnectListAllStoragePoolsSymbolInit;
    int ret;
    if (!virConnectListAllStoragePoolsSymbolInit) {
        libvirtLoad();
        virConnectListAllStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectListAllStoragePools");
        virConnectListAllStoragePoolsSymbolInit = (virConnectListAllStoragePoolsSymbol != NULL);
    }
    if (!virConnectListAllStoragePoolsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virConnectListDefinedDomainsType virConnectListDefinedDomainsSymbol;
    static bool virConnectListDefinedDomainsSymbolInit;
    int ret;
    if (!virConnectListDefinedDomainsSymbolInit) {
        libvirtLoad();
        virConnectListDefinedDomainsSymbol = libvirtSymbol(libvirt, "virConnectListDefinedDomains");
        virConnectListDefinedDomainsSymbolInit = (virConnectListDefinedDomainsSymbol != NULL);
    }
    if (!virConnectListDefinedDomainsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virConnectListDefinedInterfacesType virConnectListDefinedInterfacesSymbol;
    static bool virConnectListDefinedInterfacesSymbolInit;
    int ret;
    if (!virConnectListDefinedInterfacesSymbolInit) {
        libvirtLoad();
        virConnectListDefinedInterfacesSymbol = libvirtSymbol(libvirt, "virConnectListDefinedInterfaces");
        virConnectListDefinedInterfacesSymbolInit = (virConnectListDefinedInterfacesSymbol != NULL);
    }
    if (!virConnectListDefinedInterfacesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virConnectListDefinedNetworksType virConnectListDefinedNetworksSymbol;
    static bool virConnectListDefinedNetworksSymbolInit;
    int ret;
    if (!virConnectListDefinedNetworksSymbolInit) {
        libvirtLoad();
        virConnectListDefinedNetworksSymbol = libvirtSymbol(libvirt, "virConnectListDefinedNetworks");
        virConnectListDefinedNetworksSymbolInit = (virConnectListDefinedNetworksSymbol != NULL);
    }
    if (!virConnectListDefinedNetworksSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                         virErrorPtr err) {
    static virConnectListDefinedStoragePoolsType virConnectListDefinedStoragePoolsSymbol;
    static bool virConnectListDefinedStoragePoolsSymbolInit;
    int ret;
    if (!virConnectListDefinedStoragePoolsSymbolInit) {
        libvirtLoad();
        virConnectListDefinedStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectListDefinedStoragePools");
        virConnectListDefinedStoragePoolsSymbolInit = (virConnectListDefinedStoragePoolsSymbol != NULL);
    }
    if (!virConnectListDefinedStoragePoolsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectListDomainsType virConnectListDomainsSymbol;
    static bool virConnectListDomainsSymbolInit;
    int ret;
    if (!virConnectListDomainsSymbolInit) {
        libvirtLoad();
        virConnectListDomainsSymbol = libvirtSymbol(libvirt, "virConnectListDomains");
        virConnectListDomainsSymbolInit = (virConnectListDomainsSymbol != NULL);
    }
    if (!virConnectListDomainsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virConnectListInterfacesType virConnectListInterfacesSymbol;
    static bool virConnectListInterfacesSymbolInit;
    int ret;
    if (!virConnectListInterfacesSymbolInit) {
        libvirtLoad();
        virConnectListInterfacesSymbol = libvirtSymbol(libvirt, "virConnectListInterfaces");
        virConnectListInterfacesSymbolInit = (virConnectListInterfacesSymbol != NULL);
    }
    if (!virConnectListInterfacesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virConnectListNWFiltersType virConnectListNWFiltersSymbol;
    static bool virConnectListNWFiltersSymbolInit;
    int ret;
    if (!virConnectListNWFiltersSymbolInit) {
        libvirtLoad();
        virConnectListNWFiltersSymbol = libvirtSymbol(libvirt, "virConnectListNWFilters");
        virConnectListNWFiltersSymbolInit = (virConnectListNWFiltersSymbol != NULL);
    }
    if (!virConnectListNWFiltersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virConnectListNetworksType virConnectListNetworksSymbol;
    static bool virConnectListNetworksSymbolInit;
    int ret;
    if (!virConnectListNetworksSymbolInit) {
        libvirtLoad();
        virConnectListNetworksSymbol = libvirtSymbol(libvirt, "virConnectListNetworks");
        virConnectListNetworksSymbolInit = (virConnectListNetworksSymbol != NULL);
    }
    if (!virConnectListNetworksSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectListSecretsType virConnectListSecretsSymbol;
    static bool virConnectListSecretsSymbolInit;
    int ret;
    if (!virConnectListSecretsSymbolInit) {
        libvirtLoad();
        virConnectListSecretsSymbol = libvirtSymbol(libvirt, "virConnectListSecrets");
        virConnectListSecretsSymbolInit = (virConnectListSecretsSymbol != NULL);
    }
    if (!virConnectListSecretsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virConnectListStoragePoolsType virConnectListStoragePoolsSymbol;
    static bool virConnectListStoragePoolsSymbolInit;
    int ret;
    if (!virConnectListStoragePoolsSymbolInit) {
        libvirtLoad();
        virConnectListStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectListStoragePools");
        virConnectListStoragePoolsSymbolInit = (virConnectListStoragePoolsSymbol != NULL);
    }
    if (!virConnectListStoragePoolsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                           virErrorPtr err) {
    static virConnectNetworkEventDeregisterAnyType virConnectNetworkEventDeregisterAnySymbol;
    static bool virConnectNetworkEventDeregisterAnySymbolInit;
    int ret;
    if (!virConnectNetworkEventDeregisterAnySymbolInit) {
        libvirtLoad();
        virConnectNetworkEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectNetworkEventDeregisterAny");
        virConnectNetworkEventDeregisterAnySymbolInit = (virConnectNetworkEventDeregisterAnySymbol != NULL);
    }
    if (!virConnectNetworkEventDeregisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                         virErrorPtr err) {
    static virConnectNetworkEventRegisterAnyType virConnectNetworkEventRegisterAnySymbol;
    static bool virConnectNetworkEventRegisterAnySymbolInit;
    int ret;
    if (!virConnectNetworkEventRegisterAnySymbolInit) {
        libvirtLoad();
        virConnectNetworkEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectNetworkEventRegisterAny");
        virConnectNetworkEventRegisterAnySymbolInit = (virConnectNetworkEventRegisterAnySymbol != NULL);
    }
    if (!virConnectNetworkEventRegisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                              virErrorPtr err) {
    static virConnectNodeDeviceEventDeregisterAnyType virConnectNodeDeviceEventDeregisterAnySymbol;
    static bool virConnectNodeDeviceEventDeregisterAnySymbolInit;
    int ret;
    if (!virConnectNodeDeviceEventDeregisterAnySymbolInit) {
        libvirtLoad();
        virConnectNodeDeviceEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectNodeDeviceEventDeregisterAny");
        virConnectNodeDeviceEventDeregisterAnySymbolInit = (virConnectNodeDeviceEventDeregisterAnySymbol != NULL);
    }
    if (!virConnectNodeDeviceEventDeregisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                            virErrorPtr err) {
    static virConnectNodeDeviceEventRegisterAnyType virConnectNodeDeviceEventRegisterAnySymbol;
    static bool virConnectNodeDeviceEventRegisterAnySymbolInit;
    int ret;
    if (!virConnectNodeDeviceEventRegisterAnySymbolInit) {
        libvirtLoad();
        virConnectNodeDeviceEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectNodeDeviceEventRegisterAny");
        virConnectNodeDeviceEventRegisterAnySymbolInit = (virConnectNodeDeviceEventRegisterAnySymbol != NULL);
    }
    if (!virConnectNodeDeviceEventRegisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virConnectNumOfDefinedDomainsType virConnectNumOfDefinedDomainsSymbol;
    static bool virConnectNumOfDefinedDomainsSymbolInit;
    int ret;
    if (!virConnectNumOfDefinedDomainsSymbolInit) {
        libvirtLoad();
        virConnectNumOfDefinedDomainsSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedDomains");
        virConnectNumOfDefinedDomainsSymbolInit = (virConnectNumOfDefinedDomainsSymbol != NULL);
    }
    if (!virConnectNumOfDefinedDomainsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virConnectNumOfDefinedInterfacesType virConnectNumOfDefinedInterfacesSymbol;
    static bool virConnectNumOfDefinedInterfacesSymbolInit;
    int ret;
    if (!virConnectNumOfDefinedInterfacesSymbolInit) {
        libvirtLoad();
        virConnectNumOfDefinedInterfacesSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedInterfaces");
        virConnectNumOfDefinedInterfacesSymbolInit = (virConnectNumOfDefinedInterfacesSymbol != NULL);
    }
    if (!virConnectNumOfDefinedInterfacesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virConnectNumOfDefinedNetworksType virConnectNumOfDefinedNetworksSymbol;
    static bool virConnectNumOfDefinedNetworksSymbolInit;
    int ret;
    if (!virConnectNumOfDefinedNetworksSymbolInit) {
        libvirtLoad();
        virConnectNumOfDefinedNetworksSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedNetworks");
        virConnectNumOfDefinedNetworksSymbolInit = (virConnectNumOfDefinedNetworksSymbol != NULL);
    }
    if (!virConnectNumOfDefinedNetworksSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                          virErrorPtr err) {
    static virConnectNumOfDefinedStoragePoolsType virConnectNumOfDefinedStoragePoolsSymbol;
    static bool virConnectNumOfDefinedStoragePoolsSymbolInit;
    int ret;
    if (!virConnectNumOfDefinedStoragePoolsSymbolInit) {
        libvirtLoad();
        virConnectNumOfDefinedStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectNumOfDefinedStoragePools");
        virConnectNumOfDefinedStoragePoolsSymbolInit = (virConnectNumOfDefinedStoragePoolsSymbol != NULL);
    }
    if (!virConnectNumOfDefinedStoragePoolsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virConnectNumOfDomainsType virConnectNumOfDomainsSymbol;
    static bool virConnectNumOfDomainsSymbolInit;
    int ret;
    if (!virConnectNumOfDomainsSymbolInit) {
        libvirtLoad();
        virConnectNumOfDomainsSymbol = libvirtSymbol(libvirt, "virConnectNumOfDomains");
        virConnectNumOfDomainsSymbolInit = (virConnectNumOfDomainsSymbol != NULL);
    }
    if (!virConnectNumOfDomainsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virConnectNumOfInterfacesType virConnectNumOfInterfacesSymbol;
    static bool virConnectNumOfInterfacesSymbolInit;
    int ret;
    if (!virConnectNumOfInterfacesSymbolInit) {
        libvirtLoad();
        virConnectNumOfInterfacesSymbol = libvirtSymbol(libvirt, "virConnectNumOfInterfaces");
        virConnectNumOfInterfacesSymbolInit = (virConnectNumOfInterfacesSymbol != NULL);
    }
    if (!virConnectNumOfInterfacesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virConnectNumOfNWFiltersType virConnectNumOfNWFiltersSymbol;
    static bool virConnectNumOfNWFiltersSymbolInit;
    int ret;
    if (!virConnectNumOfNWFiltersSymbolInit) {
        libvirtLoad();
        virConnectNumOfNWFiltersSymbol = libvirtSymbol(libvirt, "virConnectNumOfNWFilters");
        virConnectNumOfNWFiltersSymbolInit = (virConnectNumOfNWFiltersSymbol != NULL);
    }
    if (!virConnectNumOfNWFiltersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virConnectNumOfNetworksType virConnectNumOfNetworksSymbol;
    static bool virConnectNumOfNetworksSymbolInit;
    int ret;
    if (!virConnectNumOfNetworksSymbolInit) {
        libvirtLoad();
        virConnectNumOfNetworksSymbol = libvirtSymbol(libvirt, "virConnectNumOfNetworks");
        virConnectNumOfNetworksSymbolInit = (virConnectNumOfNetworksSymbol != NULL);
    }
    if (!virConnectNumOfNetworksSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virConnectNumOfSecretsType virConnectNumOfSecretsSymbol;
    static bool virConnectNumOfSecretsSymbolInit;
    int ret;
    if (!virConnectNumOfSecretsSymbolInit) {
        libvirtLoad();
        virConnectNumOfSecretsSymbol = libvirtSymbol(libvirt, "virConnectNumOfSecrets");
        virConnectNumOfSecretsSymbolInit = (virConnectNumOfSecretsSymbol != NULL);
    }
    if (!virConnectNumOfSecretsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virConnectNumOfStoragePoolsType virConnectNumOfStoragePoolsSymbol;
    static bool virConnectNumOfStoragePoolsSymbolInit;
    int ret;
    if (!virConnectNumOfStoragePoolsSymbolInit) {
        libvirtLoad();
        virConnectNumOfStoragePoolsSymbol = libvirtSymbol(libvirt, "virConnectNumOfStoragePools");
        virConnectNumOfStoragePoolsSymbolInit = (virConnectNumOfStoragePoolsSymbol != NULL);
    }
    if (!virConnectNumOfStoragePoolsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virConnectOpenType virConnectOpenSymbol;
    static bool virConnectOpenSymbolInit;
    virConnectPtr ret;
    if (!virConnectOpenSymbolInit) {
        libvirtLoad();
        virConnectOpenSymbol = libvirtSymbol(libvirt, "virConnectOpen");
        virConnectOpenSymbolInit = (virConnectOpenSymbol != NULL);
    }
    if (!virConnectOpenSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virConnectOpenAuthType virConnectOpenAuthSymbol;
    static bool virConnectOpenAuthSymbolInit;
    virConnectPtr ret;
    if (!virConnectOpenAuthSymbolInit) {
        libvirtLoad();
        virConnectOpenAuthSymbol = libvirtSymbol(libvirt, "virConnectOpenAuth");
        virConnectOpenAuthSymbolInit = (virConnectOpenAuthSymbol != NULL);
    }
    if (!virConnectOpenAuthSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virConnectOpenReadOnlyType virConnectOpenReadOnlySymbol;
    static bool virConnectOpenReadOnlySymbolInit;
    virConnectPtr ret;
    if (!virConnectOpenReadOnlySymbolInit) {
        libvirtLoad();
        virConnectOpenReadOnlySymbol = libvirtSymbol(libvirt, "virConnectOpenReadOnly");
        virConnectOpenReadOnlySymbolInit = (virConnectOpenReadOnlySymbol != NULL);
    }
    if (!virConnectOpenReadOnlySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virConnectRefType virConnectRefSymbol;
    static bool virConnectRefSymbolInit;
    int ret;
    if (!virConnectRefSymbolInit) {
        libvirtLoad();
        virConnectRefSymbol = libvirtSymbol(libvirt, "virConnectRef");
        virConnectRefSymbolInit = (virConnectRefSymbol != NULL);
    }
    if (!virConnectRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virConnectRegisterCloseCallbackType virConnectRegisterCloseCallbackSymbol;
    static bool virConnectRegisterCloseCallbackSymbolInit;
    int ret;
    if (!virConnectRegisterCloseCallbackSymbolInit) {
        libvirtLoad();
        virConnectRegisterCloseCallbackSymbol = libvirtSymbol(libvirt, "virConnectRegisterCloseCallback");
        virConnectRegisterCloseCallbackSymbolInit = (virConnectRegisterCloseCallbackSymbol != NULL);
    }
    if (!virConnectRegisterCloseCallbackSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                          virErrorPtr err) {
    static virConnectSecretEventDeregisterAnyType virConnectSecretEventDeregisterAnySymbol;
    static bool virConnectSecretEventDeregisterAnySymbolInit;
    int ret;
    if (!virConnectSecretEventDeregisterAnySymbolInit) {
        libvirtLoad();
        virConnectSecretEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectSecretEventDeregisterAny");
        virConnectSecretEventDeregisterAnySymbolInit = (virConnectSecretEventDeregisterAnySymbol != NULL);
    }
    if (!virConnectSecretEventDeregisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virConnectSecretEventRegisterAnyType virConnectSecretEventRegisterAnySymbol;
    static bool virConnectSecretEventRegisterAnySymbolInit;
    int ret;
    if (!virConnectSecretEventRegisterAnySymbolInit) {
        libvirtLoad();
        virConnectSecretEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectSecretEventRegisterAny");
        virConnectSecretEventRegisterAnySymbolInit = (virConnectSecretEventRegisterAnySymbol != NULL);
    }
    if (!virConnectSecretEventRegisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virConnectSetIdentityType virConnectSetIdentitySymbol;
    static bool virConnectSetIdentitySymbolInit;
    int ret;
    if (!virConnectSetIdentitySymbolInit) {
        libvirtLoad();
        virConnectSetIdentitySymbol = libvirtSymbol(libvirt, "virConnectSetIdentity");
        virConnectSetIdentitySymbolInit = (virConnectSetIdentitySymbol != NULL);
    }
    if (!virConnectSetIdentitySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virConnectSetKeepAliveType virConnectSetKeepAliveSymbol;
    static bool virConnectSetKeepAliveSymbolInit;
    int ret;
    if (!virConnectSetKeepAliveSymbolInit) {
        libvirtLoad();
        virConnectSetKeepAliveSymbol = libvirtSymbol(libvirt, "virConnectSetKeepAlive");
        virConnectSetKeepAliveSymbolInit = (virConnectSetKeepAliveSymbol != NULL);
    }
    if (!virConnectSetKeepAliveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                               virErrorPtr err) {
    static virConnectStoragePoolEventDeregisterAnyType virConnectStoragePoolEventDeregisterAnySymbol;
    static bool virConnectStoragePoolEventDeregisterAnySymbolInit;
    int ret;
    if (!virConnectStoragePoolEventDeregisterAnySymbolInit) {
        libvirtLoad();
        virConnectStoragePoolEventDeregisterAnySymbol = libvirtSymbol(libvirt, "virConnectStoragePoolEventDeregisterAny");
        virConnectStoragePoolEventDeregisterAnySymbolInit = (virConnectStoragePoolEventDeregisterAnySymbol != NULL);
    }
    if (!virConnectStoragePoolEventDeregisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                             virErrorPtr err) {
    static virConnectStoragePoolEventRegisterAnyType virConnectStoragePoolEventRegisterAnySymbol;
    static bool virConnectStoragePoolEventRegisterAnySymbolInit;
    int ret;
    if (!virConnectStoragePoolEventRegisterAnySymbolInit) {
        libvirtLoad();
        virConnectStoragePoolEventRegisterAnySymbol = libvirtSymbol(libvirt, "virConnectStoragePoolEventRegisterAny");
        virConnectStoragePoolEventRegisterAnySymbolInit = (virConnectStoragePoolEventRegisterAnySymbol != NULL);
    }
    if (!virConnectStoragePoolEventRegisterAnySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                         virErrorPtr err) {
    static virConnectUnregisterCloseCallbackType virConnectUnregisterCloseCallbackSymbol;
    static bool virConnectUnregisterCloseCallbackSymbolInit;
    int ret;
    if (!virConnectUnregisterCloseCallbackSymbolInit) {
        libvirtLoad();
        virConnectUnregisterCloseCallbackSymbol = libvirtSymbol(libvirt, "virConnectUnregisterCloseCallback");
        virConnectUnregisterCloseCallbackSymbolInit = (virConnectUnregisterCloseCallbackSymbol != NULL);
    }
    if (!virConnectUnregisterCloseCallbackSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virDefaultErrorFuncWrapper(virErrorPtr err) {
    static virDefaultErrorFuncType virDefaultErrorFuncSymbol;
    static bool virDefaultErrorFuncSymbolInit;

    if (!virDefaultErrorFuncSymbolInit) {
        libvirtLoad();
        virDefaultErrorFuncSymbol = libvirtSymbol(libvirt, "virDefaultErrorFunc");
        virDefaultErrorFuncSymbolInit = (virDefaultErrorFuncSymbol != NULL);
    }
    if (!virDefaultErrorFuncSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virDefaultErrorFuncSymbol(err);

}
typedef int
(*virDomainAbortJobType)(virDomainPtr domain);

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err) {
    static virDomainAbortJobType virDomainAbortJobSymbol;
    static bool virDomainAbortJobSymbolInit;
    int ret;
    if (!virDomainAbortJobSymbolInit) {
        libvirtLoad();
        virDomainAbortJobSymbol = libvirtSymbol(libvirt, "virDomainAbortJob");
        virDomainAbortJobSymbolInit = (virDomainAbortJobSymbol != NULL);
    }
    if (!virDomainAbortJobSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainAddIOThreadType virDomainAddIOThreadSymbol;
    static bool virDomainAddIOThreadSymbolInit;
    int ret;
    if (!virDomainAddIOThreadSymbolInit) {
        libvirtLoad();
        virDomainAddIOThreadSymbol = libvirtSymbol(libvirt, "virDomainAddIOThread");
        virDomainAddIOThreadSymbolInit = (virDomainAddIOThreadSymbol != NULL);
    }
    if (!virDomainAddIOThreadSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virDomainAgentSetResponseTimeoutType virDomainAgentSetResponseTimeoutSymbol;
    static bool virDomainAgentSetResponseTimeoutSymbolInit;
    int ret;
    if (!virDomainAgentSetResponseTimeoutSymbolInit) {
        libvirtLoad();
        virDomainAgentSetResponseTimeoutSymbol = libvirtSymbol(libvirt, "virDomainAgentSetResponseTimeout");
        virDomainAgentSetResponseTimeoutSymbolInit = (virDomainAgentSetResponseTimeoutSymbol != NULL);
    }
    if (!virDomainAgentSetResponseTimeoutSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainAttachDeviceType virDomainAttachDeviceSymbol;
    static bool virDomainAttachDeviceSymbolInit;
    int ret;
    if (!virDomainAttachDeviceSymbolInit) {
        libvirtLoad();
        virDomainAttachDeviceSymbol = libvirtSymbol(libvirt, "virDomainAttachDevice");
        virDomainAttachDeviceSymbolInit = (virDomainAttachDeviceSymbol != NULL);
    }
    if (!virDomainAttachDeviceSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainAttachDeviceFlagsType virDomainAttachDeviceFlagsSymbol;
    static bool virDomainAttachDeviceFlagsSymbolInit;
    int ret;
    if (!virDomainAttachDeviceFlagsSymbolInit) {
        libvirtLoad();
        virDomainAttachDeviceFlagsSymbol = libvirtSymbol(libvirt, "virDomainAttachDeviceFlags");
        virDomainAttachDeviceFlagsSymbolInit = (virDomainAttachDeviceFlagsSymbol != NULL);
    }
    if (!virDomainAttachDeviceFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainAuthorizedSSHKeysGetType virDomainAuthorizedSSHKeysGetSymbol;
    static bool virDomainAuthorizedSSHKeysGetSymbolInit;
    int ret;
    if (!virDomainAuthorizedSSHKeysGetSymbolInit) {
        libvirtLoad();
        virDomainAuthorizedSSHKeysGetSymbol = libvirtSymbol(libvirt, "virDomainAuthorizedSSHKeysGet");
        virDomainAuthorizedSSHKeysGetSymbolInit = (virDomainAuthorizedSSHKeysGetSymbol != NULL);
    }
    if (!virDomainAuthorizedSSHKeysGetSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainAuthorizedSSHKeysSetType virDomainAuthorizedSSHKeysSetSymbol;
    static bool virDomainAuthorizedSSHKeysSetSymbolInit;
    int ret;
    if (!virDomainAuthorizedSSHKeysSetSymbolInit) {
        libvirtLoad();
        virDomainAuthorizedSSHKeysSetSymbol = libvirtSymbol(libvirt, "virDomainAuthorizedSSHKeysSet");
        virDomainAuthorizedSSHKeysSetSymbolInit = (virDomainAuthorizedSSHKeysSetSymbol != NULL);
    }
    if (!virDomainAuthorizedSSHKeysSetSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainBackupBeginType virDomainBackupBeginSymbol;
    static bool virDomainBackupBeginSymbolInit;
    int ret;
    if (!virDomainBackupBeginSymbolInit) {
        libvirtLoad();
        virDomainBackupBeginSymbol = libvirtSymbol(libvirt, "virDomainBackupBegin");
        virDomainBackupBeginSymbolInit = (virDomainBackupBeginSymbol != NULL);
    }
    if (!virDomainBackupBeginSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainBackupGetXMLDescType virDomainBackupGetXMLDescSymbol;
    static bool virDomainBackupGetXMLDescSymbolInit;
    char * ret;
    if (!virDomainBackupGetXMLDescSymbolInit) {
        libvirtLoad();
        virDomainBackupGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainBackupGetXMLDesc");
        virDomainBackupGetXMLDescSymbolInit = (virDomainBackupGetXMLDescSymbol != NULL);
    }
    if (!virDomainBackupGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainBlockCommitType virDomainBlockCommitSymbol;
    static bool virDomainBlockCommitSymbolInit;
    int ret;
    if (!virDomainBlockCommitSymbolInit) {
        libvirtLoad();
        virDomainBlockCommitSymbol = libvirtSymbol(libvirt, "virDomainBlockCommit");
        virDomainBlockCommitSymbolInit = (virDomainBlockCommitSymbol != NULL);
    }
    if (!virDomainBlockCommitSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainBlockCopyType virDomainBlockCopySymbol;
    static bool virDomainBlockCopySymbolInit;
    int ret;
    if (!virDomainBlockCopySymbolInit) {
        libvirtLoad();
        virDomainBlockCopySymbol = libvirtSymbol(libvirt, "virDomainBlockCopy");
        virDomainBlockCopySymbolInit = (virDomainBlockCopySymbol != NULL);
    }
    if (!virDomainBlockCopySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainBlockJobAbortType virDomainBlockJobAbortSymbol;
    static bool virDomainBlockJobAbortSymbolInit;
    int ret;
    if (!virDomainBlockJobAbortSymbolInit) {
        libvirtLoad();
        virDomainBlockJobAbortSymbol = libvirtSymbol(libvirt, "virDomainBlockJobAbort");
        virDomainBlockJobAbortSymbolInit = (virDomainBlockJobAbortSymbol != NULL);
    }
    if (!virDomainBlockJobAbortSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainBlockJobSetSpeedType virDomainBlockJobSetSpeedSymbol;
    static bool virDomainBlockJobSetSpeedSymbolInit;
    int ret;
    if (!virDomainBlockJobSetSpeedSymbolInit) {
        libvirtLoad();
        virDomainBlockJobSetSpeedSymbol = libvirtSymbol(libvirt, "virDomainBlockJobSetSpeed");
        virDomainBlockJobSetSpeedSymbolInit = (virDomainBlockJobSetSpeedSymbol != NULL);
    }
    if (!virDomainBlockJobSetSpeedSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainBlockPeekType virDomainBlockPeekSymbol;
    static bool virDomainBlockPeekSymbolInit;
    int ret;
    if (!virDomainBlockPeekSymbolInit) {
        libvirtLoad();
        virDomainBlockPeekSymbol = libvirtSymbol(libvirt, "virDomainBlockPeek");
        virDomainBlockPeekSymbolInit = (virDomainBlockPeekSymbol != NULL);
    }
    if (!virDomainBlockPeekSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainBlockPullType virDomainBlockPullSymbol;
    static bool virDomainBlockPullSymbolInit;
    int ret;
    if (!virDomainBlockPullSymbolInit) {
        libvirtLoad();
        virDomainBlockPullSymbol = libvirtSymbol(libvirt, "virDomainBlockPull");
        virDomainBlockPullSymbolInit = (virDomainBlockPullSymbol != NULL);
    }
    if (!virDomainBlockPullSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainBlockRebaseType virDomainBlockRebaseSymbol;
    static bool virDomainBlockRebaseSymbolInit;
    int ret;
    if (!virDomainBlockRebaseSymbolInit) {
        libvirtLoad();
        virDomainBlockRebaseSymbol = libvirtSymbol(libvirt, "virDomainBlockRebase");
        virDomainBlockRebaseSymbolInit = (virDomainBlockRebaseSymbol != NULL);
    }
    if (!virDomainBlockRebaseSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainBlockResizeType virDomainBlockResizeSymbol;
    static bool virDomainBlockResizeSymbolInit;
    int ret;
    if (!virDomainBlockResizeSymbolInit) {
        libvirtLoad();
        virDomainBlockResizeSymbol = libvirtSymbol(libvirt, "virDomainBlockResize");
        virDomainBlockResizeSymbolInit = (virDomainBlockResizeSymbol != NULL);
    }
    if (!virDomainBlockResizeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainBlockStatsType virDomainBlockStatsSymbol;
    static bool virDomainBlockStatsSymbolInit;
    int ret;
    if (!virDomainBlockStatsSymbolInit) {
        libvirtLoad();
        virDomainBlockStatsSymbol = libvirtSymbol(libvirt, "virDomainBlockStats");
        virDomainBlockStatsSymbolInit = (virDomainBlockStatsSymbol != NULL);
    }
    if (!virDomainBlockStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainBlockStatsFlagsType virDomainBlockStatsFlagsSymbol;
    static bool virDomainBlockStatsFlagsSymbolInit;
    int ret;
    if (!virDomainBlockStatsFlagsSymbolInit) {
        libvirtLoad();
        virDomainBlockStatsFlagsSymbol = libvirtSymbol(libvirt, "virDomainBlockStatsFlags");
        virDomainBlockStatsFlagsSymbolInit = (virDomainBlockStatsFlagsSymbol != NULL);
    }
    if (!virDomainBlockStatsFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainCheckpointCreateXMLType virDomainCheckpointCreateXMLSymbol;
    static bool virDomainCheckpointCreateXMLSymbolInit;
    virDomainCheckpointPtr ret;
    if (!virDomainCheckpointCreateXMLSymbolInit) {
        libvirtLoad();
        virDomainCheckpointCreateXMLSymbol = libvirtSymbol(libvirt, "virDomainCheckpointCreateXML");
        virDomainCheckpointCreateXMLSymbolInit = (virDomainCheckpointCreateXMLSymbol != NULL);
    }
    if (!virDomainCheckpointCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainCheckpointDeleteType virDomainCheckpointDeleteSymbol;
    static bool virDomainCheckpointDeleteSymbolInit;
    int ret;
    if (!virDomainCheckpointDeleteSymbolInit) {
        libvirtLoad();
        virDomainCheckpointDeleteSymbol = libvirtSymbol(libvirt, "virDomainCheckpointDelete");
        virDomainCheckpointDeleteSymbolInit = (virDomainCheckpointDeleteSymbol != NULL);
    }
    if (!virDomainCheckpointDeleteSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainCheckpointFreeType virDomainCheckpointFreeSymbol;
    static bool virDomainCheckpointFreeSymbolInit;
    int ret;
    if (!virDomainCheckpointFreeSymbolInit) {
        libvirtLoad();
        virDomainCheckpointFreeSymbol = libvirtSymbol(libvirt, "virDomainCheckpointFree");
        virDomainCheckpointFreeSymbolInit = (virDomainCheckpointFreeSymbol != NULL);
    }
    if (!virDomainCheckpointFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainCheckpointGetConnectType virDomainCheckpointGetConnectSymbol;
    static bool virDomainCheckpointGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virDomainCheckpointGetConnectSymbolInit) {
        libvirtLoad();
        virDomainCheckpointGetConnectSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetConnect");
        virDomainCheckpointGetConnectSymbolInit = (virDomainCheckpointGetConnectSymbol != NULL);
    }
    if (!virDomainCheckpointGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainCheckpointGetDomainType virDomainCheckpointGetDomainSymbol;
    static bool virDomainCheckpointGetDomainSymbolInit;
    virDomainPtr ret;
    if (!virDomainCheckpointGetDomainSymbolInit) {
        libvirtLoad();
        virDomainCheckpointGetDomainSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetDomain");
        virDomainCheckpointGetDomainSymbolInit = (virDomainCheckpointGetDomainSymbol != NULL);
    }
    if (!virDomainCheckpointGetDomainSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainCheckpointGetNameType virDomainCheckpointGetNameSymbol;
    static bool virDomainCheckpointGetNameSymbolInit;
    const char * ret;
    if (!virDomainCheckpointGetNameSymbolInit) {
        libvirtLoad();
        virDomainCheckpointGetNameSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetName");
        virDomainCheckpointGetNameSymbolInit = (virDomainCheckpointGetNameSymbol != NULL);
    }
    if (!virDomainCheckpointGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainCheckpointGetParentType virDomainCheckpointGetParentSymbol;
    static bool virDomainCheckpointGetParentSymbolInit;
    virDomainCheckpointPtr ret;
    if (!virDomainCheckpointGetParentSymbolInit) {
        libvirtLoad();
        virDomainCheckpointGetParentSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetParent");
        virDomainCheckpointGetParentSymbolInit = (virDomainCheckpointGetParentSymbol != NULL);
    }
    if (!virDomainCheckpointGetParentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainCheckpointGetXMLDescType virDomainCheckpointGetXMLDescSymbol;
    static bool virDomainCheckpointGetXMLDescSymbolInit;
    char * ret;
    if (!virDomainCheckpointGetXMLDescSymbolInit) {
        libvirtLoad();
        virDomainCheckpointGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainCheckpointGetXMLDesc");
        virDomainCheckpointGetXMLDescSymbolInit = (virDomainCheckpointGetXMLDescSymbol != NULL);
    }
    if (!virDomainCheckpointGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                          virErrorPtr err) {
    static virDomainCheckpointListAllChildrenType virDomainCheckpointListAllChildrenSymbol;
    static bool virDomainCheckpointListAllChildrenSymbolInit;
    int ret;
    if (!virDomainCheckpointListAllChildrenSymbolInit) {
        libvirtLoad();
        virDomainCheckpointListAllChildrenSymbol = libvirtSymbol(libvirt, "virDomainCheckpointListAllChildren");
        virDomainCheckpointListAllChildrenSymbolInit = (virDomainCheckpointListAllChildrenSymbol != NULL);
    }
    if (!virDomainCheckpointListAllChildrenSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virDomainCheckpointLookupByNameType virDomainCheckpointLookupByNameSymbol;
    static bool virDomainCheckpointLookupByNameSymbolInit;
    virDomainCheckpointPtr ret;
    if (!virDomainCheckpointLookupByNameSymbolInit) {
        libvirtLoad();
        virDomainCheckpointLookupByNameSymbol = libvirtSymbol(libvirt, "virDomainCheckpointLookupByName");
        virDomainCheckpointLookupByNameSymbolInit = (virDomainCheckpointLookupByNameSymbol != NULL);
    }
    if (!virDomainCheckpointLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainCheckpointRefType virDomainCheckpointRefSymbol;
    static bool virDomainCheckpointRefSymbolInit;
    int ret;
    if (!virDomainCheckpointRefSymbolInit) {
        libvirtLoad();
        virDomainCheckpointRefSymbol = libvirtSymbol(libvirt, "virDomainCheckpointRef");
        virDomainCheckpointRefSymbolInit = (virDomainCheckpointRefSymbol != NULL);
    }
    if (!virDomainCheckpointRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainCoreDumpType virDomainCoreDumpSymbol;
    static bool virDomainCoreDumpSymbolInit;
    int ret;
    if (!virDomainCoreDumpSymbolInit) {
        libvirtLoad();
        virDomainCoreDumpSymbol = libvirtSymbol(libvirt, "virDomainCoreDump");
        virDomainCoreDumpSymbolInit = (virDomainCoreDumpSymbol != NULL);
    }
    if (!virDomainCoreDumpSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainCoreDumpWithFormatType virDomainCoreDumpWithFormatSymbol;
    static bool virDomainCoreDumpWithFormatSymbolInit;
    int ret;
    if (!virDomainCoreDumpWithFormatSymbolInit) {
        libvirtLoad();
        virDomainCoreDumpWithFormatSymbol = libvirtSymbol(libvirt, "virDomainCoreDumpWithFormat");
        virDomainCoreDumpWithFormatSymbolInit = (virDomainCoreDumpWithFormatSymbol != NULL);
    }
    if (!virDomainCoreDumpWithFormatSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virDomainCreateType virDomainCreateSymbol;
    static bool virDomainCreateSymbolInit;
    int ret;
    if (!virDomainCreateSymbolInit) {
        libvirtLoad();
        virDomainCreateSymbol = libvirtSymbol(libvirt, "virDomainCreate");
        virDomainCreateSymbolInit = (virDomainCreateSymbol != NULL);
    }
    if (!virDomainCreateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainCreateLinuxType virDomainCreateLinuxSymbol;
    static bool virDomainCreateLinuxSymbolInit;
    virDomainPtr ret;
    if (!virDomainCreateLinuxSymbolInit) {
        libvirtLoad();
        virDomainCreateLinuxSymbol = libvirtSymbol(libvirt, "virDomainCreateLinux");
        virDomainCreateLinuxSymbolInit = (virDomainCreateLinuxSymbol != NULL);
    }
    if (!virDomainCreateLinuxSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainCreateWithFilesType virDomainCreateWithFilesSymbol;
    static bool virDomainCreateWithFilesSymbolInit;
    int ret;
    if (!virDomainCreateWithFilesSymbolInit) {
        libvirtLoad();
        virDomainCreateWithFilesSymbol = libvirtSymbol(libvirt, "virDomainCreateWithFiles");
        virDomainCreateWithFilesSymbolInit = (virDomainCreateWithFilesSymbol != NULL);
    }
    if (!virDomainCreateWithFilesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainCreateWithFlagsType virDomainCreateWithFlagsSymbol;
    static bool virDomainCreateWithFlagsSymbolInit;
    int ret;
    if (!virDomainCreateWithFlagsSymbolInit) {
        libvirtLoad();
        virDomainCreateWithFlagsSymbol = libvirtSymbol(libvirt, "virDomainCreateWithFlags");
        virDomainCreateWithFlagsSymbolInit = (virDomainCreateWithFlagsSymbol != NULL);
    }
    if (!virDomainCreateWithFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainCreateXMLType virDomainCreateXMLSymbol;
    static bool virDomainCreateXMLSymbolInit;
    virDomainPtr ret;
    if (!virDomainCreateXMLSymbolInit) {
        libvirtLoad();
        virDomainCreateXMLSymbol = libvirtSymbol(libvirt, "virDomainCreateXML");
        virDomainCreateXMLSymbolInit = (virDomainCreateXMLSymbol != NULL);
    }
    if (!virDomainCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainCreateXMLWithFilesType virDomainCreateXMLWithFilesSymbol;
    static bool virDomainCreateXMLWithFilesSymbolInit;
    virDomainPtr ret;
    if (!virDomainCreateXMLWithFilesSymbolInit) {
        libvirtLoad();
        virDomainCreateXMLWithFilesSymbol = libvirtSymbol(libvirt, "virDomainCreateXMLWithFiles");
        virDomainCreateXMLWithFilesSymbolInit = (virDomainCreateXMLWithFilesSymbol != NULL);
    }
    if (!virDomainCreateXMLWithFilesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainDefineXMLType virDomainDefineXMLSymbol;
    static bool virDomainDefineXMLSymbolInit;
    virDomainPtr ret;
    if (!virDomainDefineXMLSymbolInit) {
        libvirtLoad();
        virDomainDefineXMLSymbol = libvirtSymbol(libvirt, "virDomainDefineXML");
        virDomainDefineXMLSymbolInit = (virDomainDefineXMLSymbol != NULL);
    }
    if (!virDomainDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainDefineXMLFlagsType virDomainDefineXMLFlagsSymbol;
    static bool virDomainDefineXMLFlagsSymbolInit;
    virDomainPtr ret;
    if (!virDomainDefineXMLFlagsSymbolInit) {
        libvirtLoad();
        virDomainDefineXMLFlagsSymbol = libvirtSymbol(libvirt, "virDomainDefineXMLFlags");
        virDomainDefineXMLFlagsSymbolInit = (virDomainDefineXMLFlagsSymbol != NULL);
    }
    if (!virDomainDefineXMLFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainDelIOThreadType virDomainDelIOThreadSymbol;
    static bool virDomainDelIOThreadSymbolInit;
    int ret;
    if (!virDomainDelIOThreadSymbolInit) {
        libvirtLoad();
        virDomainDelIOThreadSymbol = libvirtSymbol(libvirt, "virDomainDelIOThread");
        virDomainDelIOThreadSymbolInit = (virDomainDelIOThreadSymbol != NULL);
    }
    if (!virDomainDelIOThreadSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainDestroyType virDomainDestroySymbol;
    static bool virDomainDestroySymbolInit;
    int ret;
    if (!virDomainDestroySymbolInit) {
        libvirtLoad();
        virDomainDestroySymbol = libvirtSymbol(libvirt, "virDomainDestroy");
        virDomainDestroySymbolInit = (virDomainDestroySymbol != NULL);
    }
    if (!virDomainDestroySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainDestroyFlagsType virDomainDestroyFlagsSymbol;
    static bool virDomainDestroyFlagsSymbolInit;
    int ret;
    if (!virDomainDestroyFlagsSymbolInit) {
        libvirtLoad();
        virDomainDestroyFlagsSymbol = libvirtSymbol(libvirt, "virDomainDestroyFlags");
        virDomainDestroyFlagsSymbolInit = (virDomainDestroyFlagsSymbol != NULL);
    }
    if (!virDomainDestroyFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainDetachDeviceType virDomainDetachDeviceSymbol;
    static bool virDomainDetachDeviceSymbolInit;
    int ret;
    if (!virDomainDetachDeviceSymbolInit) {
        libvirtLoad();
        virDomainDetachDeviceSymbol = libvirtSymbol(libvirt, "virDomainDetachDevice");
        virDomainDetachDeviceSymbolInit = (virDomainDetachDeviceSymbol != NULL);
    }
    if (!virDomainDetachDeviceSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainDetachDeviceAliasType virDomainDetachDeviceAliasSymbol;
    static bool virDomainDetachDeviceAliasSymbolInit;
    int ret;
    if (!virDomainDetachDeviceAliasSymbolInit) {
        libvirtLoad();
        virDomainDetachDeviceAliasSymbol = libvirtSymbol(libvirt, "virDomainDetachDeviceAlias");
        virDomainDetachDeviceAliasSymbolInit = (virDomainDetachDeviceAliasSymbol != NULL);
    }
    if (!virDomainDetachDeviceAliasSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainDetachDeviceFlagsType virDomainDetachDeviceFlagsSymbol;
    static bool virDomainDetachDeviceFlagsSymbolInit;
    int ret;
    if (!virDomainDetachDeviceFlagsSymbolInit) {
        libvirtLoad();
        virDomainDetachDeviceFlagsSymbol = libvirtSymbol(libvirt, "virDomainDetachDeviceFlags");
        virDomainDetachDeviceFlagsSymbolInit = (virDomainDetachDeviceFlagsSymbol != NULL);
    }
    if (!virDomainDetachDeviceFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainFSFreezeType virDomainFSFreezeSymbol;
    static bool virDomainFSFreezeSymbolInit;
    int ret;
    if (!virDomainFSFreezeSymbolInit) {
        libvirtLoad();
        virDomainFSFreezeSymbol = libvirtSymbol(libvirt, "virDomainFSFreeze");
        virDomainFSFreezeSymbolInit = (virDomainFSFreezeSymbol != NULL);
    }
    if (!virDomainFSFreezeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info) {
    static virDomainFSInfoFreeType virDomainFSInfoFreeSymbol;
    static bool virDomainFSInfoFreeSymbolInit;

    if (!virDomainFSInfoFreeSymbolInit) {
        libvirtLoad();
        virDomainFSInfoFreeSymbol = libvirtSymbol(libvirt, "virDomainFSInfoFree");
        virDomainFSInfoFreeSymbolInit = (virDomainFSInfoFreeSymbol != NULL);
    }
    if (!virDomainFSInfoFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virDomainFSThawType virDomainFSThawSymbol;
    static bool virDomainFSThawSymbolInit;
    int ret;
    if (!virDomainFSThawSymbolInit) {
        libvirtLoad();
        virDomainFSThawSymbol = libvirtSymbol(libvirt, "virDomainFSThaw");
        virDomainFSThawSymbolInit = (virDomainFSThawSymbol != NULL);
    }
    if (!virDomainFSThawSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virDomainFSTrimType virDomainFSTrimSymbol;
    static bool virDomainFSTrimSymbolInit;
    int ret;
    if (!virDomainFSTrimSymbolInit) {
        libvirtLoad();
        virDomainFSTrimSymbol = libvirtSymbol(libvirt, "virDomainFSTrim");
        virDomainFSTrimSymbolInit = (virDomainFSTrimSymbol != NULL);
    }
    if (!virDomainFSTrimSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virDomainFreeType virDomainFreeSymbol;
    static bool virDomainFreeSymbolInit;
    int ret;
    if (!virDomainFreeSymbolInit) {
        libvirtLoad();
        virDomainFreeSymbol = libvirtSymbol(libvirt, "virDomainFree");
        virDomainFreeSymbolInit = (virDomainFreeSymbol != NULL);
    }
    if (!virDomainFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainGetAutostartType virDomainGetAutostartSymbol;
    static bool virDomainGetAutostartSymbolInit;
    int ret;
    if (!virDomainGetAutostartSymbolInit) {
        libvirtLoad();
        virDomainGetAutostartSymbol = libvirtSymbol(libvirt, "virDomainGetAutostart");
        virDomainGetAutostartSymbolInit = (virDomainGetAutostartSymbol != NULL);
    }
    if (!virDomainGetAutostartSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainGetBlkioParametersType virDomainGetBlkioParametersSymbol;
    static bool virDomainGetBlkioParametersSymbolInit;
    int ret;
    if (!virDomainGetBlkioParametersSymbolInit) {
        libvirtLoad();
        virDomainGetBlkioParametersSymbol = libvirtSymbol(libvirt, "virDomainGetBlkioParameters");
        virDomainGetBlkioParametersSymbolInit = (virDomainGetBlkioParametersSymbol != NULL);
    }
    if (!virDomainGetBlkioParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainGetBlockInfoType virDomainGetBlockInfoSymbol;
    static bool virDomainGetBlockInfoSymbolInit;
    int ret;
    if (!virDomainGetBlockInfoSymbolInit) {
        libvirtLoad();
        virDomainGetBlockInfoSymbol = libvirtSymbol(libvirt, "virDomainGetBlockInfo");
        virDomainGetBlockInfoSymbolInit = (virDomainGetBlockInfoSymbol != NULL);
    }
    if (!virDomainGetBlockInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainGetBlockIoTuneType virDomainGetBlockIoTuneSymbol;
    static bool virDomainGetBlockIoTuneSymbolInit;
    int ret;
    if (!virDomainGetBlockIoTuneSymbolInit) {
        libvirtLoad();
        virDomainGetBlockIoTuneSymbol = libvirtSymbol(libvirt, "virDomainGetBlockIoTune");
        virDomainGetBlockIoTuneSymbolInit = (virDomainGetBlockIoTuneSymbol != NULL);
    }
    if (!virDomainGetBlockIoTuneSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainGetBlockJobInfoType virDomainGetBlockJobInfoSymbol;
    static bool virDomainGetBlockJobInfoSymbolInit;
    int ret;
    if (!virDomainGetBlockJobInfoSymbolInit) {
        libvirtLoad();
        virDomainGetBlockJobInfoSymbol = libvirtSymbol(libvirt, "virDomainGetBlockJobInfo");
        virDomainGetBlockJobInfoSymbolInit = (virDomainGetBlockJobInfoSymbol != NULL);
    }
    if (!virDomainGetBlockJobInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainGetCPUStatsType virDomainGetCPUStatsSymbol;
    static bool virDomainGetCPUStatsSymbolInit;
    int ret;
    if (!virDomainGetCPUStatsSymbolInit) {
        libvirtLoad();
        virDomainGetCPUStatsSymbol = libvirtSymbol(libvirt, "virDomainGetCPUStats");
        virDomainGetCPUStatsSymbolInit = (virDomainGetCPUStatsSymbol != NULL);
    }
    if (!virDomainGetCPUStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainGetConnectType virDomainGetConnectSymbol;
    static bool virDomainGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virDomainGetConnectSymbolInit) {
        libvirtLoad();
        virDomainGetConnectSymbol = libvirtSymbol(libvirt, "virDomainGetConnect");
        virDomainGetConnectSymbolInit = (virDomainGetConnectSymbol != NULL);
    }
    if (!virDomainGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainGetControlInfoType virDomainGetControlInfoSymbol;
    static bool virDomainGetControlInfoSymbolInit;
    int ret;
    if (!virDomainGetControlInfoSymbolInit) {
        libvirtLoad();
        virDomainGetControlInfoSymbol = libvirtSymbol(libvirt, "virDomainGetControlInfo");
        virDomainGetControlInfoSymbolInit = (virDomainGetControlInfoSymbol != NULL);
    }
    if (!virDomainGetControlInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainGetDiskErrorsType virDomainGetDiskErrorsSymbol;
    static bool virDomainGetDiskErrorsSymbolInit;
    int ret;
    if (!virDomainGetDiskErrorsSymbolInit) {
        libvirtLoad();
        virDomainGetDiskErrorsSymbol = libvirtSymbol(libvirt, "virDomainGetDiskErrors");
        virDomainGetDiskErrorsSymbolInit = (virDomainGetDiskErrorsSymbol != NULL);
    }
    if (!virDomainGetDiskErrorsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainGetEmulatorPinInfoType virDomainGetEmulatorPinInfoSymbol;
    static bool virDomainGetEmulatorPinInfoSymbolInit;
    int ret;
    if (!virDomainGetEmulatorPinInfoSymbolInit) {
        libvirtLoad();
        virDomainGetEmulatorPinInfoSymbol = libvirtSymbol(libvirt, "virDomainGetEmulatorPinInfo");
        virDomainGetEmulatorPinInfoSymbolInit = (virDomainGetEmulatorPinInfoSymbol != NULL);
    }
    if (!virDomainGetEmulatorPinInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainGetFSInfoType virDomainGetFSInfoSymbol;
    static bool virDomainGetFSInfoSymbolInit;
    int ret;
    if (!virDomainGetFSInfoSymbolInit) {
        libvirtLoad();
        virDomainGetFSInfoSymbol = libvirtSymbol(libvirt, "virDomainGetFSInfo");
        virDomainGetFSInfoSymbolInit = (virDomainGetFSInfoSymbol != NULL);
    }
    if (!virDomainGetFSInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainGetGuestInfoType virDomainGetGuestInfoSymbol;
    static bool virDomainGetGuestInfoSymbolInit;
    int ret;
    if (!virDomainGetGuestInfoSymbolInit) {
        libvirtLoad();
        virDomainGetGuestInfoSymbol = libvirtSymbol(libvirt, "virDomainGetGuestInfo");
        virDomainGetGuestInfoSymbolInit = (virDomainGetGuestInfoSymbol != NULL);
    }
    if (!virDomainGetGuestInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainGetGuestVcpusType virDomainGetGuestVcpusSymbol;
    static bool virDomainGetGuestVcpusSymbolInit;
    int ret;
    if (!virDomainGetGuestVcpusSymbolInit) {
        libvirtLoad();
        virDomainGetGuestVcpusSymbol = libvirtSymbol(libvirt, "virDomainGetGuestVcpus");
        virDomainGetGuestVcpusSymbolInit = (virDomainGetGuestVcpusSymbol != NULL);
    }
    if (!virDomainGetGuestVcpusSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainGetHostnameType virDomainGetHostnameSymbol;
    static bool virDomainGetHostnameSymbolInit;
    char * ret;
    if (!virDomainGetHostnameSymbolInit) {
        libvirtLoad();
        virDomainGetHostnameSymbol = libvirtSymbol(libvirt, "virDomainGetHostname");
        virDomainGetHostnameSymbolInit = (virDomainGetHostnameSymbol != NULL);
    }
    if (!virDomainGetHostnameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virDomainGetIDType virDomainGetIDSymbol;
    static bool virDomainGetIDSymbolInit;
    unsigned int ret;
    if (!virDomainGetIDSymbolInit) {
        libvirtLoad();
        virDomainGetIDSymbol = libvirtSymbol(libvirt, "virDomainGetID");
        virDomainGetIDSymbolInit = (virDomainGetIDSymbol != NULL);
    }
    if (!virDomainGetIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainGetIOThreadInfoType virDomainGetIOThreadInfoSymbol;
    static bool virDomainGetIOThreadInfoSymbolInit;
    int ret;
    if (!virDomainGetIOThreadInfoSymbolInit) {
        libvirtLoad();
        virDomainGetIOThreadInfoSymbol = libvirtSymbol(libvirt, "virDomainGetIOThreadInfo");
        virDomainGetIOThreadInfoSymbolInit = (virDomainGetIOThreadInfoSymbol != NULL);
    }
    if (!virDomainGetIOThreadInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainGetInfoType virDomainGetInfoSymbol;
    static bool virDomainGetInfoSymbolInit;
    int ret;
    if (!virDomainGetInfoSymbolInit) {
        libvirtLoad();
        virDomainGetInfoSymbol = libvirtSymbol(libvirt, "virDomainGetInfo");
        virDomainGetInfoSymbolInit = (virDomainGetInfoSymbol != NULL);
    }
    if (!virDomainGetInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virDomainGetInterfaceParametersType virDomainGetInterfaceParametersSymbol;
    static bool virDomainGetInterfaceParametersSymbolInit;
    int ret;
    if (!virDomainGetInterfaceParametersSymbolInit) {
        libvirtLoad();
        virDomainGetInterfaceParametersSymbol = libvirtSymbol(libvirt, "virDomainGetInterfaceParameters");
        virDomainGetInterfaceParametersSymbolInit = (virDomainGetInterfaceParametersSymbol != NULL);
    }
    if (!virDomainGetInterfaceParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainGetJobInfoType virDomainGetJobInfoSymbol;
    static bool virDomainGetJobInfoSymbolInit;
    int ret;
    if (!virDomainGetJobInfoSymbolInit) {
        libvirtLoad();
        virDomainGetJobInfoSymbol = libvirtSymbol(libvirt, "virDomainGetJobInfo");
        virDomainGetJobInfoSymbolInit = (virDomainGetJobInfoSymbol != NULL);
    }
    if (!virDomainGetJobInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainGetJobStatsType virDomainGetJobStatsSymbol;
    static bool virDomainGetJobStatsSymbolInit;
    int ret;
    if (!virDomainGetJobStatsSymbolInit) {
        libvirtLoad();
        virDomainGetJobStatsSymbol = libvirtSymbol(libvirt, "virDomainGetJobStats");
        virDomainGetJobStatsSymbolInit = (virDomainGetJobStatsSymbol != NULL);
    }
    if (!virDomainGetJobStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virDomainGetLaunchSecurityInfoType virDomainGetLaunchSecurityInfoSymbol;
    static bool virDomainGetLaunchSecurityInfoSymbolInit;
    int ret;
    if (!virDomainGetLaunchSecurityInfoSymbolInit) {
        libvirtLoad();
        virDomainGetLaunchSecurityInfoSymbol = libvirtSymbol(libvirt, "virDomainGetLaunchSecurityInfo");
        virDomainGetLaunchSecurityInfoSymbolInit = (virDomainGetLaunchSecurityInfoSymbol != NULL);
    }
    if (!virDomainGetLaunchSecurityInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainGetMaxMemoryType virDomainGetMaxMemorySymbol;
    static bool virDomainGetMaxMemorySymbolInit;
    unsigned long ret;
    if (!virDomainGetMaxMemorySymbolInit) {
        libvirtLoad();
        virDomainGetMaxMemorySymbol = libvirtSymbol(libvirt, "virDomainGetMaxMemory");
        virDomainGetMaxMemorySymbolInit = (virDomainGetMaxMemorySymbol != NULL);
    }
    if (!virDomainGetMaxMemorySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainGetMaxVcpusType virDomainGetMaxVcpusSymbol;
    static bool virDomainGetMaxVcpusSymbolInit;
    int ret;
    if (!virDomainGetMaxVcpusSymbolInit) {
        libvirtLoad();
        virDomainGetMaxVcpusSymbol = libvirtSymbol(libvirt, "virDomainGetMaxVcpus");
        virDomainGetMaxVcpusSymbolInit = (virDomainGetMaxVcpusSymbol != NULL);
    }
    if (!virDomainGetMaxVcpusSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainGetMemoryParametersType virDomainGetMemoryParametersSymbol;
    static bool virDomainGetMemoryParametersSymbolInit;
    int ret;
    if (!virDomainGetMemoryParametersSymbolInit) {
        libvirtLoad();
        virDomainGetMemoryParametersSymbol = libvirtSymbol(libvirt, "virDomainGetMemoryParameters");
        virDomainGetMemoryParametersSymbolInit = (virDomainGetMemoryParametersSymbol != NULL);
    }
    if (!virDomainGetMemoryParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainGetMessagesType virDomainGetMessagesSymbol;
    static bool virDomainGetMessagesSymbolInit;
    int ret;
    if (!virDomainGetMessagesSymbolInit) {
        libvirtLoad();
        virDomainGetMessagesSymbol = libvirtSymbol(libvirt, "virDomainGetMessages");
        virDomainGetMessagesSymbolInit = (virDomainGetMessagesSymbol != NULL);
    }
    if (!virDomainGetMessagesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainGetMetadataType virDomainGetMetadataSymbol;
    static bool virDomainGetMetadataSymbolInit;
    char * ret;
    if (!virDomainGetMetadataSymbolInit) {
        libvirtLoad();
        virDomainGetMetadataSymbol = libvirtSymbol(libvirt, "virDomainGetMetadata");
        virDomainGetMetadataSymbolInit = (virDomainGetMetadataSymbol != NULL);
    }
    if (!virDomainGetMetadataSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainGetNameType virDomainGetNameSymbol;
    static bool virDomainGetNameSymbolInit;
    const char * ret;
    if (!virDomainGetNameSymbolInit) {
        libvirtLoad();
        virDomainGetNameSymbol = libvirtSymbol(libvirt, "virDomainGetName");
        virDomainGetNameSymbolInit = (virDomainGetNameSymbol != NULL);
    }
    if (!virDomainGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainGetNumaParametersType virDomainGetNumaParametersSymbol;
    static bool virDomainGetNumaParametersSymbolInit;
    int ret;
    if (!virDomainGetNumaParametersSymbolInit) {
        libvirtLoad();
        virDomainGetNumaParametersSymbol = libvirtSymbol(libvirt, "virDomainGetNumaParameters");
        virDomainGetNumaParametersSymbolInit = (virDomainGetNumaParametersSymbol != NULL);
    }
    if (!virDomainGetNumaParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainGetOSTypeType virDomainGetOSTypeSymbol;
    static bool virDomainGetOSTypeSymbolInit;
    char * ret;
    if (!virDomainGetOSTypeSymbolInit) {
        libvirtLoad();
        virDomainGetOSTypeSymbol = libvirtSymbol(libvirt, "virDomainGetOSType");
        virDomainGetOSTypeSymbolInit = (virDomainGetOSTypeSymbol != NULL);
    }
    if (!virDomainGetOSTypeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainGetPerfEventsType virDomainGetPerfEventsSymbol;
    static bool virDomainGetPerfEventsSymbolInit;
    int ret;
    if (!virDomainGetPerfEventsSymbolInit) {
        libvirtLoad();
        virDomainGetPerfEventsSymbol = libvirtSymbol(libvirt, "virDomainGetPerfEvents");
        virDomainGetPerfEventsSymbolInit = (virDomainGetPerfEventsSymbol != NULL);
    }
    if (!virDomainGetPerfEventsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virDomainGetSchedulerParametersType virDomainGetSchedulerParametersSymbol;
    static bool virDomainGetSchedulerParametersSymbolInit;
    int ret;
    if (!virDomainGetSchedulerParametersSymbolInit) {
        libvirtLoad();
        virDomainGetSchedulerParametersSymbol = libvirtSymbol(libvirt, "virDomainGetSchedulerParameters");
        virDomainGetSchedulerParametersSymbolInit = (virDomainGetSchedulerParametersSymbol != NULL);
    }
    if (!virDomainGetSchedulerParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                            virErrorPtr err) {
    static virDomainGetSchedulerParametersFlagsType virDomainGetSchedulerParametersFlagsSymbol;
    static bool virDomainGetSchedulerParametersFlagsSymbolInit;
    int ret;
    if (!virDomainGetSchedulerParametersFlagsSymbolInit) {
        libvirtLoad();
        virDomainGetSchedulerParametersFlagsSymbol = libvirtSymbol(libvirt, "virDomainGetSchedulerParametersFlags");
        virDomainGetSchedulerParametersFlagsSymbolInit = (virDomainGetSchedulerParametersFlagsSymbol != NULL);
    }
    if (!virDomainGetSchedulerParametersFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainGetSchedulerTypeType virDomainGetSchedulerTypeSymbol;
    static bool virDomainGetSchedulerTypeSymbolInit;
    char * ret;
    if (!virDomainGetSchedulerTypeSymbolInit) {
        libvirtLoad();
        virDomainGetSchedulerTypeSymbol = libvirtSymbol(libvirt, "virDomainGetSchedulerType");
        virDomainGetSchedulerTypeSymbolInit = (virDomainGetSchedulerTypeSymbol != NULL);
    }
    if (!virDomainGetSchedulerTypeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainGetSecurityLabelType virDomainGetSecurityLabelSymbol;
    static bool virDomainGetSecurityLabelSymbolInit;
    int ret;
    if (!virDomainGetSecurityLabelSymbolInit) {
        libvirtLoad();
        virDomainGetSecurityLabelSymbol = libvirtSymbol(libvirt, "virDomainGetSecurityLabel");
        virDomainGetSecurityLabelSymbolInit = (virDomainGetSecurityLabelSymbol != NULL);
    }
    if (!virDomainGetSecurityLabelSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainGetSecurityLabelListType virDomainGetSecurityLabelListSymbol;
    static bool virDomainGetSecurityLabelListSymbolInit;
    int ret;
    if (!virDomainGetSecurityLabelListSymbolInit) {
        libvirtLoad();
        virDomainGetSecurityLabelListSymbol = libvirtSymbol(libvirt, "virDomainGetSecurityLabelList");
        virDomainGetSecurityLabelListSymbolInit = (virDomainGetSecurityLabelListSymbol != NULL);
    }
    if (!virDomainGetSecurityLabelListSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainGetStateType virDomainGetStateSymbol;
    static bool virDomainGetStateSymbolInit;
    int ret;
    if (!virDomainGetStateSymbolInit) {
        libvirtLoad();
        virDomainGetStateSymbol = libvirtSymbol(libvirt, "virDomainGetState");
        virDomainGetStateSymbolInit = (virDomainGetStateSymbol != NULL);
    }
    if (!virDomainGetStateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainGetTimeType virDomainGetTimeSymbol;
    static bool virDomainGetTimeSymbolInit;
    int ret;
    if (!virDomainGetTimeSymbolInit) {
        libvirtLoad();
        virDomainGetTimeSymbol = libvirtSymbol(libvirt, "virDomainGetTime");
        virDomainGetTimeSymbolInit = (virDomainGetTimeSymbol != NULL);
    }
    if (!virDomainGetTimeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainGetUUIDType virDomainGetUUIDSymbol;
    static bool virDomainGetUUIDSymbolInit;
    int ret;
    if (!virDomainGetUUIDSymbolInit) {
        libvirtLoad();
        virDomainGetUUIDSymbol = libvirtSymbol(libvirt, "virDomainGetUUID");
        virDomainGetUUIDSymbolInit = (virDomainGetUUIDSymbol != NULL);
    }
    if (!virDomainGetUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainGetUUIDStringType virDomainGetUUIDStringSymbol;
    static bool virDomainGetUUIDStringSymbolInit;
    int ret;
    if (!virDomainGetUUIDStringSymbolInit) {
        libvirtLoad();
        virDomainGetUUIDStringSymbol = libvirtSymbol(libvirt, "virDomainGetUUIDString");
        virDomainGetUUIDStringSymbolInit = (virDomainGetUUIDStringSymbol != NULL);
    }
    if (!virDomainGetUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainGetVcpuPinInfoType virDomainGetVcpuPinInfoSymbol;
    static bool virDomainGetVcpuPinInfoSymbolInit;
    int ret;
    if (!virDomainGetVcpuPinInfoSymbolInit) {
        libvirtLoad();
        virDomainGetVcpuPinInfoSymbol = libvirtSymbol(libvirt, "virDomainGetVcpuPinInfo");
        virDomainGetVcpuPinInfoSymbolInit = (virDomainGetVcpuPinInfoSymbol != NULL);
    }
    if (!virDomainGetVcpuPinInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainGetVcpusType virDomainGetVcpusSymbol;
    static bool virDomainGetVcpusSymbolInit;
    int ret;
    if (!virDomainGetVcpusSymbolInit) {
        libvirtLoad();
        virDomainGetVcpusSymbol = libvirtSymbol(libvirt, "virDomainGetVcpus");
        virDomainGetVcpusSymbolInit = (virDomainGetVcpusSymbol != NULL);
    }
    if (!virDomainGetVcpusSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainGetVcpusFlagsType virDomainGetVcpusFlagsSymbol;
    static bool virDomainGetVcpusFlagsSymbolInit;
    int ret;
    if (!virDomainGetVcpusFlagsSymbolInit) {
        libvirtLoad();
        virDomainGetVcpusFlagsSymbol = libvirtSymbol(libvirt, "virDomainGetVcpusFlags");
        virDomainGetVcpusFlagsSymbolInit = (virDomainGetVcpusFlagsSymbol != NULL);
    }
    if (!virDomainGetVcpusFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainGetXMLDescType virDomainGetXMLDescSymbol;
    static bool virDomainGetXMLDescSymbolInit;
    char * ret;
    if (!virDomainGetXMLDescSymbolInit) {
        libvirtLoad();
        virDomainGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainGetXMLDesc");
        virDomainGetXMLDescSymbolInit = (virDomainGetXMLDescSymbol != NULL);
    }
    if (!virDomainGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainHasCurrentSnapshotType virDomainHasCurrentSnapshotSymbol;
    static bool virDomainHasCurrentSnapshotSymbolInit;
    int ret;
    if (!virDomainHasCurrentSnapshotSymbolInit) {
        libvirtLoad();
        virDomainHasCurrentSnapshotSymbol = libvirtSymbol(libvirt, "virDomainHasCurrentSnapshot");
        virDomainHasCurrentSnapshotSymbolInit = (virDomainHasCurrentSnapshotSymbol != NULL);
    }
    if (!virDomainHasCurrentSnapshotSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainHasManagedSaveImageType virDomainHasManagedSaveImageSymbol;
    static bool virDomainHasManagedSaveImageSymbolInit;
    int ret;
    if (!virDomainHasManagedSaveImageSymbolInit) {
        libvirtLoad();
        virDomainHasManagedSaveImageSymbol = libvirtSymbol(libvirt, "virDomainHasManagedSaveImage");
        virDomainHasManagedSaveImageSymbolInit = (virDomainHasManagedSaveImageSymbol != NULL);
    }
    if (!virDomainHasManagedSaveImageSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info) {
    static virDomainIOThreadInfoFreeType virDomainIOThreadInfoFreeSymbol;
    static bool virDomainIOThreadInfoFreeSymbolInit;

    if (!virDomainIOThreadInfoFreeSymbolInit) {
        libvirtLoad();
        virDomainIOThreadInfoFreeSymbol = libvirtSymbol(libvirt, "virDomainIOThreadInfoFree");
        virDomainIOThreadInfoFreeSymbolInit = (virDomainIOThreadInfoFreeSymbol != NULL);
    }
    if (!virDomainIOThreadInfoFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virDomainIOThreadInfoFreeSymbol(info);

}
typedef int
(*virDomainInjectNMIType)(virDomainPtr domain,
                          unsigned int flags);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err) {
    static virDomainInjectNMIType virDomainInjectNMISymbol;
    static bool virDomainInjectNMISymbolInit;
    int ret;
    if (!virDomainInjectNMISymbolInit) {
        libvirtLoad();
        virDomainInjectNMISymbol = libvirtSymbol(libvirt, "virDomainInjectNMI");
        virDomainInjectNMISymbolInit = (virDomainInjectNMISymbol != NULL);
    }
    if (!virDomainInjectNMISymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainInterfaceAddressesType virDomainInterfaceAddressesSymbol;
    static bool virDomainInterfaceAddressesSymbolInit;
    int ret;
    if (!virDomainInterfaceAddressesSymbolInit) {
        libvirtLoad();
        virDomainInterfaceAddressesSymbol = libvirtSymbol(libvirt, "virDomainInterfaceAddresses");
        virDomainInterfaceAddressesSymbolInit = (virDomainInterfaceAddressesSymbol != NULL);
    }
    if (!virDomainInterfaceAddressesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface) {
    static virDomainInterfaceFreeType virDomainInterfaceFreeSymbol;
    static bool virDomainInterfaceFreeSymbolInit;

    if (!virDomainInterfaceFreeSymbolInit) {
        libvirtLoad();
        virDomainInterfaceFreeSymbol = libvirtSymbol(libvirt, "virDomainInterfaceFree");
        virDomainInterfaceFreeSymbolInit = (virDomainInterfaceFreeSymbol != NULL);
    }
    if (!virDomainInterfaceFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainInterfaceStatsType virDomainInterfaceStatsSymbol;
    static bool virDomainInterfaceStatsSymbolInit;
    int ret;
    if (!virDomainInterfaceStatsSymbolInit) {
        libvirtLoad();
        virDomainInterfaceStatsSymbol = libvirtSymbol(libvirt, "virDomainInterfaceStats");
        virDomainInterfaceStatsSymbolInit = (virDomainInterfaceStatsSymbol != NULL);
    }
    if (!virDomainInterfaceStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainIsActiveType virDomainIsActiveSymbol;
    static bool virDomainIsActiveSymbolInit;
    int ret;
    if (!virDomainIsActiveSymbolInit) {
        libvirtLoad();
        virDomainIsActiveSymbol = libvirtSymbol(libvirt, "virDomainIsActive");
        virDomainIsActiveSymbolInit = (virDomainIsActiveSymbol != NULL);
    }
    if (!virDomainIsActiveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainIsPersistentType virDomainIsPersistentSymbol;
    static bool virDomainIsPersistentSymbolInit;
    int ret;
    if (!virDomainIsPersistentSymbolInit) {
        libvirtLoad();
        virDomainIsPersistentSymbol = libvirtSymbol(libvirt, "virDomainIsPersistent");
        virDomainIsPersistentSymbolInit = (virDomainIsPersistentSymbol != NULL);
    }
    if (!virDomainIsPersistentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainIsUpdatedType virDomainIsUpdatedSymbol;
    static bool virDomainIsUpdatedSymbolInit;
    int ret;
    if (!virDomainIsUpdatedSymbolInit) {
        libvirtLoad();
        virDomainIsUpdatedSymbol = libvirtSymbol(libvirt, "virDomainIsUpdated");
        virDomainIsUpdatedSymbolInit = (virDomainIsUpdatedSymbol != NULL);
    }
    if (!virDomainIsUpdatedSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainListAllCheckpointsType virDomainListAllCheckpointsSymbol;
    static bool virDomainListAllCheckpointsSymbolInit;
    int ret;
    if (!virDomainListAllCheckpointsSymbolInit) {
        libvirtLoad();
        virDomainListAllCheckpointsSymbol = libvirtSymbol(libvirt, "virDomainListAllCheckpoints");
        virDomainListAllCheckpointsSymbolInit = (virDomainListAllCheckpointsSymbol != NULL);
    }
    if (!virDomainListAllCheckpointsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainListAllSnapshotsType virDomainListAllSnapshotsSymbol;
    static bool virDomainListAllSnapshotsSymbolInit;
    int ret;
    if (!virDomainListAllSnapshotsSymbolInit) {
        libvirtLoad();
        virDomainListAllSnapshotsSymbol = libvirtSymbol(libvirt, "virDomainListAllSnapshots");
        virDomainListAllSnapshotsSymbolInit = (virDomainListAllSnapshotsSymbol != NULL);
    }
    if (!virDomainListAllSnapshotsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainListGetStatsType virDomainListGetStatsSymbol;
    static bool virDomainListGetStatsSymbolInit;
    int ret;
    if (!virDomainListGetStatsSymbolInit) {
        libvirtLoad();
        virDomainListGetStatsSymbol = libvirtSymbol(libvirt, "virDomainListGetStats");
        virDomainListGetStatsSymbolInit = (virDomainListGetStatsSymbol != NULL);
    }
    if (!virDomainListGetStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainLookupByIDType virDomainLookupByIDSymbol;
    static bool virDomainLookupByIDSymbolInit;
    virDomainPtr ret;
    if (!virDomainLookupByIDSymbolInit) {
        libvirtLoad();
        virDomainLookupByIDSymbol = libvirtSymbol(libvirt, "virDomainLookupByID");
        virDomainLookupByIDSymbolInit = (virDomainLookupByIDSymbol != NULL);
    }
    if (!virDomainLookupByIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainLookupByNameType virDomainLookupByNameSymbol;
    static bool virDomainLookupByNameSymbolInit;
    virDomainPtr ret;
    if (!virDomainLookupByNameSymbolInit) {
        libvirtLoad();
        virDomainLookupByNameSymbol = libvirtSymbol(libvirt, "virDomainLookupByName");
        virDomainLookupByNameSymbolInit = (virDomainLookupByNameSymbol != NULL);
    }
    if (!virDomainLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainLookupByUUIDType virDomainLookupByUUIDSymbol;
    static bool virDomainLookupByUUIDSymbolInit;
    virDomainPtr ret;
    if (!virDomainLookupByUUIDSymbolInit) {
        libvirtLoad();
        virDomainLookupByUUIDSymbol = libvirtSymbol(libvirt, "virDomainLookupByUUID");
        virDomainLookupByUUIDSymbolInit = (virDomainLookupByUUIDSymbol != NULL);
    }
    if (!virDomainLookupByUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainLookupByUUIDStringType virDomainLookupByUUIDStringSymbol;
    static bool virDomainLookupByUUIDStringSymbolInit;
    virDomainPtr ret;
    if (!virDomainLookupByUUIDStringSymbolInit) {
        libvirtLoad();
        virDomainLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virDomainLookupByUUIDString");
        virDomainLookupByUUIDStringSymbolInit = (virDomainLookupByUUIDStringSymbol != NULL);
    }
    if (!virDomainLookupByUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainManagedSaveType virDomainManagedSaveSymbol;
    static bool virDomainManagedSaveSymbolInit;
    int ret;
    if (!virDomainManagedSaveSymbolInit) {
        libvirtLoad();
        virDomainManagedSaveSymbol = libvirtSymbol(libvirt, "virDomainManagedSave");
        virDomainManagedSaveSymbolInit = (virDomainManagedSaveSymbol != NULL);
    }
    if (!virDomainManagedSaveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainManagedSaveDefineXMLType virDomainManagedSaveDefineXMLSymbol;
    static bool virDomainManagedSaveDefineXMLSymbolInit;
    int ret;
    if (!virDomainManagedSaveDefineXMLSymbolInit) {
        libvirtLoad();
        virDomainManagedSaveDefineXMLSymbol = libvirtSymbol(libvirt, "virDomainManagedSaveDefineXML");
        virDomainManagedSaveDefineXMLSymbolInit = (virDomainManagedSaveDefineXMLSymbol != NULL);
    }
    if (!virDomainManagedSaveDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virDomainManagedSaveGetXMLDescType virDomainManagedSaveGetXMLDescSymbol;
    static bool virDomainManagedSaveGetXMLDescSymbolInit;
    char * ret;
    if (!virDomainManagedSaveGetXMLDescSymbolInit) {
        libvirtLoad();
        virDomainManagedSaveGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainManagedSaveGetXMLDesc");
        virDomainManagedSaveGetXMLDescSymbolInit = (virDomainManagedSaveGetXMLDescSymbol != NULL);
    }
    if (!virDomainManagedSaveGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainManagedSaveRemoveType virDomainManagedSaveRemoveSymbol;
    static bool virDomainManagedSaveRemoveSymbolInit;
    int ret;
    if (!virDomainManagedSaveRemoveSymbolInit) {
        libvirtLoad();
        virDomainManagedSaveRemoveSymbol = libvirtSymbol(libvirt, "virDomainManagedSaveRemove");
        virDomainManagedSaveRemoveSymbolInit = (virDomainManagedSaveRemoveSymbol != NULL);
    }
    if (!virDomainManagedSaveRemoveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainMemoryPeekType virDomainMemoryPeekSymbol;
    static bool virDomainMemoryPeekSymbolInit;
    int ret;
    if (!virDomainMemoryPeekSymbolInit) {
        libvirtLoad();
        virDomainMemoryPeekSymbol = libvirtSymbol(libvirt, "virDomainMemoryPeek");
        virDomainMemoryPeekSymbolInit = (virDomainMemoryPeekSymbol != NULL);
    }
    if (!virDomainMemoryPeekSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainMemoryStatsType virDomainMemoryStatsSymbol;
    static bool virDomainMemoryStatsSymbolInit;
    int ret;
    if (!virDomainMemoryStatsSymbolInit) {
        libvirtLoad();
        virDomainMemoryStatsSymbol = libvirtSymbol(libvirt, "virDomainMemoryStats");
        virDomainMemoryStatsSymbolInit = (virDomainMemoryStatsSymbol != NULL);
    }
    if (!virDomainMemoryStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainMigrateType virDomainMigrateSymbol;
    static bool virDomainMigrateSymbolInit;
    virDomainPtr ret;
    if (!virDomainMigrateSymbolInit) {
        libvirtLoad();
        virDomainMigrateSymbol = libvirtSymbol(libvirt, "virDomainMigrate");
        virDomainMigrateSymbolInit = (virDomainMigrateSymbol != NULL);
    }
    if (!virDomainMigrateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainMigrate2Type virDomainMigrate2Symbol;
    static bool virDomainMigrate2SymbolInit;
    virDomainPtr ret;
    if (!virDomainMigrate2SymbolInit) {
        libvirtLoad();
        virDomainMigrate2Symbol = libvirtSymbol(libvirt, "virDomainMigrate2");
        virDomainMigrate2SymbolInit = (virDomainMigrate2Symbol != NULL);
    }
    if (!virDomainMigrate2SymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainMigrate3Type virDomainMigrate3Symbol;
    static bool virDomainMigrate3SymbolInit;
    virDomainPtr ret;
    if (!virDomainMigrate3SymbolInit) {
        libvirtLoad();
        virDomainMigrate3Symbol = libvirtSymbol(libvirt, "virDomainMigrate3");
        virDomainMigrate3SymbolInit = (virDomainMigrate3Symbol != NULL);
    }
    if (!virDomainMigrate3SymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                           virErrorPtr err) {
    static virDomainMigrateGetCompressionCacheType virDomainMigrateGetCompressionCacheSymbol;
    static bool virDomainMigrateGetCompressionCacheSymbolInit;
    int ret;
    if (!virDomainMigrateGetCompressionCacheSymbolInit) {
        libvirtLoad();
        virDomainMigrateGetCompressionCacheSymbol = libvirtSymbol(libvirt, "virDomainMigrateGetCompressionCache");
        virDomainMigrateGetCompressionCacheSymbolInit = (virDomainMigrateGetCompressionCacheSymbol != NULL);
    }
    if (!virDomainMigrateGetCompressionCacheSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virDomainMigrateGetMaxDowntimeType virDomainMigrateGetMaxDowntimeSymbol;
    static bool virDomainMigrateGetMaxDowntimeSymbolInit;
    int ret;
    if (!virDomainMigrateGetMaxDowntimeSymbolInit) {
        libvirtLoad();
        virDomainMigrateGetMaxDowntimeSymbol = libvirtSymbol(libvirt, "virDomainMigrateGetMaxDowntime");
        virDomainMigrateGetMaxDowntimeSymbolInit = (virDomainMigrateGetMaxDowntimeSymbol != NULL);
    }
    if (!virDomainMigrateGetMaxDowntimeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainMigrateGetMaxSpeedType virDomainMigrateGetMaxSpeedSymbol;
    static bool virDomainMigrateGetMaxSpeedSymbolInit;
    int ret;
    if (!virDomainMigrateGetMaxSpeedSymbolInit) {
        libvirtLoad();
        virDomainMigrateGetMaxSpeedSymbol = libvirtSymbol(libvirt, "virDomainMigrateGetMaxSpeed");
        virDomainMigrateGetMaxSpeedSymbolInit = (virDomainMigrateGetMaxSpeedSymbol != NULL);
    }
    if (!virDomainMigrateGetMaxSpeedSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                           virErrorPtr err) {
    static virDomainMigrateSetCompressionCacheType virDomainMigrateSetCompressionCacheSymbol;
    static bool virDomainMigrateSetCompressionCacheSymbolInit;
    int ret;
    if (!virDomainMigrateSetCompressionCacheSymbolInit) {
        libvirtLoad();
        virDomainMigrateSetCompressionCacheSymbol = libvirtSymbol(libvirt, "virDomainMigrateSetCompressionCache");
        virDomainMigrateSetCompressionCacheSymbolInit = (virDomainMigrateSetCompressionCacheSymbol != NULL);
    }
    if (!virDomainMigrateSetCompressionCacheSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virDomainMigrateSetMaxDowntimeType virDomainMigrateSetMaxDowntimeSymbol;
    static bool virDomainMigrateSetMaxDowntimeSymbolInit;
    int ret;
    if (!virDomainMigrateSetMaxDowntimeSymbolInit) {
        libvirtLoad();
        virDomainMigrateSetMaxDowntimeSymbol = libvirtSymbol(libvirt, "virDomainMigrateSetMaxDowntime");
        virDomainMigrateSetMaxDowntimeSymbolInit = (virDomainMigrateSetMaxDowntimeSymbol != NULL);
    }
    if (!virDomainMigrateSetMaxDowntimeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainMigrateSetMaxSpeedType virDomainMigrateSetMaxSpeedSymbol;
    static bool virDomainMigrateSetMaxSpeedSymbolInit;
    int ret;
    if (!virDomainMigrateSetMaxSpeedSymbolInit) {
        libvirtLoad();
        virDomainMigrateSetMaxSpeedSymbol = libvirtSymbol(libvirt, "virDomainMigrateSetMaxSpeed");
        virDomainMigrateSetMaxSpeedSymbolInit = (virDomainMigrateSetMaxSpeedSymbol != NULL);
    }
    if (!virDomainMigrateSetMaxSpeedSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainMigrateStartPostCopyType virDomainMigrateStartPostCopySymbol;
    static bool virDomainMigrateStartPostCopySymbolInit;
    int ret;
    if (!virDomainMigrateStartPostCopySymbolInit) {
        libvirtLoad();
        virDomainMigrateStartPostCopySymbol = libvirtSymbol(libvirt, "virDomainMigrateStartPostCopy");
        virDomainMigrateStartPostCopySymbolInit = (virDomainMigrateStartPostCopySymbol != NULL);
    }
    if (!virDomainMigrateStartPostCopySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainMigrateToURIType virDomainMigrateToURISymbol;
    static bool virDomainMigrateToURISymbolInit;
    int ret;
    if (!virDomainMigrateToURISymbolInit) {
        libvirtLoad();
        virDomainMigrateToURISymbol = libvirtSymbol(libvirt, "virDomainMigrateToURI");
        virDomainMigrateToURISymbolInit = (virDomainMigrateToURISymbol != NULL);
    }
    if (!virDomainMigrateToURISymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainMigrateToURI2Type virDomainMigrateToURI2Symbol;
    static bool virDomainMigrateToURI2SymbolInit;
    int ret;
    if (!virDomainMigrateToURI2SymbolInit) {
        libvirtLoad();
        virDomainMigrateToURI2Symbol = libvirtSymbol(libvirt, "virDomainMigrateToURI2");
        virDomainMigrateToURI2SymbolInit = (virDomainMigrateToURI2Symbol != NULL);
    }
    if (!virDomainMigrateToURI2SymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainMigrateToURI3Type virDomainMigrateToURI3Symbol;
    static bool virDomainMigrateToURI3SymbolInit;
    int ret;
    if (!virDomainMigrateToURI3SymbolInit) {
        libvirtLoad();
        virDomainMigrateToURI3Symbol = libvirtSymbol(libvirt, "virDomainMigrateToURI3");
        virDomainMigrateToURI3SymbolInit = (virDomainMigrateToURI3Symbol != NULL);
    }
    if (!virDomainMigrateToURI3SymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainOpenChannelType virDomainOpenChannelSymbol;
    static bool virDomainOpenChannelSymbolInit;
    int ret;
    if (!virDomainOpenChannelSymbolInit) {
        libvirtLoad();
        virDomainOpenChannelSymbol = libvirtSymbol(libvirt, "virDomainOpenChannel");
        virDomainOpenChannelSymbolInit = (virDomainOpenChannelSymbol != NULL);
    }
    if (!virDomainOpenChannelSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainOpenConsoleType virDomainOpenConsoleSymbol;
    static bool virDomainOpenConsoleSymbolInit;
    int ret;
    if (!virDomainOpenConsoleSymbolInit) {
        libvirtLoad();
        virDomainOpenConsoleSymbol = libvirtSymbol(libvirt, "virDomainOpenConsole");
        virDomainOpenConsoleSymbolInit = (virDomainOpenConsoleSymbol != NULL);
    }
    if (!virDomainOpenConsoleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainOpenGraphicsType virDomainOpenGraphicsSymbol;
    static bool virDomainOpenGraphicsSymbolInit;
    int ret;
    if (!virDomainOpenGraphicsSymbolInit) {
        libvirtLoad();
        virDomainOpenGraphicsSymbol = libvirtSymbol(libvirt, "virDomainOpenGraphics");
        virDomainOpenGraphicsSymbolInit = (virDomainOpenGraphicsSymbol != NULL);
    }
    if (!virDomainOpenGraphicsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainOpenGraphicsFDType virDomainOpenGraphicsFDSymbol;
    static bool virDomainOpenGraphicsFDSymbolInit;
    int ret;
    if (!virDomainOpenGraphicsFDSymbolInit) {
        libvirtLoad();
        virDomainOpenGraphicsFDSymbol = libvirtSymbol(libvirt, "virDomainOpenGraphicsFD");
        virDomainOpenGraphicsFDSymbolInit = (virDomainOpenGraphicsFDSymbol != NULL);
    }
    if (!virDomainOpenGraphicsFDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainPMSuspendForDurationType virDomainPMSuspendForDurationSymbol;
    static bool virDomainPMSuspendForDurationSymbolInit;
    int ret;
    if (!virDomainPMSuspendForDurationSymbolInit) {
        libvirtLoad();
        virDomainPMSuspendForDurationSymbol = libvirtSymbol(libvirt, "virDomainPMSuspendForDuration");
        virDomainPMSuspendForDurationSymbolInit = (virDomainPMSuspendForDurationSymbol != NULL);
    }
    if (!virDomainPMSuspendForDurationSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainPMWakeupType virDomainPMWakeupSymbol;
    static bool virDomainPMWakeupSymbolInit;
    int ret;
    if (!virDomainPMWakeupSymbolInit) {
        libvirtLoad();
        virDomainPMWakeupSymbol = libvirtSymbol(libvirt, "virDomainPMWakeup");
        virDomainPMWakeupSymbolInit = (virDomainPMWakeupSymbol != NULL);
    }
    if (!virDomainPMWakeupSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainPinEmulatorType virDomainPinEmulatorSymbol;
    static bool virDomainPinEmulatorSymbolInit;
    int ret;
    if (!virDomainPinEmulatorSymbolInit) {
        libvirtLoad();
        virDomainPinEmulatorSymbol = libvirtSymbol(libvirt, "virDomainPinEmulator");
        virDomainPinEmulatorSymbolInit = (virDomainPinEmulatorSymbol != NULL);
    }
    if (!virDomainPinEmulatorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainPinIOThreadType virDomainPinIOThreadSymbol;
    static bool virDomainPinIOThreadSymbolInit;
    int ret;
    if (!virDomainPinIOThreadSymbolInit) {
        libvirtLoad();
        virDomainPinIOThreadSymbol = libvirtSymbol(libvirt, "virDomainPinIOThread");
        virDomainPinIOThreadSymbolInit = (virDomainPinIOThreadSymbol != NULL);
    }
    if (!virDomainPinIOThreadSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainPinVcpuType virDomainPinVcpuSymbol;
    static bool virDomainPinVcpuSymbolInit;
    int ret;
    if (!virDomainPinVcpuSymbolInit) {
        libvirtLoad();
        virDomainPinVcpuSymbol = libvirtSymbol(libvirt, "virDomainPinVcpu");
        virDomainPinVcpuSymbolInit = (virDomainPinVcpuSymbol != NULL);
    }
    if (!virDomainPinVcpuSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainPinVcpuFlagsType virDomainPinVcpuFlagsSymbol;
    static bool virDomainPinVcpuFlagsSymbolInit;
    int ret;
    if (!virDomainPinVcpuFlagsSymbolInit) {
        libvirtLoad();
        virDomainPinVcpuFlagsSymbol = libvirtSymbol(libvirt, "virDomainPinVcpuFlags");
        virDomainPinVcpuFlagsSymbolInit = (virDomainPinVcpuFlagsSymbol != NULL);
    }
    if (!virDomainPinVcpuFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virDomainRebootType virDomainRebootSymbol;
    static bool virDomainRebootSymbolInit;
    int ret;
    if (!virDomainRebootSymbolInit) {
        libvirtLoad();
        virDomainRebootSymbol = libvirtSymbol(libvirt, "virDomainReboot");
        virDomainRebootSymbolInit = (virDomainRebootSymbol != NULL);
    }
    if (!virDomainRebootSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                    virErrorPtr err) {
    static virDomainRefType virDomainRefSymbol;
    static bool virDomainRefSymbolInit;
    int ret;
    if (!virDomainRefSymbolInit) {
        libvirtLoad();
        virDomainRefSymbol = libvirtSymbol(libvirt, "virDomainRef");
        virDomainRefSymbolInit = (virDomainRefSymbol != NULL);
    }
    if (!virDomainRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virDomainRenameType virDomainRenameSymbol;
    static bool virDomainRenameSymbolInit;
    int ret;
    if (!virDomainRenameSymbolInit) {
        libvirtLoad();
        virDomainRenameSymbol = libvirtSymbol(libvirt, "virDomainRename");
        virDomainRenameSymbolInit = (virDomainRenameSymbol != NULL);
    }
    if (!virDomainRenameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virDomainResetType virDomainResetSymbol;
    static bool virDomainResetSymbolInit;
    int ret;
    if (!virDomainResetSymbolInit) {
        libvirtLoad();
        virDomainResetSymbol = libvirtSymbol(libvirt, "virDomainReset");
        virDomainResetSymbolInit = (virDomainResetSymbol != NULL);
    }
    if (!virDomainResetSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainRestoreType virDomainRestoreSymbol;
    static bool virDomainRestoreSymbolInit;
    int ret;
    if (!virDomainRestoreSymbolInit) {
        libvirtLoad();
        virDomainRestoreSymbol = libvirtSymbol(libvirt, "virDomainRestore");
        virDomainRestoreSymbolInit = (virDomainRestoreSymbol != NULL);
    }
    if (!virDomainRestoreSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainRestoreFlagsType virDomainRestoreFlagsSymbol;
    static bool virDomainRestoreFlagsSymbolInit;
    int ret;
    if (!virDomainRestoreFlagsSymbolInit) {
        libvirtLoad();
        virDomainRestoreFlagsSymbol = libvirtSymbol(libvirt, "virDomainRestoreFlags");
        virDomainRestoreFlagsSymbolInit = (virDomainRestoreFlagsSymbol != NULL);
    }
    if (!virDomainRestoreFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virDomainResumeType virDomainResumeSymbol;
    static bool virDomainResumeSymbolInit;
    int ret;
    if (!virDomainResumeSymbolInit) {
        libvirtLoad();
        virDomainResumeSymbol = libvirtSymbol(libvirt, "virDomainResume");
        virDomainResumeSymbolInit = (virDomainResumeSymbol != NULL);
    }
    if (!virDomainResumeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainRevertToSnapshotType virDomainRevertToSnapshotSymbol;
    static bool virDomainRevertToSnapshotSymbolInit;
    int ret;
    if (!virDomainRevertToSnapshotSymbolInit) {
        libvirtLoad();
        virDomainRevertToSnapshotSymbol = libvirtSymbol(libvirt, "virDomainRevertToSnapshot");
        virDomainRevertToSnapshotSymbolInit = (virDomainRevertToSnapshotSymbol != NULL);
    }
    if (!virDomainRevertToSnapshotSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virDomainSaveType virDomainSaveSymbol;
    static bool virDomainSaveSymbolInit;
    int ret;
    if (!virDomainSaveSymbolInit) {
        libvirtLoad();
        virDomainSaveSymbol = libvirtSymbol(libvirt, "virDomainSave");
        virDomainSaveSymbolInit = (virDomainSaveSymbol != NULL);
    }
    if (!virDomainSaveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainSaveFlagsType virDomainSaveFlagsSymbol;
    static bool virDomainSaveFlagsSymbolInit;
    int ret;
    if (!virDomainSaveFlagsSymbolInit) {
        libvirtLoad();
        virDomainSaveFlagsSymbol = libvirtSymbol(libvirt, "virDomainSaveFlags");
        virDomainSaveFlagsSymbolInit = (virDomainSaveFlagsSymbol != NULL);
    }
    if (!virDomainSaveFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainSaveImageDefineXMLType virDomainSaveImageDefineXMLSymbol;
    static bool virDomainSaveImageDefineXMLSymbolInit;
    int ret;
    if (!virDomainSaveImageDefineXMLSymbolInit) {
        libvirtLoad();
        virDomainSaveImageDefineXMLSymbol = libvirtSymbol(libvirt, "virDomainSaveImageDefineXML");
        virDomainSaveImageDefineXMLSymbolInit = (virDomainSaveImageDefineXMLSymbol != NULL);
    }
    if (!virDomainSaveImageDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainSaveImageGetXMLDescType virDomainSaveImageGetXMLDescSymbol;
    static bool virDomainSaveImageGetXMLDescSymbolInit;
    char * ret;
    if (!virDomainSaveImageGetXMLDescSymbolInit) {
        libvirtLoad();
        virDomainSaveImageGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainSaveImageGetXMLDesc");
        virDomainSaveImageGetXMLDescSymbolInit = (virDomainSaveImageGetXMLDescSymbol != NULL);
    }
    if (!virDomainSaveImageGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainScreenshotType virDomainScreenshotSymbol;
    static bool virDomainScreenshotSymbolInit;
    char * ret;
    if (!virDomainScreenshotSymbolInit) {
        libvirtLoad();
        virDomainScreenshotSymbol = libvirtSymbol(libvirt, "virDomainScreenshot");
        virDomainScreenshotSymbolInit = (virDomainScreenshotSymbol != NULL);
    }
    if (!virDomainScreenshotSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainSendKeyType virDomainSendKeySymbol;
    static bool virDomainSendKeySymbolInit;
    int ret;
    if (!virDomainSendKeySymbolInit) {
        libvirtLoad();
        virDomainSendKeySymbol = libvirtSymbol(libvirt, "virDomainSendKey");
        virDomainSendKeySymbolInit = (virDomainSendKeySymbol != NULL);
    }
    if (!virDomainSendKeySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSendProcessSignalType virDomainSendProcessSignalSymbol;
    static bool virDomainSendProcessSignalSymbolInit;
    int ret;
    if (!virDomainSendProcessSignalSymbolInit) {
        libvirtLoad();
        virDomainSendProcessSignalSymbol = libvirtSymbol(libvirt, "virDomainSendProcessSignal");
        virDomainSendProcessSignalSymbolInit = (virDomainSendProcessSignalSymbol != NULL);
    }
    if (!virDomainSendProcessSignalSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainSetAutostartType virDomainSetAutostartSymbol;
    static bool virDomainSetAutostartSymbolInit;
    int ret;
    if (!virDomainSetAutostartSymbolInit) {
        libvirtLoad();
        virDomainSetAutostartSymbol = libvirtSymbol(libvirt, "virDomainSetAutostart");
        virDomainSetAutostartSymbolInit = (virDomainSetAutostartSymbol != NULL);
    }
    if (!virDomainSetAutostartSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainSetBlkioParametersType virDomainSetBlkioParametersSymbol;
    static bool virDomainSetBlkioParametersSymbolInit;
    int ret;
    if (!virDomainSetBlkioParametersSymbolInit) {
        libvirtLoad();
        virDomainSetBlkioParametersSymbol = libvirtSymbol(libvirt, "virDomainSetBlkioParameters");
        virDomainSetBlkioParametersSymbolInit = (virDomainSetBlkioParametersSymbol != NULL);
    }
    if (!virDomainSetBlkioParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainSetBlockIoTuneType virDomainSetBlockIoTuneSymbol;
    static bool virDomainSetBlockIoTuneSymbolInit;
    int ret;
    if (!virDomainSetBlockIoTuneSymbolInit) {
        libvirtLoad();
        virDomainSetBlockIoTuneSymbol = libvirtSymbol(libvirt, "virDomainSetBlockIoTune");
        virDomainSetBlockIoTuneSymbolInit = (virDomainSetBlockIoTuneSymbol != NULL);
    }
    if (!virDomainSetBlockIoTuneSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSetBlockThresholdType virDomainSetBlockThresholdSymbol;
    static bool virDomainSetBlockThresholdSymbolInit;
    int ret;
    if (!virDomainSetBlockThresholdSymbolInit) {
        libvirtLoad();
        virDomainSetBlockThresholdSymbol = libvirtSymbol(libvirt, "virDomainSetBlockThreshold");
        virDomainSetBlockThresholdSymbolInit = (virDomainSetBlockThresholdSymbol != NULL);
    }
    if (!virDomainSetBlockThresholdSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainSetGuestVcpusType virDomainSetGuestVcpusSymbol;
    static bool virDomainSetGuestVcpusSymbolInit;
    int ret;
    if (!virDomainSetGuestVcpusSymbolInit) {
        libvirtLoad();
        virDomainSetGuestVcpusSymbol = libvirtSymbol(libvirt, "virDomainSetGuestVcpus");
        virDomainSetGuestVcpusSymbolInit = (virDomainSetGuestVcpusSymbol != NULL);
    }
    if (!virDomainSetGuestVcpusSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSetIOThreadParamsType virDomainSetIOThreadParamsSymbol;
    static bool virDomainSetIOThreadParamsSymbolInit;
    int ret;
    if (!virDomainSetIOThreadParamsSymbolInit) {
        libvirtLoad();
        virDomainSetIOThreadParamsSymbol = libvirtSymbol(libvirt, "virDomainSetIOThreadParams");
        virDomainSetIOThreadParamsSymbolInit = (virDomainSetIOThreadParamsSymbol != NULL);
    }
    if (!virDomainSetIOThreadParamsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virDomainSetInterfaceParametersType virDomainSetInterfaceParametersSymbol;
    static bool virDomainSetInterfaceParametersSymbolInit;
    int ret;
    if (!virDomainSetInterfaceParametersSymbolInit) {
        libvirtLoad();
        virDomainSetInterfaceParametersSymbol = libvirtSymbol(libvirt, "virDomainSetInterfaceParameters");
        virDomainSetInterfaceParametersSymbolInit = (virDomainSetInterfaceParametersSymbol != NULL);
    }
    if (!virDomainSetInterfaceParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainSetLifecycleActionType virDomainSetLifecycleActionSymbol;
    static bool virDomainSetLifecycleActionSymbolInit;
    int ret;
    if (!virDomainSetLifecycleActionSymbolInit) {
        libvirtLoad();
        virDomainSetLifecycleActionSymbol = libvirtSymbol(libvirt, "virDomainSetLifecycleAction");
        virDomainSetLifecycleActionSymbolInit = (virDomainSetLifecycleActionSymbol != NULL);
    }
    if (!virDomainSetLifecycleActionSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainSetMaxMemoryType virDomainSetMaxMemorySymbol;
    static bool virDomainSetMaxMemorySymbolInit;
    int ret;
    if (!virDomainSetMaxMemorySymbolInit) {
        libvirtLoad();
        virDomainSetMaxMemorySymbol = libvirtSymbol(libvirt, "virDomainSetMaxMemory");
        virDomainSetMaxMemorySymbolInit = (virDomainSetMaxMemorySymbol != NULL);
    }
    if (!virDomainSetMaxMemorySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virDomainSetMemoryType virDomainSetMemorySymbol;
    static bool virDomainSetMemorySymbolInit;
    int ret;
    if (!virDomainSetMemorySymbolInit) {
        libvirtLoad();
        virDomainSetMemorySymbol = libvirtSymbol(libvirt, "virDomainSetMemory");
        virDomainSetMemorySymbolInit = (virDomainSetMemorySymbol != NULL);
    }
    if (!virDomainSetMemorySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainSetMemoryFlagsType virDomainSetMemoryFlagsSymbol;
    static bool virDomainSetMemoryFlagsSymbolInit;
    int ret;
    if (!virDomainSetMemoryFlagsSymbolInit) {
        libvirtLoad();
        virDomainSetMemoryFlagsSymbol = libvirtSymbol(libvirt, "virDomainSetMemoryFlags");
        virDomainSetMemoryFlagsSymbolInit = (virDomainSetMemoryFlagsSymbol != NULL);
    }
    if (!virDomainSetMemoryFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainSetMemoryParametersType virDomainSetMemoryParametersSymbol;
    static bool virDomainSetMemoryParametersSymbolInit;
    int ret;
    if (!virDomainSetMemoryParametersSymbolInit) {
        libvirtLoad();
        virDomainSetMemoryParametersSymbol = libvirtSymbol(libvirt, "virDomainSetMemoryParameters");
        virDomainSetMemoryParametersSymbolInit = (virDomainSetMemoryParametersSymbol != NULL);
    }
    if (!virDomainSetMemoryParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainSetMemoryStatsPeriodType virDomainSetMemoryStatsPeriodSymbol;
    static bool virDomainSetMemoryStatsPeriodSymbolInit;
    int ret;
    if (!virDomainSetMemoryStatsPeriodSymbolInit) {
        libvirtLoad();
        virDomainSetMemoryStatsPeriodSymbol = libvirtSymbol(libvirt, "virDomainSetMemoryStatsPeriod");
        virDomainSetMemoryStatsPeriodSymbolInit = (virDomainSetMemoryStatsPeriodSymbol != NULL);
    }
    if (!virDomainSetMemoryStatsPeriodSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainSetMetadataType virDomainSetMetadataSymbol;
    static bool virDomainSetMetadataSymbolInit;
    int ret;
    if (!virDomainSetMetadataSymbolInit) {
        libvirtLoad();
        virDomainSetMetadataSymbol = libvirtSymbol(libvirt, "virDomainSetMetadata");
        virDomainSetMetadataSymbolInit = (virDomainSetMetadataSymbol != NULL);
    }
    if (!virDomainSetMetadataSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSetNumaParametersType virDomainSetNumaParametersSymbol;
    static bool virDomainSetNumaParametersSymbolInit;
    int ret;
    if (!virDomainSetNumaParametersSymbolInit) {
        libvirtLoad();
        virDomainSetNumaParametersSymbol = libvirtSymbol(libvirt, "virDomainSetNumaParameters");
        virDomainSetNumaParametersSymbolInit = (virDomainSetNumaParametersSymbol != NULL);
    }
    if (!virDomainSetNumaParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainSetPerfEventsType virDomainSetPerfEventsSymbol;
    static bool virDomainSetPerfEventsSymbolInit;
    int ret;
    if (!virDomainSetPerfEventsSymbolInit) {
        libvirtLoad();
        virDomainSetPerfEventsSymbol = libvirtSymbol(libvirt, "virDomainSetPerfEvents");
        virDomainSetPerfEventsSymbolInit = (virDomainSetPerfEventsSymbol != NULL);
    }
    if (!virDomainSetPerfEventsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virDomainSetSchedulerParametersType virDomainSetSchedulerParametersSymbol;
    static bool virDomainSetSchedulerParametersSymbolInit;
    int ret;
    if (!virDomainSetSchedulerParametersSymbolInit) {
        libvirtLoad();
        virDomainSetSchedulerParametersSymbol = libvirtSymbol(libvirt, "virDomainSetSchedulerParameters");
        virDomainSetSchedulerParametersSymbolInit = (virDomainSetSchedulerParametersSymbol != NULL);
    }
    if (!virDomainSetSchedulerParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                            virErrorPtr err) {
    static virDomainSetSchedulerParametersFlagsType virDomainSetSchedulerParametersFlagsSymbol;
    static bool virDomainSetSchedulerParametersFlagsSymbolInit;
    int ret;
    if (!virDomainSetSchedulerParametersFlagsSymbolInit) {
        libvirtLoad();
        virDomainSetSchedulerParametersFlagsSymbol = libvirtSymbol(libvirt, "virDomainSetSchedulerParametersFlags");
        virDomainSetSchedulerParametersFlagsSymbolInit = (virDomainSetSchedulerParametersFlagsSymbol != NULL);
    }
    if (!virDomainSetSchedulerParametersFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainSetTimeType virDomainSetTimeSymbol;
    static bool virDomainSetTimeSymbolInit;
    int ret;
    if (!virDomainSetTimeSymbolInit) {
        libvirtLoad();
        virDomainSetTimeSymbol = libvirtSymbol(libvirt, "virDomainSetTime");
        virDomainSetTimeSymbolInit = (virDomainSetTimeSymbol != NULL);
    }
    if (!virDomainSetTimeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainSetUserPasswordType virDomainSetUserPasswordSymbol;
    static bool virDomainSetUserPasswordSymbolInit;
    int ret;
    if (!virDomainSetUserPasswordSymbolInit) {
        libvirtLoad();
        virDomainSetUserPasswordSymbol = libvirtSymbol(libvirt, "virDomainSetUserPassword");
        virDomainSetUserPasswordSymbolInit = (virDomainSetUserPasswordSymbol != NULL);
    }
    if (!virDomainSetUserPasswordSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virDomainSetVcpuType virDomainSetVcpuSymbol;
    static bool virDomainSetVcpuSymbolInit;
    int ret;
    if (!virDomainSetVcpuSymbolInit) {
        libvirtLoad();
        virDomainSetVcpuSymbol = libvirtSymbol(libvirt, "virDomainSetVcpu");
        virDomainSetVcpuSymbolInit = (virDomainSetVcpuSymbol != NULL);
    }
    if (!virDomainSetVcpuSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainSetVcpusType virDomainSetVcpusSymbol;
    static bool virDomainSetVcpusSymbolInit;
    int ret;
    if (!virDomainSetVcpusSymbolInit) {
        libvirtLoad();
        virDomainSetVcpusSymbol = libvirtSymbol(libvirt, "virDomainSetVcpus");
        virDomainSetVcpusSymbolInit = (virDomainSetVcpusSymbol != NULL);
    }
    if (!virDomainSetVcpusSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainSetVcpusFlagsType virDomainSetVcpusFlagsSymbol;
    static bool virDomainSetVcpusFlagsSymbolInit;
    int ret;
    if (!virDomainSetVcpusFlagsSymbolInit) {
        libvirtLoad();
        virDomainSetVcpusFlagsSymbol = libvirtSymbol(libvirt, "virDomainSetVcpusFlags");
        virDomainSetVcpusFlagsSymbolInit = (virDomainSetVcpusFlagsSymbol != NULL);
    }
    if (!virDomainSetVcpusFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainShutdownType virDomainShutdownSymbol;
    static bool virDomainShutdownSymbolInit;
    int ret;
    if (!virDomainShutdownSymbolInit) {
        libvirtLoad();
        virDomainShutdownSymbol = libvirtSymbol(libvirt, "virDomainShutdown");
        virDomainShutdownSymbolInit = (virDomainShutdownSymbol != NULL);
    }
    if (!virDomainShutdownSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainShutdownFlagsType virDomainShutdownFlagsSymbol;
    static bool virDomainShutdownFlagsSymbolInit;
    int ret;
    if (!virDomainShutdownFlagsSymbolInit) {
        libvirtLoad();
        virDomainShutdownFlagsSymbol = libvirtSymbol(libvirt, "virDomainShutdownFlags");
        virDomainShutdownFlagsSymbolInit = (virDomainShutdownFlagsSymbol != NULL);
    }
    if (!virDomainShutdownFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSnapshotCreateXMLType virDomainSnapshotCreateXMLSymbol;
    static bool virDomainSnapshotCreateXMLSymbolInit;
    virDomainSnapshotPtr ret;
    if (!virDomainSnapshotCreateXMLSymbolInit) {
        libvirtLoad();
        virDomainSnapshotCreateXMLSymbol = libvirtSymbol(libvirt, "virDomainSnapshotCreateXML");
        virDomainSnapshotCreateXMLSymbolInit = (virDomainSnapshotCreateXMLSymbol != NULL);
    }
    if (!virDomainSnapshotCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainSnapshotCurrentType virDomainSnapshotCurrentSymbol;
    static bool virDomainSnapshotCurrentSymbolInit;
    virDomainSnapshotPtr ret;
    if (!virDomainSnapshotCurrentSymbolInit) {
        libvirtLoad();
        virDomainSnapshotCurrentSymbol = libvirtSymbol(libvirt, "virDomainSnapshotCurrent");
        virDomainSnapshotCurrentSymbolInit = (virDomainSnapshotCurrentSymbol != NULL);
    }
    if (!virDomainSnapshotCurrentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virDomainSnapshotDeleteType virDomainSnapshotDeleteSymbol;
    static bool virDomainSnapshotDeleteSymbolInit;
    int ret;
    if (!virDomainSnapshotDeleteSymbolInit) {
        libvirtLoad();
        virDomainSnapshotDeleteSymbol = libvirtSymbol(libvirt, "virDomainSnapshotDelete");
        virDomainSnapshotDeleteSymbolInit = (virDomainSnapshotDeleteSymbol != NULL);
    }
    if (!virDomainSnapshotDeleteSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virDomainSnapshotFreeType virDomainSnapshotFreeSymbol;
    static bool virDomainSnapshotFreeSymbolInit;
    int ret;
    if (!virDomainSnapshotFreeSymbolInit) {
        libvirtLoad();
        virDomainSnapshotFreeSymbol = libvirtSymbol(libvirt, "virDomainSnapshotFree");
        virDomainSnapshotFreeSymbolInit = (virDomainSnapshotFreeSymbol != NULL);
    }
    if (!virDomainSnapshotFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainSnapshotGetConnectType virDomainSnapshotGetConnectSymbol;
    static bool virDomainSnapshotGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virDomainSnapshotGetConnectSymbolInit) {
        libvirtLoad();
        virDomainSnapshotGetConnectSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetConnect");
        virDomainSnapshotGetConnectSymbolInit = (virDomainSnapshotGetConnectSymbol != NULL);
    }
    if (!virDomainSnapshotGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSnapshotGetDomainType virDomainSnapshotGetDomainSymbol;
    static bool virDomainSnapshotGetDomainSymbolInit;
    virDomainPtr ret;
    if (!virDomainSnapshotGetDomainSymbolInit) {
        libvirtLoad();
        virDomainSnapshotGetDomainSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetDomain");
        virDomainSnapshotGetDomainSymbolInit = (virDomainSnapshotGetDomainSymbol != NULL);
    }
    if (!virDomainSnapshotGetDomainSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virDomainSnapshotGetNameType virDomainSnapshotGetNameSymbol;
    static bool virDomainSnapshotGetNameSymbolInit;
    const char * ret;
    if (!virDomainSnapshotGetNameSymbolInit) {
        libvirtLoad();
        virDomainSnapshotGetNameSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetName");
        virDomainSnapshotGetNameSymbolInit = (virDomainSnapshotGetNameSymbol != NULL);
    }
    if (!virDomainSnapshotGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSnapshotGetParentType virDomainSnapshotGetParentSymbol;
    static bool virDomainSnapshotGetParentSymbolInit;
    virDomainSnapshotPtr ret;
    if (!virDomainSnapshotGetParentSymbolInit) {
        libvirtLoad();
        virDomainSnapshotGetParentSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetParent");
        virDomainSnapshotGetParentSymbolInit = (virDomainSnapshotGetParentSymbol != NULL);
    }
    if (!virDomainSnapshotGetParentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainSnapshotGetXMLDescType virDomainSnapshotGetXMLDescSymbol;
    static bool virDomainSnapshotGetXMLDescSymbolInit;
    char * ret;
    if (!virDomainSnapshotGetXMLDescSymbolInit) {
        libvirtLoad();
        virDomainSnapshotGetXMLDescSymbol = libvirtSymbol(libvirt, "virDomainSnapshotGetXMLDesc");
        virDomainSnapshotGetXMLDescSymbolInit = (virDomainSnapshotGetXMLDescSymbol != NULL);
    }
    if (!virDomainSnapshotGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainSnapshotHasMetadataType virDomainSnapshotHasMetadataSymbol;
    static bool virDomainSnapshotHasMetadataSymbolInit;
    int ret;
    if (!virDomainSnapshotHasMetadataSymbolInit) {
        libvirtLoad();
        virDomainSnapshotHasMetadataSymbol = libvirtSymbol(libvirt, "virDomainSnapshotHasMetadata");
        virDomainSnapshotHasMetadataSymbolInit = (virDomainSnapshotHasMetadataSymbol != NULL);
    }
    if (!virDomainSnapshotHasMetadataSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSnapshotIsCurrentType virDomainSnapshotIsCurrentSymbol;
    static bool virDomainSnapshotIsCurrentSymbolInit;
    int ret;
    if (!virDomainSnapshotIsCurrentSymbolInit) {
        libvirtLoad();
        virDomainSnapshotIsCurrentSymbol = libvirtSymbol(libvirt, "virDomainSnapshotIsCurrent");
        virDomainSnapshotIsCurrentSymbolInit = (virDomainSnapshotIsCurrentSymbol != NULL);
    }
    if (!virDomainSnapshotIsCurrentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virDomainSnapshotListAllChildrenType virDomainSnapshotListAllChildrenSymbol;
    static bool virDomainSnapshotListAllChildrenSymbolInit;
    int ret;
    if (!virDomainSnapshotListAllChildrenSymbolInit) {
        libvirtLoad();
        virDomainSnapshotListAllChildrenSymbol = libvirtSymbol(libvirt, "virDomainSnapshotListAllChildren");
        virDomainSnapshotListAllChildrenSymbolInit = (virDomainSnapshotListAllChildrenSymbol != NULL);
    }
    if (!virDomainSnapshotListAllChildrenSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                          virErrorPtr err) {
    static virDomainSnapshotListChildrenNamesType virDomainSnapshotListChildrenNamesSymbol;
    static bool virDomainSnapshotListChildrenNamesSymbolInit;
    int ret;
    if (!virDomainSnapshotListChildrenNamesSymbolInit) {
        libvirtLoad();
        virDomainSnapshotListChildrenNamesSymbol = libvirtSymbol(libvirt, "virDomainSnapshotListChildrenNames");
        virDomainSnapshotListChildrenNamesSymbolInit = (virDomainSnapshotListChildrenNamesSymbol != NULL);
    }
    if (!virDomainSnapshotListChildrenNamesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainSnapshotListNamesType virDomainSnapshotListNamesSymbol;
    static bool virDomainSnapshotListNamesSymbolInit;
    int ret;
    if (!virDomainSnapshotListNamesSymbolInit) {
        libvirtLoad();
        virDomainSnapshotListNamesSymbol = libvirtSymbol(libvirt, "virDomainSnapshotListNames");
        virDomainSnapshotListNamesSymbolInit = (virDomainSnapshotListNamesSymbol != NULL);
    }
    if (!virDomainSnapshotListNamesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virDomainSnapshotLookupByNameType virDomainSnapshotLookupByNameSymbol;
    static bool virDomainSnapshotLookupByNameSymbolInit;
    virDomainSnapshotPtr ret;
    if (!virDomainSnapshotLookupByNameSymbolInit) {
        libvirtLoad();
        virDomainSnapshotLookupByNameSymbol = libvirtSymbol(libvirt, "virDomainSnapshotLookupByName");
        virDomainSnapshotLookupByNameSymbolInit = (virDomainSnapshotLookupByNameSymbol != NULL);
    }
    if (!virDomainSnapshotLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainSnapshotNumType virDomainSnapshotNumSymbol;
    static bool virDomainSnapshotNumSymbolInit;
    int ret;
    if (!virDomainSnapshotNumSymbolInit) {
        libvirtLoad();
        virDomainSnapshotNumSymbol = libvirtSymbol(libvirt, "virDomainSnapshotNum");
        virDomainSnapshotNumSymbolInit = (virDomainSnapshotNumSymbol != NULL);
    }
    if (!virDomainSnapshotNumSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virDomainSnapshotNumChildrenType virDomainSnapshotNumChildrenSymbol;
    static bool virDomainSnapshotNumChildrenSymbolInit;
    int ret;
    if (!virDomainSnapshotNumChildrenSymbolInit) {
        libvirtLoad();
        virDomainSnapshotNumChildrenSymbol = libvirtSymbol(libvirt, "virDomainSnapshotNumChildren");
        virDomainSnapshotNumChildrenSymbolInit = (virDomainSnapshotNumChildrenSymbol != NULL);
    }
    if (!virDomainSnapshotNumChildrenSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virDomainSnapshotRefType virDomainSnapshotRefSymbol;
    static bool virDomainSnapshotRefSymbolInit;
    int ret;
    if (!virDomainSnapshotRefSymbolInit) {
        libvirtLoad();
        virDomainSnapshotRefSymbol = libvirtSymbol(libvirt, "virDomainSnapshotRef");
        virDomainSnapshotRefSymbolInit = (virDomainSnapshotRefSymbol != NULL);
    }
    if (!virDomainSnapshotRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainStartDirtyRateCalcType virDomainStartDirtyRateCalcSymbol;
    static bool virDomainStartDirtyRateCalcSymbolInit;
    int ret;
    if (!virDomainStartDirtyRateCalcSymbolInit) {
        libvirtLoad();
        virDomainStartDirtyRateCalcSymbol = libvirtSymbol(libvirt, "virDomainStartDirtyRateCalc");
        virDomainStartDirtyRateCalcSymbolInit = (virDomainStartDirtyRateCalcSymbol != NULL);
    }
    if (!virDomainStartDirtyRateCalcSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats) {
    static virDomainStatsRecordListFreeType virDomainStatsRecordListFreeSymbol;
    static bool virDomainStatsRecordListFreeSymbolInit;

    if (!virDomainStatsRecordListFreeSymbolInit) {
        libvirtLoad();
        virDomainStatsRecordListFreeSymbol = libvirtSymbol(libvirt, "virDomainStatsRecordListFree");
        virDomainStatsRecordListFreeSymbolInit = (virDomainStatsRecordListFreeSymbol != NULL);
    }
    if (!virDomainStatsRecordListFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virDomainStatsRecordListFreeSymbol(stats);

}
typedef int
(*virDomainSuspendType)(virDomainPtr domain);

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err) {
    static virDomainSuspendType virDomainSuspendSymbol;
    static bool virDomainSuspendSymbolInit;
    int ret;
    if (!virDomainSuspendSymbolInit) {
        libvirtLoad();
        virDomainSuspendSymbol = libvirtSymbol(libvirt, "virDomainSuspend");
        virDomainSuspendSymbolInit = (virDomainSuspendSymbol != NULL);
    }
    if (!virDomainSuspendSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virDomainUndefineType virDomainUndefineSymbol;
    static bool virDomainUndefineSymbolInit;
    int ret;
    if (!virDomainUndefineSymbolInit) {
        libvirtLoad();
        virDomainUndefineSymbol = libvirtSymbol(libvirt, "virDomainUndefine");
        virDomainUndefineSymbolInit = (virDomainUndefineSymbol != NULL);
    }
    if (!virDomainUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virDomainUndefineFlagsType virDomainUndefineFlagsSymbol;
    static bool virDomainUndefineFlagsSymbolInit;
    int ret;
    if (!virDomainUndefineFlagsSymbolInit) {
        libvirtLoad();
        virDomainUndefineFlagsSymbol = libvirtSymbol(libvirt, "virDomainUndefineFlags");
        virDomainUndefineFlagsSymbolInit = (virDomainUndefineFlagsSymbol != NULL);
    }
    if (!virDomainUndefineFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainUpdateDeviceFlagsType virDomainUpdateDeviceFlagsSymbol;
    static bool virDomainUpdateDeviceFlagsSymbolInit;
    int ret;
    if (!virDomainUpdateDeviceFlagsSymbolInit) {
        libvirtLoad();
        virDomainUpdateDeviceFlagsSymbol = libvirtSymbol(libvirt, "virDomainUpdateDeviceFlags");
        virDomainUpdateDeviceFlagsSymbolInit = (virDomainUpdateDeviceFlagsSymbol != NULL);
    }
    if (!virDomainUpdateDeviceFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virEventAddHandleType virEventAddHandleSymbol;
    static bool virEventAddHandleSymbolInit;
    int ret;
    if (!virEventAddHandleSymbolInit) {
        libvirtLoad();
        virEventAddHandleSymbol = libvirtSymbol(libvirt, "virEventAddHandle");
        virEventAddHandleSymbolInit = (virEventAddHandleSymbol != NULL);
    }
    if (!virEventAddHandleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virEventAddTimeoutType virEventAddTimeoutSymbol;
    static bool virEventAddTimeoutSymbolInit;
    int ret;
    if (!virEventAddTimeoutSymbolInit) {
        libvirtLoad();
        virEventAddTimeoutSymbol = libvirtSymbol(libvirt, "virEventAddTimeout");
        virEventAddTimeoutSymbolInit = (virEventAddTimeoutSymbol != NULL);
    }
    if (!virEventAddTimeoutSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virEventRegisterDefaultImplWrapper(virErrorPtr err) {
    static virEventRegisterDefaultImplType virEventRegisterDefaultImplSymbol;
    static bool virEventRegisterDefaultImplSymbolInit;
    int ret;
    if (!virEventRegisterDefaultImplSymbolInit) {
        libvirtLoad();
        virEventRegisterDefaultImplSymbol = libvirtSymbol(libvirt, "virEventRegisterDefaultImpl");
        virEventRegisterDefaultImplSymbolInit = (virEventRegisterDefaultImplSymbol != NULL);
    }
    if (!virEventRegisterDefaultImplSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virEventRemoveTimeoutFunc removeTimeout) {
    static virEventRegisterImplType virEventRegisterImplSymbol;
    static bool virEventRegisterImplSymbolInit;

    if (!virEventRegisterImplSymbolInit) {
        libvirtLoad();
        virEventRegisterImplSymbol = libvirtSymbol(libvirt, "virEventRegisterImpl");
        virEventRegisterImplSymbolInit = (virEventRegisterImplSymbol != NULL);
    }
    if (!virEventRegisterImplSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virEventRemoveHandleType virEventRemoveHandleSymbol;
    static bool virEventRemoveHandleSymbolInit;
    int ret;
    if (!virEventRemoveHandleSymbolInit) {
        libvirtLoad();
        virEventRemoveHandleSymbol = libvirtSymbol(libvirt, "virEventRemoveHandle");
        virEventRemoveHandleSymbolInit = (virEventRemoveHandleSymbol != NULL);
    }
    if (!virEventRemoveHandleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virEventRemoveTimeoutType virEventRemoveTimeoutSymbol;
    static bool virEventRemoveTimeoutSymbolInit;
    int ret;
    if (!virEventRemoveTimeoutSymbolInit) {
        libvirtLoad();
        virEventRemoveTimeoutSymbol = libvirtSymbol(libvirt, "virEventRemoveTimeout");
        virEventRemoveTimeoutSymbolInit = (virEventRemoveTimeoutSymbol != NULL);
    }
    if (!virEventRemoveTimeoutSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virEventRunDefaultImplWrapper(virErrorPtr err) {
    static virEventRunDefaultImplType virEventRunDefaultImplSymbol;
    static bool virEventRunDefaultImplSymbolInit;
    int ret;
    if (!virEventRunDefaultImplSymbolInit) {
        libvirtLoad();
        virEventRunDefaultImplSymbol = libvirtSymbol(libvirt, "virEventRunDefaultImpl");
        virEventRunDefaultImplSymbolInit = (virEventRunDefaultImplSymbol != NULL);
    }
    if (!virEventRunDefaultImplSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            int events) {
    static virEventUpdateHandleType virEventUpdateHandleSymbol;
    static bool virEventUpdateHandleSymbolInit;

    if (!virEventUpdateHandleSymbolInit) {
        libvirtLoad();
        virEventUpdateHandleSymbol = libvirtSymbol(libvirt, "virEventUpdateHandle");
        virEventUpdateHandleSymbolInit = (virEventUpdateHandleSymbol != NULL);
    }
    if (!virEventUpdateHandleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virEventUpdateHandleSymbol(watch,
                               events);

}
typedef void
(*virEventUpdateTimeoutType)(int timer,
                             int timeout);

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout) {
    static virEventUpdateTimeoutType virEventUpdateTimeoutSymbol;
    static bool virEventUpdateTimeoutSymbolInit;

    if (!virEventUpdateTimeoutSymbolInit) {
        libvirtLoad();
        virEventUpdateTimeoutSymbol = libvirtSymbol(libvirt, "virEventUpdateTimeout");
        virEventUpdateTimeoutSymbolInit = (virEventUpdateTimeoutSymbol != NULL);
    }
    if (!virEventUpdateTimeoutSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virEventUpdateTimeoutSymbol(timer,
                                timeout);

}
typedef void
(*virFreeErrorType)(virErrorPtr err);

void
virFreeErrorWrapper(virErrorPtr err) {
    static virFreeErrorType virFreeErrorSymbol;
    static bool virFreeErrorSymbolInit;

    if (!virFreeErrorSymbolInit) {
        libvirtLoad();
        virFreeErrorSymbol = libvirtSymbol(libvirt, "virFreeError");
        virFreeErrorSymbolInit = (virFreeErrorSymbol != NULL);
    }
    if (!virFreeErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virFreeErrorSymbol(err);

}
typedef virErrorPtr
(*virGetLastErrorType)(void);

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err) {
    static virGetLastErrorType virGetLastErrorSymbol;
    static bool virGetLastErrorSymbolInit;
    virErrorPtr ret;
    if (!virGetLastErrorSymbolInit) {
        libvirtLoad();
        virGetLastErrorSymbol = libvirtSymbol(libvirt, "virGetLastError");
        virGetLastErrorSymbolInit = (virGetLastErrorSymbol != NULL);
    }
    if (!virGetLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virGetLastErrorCodeWrapper(virErrorPtr err) {
    static virGetLastErrorCodeType virGetLastErrorCodeSymbol;
    static bool virGetLastErrorCodeSymbolInit;
    int ret;
    if (!virGetLastErrorCodeSymbolInit) {
        libvirtLoad();
        virGetLastErrorCodeSymbol = libvirtSymbol(libvirt, "virGetLastErrorCode");
        virGetLastErrorCodeSymbolInit = (virGetLastErrorCodeSymbol != NULL);
    }
    if (!virGetLastErrorCodeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virGetLastErrorDomainWrapper(virErrorPtr err) {
    static virGetLastErrorDomainType virGetLastErrorDomainSymbol;
    static bool virGetLastErrorDomainSymbolInit;
    int ret;
    if (!virGetLastErrorDomainSymbolInit) {
        libvirtLoad();
        virGetLastErrorDomainSymbol = libvirtSymbol(libvirt, "virGetLastErrorDomain");
        virGetLastErrorDomainSymbolInit = (virGetLastErrorDomainSymbol != NULL);
    }
    if (!virGetLastErrorDomainSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virGetLastErrorMessageWrapper(virErrorPtr err) {
    static virGetLastErrorMessageType virGetLastErrorMessageSymbol;
    static bool virGetLastErrorMessageSymbolInit;
    const char * ret;
    if (!virGetLastErrorMessageSymbolInit) {
        libvirtLoad();
        virGetLastErrorMessageSymbol = libvirtSymbol(libvirt, "virGetLastErrorMessage");
        virGetLastErrorMessageSymbolInit = (virGetLastErrorMessageSymbol != NULL);
    }
    if (!virGetLastErrorMessageSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virGetVersionType virGetVersionSymbol;
    static bool virGetVersionSymbolInit;
    int ret;
    if (!virGetVersionSymbolInit) {
        libvirtLoad();
        virGetVersionSymbol = libvirtSymbol(libvirt, "virGetVersion");
        virGetVersionSymbolInit = (virGetVersionSymbol != NULL);
    }
    if (!virGetVersionSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virInitializeWrapper(virErrorPtr err) {
    static virInitializeType virInitializeSymbol;
    static bool virInitializeSymbolInit;
    int ret;
    if (!virInitializeSymbolInit) {
        libvirtLoad();
        virInitializeSymbol = libvirtSymbol(libvirt, "virInitialize");
        virInitializeSymbolInit = (virInitializeSymbol != NULL);
    }
    if (!virInitializeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virInterfaceChangeBeginType virInterfaceChangeBeginSymbol;
    static bool virInterfaceChangeBeginSymbolInit;
    int ret;
    if (!virInterfaceChangeBeginSymbolInit) {
        libvirtLoad();
        virInterfaceChangeBeginSymbol = libvirtSymbol(libvirt, "virInterfaceChangeBegin");
        virInterfaceChangeBeginSymbolInit = (virInterfaceChangeBeginSymbol != NULL);
    }
    if (!virInterfaceChangeBeginSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virInterfaceChangeCommitType virInterfaceChangeCommitSymbol;
    static bool virInterfaceChangeCommitSymbolInit;
    int ret;
    if (!virInterfaceChangeCommitSymbolInit) {
        libvirtLoad();
        virInterfaceChangeCommitSymbol = libvirtSymbol(libvirt, "virInterfaceChangeCommit");
        virInterfaceChangeCommitSymbolInit = (virInterfaceChangeCommitSymbol != NULL);
    }
    if (!virInterfaceChangeCommitSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virInterfaceChangeRollbackType virInterfaceChangeRollbackSymbol;
    static bool virInterfaceChangeRollbackSymbolInit;
    int ret;
    if (!virInterfaceChangeRollbackSymbolInit) {
        libvirtLoad();
        virInterfaceChangeRollbackSymbol = libvirtSymbol(libvirt, "virInterfaceChangeRollback");
        virInterfaceChangeRollbackSymbolInit = (virInterfaceChangeRollbackSymbol != NULL);
    }
    if (!virInterfaceChangeRollbackSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virInterfaceCreateType virInterfaceCreateSymbol;
    static bool virInterfaceCreateSymbolInit;
    int ret;
    if (!virInterfaceCreateSymbolInit) {
        libvirtLoad();
        virInterfaceCreateSymbol = libvirtSymbol(libvirt, "virInterfaceCreate");
        virInterfaceCreateSymbolInit = (virInterfaceCreateSymbol != NULL);
    }
    if (!virInterfaceCreateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virInterfaceDefineXMLType virInterfaceDefineXMLSymbol;
    static bool virInterfaceDefineXMLSymbolInit;
    virInterfacePtr ret;
    if (!virInterfaceDefineXMLSymbolInit) {
        libvirtLoad();
        virInterfaceDefineXMLSymbol = libvirtSymbol(libvirt, "virInterfaceDefineXML");
        virInterfaceDefineXMLSymbolInit = (virInterfaceDefineXMLSymbol != NULL);
    }
    if (!virInterfaceDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virInterfaceDestroyType virInterfaceDestroySymbol;
    static bool virInterfaceDestroySymbolInit;
    int ret;
    if (!virInterfaceDestroySymbolInit) {
        libvirtLoad();
        virInterfaceDestroySymbol = libvirtSymbol(libvirt, "virInterfaceDestroy");
        virInterfaceDestroySymbolInit = (virInterfaceDestroySymbol != NULL);
    }
    if (!virInterfaceDestroySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virInterfaceFreeType virInterfaceFreeSymbol;
    static bool virInterfaceFreeSymbolInit;
    int ret;
    if (!virInterfaceFreeSymbolInit) {
        libvirtLoad();
        virInterfaceFreeSymbol = libvirtSymbol(libvirt, "virInterfaceFree");
        virInterfaceFreeSymbolInit = (virInterfaceFreeSymbol != NULL);
    }
    if (!virInterfaceFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virInterfaceGetConnectType virInterfaceGetConnectSymbol;
    static bool virInterfaceGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virInterfaceGetConnectSymbolInit) {
        libvirtLoad();
        virInterfaceGetConnectSymbol = libvirtSymbol(libvirt, "virInterfaceGetConnect");
        virInterfaceGetConnectSymbolInit = (virInterfaceGetConnectSymbol != NULL);
    }
    if (!virInterfaceGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virInterfaceGetMACStringType virInterfaceGetMACStringSymbol;
    static bool virInterfaceGetMACStringSymbolInit;
    const char * ret;
    if (!virInterfaceGetMACStringSymbolInit) {
        libvirtLoad();
        virInterfaceGetMACStringSymbol = libvirtSymbol(libvirt, "virInterfaceGetMACString");
        virInterfaceGetMACStringSymbolInit = (virInterfaceGetMACStringSymbol != NULL);
    }
    if (!virInterfaceGetMACStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virInterfaceGetNameType virInterfaceGetNameSymbol;
    static bool virInterfaceGetNameSymbolInit;
    const char * ret;
    if (!virInterfaceGetNameSymbolInit) {
        libvirtLoad();
        virInterfaceGetNameSymbol = libvirtSymbol(libvirt, "virInterfaceGetName");
        virInterfaceGetNameSymbolInit = (virInterfaceGetNameSymbol != NULL);
    }
    if (!virInterfaceGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virInterfaceGetXMLDescType virInterfaceGetXMLDescSymbol;
    static bool virInterfaceGetXMLDescSymbolInit;
    char * ret;
    if (!virInterfaceGetXMLDescSymbolInit) {
        libvirtLoad();
        virInterfaceGetXMLDescSymbol = libvirtSymbol(libvirt, "virInterfaceGetXMLDesc");
        virInterfaceGetXMLDescSymbolInit = (virInterfaceGetXMLDescSymbol != NULL);
    }
    if (!virInterfaceGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virInterfaceIsActiveType virInterfaceIsActiveSymbol;
    static bool virInterfaceIsActiveSymbolInit;
    int ret;
    if (!virInterfaceIsActiveSymbolInit) {
        libvirtLoad();
        virInterfaceIsActiveSymbol = libvirtSymbol(libvirt, "virInterfaceIsActive");
        virInterfaceIsActiveSymbolInit = (virInterfaceIsActiveSymbol != NULL);
    }
    if (!virInterfaceIsActiveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virInterfaceLookupByMACStringType virInterfaceLookupByMACStringSymbol;
    static bool virInterfaceLookupByMACStringSymbolInit;
    virInterfacePtr ret;
    if (!virInterfaceLookupByMACStringSymbolInit) {
        libvirtLoad();
        virInterfaceLookupByMACStringSymbol = libvirtSymbol(libvirt, "virInterfaceLookupByMACString");
        virInterfaceLookupByMACStringSymbolInit = (virInterfaceLookupByMACStringSymbol != NULL);
    }
    if (!virInterfaceLookupByMACStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virInterfaceLookupByNameType virInterfaceLookupByNameSymbol;
    static bool virInterfaceLookupByNameSymbolInit;
    virInterfacePtr ret;
    if (!virInterfaceLookupByNameSymbolInit) {
        libvirtLoad();
        virInterfaceLookupByNameSymbol = libvirtSymbol(libvirt, "virInterfaceLookupByName");
        virInterfaceLookupByNameSymbolInit = (virInterfaceLookupByNameSymbol != NULL);
    }
    if (!virInterfaceLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virInterfaceRefType virInterfaceRefSymbol;
    static bool virInterfaceRefSymbolInit;
    int ret;
    if (!virInterfaceRefSymbolInit) {
        libvirtLoad();
        virInterfaceRefSymbol = libvirtSymbol(libvirt, "virInterfaceRef");
        virInterfaceRefSymbolInit = (virInterfaceRefSymbol != NULL);
    }
    if (!virInterfaceRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virInterfaceUndefineType virInterfaceUndefineSymbol;
    static bool virInterfaceUndefineSymbolInit;
    int ret;
    if (!virInterfaceUndefineSymbolInit) {
        libvirtLoad();
        virInterfaceUndefineSymbol = libvirtSymbol(libvirt, "virInterfaceUndefine");
        virInterfaceUndefineSymbolInit = (virInterfaceUndefineSymbol != NULL);
    }
    if (!virInterfaceUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virNWFilterBindingCreateXMLType virNWFilterBindingCreateXMLSymbol;
    static bool virNWFilterBindingCreateXMLSymbolInit;
    virNWFilterBindingPtr ret;
    if (!virNWFilterBindingCreateXMLSymbolInit) {
        libvirtLoad();
        virNWFilterBindingCreateXMLSymbol = libvirtSymbol(libvirt, "virNWFilterBindingCreateXML");
        virNWFilterBindingCreateXMLSymbolInit = (virNWFilterBindingCreateXMLSymbol != NULL);
    }
    if (!virNWFilterBindingCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virNWFilterBindingDeleteType virNWFilterBindingDeleteSymbol;
    static bool virNWFilterBindingDeleteSymbolInit;
    int ret;
    if (!virNWFilterBindingDeleteSymbolInit) {
        libvirtLoad();
        virNWFilterBindingDeleteSymbol = libvirtSymbol(libvirt, "virNWFilterBindingDelete");
        virNWFilterBindingDeleteSymbolInit = (virNWFilterBindingDeleteSymbol != NULL);
    }
    if (!virNWFilterBindingDeleteSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNWFilterBindingFreeType virNWFilterBindingFreeSymbol;
    static bool virNWFilterBindingFreeSymbolInit;
    int ret;
    if (!virNWFilterBindingFreeSymbolInit) {
        libvirtLoad();
        virNWFilterBindingFreeSymbol = libvirtSymbol(libvirt, "virNWFilterBindingFree");
        virNWFilterBindingFreeSymbolInit = (virNWFilterBindingFreeSymbol != NULL);
    }
    if (!virNWFilterBindingFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                       virErrorPtr err) {
    static virNWFilterBindingGetFilterNameType virNWFilterBindingGetFilterNameSymbol;
    static bool virNWFilterBindingGetFilterNameSymbolInit;
    const char * ret;
    if (!virNWFilterBindingGetFilterNameSymbolInit) {
        libvirtLoad();
        virNWFilterBindingGetFilterNameSymbol = libvirtSymbol(libvirt, "virNWFilterBindingGetFilterName");
        virNWFilterBindingGetFilterNameSymbolInit = (virNWFilterBindingGetFilterNameSymbol != NULL);
    }
    if (!virNWFilterBindingGetFilterNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virNWFilterBindingGetPortDevType virNWFilterBindingGetPortDevSymbol;
    static bool virNWFilterBindingGetPortDevSymbolInit;
    const char * ret;
    if (!virNWFilterBindingGetPortDevSymbolInit) {
        libvirtLoad();
        virNWFilterBindingGetPortDevSymbol = libvirtSymbol(libvirt, "virNWFilterBindingGetPortDev");
        virNWFilterBindingGetPortDevSymbolInit = (virNWFilterBindingGetPortDevSymbol != NULL);
    }
    if (!virNWFilterBindingGetPortDevSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virNWFilterBindingGetXMLDescType virNWFilterBindingGetXMLDescSymbol;
    static bool virNWFilterBindingGetXMLDescSymbolInit;
    char * ret;
    if (!virNWFilterBindingGetXMLDescSymbolInit) {
        libvirtLoad();
        virNWFilterBindingGetXMLDescSymbol = libvirtSymbol(libvirt, "virNWFilterBindingGetXMLDesc");
        virNWFilterBindingGetXMLDescSymbolInit = (virNWFilterBindingGetXMLDescSymbol != NULL);
    }
    if (!virNWFilterBindingGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                         virErrorPtr err) {
    static virNWFilterBindingLookupByPortDevType virNWFilterBindingLookupByPortDevSymbol;
    static bool virNWFilterBindingLookupByPortDevSymbolInit;
    virNWFilterBindingPtr ret;
    if (!virNWFilterBindingLookupByPortDevSymbolInit) {
        libvirtLoad();
        virNWFilterBindingLookupByPortDevSymbol = libvirtSymbol(libvirt, "virNWFilterBindingLookupByPortDev");
        virNWFilterBindingLookupByPortDevSymbolInit = (virNWFilterBindingLookupByPortDevSymbol != NULL);
    }
    if (!virNWFilterBindingLookupByPortDevSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNWFilterBindingRefType virNWFilterBindingRefSymbol;
    static bool virNWFilterBindingRefSymbolInit;
    int ret;
    if (!virNWFilterBindingRefSymbolInit) {
        libvirtLoad();
        virNWFilterBindingRefSymbol = libvirtSymbol(libvirt, "virNWFilterBindingRef");
        virNWFilterBindingRefSymbolInit = (virNWFilterBindingRefSymbol != NULL);
    }
    if (!virNWFilterBindingRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNWFilterDefineXMLType virNWFilterDefineXMLSymbol;
    static bool virNWFilterDefineXMLSymbolInit;
    virNWFilterPtr ret;
    if (!virNWFilterDefineXMLSymbolInit) {
        libvirtLoad();
        virNWFilterDefineXMLSymbol = libvirtSymbol(libvirt, "virNWFilterDefineXML");
        virNWFilterDefineXMLSymbolInit = (virNWFilterDefineXMLSymbol != NULL);
    }
    if (!virNWFilterDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virNWFilterFreeType virNWFilterFreeSymbol;
    static bool virNWFilterFreeSymbolInit;
    int ret;
    if (!virNWFilterFreeSymbolInit) {
        libvirtLoad();
        virNWFilterFreeSymbol = libvirtSymbol(libvirt, "virNWFilterFree");
        virNWFilterFreeSymbolInit = (virNWFilterFreeSymbol != NULL);
    }
    if (!virNWFilterFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNWFilterGetNameType virNWFilterGetNameSymbol;
    static bool virNWFilterGetNameSymbolInit;
    const char * ret;
    if (!virNWFilterGetNameSymbolInit) {
        libvirtLoad();
        virNWFilterGetNameSymbol = libvirtSymbol(libvirt, "virNWFilterGetName");
        virNWFilterGetNameSymbolInit = (virNWFilterGetNameSymbol != NULL);
    }
    if (!virNWFilterGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNWFilterGetUUIDType virNWFilterGetUUIDSymbol;
    static bool virNWFilterGetUUIDSymbolInit;
    int ret;
    if (!virNWFilterGetUUIDSymbolInit) {
        libvirtLoad();
        virNWFilterGetUUIDSymbol = libvirtSymbol(libvirt, "virNWFilterGetUUID");
        virNWFilterGetUUIDSymbolInit = (virNWFilterGetUUIDSymbol != NULL);
    }
    if (!virNWFilterGetUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virNWFilterGetUUIDStringType virNWFilterGetUUIDStringSymbol;
    static bool virNWFilterGetUUIDStringSymbolInit;
    int ret;
    if (!virNWFilterGetUUIDStringSymbolInit) {
        libvirtLoad();
        virNWFilterGetUUIDStringSymbol = libvirtSymbol(libvirt, "virNWFilterGetUUIDString");
        virNWFilterGetUUIDStringSymbolInit = (virNWFilterGetUUIDStringSymbol != NULL);
    }
    if (!virNWFilterGetUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNWFilterGetXMLDescType virNWFilterGetXMLDescSymbol;
    static bool virNWFilterGetXMLDescSymbolInit;
    char * ret;
    if (!virNWFilterGetXMLDescSymbolInit) {
        libvirtLoad();
        virNWFilterGetXMLDescSymbol = libvirtSymbol(libvirt, "virNWFilterGetXMLDesc");
        virNWFilterGetXMLDescSymbolInit = (virNWFilterGetXMLDescSymbol != NULL);
    }
    if (!virNWFilterGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNWFilterLookupByNameType virNWFilterLookupByNameSymbol;
    static bool virNWFilterLookupByNameSymbolInit;
    virNWFilterPtr ret;
    if (!virNWFilterLookupByNameSymbolInit) {
        libvirtLoad();
        virNWFilterLookupByNameSymbol = libvirtSymbol(libvirt, "virNWFilterLookupByName");
        virNWFilterLookupByNameSymbolInit = (virNWFilterLookupByNameSymbol != NULL);
    }
    if (!virNWFilterLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNWFilterLookupByUUIDType virNWFilterLookupByUUIDSymbol;
    static bool virNWFilterLookupByUUIDSymbolInit;
    virNWFilterPtr ret;
    if (!virNWFilterLookupByUUIDSymbolInit) {
        libvirtLoad();
        virNWFilterLookupByUUIDSymbol = libvirtSymbol(libvirt, "virNWFilterLookupByUUID");
        virNWFilterLookupByUUIDSymbolInit = (virNWFilterLookupByUUIDSymbol != NULL);
    }
    if (!virNWFilterLookupByUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                     virErrorPtr err) {
    static virNWFilterLookupByUUIDStringType virNWFilterLookupByUUIDStringSymbol;
    static bool virNWFilterLookupByUUIDStringSymbolInit;
    virNWFilterPtr ret;
    if (!virNWFilterLookupByUUIDStringSymbolInit) {
        libvirtLoad();
        virNWFilterLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virNWFilterLookupByUUIDString");
        virNWFilterLookupByUUIDStringSymbolInit = (virNWFilterLookupByUUIDStringSymbol != NULL);
    }
    if (!virNWFilterLookupByUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virNWFilterRefType virNWFilterRefSymbol;
    static bool virNWFilterRefSymbolInit;
    int ret;
    if (!virNWFilterRefSymbolInit) {
        libvirtLoad();
        virNWFilterRefSymbol = libvirtSymbol(libvirt, "virNWFilterRef");
        virNWFilterRefSymbolInit = (virNWFilterRefSymbol != NULL);
    }
    if (!virNWFilterRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virNWFilterUndefineType virNWFilterUndefineSymbol;
    static bool virNWFilterUndefineSymbolInit;
    int ret;
    if (!virNWFilterUndefineSymbolInit) {
        libvirtLoad();
        virNWFilterUndefineSymbol = libvirtSymbol(libvirt, "virNWFilterUndefine");
        virNWFilterUndefineSymbolInit = (virNWFilterUndefineSymbol != NULL);
    }
    if (!virNWFilterUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virNetworkCreateType virNetworkCreateSymbol;
    static bool virNetworkCreateSymbolInit;
    int ret;
    if (!virNetworkCreateSymbolInit) {
        libvirtLoad();
        virNetworkCreateSymbol = libvirtSymbol(libvirt, "virNetworkCreate");
        virNetworkCreateSymbolInit = (virNetworkCreateSymbol != NULL);
    }
    if (!virNetworkCreateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virNetworkCreateXMLType virNetworkCreateXMLSymbol;
    static bool virNetworkCreateXMLSymbolInit;
    virNetworkPtr ret;
    if (!virNetworkCreateXMLSymbolInit) {
        libvirtLoad();
        virNetworkCreateXMLSymbol = libvirtSymbol(libvirt, "virNetworkCreateXML");
        virNetworkCreateXMLSymbolInit = (virNetworkCreateXMLSymbol != NULL);
    }
    if (!virNetworkCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease) {
    static virNetworkDHCPLeaseFreeType virNetworkDHCPLeaseFreeSymbol;
    static bool virNetworkDHCPLeaseFreeSymbolInit;

    if (!virNetworkDHCPLeaseFreeSymbolInit) {
        libvirtLoad();
        virNetworkDHCPLeaseFreeSymbol = libvirtSymbol(libvirt, "virNetworkDHCPLeaseFree");
        virNetworkDHCPLeaseFreeSymbolInit = (virNetworkDHCPLeaseFreeSymbol != NULL);
    }
    if (!virNetworkDHCPLeaseFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virNetworkDHCPLeaseFreeSymbol(lease);

}
typedef virNetworkPtr
(*virNetworkDefineXMLType)(virConnectPtr conn,
                           const char * xml);

virNetworkPtr
virNetworkDefineXMLWrapper(virConnectPtr conn,
                           const char * xml,
                           virErrorPtr err) {
    static virNetworkDefineXMLType virNetworkDefineXMLSymbol;
    static bool virNetworkDefineXMLSymbolInit;
    virNetworkPtr ret;
    if (!virNetworkDefineXMLSymbolInit) {
        libvirtLoad();
        virNetworkDefineXMLSymbol = libvirtSymbol(libvirt, "virNetworkDefineXML");
        virNetworkDefineXMLSymbolInit = (virNetworkDefineXMLSymbol != NULL);
    }
    if (!virNetworkDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNetworkDestroyType virNetworkDestroySymbol;
    static bool virNetworkDestroySymbolInit;
    int ret;
    if (!virNetworkDestroySymbolInit) {
        libvirtLoad();
        virNetworkDestroySymbol = libvirtSymbol(libvirt, "virNetworkDestroy");
        virNetworkDestroySymbolInit = (virNetworkDestroySymbol != NULL);
    }
    if (!virNetworkDestroySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virNetworkFreeType virNetworkFreeSymbol;
    static bool virNetworkFreeSymbolInit;
    int ret;
    if (!virNetworkFreeSymbolInit) {
        libvirtLoad();
        virNetworkFreeSymbol = libvirtSymbol(libvirt, "virNetworkFree");
        virNetworkFreeSymbolInit = (virNetworkFreeSymbol != NULL);
    }
    if (!virNetworkFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNetworkGetAutostartType virNetworkGetAutostartSymbol;
    static bool virNetworkGetAutostartSymbolInit;
    int ret;
    if (!virNetworkGetAutostartSymbolInit) {
        libvirtLoad();
        virNetworkGetAutostartSymbol = libvirtSymbol(libvirt, "virNetworkGetAutostart");
        virNetworkGetAutostartSymbolInit = (virNetworkGetAutostartSymbol != NULL);
    }
    if (!virNetworkGetAutostartSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNetworkGetBridgeNameType virNetworkGetBridgeNameSymbol;
    static bool virNetworkGetBridgeNameSymbolInit;
    char * ret;
    if (!virNetworkGetBridgeNameSymbolInit) {
        libvirtLoad();
        virNetworkGetBridgeNameSymbol = libvirtSymbol(libvirt, "virNetworkGetBridgeName");
        virNetworkGetBridgeNameSymbolInit = (virNetworkGetBridgeNameSymbol != NULL);
    }
    if (!virNetworkGetBridgeNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNetworkGetConnectType virNetworkGetConnectSymbol;
    static bool virNetworkGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virNetworkGetConnectSymbolInit) {
        libvirtLoad();
        virNetworkGetConnectSymbol = libvirtSymbol(libvirt, "virNetworkGetConnect");
        virNetworkGetConnectSymbolInit = (virNetworkGetConnectSymbol != NULL);
    }
    if (!virNetworkGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNetworkGetDHCPLeasesType virNetworkGetDHCPLeasesSymbol;
    static bool virNetworkGetDHCPLeasesSymbolInit;
    int ret;
    if (!virNetworkGetDHCPLeasesSymbolInit) {
        libvirtLoad();
        virNetworkGetDHCPLeasesSymbol = libvirtSymbol(libvirt, "virNetworkGetDHCPLeases");
        virNetworkGetDHCPLeasesSymbolInit = (virNetworkGetDHCPLeasesSymbol != NULL);
    }
    if (!virNetworkGetDHCPLeasesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNetworkGetNameType virNetworkGetNameSymbol;
    static bool virNetworkGetNameSymbolInit;
    const char * ret;
    if (!virNetworkGetNameSymbolInit) {
        libvirtLoad();
        virNetworkGetNameSymbol = libvirtSymbol(libvirt, "virNetworkGetName");
        virNetworkGetNameSymbolInit = (virNetworkGetNameSymbol != NULL);
    }
    if (!virNetworkGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNetworkGetUUIDType virNetworkGetUUIDSymbol;
    static bool virNetworkGetUUIDSymbolInit;
    int ret;
    if (!virNetworkGetUUIDSymbolInit) {
        libvirtLoad();
        virNetworkGetUUIDSymbol = libvirtSymbol(libvirt, "virNetworkGetUUID");
        virNetworkGetUUIDSymbolInit = (virNetworkGetUUIDSymbol != NULL);
    }
    if (!virNetworkGetUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNetworkGetUUIDStringType virNetworkGetUUIDStringSymbol;
    static bool virNetworkGetUUIDStringSymbolInit;
    int ret;
    if (!virNetworkGetUUIDStringSymbolInit) {
        libvirtLoad();
        virNetworkGetUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkGetUUIDString");
        virNetworkGetUUIDStringSymbolInit = (virNetworkGetUUIDStringSymbol != NULL);
    }
    if (!virNetworkGetUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNetworkGetXMLDescType virNetworkGetXMLDescSymbol;
    static bool virNetworkGetXMLDescSymbolInit;
    char * ret;
    if (!virNetworkGetXMLDescSymbolInit) {
        libvirtLoad();
        virNetworkGetXMLDescSymbol = libvirtSymbol(libvirt, "virNetworkGetXMLDesc");
        virNetworkGetXMLDescSymbolInit = (virNetworkGetXMLDescSymbol != NULL);
    }
    if (!virNetworkGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNetworkIsActiveType virNetworkIsActiveSymbol;
    static bool virNetworkIsActiveSymbolInit;
    int ret;
    if (!virNetworkIsActiveSymbolInit) {
        libvirtLoad();
        virNetworkIsActiveSymbol = libvirtSymbol(libvirt, "virNetworkIsActive");
        virNetworkIsActiveSymbolInit = (virNetworkIsActiveSymbol != NULL);
    }
    if (!virNetworkIsActiveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNetworkIsPersistentType virNetworkIsPersistentSymbol;
    static bool virNetworkIsPersistentSymbolInit;
    int ret;
    if (!virNetworkIsPersistentSymbolInit) {
        libvirtLoad();
        virNetworkIsPersistentSymbol = libvirtSymbol(libvirt, "virNetworkIsPersistent");
        virNetworkIsPersistentSymbolInit = (virNetworkIsPersistentSymbol != NULL);
    }
    if (!virNetworkIsPersistentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNetworkListAllPortsType virNetworkListAllPortsSymbol;
    static bool virNetworkListAllPortsSymbolInit;
    int ret;
    if (!virNetworkListAllPortsSymbolInit) {
        libvirtLoad();
        virNetworkListAllPortsSymbol = libvirtSymbol(libvirt, "virNetworkListAllPorts");
        virNetworkListAllPortsSymbolInit = (virNetworkListAllPortsSymbol != NULL);
    }
    if (!virNetworkListAllPortsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNetworkLookupByNameType virNetworkLookupByNameSymbol;
    static bool virNetworkLookupByNameSymbolInit;
    virNetworkPtr ret;
    if (!virNetworkLookupByNameSymbolInit) {
        libvirtLoad();
        virNetworkLookupByNameSymbol = libvirtSymbol(libvirt, "virNetworkLookupByName");
        virNetworkLookupByNameSymbolInit = (virNetworkLookupByNameSymbol != NULL);
    }
    if (!virNetworkLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNetworkLookupByUUIDType virNetworkLookupByUUIDSymbol;
    static bool virNetworkLookupByUUIDSymbolInit;
    virNetworkPtr ret;
    if (!virNetworkLookupByUUIDSymbolInit) {
        libvirtLoad();
        virNetworkLookupByUUIDSymbol = libvirtSymbol(libvirt, "virNetworkLookupByUUID");
        virNetworkLookupByUUIDSymbolInit = (virNetworkLookupByUUIDSymbol != NULL);
    }
    if (!virNetworkLookupByUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virNetworkLookupByUUIDStringType virNetworkLookupByUUIDStringSymbol;
    static bool virNetworkLookupByUUIDStringSymbolInit;
    virNetworkPtr ret;
    if (!virNetworkLookupByUUIDStringSymbolInit) {
        libvirtLoad();
        virNetworkLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkLookupByUUIDString");
        virNetworkLookupByUUIDStringSymbolInit = (virNetworkLookupByUUIDStringSymbol != NULL);
    }
    if (!virNetworkLookupByUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNetworkPortCreateXMLType virNetworkPortCreateXMLSymbol;
    static bool virNetworkPortCreateXMLSymbolInit;
    virNetworkPortPtr ret;
    if (!virNetworkPortCreateXMLSymbolInit) {
        libvirtLoad();
        virNetworkPortCreateXMLSymbol = libvirtSymbol(libvirt, "virNetworkPortCreateXML");
        virNetworkPortCreateXMLSymbolInit = (virNetworkPortCreateXMLSymbol != NULL);
    }
    if (!virNetworkPortCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNetworkPortDeleteType virNetworkPortDeleteSymbol;
    static bool virNetworkPortDeleteSymbolInit;
    int ret;
    if (!virNetworkPortDeleteSymbolInit) {
        libvirtLoad();
        virNetworkPortDeleteSymbol = libvirtSymbol(libvirt, "virNetworkPortDelete");
        virNetworkPortDeleteSymbolInit = (virNetworkPortDeleteSymbol != NULL);
    }
    if (!virNetworkPortDeleteSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNetworkPortFreeType virNetworkPortFreeSymbol;
    static bool virNetworkPortFreeSymbolInit;
    int ret;
    if (!virNetworkPortFreeSymbolInit) {
        libvirtLoad();
        virNetworkPortFreeSymbol = libvirtSymbol(libvirt, "virNetworkPortFree");
        virNetworkPortFreeSymbolInit = (virNetworkPortFreeSymbol != NULL);
    }
    if (!virNetworkPortFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virNetworkPortGetNetworkType virNetworkPortGetNetworkSymbol;
    static bool virNetworkPortGetNetworkSymbolInit;
    virNetworkPtr ret;
    if (!virNetworkPortGetNetworkSymbolInit) {
        libvirtLoad();
        virNetworkPortGetNetworkSymbol = libvirtSymbol(libvirt, "virNetworkPortGetNetwork");
        virNetworkPortGetNetworkSymbolInit = (virNetworkPortGetNetworkSymbol != NULL);
    }
    if (!virNetworkPortGetNetworkSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virNetworkPortGetParametersType virNetworkPortGetParametersSymbol;
    static bool virNetworkPortGetParametersSymbolInit;
    int ret;
    if (!virNetworkPortGetParametersSymbolInit) {
        libvirtLoad();
        virNetworkPortGetParametersSymbol = libvirtSymbol(libvirt, "virNetworkPortGetParameters");
        virNetworkPortGetParametersSymbolInit = (virNetworkPortGetParametersSymbol != NULL);
    }
    if (!virNetworkPortGetParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNetworkPortGetUUIDType virNetworkPortGetUUIDSymbol;
    static bool virNetworkPortGetUUIDSymbolInit;
    int ret;
    if (!virNetworkPortGetUUIDSymbolInit) {
        libvirtLoad();
        virNetworkPortGetUUIDSymbol = libvirtSymbol(libvirt, "virNetworkPortGetUUID");
        virNetworkPortGetUUIDSymbolInit = (virNetworkPortGetUUIDSymbol != NULL);
    }
    if (!virNetworkPortGetUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virNetworkPortGetUUIDStringType virNetworkPortGetUUIDStringSymbol;
    static bool virNetworkPortGetUUIDStringSymbolInit;
    int ret;
    if (!virNetworkPortGetUUIDStringSymbolInit) {
        libvirtLoad();
        virNetworkPortGetUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkPortGetUUIDString");
        virNetworkPortGetUUIDStringSymbolInit = (virNetworkPortGetUUIDStringSymbol != NULL);
    }
    if (!virNetworkPortGetUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virNetworkPortGetXMLDescType virNetworkPortGetXMLDescSymbol;
    static bool virNetworkPortGetXMLDescSymbolInit;
    char * ret;
    if (!virNetworkPortGetXMLDescSymbolInit) {
        libvirtLoad();
        virNetworkPortGetXMLDescSymbol = libvirtSymbol(libvirt, "virNetworkPortGetXMLDesc");
        virNetworkPortGetXMLDescSymbolInit = (virNetworkPortGetXMLDescSymbol != NULL);
    }
    if (!virNetworkPortGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virNetworkPortLookupByUUIDType virNetworkPortLookupByUUIDSymbol;
    static bool virNetworkPortLookupByUUIDSymbolInit;
    virNetworkPortPtr ret;
    if (!virNetworkPortLookupByUUIDSymbolInit) {
        libvirtLoad();
        virNetworkPortLookupByUUIDSymbol = libvirtSymbol(libvirt, "virNetworkPortLookupByUUID");
        virNetworkPortLookupByUUIDSymbolInit = (virNetworkPortLookupByUUIDSymbol != NULL);
    }
    if (!virNetworkPortLookupByUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virNetworkPortLookupByUUIDStringType virNetworkPortLookupByUUIDStringSymbol;
    static bool virNetworkPortLookupByUUIDStringSymbolInit;
    virNetworkPortPtr ret;
    if (!virNetworkPortLookupByUUIDStringSymbolInit) {
        libvirtLoad();
        virNetworkPortLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virNetworkPortLookupByUUIDString");
        virNetworkPortLookupByUUIDStringSymbolInit = (virNetworkPortLookupByUUIDStringSymbol != NULL);
    }
    if (!virNetworkPortLookupByUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNetworkPortRefType virNetworkPortRefSymbol;
    static bool virNetworkPortRefSymbolInit;
    int ret;
    if (!virNetworkPortRefSymbolInit) {
        libvirtLoad();
        virNetworkPortRefSymbol = libvirtSymbol(libvirt, "virNetworkPortRef");
        virNetworkPortRefSymbolInit = (virNetworkPortRefSymbol != NULL);
    }
    if (!virNetworkPortRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virNetworkPortSetParametersType virNetworkPortSetParametersSymbol;
    static bool virNetworkPortSetParametersSymbolInit;
    int ret;
    if (!virNetworkPortSetParametersSymbolInit) {
        libvirtLoad();
        virNetworkPortSetParametersSymbol = libvirtSymbol(libvirt, "virNetworkPortSetParameters");
        virNetworkPortSetParametersSymbolInit = (virNetworkPortSetParametersSymbol != NULL);
    }
    if (!virNetworkPortSetParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virNetworkRefType virNetworkRefSymbol;
    static bool virNetworkRefSymbolInit;
    int ret;
    if (!virNetworkRefSymbolInit) {
        libvirtLoad();
        virNetworkRefSymbol = libvirtSymbol(libvirt, "virNetworkRef");
        virNetworkRefSymbolInit = (virNetworkRefSymbol != NULL);
    }
    if (!virNetworkRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNetworkSetAutostartType virNetworkSetAutostartSymbol;
    static bool virNetworkSetAutostartSymbolInit;
    int ret;
    if (!virNetworkSetAutostartSymbolInit) {
        libvirtLoad();
        virNetworkSetAutostartSymbol = libvirtSymbol(libvirt, "virNetworkSetAutostart");
        virNetworkSetAutostartSymbolInit = (virNetworkSetAutostartSymbol != NULL);
    }
    if (!virNetworkSetAutostartSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNetworkUndefineType virNetworkUndefineSymbol;
    static bool virNetworkUndefineSymbolInit;
    int ret;
    if (!virNetworkUndefineSymbolInit) {
        libvirtLoad();
        virNetworkUndefineSymbol = libvirtSymbol(libvirt, "virNetworkUndefine");
        virNetworkUndefineSymbolInit = (virNetworkUndefineSymbol != NULL);
    }
    if (!virNetworkUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virNetworkUpdateType virNetworkUpdateSymbol;
    static bool virNetworkUpdateSymbolInit;
    int ret;
    if (!virNetworkUpdateSymbolInit) {
        libvirtLoad();
        virNetworkUpdateSymbol = libvirtSymbol(libvirt, "virNetworkUpdate");
        virNetworkUpdateSymbolInit = (virNetworkUpdateSymbol != NULL);
    }
    if (!virNetworkUpdateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNodeAllocPagesType virNodeAllocPagesSymbol;
    static bool virNodeAllocPagesSymbolInit;
    int ret;
    if (!virNodeAllocPagesSymbolInit) {
        libvirtLoad();
        virNodeAllocPagesSymbol = libvirtSymbol(libvirt, "virNodeAllocPages");
        virNodeAllocPagesSymbolInit = (virNodeAllocPagesSymbol != NULL);
    }
    if (!virNodeAllocPagesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virNodeDeviceCreateType virNodeDeviceCreateSymbol;
    static bool virNodeDeviceCreateSymbolInit;
    int ret;
    if (!virNodeDeviceCreateSymbolInit) {
        libvirtLoad();
        virNodeDeviceCreateSymbol = libvirtSymbol(libvirt, "virNodeDeviceCreate");
        virNodeDeviceCreateSymbolInit = (virNodeDeviceCreateSymbol != NULL);
    }
    if (!virNodeDeviceCreateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNodeDeviceCreateXMLType virNodeDeviceCreateXMLSymbol;
    static bool virNodeDeviceCreateXMLSymbolInit;
    virNodeDevicePtr ret;
    if (!virNodeDeviceCreateXMLSymbolInit) {
        libvirtLoad();
        virNodeDeviceCreateXMLSymbol = libvirtSymbol(libvirt, "virNodeDeviceCreateXML");
        virNodeDeviceCreateXMLSymbolInit = (virNodeDeviceCreateXMLSymbol != NULL);
    }
    if (!virNodeDeviceCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNodeDeviceDefineXMLType virNodeDeviceDefineXMLSymbol;
    static bool virNodeDeviceDefineXMLSymbolInit;
    virNodeDevicePtr ret;
    if (!virNodeDeviceDefineXMLSymbolInit) {
        libvirtLoad();
        virNodeDeviceDefineXMLSymbol = libvirtSymbol(libvirt, "virNodeDeviceDefineXML");
        virNodeDeviceDefineXMLSymbolInit = (virNodeDeviceDefineXMLSymbol != NULL);
    }
    if (!virNodeDeviceDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNodeDeviceDestroyType virNodeDeviceDestroySymbol;
    static bool virNodeDeviceDestroySymbolInit;
    int ret;
    if (!virNodeDeviceDestroySymbolInit) {
        libvirtLoad();
        virNodeDeviceDestroySymbol = libvirtSymbol(libvirt, "virNodeDeviceDestroy");
        virNodeDeviceDestroySymbolInit = (virNodeDeviceDestroySymbol != NULL);
    }
    if (!virNodeDeviceDestroySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virNodeDeviceDetachFlagsType virNodeDeviceDetachFlagsSymbol;
    static bool virNodeDeviceDetachFlagsSymbolInit;
    int ret;
    if (!virNodeDeviceDetachFlagsSymbolInit) {
        libvirtLoad();
        virNodeDeviceDetachFlagsSymbol = libvirtSymbol(libvirt, "virNodeDeviceDetachFlags");
        virNodeDeviceDetachFlagsSymbolInit = (virNodeDeviceDetachFlagsSymbol != NULL);
    }
    if (!virNodeDeviceDetachFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNodeDeviceDettachType virNodeDeviceDettachSymbol;
    static bool virNodeDeviceDettachSymbolInit;
    int ret;
    if (!virNodeDeviceDettachSymbolInit) {
        libvirtLoad();
        virNodeDeviceDettachSymbol = libvirtSymbol(libvirt, "virNodeDeviceDettach");
        virNodeDeviceDettachSymbolInit = (virNodeDeviceDettachSymbol != NULL);
    }
    if (!virNodeDeviceDettachSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNodeDeviceFreeType virNodeDeviceFreeSymbol;
    static bool virNodeDeviceFreeSymbolInit;
    int ret;
    if (!virNodeDeviceFreeSymbolInit) {
        libvirtLoad();
        virNodeDeviceFreeSymbol = libvirtSymbol(libvirt, "virNodeDeviceFree");
        virNodeDeviceFreeSymbolInit = (virNodeDeviceFreeSymbol != NULL);
    }
    if (!virNodeDeviceFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNodeDeviceGetNameType virNodeDeviceGetNameSymbol;
    static bool virNodeDeviceGetNameSymbolInit;
    const char * ret;
    if (!virNodeDeviceGetNameSymbolInit) {
        libvirtLoad();
        virNodeDeviceGetNameSymbol = libvirtSymbol(libvirt, "virNodeDeviceGetName");
        virNodeDeviceGetNameSymbolInit = (virNodeDeviceGetNameSymbol != NULL);
    }
    if (!virNodeDeviceGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNodeDeviceGetParentType virNodeDeviceGetParentSymbol;
    static bool virNodeDeviceGetParentSymbolInit;
    const char * ret;
    if (!virNodeDeviceGetParentSymbolInit) {
        libvirtLoad();
        virNodeDeviceGetParentSymbol = libvirtSymbol(libvirt, "virNodeDeviceGetParent");
        virNodeDeviceGetParentSymbolInit = (virNodeDeviceGetParentSymbol != NULL);
    }
    if (!virNodeDeviceGetParentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNodeDeviceGetXMLDescType virNodeDeviceGetXMLDescSymbol;
    static bool virNodeDeviceGetXMLDescSymbolInit;
    char * ret;
    if (!virNodeDeviceGetXMLDescSymbolInit) {
        libvirtLoad();
        virNodeDeviceGetXMLDescSymbol = libvirtSymbol(libvirt, "virNodeDeviceGetXMLDesc");
        virNodeDeviceGetXMLDescSymbolInit = (virNodeDeviceGetXMLDescSymbol != NULL);
    }
    if (!virNodeDeviceGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNodeDeviceListCapsType virNodeDeviceListCapsSymbol;
    static bool virNodeDeviceListCapsSymbolInit;
    int ret;
    if (!virNodeDeviceListCapsSymbolInit) {
        libvirtLoad();
        virNodeDeviceListCapsSymbol = libvirtSymbol(libvirt, "virNodeDeviceListCaps");
        virNodeDeviceListCapsSymbolInit = (virNodeDeviceListCapsSymbol != NULL);
    }
    if (!virNodeDeviceListCapsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virNodeDeviceLookupByNameType virNodeDeviceLookupByNameSymbol;
    static bool virNodeDeviceLookupByNameSymbolInit;
    virNodeDevicePtr ret;
    if (!virNodeDeviceLookupByNameSymbolInit) {
        libvirtLoad();
        virNodeDeviceLookupByNameSymbol = libvirtSymbol(libvirt, "virNodeDeviceLookupByName");
        virNodeDeviceLookupByNameSymbolInit = (virNodeDeviceLookupByNameSymbol != NULL);
    }
    if (!virNodeDeviceLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virNodeDeviceLookupSCSIHostByWWNType virNodeDeviceLookupSCSIHostByWWNSymbol;
    static bool virNodeDeviceLookupSCSIHostByWWNSymbolInit;
    virNodeDevicePtr ret;
    if (!virNodeDeviceLookupSCSIHostByWWNSymbolInit) {
        libvirtLoad();
        virNodeDeviceLookupSCSIHostByWWNSymbol = libvirtSymbol(libvirt, "virNodeDeviceLookupSCSIHostByWWN");
        virNodeDeviceLookupSCSIHostByWWNSymbolInit = (virNodeDeviceLookupSCSIHostByWWNSymbol != NULL);
    }
    if (!virNodeDeviceLookupSCSIHostByWWNSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virNodeDeviceNumOfCapsType virNodeDeviceNumOfCapsSymbol;
    static bool virNodeDeviceNumOfCapsSymbolInit;
    int ret;
    if (!virNodeDeviceNumOfCapsSymbolInit) {
        libvirtLoad();
        virNodeDeviceNumOfCapsSymbol = libvirtSymbol(libvirt, "virNodeDeviceNumOfCaps");
        virNodeDeviceNumOfCapsSymbolInit = (virNodeDeviceNumOfCapsSymbol != NULL);
    }
    if (!virNodeDeviceNumOfCapsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNodeDeviceReAttachType virNodeDeviceReAttachSymbol;
    static bool virNodeDeviceReAttachSymbolInit;
    int ret;
    if (!virNodeDeviceReAttachSymbolInit) {
        libvirtLoad();
        virNodeDeviceReAttachSymbol = libvirtSymbol(libvirt, "virNodeDeviceReAttach");
        virNodeDeviceReAttachSymbolInit = (virNodeDeviceReAttachSymbol != NULL);
    }
    if (!virNodeDeviceReAttachSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virNodeDeviceRefType virNodeDeviceRefSymbol;
    static bool virNodeDeviceRefSymbolInit;
    int ret;
    if (!virNodeDeviceRefSymbolInit) {
        libvirtLoad();
        virNodeDeviceRefSymbol = libvirtSymbol(libvirt, "virNodeDeviceRef");
        virNodeDeviceRefSymbolInit = (virNodeDeviceRefSymbol != NULL);
    }
    if (!virNodeDeviceRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNodeDeviceResetType virNodeDeviceResetSymbol;
    static bool virNodeDeviceResetSymbolInit;
    int ret;
    if (!virNodeDeviceResetSymbolInit) {
        libvirtLoad();
        virNodeDeviceResetSymbol = libvirtSymbol(libvirt, "virNodeDeviceReset");
        virNodeDeviceResetSymbolInit = (virNodeDeviceResetSymbol != NULL);
    }
    if (!virNodeDeviceResetSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNodeDeviceUndefineType virNodeDeviceUndefineSymbol;
    static bool virNodeDeviceUndefineSymbolInit;
    int ret;
    if (!virNodeDeviceUndefineSymbolInit) {
        libvirtLoad();
        virNodeDeviceUndefineSymbol = libvirtSymbol(libvirt, "virNodeDeviceUndefine");
        virNodeDeviceUndefineSymbolInit = (virNodeDeviceUndefineSymbol != NULL);
    }
    if (!virNodeDeviceUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virNodeGetCPUMapType virNodeGetCPUMapSymbol;
    static bool virNodeGetCPUMapSymbolInit;
    int ret;
    if (!virNodeGetCPUMapSymbolInit) {
        libvirtLoad();
        virNodeGetCPUMapSymbol = libvirtSymbol(libvirt, "virNodeGetCPUMap");
        virNodeGetCPUMapSymbolInit = (virNodeGetCPUMapSymbol != NULL);
    }
    if (!virNodeGetCPUMapSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNodeGetCPUStatsType virNodeGetCPUStatsSymbol;
    static bool virNodeGetCPUStatsSymbolInit;
    int ret;
    if (!virNodeGetCPUStatsSymbolInit) {
        libvirtLoad();
        virNodeGetCPUStatsSymbol = libvirtSymbol(libvirt, "virNodeGetCPUStats");
        virNodeGetCPUStatsSymbolInit = (virNodeGetCPUStatsSymbol != NULL);
    }
    if (!virNodeGetCPUStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virNodeGetCellsFreeMemoryType virNodeGetCellsFreeMemorySymbol;
    static bool virNodeGetCellsFreeMemorySymbolInit;
    int ret;
    if (!virNodeGetCellsFreeMemorySymbolInit) {
        libvirtLoad();
        virNodeGetCellsFreeMemorySymbol = libvirtSymbol(libvirt, "virNodeGetCellsFreeMemory");
        virNodeGetCellsFreeMemorySymbolInit = (virNodeGetCellsFreeMemorySymbol != NULL);
    }
    if (!virNodeGetCellsFreeMemorySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virNodeGetFreeMemoryType virNodeGetFreeMemorySymbol;
    static bool virNodeGetFreeMemorySymbolInit;
    unsigned long long ret;
    if (!virNodeGetFreeMemorySymbolInit) {
        libvirtLoad();
        virNodeGetFreeMemorySymbol = libvirtSymbol(libvirt, "virNodeGetFreeMemory");
        virNodeGetFreeMemorySymbolInit = (virNodeGetFreeMemorySymbol != NULL);
    }
    if (!virNodeGetFreeMemorySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virNodeGetFreePagesType virNodeGetFreePagesSymbol;
    static bool virNodeGetFreePagesSymbolInit;
    int ret;
    if (!virNodeGetFreePagesSymbolInit) {
        libvirtLoad();
        virNodeGetFreePagesSymbol = libvirtSymbol(libvirt, "virNodeGetFreePages");
        virNodeGetFreePagesSymbolInit = (virNodeGetFreePagesSymbol != NULL);
    }
    if (!virNodeGetFreePagesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virNodeGetInfoType virNodeGetInfoSymbol;
    static bool virNodeGetInfoSymbolInit;
    int ret;
    if (!virNodeGetInfoSymbolInit) {
        libvirtLoad();
        virNodeGetInfoSymbol = libvirtSymbol(libvirt, "virNodeGetInfo");
        virNodeGetInfoSymbolInit = (virNodeGetInfoSymbol != NULL);
    }
    if (!virNodeGetInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virNodeGetMemoryParametersType virNodeGetMemoryParametersSymbol;
    static bool virNodeGetMemoryParametersSymbolInit;
    int ret;
    if (!virNodeGetMemoryParametersSymbolInit) {
        libvirtLoad();
        virNodeGetMemoryParametersSymbol = libvirtSymbol(libvirt, "virNodeGetMemoryParameters");
        virNodeGetMemoryParametersSymbolInit = (virNodeGetMemoryParametersSymbol != NULL);
    }
    if (!virNodeGetMemoryParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virNodeGetMemoryStatsType virNodeGetMemoryStatsSymbol;
    static bool virNodeGetMemoryStatsSymbolInit;
    int ret;
    if (!virNodeGetMemoryStatsSymbolInit) {
        libvirtLoad();
        virNodeGetMemoryStatsSymbol = libvirtSymbol(libvirt, "virNodeGetMemoryStats");
        virNodeGetMemoryStatsSymbolInit = (virNodeGetMemoryStatsSymbol != NULL);
    }
    if (!virNodeGetMemoryStatsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virNodeGetSEVInfoType virNodeGetSEVInfoSymbol;
    static bool virNodeGetSEVInfoSymbolInit;
    int ret;
    if (!virNodeGetSEVInfoSymbolInit) {
        libvirtLoad();
        virNodeGetSEVInfoSymbol = libvirtSymbol(libvirt, "virNodeGetSEVInfo");
        virNodeGetSEVInfoSymbolInit = (virNodeGetSEVInfoSymbol != NULL);
    }
    if (!virNodeGetSEVInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virNodeGetSecurityModelType virNodeGetSecurityModelSymbol;
    static bool virNodeGetSecurityModelSymbolInit;
    int ret;
    if (!virNodeGetSecurityModelSymbolInit) {
        libvirtLoad();
        virNodeGetSecurityModelSymbol = libvirtSymbol(libvirt, "virNodeGetSecurityModel");
        virNodeGetSecurityModelSymbolInit = (virNodeGetSecurityModelSymbol != NULL);
    }
    if (!virNodeGetSecurityModelSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virNodeListDevicesType virNodeListDevicesSymbol;
    static bool virNodeListDevicesSymbolInit;
    int ret;
    if (!virNodeListDevicesSymbolInit) {
        libvirtLoad();
        virNodeListDevicesSymbol = libvirtSymbol(libvirt, "virNodeListDevices");
        virNodeListDevicesSymbolInit = (virNodeListDevicesSymbol != NULL);
    }
    if (!virNodeListDevicesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virNodeNumOfDevicesType virNodeNumOfDevicesSymbol;
    static bool virNodeNumOfDevicesSymbolInit;
    int ret;
    if (!virNodeNumOfDevicesSymbolInit) {
        libvirtLoad();
        virNodeNumOfDevicesSymbol = libvirtSymbol(libvirt, "virNodeNumOfDevices");
        virNodeNumOfDevicesSymbolInit = (virNodeNumOfDevicesSymbol != NULL);
    }
    if (!virNodeNumOfDevicesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virNodeSetMemoryParametersType virNodeSetMemoryParametersSymbol;
    static bool virNodeSetMemoryParametersSymbolInit;
    int ret;
    if (!virNodeSetMemoryParametersSymbolInit) {
        libvirtLoad();
        virNodeSetMemoryParametersSymbol = libvirtSymbol(libvirt, "virNodeSetMemoryParameters");
        virNodeSetMemoryParametersSymbolInit = (virNodeSetMemoryParametersSymbol != NULL);
    }
    if (!virNodeSetMemoryParametersSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virNodeSuspendForDurationType virNodeSuspendForDurationSymbol;
    static bool virNodeSuspendForDurationSymbolInit;
    int ret;
    if (!virNodeSuspendForDurationSymbolInit) {
        libvirtLoad();
        virNodeSuspendForDurationSymbol = libvirtSymbol(libvirt, "virNodeSuspendForDuration");
        virNodeSuspendForDurationSymbolInit = (virNodeSuspendForDurationSymbol != NULL);
    }
    if (!virNodeSuspendForDurationSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
virResetErrorWrapper(virErrorPtr err) {
    static virResetErrorType virResetErrorSymbol;
    static bool virResetErrorSymbolInit;

    if (!virResetErrorSymbolInit) {
        libvirtLoad();
        virResetErrorSymbol = libvirtSymbol(libvirt, "virResetError");
        virResetErrorSymbolInit = (virResetErrorSymbol != NULL);
    }
    if (!virResetErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virResetErrorSymbol(err);

}
typedef void
(*virResetLastErrorType)(void);

void
virResetLastErrorWrapper(void) {
    static virResetLastErrorType virResetLastErrorSymbol;
    static bool virResetLastErrorSymbolInit;

    if (!virResetLastErrorSymbolInit) {
        libvirtLoad();
        virResetLastErrorSymbol = libvirtSymbol(libvirt, "virResetLastError");
        virResetLastErrorSymbolInit = (virResetLastErrorSymbol != NULL);
    }
    if (!virResetLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virResetLastErrorSymbol();

}
typedef virErrorPtr
(*virSaveLastErrorType)(void);

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err) {
    static virSaveLastErrorType virSaveLastErrorSymbol;
    static bool virSaveLastErrorSymbolInit;
    virErrorPtr ret;
    if (!virSaveLastErrorSymbolInit) {
        libvirtLoad();
        virSaveLastErrorSymbol = libvirtSymbol(libvirt, "virSaveLastError");
        virSaveLastErrorSymbolInit = (virSaveLastErrorSymbol != NULL);
    }
    if (!virSaveLastErrorSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virSecretDefineXMLType virSecretDefineXMLSymbol;
    static bool virSecretDefineXMLSymbolInit;
    virSecretPtr ret;
    if (!virSecretDefineXMLSymbolInit) {
        libvirtLoad();
        virSecretDefineXMLSymbol = libvirtSymbol(libvirt, "virSecretDefineXML");
        virSecretDefineXMLSymbolInit = (virSecretDefineXMLSymbol != NULL);
    }
    if (!virSecretDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virSecretFreeType virSecretFreeSymbol;
    static bool virSecretFreeSymbolInit;
    int ret;
    if (!virSecretFreeSymbolInit) {
        libvirtLoad();
        virSecretFreeSymbol = libvirtSymbol(libvirt, "virSecretFree");
        virSecretFreeSymbolInit = (virSecretFreeSymbol != NULL);
    }
    if (!virSecretFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virSecretGetConnectType virSecretGetConnectSymbol;
    static bool virSecretGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virSecretGetConnectSymbolInit) {
        libvirtLoad();
        virSecretGetConnectSymbol = libvirtSymbol(libvirt, "virSecretGetConnect");
        virSecretGetConnectSymbolInit = (virSecretGetConnectSymbol != NULL);
    }
    if (!virSecretGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virSecretGetUUIDType virSecretGetUUIDSymbol;
    static bool virSecretGetUUIDSymbolInit;
    int ret;
    if (!virSecretGetUUIDSymbolInit) {
        libvirtLoad();
        virSecretGetUUIDSymbol = libvirtSymbol(libvirt, "virSecretGetUUID");
        virSecretGetUUIDSymbolInit = (virSecretGetUUIDSymbol != NULL);
    }
    if (!virSecretGetUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virSecretGetUUIDStringType virSecretGetUUIDStringSymbol;
    static bool virSecretGetUUIDStringSymbolInit;
    int ret;
    if (!virSecretGetUUIDStringSymbolInit) {
        libvirtLoad();
        virSecretGetUUIDStringSymbol = libvirtSymbol(libvirt, "virSecretGetUUIDString");
        virSecretGetUUIDStringSymbolInit = (virSecretGetUUIDStringSymbol != NULL);
    }
    if (!virSecretGetUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virSecretGetUsageIDType virSecretGetUsageIDSymbol;
    static bool virSecretGetUsageIDSymbolInit;
    const char * ret;
    if (!virSecretGetUsageIDSymbolInit) {
        libvirtLoad();
        virSecretGetUsageIDSymbol = libvirtSymbol(libvirt, "virSecretGetUsageID");
        virSecretGetUsageIDSymbolInit = (virSecretGetUsageIDSymbol != NULL);
    }
    if (!virSecretGetUsageIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virSecretGetUsageTypeType virSecretGetUsageTypeSymbol;
    static bool virSecretGetUsageTypeSymbolInit;
    int ret;
    if (!virSecretGetUsageTypeSymbolInit) {
        libvirtLoad();
        virSecretGetUsageTypeSymbol = libvirtSymbol(libvirt, "virSecretGetUsageType");
        virSecretGetUsageTypeSymbolInit = (virSecretGetUsageTypeSymbol != NULL);
    }
    if (!virSecretGetUsageTypeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virSecretGetValueType virSecretGetValueSymbol;
    static bool virSecretGetValueSymbolInit;
    unsigned char * ret;
    if (!virSecretGetValueSymbolInit) {
        libvirtLoad();
        virSecretGetValueSymbol = libvirtSymbol(libvirt, "virSecretGetValue");
        virSecretGetValueSymbolInit = (virSecretGetValueSymbol != NULL);
    }
    if (!virSecretGetValueSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virSecretGetXMLDescType virSecretGetXMLDescSymbol;
    static bool virSecretGetXMLDescSymbolInit;
    char * ret;
    if (!virSecretGetXMLDescSymbolInit) {
        libvirtLoad();
        virSecretGetXMLDescSymbol = libvirtSymbol(libvirt, "virSecretGetXMLDesc");
        virSecretGetXMLDescSymbolInit = (virSecretGetXMLDescSymbol != NULL);
    }
    if (!virSecretGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virSecretLookupByUUIDType virSecretLookupByUUIDSymbol;
    static bool virSecretLookupByUUIDSymbolInit;
    virSecretPtr ret;
    if (!virSecretLookupByUUIDSymbolInit) {
        libvirtLoad();
        virSecretLookupByUUIDSymbol = libvirtSymbol(libvirt, "virSecretLookupByUUID");
        virSecretLookupByUUIDSymbolInit = (virSecretLookupByUUIDSymbol != NULL);
    }
    if (!virSecretLookupByUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virSecretLookupByUUIDStringType virSecretLookupByUUIDStringSymbol;
    static bool virSecretLookupByUUIDStringSymbolInit;
    virSecretPtr ret;
    if (!virSecretLookupByUUIDStringSymbolInit) {
        libvirtLoad();
        virSecretLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virSecretLookupByUUIDString");
        virSecretLookupByUUIDStringSymbolInit = (virSecretLookupByUUIDStringSymbol != NULL);
    }
    if (!virSecretLookupByUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virSecretLookupByUsageType virSecretLookupByUsageSymbol;
    static bool virSecretLookupByUsageSymbolInit;
    virSecretPtr ret;
    if (!virSecretLookupByUsageSymbolInit) {
        libvirtLoad();
        virSecretLookupByUsageSymbol = libvirtSymbol(libvirt, "virSecretLookupByUsage");
        virSecretLookupByUsageSymbolInit = (virSecretLookupByUsageSymbol != NULL);
    }
    if (!virSecretLookupByUsageSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                    virErrorPtr err) {
    static virSecretRefType virSecretRefSymbol;
    static bool virSecretRefSymbolInit;
    int ret;
    if (!virSecretRefSymbolInit) {
        libvirtLoad();
        virSecretRefSymbol = libvirtSymbol(libvirt, "virSecretRef");
        virSecretRefSymbolInit = (virSecretRefSymbol != NULL);
    }
    if (!virSecretRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virSecretSetValueType virSecretSetValueSymbol;
    static bool virSecretSetValueSymbolInit;
    int ret;
    if (!virSecretSetValueSymbolInit) {
        libvirtLoad();
        virSecretSetValueSymbol = libvirtSymbol(libvirt, "virSecretSetValue");
        virSecretSetValueSymbolInit = (virSecretSetValueSymbol != NULL);
    }
    if (!virSecretSetValueSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virSecretUndefineType virSecretUndefineSymbol;
    static bool virSecretUndefineSymbolInit;
    int ret;
    if (!virSecretUndefineSymbolInit) {
        libvirtLoad();
        virSecretUndefineSymbol = libvirtSymbol(libvirt, "virSecretUndefine");
        virSecretUndefineSymbolInit = (virSecretUndefineSymbol != NULL);
    }
    if (!virSecretUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorFunc handler) {
    static virSetErrorFuncType virSetErrorFuncSymbol;
    static bool virSetErrorFuncSymbolInit;

    if (!virSetErrorFuncSymbolInit) {
        libvirtLoad();
        virSetErrorFuncSymbol = libvirtSymbol(libvirt, "virSetErrorFunc");
        virSetErrorFuncSymbolInit = (virSetErrorFuncSymbol != NULL);
    }
    if (!virSetErrorFuncSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virStoragePoolBuildType virStoragePoolBuildSymbol;
    static bool virStoragePoolBuildSymbolInit;
    int ret;
    if (!virStoragePoolBuildSymbolInit) {
        libvirtLoad();
        virStoragePoolBuildSymbol = libvirtSymbol(libvirt, "virStoragePoolBuild");
        virStoragePoolBuildSymbolInit = (virStoragePoolBuildSymbol != NULL);
    }
    if (!virStoragePoolBuildSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virStoragePoolCreateType virStoragePoolCreateSymbol;
    static bool virStoragePoolCreateSymbolInit;
    int ret;
    if (!virStoragePoolCreateSymbolInit) {
        libvirtLoad();
        virStoragePoolCreateSymbol = libvirtSymbol(libvirt, "virStoragePoolCreate");
        virStoragePoolCreateSymbolInit = (virStoragePoolCreateSymbol != NULL);
    }
    if (!virStoragePoolCreateSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virStoragePoolCreateXMLType virStoragePoolCreateXMLSymbol;
    static bool virStoragePoolCreateXMLSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolCreateXMLSymbolInit) {
        libvirtLoad();
        virStoragePoolCreateXMLSymbol = libvirtSymbol(libvirt, "virStoragePoolCreateXML");
        virStoragePoolCreateXMLSymbolInit = (virStoragePoolCreateXMLSymbol != NULL);
    }
    if (!virStoragePoolCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virStoragePoolDefineXMLType virStoragePoolDefineXMLSymbol;
    static bool virStoragePoolDefineXMLSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolDefineXMLSymbolInit) {
        libvirtLoad();
        virStoragePoolDefineXMLSymbol = libvirtSymbol(libvirt, "virStoragePoolDefineXML");
        virStoragePoolDefineXMLSymbolInit = (virStoragePoolDefineXMLSymbol != NULL);
    }
    if (!virStoragePoolDefineXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virStoragePoolDeleteType virStoragePoolDeleteSymbol;
    static bool virStoragePoolDeleteSymbolInit;
    int ret;
    if (!virStoragePoolDeleteSymbolInit) {
        libvirtLoad();
        virStoragePoolDeleteSymbol = libvirtSymbol(libvirt, "virStoragePoolDelete");
        virStoragePoolDeleteSymbolInit = (virStoragePoolDeleteSymbol != NULL);
    }
    if (!virStoragePoolDeleteSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virStoragePoolDestroyType virStoragePoolDestroySymbol;
    static bool virStoragePoolDestroySymbolInit;
    int ret;
    if (!virStoragePoolDestroySymbolInit) {
        libvirtLoad();
        virStoragePoolDestroySymbol = libvirtSymbol(libvirt, "virStoragePoolDestroy");
        virStoragePoolDestroySymbolInit = (virStoragePoolDestroySymbol != NULL);
    }
    if (!virStoragePoolDestroySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virStoragePoolFreeType virStoragePoolFreeSymbol;
    static bool virStoragePoolFreeSymbolInit;
    int ret;
    if (!virStoragePoolFreeSymbolInit) {
        libvirtLoad();
        virStoragePoolFreeSymbol = libvirtSymbol(libvirt, "virStoragePoolFree");
        virStoragePoolFreeSymbolInit = (virStoragePoolFreeSymbol != NULL);
    }
    if (!virStoragePoolFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStoragePoolGetAutostartType virStoragePoolGetAutostartSymbol;
    static bool virStoragePoolGetAutostartSymbolInit;
    int ret;
    if (!virStoragePoolGetAutostartSymbolInit) {
        libvirtLoad();
        virStoragePoolGetAutostartSymbol = libvirtSymbol(libvirt, "virStoragePoolGetAutostart");
        virStoragePoolGetAutostartSymbolInit = (virStoragePoolGetAutostartSymbol != NULL);
    }
    if (!virStoragePoolGetAutostartSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virStoragePoolGetConnectType virStoragePoolGetConnectSymbol;
    static bool virStoragePoolGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virStoragePoolGetConnectSymbolInit) {
        libvirtLoad();
        virStoragePoolGetConnectSymbol = libvirtSymbol(libvirt, "virStoragePoolGetConnect");
        virStoragePoolGetConnectSymbolInit = (virStoragePoolGetConnectSymbol != NULL);
    }
    if (!virStoragePoolGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virStoragePoolGetInfoType virStoragePoolGetInfoSymbol;
    static bool virStoragePoolGetInfoSymbolInit;
    int ret;
    if (!virStoragePoolGetInfoSymbolInit) {
        libvirtLoad();
        virStoragePoolGetInfoSymbol = libvirtSymbol(libvirt, "virStoragePoolGetInfo");
        virStoragePoolGetInfoSymbolInit = (virStoragePoolGetInfoSymbol != NULL);
    }
    if (!virStoragePoolGetInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virStoragePoolGetNameType virStoragePoolGetNameSymbol;
    static bool virStoragePoolGetNameSymbolInit;
    const char * ret;
    if (!virStoragePoolGetNameSymbolInit) {
        libvirtLoad();
        virStoragePoolGetNameSymbol = libvirtSymbol(libvirt, "virStoragePoolGetName");
        virStoragePoolGetNameSymbolInit = (virStoragePoolGetNameSymbol != NULL);
    }
    if (!virStoragePoolGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virStoragePoolGetUUIDType virStoragePoolGetUUIDSymbol;
    static bool virStoragePoolGetUUIDSymbolInit;
    int ret;
    if (!virStoragePoolGetUUIDSymbolInit) {
        libvirtLoad();
        virStoragePoolGetUUIDSymbol = libvirtSymbol(libvirt, "virStoragePoolGetUUID");
        virStoragePoolGetUUIDSymbolInit = (virStoragePoolGetUUIDSymbol != NULL);
    }
    if (!virStoragePoolGetUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virStoragePoolGetUUIDStringType virStoragePoolGetUUIDStringSymbol;
    static bool virStoragePoolGetUUIDStringSymbolInit;
    int ret;
    if (!virStoragePoolGetUUIDStringSymbolInit) {
        libvirtLoad();
        virStoragePoolGetUUIDStringSymbol = libvirtSymbol(libvirt, "virStoragePoolGetUUIDString");
        virStoragePoolGetUUIDStringSymbolInit = (virStoragePoolGetUUIDStringSymbol != NULL);
    }
    if (!virStoragePoolGetUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virStoragePoolGetXMLDescType virStoragePoolGetXMLDescSymbol;
    static bool virStoragePoolGetXMLDescSymbolInit;
    char * ret;
    if (!virStoragePoolGetXMLDescSymbolInit) {
        libvirtLoad();
        virStoragePoolGetXMLDescSymbol = libvirtSymbol(libvirt, "virStoragePoolGetXMLDesc");
        virStoragePoolGetXMLDescSymbolInit = (virStoragePoolGetXMLDescSymbol != NULL);
    }
    if (!virStoragePoolGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virStoragePoolIsActiveType virStoragePoolIsActiveSymbol;
    static bool virStoragePoolIsActiveSymbolInit;
    int ret;
    if (!virStoragePoolIsActiveSymbolInit) {
        libvirtLoad();
        virStoragePoolIsActiveSymbol = libvirtSymbol(libvirt, "virStoragePoolIsActive");
        virStoragePoolIsActiveSymbolInit = (virStoragePoolIsActiveSymbol != NULL);
    }
    if (!virStoragePoolIsActiveSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStoragePoolIsPersistentType virStoragePoolIsPersistentSymbol;
    static bool virStoragePoolIsPersistentSymbolInit;
    int ret;
    if (!virStoragePoolIsPersistentSymbolInit) {
        libvirtLoad();
        virStoragePoolIsPersistentSymbol = libvirtSymbol(libvirt, "virStoragePoolIsPersistent");
        virStoragePoolIsPersistentSymbolInit = (virStoragePoolIsPersistentSymbol != NULL);
    }
    if (!virStoragePoolIsPersistentSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virStoragePoolListAllVolumesType virStoragePoolListAllVolumesSymbol;
    static bool virStoragePoolListAllVolumesSymbolInit;
    int ret;
    if (!virStoragePoolListAllVolumesSymbolInit) {
        libvirtLoad();
        virStoragePoolListAllVolumesSymbol = libvirtSymbol(libvirt, "virStoragePoolListAllVolumes");
        virStoragePoolListAllVolumesSymbolInit = (virStoragePoolListAllVolumesSymbol != NULL);
    }
    if (!virStoragePoolListAllVolumesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virStoragePoolListVolumesType virStoragePoolListVolumesSymbol;
    static bool virStoragePoolListVolumesSymbolInit;
    int ret;
    if (!virStoragePoolListVolumesSymbolInit) {
        libvirtLoad();
        virStoragePoolListVolumesSymbol = libvirtSymbol(libvirt, "virStoragePoolListVolumes");
        virStoragePoolListVolumesSymbolInit = (virStoragePoolListVolumesSymbol != NULL);
    }
    if (!virStoragePoolListVolumesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStoragePoolLookupByNameType virStoragePoolLookupByNameSymbol;
    static bool virStoragePoolLookupByNameSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolLookupByNameSymbolInit) {
        libvirtLoad();
        virStoragePoolLookupByNameSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByName");
        virStoragePoolLookupByNameSymbolInit = (virStoragePoolLookupByNameSymbol != NULL);
    }
    if (!virStoragePoolLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virStoragePoolLookupByTargetPathType virStoragePoolLookupByTargetPathSymbol;
    static bool virStoragePoolLookupByTargetPathSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolLookupByTargetPathSymbolInit) {
        libvirtLoad();
        virStoragePoolLookupByTargetPathSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByTargetPath");
        virStoragePoolLookupByTargetPathSymbolInit = (virStoragePoolLookupByTargetPathSymbol != NULL);
    }
    if (!virStoragePoolLookupByTargetPathSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStoragePoolLookupByUUIDType virStoragePoolLookupByUUIDSymbol;
    static bool virStoragePoolLookupByUUIDSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolLookupByUUIDSymbolInit) {
        libvirtLoad();
        virStoragePoolLookupByUUIDSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByUUID");
        virStoragePoolLookupByUUIDSymbolInit = (virStoragePoolLookupByUUIDSymbol != NULL);
    }
    if (!virStoragePoolLookupByUUIDSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                        virErrorPtr err) {
    static virStoragePoolLookupByUUIDStringType virStoragePoolLookupByUUIDStringSymbol;
    static bool virStoragePoolLookupByUUIDStringSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolLookupByUUIDStringSymbolInit) {
        libvirtLoad();
        virStoragePoolLookupByUUIDStringSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByUUIDString");
        virStoragePoolLookupByUUIDStringSymbolInit = (virStoragePoolLookupByUUIDStringSymbol != NULL);
    }
    if (!virStoragePoolLookupByUUIDStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virStoragePoolLookupByVolumeType virStoragePoolLookupByVolumeSymbol;
    static bool virStoragePoolLookupByVolumeSymbolInit;
    virStoragePoolPtr ret;
    if (!virStoragePoolLookupByVolumeSymbolInit) {
        libvirtLoad();
        virStoragePoolLookupByVolumeSymbol = libvirtSymbol(libvirt, "virStoragePoolLookupByVolume");
        virStoragePoolLookupByVolumeSymbolInit = (virStoragePoolLookupByVolumeSymbol != NULL);
    }
    if (!virStoragePoolLookupByVolumeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStoragePoolNumOfVolumesType virStoragePoolNumOfVolumesSymbol;
    static bool virStoragePoolNumOfVolumesSymbolInit;
    int ret;
    if (!virStoragePoolNumOfVolumesSymbolInit) {
        libvirtLoad();
        virStoragePoolNumOfVolumesSymbol = libvirtSymbol(libvirt, "virStoragePoolNumOfVolumes");
        virStoragePoolNumOfVolumesSymbolInit = (virStoragePoolNumOfVolumesSymbol != NULL);
    }
    if (!virStoragePoolNumOfVolumesSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virStoragePoolRefType virStoragePoolRefSymbol;
    static bool virStoragePoolRefSymbolInit;
    int ret;
    if (!virStoragePoolRefSymbolInit) {
        libvirtLoad();
        virStoragePoolRefSymbol = libvirtSymbol(libvirt, "virStoragePoolRef");
        virStoragePoolRefSymbolInit = (virStoragePoolRefSymbol != NULL);
    }
    if (!virStoragePoolRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virStoragePoolRefreshType virStoragePoolRefreshSymbol;
    static bool virStoragePoolRefreshSymbolInit;
    int ret;
    if (!virStoragePoolRefreshSymbolInit) {
        libvirtLoad();
        virStoragePoolRefreshSymbol = libvirtSymbol(libvirt, "virStoragePoolRefresh");
        virStoragePoolRefreshSymbolInit = (virStoragePoolRefreshSymbol != NULL);
    }
    if (!virStoragePoolRefreshSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStoragePoolSetAutostartType virStoragePoolSetAutostartSymbol;
    static bool virStoragePoolSetAutostartSymbolInit;
    int ret;
    if (!virStoragePoolSetAutostartSymbolInit) {
        libvirtLoad();
        virStoragePoolSetAutostartSymbol = libvirtSymbol(libvirt, "virStoragePoolSetAutostart");
        virStoragePoolSetAutostartSymbolInit = (virStoragePoolSetAutostartSymbol != NULL);
    }
    if (!virStoragePoolSetAutostartSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virStoragePoolUndefineType virStoragePoolUndefineSymbol;
    static bool virStoragePoolUndefineSymbolInit;
    int ret;
    if (!virStoragePoolUndefineSymbolInit) {
        libvirtLoad();
        virStoragePoolUndefineSymbol = libvirtSymbol(libvirt, "virStoragePoolUndefine");
        virStoragePoolUndefineSymbolInit = (virStoragePoolUndefineSymbol != NULL);
    }
    if (!virStoragePoolUndefineSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virStorageVolCreateXMLType virStorageVolCreateXMLSymbol;
    static bool virStorageVolCreateXMLSymbolInit;
    virStorageVolPtr ret;
    if (!virStorageVolCreateXMLSymbolInit) {
        libvirtLoad();
        virStorageVolCreateXMLSymbol = libvirtSymbol(libvirt, "virStorageVolCreateXML");
        virStorageVolCreateXMLSymbolInit = (virStorageVolCreateXMLSymbol != NULL);
    }
    if (!virStorageVolCreateXMLSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virStorageVolCreateXMLFromType virStorageVolCreateXMLFromSymbol;
    static bool virStorageVolCreateXMLFromSymbolInit;
    virStorageVolPtr ret;
    if (!virStorageVolCreateXMLFromSymbolInit) {
        libvirtLoad();
        virStorageVolCreateXMLFromSymbol = libvirtSymbol(libvirt, "virStorageVolCreateXMLFrom");
        virStorageVolCreateXMLFromSymbolInit = (virStorageVolCreateXMLFromSymbol != NULL);
    }
    if (!virStorageVolCreateXMLFromSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virStorageVolDeleteType virStorageVolDeleteSymbol;
    static bool virStorageVolDeleteSymbolInit;
    int ret;
    if (!virStorageVolDeleteSymbolInit) {
        libvirtLoad();
        virStorageVolDeleteSymbol = libvirtSymbol(libvirt, "virStorageVolDelete");
        virStorageVolDeleteSymbolInit = (virStorageVolDeleteSymbol != NULL);
    }
    if (!virStorageVolDeleteSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virStorageVolDownloadType virStorageVolDownloadSymbol;
    static bool virStorageVolDownloadSymbolInit;
    int ret;
    if (!virStorageVolDownloadSymbolInit) {
        libvirtLoad();
        virStorageVolDownloadSymbol = libvirtSymbol(libvirt, "virStorageVolDownload");
        virStorageVolDownloadSymbolInit = (virStorageVolDownloadSymbol != NULL);
    }
    if (!virStorageVolDownloadSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virStorageVolFreeType virStorageVolFreeSymbol;
    static bool virStorageVolFreeSymbolInit;
    int ret;
    if (!virStorageVolFreeSymbolInit) {
        libvirtLoad();
        virStorageVolFreeSymbol = libvirtSymbol(libvirt, "virStorageVolFree");
        virStorageVolFreeSymbolInit = (virStorageVolFreeSymbol != NULL);
    }
    if (!virStorageVolFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virStorageVolGetConnectType virStorageVolGetConnectSymbol;
    static bool virStorageVolGetConnectSymbolInit;
    virConnectPtr ret;
    if (!virStorageVolGetConnectSymbolInit) {
        libvirtLoad();
        virStorageVolGetConnectSymbol = libvirtSymbol(libvirt, "virStorageVolGetConnect");
        virStorageVolGetConnectSymbolInit = (virStorageVolGetConnectSymbol != NULL);
    }
    if (!virStorageVolGetConnectSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virStorageVolGetInfoType virStorageVolGetInfoSymbol;
    static bool virStorageVolGetInfoSymbolInit;
    int ret;
    if (!virStorageVolGetInfoSymbolInit) {
        libvirtLoad();
        virStorageVolGetInfoSymbol = libvirtSymbol(libvirt, "virStorageVolGetInfo");
        virStorageVolGetInfoSymbolInit = (virStorageVolGetInfoSymbol != NULL);
    }
    if (!virStorageVolGetInfoSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virStorageVolGetInfoFlagsType virStorageVolGetInfoFlagsSymbol;
    static bool virStorageVolGetInfoFlagsSymbolInit;
    int ret;
    if (!virStorageVolGetInfoFlagsSymbolInit) {
        libvirtLoad();
        virStorageVolGetInfoFlagsSymbol = libvirtSymbol(libvirt, "virStorageVolGetInfoFlags");
        virStorageVolGetInfoFlagsSymbolInit = (virStorageVolGetInfoFlagsSymbol != NULL);
    }
    if (!virStorageVolGetInfoFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virStorageVolGetKeyType virStorageVolGetKeySymbol;
    static bool virStorageVolGetKeySymbolInit;
    const char * ret;
    if (!virStorageVolGetKeySymbolInit) {
        libvirtLoad();
        virStorageVolGetKeySymbol = libvirtSymbol(libvirt, "virStorageVolGetKey");
        virStorageVolGetKeySymbolInit = (virStorageVolGetKeySymbol != NULL);
    }
    if (!virStorageVolGetKeySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virStorageVolGetNameType virStorageVolGetNameSymbol;
    static bool virStorageVolGetNameSymbolInit;
    const char * ret;
    if (!virStorageVolGetNameSymbolInit) {
        libvirtLoad();
        virStorageVolGetNameSymbol = libvirtSymbol(libvirt, "virStorageVolGetName");
        virStorageVolGetNameSymbolInit = (virStorageVolGetNameSymbol != NULL);
    }
    if (!virStorageVolGetNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virStorageVolGetPathType virStorageVolGetPathSymbol;
    static bool virStorageVolGetPathSymbolInit;
    char * ret;
    if (!virStorageVolGetPathSymbolInit) {
        libvirtLoad();
        virStorageVolGetPathSymbol = libvirtSymbol(libvirt, "virStorageVolGetPath");
        virStorageVolGetPathSymbolInit = (virStorageVolGetPathSymbol != NULL);
    }
    if (!virStorageVolGetPathSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virStorageVolGetXMLDescType virStorageVolGetXMLDescSymbol;
    static bool virStorageVolGetXMLDescSymbolInit;
    char * ret;
    if (!virStorageVolGetXMLDescSymbolInit) {
        libvirtLoad();
        virStorageVolGetXMLDescSymbol = libvirtSymbol(libvirt, "virStorageVolGetXMLDesc");
        virStorageVolGetXMLDescSymbolInit = (virStorageVolGetXMLDescSymbol != NULL);
    }
    if (!virStorageVolGetXMLDescSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virStorageVolLookupByKeyType virStorageVolLookupByKeySymbol;
    static bool virStorageVolLookupByKeySymbolInit;
    virStorageVolPtr ret;
    if (!virStorageVolLookupByKeySymbolInit) {
        libvirtLoad();
        virStorageVolLookupByKeySymbol = libvirtSymbol(libvirt, "virStorageVolLookupByKey");
        virStorageVolLookupByKeySymbolInit = (virStorageVolLookupByKeySymbol != NULL);
    }
    if (!virStorageVolLookupByKeySymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virStorageVolLookupByNameType virStorageVolLookupByNameSymbol;
    static bool virStorageVolLookupByNameSymbolInit;
    virStorageVolPtr ret;
    if (!virStorageVolLookupByNameSymbolInit) {
        libvirtLoad();
        virStorageVolLookupByNameSymbol = libvirtSymbol(libvirt, "virStorageVolLookupByName");
        virStorageVolLookupByNameSymbolInit = (virStorageVolLookupByNameSymbol != NULL);
    }
    if (!virStorageVolLookupByNameSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virStorageVolLookupByPathType virStorageVolLookupByPathSymbol;
    static bool virStorageVolLookupByPathSymbolInit;
    virStorageVolPtr ret;
    if (!virStorageVolLookupByPathSymbolInit) {
        libvirtLoad();
        virStorageVolLookupByPathSymbol = libvirtSymbol(libvirt, "virStorageVolLookupByPath");
        virStorageVolLookupByPathSymbolInit = (virStorageVolLookupByPathSymbol != NULL);
    }
    if (!virStorageVolLookupByPathSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virStorageVolRefType virStorageVolRefSymbol;
    static bool virStorageVolRefSymbolInit;
    int ret;
    if (!virStorageVolRefSymbolInit) {
        libvirtLoad();
        virStorageVolRefSymbol = libvirtSymbol(libvirt, "virStorageVolRef");
        virStorageVolRefSymbolInit = (virStorageVolRefSymbol != NULL);
    }
    if (!virStorageVolRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virStorageVolResizeType virStorageVolResizeSymbol;
    static bool virStorageVolResizeSymbolInit;
    int ret;
    if (!virStorageVolResizeSymbolInit) {
        libvirtLoad();
        virStorageVolResizeSymbol = libvirtSymbol(libvirt, "virStorageVolResize");
        virStorageVolResizeSymbolInit = (virStorageVolResizeSymbol != NULL);
    }
    if (!virStorageVolResizeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virStorageVolUploadType virStorageVolUploadSymbol;
    static bool virStorageVolUploadSymbolInit;
    int ret;
    if (!virStorageVolUploadSymbolInit) {
        libvirtLoad();
        virStorageVolUploadSymbol = libvirtSymbol(libvirt, "virStorageVolUpload");
        virStorageVolUploadSymbolInit = (virStorageVolUploadSymbol != NULL);
    }
    if (!virStorageVolUploadSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virStorageVolWipeType virStorageVolWipeSymbol;
    static bool virStorageVolWipeSymbolInit;
    int ret;
    if (!virStorageVolWipeSymbolInit) {
        libvirtLoad();
        virStorageVolWipeSymbol = libvirtSymbol(libvirt, "virStorageVolWipe");
        virStorageVolWipeSymbolInit = (virStorageVolWipeSymbol != NULL);
    }
    if (!virStorageVolWipeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virStorageVolWipePatternType virStorageVolWipePatternSymbol;
    static bool virStorageVolWipePatternSymbolInit;
    int ret;
    if (!virStorageVolWipePatternSymbolInit) {
        libvirtLoad();
        virStorageVolWipePatternSymbol = libvirtSymbol(libvirt, "virStorageVolWipePattern");
        virStorageVolWipePatternSymbolInit = (virStorageVolWipePatternSymbol != NULL);
    }
    if (!virStorageVolWipePatternSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                      virErrorPtr err) {
    static virStreamAbortType virStreamAbortSymbol;
    static bool virStreamAbortSymbolInit;
    int ret;
    if (!virStreamAbortSymbolInit) {
        libvirtLoad();
        virStreamAbortSymbol = libvirtSymbol(libvirt, "virStreamAbort");
        virStreamAbortSymbolInit = (virStreamAbortSymbol != NULL);
    }
    if (!virStreamAbortSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virStreamEventAddCallbackType virStreamEventAddCallbackSymbol;
    static bool virStreamEventAddCallbackSymbolInit;
    int ret;
    if (!virStreamEventAddCallbackSymbolInit) {
        libvirtLoad();
        virStreamEventAddCallbackSymbol = libvirtSymbol(libvirt, "virStreamEventAddCallback");
        virStreamEventAddCallbackSymbolInit = (virStreamEventAddCallbackSymbol != NULL);
    }
    if (!virStreamEventAddCallbackSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virStreamEventRemoveCallbackType virStreamEventRemoveCallbackSymbol;
    static bool virStreamEventRemoveCallbackSymbolInit;
    int ret;
    if (!virStreamEventRemoveCallbackSymbolInit) {
        libvirtLoad();
        virStreamEventRemoveCallbackSymbol = libvirtSymbol(libvirt, "virStreamEventRemoveCallback");
        virStreamEventRemoveCallbackSymbolInit = (virStreamEventRemoveCallbackSymbol != NULL);
    }
    if (!virStreamEventRemoveCallbackSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                    virErrorPtr err) {
    static virStreamEventUpdateCallbackType virStreamEventUpdateCallbackSymbol;
    static bool virStreamEventUpdateCallbackSymbolInit;
    int ret;
    if (!virStreamEventUpdateCallbackSymbolInit) {
        libvirtLoad();
        virStreamEventUpdateCallbackSymbol = libvirtSymbol(libvirt, "virStreamEventUpdateCallback");
        virStreamEventUpdateCallbackSymbolInit = (virStreamEventUpdateCallbackSymbol != NULL);
    }
    if (!virStreamEventUpdateCallbackSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                       virErrorPtr err) {
    static virStreamFinishType virStreamFinishSymbol;
    static bool virStreamFinishSymbolInit;
    int ret;
    if (!virStreamFinishSymbolInit) {
        libvirtLoad();
        virStreamFinishSymbol = libvirtSymbol(libvirt, "virStreamFinish");
        virStreamFinishSymbolInit = (virStreamFinishSymbol != NULL);
    }
    if (!virStreamFinishSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virStreamFreeType virStreamFreeSymbol;
    static bool virStreamFreeSymbolInit;
    int ret;
    if (!virStreamFreeSymbolInit) {
        libvirtLoad();
        virStreamFreeSymbol = libvirtSymbol(libvirt, "virStreamFree");
        virStreamFreeSymbolInit = (virStreamFreeSymbol != NULL);
    }
    if (!virStreamFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                    virErrorPtr err) {
    static virStreamNewType virStreamNewSymbol;
    static bool virStreamNewSymbolInit;
    virStreamPtr ret;
    if (!virStreamNewSymbolInit) {
        libvirtLoad();
        virStreamNewSymbol = libvirtSymbol(libvirt, "virStreamNew");
        virStreamNewSymbolInit = (virStreamNewSymbol != NULL);
    }
    if (!virStreamNewSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virStreamRecvType virStreamRecvSymbol;
    static bool virStreamRecvSymbolInit;
    int ret;
    if (!virStreamRecvSymbolInit) {
        libvirtLoad();
        virStreamRecvSymbol = libvirtSymbol(libvirt, "virStreamRecv");
        virStreamRecvSymbolInit = (virStreamRecvSymbol != NULL);
    }
    if (!virStreamRecvSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virStreamRecvAllType virStreamRecvAllSymbol;
    static bool virStreamRecvAllSymbolInit;
    int ret;
    if (!virStreamRecvAllSymbolInit) {
        libvirtLoad();
        virStreamRecvAllSymbol = libvirtSymbol(libvirt, "virStreamRecvAll");
        virStreamRecvAllSymbolInit = (virStreamRecvAllSymbol != NULL);
    }
    if (!virStreamRecvAllSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                          virErrorPtr err) {
    static virStreamRecvFlagsType virStreamRecvFlagsSymbol;
    static bool virStreamRecvFlagsSymbolInit;
    int ret;
    if (!virStreamRecvFlagsSymbolInit) {
        libvirtLoad();
        virStreamRecvFlagsSymbol = libvirtSymbol(libvirt, "virStreamRecvFlags");
        virStreamRecvFlagsSymbolInit = (virStreamRecvFlagsSymbol != NULL);
    }
    if (!virStreamRecvFlagsSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virStreamRecvHoleType virStreamRecvHoleSymbol;
    static bool virStreamRecvHoleSymbolInit;
    int ret;
    if (!virStreamRecvHoleSymbolInit) {
        libvirtLoad();
        virStreamRecvHoleSymbol = libvirtSymbol(libvirt, "virStreamRecvHole");
        virStreamRecvHoleSymbolInit = (virStreamRecvHoleSymbol != NULL);
    }
    if (!virStreamRecvHoleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                    virErrorPtr err) {
    static virStreamRefType virStreamRefSymbol;
    static bool virStreamRefSymbolInit;
    int ret;
    if (!virStreamRefSymbolInit) {
        libvirtLoad();
        virStreamRefSymbol = libvirtSymbol(libvirt, "virStreamRef");
        virStreamRefSymbolInit = (virStreamRefSymbol != NULL);
    }
    if (!virStreamRefSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                     virErrorPtr err) {
    static virStreamSendType virStreamSendSymbol;
    static bool virStreamSendSymbolInit;
    int ret;
    if (!virStreamSendSymbolInit) {
        libvirtLoad();
        virStreamSendSymbol = libvirtSymbol(libvirt, "virStreamSend");
        virStreamSendSymbolInit = (virStreamSendSymbol != NULL);
    }
    if (!virStreamSendSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                        virErrorPtr err) {
    static virStreamSendAllType virStreamSendAllSymbol;
    static bool virStreamSendAllSymbolInit;
    int ret;
    if (!virStreamSendAllSymbolInit) {
        libvirtLoad();
        virStreamSendAllSymbol = libvirtSymbol(libvirt, "virStreamSendAll");
        virStreamSendAllSymbolInit = (virStreamSendAllSymbol != NULL);
    }
    if (!virStreamSendAllSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virStreamSendHoleType virStreamSendHoleSymbol;
    static bool virStreamSendHoleSymbolInit;
    int ret;
    if (!virStreamSendHoleSymbolInit) {
        libvirtLoad();
        virStreamSendHoleSymbol = libvirtSymbol(libvirt, "virStreamSendHole");
        virStreamSendHoleSymbolInit = (virStreamSendHoleSymbol != NULL);
    }
    if (!virStreamSendHoleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virStreamSparseRecvAllType virStreamSparseRecvAllSymbol;
    static bool virStreamSparseRecvAllSymbolInit;
    int ret;
    if (!virStreamSparseRecvAllSymbolInit) {
        libvirtLoad();
        virStreamSparseRecvAllSymbol = libvirtSymbol(libvirt, "virStreamSparseRecvAll");
        virStreamSparseRecvAllSymbolInit = (virStreamSparseRecvAllSymbol != NULL);
    }
    if (!virStreamSparseRecvAllSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virStreamSparseSendAllType virStreamSparseSendAllSymbol;
    static bool virStreamSparseSendAllSymbolInit;
    int ret;
    if (!virStreamSparseSendAllSymbolInit) {
        libvirtLoad();
        virStreamSparseSendAllSymbol = libvirtSymbol(libvirt, "virStreamSparseSendAll");
        virStreamSparseSendAllSymbolInit = (virStreamSparseSendAllSymbol != NULL);
    }
    if (!virStreamSparseSendAllSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virTypedParamsAddBooleanType virTypedParamsAddBooleanSymbol;
    static bool virTypedParamsAddBooleanSymbolInit;
    int ret;
    if (!virTypedParamsAddBooleanSymbolInit) {
        libvirtLoad();
        virTypedParamsAddBooleanSymbol = libvirtSymbol(libvirt, "virTypedParamsAddBoolean");
        virTypedParamsAddBooleanSymbolInit = (virTypedParamsAddBooleanSymbol != NULL);
    }
    if (!virTypedParamsAddBooleanSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virTypedParamsAddDoubleType virTypedParamsAddDoubleSymbol;
    static bool virTypedParamsAddDoubleSymbolInit;
    int ret;
    if (!virTypedParamsAddDoubleSymbolInit) {
        libvirtLoad();
        virTypedParamsAddDoubleSymbol = libvirtSymbol(libvirt, "virTypedParamsAddDouble");
        virTypedParamsAddDoubleSymbolInit = (virTypedParamsAddDoubleSymbol != NULL);
    }
    if (!virTypedParamsAddDoubleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virTypedParamsAddFromStringType virTypedParamsAddFromStringSymbol;
    static bool virTypedParamsAddFromStringSymbolInit;
    int ret;
    if (!virTypedParamsAddFromStringSymbolInit) {
        libvirtLoad();
        virTypedParamsAddFromStringSymbol = libvirtSymbol(libvirt, "virTypedParamsAddFromString");
        virTypedParamsAddFromStringSymbolInit = (virTypedParamsAddFromStringSymbol != NULL);
    }
    if (!virTypedParamsAddFromStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virTypedParamsAddIntType virTypedParamsAddIntSymbol;
    static bool virTypedParamsAddIntSymbolInit;
    int ret;
    if (!virTypedParamsAddIntSymbolInit) {
        libvirtLoad();
        virTypedParamsAddIntSymbol = libvirtSymbol(libvirt, "virTypedParamsAddInt");
        virTypedParamsAddIntSymbolInit = (virTypedParamsAddIntSymbol != NULL);
    }
    if (!virTypedParamsAddIntSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virTypedParamsAddLLongType virTypedParamsAddLLongSymbol;
    static bool virTypedParamsAddLLongSymbolInit;
    int ret;
    if (!virTypedParamsAddLLongSymbolInit) {
        libvirtLoad();
        virTypedParamsAddLLongSymbol = libvirtSymbol(libvirt, "virTypedParamsAddLLong");
        virTypedParamsAddLLongSymbolInit = (virTypedParamsAddLLongSymbol != NULL);
    }
    if (!virTypedParamsAddLLongSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virTypedParamsAddStringType virTypedParamsAddStringSymbol;
    static bool virTypedParamsAddStringSymbolInit;
    int ret;
    if (!virTypedParamsAddStringSymbolInit) {
        libvirtLoad();
        virTypedParamsAddStringSymbol = libvirtSymbol(libvirt, "virTypedParamsAddString");
        virTypedParamsAddStringSymbolInit = (virTypedParamsAddStringSymbol != NULL);
    }
    if (!virTypedParamsAddStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virTypedParamsAddStringListType virTypedParamsAddStringListSymbol;
    static bool virTypedParamsAddStringListSymbolInit;
    int ret;
    if (!virTypedParamsAddStringListSymbolInit) {
        libvirtLoad();
        virTypedParamsAddStringListSymbol = libvirtSymbol(libvirt, "virTypedParamsAddStringList");
        virTypedParamsAddStringListSymbolInit = (virTypedParamsAddStringListSymbol != NULL);
    }
    if (!virTypedParamsAddStringListSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virTypedParamsAddUIntType virTypedParamsAddUIntSymbol;
    static bool virTypedParamsAddUIntSymbolInit;
    int ret;
    if (!virTypedParamsAddUIntSymbolInit) {
        libvirtLoad();
        virTypedParamsAddUIntSymbol = libvirtSymbol(libvirt, "virTypedParamsAddUInt");
        virTypedParamsAddUIntSymbolInit = (virTypedParamsAddUIntSymbol != NULL);
    }
    if (!virTypedParamsAddUIntSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virTypedParamsAddULLongType virTypedParamsAddULLongSymbol;
    static bool virTypedParamsAddULLongSymbolInit;
    int ret;
    if (!virTypedParamsAddULLongSymbolInit) {
        libvirtLoad();
        virTypedParamsAddULLongSymbol = libvirtSymbol(libvirt, "virTypedParamsAddULLong");
        virTypedParamsAddULLongSymbolInit = (virTypedParamsAddULLongSymbol != NULL);
    }
    if (!virTypedParamsAddULLongSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           int nparams) {
    static virTypedParamsClearType virTypedParamsClearSymbol;
    static bool virTypedParamsClearSymbolInit;

    if (!virTypedParamsClearSymbolInit) {
        libvirtLoad();
        virTypedParamsClearSymbol = libvirtSymbol(libvirt, "virTypedParamsClear");
        virTypedParamsClearSymbolInit = (virTypedParamsClearSymbol != NULL);
    }
    if (!virTypedParamsClearSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    virTypedParamsClearSymbol(params,
                              nparams);

}
typedef void
(*virTypedParamsFreeType)(virTypedParameterPtr params,
                          int nparams);

void
virTypedParamsFreeWrapper(virTypedParameterPtr params,
                          int nparams) {
    static virTypedParamsFreeType virTypedParamsFreeSymbol;
    static bool virTypedParamsFreeSymbolInit;

    if (!virTypedParamsFreeSymbolInit) {
        libvirtLoad();
        virTypedParamsFreeSymbol = libvirtSymbol(libvirt, "virTypedParamsFree");
        virTypedParamsFreeSymbolInit = (virTypedParamsFreeSymbol != NULL);
    }
    if (!virTypedParamsFreeSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                         virErrorPtr err) {
    static virTypedParamsGetType virTypedParamsGetSymbol;
    static bool virTypedParamsGetSymbolInit;
    virTypedParameterPtr ret;
    if (!virTypedParamsGetSymbolInit) {
        libvirtLoad();
        virTypedParamsGetSymbol = libvirtSymbol(libvirt, "virTypedParamsGet");
        virTypedParamsGetSymbolInit = (virTypedParamsGetSymbol != NULL);
    }
    if (!virTypedParamsGetSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                virErrorPtr err) {
    static virTypedParamsGetBooleanType virTypedParamsGetBooleanSymbol;
    static bool virTypedParamsGetBooleanSymbolInit;
    int ret;
    if (!virTypedParamsGetBooleanSymbolInit) {
        libvirtLoad();
        virTypedParamsGetBooleanSymbol = libvirtSymbol(libvirt, "virTypedParamsGetBoolean");
        virTypedParamsGetBooleanSymbolInit = (virTypedParamsGetBooleanSymbol != NULL);
    }
    if (!virTypedParamsGetBooleanSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virTypedParamsGetDoubleType virTypedParamsGetDoubleSymbol;
    static bool virTypedParamsGetDoubleSymbolInit;
    int ret;
    if (!virTypedParamsGetDoubleSymbolInit) {
        libvirtLoad();
        virTypedParamsGetDoubleSymbol = libvirtSymbol(libvirt, "virTypedParamsGetDouble");
        virTypedParamsGetDoubleSymbolInit = (virTypedParamsGetDoubleSymbol != NULL);
    }
    if (!virTypedParamsGetDoubleSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                            virErrorPtr err) {
    static virTypedParamsGetIntType virTypedParamsGetIntSymbol;
    static bool virTypedParamsGetIntSymbolInit;
    int ret;
    if (!virTypedParamsGetIntSymbolInit) {
        libvirtLoad();
        virTypedParamsGetIntSymbol = libvirtSymbol(libvirt, "virTypedParamsGetInt");
        virTypedParamsGetIntSymbolInit = (virTypedParamsGetIntSymbol != NULL);
    }
    if (!virTypedParamsGetIntSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                              virErrorPtr err) {
    static virTypedParamsGetLLongType virTypedParamsGetLLongSymbol;
    static bool virTypedParamsGetLLongSymbolInit;
    int ret;
    if (!virTypedParamsGetLLongSymbolInit) {
        libvirtLoad();
        virTypedParamsGetLLongSymbol = libvirtSymbol(libvirt, "virTypedParamsGetLLong");
        virTypedParamsGetLLongSymbolInit = (virTypedParamsGetLLongSymbol != NULL);
    }
    if (!virTypedParamsGetLLongSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virTypedParamsGetStringType virTypedParamsGetStringSymbol;
    static bool virTypedParamsGetStringSymbolInit;
    int ret;
    if (!virTypedParamsGetStringSymbolInit) {
        libvirtLoad();
        virTypedParamsGetStringSymbol = libvirtSymbol(libvirt, "virTypedParamsGetString");
        virTypedParamsGetStringSymbolInit = (virTypedParamsGetStringSymbol != NULL);
    }
    if (!virTypedParamsGetStringSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                             virErrorPtr err) {
    static virTypedParamsGetUIntType virTypedParamsGetUIntSymbol;
    static bool virTypedParamsGetUIntSymbolInit;
    int ret;
    if (!virTypedParamsGetUIntSymbolInit) {
        libvirtLoad();
        virTypedParamsGetUIntSymbol = libvirtSymbol(libvirt, "virTypedParamsGetUInt");
        virTypedParamsGetUIntSymbolInit = (virTypedParamsGetUIntSymbol != NULL);
    }
    if (!virTypedParamsGetUIntSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                               virErrorPtr err) {
    static virTypedParamsGetULLongType virTypedParamsGetULLongSymbol;
    static bool virTypedParamsGetULLongSymbolInit;
    int ret;
    if (!virTypedParamsGetULLongSymbolInit) {
        libvirtLoad();
        virTypedParamsGetULLongSymbol = libvirtSymbol(libvirt, "virTypedParamsGetULLong");
        virTypedParamsGetULLongSymbolInit = (virTypedParamsGetULLongSymbol != NULL);
    }
    if (!virTypedParamsGetULLongSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
(*virConnectDomainQemuMonitorEventDeregisterType)(virConnectPtr conn,
                                                  int callbackID);

int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err) {
    static virConnectDomainQemuMonitorEventDeregisterType virConnectDomainQemuMonitorEventDeregisterSymbol;
    static bool virConnectDomainQemuMonitorEventDeregisterSymbolInit;
    int ret;
    if (!virConnectDomainQemuMonitorEventDeregisterSymbolInit) {
        libvirtLoad();
        virConnectDomainQemuMonitorEventDeregisterSymbol = libvirtSymbol(qemu, "virConnectDomainQemuMonitorEventDeregister");
        virConnectDomainQemuMonitorEventDeregisterSymbolInit = (virConnectDomainQemuMonitorEventDeregisterSymbol != NULL);
    }
    if (!virConnectDomainQemuMonitorEventDeregisterSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                                virErrorPtr err) {
    static virConnectDomainQemuMonitorEventRegisterType virConnectDomainQemuMonitorEventRegisterSymbol;
    static bool virConnectDomainQemuMonitorEventRegisterSymbolInit;
    int ret;
    if (!virConnectDomainQemuMonitorEventRegisterSymbolInit) {
        libvirtLoad();
        virConnectDomainQemuMonitorEventRegisterSymbol = libvirtSymbol(qemu, "virConnectDomainQemuMonitorEventRegister");
        virConnectDomainQemuMonitorEventRegisterSymbolInit = (virConnectDomainQemuMonitorEventRegisterSymbol != NULL);
    }
    if (!virConnectDomainQemuMonitorEventRegisterSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainQemuAgentCommandType virDomainQemuAgentCommandSymbol;
    static bool virDomainQemuAgentCommandSymbolInit;
    char * ret;
    if (!virDomainQemuAgentCommandSymbolInit) {
        libvirtLoad();
        virDomainQemuAgentCommandSymbol = libvirtSymbol(qemu, "virDomainQemuAgentCommand");
        virDomainQemuAgentCommandSymbolInit = (virDomainQemuAgentCommandSymbol != NULL);
    }
    if (!virDomainQemuAgentCommandSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                           virErrorPtr err) {
    static virDomainQemuAttachType virDomainQemuAttachSymbol;
    static bool virDomainQemuAttachSymbolInit;
    virDomainPtr ret;
    if (!virDomainQemuAttachSymbolInit) {
        libvirtLoad();
        virDomainQemuAttachSymbol = libvirtSymbol(qemu, "virDomainQemuAttach");
        virDomainQemuAttachSymbolInit = (virDomainQemuAttachSymbol != NULL);
    }
    if (!virDomainQemuAttachSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                   virErrorPtr err) {
    static virDomainQemuMonitorCommandType virDomainQemuMonitorCommandSymbol;
    static bool virDomainQemuMonitorCommandSymbolInit;
    int ret;
    if (!virDomainQemuMonitorCommandSymbolInit) {
        libvirtLoad();
        virDomainQemuMonitorCommandSymbol = libvirtSymbol(qemu, "virDomainQemuMonitorCommand");
        virDomainQemuMonitorCommandSymbolInit = (virDomainQemuMonitorCommandSymbol != NULL);
    }
    if (!virDomainQemuMonitorCommandSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
typedef int
(*virDomainLxcEnterCGroupType)(virDomainPtr domain,
                               unsigned int flags);

int
virDomainLxcEnterCGroupWrapper(virDomainPtr domain,
                               unsigned int flags,
                               virErrorPtr err) {
    static virDomainLxcEnterCGroupType virDomainLxcEnterCGroupSymbol;
    static bool virDomainLxcEnterCGroupSymbolInit;
    int ret;
    if (!virDomainLxcEnterCGroupSymbolInit) {
        libvirtLoad();
        virDomainLxcEnterCGroupSymbol = libvirtSymbol(lxc, "virDomainLxcEnterCGroup");
        virDomainLxcEnterCGroupSymbolInit = (virDomainLxcEnterCGroupSymbol != NULL);
    }
    if (!virDomainLxcEnterCGroupSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                  virErrorPtr err) {
    static virDomainLxcEnterNamespaceType virDomainLxcEnterNamespaceSymbol;
    static bool virDomainLxcEnterNamespaceSymbolInit;
    int ret;
    if (!virDomainLxcEnterNamespaceSymbolInit) {
        libvirtLoad();
        virDomainLxcEnterNamespaceSymbol = libvirtSymbol(lxc, "virDomainLxcEnterNamespace");
        virDomainLxcEnterNamespaceSymbolInit = (virDomainLxcEnterNamespaceSymbol != NULL);
    }
    if (!virDomainLxcEnterNamespaceSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                      virErrorPtr err) {
    static virDomainLxcEnterSecurityLabelType virDomainLxcEnterSecurityLabelSymbol;
    static bool virDomainLxcEnterSecurityLabelSymbolInit;
    int ret;
    if (!virDomainLxcEnterSecurityLabelSymbolInit) {
        libvirtLoad();
        virDomainLxcEnterSecurityLabelSymbol = libvirtSymbol(lxc, "virDomainLxcEnterSecurityLabel");
        virDomainLxcEnterSecurityLabelSymbolInit = (virDomainLxcEnterSecurityLabelSymbol != NULL);
    }
    if (!virDomainLxcEnterSecurityLabelSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
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
                                 virErrorPtr err) {
    static virDomainLxcOpenNamespaceType virDomainLxcOpenNamespaceSymbol;
    static bool virDomainLxcOpenNamespaceSymbolInit;
    int ret;
    if (!virDomainLxcOpenNamespaceSymbolInit) {
        libvirtLoad();
        virDomainLxcOpenNamespaceSymbol = libvirtSymbol(lxc, "virDomainLxcOpenNamespace");
        virDomainLxcOpenNamespaceSymbolInit = (virDomainLxcOpenNamespaceSymbol != NULL);
    }
    if (!virDomainLxcOpenNamespaceSymbolInit) {
        fprintf(stderr, "symbol err: %s\n", dlerror());
        assert(false);
    }
    ret = virDomainLxcOpenNamespaceSymbol(domain,
                                          fdlist,
                                          flags);

    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}
