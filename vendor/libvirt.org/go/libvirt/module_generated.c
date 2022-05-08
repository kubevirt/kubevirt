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
#include <string.h>
#ifdef USE_DLOPEN
#include <dlfcn.h>
#endif
#include "module_generated.h"

/* Exported variables */
virConnectAuthPtr virConnectAuthPtrDefaultVar;

#ifdef USE_DLOPEN
#  define LIBVIR_CHECK_VERSION(major, minor, micro) true
/* dlopen's handlers */
static void *libvirt;
static void *qemu;
static void *lxc;
#endif

static void
setVirError(virErrorPtr err, const char *message)
{
    if (err == NULL) {
        return;
    }
    memset(err, 0, sizeof(*err));

    err->code = VIR_ERR_INTERNAL_ERROR;
    err->domain = VIR_FROM_NONE;
    /* XXX: This is wrong due the fact that libvirt uses g_malloc() while
     * strdup() uses malloc() - problem is that doing a #include <glib> would
     * add another dep to libvirt-go-modules.
     * 1) for dlopen module, we need to dlsym g_strdup() to avoid linking
     * 2) for normal linking, should be fine to use g_strdup()
     */
    err->message = strdup(message);
    err->level = VIR_ERR_ERROR;
}

#ifdef USE_DLOPEN
static void *
libvirtSymbol(void *handle, const char *name, bool *success, virErrorPtr err)
{
    void *symbol = NULL;
    bool ok = true;
    if (handle == NULL) {
        ok = false;
        setVirError(err, "Library not loaded, can't load symbol");
        goto end_symbol;
    }

    /* Documentation of dlsym says we should use dlerror() to check for failure
     * in dlsym() as a NULL might be the right address for a given symbol.
     * This is also the reason to have the @success argument.
     */
    symbol = dlsym(handle, name);
    char *errMsg = dlerror();
    if (errMsg != NULL) {
        ok = false;
        setVirError(err, errMsg);
    }

end_symbol:
    if (success) {
        *success = ok;
    }
    return symbol;
}
#endif

static void
libvirtLoadLibvirtVariables(void)
{
#ifdef USE_DLOPEN
    virConnectAuthPtrDefaultVar = libvirtSymbol(libvirt, "virConnectAuthPtrDefault", NULL, NULL);
#else
virConnectAuthPtrDefaultVar = virConnectAuthPtrDefault;
#endif
}

#ifdef USE_DLOPEN
static bool
libvirtLoadOnceMain(virErrorPtr err)
{
    static bool once;
    if (once) {
        if (libvirt == NULL) {
            setVirError(err, "Could not dlopen libvirt");
            return false;
        }
        return true;
    }
    once = true;

    libvirt = dlopen("libvirt.so.0", RTLD_NOW|RTLD_LOCAL);
    if (libvirt == NULL) {
        setVirError(err, dlerror());
        return false;
    }
    libvirtLoadLibvirtVariables();
    return true;
}

static bool
libvirtLoadOnceLxc(virErrorPtr err)
{
    static bool once;
    if (once) {
        if (lxc == NULL) {
            setVirError(err, "Could not dlopen libvirt-lxc");
            return false;
        }
        return true;
    }
    once = true;

    lxc = dlopen("libvirt-lxc.so.0", RTLD_NOW|RTLD_LOCAL);
    if (lxc == NULL) {
        setVirError(err, dlerror());
        return false;
    }
    return true;
}

static bool
libvirtLoadOnceQemu(virErrorPtr err)
{
    static bool once;
    if (once) {
        if (qemu == NULL) {
            setVirError(err, "Could not dlopen libvirt-qemu");
            return false;
        }
        return true;
    }
    once = true;

    qemu = dlopen("libvirt-qemu.so.0", RTLD_NOW|RTLD_LOCAL); 
    if (qemu == NULL) {
        setVirError(err, dlerror());
        return false;
    }
    return true;
}

#endif /* USE_DLOPEN */
typedef int
(*virCopyLastErrorType)(virErrorPtr to);

int
virCopyLastErrorWrapper(virErrorPtr to) {
#ifdef USE_DLOPEN
    static virCopyLastErrorType virCopyLastErrorSymbol;
    static bool once;
    static bool success;
    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virCopyLastErrorSymbol = libvirtSymbol(libvirt, "virCopyLastError", &success, NULL);
        }
    }
    if (!success) {
        return -1;
    }
    return virCopyLastErrorSymbol(to);
#else
    return virCopyLastError(to);
#endif
}
typedef int
(*virConnCopyLastErrorType)(virConnectPtr conn,
                            virErrorPtr to);

int
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virConnCopyLastError compiled out (from 0.1.0)");
    return ret;
#else
    static virConnCopyLastErrorType virConnCopyLastErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnCopyLastErrorSymbol = libvirtSymbol(libvirt,
                                                       "virConnCopyLastError",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnCopyLastError");
        return ret;
    }
#  else
    virConnCopyLastErrorSymbol = &virConnCopyLastError;
#  endif

    ret = virConnCopyLastErrorSymbol(conn,
                                     to);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virErrorPtr
(*virConnGetLastErrorType)(virConnectPtr conn);

virErrorPtr
virConnGetLastErrorWrapper(virConnectPtr conn,
                           virErrorPtr err)
{
    virErrorPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virConnGetLastError compiled out (from 0.1.0)");
    return ret;
#else
    static virConnGetLastErrorType virConnGetLastErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnGetLastErrorSymbol = libvirtSymbol(libvirt,
                                                      "virConnGetLastError",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnGetLastError");
        return ret;
    }
#  else
    virConnGetLastErrorSymbol = &virConnGetLastError;
#  endif

    ret = virConnGetLastErrorSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virConnResetLastErrorType)(virConnectPtr conn);

void
virConnResetLastErrorWrapper(virConnectPtr conn)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virConnResetLastError compiled out (from 0.1.0)");
    return;
#else
    static virConnResetLastErrorType virConnResetLastErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virConnResetLastErrorSymbol = libvirtSymbol(libvirt,
                                                        "virConnResetLastError",
                                                        &success,
                                                        NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virConnResetLastError");
        return;
    }
#  else
    virConnResetLastErrorSymbol = &virConnResetLastError;
#  endif

    virConnResetLastErrorSymbol(conn);
#endif
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

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virConnSetErrorFunc compiled out (from 0.1.0)");
    return;
#else
    static virConnSetErrorFuncType virConnSetErrorFuncSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virConnSetErrorFuncSymbol = libvirtSymbol(libvirt,
                                                      "virConnSetErrorFunc",
                                                      &success,
                                                      NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virConnSetErrorFunc");
        return;
    }
#  else
    virConnSetErrorFuncSymbol = &virConnSetErrorFunc;
#  endif

    virConnSetErrorFuncSymbol(conn,
                              userData,
                              handler);
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virConnectBaselineCPU compiled out (from 0.7.7)");
    return ret;
#else
    static virConnectBaselineCPUType virConnectBaselineCPUSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectBaselineCPUSymbol = libvirtSymbol(libvirt,
                                                        "virConnectBaselineCPU",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectBaselineCPU");
        return ret;
    }
#  else
    virConnectBaselineCPUSymbol = &virConnectBaselineCPU;
#  endif

    ret = virConnectBaselineCPUSymbol(conn,
                                      xmlCPUs,
                                      ncpus,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 4, 0)
    setVirError(err, "Function virConnectBaselineHypervisorCPU compiled out (from 4.4.0)");
    return ret;
#else
    static virConnectBaselineHypervisorCPUType virConnectBaselineHypervisorCPUSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectBaselineHypervisorCPUSymbol = libvirtSymbol(libvirt,
                                                                  "virConnectBaselineHypervisorCPU",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectBaselineHypervisorCPU");
        return ret;
    }
#  else
    virConnectBaselineHypervisorCPUSymbol = &virConnectBaselineHypervisorCPU;
#  endif

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
#endif
}

typedef int
(*virConnectCloseType)(virConnectPtr conn);

int
virConnectCloseWrapper(virConnectPtr conn,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectClose compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectCloseType virConnectCloseSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectCloseSymbol = libvirtSymbol(libvirt,
                                                  "virConnectClose",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectClose");
        return ret;
    }
#  else
    virConnectCloseSymbol = &virConnectClose;
#  endif

    ret = virConnectCloseSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 5)
    setVirError(err, "Function virConnectCompareCPU compiled out (from 0.7.5)");
    return ret;
#else
    static virConnectCompareCPUType virConnectCompareCPUSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectCompareCPUSymbol = libvirtSymbol(libvirt,
                                                       "virConnectCompareCPU",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectCompareCPU");
        return ret;
    }
#  else
    virConnectCompareCPUSymbol = &virConnectCompareCPU;
#  endif

    ret = virConnectCompareCPUSymbol(conn,
                                     xmlDesc,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 4, 0)
    setVirError(err, "Function virConnectCompareHypervisorCPU compiled out (from 4.4.0)");
    return ret;
#else
    static virConnectCompareHypervisorCPUType virConnectCompareHypervisorCPUSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectCompareHypervisorCPUSymbol = libvirtSymbol(libvirt,
                                                                 "virConnectCompareHypervisorCPU",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectCompareHypervisorCPU");
        return ret;
    }
#  else
    virConnectCompareHypervisorCPUSymbol = &virConnectCompareHypervisorCPU;
#  endif

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
#endif
}

typedef int
(*virConnectDomainEventDeregisterType)(virConnectPtr conn,
                                       virConnectDomainEventCallback cb);

int
virConnectDomainEventDeregisterWrapper(virConnectPtr conn,
                                       virConnectDomainEventCallback cb,
                                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virConnectDomainEventDeregister compiled out (from 0.5.0)");
    return ret;
#else
    static virConnectDomainEventDeregisterType virConnectDomainEventDeregisterSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainEventDeregisterSymbol = libvirtSymbol(libvirt,
                                                                  "virConnectDomainEventDeregister",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainEventDeregister");
        return ret;
    }
#  else
    virConnectDomainEventDeregisterSymbol = &virConnectDomainEventDeregister;
#  endif

    ret = virConnectDomainEventDeregisterSymbol(conn,
                                                cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectDomainEventDeregisterAnyType)(virConnectPtr conn,
                                          int callbackID);

int
virConnectDomainEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectDomainEventDeregisterAny compiled out (from 0.8.0)");
    return ret;
#else
    static virConnectDomainEventDeregisterAnyType virConnectDomainEventDeregisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainEventDeregisterAnySymbol = libvirtSymbol(libvirt,
                                                                     "virConnectDomainEventDeregisterAny",
                                                                     &success,
                                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainEventDeregisterAny");
        return ret;
    }
#  else
    virConnectDomainEventDeregisterAnySymbol = &virConnectDomainEventDeregisterAny;
#  endif

    ret = virConnectDomainEventDeregisterAnySymbol(conn,
                                                   callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virConnectDomainEventRegister compiled out (from 0.5.0)");
    return ret;
#else
    static virConnectDomainEventRegisterType virConnectDomainEventRegisterSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainEventRegisterSymbol = libvirtSymbol(libvirt,
                                                                "virConnectDomainEventRegister",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainEventRegister");
        return ret;
    }
#  else
    virConnectDomainEventRegisterSymbol = &virConnectDomainEventRegister;
#  endif

    ret = virConnectDomainEventRegisterSymbol(conn,
                                              cb,
                                              opaque,
                                              freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectDomainEventRegisterAny compiled out (from 0.8.0)");
    return ret;
#else
    static virConnectDomainEventRegisterAnyType virConnectDomainEventRegisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainEventRegisterAnySymbol = libvirtSymbol(libvirt,
                                                                   "virConnectDomainEventRegisterAny",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainEventRegisterAny");
        return ret;
    }
#  else
    virConnectDomainEventRegisterAnySymbol = &virConnectDomainEventRegisterAny;
#  endif

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
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectDomainXMLFromNative compiled out (from 0.6.4)");
    return ret;
#else
    static virConnectDomainXMLFromNativeType virConnectDomainXMLFromNativeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainXMLFromNativeSymbol = libvirtSymbol(libvirt,
                                                                "virConnectDomainXMLFromNative",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainXMLFromNative");
        return ret;
    }
#  else
    virConnectDomainXMLFromNativeSymbol = &virConnectDomainXMLFromNative;
#  endif

    ret = virConnectDomainXMLFromNativeSymbol(conn,
                                              nativeFormat,
                                              nativeConfig,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectDomainXMLToNative compiled out (from 0.6.4)");
    return ret;
#else
    static virConnectDomainXMLToNativeType virConnectDomainXMLToNativeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainXMLToNativeSymbol = libvirtSymbol(libvirt,
                                                              "virConnectDomainXMLToNative",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainXMLToNative");
        return ret;
    }
#  else
    virConnectDomainXMLToNativeSymbol = &virConnectDomainXMLToNative;
#  endif

    ret = virConnectDomainXMLToNativeSymbol(conn,
                                            nativeFormat,
                                            domainXml,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 5)
    setVirError(err, "Function virConnectFindStoragePoolSources compiled out (from 0.4.5)");
    return ret;
#else
    static virConnectFindStoragePoolSourcesType virConnectFindStoragePoolSourcesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectFindStoragePoolSourcesSymbol = libvirtSymbol(libvirt,
                                                                   "virConnectFindStoragePoolSources",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectFindStoragePoolSources");
        return ret;
    }
#  else
    virConnectFindStoragePoolSourcesSymbol = &virConnectFindStoragePoolSources;
#  endif

    ret = virConnectFindStoragePoolSourcesSymbol(conn,
                                                 type,
                                                 srcSpec,
                                                 flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virConnectGetAllDomainStats compiled out (from 1.2.8)");
    return ret;
#else
    static virConnectGetAllDomainStatsType virConnectGetAllDomainStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetAllDomainStatsSymbol = libvirtSymbol(libvirt,
                                                              "virConnectGetAllDomainStats",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetAllDomainStats");
        return ret;
    }
#  else
    virConnectGetAllDomainStatsSymbol = &virConnectGetAllDomainStats;
#  endif

    ret = virConnectGetAllDomainStatsSymbol(conn,
                                            stats,
                                            retStats,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 1, 3)
    setVirError(err, "Function virConnectGetCPUModelNames compiled out (from 1.1.3)");
    return ret;
#else
    static virConnectGetCPUModelNamesType virConnectGetCPUModelNamesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetCPUModelNamesSymbol = libvirtSymbol(libvirt,
                                                             "virConnectGetCPUModelNames",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetCPUModelNames");
        return ret;
    }
#  else
    virConnectGetCPUModelNamesSymbol = &virConnectGetCPUModelNames;
#  endif

    ret = virConnectGetCPUModelNamesSymbol(conn,
                                           arch,
                                           models,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virConnectGetCapabilitiesType)(virConnectPtr conn);

char *
virConnectGetCapabilitiesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virConnectGetCapabilities compiled out (from 0.2.1)");
    return ret;
#else
    static virConnectGetCapabilitiesType virConnectGetCapabilitiesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetCapabilitiesSymbol = libvirtSymbol(libvirt,
                                                            "virConnectGetCapabilities",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetCapabilities");
        return ret;
    }
#  else
    virConnectGetCapabilitiesSymbol = &virConnectGetCapabilities;
#  endif

    ret = virConnectGetCapabilitiesSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 7)
    setVirError(err, "Function virConnectGetDomainCapabilities compiled out (from 1.2.7)");
    return ret;
#else
    static virConnectGetDomainCapabilitiesType virConnectGetDomainCapabilitiesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetDomainCapabilitiesSymbol = libvirtSymbol(libvirt,
                                                                  "virConnectGetDomainCapabilities",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetDomainCapabilities");
        return ret;
    }
#  else
    virConnectGetDomainCapabilitiesSymbol = &virConnectGetDomainCapabilities;
#  endif

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
#endif
}

typedef char *
(*virConnectGetHostnameType)(virConnectPtr conn);

char *
virConnectGetHostnameWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virConnectGetHostname compiled out (from 0.3.0)");
    return ret;
#else
    static virConnectGetHostnameType virConnectGetHostnameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetHostnameSymbol = libvirtSymbol(libvirt,
                                                        "virConnectGetHostname",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetHostname");
        return ret;
    }
#  else
    virConnectGetHostnameSymbol = &virConnectGetHostname;
#  endif

    ret = virConnectGetHostnameSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virConnectGetLibVersion compiled out (from 0.7.3)");
    return ret;
#else
    static virConnectGetLibVersionType virConnectGetLibVersionSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetLibVersionSymbol = libvirtSymbol(libvirt,
                                                          "virConnectGetLibVersion",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetLibVersion");
        return ret;
    }
#  else
    virConnectGetLibVersionSymbol = &virConnectGetLibVersion;
#  endif

    ret = virConnectGetLibVersionSymbol(conn,
                                        libVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virConnectGetMaxVcpus compiled out (from 0.2.1)");
    return ret;
#else
    static virConnectGetMaxVcpusType virConnectGetMaxVcpusSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetMaxVcpusSymbol = libvirtSymbol(libvirt,
                                                        "virConnectGetMaxVcpus",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetMaxVcpus");
        return ret;
    }
#  else
    virConnectGetMaxVcpusSymbol = &virConnectGetMaxVcpus;
#  endif

    ret = virConnectGetMaxVcpusSymbol(conn,
                                      type);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virConnectGetStoragePoolCapabilitiesType)(virConnectPtr conn,
                                            unsigned int flags);

char *
virConnectGetStoragePoolCapabilitiesWrapper(virConnectPtr conn,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 2, 0)
    setVirError(err, "Function virConnectGetStoragePoolCapabilities compiled out (from 5.2.0)");
    return ret;
#else
    static virConnectGetStoragePoolCapabilitiesType virConnectGetStoragePoolCapabilitiesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetStoragePoolCapabilitiesSymbol = libvirtSymbol(libvirt,
                                                                       "virConnectGetStoragePoolCapabilities",
                                                                       &success,
                                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetStoragePoolCapabilities");
        return ret;
    }
#  else
    virConnectGetStoragePoolCapabilitiesSymbol = &virConnectGetStoragePoolCapabilities;
#  endif

    ret = virConnectGetStoragePoolCapabilitiesSymbol(conn,
                                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 8)
    setVirError(err, "Function virConnectGetSysinfo compiled out (from 0.8.8)");
    return ret;
#else
    static virConnectGetSysinfoType virConnectGetSysinfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetSysinfoSymbol = libvirtSymbol(libvirt,
                                                       "virConnectGetSysinfo",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetSysinfo");
        return ret;
    }
#  else
    virConnectGetSysinfoSymbol = &virConnectGetSysinfo;
#  endif

    ret = virConnectGetSysinfoSymbol(conn,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virConnectGetTypeType)(virConnectPtr conn);

const char *
virConnectGetTypeWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectGetType compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectGetTypeType virConnectGetTypeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetTypeSymbol = libvirtSymbol(libvirt,
                                                    "virConnectGetType",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetType");
        return ret;
    }
#  else
    virConnectGetTypeSymbol = &virConnectGetType;
#  endif

    ret = virConnectGetTypeSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virConnectGetURIType)(virConnectPtr conn);

char *
virConnectGetURIWrapper(virConnectPtr conn,
                        virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virConnectGetURI compiled out (from 0.3.0)");
    return ret;
#else
    static virConnectGetURIType virConnectGetURISymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetURISymbol = libvirtSymbol(libvirt,
                                                   "virConnectGetURI",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetURI");
        return ret;
    }
#  else
    virConnectGetURISymbol = &virConnectGetURI;
#  endif

    ret = virConnectGetURISymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectGetVersion compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectGetVersionType virConnectGetVersionSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectGetVersionSymbol = libvirtSymbol(libvirt,
                                                       "virConnectGetVersion",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectGetVersion");
        return ret;
    }
#  else
    virConnectGetVersionSymbol = &virConnectGetVersion;
#  endif

    ret = virConnectGetVersionSymbol(conn,
                                     hvVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectIsAliveType)(virConnectPtr conn);

int
virConnectIsAliveWrapper(virConnectPtr conn,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virConnectIsAlive compiled out (from 0.9.8)");
    return ret;
#else
    static virConnectIsAliveType virConnectIsAliveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectIsAliveSymbol = libvirtSymbol(libvirt,
                                                    "virConnectIsAlive",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectIsAlive");
        return ret;
    }
#  else
    virConnectIsAliveSymbol = &virConnectIsAlive;
#  endif

    ret = virConnectIsAliveSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectIsEncryptedType)(virConnectPtr conn);

int
virConnectIsEncryptedWrapper(virConnectPtr conn,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virConnectIsEncrypted compiled out (from 0.7.3)");
    return ret;
#else
    static virConnectIsEncryptedType virConnectIsEncryptedSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectIsEncryptedSymbol = libvirtSymbol(libvirt,
                                                        "virConnectIsEncrypted",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectIsEncrypted");
        return ret;
    }
#  else
    virConnectIsEncryptedSymbol = &virConnectIsEncrypted;
#  endif

    ret = virConnectIsEncryptedSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectIsSecureType)(virConnectPtr conn);

int
virConnectIsSecureWrapper(virConnectPtr conn,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virConnectIsSecure compiled out (from 0.7.3)");
    return ret;
#else
    static virConnectIsSecureType virConnectIsSecureSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectIsSecureSymbol = libvirtSymbol(libvirt,
                                                     "virConnectIsSecure",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectIsSecure");
        return ret;
    }
#  else
    virConnectIsSecureSymbol = &virConnectIsSecure;
#  endif

    ret = virConnectIsSecureSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virConnectListAllDomains compiled out (from 0.9.13)");
    return ret;
#else
    static virConnectListAllDomainsType virConnectListAllDomainsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllDomainsSymbol = libvirtSymbol(libvirt,
                                                           "virConnectListAllDomains",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllDomains");
        return ret;
    }
#  else
    virConnectListAllDomainsSymbol = &virConnectListAllDomains;
#  endif

    ret = virConnectListAllDomainsSymbol(conn,
                                         domains,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllInterfaces compiled out (from 0.10.2)");
    return ret;
#else
    static virConnectListAllInterfacesType virConnectListAllInterfacesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllInterfacesSymbol = libvirtSymbol(libvirt,
                                                              "virConnectListAllInterfaces",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllInterfaces");
        return ret;
    }
#  else
    virConnectListAllInterfacesSymbol = &virConnectListAllInterfaces;
#  endif

    ret = virConnectListAllInterfacesSymbol(conn,
                                            ifaces,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virConnectListAllNWFilterBindings compiled out (from 4.5.0)");
    return ret;
#else
    static virConnectListAllNWFilterBindingsType virConnectListAllNWFilterBindingsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllNWFilterBindingsSymbol = libvirtSymbol(libvirt,
                                                                    "virConnectListAllNWFilterBindings",
                                                                    &success,
                                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllNWFilterBindings");
        return ret;
    }
#  else
    virConnectListAllNWFilterBindingsSymbol = &virConnectListAllNWFilterBindings;
#  endif

    ret = virConnectListAllNWFilterBindingsSymbol(conn,
                                                  bindings,
                                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllNWFilters compiled out (from 0.10.2)");
    return ret;
#else
    static virConnectListAllNWFiltersType virConnectListAllNWFiltersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllNWFiltersSymbol = libvirtSymbol(libvirt,
                                                             "virConnectListAllNWFilters",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllNWFilters");
        return ret;
    }
#  else
    virConnectListAllNWFiltersSymbol = &virConnectListAllNWFilters;
#  endif

    ret = virConnectListAllNWFiltersSymbol(conn,
                                           filters,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllNetworks compiled out (from 0.10.2)");
    return ret;
#else
    static virConnectListAllNetworksType virConnectListAllNetworksSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllNetworksSymbol = libvirtSymbol(libvirt,
                                                            "virConnectListAllNetworks",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllNetworks");
        return ret;
    }
#  else
    virConnectListAllNetworksSymbol = &virConnectListAllNetworks;
#  endif

    ret = virConnectListAllNetworksSymbol(conn,
                                          nets,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllNodeDevices compiled out (from 0.10.2)");
    return ret;
#else
    static virConnectListAllNodeDevicesType virConnectListAllNodeDevicesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllNodeDevicesSymbol = libvirtSymbol(libvirt,
                                                               "virConnectListAllNodeDevices",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllNodeDevices");
        return ret;
    }
#  else
    virConnectListAllNodeDevicesSymbol = &virConnectListAllNodeDevices;
#  endif

    ret = virConnectListAllNodeDevicesSymbol(conn,
                                             devices,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllSecrets compiled out (from 0.10.2)");
    return ret;
#else
    static virConnectListAllSecretsType virConnectListAllSecretsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllSecretsSymbol = libvirtSymbol(libvirt,
                                                           "virConnectListAllSecrets",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllSecrets");
        return ret;
    }
#  else
    virConnectListAllSecretsSymbol = &virConnectListAllSecrets;
#  endif

    ret = virConnectListAllSecretsSymbol(conn,
                                         secrets,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virConnectListAllStoragePools compiled out (from 0.10.2)");
    return ret;
#else
    static virConnectListAllStoragePoolsType virConnectListAllStoragePoolsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListAllStoragePoolsSymbol = libvirtSymbol(libvirt,
                                                                "virConnectListAllStoragePools",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListAllStoragePools");
        return ret;
    }
#  else
    virConnectListAllStoragePoolsSymbol = &virConnectListAllStoragePools;
#  endif

    ret = virConnectListAllStoragePoolsSymbol(conn,
                                              pools,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virConnectListDefinedDomains compiled out (from 0.1.1)");
    return ret;
#else
    static virConnectListDefinedDomainsType virConnectListDefinedDomainsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListDefinedDomainsSymbol = libvirtSymbol(libvirt,
                                                               "virConnectListDefinedDomains",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListDefinedDomains");
        return ret;
    }
#  else
    virConnectListDefinedDomainsSymbol = &virConnectListDefinedDomains;
#  endif

    ret = virConnectListDefinedDomainsSymbol(conn,
                                             names,
                                             maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 0)
    setVirError(err, "Function virConnectListDefinedInterfaces compiled out (from 0.7.0)");
    return ret;
#else
    static virConnectListDefinedInterfacesType virConnectListDefinedInterfacesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListDefinedInterfacesSymbol = libvirtSymbol(libvirt,
                                                                  "virConnectListDefinedInterfaces",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListDefinedInterfaces");
        return ret;
    }
#  else
    virConnectListDefinedInterfacesSymbol = &virConnectListDefinedInterfaces;
#  endif

    ret = virConnectListDefinedInterfacesSymbol(conn,
                                                names,
                                                maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectListDefinedNetworks compiled out (from 0.2.0)");
    return ret;
#else
    static virConnectListDefinedNetworksType virConnectListDefinedNetworksSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListDefinedNetworksSymbol = libvirtSymbol(libvirt,
                                                                "virConnectListDefinedNetworks",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListDefinedNetworks");
        return ret;
    }
#  else
    virConnectListDefinedNetworksSymbol = &virConnectListDefinedNetworks;
#  endif

    ret = virConnectListDefinedNetworksSymbol(conn,
                                              names,
                                              maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectListDefinedStoragePools compiled out (from 0.4.1)");
    return ret;
#else
    static virConnectListDefinedStoragePoolsType virConnectListDefinedStoragePoolsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListDefinedStoragePoolsSymbol = libvirtSymbol(libvirt,
                                                                    "virConnectListDefinedStoragePools",
                                                                    &success,
                                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListDefinedStoragePools");
        return ret;
    }
#  else
    virConnectListDefinedStoragePoolsSymbol = &virConnectListDefinedStoragePools;
#  endif

    ret = virConnectListDefinedStoragePoolsSymbol(conn,
                                                  names,
                                                  maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectListDomains compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectListDomainsType virConnectListDomainsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListDomainsSymbol = libvirtSymbol(libvirt,
                                                        "virConnectListDomains",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListDomains");
        return ret;
    }
#  else
    virConnectListDomainsSymbol = &virConnectListDomains;
#  endif

    ret = virConnectListDomainsSymbol(conn,
                                      ids,
                                      maxids);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectListInterfaces compiled out (from 0.6.4)");
    return ret;
#else
    static virConnectListInterfacesType virConnectListInterfacesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListInterfacesSymbol = libvirtSymbol(libvirt,
                                                           "virConnectListInterfaces",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListInterfaces");
        return ret;
    }
#  else
    virConnectListInterfacesSymbol = &virConnectListInterfaces;
#  endif

    ret = virConnectListInterfacesSymbol(conn,
                                         names,
                                         maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectListNWFilters compiled out (from 0.8.0)");
    return ret;
#else
    static virConnectListNWFiltersType virConnectListNWFiltersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListNWFiltersSymbol = libvirtSymbol(libvirt,
                                                          "virConnectListNWFilters",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListNWFilters");
        return ret;
    }
#  else
    virConnectListNWFiltersSymbol = &virConnectListNWFilters;
#  endif

    ret = virConnectListNWFiltersSymbol(conn,
                                        names,
                                        maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectListNetworks compiled out (from 0.2.0)");
    return ret;
#else
    static virConnectListNetworksType virConnectListNetworksSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListNetworksSymbol = libvirtSymbol(libvirt,
                                                         "virConnectListNetworks",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListNetworks");
        return ret;
    }
#  else
    virConnectListNetworksSymbol = &virConnectListNetworks;
#  endif

    ret = virConnectListNetworksSymbol(conn,
                                       names,
                                       maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virConnectListSecrets compiled out (from 0.7.1)");
    return ret;
#else
    static virConnectListSecretsType virConnectListSecretsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListSecretsSymbol = libvirtSymbol(libvirt,
                                                        "virConnectListSecrets",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListSecrets");
        return ret;
    }
#  else
    virConnectListSecretsSymbol = &virConnectListSecrets;
#  endif

    ret = virConnectListSecretsSymbol(conn,
                                      uuids,
                                      maxuuids);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectListStoragePools compiled out (from 0.4.1)");
    return ret;
#else
    static virConnectListStoragePoolsType virConnectListStoragePoolsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectListStoragePoolsSymbol = libvirtSymbol(libvirt,
                                                             "virConnectListStoragePools",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectListStoragePools");
        return ret;
    }
#  else
    virConnectListStoragePoolsSymbol = &virConnectListStoragePools;
#  endif

    ret = virConnectListStoragePoolsSymbol(conn,
                                           names,
                                           maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 1)
    setVirError(err, "Function virConnectNetworkEventDeregisterAny compiled out (from 1.2.1)");
    return ret;
#else
    static virConnectNetworkEventDeregisterAnyType virConnectNetworkEventDeregisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNetworkEventDeregisterAnySymbol = libvirtSymbol(libvirt,
                                                                      "virConnectNetworkEventDeregisterAny",
                                                                      &success,
                                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNetworkEventDeregisterAny");
        return ret;
    }
#  else
    virConnectNetworkEventDeregisterAnySymbol = &virConnectNetworkEventDeregisterAny;
#  endif

    ret = virConnectNetworkEventDeregisterAnySymbol(conn,
                                                    callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 1)
    setVirError(err, "Function virConnectNetworkEventRegisterAny compiled out (from 1.2.1)");
    return ret;
#else
    static virConnectNetworkEventRegisterAnyType virConnectNetworkEventRegisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNetworkEventRegisterAnySymbol = libvirtSymbol(libvirt,
                                                                    "virConnectNetworkEventRegisterAny",
                                                                    &success,
                                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNetworkEventRegisterAny");
        return ret;
    }
#  else
    virConnectNetworkEventRegisterAnySymbol = &virConnectNetworkEventRegisterAny;
#  endif

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
#endif
}

typedef int
(*virConnectNodeDeviceEventDeregisterAnyType)(virConnectPtr conn,
                                              int callbackID);

int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 2, 0)
    setVirError(err, "Function virConnectNodeDeviceEventDeregisterAny compiled out (from 2.2.0)");
    return ret;
#else
    static virConnectNodeDeviceEventDeregisterAnyType virConnectNodeDeviceEventDeregisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNodeDeviceEventDeregisterAnySymbol = libvirtSymbol(libvirt,
                                                                         "virConnectNodeDeviceEventDeregisterAny",
                                                                         &success,
                                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNodeDeviceEventDeregisterAny");
        return ret;
    }
#  else
    virConnectNodeDeviceEventDeregisterAnySymbol = &virConnectNodeDeviceEventDeregisterAny;
#  endif

    ret = virConnectNodeDeviceEventDeregisterAnySymbol(conn,
                                                       callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 2, 0)
    setVirError(err, "Function virConnectNodeDeviceEventRegisterAny compiled out (from 2.2.0)");
    return ret;
#else
    static virConnectNodeDeviceEventRegisterAnyType virConnectNodeDeviceEventRegisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNodeDeviceEventRegisterAnySymbol = libvirtSymbol(libvirt,
                                                                       "virConnectNodeDeviceEventRegisterAny",
                                                                       &success,
                                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNodeDeviceEventRegisterAny");
        return ret;
    }
#  else
    virConnectNodeDeviceEventRegisterAnySymbol = &virConnectNodeDeviceEventRegisterAny;
#  endif

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
#endif
}

typedef int
(*virConnectNumOfDefinedDomainsType)(virConnectPtr conn);

int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 5)
    setVirError(err, "Function virConnectNumOfDefinedDomains compiled out (from 0.1.5)");
    return ret;
#else
    static virConnectNumOfDefinedDomainsType virConnectNumOfDefinedDomainsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfDefinedDomainsSymbol = libvirtSymbol(libvirt,
                                                                "virConnectNumOfDefinedDomains",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfDefinedDomains");
        return ret;
    }
#  else
    virConnectNumOfDefinedDomainsSymbol = &virConnectNumOfDefinedDomains;
#  endif

    ret = virConnectNumOfDefinedDomainsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfDefinedInterfacesType)(virConnectPtr conn);

int
virConnectNumOfDefinedInterfacesWrapper(virConnectPtr conn,
                                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 0)
    setVirError(err, "Function virConnectNumOfDefinedInterfaces compiled out (from 0.7.0)");
    return ret;
#else
    static virConnectNumOfDefinedInterfacesType virConnectNumOfDefinedInterfacesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfDefinedInterfacesSymbol = libvirtSymbol(libvirt,
                                                                   "virConnectNumOfDefinedInterfaces",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfDefinedInterfaces");
        return ret;
    }
#  else
    virConnectNumOfDefinedInterfacesSymbol = &virConnectNumOfDefinedInterfaces;
#  endif

    ret = virConnectNumOfDefinedInterfacesSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfDefinedNetworksType)(virConnectPtr conn);

int
virConnectNumOfDefinedNetworksWrapper(virConnectPtr conn,
                                      virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectNumOfDefinedNetworks compiled out (from 0.2.0)");
    return ret;
#else
    static virConnectNumOfDefinedNetworksType virConnectNumOfDefinedNetworksSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfDefinedNetworksSymbol = libvirtSymbol(libvirt,
                                                                 "virConnectNumOfDefinedNetworks",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfDefinedNetworks");
        return ret;
    }
#  else
    virConnectNumOfDefinedNetworksSymbol = &virConnectNumOfDefinedNetworks;
#  endif

    ret = virConnectNumOfDefinedNetworksSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfDefinedStoragePoolsType)(virConnectPtr conn);

int
virConnectNumOfDefinedStoragePoolsWrapper(virConnectPtr conn,
                                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectNumOfDefinedStoragePools compiled out (from 0.4.1)");
    return ret;
#else
    static virConnectNumOfDefinedStoragePoolsType virConnectNumOfDefinedStoragePoolsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfDefinedStoragePoolsSymbol = libvirtSymbol(libvirt,
                                                                     "virConnectNumOfDefinedStoragePools",
                                                                     &success,
                                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfDefinedStoragePools");
        return ret;
    }
#  else
    virConnectNumOfDefinedStoragePoolsSymbol = &virConnectNumOfDefinedStoragePools;
#  endif

    ret = virConnectNumOfDefinedStoragePoolsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfDomainsType)(virConnectPtr conn);

int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectNumOfDomains compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectNumOfDomainsType virConnectNumOfDomainsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfDomainsSymbol = libvirtSymbol(libvirt,
                                                         "virConnectNumOfDomains",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfDomains");
        return ret;
    }
#  else
    virConnectNumOfDomainsSymbol = &virConnectNumOfDomains;
#  endif

    ret = virConnectNumOfDomainsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfInterfacesType)(virConnectPtr conn);

int
virConnectNumOfInterfacesWrapper(virConnectPtr conn,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectNumOfInterfaces compiled out (from 0.6.4)");
    return ret;
#else
    static virConnectNumOfInterfacesType virConnectNumOfInterfacesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfInterfacesSymbol = libvirtSymbol(libvirt,
                                                            "virConnectNumOfInterfaces",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfInterfaces");
        return ret;
    }
#  else
    virConnectNumOfInterfacesSymbol = &virConnectNumOfInterfaces;
#  endif

    ret = virConnectNumOfInterfacesSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfNWFiltersType)(virConnectPtr conn);

int
virConnectNumOfNWFiltersWrapper(virConnectPtr conn,
                                virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectNumOfNWFilters compiled out (from 0.8.0)");
    return ret;
#else
    static virConnectNumOfNWFiltersType virConnectNumOfNWFiltersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfNWFiltersSymbol = libvirtSymbol(libvirt,
                                                           "virConnectNumOfNWFilters",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfNWFilters");
        return ret;
    }
#  else
    virConnectNumOfNWFiltersSymbol = &virConnectNumOfNWFilters;
#  endif

    ret = virConnectNumOfNWFiltersSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfNetworksType)(virConnectPtr conn);

int
virConnectNumOfNetworksWrapper(virConnectPtr conn,
                               virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virConnectNumOfNetworks compiled out (from 0.2.0)");
    return ret;
#else
    static virConnectNumOfNetworksType virConnectNumOfNetworksSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfNetworksSymbol = libvirtSymbol(libvirt,
                                                          "virConnectNumOfNetworks",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfNetworks");
        return ret;
    }
#  else
    virConnectNumOfNetworksSymbol = &virConnectNumOfNetworks;
#  endif

    ret = virConnectNumOfNetworksSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfSecretsType)(virConnectPtr conn);

int
virConnectNumOfSecretsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virConnectNumOfSecrets compiled out (from 0.7.1)");
    return ret;
#else
    static virConnectNumOfSecretsType virConnectNumOfSecretsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfSecretsSymbol = libvirtSymbol(libvirt,
                                                         "virConnectNumOfSecrets",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfSecrets");
        return ret;
    }
#  else
    virConnectNumOfSecretsSymbol = &virConnectNumOfSecrets;
#  endif

    ret = virConnectNumOfSecretsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectNumOfStoragePoolsType)(virConnectPtr conn);

int
virConnectNumOfStoragePoolsWrapper(virConnectPtr conn,
                                   virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virConnectNumOfStoragePools compiled out (from 0.4.1)");
    return ret;
#else
    static virConnectNumOfStoragePoolsType virConnectNumOfStoragePoolsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectNumOfStoragePoolsSymbol = libvirtSymbol(libvirt,
                                                              "virConnectNumOfStoragePools",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectNumOfStoragePools");
        return ret;
    }
#  else
    virConnectNumOfStoragePoolsSymbol = &virConnectNumOfStoragePools;
#  endif

    ret = virConnectNumOfStoragePoolsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virConnectOpenType)(const char * name);

virConnectPtr
virConnectOpenWrapper(const char * name,
                      virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectOpen compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectOpenType virConnectOpenSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectOpenSymbol = libvirtSymbol(libvirt,
                                                 "virConnectOpen",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectOpen");
        return ret;
    }
#  else
    virConnectOpenSymbol = &virConnectOpen;
#  endif

    ret = virConnectOpenSymbol(name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 0)
    setVirError(err, "Function virConnectOpenAuth compiled out (from 0.4.0)");
    return ret;
#else
    static virConnectOpenAuthType virConnectOpenAuthSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectOpenAuthSymbol = libvirtSymbol(libvirt,
                                                     "virConnectOpenAuth",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectOpenAuth");
        return ret;
    }
#  else
    virConnectOpenAuthSymbol = &virConnectOpenAuth;
#  endif

    ret = virConnectOpenAuthSymbol(name,
                                   auth,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virConnectOpenReadOnlyType)(const char * name);

virConnectPtr
virConnectOpenReadOnlyWrapper(const char * name,
                              virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectOpenReadOnly compiled out (from 0.0.3)");
    return ret;
#else
    static virConnectOpenReadOnlyType virConnectOpenReadOnlySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectOpenReadOnlySymbol = libvirtSymbol(libvirt,
                                                         "virConnectOpenReadOnly",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectOpenReadOnly");
        return ret;
    }
#  else
    virConnectOpenReadOnlySymbol = &virConnectOpenReadOnly;
#  endif

    ret = virConnectOpenReadOnlySymbol(name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectRefType)(virConnectPtr conn);

int
virConnectRefWrapper(virConnectPtr conn,
                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virConnectRef compiled out (from 0.6.0)");
    return ret;
#else
    static virConnectRefType virConnectRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectRefSymbol = libvirtSymbol(libvirt,
                                                "virConnectRef",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectRef");
        return ret;
    }
#  else
    virConnectRefSymbol = &virConnectRef;
#  endif

    ret = virConnectRefSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virConnectRegisterCloseCallback compiled out (from 0.10.0)");
    return ret;
#else
    static virConnectRegisterCloseCallbackType virConnectRegisterCloseCallbackSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectRegisterCloseCallbackSymbol = libvirtSymbol(libvirt,
                                                                  "virConnectRegisterCloseCallback",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectRegisterCloseCallback");
        return ret;
    }
#  else
    virConnectRegisterCloseCallbackSymbol = &virConnectRegisterCloseCallback;
#  endif

    ret = virConnectRegisterCloseCallbackSymbol(conn,
                                                cb,
                                                opaque,
                                                freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virConnectSecretEventDeregisterAny compiled out (from 3.0.0)");
    return ret;
#else
    static virConnectSecretEventDeregisterAnyType virConnectSecretEventDeregisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectSecretEventDeregisterAnySymbol = libvirtSymbol(libvirt,
                                                                     "virConnectSecretEventDeregisterAny",
                                                                     &success,
                                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectSecretEventDeregisterAny");
        return ret;
    }
#  else
    virConnectSecretEventDeregisterAnySymbol = &virConnectSecretEventDeregisterAny;
#  endif

    ret = virConnectSecretEventDeregisterAnySymbol(conn,
                                                   callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virConnectSecretEventRegisterAny compiled out (from 3.0.0)");
    return ret;
#else
    static virConnectSecretEventRegisterAnyType virConnectSecretEventRegisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectSecretEventRegisterAnySymbol = libvirtSymbol(libvirt,
                                                                   "virConnectSecretEventRegisterAny",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectSecretEventRegisterAny");
        return ret;
    }
#  else
    virConnectSecretEventRegisterAnySymbol = &virConnectSecretEventRegisterAny;
#  endif

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
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 8, 0)
    setVirError(err, "Function virConnectSetIdentity compiled out (from 5.8.0)");
    return ret;
#else
    static virConnectSetIdentityType virConnectSetIdentitySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectSetIdentitySymbol = libvirtSymbol(libvirt,
                                                        "virConnectSetIdentity",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectSetIdentity");
        return ret;
    }
#  else
    virConnectSetIdentitySymbol = &virConnectSetIdentity;
#  endif

    ret = virConnectSetIdentitySymbol(conn,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virConnectSetKeepAlive compiled out (from 0.9.8)");
    return ret;
#else
    static virConnectSetKeepAliveType virConnectSetKeepAliveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectSetKeepAliveSymbol = libvirtSymbol(libvirt,
                                                         "virConnectSetKeepAlive",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectSetKeepAlive");
        return ret;
    }
#  else
    virConnectSetKeepAliveSymbol = &virConnectSetKeepAlive;
#  endif

    ret = virConnectSetKeepAliveSymbol(conn,
                                       interval,
                                       count);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectStoragePoolEventDeregisterAnyType)(virConnectPtr conn,
                                               int callbackID);

int
virConnectStoragePoolEventDeregisterAnyWrapper(virConnectPtr conn,
                                               int callbackID,
                                               virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virConnectStoragePoolEventDeregisterAny compiled out (from 2.0.0)");
    return ret;
#else
    static virConnectStoragePoolEventDeregisterAnyType virConnectStoragePoolEventDeregisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectStoragePoolEventDeregisterAnySymbol = libvirtSymbol(libvirt,
                                                                          "virConnectStoragePoolEventDeregisterAny",
                                                                          &success,
                                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectStoragePoolEventDeregisterAny");
        return ret;
    }
#  else
    virConnectStoragePoolEventDeregisterAnySymbol = &virConnectStoragePoolEventDeregisterAny;
#  endif

    ret = virConnectStoragePoolEventDeregisterAnySymbol(conn,
                                                        callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virConnectStoragePoolEventRegisterAny compiled out (from 2.0.0)");
    return ret;
#else
    static virConnectStoragePoolEventRegisterAnyType virConnectStoragePoolEventRegisterAnySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectStoragePoolEventRegisterAnySymbol = libvirtSymbol(libvirt,
                                                                        "virConnectStoragePoolEventRegisterAny",
                                                                        &success,
                                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectStoragePoolEventRegisterAny");
        return ret;
    }
#  else
    virConnectStoragePoolEventRegisterAnySymbol = &virConnectStoragePoolEventRegisterAny;
#  endif

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
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virConnectUnregisterCloseCallback compiled out (from 0.10.0)");
    return ret;
#else
    static virConnectUnregisterCloseCallbackType virConnectUnregisterCloseCallbackSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectUnregisterCloseCallbackSymbol = libvirtSymbol(libvirt,
                                                                    "virConnectUnregisterCloseCallback",
                                                                    &success,
                                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectUnregisterCloseCallback");
        return ret;
    }
#  else
    virConnectUnregisterCloseCallbackSymbol = &virConnectUnregisterCloseCallback;
#  endif

    ret = virConnectUnregisterCloseCallbackSymbol(conn,
                                                  cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virDefaultErrorFuncType)(virErrorPtr err);

void
virDefaultErrorFuncWrapper(virErrorPtr err)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virDefaultErrorFunc compiled out (from 0.1.0)");
    return;
#else
    static virDefaultErrorFuncType virDefaultErrorFuncSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virDefaultErrorFuncSymbol = libvirtSymbol(libvirt,
                                                      "virDefaultErrorFunc",
                                                      &success,
                                                      NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virDefaultErrorFunc");
        return;
    }
#  else
    virDefaultErrorFuncSymbol = &virDefaultErrorFunc;
#  endif

    virDefaultErrorFuncSymbol(err);
#endif
}

typedef int
(*virDomainAbortJobType)(virDomainPtr domain);

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainAbortJob compiled out (from 0.7.7)");
    return ret;
#else
    static virDomainAbortJobType virDomainAbortJobSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAbortJobSymbol = libvirtSymbol(libvirt,
                                                    "virDomainAbortJob",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAbortJob");
        return ret;
    }
#  else
    virDomainAbortJobSymbol = &virDomainAbortJob;
#  endif

    ret = virDomainAbortJobSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 15)
    setVirError(err, "Function virDomainAddIOThread compiled out (from 1.2.15)");
    return ret;
#else
    static virDomainAddIOThreadType virDomainAddIOThreadSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAddIOThreadSymbol = libvirtSymbol(libvirt,
                                                       "virDomainAddIOThread",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAddIOThread");
        return ret;
    }
#  else
    virDomainAddIOThreadSymbol = &virDomainAddIOThread;
#  endif

    ret = virDomainAddIOThreadSymbol(domain,
                                     iothread_id,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 10, 0)
    setVirError(err, "Function virDomainAgentSetResponseTimeout compiled out (from 5.10.0)");
    return ret;
#else
    static virDomainAgentSetResponseTimeoutType virDomainAgentSetResponseTimeoutSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAgentSetResponseTimeoutSymbol = libvirtSymbol(libvirt,
                                                                   "virDomainAgentSetResponseTimeout",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAgentSetResponseTimeout");
        return ret;
    }
#  else
    virDomainAgentSetResponseTimeoutSymbol = &virDomainAgentSetResponseTimeout;
#  endif

    ret = virDomainAgentSetResponseTimeoutSymbol(domain,
                                                 timeout,
                                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainAttachDeviceType)(virDomainPtr domain,
                             const char * xml);

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 9)
    setVirError(err, "Function virDomainAttachDevice compiled out (from 0.1.9)");
    return ret;
#else
    static virDomainAttachDeviceType virDomainAttachDeviceSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAttachDeviceSymbol = libvirtSymbol(libvirt,
                                                        "virDomainAttachDevice",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAttachDevice");
        return ret;
    }
#  else
    virDomainAttachDeviceSymbol = &virDomainAttachDevice;
#  endif

    ret = virDomainAttachDeviceSymbol(domain,
                                      xml);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainAttachDeviceFlags compiled out (from 0.7.7)");
    return ret;
#else
    static virDomainAttachDeviceFlagsType virDomainAttachDeviceFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAttachDeviceFlagsSymbol = libvirtSymbol(libvirt,
                                                             "virDomainAttachDeviceFlags",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAttachDeviceFlags");
        return ret;
    }
#  else
    virDomainAttachDeviceFlagsSymbol = &virDomainAttachDeviceFlags;
#  endif

    ret = virDomainAttachDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainAuthorizedSSHKeysGetType)(virDomainPtr domain,
                                     const char * user,
                                     char *** keys,
                                     unsigned int flags);

int
virDomainAuthorizedSSHKeysGetWrapper(virDomainPtr domain,
                                     const char * user,
                                     char *** keys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(6, 10, 0)
    setVirError(err, "Function virDomainAuthorizedSSHKeysGet compiled out (from 6.10.0)");
    return ret;
#else
    static virDomainAuthorizedSSHKeysGetType virDomainAuthorizedSSHKeysGetSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAuthorizedSSHKeysGetSymbol = libvirtSymbol(libvirt,
                                                                "virDomainAuthorizedSSHKeysGet",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAuthorizedSSHKeysGet");
        return ret;
    }
#  else
    virDomainAuthorizedSSHKeysGetSymbol = &virDomainAuthorizedSSHKeysGet;
#  endif

    ret = virDomainAuthorizedSSHKeysGetSymbol(domain,
                                              user,
                                              keys,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(6, 10, 0)
    setVirError(err, "Function virDomainAuthorizedSSHKeysSet compiled out (from 6.10.0)");
    return ret;
#else
    static virDomainAuthorizedSSHKeysSetType virDomainAuthorizedSSHKeysSetSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainAuthorizedSSHKeysSetSymbol = libvirtSymbol(libvirt,
                                                                "virDomainAuthorizedSSHKeysSet",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainAuthorizedSSHKeysSet");
        return ret;
    }
#  else
    virDomainAuthorizedSSHKeysSetSymbol = &virDomainAuthorizedSSHKeysSet;
#  endif

    ret = virDomainAuthorizedSSHKeysSetSymbol(domain,
                                              user,
                                              keys,
                                              nkeys,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(6, 0, 0)
    setVirError(err, "Function virDomainBackupBegin compiled out (from 6.0.0)");
    return ret;
#else
    static virDomainBackupBeginType virDomainBackupBeginSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBackupBeginSymbol = libvirtSymbol(libvirt,
                                                       "virDomainBackupBegin",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBackupBegin");
        return ret;
    }
#  else
    virDomainBackupBeginSymbol = &virDomainBackupBegin;
#  endif

    ret = virDomainBackupBeginSymbol(domain,
                                     backupXML,
                                     checkpointXML,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainBackupGetXMLDescType)(virDomainPtr domain,
                                 unsigned int flags);

char *
virDomainBackupGetXMLDescWrapper(virDomainPtr domain,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(6, 0, 0)
    setVirError(err, "Function virDomainBackupGetXMLDesc compiled out (from 6.0.0)");
    return ret;
#else
    static virDomainBackupGetXMLDescType virDomainBackupGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBackupGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                            "virDomainBackupGetXMLDesc",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBackupGetXMLDesc");
        return ret;
    }
#  else
    virDomainBackupGetXMLDescSymbol = &virDomainBackupGetXMLDesc;
#  endif

    ret = virDomainBackupGetXMLDescSymbol(domain,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virDomainBlockCommit compiled out (from 0.10.2)");
    return ret;
#else
    static virDomainBlockCommitType virDomainBlockCommitSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockCommitSymbol = libvirtSymbol(libvirt,
                                                       "virDomainBlockCommit",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockCommit");
        return ret;
    }
#  else
    virDomainBlockCommitSymbol = &virDomainBlockCommit;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virDomainBlockCopy compiled out (from 1.2.8)");
    return ret;
#else
    static virDomainBlockCopyType virDomainBlockCopySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockCopySymbol = libvirtSymbol(libvirt,
                                                     "virDomainBlockCopy",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockCopy");
        return ret;
    }
#  else
    virDomainBlockCopySymbol = &virDomainBlockCopy;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainBlockJobAbort compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainBlockJobAbortType virDomainBlockJobAbortSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockJobAbortSymbol = libvirtSymbol(libvirt,
                                                         "virDomainBlockJobAbort",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockJobAbort");
        return ret;
    }
#  else
    virDomainBlockJobAbortSymbol = &virDomainBlockJobAbort;
#  endif

    ret = virDomainBlockJobAbortSymbol(dom,
                                       disk,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainBlockJobSetSpeed compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainBlockJobSetSpeedType virDomainBlockJobSetSpeedSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockJobSetSpeedSymbol = libvirtSymbol(libvirt,
                                                            "virDomainBlockJobSetSpeed",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockJobSetSpeed");
        return ret;
    }
#  else
    virDomainBlockJobSetSpeedSymbol = &virDomainBlockJobSetSpeed;
#  endif

    ret = virDomainBlockJobSetSpeedSymbol(dom,
                                          disk,
                                          bandwidth,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 2)
    setVirError(err, "Function virDomainBlockPeek compiled out (from 0.4.2)");
    return ret;
#else
    static virDomainBlockPeekType virDomainBlockPeekSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockPeekSymbol = libvirtSymbol(libvirt,
                                                     "virDomainBlockPeek",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockPeek");
        return ret;
    }
#  else
    virDomainBlockPeekSymbol = &virDomainBlockPeek;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainBlockPull compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainBlockPullType virDomainBlockPullSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockPullSymbol = libvirtSymbol(libvirt,
                                                     "virDomainBlockPull",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockPull");
        return ret;
    }
#  else
    virDomainBlockPullSymbol = &virDomainBlockPull;
#  endif

    ret = virDomainBlockPullSymbol(dom,
                                   disk,
                                   bandwidth,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainBlockRebase compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainBlockRebaseType virDomainBlockRebaseSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockRebaseSymbol = libvirtSymbol(libvirt,
                                                       "virDomainBlockRebase",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockRebase");
        return ret;
    }
#  else
    virDomainBlockRebaseSymbol = &virDomainBlockRebase;
#  endif

    ret = virDomainBlockRebaseSymbol(dom,
                                     disk,
                                     base,
                                     bandwidth,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virDomainBlockResize compiled out (from 0.9.8)");
    return ret;
#else
    static virDomainBlockResizeType virDomainBlockResizeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockResizeSymbol = libvirtSymbol(libvirt,
                                                       "virDomainBlockResize",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockResize");
        return ret;
    }
#  else
    virDomainBlockResizeSymbol = &virDomainBlockResize;
#  endif

    ret = virDomainBlockResizeSymbol(dom,
                                     disk,
                                     size,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 2)
    setVirError(err, "Function virDomainBlockStats compiled out (from 0.3.2)");
    return ret;
#else
    static virDomainBlockStatsType virDomainBlockStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockStatsSymbol = libvirtSymbol(libvirt,
                                                      "virDomainBlockStats",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockStats");
        return ret;
    }
#  else
    virDomainBlockStatsSymbol = &virDomainBlockStats;
#  endif

    ret = virDomainBlockStatsSymbol(dom,
                                    disk,
                                    stats,
                                    size);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainBlockStatsFlags compiled out (from 0.9.5)");
    return ret;
#else
    static virDomainBlockStatsFlagsType virDomainBlockStatsFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainBlockStatsFlagsSymbol = libvirtSymbol(libvirt,
                                                           "virDomainBlockStatsFlags",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainBlockStatsFlags");
        return ret;
    }
#  else
    virDomainBlockStatsFlagsSymbol = &virDomainBlockStatsFlags;
#  endif

    ret = virDomainBlockStatsFlagsSymbol(dom,
                                         disk,
                                         params,
                                         nparams,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainCheckpointPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointCreateXML compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointCreateXMLType virDomainCheckpointCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointCreateXMLSymbol = libvirtSymbol(libvirt,
                                                               "virDomainCheckpointCreateXML",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointCreateXML");
        return ret;
    }
#  else
    virDomainCheckpointCreateXMLSymbol = &virDomainCheckpointCreateXML;
#  endif

    ret = virDomainCheckpointCreateXMLSymbol(domain,
                                             xmlDesc,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainCheckpointDeleteType)(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags);

int
virDomainCheckpointDeleteWrapper(virDomainCheckpointPtr checkpoint,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointDelete compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointDeleteType virDomainCheckpointDeleteSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointDeleteSymbol = libvirtSymbol(libvirt,
                                                            "virDomainCheckpointDelete",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointDelete");
        return ret;
    }
#  else
    virDomainCheckpointDeleteSymbol = &virDomainCheckpointDelete;
#  endif

    ret = virDomainCheckpointDeleteSymbol(checkpoint,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainCheckpointFreeType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointFreeWrapper(virDomainCheckpointPtr checkpoint,
                               virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointFree compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointFreeType virDomainCheckpointFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointFreeSymbol = libvirtSymbol(libvirt,
                                                          "virDomainCheckpointFree",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointFree");
        return ret;
    }
#  else
    virDomainCheckpointFreeSymbol = &virDomainCheckpointFree;
#  endif

    ret = virDomainCheckpointFreeSymbol(checkpoint);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virDomainCheckpointGetConnectType)(virDomainCheckpointPtr checkpoint);

virConnectPtr
virDomainCheckpointGetConnectWrapper(virDomainCheckpointPtr checkpoint,
                                     virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetConnect compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointGetConnectType virDomainCheckpointGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointGetConnectSymbol = libvirtSymbol(libvirt,
                                                                "virDomainCheckpointGetConnect",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointGetConnect");
        return ret;
    }
#  else
    virDomainCheckpointGetConnectSymbol = &virDomainCheckpointGetConnect;
#  endif

    ret = virDomainCheckpointGetConnectSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainCheckpointGetDomainType)(virDomainCheckpointPtr checkpoint);

virDomainPtr
virDomainCheckpointGetDomainWrapper(virDomainCheckpointPtr checkpoint,
                                    virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetDomain compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointGetDomainType virDomainCheckpointGetDomainSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointGetDomainSymbol = libvirtSymbol(libvirt,
                                                               "virDomainCheckpointGetDomain",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointGetDomain");
        return ret;
    }
#  else
    virDomainCheckpointGetDomainSymbol = &virDomainCheckpointGetDomain;
#  endif

    ret = virDomainCheckpointGetDomainSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virDomainCheckpointGetNameType)(virDomainCheckpointPtr checkpoint);

const char *
virDomainCheckpointGetNameWrapper(virDomainCheckpointPtr checkpoint,
                                  virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetName compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointGetNameType virDomainCheckpointGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointGetNameSymbol = libvirtSymbol(libvirt,
                                                             "virDomainCheckpointGetName",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointGetName");
        return ret;
    }
#  else
    virDomainCheckpointGetNameSymbol = &virDomainCheckpointGetName;
#  endif

    ret = virDomainCheckpointGetNameSymbol(checkpoint);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainCheckpointPtr
(*virDomainCheckpointGetParentType)(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags);

virDomainCheckpointPtr
virDomainCheckpointGetParentWrapper(virDomainCheckpointPtr checkpoint,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    virDomainCheckpointPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetParent compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointGetParentType virDomainCheckpointGetParentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointGetParentSymbol = libvirtSymbol(libvirt,
                                                               "virDomainCheckpointGetParent",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointGetParent");
        return ret;
    }
#  else
    virDomainCheckpointGetParentSymbol = &virDomainCheckpointGetParent;
#  endif

    ret = virDomainCheckpointGetParentSymbol(checkpoint,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainCheckpointGetXMLDescType)(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags);

char *
virDomainCheckpointGetXMLDescWrapper(virDomainCheckpointPtr checkpoint,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointGetXMLDesc compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointGetXMLDescType virDomainCheckpointGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                                "virDomainCheckpointGetXMLDesc",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointGetXMLDesc");
        return ret;
    }
#  else
    virDomainCheckpointGetXMLDescSymbol = &virDomainCheckpointGetXMLDesc;
#  endif

    ret = virDomainCheckpointGetXMLDescSymbol(checkpoint,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointListAllChildren compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointListAllChildrenType virDomainCheckpointListAllChildrenSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointListAllChildrenSymbol = libvirtSymbol(libvirt,
                                                                     "virDomainCheckpointListAllChildren",
                                                                     &success,
                                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointListAllChildren");
        return ret;
    }
#  else
    virDomainCheckpointListAllChildrenSymbol = &virDomainCheckpointListAllChildren;
#  endif

    ret = virDomainCheckpointListAllChildrenSymbol(checkpoint,
                                                   children,
                                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainCheckpointPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointLookupByName compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointLookupByNameType virDomainCheckpointLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointLookupByNameSymbol = libvirtSymbol(libvirt,
                                                                  "virDomainCheckpointLookupByName",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointLookupByName");
        return ret;
    }
#  else
    virDomainCheckpointLookupByNameSymbol = &virDomainCheckpointLookupByName;
#  endif

    ret = virDomainCheckpointLookupByNameSymbol(domain,
                                                name,
                                                flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainCheckpointRefType)(virDomainCheckpointPtr checkpoint);

int
virDomainCheckpointRefWrapper(virDomainCheckpointPtr checkpoint,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainCheckpointRef compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainCheckpointRefType virDomainCheckpointRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCheckpointRefSymbol = libvirtSymbol(libvirt,
                                                         "virDomainCheckpointRef",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCheckpointRef");
        return ret;
    }
#  else
    virDomainCheckpointRefSymbol = &virDomainCheckpointRef;
#  endif

    ret = virDomainCheckpointRefSymbol(checkpoint);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 9)
    setVirError(err, "Function virDomainCoreDump compiled out (from 0.1.9)");
    return ret;
#else
    static virDomainCoreDumpType virDomainCoreDumpSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCoreDumpSymbol = libvirtSymbol(libvirt,
                                                    "virDomainCoreDump",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCoreDump");
        return ret;
    }
#  else
    virDomainCoreDumpSymbol = &virDomainCoreDump;
#  endif

    ret = virDomainCoreDumpSymbol(domain,
                                  to,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 3)
    setVirError(err, "Function virDomainCoreDumpWithFormat compiled out (from 1.2.3)");
    return ret;
#else
    static virDomainCoreDumpWithFormatType virDomainCoreDumpWithFormatSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCoreDumpWithFormatSymbol = libvirtSymbol(libvirt,
                                                              "virDomainCoreDumpWithFormat",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCoreDumpWithFormat");
        return ret;
    }
#  else
    virDomainCoreDumpWithFormatSymbol = &virDomainCoreDumpWithFormat;
#  endif

    ret = virDomainCoreDumpWithFormatSymbol(domain,
                                            to,
                                            dumpformat,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainCreateType)(virDomainPtr domain);

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainCreate compiled out (from 0.1.1)");
    return ret;
#else
    static virDomainCreateType virDomainCreateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCreateSymbol = libvirtSymbol(libvirt,
                                                  "virDomainCreate",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCreate");
        return ret;
    }
#  else
    virDomainCreateSymbol = &virDomainCreate;
#  endif

    ret = virDomainCreateSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainCreateLinux compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainCreateLinuxType virDomainCreateLinuxSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCreateLinuxSymbol = libvirtSymbol(libvirt,
                                                       "virDomainCreateLinux",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCreateLinux");
        return ret;
    }
#  else
    virDomainCreateLinuxSymbol = &virDomainCreateLinux;
#  endif

    ret = virDomainCreateLinuxSymbol(conn,
                                     xmlDesc,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 1, 1)
    setVirError(err, "Function virDomainCreateWithFiles compiled out (from 1.1.1)");
    return ret;
#else
    static virDomainCreateWithFilesType virDomainCreateWithFilesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCreateWithFilesSymbol = libvirtSymbol(libvirt,
                                                           "virDomainCreateWithFiles",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCreateWithFiles");
        return ret;
    }
#  else
    virDomainCreateWithFilesSymbol = &virDomainCreateWithFiles;
#  endif

    ret = virDomainCreateWithFilesSymbol(domain,
                                         nfiles,
                                         files,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainCreateWithFlagsType)(virDomainPtr domain,
                                unsigned int flags);

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 2)
    setVirError(err, "Function virDomainCreateWithFlags compiled out (from 0.8.2)");
    return ret;
#else
    static virDomainCreateWithFlagsType virDomainCreateWithFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCreateWithFlagsSymbol = libvirtSymbol(libvirt,
                                                           "virDomainCreateWithFlags",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCreateWithFlags");
        return ret;
    }
#  else
    virDomainCreateWithFlagsSymbol = &virDomainCreateWithFlags;
#  endif

    ret = virDomainCreateWithFlagsSymbol(domain,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virDomainCreateXML compiled out (from 0.5.0)");
    return ret;
#else
    static virDomainCreateXMLType virDomainCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCreateXMLSymbol = libvirtSymbol(libvirt,
                                                     "virDomainCreateXML",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCreateXML");
        return ret;
    }
#  else
    virDomainCreateXMLSymbol = &virDomainCreateXML;
#  endif

    ret = virDomainCreateXMLSymbol(conn,
                                   xmlDesc,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 1, 1)
    setVirError(err, "Function virDomainCreateXMLWithFiles compiled out (from 1.1.1)");
    return ret;
#else
    static virDomainCreateXMLWithFilesType virDomainCreateXMLWithFilesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainCreateXMLWithFilesSymbol = libvirtSymbol(libvirt,
                                                              "virDomainCreateXMLWithFiles",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainCreateXMLWithFiles");
        return ret;
    }
#  else
    virDomainCreateXMLWithFilesSymbol = &virDomainCreateXMLWithFiles;
#  endif

    ret = virDomainCreateXMLWithFilesSymbol(conn,
                                            xmlDesc,
                                            nfiles,
                                            files,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainDefineXMLType)(virConnectPtr conn,
                          const char * xml);

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainDefineXML compiled out (from 0.1.1)");
    return ret;
#else
    static virDomainDefineXMLType virDomainDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDefineXMLSymbol = libvirtSymbol(libvirt,
                                                     "virDomainDefineXML",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDefineXML");
        return ret;
    }
#  else
    virDomainDefineXMLSymbol = &virDomainDefineXML;
#  endif

    ret = virDomainDefineXMLSymbol(conn,
                                   xml);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 12)
    setVirError(err, "Function virDomainDefineXMLFlags compiled out (from 1.2.12)");
    return ret;
#else
    static virDomainDefineXMLFlagsType virDomainDefineXMLFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDefineXMLFlagsSymbol = libvirtSymbol(libvirt,
                                                          "virDomainDefineXMLFlags",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDefineXMLFlags");
        return ret;
    }
#  else
    virDomainDefineXMLFlagsSymbol = &virDomainDefineXMLFlags;
#  endif

    ret = virDomainDefineXMLFlagsSymbol(conn,
                                        xml,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 15)
    setVirError(err, "Function virDomainDelIOThread compiled out (from 1.2.15)");
    return ret;
#else
    static virDomainDelIOThreadType virDomainDelIOThreadSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDelIOThreadSymbol = libvirtSymbol(libvirt,
                                                       "virDomainDelIOThread",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDelIOThread");
        return ret;
    }
#  else
    virDomainDelIOThreadSymbol = &virDomainDelIOThread;
#  endif

    ret = virDomainDelIOThreadSymbol(domain,
                                     iothread_id,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainDestroyType)(virDomainPtr domain);

int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainDestroy compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainDestroyType virDomainDestroySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDestroySymbol = libvirtSymbol(libvirt,
                                                   "virDomainDestroy",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDestroy");
        return ret;
    }
#  else
    virDomainDestroySymbol = &virDomainDestroy;
#  endif

    ret = virDomainDestroySymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainDestroyFlagsType)(virDomainPtr domain,
                             unsigned int flags);

int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainDestroyFlags compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainDestroyFlagsType virDomainDestroyFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDestroyFlagsSymbol = libvirtSymbol(libvirt,
                                                        "virDomainDestroyFlags",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDestroyFlags");
        return ret;
    }
#  else
    virDomainDestroyFlagsSymbol = &virDomainDestroyFlags;
#  endif

    ret = virDomainDestroyFlagsSymbol(domain,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainDetachDeviceType)(virDomainPtr domain,
                             const char * xml);

int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 9)
    setVirError(err, "Function virDomainDetachDevice compiled out (from 0.1.9)");
    return ret;
#else
    static virDomainDetachDeviceType virDomainDetachDeviceSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDetachDeviceSymbol = libvirtSymbol(libvirt,
                                                        "virDomainDetachDevice",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDetachDevice");
        return ret;
    }
#  else
    virDomainDetachDeviceSymbol = &virDomainDetachDevice;
#  endif

    ret = virDomainDetachDeviceSymbol(domain,
                                      xml);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 4, 0)
    setVirError(err, "Function virDomainDetachDeviceAlias compiled out (from 4.4.0)");
    return ret;
#else
    static virDomainDetachDeviceAliasType virDomainDetachDeviceAliasSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDetachDeviceAliasSymbol = libvirtSymbol(libvirt,
                                                             "virDomainDetachDeviceAlias",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDetachDeviceAlias");
        return ret;
    }
#  else
    virDomainDetachDeviceAliasSymbol = &virDomainDetachDeviceAlias;
#  endif

    ret = virDomainDetachDeviceAliasSymbol(domain,
                                           alias,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainDetachDeviceFlags compiled out (from 0.7.7)");
    return ret;
#else
    static virDomainDetachDeviceFlagsType virDomainDetachDeviceFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainDetachDeviceFlagsSymbol = libvirtSymbol(libvirt,
                                                             "virDomainDetachDeviceFlags",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainDetachDeviceFlags");
        return ret;
    }
#  else
    virDomainDetachDeviceFlagsSymbol = &virDomainDetachDeviceFlags;
#  endif

    ret = virDomainDetachDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainFSFreeze compiled out (from 1.2.5)");
    return ret;
#else
    static virDomainFSFreezeType virDomainFSFreezeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainFSFreezeSymbol = libvirtSymbol(libvirt,
                                                    "virDomainFSFreeze",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainFSFreeze");
        return ret;
    }
#  else
    virDomainFSFreezeSymbol = &virDomainFSFreeze;
#  endif

    ret = virDomainFSFreezeSymbol(dom,
                                  mountpoints,
                                  nmountpoints,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virDomainFSInfoFreeType)(virDomainFSInfoPtr info);

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 11)
    setVirError(NULL, "Function virDomainFSInfoFree compiled out (from 1.2.11)");
    return;
#else
    static virDomainFSInfoFreeType virDomainFSInfoFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virDomainFSInfoFreeSymbol = libvirtSymbol(libvirt,
                                                      "virDomainFSInfoFree",
                                                      &success,
                                                      NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virDomainFSInfoFree");
        return;
    }
#  else
    virDomainFSInfoFreeSymbol = &virDomainFSInfoFree;
#  endif

    virDomainFSInfoFreeSymbol(info);
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainFSThaw compiled out (from 1.2.5)");
    return ret;
#else
    static virDomainFSThawType virDomainFSThawSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainFSThawSymbol = libvirtSymbol(libvirt,
                                                  "virDomainFSThaw",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainFSThaw");
        return ret;
    }
#  else
    virDomainFSThawSymbol = &virDomainFSThaw;
#  endif

    ret = virDomainFSThawSymbol(dom,
                                mountpoints,
                                nmountpoints,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 1)
    setVirError(err, "Function virDomainFSTrim compiled out (from 1.0.1)");
    return ret;
#else
    static virDomainFSTrimType virDomainFSTrimSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainFSTrimSymbol = libvirtSymbol(libvirt,
                                                  "virDomainFSTrim",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainFSTrim");
        return ret;
    }
#  else
    virDomainFSTrimSymbol = &virDomainFSTrim;
#  endif

    ret = virDomainFSTrimSymbol(dom,
                                mountPoint,
                                minimum,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainFreeType)(virDomainPtr domain);

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainFree compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainFreeType virDomainFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainFreeSymbol = libvirtSymbol(libvirt,
                                                "virDomainFree",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainFree");
        return ret;
    }
#  else
    virDomainFreeSymbol = &virDomainFree;
#  endif

    ret = virDomainFreeSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetAutostartType)(virDomainPtr domain,
                             int * autostart);

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int * autostart,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virDomainGetAutostart compiled out (from 0.2.1)");
    return ret;
#else
    static virDomainGetAutostartType virDomainGetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetAutostartSymbol = libvirtSymbol(libvirt,
                                                        "virDomainGetAutostart",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetAutostart");
        return ret;
    }
#  else
    virDomainGetAutostartSymbol = &virDomainGetAutostart;
#  endif

    ret = virDomainGetAutostartSymbol(domain,
                                      autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainGetBlkioParameters compiled out (from 0.9.0)");
    return ret;
#else
    static virDomainGetBlkioParametersType virDomainGetBlkioParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetBlkioParametersSymbol = libvirtSymbol(libvirt,
                                                              "virDomainGetBlkioParameters",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetBlkioParameters");
        return ret;
    }
#  else
    virDomainGetBlkioParametersSymbol = &virDomainGetBlkioParameters;
#  endif

    ret = virDomainGetBlkioParametersSymbol(domain,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 1)
    setVirError(err, "Function virDomainGetBlockInfo compiled out (from 0.8.1)");
    return ret;
#else
    static virDomainGetBlockInfoType virDomainGetBlockInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetBlockInfoSymbol = libvirtSymbol(libvirt,
                                                        "virDomainGetBlockInfo",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetBlockInfo");
        return ret;
    }
#  else
    virDomainGetBlockInfoSymbol = &virDomainGetBlockInfo;
#  endif

    ret = virDomainGetBlockInfoSymbol(domain,
                                      disk,
                                      info,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virDomainGetBlockIoTune compiled out (from 0.9.8)");
    return ret;
#else
    static virDomainGetBlockIoTuneType virDomainGetBlockIoTuneSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetBlockIoTuneSymbol = libvirtSymbol(libvirt,
                                                          "virDomainGetBlockIoTune",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetBlockIoTune");
        return ret;
    }
#  else
    virDomainGetBlockIoTuneSymbol = &virDomainGetBlockIoTune;
#  endif

    ret = virDomainGetBlockIoTuneSymbol(dom,
                                        disk,
                                        params,
                                        nparams,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainGetBlockJobInfo compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainGetBlockJobInfoType virDomainGetBlockJobInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetBlockJobInfoSymbol = libvirtSymbol(libvirt,
                                                           "virDomainGetBlockJobInfo",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetBlockJobInfo");
        return ret;
    }
#  else
    virDomainGetBlockJobInfoSymbol = &virDomainGetBlockJobInfo;
#  endif

    ret = virDomainGetBlockJobInfoSymbol(dom,
                                         disk,
                                         info,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainGetCPUStats compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainGetCPUStatsType virDomainGetCPUStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetCPUStatsSymbol = libvirtSymbol(libvirt,
                                                       "virDomainGetCPUStats",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetCPUStats");
        return ret;
    }
#  else
    virDomainGetCPUStatsSymbol = &virDomainGetCPUStats;
#  endif

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
#endif
}

typedef virConnectPtr
(*virDomainGetConnectType)(virDomainPtr dom);

virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virDomainGetConnect compiled out (from 0.3.0)");
    return ret;
#else
    static virDomainGetConnectType virDomainGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetConnectSymbol = libvirtSymbol(libvirt,
                                                      "virDomainGetConnect",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetConnect");
        return ret;
    }
#  else
    virDomainGetConnectSymbol = &virDomainGetConnect;
#  endif

    ret = virDomainGetConnectSymbol(dom);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainGetControlInfo compiled out (from 0.9.3)");
    return ret;
#else
    static virDomainGetControlInfoType virDomainGetControlInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetControlInfoSymbol = libvirtSymbol(libvirt,
                                                          "virDomainGetControlInfo",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetControlInfo");
        return ret;
    }
#  else
    virDomainGetControlInfoSymbol = &virDomainGetControlInfo;
#  endif

    ret = virDomainGetControlInfoSymbol(domain,
                                        info,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainGetDiskErrors compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainGetDiskErrorsType virDomainGetDiskErrorsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetDiskErrorsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainGetDiskErrors",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetDiskErrors");
        return ret;
    }
#  else
    virDomainGetDiskErrorsSymbol = &virDomainGetDiskErrors;
#  endif

    ret = virDomainGetDiskErrorsSymbol(dom,
                                       errors,
                                       maxerrors,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainGetEmulatorPinInfo compiled out (from 0.10.0)");
    return ret;
#else
    static virDomainGetEmulatorPinInfoType virDomainGetEmulatorPinInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetEmulatorPinInfoSymbol = libvirtSymbol(libvirt,
                                                              "virDomainGetEmulatorPinInfo",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetEmulatorPinInfo");
        return ret;
    }
#  else
    virDomainGetEmulatorPinInfoSymbol = &virDomainGetEmulatorPinInfo;
#  endif

    ret = virDomainGetEmulatorPinInfoSymbol(domain,
                                            cpumap,
                                            maplen,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 11)
    setVirError(err, "Function virDomainGetFSInfo compiled out (from 1.2.11)");
    return ret;
#else
    static virDomainGetFSInfoType virDomainGetFSInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetFSInfoSymbol = libvirtSymbol(libvirt,
                                                     "virDomainGetFSInfo",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetFSInfo");
        return ret;
    }
#  else
    virDomainGetFSInfoSymbol = &virDomainGetFSInfo;
#  endif

    ret = virDomainGetFSInfoSymbol(dom,
                                   info,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 7, 0)
    setVirError(err, "Function virDomainGetGuestInfo compiled out (from 5.7.0)");
    return ret;
#else
    static virDomainGetGuestInfoType virDomainGetGuestInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetGuestInfoSymbol = libvirtSymbol(libvirt,
                                                        "virDomainGetGuestInfo",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetGuestInfo");
        return ret;
    }
#  else
    virDomainGetGuestInfoSymbol = &virDomainGetGuestInfo;
#  endif

    ret = virDomainGetGuestInfoSymbol(domain,
                                      types,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virDomainGetGuestVcpus compiled out (from 2.0.0)");
    return ret;
#else
    static virDomainGetGuestVcpusType virDomainGetGuestVcpusSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetGuestVcpusSymbol = libvirtSymbol(libvirt,
                                                         "virDomainGetGuestVcpus",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetGuestVcpus");
        return ret;
    }
#  else
    virDomainGetGuestVcpusSymbol = &virDomainGetGuestVcpus;
#  endif

    ret = virDomainGetGuestVcpusSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainGetHostnameType)(virDomainPtr domain,
                            unsigned int flags);

char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainGetHostname compiled out (from 0.10.0)");
    return ret;
#else
    static virDomainGetHostnameType virDomainGetHostnameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetHostnameSymbol = libvirtSymbol(libvirt,
                                                       "virDomainGetHostname",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetHostname");
        return ret;
    }
#  else
    virDomainGetHostnameSymbol = &virDomainGetHostname;
#  endif

    ret = virDomainGetHostnameSymbol(domain,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef unsigned int
(*virDomainGetIDType)(virDomainPtr domain);

unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err)
{
    unsigned int ret = 0;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetID compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainGetIDType virDomainGetIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetIDSymbol = libvirtSymbol(libvirt,
                                                 "virDomainGetID",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetID");
        return ret;
    }
#  else
    virDomainGetIDSymbol = &virDomainGetID;
#  endif

    ret = virDomainGetIDSymbol(domain);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(err, "Function virDomainGetIOThreadInfo compiled out (from 1.2.14)");
    return ret;
#else
    static virDomainGetIOThreadInfoType virDomainGetIOThreadInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetIOThreadInfoSymbol = libvirtSymbol(libvirt,
                                                           "virDomainGetIOThreadInfo",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetIOThreadInfo");
        return ret;
    }
#  else
    virDomainGetIOThreadInfoSymbol = &virDomainGetIOThreadInfo;
#  endif

    ret = virDomainGetIOThreadInfoSymbol(dom,
                                         info,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetInfoType)(virDomainPtr domain,
                        virDomainInfoPtr info);

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetInfo compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainGetInfoType virDomainGetInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetInfoSymbol = libvirtSymbol(libvirt,
                                                   "virDomainGetInfo",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetInfo");
        return ret;
    }
#  else
    virDomainGetInfoSymbol = &virDomainGetInfo;
#  endif

    ret = virDomainGetInfoSymbol(domain,
                                 info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainGetInterfaceParameters compiled out (from 0.9.9)");
    return ret;
#else
    static virDomainGetInterfaceParametersType virDomainGetInterfaceParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetInterfaceParametersSymbol = libvirtSymbol(libvirt,
                                                                  "virDomainGetInterfaceParameters",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetInterfaceParameters");
        return ret;
    }
#  else
    virDomainGetInterfaceParametersSymbol = &virDomainGetInterfaceParameters;
#  endif

    ret = virDomainGetInterfaceParametersSymbol(domain,
                                                device,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetJobInfoType)(virDomainPtr domain,
                           virDomainJobInfoPtr info);

int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainGetJobInfo compiled out (from 0.7.7)");
    return ret;
#else
    static virDomainGetJobInfoType virDomainGetJobInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetJobInfoSymbol = libvirtSymbol(libvirt,
                                                      "virDomainGetJobInfo",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetJobInfo");
        return ret;
    }
#  else
    virDomainGetJobInfoSymbol = &virDomainGetJobInfo;
#  endif

    ret = virDomainGetJobInfoSymbol(domain,
                                    info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virDomainGetJobStats compiled out (from 1.0.3)");
    return ret;
#else
    static virDomainGetJobStatsType virDomainGetJobStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetJobStatsSymbol = libvirtSymbol(libvirt,
                                                       "virDomainGetJobStats",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetJobStats");
        return ret;
    }
#  else
    virDomainGetJobStatsSymbol = &virDomainGetJobStats;
#  endif

    ret = virDomainGetJobStatsSymbol(domain,
                                     type,
                                     params,
                                     nparams,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virDomainGetLaunchSecurityInfo compiled out (from 4.5.0)");
    return ret;
#else
    static virDomainGetLaunchSecurityInfoType virDomainGetLaunchSecurityInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetLaunchSecurityInfoSymbol = libvirtSymbol(libvirt,
                                                                 "virDomainGetLaunchSecurityInfo",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetLaunchSecurityInfo");
        return ret;
    }
#  else
    virDomainGetLaunchSecurityInfoSymbol = &virDomainGetLaunchSecurityInfo;
#  endif

    ret = virDomainGetLaunchSecurityInfoSymbol(domain,
                                               params,
                                               nparams,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef unsigned long
(*virDomainGetMaxMemoryType)(virDomainPtr domain);

unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err)
{
    unsigned long ret = 0;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetMaxMemory compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainGetMaxMemoryType virDomainGetMaxMemorySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetMaxMemorySymbol = libvirtSymbol(libvirt,
                                                        "virDomainGetMaxMemory",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetMaxMemory");
        return ret;
    }
#  else
    virDomainGetMaxMemorySymbol = &virDomainGetMaxMemory;
#  endif

    ret = virDomainGetMaxMemorySymbol(domain);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetMaxVcpusType)(virDomainPtr domain);

int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virDomainGetMaxVcpus compiled out (from 0.2.1)");
    return ret;
#else
    static virDomainGetMaxVcpusType virDomainGetMaxVcpusSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetMaxVcpusSymbol = libvirtSymbol(libvirt,
                                                       "virDomainGetMaxVcpus",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetMaxVcpus");
        return ret;
    }
#  else
    virDomainGetMaxVcpusSymbol = &virDomainGetMaxVcpus;
#  endif

    ret = virDomainGetMaxVcpusSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainGetMemoryParameters compiled out (from 0.8.5)");
    return ret;
#else
    static virDomainGetMemoryParametersType virDomainGetMemoryParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetMemoryParametersSymbol = libvirtSymbol(libvirt,
                                                               "virDomainGetMemoryParameters",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetMemoryParameters");
        return ret;
    }
#  else
    virDomainGetMemoryParametersSymbol = &virDomainGetMemoryParameters;
#  endif

    ret = virDomainGetMemoryParametersSymbol(domain,
                                             params,
                                             nparams,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetMessagesType)(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags);

int
virDomainGetMessagesWrapper(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 1, 0)
    setVirError(err, "Function virDomainGetMessages compiled out (from 7.1.0)");
    return ret;
#else
    static virDomainGetMessagesType virDomainGetMessagesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetMessagesSymbol = libvirtSymbol(libvirt,
                                                       "virDomainGetMessages",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetMessages");
        return ret;
    }
#  else
    virDomainGetMessagesSymbol = &virDomainGetMessages;
#  endif

    ret = virDomainGetMessagesSymbol(domain,
                                     msgs,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainGetMetadata compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainGetMetadataType virDomainGetMetadataSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetMetadataSymbol = libvirtSymbol(libvirt,
                                                       "virDomainGetMetadata",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetMetadata");
        return ret;
    }
#  else
    virDomainGetMetadataSymbol = &virDomainGetMetadata;
#  endif

    ret = virDomainGetMetadataSymbol(domain,
                                     type,
                                     uri,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virDomainGetNameType)(virDomainPtr domain);

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetName compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainGetNameType virDomainGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetNameSymbol = libvirtSymbol(libvirt,
                                                   "virDomainGetName",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetName");
        return ret;
    }
#  else
    virDomainGetNameSymbol = &virDomainGetName;
#  endif

    ret = virDomainGetNameSymbol(domain);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainGetNumaParameters compiled out (from 0.9.9)");
    return ret;
#else
    static virDomainGetNumaParametersType virDomainGetNumaParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetNumaParametersSymbol = libvirtSymbol(libvirt,
                                                             "virDomainGetNumaParameters",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetNumaParameters");
        return ret;
    }
#  else
    virDomainGetNumaParametersSymbol = &virDomainGetNumaParameters;
#  endif

    ret = virDomainGetNumaParametersSymbol(domain,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainGetOSTypeType)(virDomainPtr domain);

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetOSType compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainGetOSTypeType virDomainGetOSTypeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetOSTypeSymbol = libvirtSymbol(libvirt,
                                                     "virDomainGetOSType",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetOSType");
        return ret;
    }
#  else
    virDomainGetOSTypeSymbol = &virDomainGetOSType;
#  endif

    ret = virDomainGetOSTypeSymbol(domain);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 3, 3)
    setVirError(err, "Function virDomainGetPerfEvents compiled out (from 1.3.3)");
    return ret;
#else
    static virDomainGetPerfEventsType virDomainGetPerfEventsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetPerfEventsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainGetPerfEvents",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetPerfEvents");
        return ret;
    }
#  else
    virDomainGetPerfEventsSymbol = &virDomainGetPerfEvents;
#  endif

    ret = virDomainGetPerfEventsSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 3)
    setVirError(err, "Function virDomainGetSchedulerParameters compiled out (from 0.2.3)");
    return ret;
#else
    static virDomainGetSchedulerParametersType virDomainGetSchedulerParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetSchedulerParametersSymbol = libvirtSymbol(libvirt,
                                                                  "virDomainGetSchedulerParameters",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetSchedulerParameters");
        return ret;
    }
#  else
    virDomainGetSchedulerParametersSymbol = &virDomainGetSchedulerParameters;
#  endif

    ret = virDomainGetSchedulerParametersSymbol(domain,
                                                params,
                                                nparams);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainGetSchedulerParametersFlags compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainGetSchedulerParametersFlagsType virDomainGetSchedulerParametersFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetSchedulerParametersFlagsSymbol = libvirtSymbol(libvirt,
                                                                       "virDomainGetSchedulerParametersFlags",
                                                                       &success,
                                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetSchedulerParametersFlags");
        return ret;
    }
#  else
    virDomainGetSchedulerParametersFlagsSymbol = &virDomainGetSchedulerParametersFlags;
#  endif

    ret = virDomainGetSchedulerParametersFlagsSymbol(domain,
                                                     params,
                                                     nparams,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainGetSchedulerTypeType)(virDomainPtr domain,
                                 int * nparams);

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int * nparams,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 3)
    setVirError(err, "Function virDomainGetSchedulerType compiled out (from 0.2.3)");
    return ret;
#else
    static virDomainGetSchedulerTypeType virDomainGetSchedulerTypeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetSchedulerTypeSymbol = libvirtSymbol(libvirt,
                                                            "virDomainGetSchedulerType",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetSchedulerType");
        return ret;
    }
#  else
    virDomainGetSchedulerTypeSymbol = &virDomainGetSchedulerType;
#  endif

    ret = virDomainGetSchedulerTypeSymbol(domain,
                                          nparams);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetSecurityLabelType)(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel);

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virDomainGetSecurityLabel compiled out (from 0.6.1)");
    return ret;
#else
    static virDomainGetSecurityLabelType virDomainGetSecurityLabelSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetSecurityLabelSymbol = libvirtSymbol(libvirt,
                                                            "virDomainGetSecurityLabel",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetSecurityLabel");
        return ret;
    }
#  else
    virDomainGetSecurityLabelSymbol = &virDomainGetSecurityLabel;
#  endif

    ret = virDomainGetSecurityLabelSymbol(domain,
                                          seclabel);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetSecurityLabelListType)(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels);

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels,
                                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainGetSecurityLabelList compiled out (from 0.10.0)");
    return ret;
#else
    static virDomainGetSecurityLabelListType virDomainGetSecurityLabelListSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetSecurityLabelListSymbol = libvirtSymbol(libvirt,
                                                                "virDomainGetSecurityLabelList",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetSecurityLabelList");
        return ret;
    }
#  else
    virDomainGetSecurityLabelListSymbol = &virDomainGetSecurityLabelList;
#  endif

    ret = virDomainGetSecurityLabelListSymbol(domain,
                                              seclabels);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainGetState compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainGetStateType virDomainGetStateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetStateSymbol = libvirtSymbol(libvirt,
                                                    "virDomainGetState",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetState");
        return ret;
    }
#  else
    virDomainGetStateSymbol = &virDomainGetState;
#  endif

    ret = virDomainGetStateSymbol(domain,
                                  state,
                                  reason,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainGetTime compiled out (from 1.2.5)");
    return ret;
#else
    static virDomainGetTimeType virDomainGetTimeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetTimeSymbol = libvirtSymbol(libvirt,
                                                   "virDomainGetTime",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetTime");
        return ret;
    }
#  else
    virDomainGetTimeSymbol = &virDomainGetTime;
#  endif

    ret = virDomainGetTimeSymbol(dom,
                                 seconds,
                                 nseconds,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetUUIDType)(virDomainPtr domain,
                        unsigned char * uuid);

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 5)
    setVirError(err, "Function virDomainGetUUID compiled out (from 0.0.5)");
    return ret;
#else
    static virDomainGetUUIDType virDomainGetUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetUUIDSymbol = libvirtSymbol(libvirt,
                                                   "virDomainGetUUID",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetUUID");
        return ret;
    }
#  else
    virDomainGetUUIDSymbol = &virDomainGetUUID;
#  endif

    ret = virDomainGetUUIDSymbol(domain,
                                 uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetUUIDStringType)(virDomainPtr domain,
                              char * buf);

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char * buf,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainGetUUIDString compiled out (from 0.1.1)");
    return ret;
#else
    static virDomainGetUUIDStringType virDomainGetUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetUUIDStringSymbol = libvirtSymbol(libvirt,
                                                         "virDomainGetUUIDString",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetUUIDString");
        return ret;
    }
#  else
    virDomainGetUUIDStringSymbol = &virDomainGetUUIDString;
#  endif

    ret = virDomainGetUUIDStringSymbol(domain,
                                       buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainGetVcpuPinInfo compiled out (from 0.9.3)");
    return ret;
#else
    static virDomainGetVcpuPinInfoType virDomainGetVcpuPinInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetVcpuPinInfoSymbol = libvirtSymbol(libvirt,
                                                          "virDomainGetVcpuPinInfo",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetVcpuPinInfo");
        return ret;
    }
#  else
    virDomainGetVcpuPinInfoSymbol = &virDomainGetVcpuPinInfo;
#  endif

    ret = virDomainGetVcpuPinInfoSymbol(domain,
                                        ncpumaps,
                                        cpumaps,
                                        maplen,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 4)
    setVirError(err, "Function virDomainGetVcpus compiled out (from 0.1.4)");
    return ret;
#else
    static virDomainGetVcpusType virDomainGetVcpusSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetVcpusSymbol = libvirtSymbol(libvirt,
                                                    "virDomainGetVcpus",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetVcpus");
        return ret;
    }
#  else
    virDomainGetVcpusSymbol = &virDomainGetVcpus;
#  endif

    ret = virDomainGetVcpusSymbol(domain,
                                  info,
                                  maxinfo,
                                  cpumaps,
                                  maplen);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainGetVcpusFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainGetVcpusFlags compiled out (from 0.8.5)");
    return ret;
#else
    static virDomainGetVcpusFlagsType virDomainGetVcpusFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetVcpusFlagsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainGetVcpusFlags",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetVcpusFlags");
        return ret;
    }
#  else
    virDomainGetVcpusFlagsSymbol = &virDomainGetVcpusFlags;
#  endif

    ret = virDomainGetVcpusFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainGetXMLDescType)(virDomainPtr domain,
                           unsigned int flags);

char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetXMLDesc compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainGetXMLDescType virDomainGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                      "virDomainGetXMLDesc",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainGetXMLDesc");
        return ret;
    }
#  else
    virDomainGetXMLDescSymbol = &virDomainGetXMLDesc;
#  endif

    ret = virDomainGetXMLDescSymbol(domain,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainHasCurrentSnapshotType)(virDomainPtr domain,
                                   unsigned int flags);

int
virDomainHasCurrentSnapshotWrapper(virDomainPtr domain,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainHasCurrentSnapshot compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainHasCurrentSnapshotType virDomainHasCurrentSnapshotSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainHasCurrentSnapshotSymbol = libvirtSymbol(libvirt,
                                                              "virDomainHasCurrentSnapshot",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainHasCurrentSnapshot");
        return ret;
    }
#  else
    virDomainHasCurrentSnapshotSymbol = &virDomainHasCurrentSnapshot;
#  endif

    ret = virDomainHasCurrentSnapshotSymbol(domain,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainHasManagedSaveImageType)(virDomainPtr dom,
                                    unsigned int flags);

int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainHasManagedSaveImage compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainHasManagedSaveImageType virDomainHasManagedSaveImageSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainHasManagedSaveImageSymbol = libvirtSymbol(libvirt,
                                                               "virDomainHasManagedSaveImage",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainHasManagedSaveImage");
        return ret;
    }
#  else
    virDomainHasManagedSaveImageSymbol = &virDomainHasManagedSaveImage;
#  endif

    ret = virDomainHasManagedSaveImageSymbol(dom,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virDomainIOThreadInfoFreeType)(virDomainIOThreadInfoPtr info);

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(NULL, "Function virDomainIOThreadInfoFree compiled out (from 1.2.14)");
    return;
#else
    static virDomainIOThreadInfoFreeType virDomainIOThreadInfoFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virDomainIOThreadInfoFreeSymbol = libvirtSymbol(libvirt,
                                                            "virDomainIOThreadInfoFree",
                                                            &success,
                                                            NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virDomainIOThreadInfoFree");
        return;
    }
#  else
    virDomainIOThreadInfoFreeSymbol = &virDomainIOThreadInfoFree;
#  endif

    virDomainIOThreadInfoFreeSymbol(info);
#endif
}

typedef int
(*virDomainInjectNMIType)(virDomainPtr domain,
                          unsigned int flags);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainInjectNMI compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainInjectNMIType virDomainInjectNMISymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainInjectNMISymbol = libvirtSymbol(libvirt,
                                                     "virDomainInjectNMI",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainInjectNMI");
        return ret;
    }
#  else
    virDomainInjectNMISymbol = &virDomainInjectNMI;
#  endif

    ret = virDomainInjectNMISymbol(domain,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(err, "Function virDomainInterfaceAddresses compiled out (from 1.2.14)");
    return ret;
#else
    static virDomainInterfaceAddressesType virDomainInterfaceAddressesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainInterfaceAddressesSymbol = libvirtSymbol(libvirt,
                                                              "virDomainInterfaceAddresses",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainInterfaceAddresses");
        return ret;
    }
#  else
    virDomainInterfaceAddressesSymbol = &virDomainInterfaceAddresses;
#  endif

    ret = virDomainInterfaceAddressesSymbol(dom,
                                            ifaces,
                                            source,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virDomainInterfaceFreeType)(virDomainInterfacePtr iface);

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(NULL, "Function virDomainInterfaceFree compiled out (from 1.2.14)");
    return;
#else
    static virDomainInterfaceFreeType virDomainInterfaceFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virDomainInterfaceFreeSymbol = libvirtSymbol(libvirt,
                                                         "virDomainInterfaceFree",
                                                         &success,
                                                         NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virDomainInterfaceFree");
        return;
    }
#  else
    virDomainInterfaceFreeSymbol = &virDomainInterfaceFree;
#  endif

    virDomainInterfaceFreeSymbol(iface);
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 2)
    setVirError(err, "Function virDomainInterfaceStats compiled out (from 0.3.2)");
    return ret;
#else
    static virDomainInterfaceStatsType virDomainInterfaceStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainInterfaceStatsSymbol = libvirtSymbol(libvirt,
                                                          "virDomainInterfaceStats",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainInterfaceStats");
        return ret;
    }
#  else
    virDomainInterfaceStatsSymbol = &virDomainInterfaceStats;
#  endif

    ret = virDomainInterfaceStatsSymbol(dom,
                                        device,
                                        stats,
                                        size);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainIsActiveType)(virDomainPtr dom);

int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virDomainIsActive compiled out (from 0.7.3)");
    return ret;
#else
    static virDomainIsActiveType virDomainIsActiveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainIsActiveSymbol = libvirtSymbol(libvirt,
                                                    "virDomainIsActive",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainIsActive");
        return ret;
    }
#  else
    virDomainIsActiveSymbol = &virDomainIsActive;
#  endif

    ret = virDomainIsActiveSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainIsPersistentType)(virDomainPtr dom);

int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virDomainIsPersistent compiled out (from 0.7.3)");
    return ret;
#else
    static virDomainIsPersistentType virDomainIsPersistentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainIsPersistentSymbol = libvirtSymbol(libvirt,
                                                        "virDomainIsPersistent",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainIsPersistent");
        return ret;
    }
#  else
    virDomainIsPersistentSymbol = &virDomainIsPersistent;
#  endif

    ret = virDomainIsPersistentSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainIsUpdatedType)(virDomainPtr dom);

int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 6)
    setVirError(err, "Function virDomainIsUpdated compiled out (from 0.8.6)");
    return ret;
#else
    static virDomainIsUpdatedType virDomainIsUpdatedSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainIsUpdatedSymbol = libvirtSymbol(libvirt,
                                                     "virDomainIsUpdated",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainIsUpdated");
        return ret;
    }
#  else
    virDomainIsUpdatedSymbol = &virDomainIsUpdated;
#  endif

    ret = virDomainIsUpdatedSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 6, 0)
    setVirError(err, "Function virDomainListAllCheckpoints compiled out (from 5.6.0)");
    return ret;
#else
    static virDomainListAllCheckpointsType virDomainListAllCheckpointsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainListAllCheckpointsSymbol = libvirtSymbol(libvirt,
                                                              "virDomainListAllCheckpoints",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainListAllCheckpoints");
        return ret;
    }
#  else
    virDomainListAllCheckpointsSymbol = &virDomainListAllCheckpoints;
#  endif

    ret = virDomainListAllCheckpointsSymbol(domain,
                                            checkpoints,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainListAllSnapshots compiled out (from 0.9.13)");
    return ret;
#else
    static virDomainListAllSnapshotsType virDomainListAllSnapshotsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainListAllSnapshotsSymbol = libvirtSymbol(libvirt,
                                                            "virDomainListAllSnapshots",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainListAllSnapshots");
        return ret;
    }
#  else
    virDomainListAllSnapshotsSymbol = &virDomainListAllSnapshots;
#  endif

    ret = virDomainListAllSnapshotsSymbol(domain,
                                          snaps,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virDomainListGetStats compiled out (from 1.2.8)");
    return ret;
#else
    static virDomainListGetStatsType virDomainListGetStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainListGetStatsSymbol = libvirtSymbol(libvirt,
                                                        "virDomainListGetStats",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainListGetStats");
        return ret;
    }
#  else
    virDomainListGetStatsSymbol = &virDomainListGetStats;
#  endif

    ret = virDomainListGetStatsSymbol(doms,
                                      stats,
                                      retStats,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainLookupByIDType)(virConnectPtr conn,
                           int id);

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainLookupByID compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainLookupByIDType virDomainLookupByIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLookupByIDSymbol = libvirtSymbol(libvirt,
                                                      "virDomainLookupByID",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLookupByID");
        return ret;
    }
#  else
    virDomainLookupByIDSymbol = &virDomainLookupByID;
#  endif

    ret = virDomainLookupByIDSymbol(conn,
                                    id);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainLookupByNameType)(virConnectPtr conn,
                             const char * name);

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char * name,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainLookupByName compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainLookupByNameType virDomainLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLookupByNameSymbol = libvirtSymbol(libvirt,
                                                        "virDomainLookupByName",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLookupByName");
        return ret;
    }
#  else
    virDomainLookupByNameSymbol = &virDomainLookupByName;
#  endif

    ret = virDomainLookupByNameSymbol(conn,
                                      name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainLookupByUUIDType)(virConnectPtr conn,
                             const unsigned char * uuid);

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 5)
    setVirError(err, "Function virDomainLookupByUUID compiled out (from 0.0.5)");
    return ret;
#else
    static virDomainLookupByUUIDType virDomainLookupByUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLookupByUUIDSymbol = libvirtSymbol(libvirt,
                                                        "virDomainLookupByUUID",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLookupByUUID");
        return ret;
    }
#  else
    virDomainLookupByUUIDSymbol = &virDomainLookupByUUID;
#  endif

    ret = virDomainLookupByUUIDSymbol(conn,
                                      uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainLookupByUUIDStringType)(virConnectPtr conn,
                                   const char * uuidstr);

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainLookupByUUIDString compiled out (from 0.1.1)");
    return ret;
#else
    static virDomainLookupByUUIDStringType virDomainLookupByUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLookupByUUIDStringSymbol = libvirtSymbol(libvirt,
                                                              "virDomainLookupByUUIDString",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLookupByUUIDString");
        return ret;
    }
#  else
    virDomainLookupByUUIDStringSymbol = &virDomainLookupByUUIDString;
#  endif

    ret = virDomainLookupByUUIDStringSymbol(conn,
                                            uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainManagedSaveType)(virDomainPtr dom,
                            unsigned int flags);

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainManagedSave compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainManagedSaveType virDomainManagedSaveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainManagedSaveSymbol = libvirtSymbol(libvirt,
                                                       "virDomainManagedSave",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainManagedSave");
        return ret;
    }
#  else
    virDomainManagedSaveSymbol = &virDomainManagedSave;
#  endif

    ret = virDomainManagedSaveSymbol(dom,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 7, 0)
    setVirError(err, "Function virDomainManagedSaveDefineXML compiled out (from 3.7.0)");
    return ret;
#else
    static virDomainManagedSaveDefineXMLType virDomainManagedSaveDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainManagedSaveDefineXMLSymbol = libvirtSymbol(libvirt,
                                                                "virDomainManagedSaveDefineXML",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainManagedSaveDefineXML");
        return ret;
    }
#  else
    virDomainManagedSaveDefineXMLSymbol = &virDomainManagedSaveDefineXML;
#  endif

    ret = virDomainManagedSaveDefineXMLSymbol(domain,
                                              dxml,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainManagedSaveGetXMLDescType)(virDomainPtr domain,
                                      unsigned int flags);

char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 7, 0)
    setVirError(err, "Function virDomainManagedSaveGetXMLDesc compiled out (from 3.7.0)");
    return ret;
#else
    static virDomainManagedSaveGetXMLDescType virDomainManagedSaveGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainManagedSaveGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                                 "virDomainManagedSaveGetXMLDesc",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainManagedSaveGetXMLDesc");
        return ret;
    }
#  else
    virDomainManagedSaveGetXMLDescSymbol = &virDomainManagedSaveGetXMLDesc;
#  endif

    ret = virDomainManagedSaveGetXMLDescSymbol(domain,
                                               flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainManagedSaveRemoveType)(virDomainPtr dom,
                                  unsigned int flags);

int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainManagedSaveRemove compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainManagedSaveRemoveType virDomainManagedSaveRemoveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainManagedSaveRemoveSymbol = libvirtSymbol(libvirt,
                                                             "virDomainManagedSaveRemove",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainManagedSaveRemove");
        return ret;
    }
#  else
    virDomainManagedSaveRemoveSymbol = &virDomainManagedSaveRemove;
#  endif

    ret = virDomainManagedSaveRemoveSymbol(dom,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 2)
    setVirError(err, "Function virDomainMemoryPeek compiled out (from 0.4.2)");
    return ret;
#else
    static virDomainMemoryPeekType virDomainMemoryPeekSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMemoryPeekSymbol = libvirtSymbol(libvirt,
                                                      "virDomainMemoryPeek",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMemoryPeek");
        return ret;
    }
#  else
    virDomainMemoryPeekSymbol = &virDomainMemoryPeek;
#  endif

    ret = virDomainMemoryPeekSymbol(dom,
                                    start,
                                    size,
                                    buffer,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 5)
    setVirError(err, "Function virDomainMemoryStats compiled out (from 0.7.5)");
    return ret;
#else
    static virDomainMemoryStatsType virDomainMemoryStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMemoryStatsSymbol = libvirtSymbol(libvirt,
                                                       "virDomainMemoryStats",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMemoryStats");
        return ret;
    }
#  else
    virDomainMemoryStatsSymbol = &virDomainMemoryStats;
#  endif

    ret = virDomainMemoryStatsSymbol(dom,
                                     stats,
                                     nr_stats,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 2)
    setVirError(err, "Function virDomainMigrate compiled out (from 0.3.2)");
    return ret;
#else
    static virDomainMigrateType virDomainMigrateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateSymbol = libvirtSymbol(libvirt,
                                                   "virDomainMigrate",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrate");
        return ret;
    }
#  else
    virDomainMigrateSymbol = &virDomainMigrate;
#  endif

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
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainMigrate2 compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainMigrate2Type virDomainMigrate2Symbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrate2Symbol = libvirtSymbol(libvirt,
                                                    "virDomainMigrate2",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrate2");
        return ret;
    }
#  else
    virDomainMigrate2Symbol = &virDomainMigrate2;
#  endif

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
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 1, 0)
    setVirError(err, "Function virDomainMigrate3 compiled out (from 1.1.0)");
    return ret;
#else
    static virDomainMigrate3Type virDomainMigrate3Symbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrate3Symbol = libvirtSymbol(libvirt,
                                                    "virDomainMigrate3",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrate3");
        return ret;
    }
#  else
    virDomainMigrate3Symbol = &virDomainMigrate3;
#  endif

    ret = virDomainMigrate3Symbol(domain,
                                  dconn,
                                  params,
                                  nparams,
                                  flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virDomainMigrateGetCompressionCache compiled out (from 1.0.3)");
    return ret;
#else
    static virDomainMigrateGetCompressionCacheType virDomainMigrateGetCompressionCacheSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateGetCompressionCacheSymbol = libvirtSymbol(libvirt,
                                                                      "virDomainMigrateGetCompressionCache",
                                                                      &success,
                                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateGetCompressionCache");
        return ret;
    }
#  else
    virDomainMigrateGetCompressionCacheSymbol = &virDomainMigrateGetCompressionCache;
#  endif

    ret = virDomainMigrateGetCompressionCacheSymbol(domain,
                                                    cacheSize,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 7, 0)
    setVirError(err, "Function virDomainMigrateGetMaxDowntime compiled out (from 3.7.0)");
    return ret;
#else
    static virDomainMigrateGetMaxDowntimeType virDomainMigrateGetMaxDowntimeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateGetMaxDowntimeSymbol = libvirtSymbol(libvirt,
                                                                 "virDomainMigrateGetMaxDowntime",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateGetMaxDowntime");
        return ret;
    }
#  else
    virDomainMigrateGetMaxDowntimeSymbol = &virDomainMigrateGetMaxDowntime;
#  endif

    ret = virDomainMigrateGetMaxDowntimeSymbol(domain,
                                               downtime,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainMigrateGetMaxSpeed compiled out (from 0.9.5)");
    return ret;
#else
    static virDomainMigrateGetMaxSpeedType virDomainMigrateGetMaxSpeedSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateGetMaxSpeedSymbol = libvirtSymbol(libvirt,
                                                              "virDomainMigrateGetMaxSpeed",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateGetMaxSpeed");
        return ret;
    }
#  else
    virDomainMigrateGetMaxSpeedSymbol = &virDomainMigrateGetMaxSpeed;
#  endif

    ret = virDomainMigrateGetMaxSpeedSymbol(domain,
                                            bandwidth,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virDomainMigrateSetCompressionCache compiled out (from 1.0.3)");
    return ret;
#else
    static virDomainMigrateSetCompressionCacheType virDomainMigrateSetCompressionCacheSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateSetCompressionCacheSymbol = libvirtSymbol(libvirt,
                                                                      "virDomainMigrateSetCompressionCache",
                                                                      &success,
                                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateSetCompressionCache");
        return ret;
    }
#  else
    virDomainMigrateSetCompressionCacheSymbol = &virDomainMigrateSetCompressionCache;
#  endif

    ret = virDomainMigrateSetCompressionCacheSymbol(domain,
                                                    cacheSize,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainMigrateSetMaxDowntime compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainMigrateSetMaxDowntimeType virDomainMigrateSetMaxDowntimeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateSetMaxDowntimeSymbol = libvirtSymbol(libvirt,
                                                                 "virDomainMigrateSetMaxDowntime",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateSetMaxDowntime");
        return ret;
    }
#  else
    virDomainMigrateSetMaxDowntimeSymbol = &virDomainMigrateSetMaxDowntime;
#  endif

    ret = virDomainMigrateSetMaxDowntimeSymbol(domain,
                                               downtime,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainMigrateSetMaxSpeed compiled out (from 0.9.0)");
    return ret;
#else
    static virDomainMigrateSetMaxSpeedType virDomainMigrateSetMaxSpeedSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateSetMaxSpeedSymbol = libvirtSymbol(libvirt,
                                                              "virDomainMigrateSetMaxSpeed",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateSetMaxSpeed");
        return ret;
    }
#  else
    virDomainMigrateSetMaxSpeedSymbol = &virDomainMigrateSetMaxSpeed;
#  endif

    ret = virDomainMigrateSetMaxSpeedSymbol(domain,
                                            bandwidth,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainMigrateStartPostCopyType)(virDomainPtr domain,
                                     unsigned int flags);

int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 3, 3)
    setVirError(err, "Function virDomainMigrateStartPostCopy compiled out (from 1.3.3)");
    return ret;
#else
    static virDomainMigrateStartPostCopyType virDomainMigrateStartPostCopySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateStartPostCopySymbol = libvirtSymbol(libvirt,
                                                                "virDomainMigrateStartPostCopy",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateStartPostCopy");
        return ret;
    }
#  else
    virDomainMigrateStartPostCopySymbol = &virDomainMigrateStartPostCopy;
#  endif

    ret = virDomainMigrateStartPostCopySymbol(domain,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virDomainMigrateToURI compiled out (from 0.7.2)");
    return ret;
#else
    static virDomainMigrateToURIType virDomainMigrateToURISymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateToURISymbol = libvirtSymbol(libvirt,
                                                        "virDomainMigrateToURI",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateToURI");
        return ret;
    }
#  else
    virDomainMigrateToURISymbol = &virDomainMigrateToURI;
#  endif

    ret = virDomainMigrateToURISymbol(domain,
                                      duri,
                                      flags,
                                      dname,
                                      bandwidth);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainMigrateToURI2 compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainMigrateToURI2Type virDomainMigrateToURI2Symbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateToURI2Symbol = libvirtSymbol(libvirt,
                                                         "virDomainMigrateToURI2",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateToURI2");
        return ret;
    }
#  else
    virDomainMigrateToURI2Symbol = &virDomainMigrateToURI2;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 1, 0)
    setVirError(err, "Function virDomainMigrateToURI3 compiled out (from 1.1.0)");
    return ret;
#else
    static virDomainMigrateToURI3Type virDomainMigrateToURI3Symbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainMigrateToURI3Symbol = libvirtSymbol(libvirt,
                                                         "virDomainMigrateToURI3",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainMigrateToURI3");
        return ret;
    }
#  else
    virDomainMigrateToURI3Symbol = &virDomainMigrateToURI3;
#  endif

    ret = virDomainMigrateToURI3Symbol(domain,
                                       dconnuri,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virDomainOpenChannel compiled out (from 1.0.2)");
    return ret;
#else
    static virDomainOpenChannelType virDomainOpenChannelSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainOpenChannelSymbol = libvirtSymbol(libvirt,
                                                       "virDomainOpenChannel",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainOpenChannel");
        return ret;
    }
#  else
    virDomainOpenChannelSymbol = &virDomainOpenChannel;
#  endif

    ret = virDomainOpenChannelSymbol(dom,
                                     name,
                                     st,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 6)
    setVirError(err, "Function virDomainOpenConsole compiled out (from 0.8.6)");
    return ret;
#else
    static virDomainOpenConsoleType virDomainOpenConsoleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainOpenConsoleSymbol = libvirtSymbol(libvirt,
                                                       "virDomainOpenConsole",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainOpenConsole");
        return ret;
    }
#  else
    virDomainOpenConsoleSymbol = &virDomainOpenConsole;
#  endif

    ret = virDomainOpenConsoleSymbol(dom,
                                     dev_name,
                                     st,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainOpenGraphics compiled out (from 0.9.7)");
    return ret;
#else
    static virDomainOpenGraphicsType virDomainOpenGraphicsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainOpenGraphicsSymbol = libvirtSymbol(libvirt,
                                                        "virDomainOpenGraphics",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainOpenGraphics");
        return ret;
    }
#  else
    virDomainOpenGraphicsSymbol = &virDomainOpenGraphics;
#  endif

    ret = virDomainOpenGraphicsSymbol(dom,
                                      idx,
                                      fd,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virDomainOpenGraphicsFD compiled out (from 1.2.8)");
    return ret;
#else
    static virDomainOpenGraphicsFDType virDomainOpenGraphicsFDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainOpenGraphicsFDSymbol = libvirtSymbol(libvirt,
                                                          "virDomainOpenGraphicsFD",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainOpenGraphicsFD");
        return ret;
    }
#  else
    virDomainOpenGraphicsFDSymbol = &virDomainOpenGraphicsFD;
#  endif

    ret = virDomainOpenGraphicsFDSymbol(dom,
                                        idx,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainPMSuspendForDuration compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainPMSuspendForDurationType virDomainPMSuspendForDurationSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainPMSuspendForDurationSymbol = libvirtSymbol(libvirt,
                                                                "virDomainPMSuspendForDuration",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainPMSuspendForDuration");
        return ret;
    }
#  else
    virDomainPMSuspendForDurationSymbol = &virDomainPMSuspendForDuration;
#  endif

    ret = virDomainPMSuspendForDurationSymbol(dom,
                                              target,
                                              duration,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainPMWakeupType)(virDomainPtr dom,
                         unsigned int flags);

int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 11)
    setVirError(err, "Function virDomainPMWakeup compiled out (from 0.9.11)");
    return ret;
#else
    static virDomainPMWakeupType virDomainPMWakeupSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainPMWakeupSymbol = libvirtSymbol(libvirt,
                                                    "virDomainPMWakeup",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainPMWakeup");
        return ret;
    }
#  else
    virDomainPMWakeupSymbol = &virDomainPMWakeup;
#  endif

    ret = virDomainPMWakeupSymbol(dom,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainPinEmulator compiled out (from 0.10.0)");
    return ret;
#else
    static virDomainPinEmulatorType virDomainPinEmulatorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainPinEmulatorSymbol = libvirtSymbol(libvirt,
                                                       "virDomainPinEmulator",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainPinEmulator");
        return ret;
    }
#  else
    virDomainPinEmulatorSymbol = &virDomainPinEmulator;
#  endif

    ret = virDomainPinEmulatorSymbol(domain,
                                     cpumap,
                                     maplen,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(err, "Function virDomainPinIOThread compiled out (from 1.2.14)");
    return ret;
#else
    static virDomainPinIOThreadType virDomainPinIOThreadSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainPinIOThreadSymbol = libvirtSymbol(libvirt,
                                                       "virDomainPinIOThread",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainPinIOThread");
        return ret;
    }
#  else
    virDomainPinIOThreadSymbol = &virDomainPinIOThread;
#  endif

    ret = virDomainPinIOThreadSymbol(domain,
                                     iothread_id,
                                     cpumap,
                                     maplen,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 4)
    setVirError(err, "Function virDomainPinVcpu compiled out (from 0.1.4)");
    return ret;
#else
    static virDomainPinVcpuType virDomainPinVcpuSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainPinVcpuSymbol = libvirtSymbol(libvirt,
                                                   "virDomainPinVcpu",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainPinVcpu");
        return ret;
    }
#  else
    virDomainPinVcpuSymbol = &virDomainPinVcpu;
#  endif

    ret = virDomainPinVcpuSymbol(domain,
                                 vcpu,
                                 cpumap,
                                 maplen);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainPinVcpuFlags compiled out (from 0.9.3)");
    return ret;
#else
    static virDomainPinVcpuFlagsType virDomainPinVcpuFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainPinVcpuFlagsSymbol = libvirtSymbol(libvirt,
                                                        "virDomainPinVcpuFlags",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainPinVcpuFlags");
        return ret;
    }
#  else
    virDomainPinVcpuFlagsSymbol = &virDomainPinVcpuFlags;
#  endif

    ret = virDomainPinVcpuFlagsSymbol(domain,
                                      vcpu,
                                      cpumap,
                                      maplen,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainRebootType)(virDomainPtr domain,
                       unsigned int flags);

int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virDomainReboot compiled out (from 0.1.0)");
    return ret;
#else
    static virDomainRebootType virDomainRebootSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRebootSymbol = libvirtSymbol(libvirt,
                                                  "virDomainReboot",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainReboot");
        return ret;
    }
#  else
    virDomainRebootSymbol = &virDomainReboot;
#  endif

    ret = virDomainRebootSymbol(domain,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainRefType)(virDomainPtr domain);

int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virDomainRef compiled out (from 0.6.0)");
    return ret;
#else
    static virDomainRefType virDomainRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRefSymbol = libvirtSymbol(libvirt,
                                               "virDomainRef",
                                               &success,
                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainRef");
        return ret;
    }
#  else
    virDomainRefSymbol = &virDomainRef;
#  endif

    ret = virDomainRefSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 19)
    setVirError(err, "Function virDomainRename compiled out (from 1.2.19)");
    return ret;
#else
    static virDomainRenameType virDomainRenameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRenameSymbol = libvirtSymbol(libvirt,
                                                  "virDomainRename",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainRename");
        return ret;
    }
#  else
    virDomainRenameSymbol = &virDomainRename;
#  endif

    ret = virDomainRenameSymbol(dom,
                                new_name,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainResetType)(virDomainPtr domain,
                      unsigned int flags);

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainReset compiled out (from 0.9.7)");
    return ret;
#else
    static virDomainResetType virDomainResetSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainResetSymbol = libvirtSymbol(libvirt,
                                                 "virDomainReset",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainReset");
        return ret;
    }
#  else
    virDomainResetSymbol = &virDomainReset;
#  endif

    ret = virDomainResetSymbol(domain,
                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainRestoreType)(virConnectPtr conn,
                        const char * from);

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char * from,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainRestore compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainRestoreType virDomainRestoreSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRestoreSymbol = libvirtSymbol(libvirt,
                                                   "virDomainRestore",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainRestore");
        return ret;
    }
#  else
    virDomainRestoreSymbol = &virDomainRestore;
#  endif

    ret = virDomainRestoreSymbol(conn,
                                 from);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainRestoreFlags compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainRestoreFlagsType virDomainRestoreFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRestoreFlagsSymbol = libvirtSymbol(libvirt,
                                                        "virDomainRestoreFlags",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainRestoreFlags");
        return ret;
    }
#  else
    virDomainRestoreFlagsSymbol = &virDomainRestoreFlags;
#  endif

    ret = virDomainRestoreFlagsSymbol(conn,
                                      from,
                                      dxml,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainRestoreParamsType)(virConnectPtr conn,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags);

int
virDomainRestoreParamsWrapper(virConnectPtr conn,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(8, 4, 0)
    setVirError(err, "Function virDomainRestoreParams compiled out (from 8.4.0)");
    return ret;
#else
    static virDomainRestoreParamsType virDomainRestoreParamsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRestoreParamsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainRestoreParams",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainRestoreParams");
        return ret;
    }
#  else
    virDomainRestoreParamsSymbol = &virDomainRestoreParams;
#  endif

    ret = virDomainRestoreParamsSymbol(conn,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainResumeType)(virDomainPtr domain);

int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainResume compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainResumeType virDomainResumeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainResumeSymbol = libvirtSymbol(libvirt,
                                                  "virDomainResume",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainResume");
        return ret;
    }
#  else
    virDomainResumeSymbol = &virDomainResume;
#  endif

    ret = virDomainResumeSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainRevertToSnapshotType)(virDomainSnapshotPtr snapshot,
                                 unsigned int flags);

int
virDomainRevertToSnapshotWrapper(virDomainSnapshotPtr snapshot,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainRevertToSnapshot compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainRevertToSnapshotType virDomainRevertToSnapshotSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainRevertToSnapshotSymbol = libvirtSymbol(libvirt,
                                                            "virDomainRevertToSnapshot",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainRevertToSnapshot");
        return ret;
    }
#  else
    virDomainRevertToSnapshotSymbol = &virDomainRevertToSnapshot;
#  endif

    ret = virDomainRevertToSnapshotSymbol(snapshot,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSaveType)(virDomainPtr domain,
                     const char * to);

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char * to,
                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainSave compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainSaveType virDomainSaveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSaveSymbol = libvirtSymbol(libvirt,
                                                "virDomainSave",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSave");
        return ret;
    }
#  else
    virDomainSaveSymbol = &virDomainSave;
#  endif

    ret = virDomainSaveSymbol(domain,
                              to);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainSaveFlags compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainSaveFlagsType virDomainSaveFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSaveFlagsSymbol = libvirtSymbol(libvirt,
                                                     "virDomainSaveFlags",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSaveFlags");
        return ret;
    }
#  else
    virDomainSaveFlagsSymbol = &virDomainSaveFlags;
#  endif

    ret = virDomainSaveFlagsSymbol(domain,
                                   to,
                                   dxml,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainSaveImageDefineXML compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainSaveImageDefineXMLType virDomainSaveImageDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSaveImageDefineXMLSymbol = libvirtSymbol(libvirt,
                                                              "virDomainSaveImageDefineXML",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSaveImageDefineXML");
        return ret;
    }
#  else
    virDomainSaveImageDefineXMLSymbol = &virDomainSaveImageDefineXML;
#  endif

    ret = virDomainSaveImageDefineXMLSymbol(conn,
                                            file,
                                            dxml,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainSaveImageGetXMLDesc compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainSaveImageGetXMLDescType virDomainSaveImageGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSaveImageGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                               "virDomainSaveImageGetXMLDesc",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSaveImageGetXMLDesc");
        return ret;
    }
#  else
    virDomainSaveImageGetXMLDescSymbol = &virDomainSaveImageGetXMLDesc;
#  endif

    ret = virDomainSaveImageGetXMLDescSymbol(conn,
                                             file,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSaveParamsType)(virDomainPtr domain,
                           virTypedParameterPtr params,
                           int nparams,
                           unsigned int flags);

int
virDomainSaveParamsWrapper(virDomainPtr domain,
                           virTypedParameterPtr params,
                           int nparams,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(8, 4, 0)
    setVirError(err, "Function virDomainSaveParams compiled out (from 8.4.0)");
    return ret;
#else
    static virDomainSaveParamsType virDomainSaveParamsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSaveParamsSymbol = libvirtSymbol(libvirt,
                                                      "virDomainSaveParams",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSaveParams");
        return ret;
    }
#  else
    virDomainSaveParamsSymbol = &virDomainSaveParams;
#  endif

    ret = virDomainSaveParamsSymbol(domain,
                                    params,
                                    nparams,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainScreenshot compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainScreenshotType virDomainScreenshotSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainScreenshotSymbol = libvirtSymbol(libvirt,
                                                      "virDomainScreenshot",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainScreenshot");
        return ret;
    }
#  else
    virDomainScreenshotSymbol = &virDomainScreenshot;
#  endif

    ret = virDomainScreenshotSymbol(domain,
                                    stream,
                                    screen,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainSendKey compiled out (from 0.9.3)");
    return ret;
#else
    static virDomainSendKeyType virDomainSendKeySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSendKeySymbol = libvirtSymbol(libvirt,
                                                   "virDomainSendKey",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSendKey");
        return ret;
    }
#  else
    virDomainSendKeySymbol = &virDomainSendKey;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 1)
    setVirError(err, "Function virDomainSendProcessSignal compiled out (from 1.0.1)");
    return ret;
#else
    static virDomainSendProcessSignalType virDomainSendProcessSignalSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSendProcessSignalSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSendProcessSignal",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSendProcessSignal");
        return ret;
    }
#  else
    virDomainSendProcessSignalSymbol = &virDomainSendProcessSignal;
#  endif

    ret = virDomainSendProcessSignalSymbol(domain,
                                           pid_value,
                                           signum,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSetAutostartType)(virDomainPtr domain,
                             int autostart);

int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virDomainSetAutostart compiled out (from 0.2.1)");
    return ret;
#else
    static virDomainSetAutostartType virDomainSetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetAutostartSymbol = libvirtSymbol(libvirt,
                                                        "virDomainSetAutostart",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetAutostart");
        return ret;
    }
#  else
    virDomainSetAutostartSymbol = &virDomainSetAutostart;
#  endif

    ret = virDomainSetAutostartSymbol(domain,
                                      autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainSetBlkioParameters compiled out (from 0.9.0)");
    return ret;
#else
    static virDomainSetBlkioParametersType virDomainSetBlkioParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetBlkioParametersSymbol = libvirtSymbol(libvirt,
                                                              "virDomainSetBlkioParameters",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetBlkioParameters");
        return ret;
    }
#  else
    virDomainSetBlkioParametersSymbol = &virDomainSetBlkioParameters;
#  endif

    ret = virDomainSetBlkioParametersSymbol(domain,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virDomainSetBlockIoTune compiled out (from 0.9.8)");
    return ret;
#else
    static virDomainSetBlockIoTuneType virDomainSetBlockIoTuneSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetBlockIoTuneSymbol = libvirtSymbol(libvirt,
                                                          "virDomainSetBlockIoTune",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetBlockIoTune");
        return ret;
    }
#  else
    virDomainSetBlockIoTuneSymbol = &virDomainSetBlockIoTune;
#  endif

    ret = virDomainSetBlockIoTuneSymbol(dom,
                                        disk,
                                        params,
                                        nparams,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 1, 0)
    setVirError(err, "Function virDomainSetBlockThreshold compiled out (from 3.1.0)");
    return ret;
#else
    static virDomainSetBlockThresholdType virDomainSetBlockThresholdSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetBlockThresholdSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSetBlockThreshold",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetBlockThreshold");
        return ret;
    }
#  else
    virDomainSetBlockThresholdSymbol = &virDomainSetBlockThreshold;
#  endif

    ret = virDomainSetBlockThresholdSymbol(domain,
                                           dev,
                                           threshold,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virDomainSetGuestVcpus compiled out (from 2.0.0)");
    return ret;
#else
    static virDomainSetGuestVcpusType virDomainSetGuestVcpusSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetGuestVcpusSymbol = libvirtSymbol(libvirt,
                                                         "virDomainSetGuestVcpus",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetGuestVcpus");
        return ret;
    }
#  else
    virDomainSetGuestVcpusSymbol = &virDomainSetGuestVcpus;
#  endif

    ret = virDomainSetGuestVcpusSymbol(domain,
                                       cpumap,
                                       state,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 10, 0)
    setVirError(err, "Function virDomainSetIOThreadParams compiled out (from 4.10.0)");
    return ret;
#else
    static virDomainSetIOThreadParamsType virDomainSetIOThreadParamsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetIOThreadParamsSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSetIOThreadParams",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetIOThreadParams");
        return ret;
    }
#  else
    virDomainSetIOThreadParamsSymbol = &virDomainSetIOThreadParams;
#  endif

    ret = virDomainSetIOThreadParamsSymbol(domain,
                                           iothread_id,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainSetInterfaceParameters compiled out (from 0.9.9)");
    return ret;
#else
    static virDomainSetInterfaceParametersType virDomainSetInterfaceParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetInterfaceParametersSymbol = libvirtSymbol(libvirt,
                                                                  "virDomainSetInterfaceParameters",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetInterfaceParameters");
        return ret;
    }
#  else
    virDomainSetInterfaceParametersSymbol = &virDomainSetInterfaceParameters;
#  endif

    ret = virDomainSetInterfaceParametersSymbol(domain,
                                                device,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSetLaunchSecurityStateType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags);

int
virDomainSetLaunchSecurityStateWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(8, 0, 0)
    setVirError(err, "Function virDomainSetLaunchSecurityState compiled out (from 8.0.0)");
    return ret;
#else
    static virDomainSetLaunchSecurityStateType virDomainSetLaunchSecurityStateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetLaunchSecurityStateSymbol = libvirtSymbol(libvirt,
                                                                  "virDomainSetLaunchSecurityState",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetLaunchSecurityState");
        return ret;
    }
#  else
    virDomainSetLaunchSecurityStateSymbol = &virDomainSetLaunchSecurityState;
#  endif

    ret = virDomainSetLaunchSecurityStateSymbol(domain,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 9, 0)
    setVirError(err, "Function virDomainSetLifecycleAction compiled out (from 3.9.0)");
    return ret;
#else
    static virDomainSetLifecycleActionType virDomainSetLifecycleActionSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetLifecycleActionSymbol = libvirtSymbol(libvirt,
                                                              "virDomainSetLifecycleAction",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetLifecycleAction");
        return ret;
    }
#  else
    virDomainSetLifecycleActionSymbol = &virDomainSetLifecycleAction;
#  endif

    ret = virDomainSetLifecycleActionSymbol(domain,
                                            type,
                                            action,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSetMaxMemoryType)(virDomainPtr domain,
                             unsigned long memory);

int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainSetMaxMemory compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainSetMaxMemoryType virDomainSetMaxMemorySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetMaxMemorySymbol = libvirtSymbol(libvirt,
                                                        "virDomainSetMaxMemory",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetMaxMemory");
        return ret;
    }
#  else
    virDomainSetMaxMemorySymbol = &virDomainSetMaxMemory;
#  endif

    ret = virDomainSetMaxMemorySymbol(domain,
                                      memory);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSetMemoryType)(virDomainPtr domain,
                          unsigned long memory);

int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainSetMemory compiled out (from 0.1.1)");
    return ret;
#else
    static virDomainSetMemoryType virDomainSetMemorySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetMemorySymbol = libvirtSymbol(libvirt,
                                                     "virDomainSetMemory",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetMemory");
        return ret;
    }
#  else
    virDomainSetMemorySymbol = &virDomainSetMemory;
#  endif

    ret = virDomainSetMemorySymbol(domain,
                                   memory);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainSetMemoryFlags compiled out (from 0.9.0)");
    return ret;
#else
    static virDomainSetMemoryFlagsType virDomainSetMemoryFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetMemoryFlagsSymbol = libvirtSymbol(libvirt,
                                                          "virDomainSetMemoryFlags",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetMemoryFlags");
        return ret;
    }
#  else
    virDomainSetMemoryFlagsSymbol = &virDomainSetMemoryFlags;
#  endif

    ret = virDomainSetMemoryFlagsSymbol(domain,
                                        memory,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainSetMemoryParameters compiled out (from 0.8.5)");
    return ret;
#else
    static virDomainSetMemoryParametersType virDomainSetMemoryParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetMemoryParametersSymbol = libvirtSymbol(libvirt,
                                                               "virDomainSetMemoryParameters",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetMemoryParameters");
        return ret;
    }
#  else
    virDomainSetMemoryParametersSymbol = &virDomainSetMemoryParameters;
#  endif

    ret = virDomainSetMemoryParametersSymbol(domain,
                                             params,
                                             nparams,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 1, 1)
    setVirError(err, "Function virDomainSetMemoryStatsPeriod compiled out (from 1.1.1)");
    return ret;
#else
    static virDomainSetMemoryStatsPeriodType virDomainSetMemoryStatsPeriodSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetMemoryStatsPeriodSymbol = libvirtSymbol(libvirt,
                                                                "virDomainSetMemoryStatsPeriod",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetMemoryStatsPeriod");
        return ret;
    }
#  else
    virDomainSetMemoryStatsPeriodSymbol = &virDomainSetMemoryStatsPeriod;
#  endif

    ret = virDomainSetMemoryStatsPeriodSymbol(domain,
                                              period,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainSetMetadata compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainSetMetadataType virDomainSetMetadataSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetMetadataSymbol = libvirtSymbol(libvirt,
                                                       "virDomainSetMetadata",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetMetadata");
        return ret;
    }
#  else
    virDomainSetMetadataSymbol = &virDomainSetMetadata;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainSetNumaParameters compiled out (from 0.9.9)");
    return ret;
#else
    static virDomainSetNumaParametersType virDomainSetNumaParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetNumaParametersSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSetNumaParameters",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetNumaParameters");
        return ret;
    }
#  else
    virDomainSetNumaParametersSymbol = &virDomainSetNumaParameters;
#  endif

    ret = virDomainSetNumaParametersSymbol(domain,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 3, 3)
    setVirError(err, "Function virDomainSetPerfEvents compiled out (from 1.3.3)");
    return ret;
#else
    static virDomainSetPerfEventsType virDomainSetPerfEventsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetPerfEventsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainSetPerfEvents",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetPerfEvents");
        return ret;
    }
#  else
    virDomainSetPerfEventsSymbol = &virDomainSetPerfEvents;
#  endif

    ret = virDomainSetPerfEventsSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 3)
    setVirError(err, "Function virDomainSetSchedulerParameters compiled out (from 0.2.3)");
    return ret;
#else
    static virDomainSetSchedulerParametersType virDomainSetSchedulerParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetSchedulerParametersSymbol = libvirtSymbol(libvirt,
                                                                  "virDomainSetSchedulerParameters",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetSchedulerParameters");
        return ret;
    }
#  else
    virDomainSetSchedulerParametersSymbol = &virDomainSetSchedulerParameters;
#  endif

    ret = virDomainSetSchedulerParametersSymbol(domain,
                                                params,
                                                nparams);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainSetSchedulerParametersFlags compiled out (from 0.9.2)");
    return ret;
#else
    static virDomainSetSchedulerParametersFlagsType virDomainSetSchedulerParametersFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetSchedulerParametersFlagsSymbol = libvirtSymbol(libvirt,
                                                                       "virDomainSetSchedulerParametersFlags",
                                                                       &success,
                                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetSchedulerParametersFlags");
        return ret;
    }
#  else
    virDomainSetSchedulerParametersFlagsSymbol = &virDomainSetSchedulerParametersFlags;
#  endif

    ret = virDomainSetSchedulerParametersFlagsSymbol(domain,
                                                     params,
                                                     nparams,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainSetTime compiled out (from 1.2.5)");
    return ret;
#else
    static virDomainSetTimeType virDomainSetTimeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetTimeSymbol = libvirtSymbol(libvirt,
                                                   "virDomainSetTime",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetTime");
        return ret;
    }
#  else
    virDomainSetTimeSymbol = &virDomainSetTime;
#  endif

    ret = virDomainSetTimeSymbol(dom,
                                 seconds,
                                 nseconds,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 16)
    setVirError(err, "Function virDomainSetUserPassword compiled out (from 1.2.16)");
    return ret;
#else
    static virDomainSetUserPasswordType virDomainSetUserPasswordSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetUserPasswordSymbol = libvirtSymbol(libvirt,
                                                           "virDomainSetUserPassword",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetUserPassword");
        return ret;
    }
#  else
    virDomainSetUserPasswordSymbol = &virDomainSetUserPassword;
#  endif

    ret = virDomainSetUserPasswordSymbol(dom,
                                         user,
                                         password,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 1, 0)
    setVirError(err, "Function virDomainSetVcpu compiled out (from 3.1.0)");
    return ret;
#else
    static virDomainSetVcpuType virDomainSetVcpuSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetVcpuSymbol = libvirtSymbol(libvirt,
                                                   "virDomainSetVcpu",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetVcpu");
        return ret;
    }
#  else
    virDomainSetVcpuSymbol = &virDomainSetVcpu;
#  endif

    ret = virDomainSetVcpuSymbol(domain,
                                 vcpumap,
                                 state,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSetVcpusType)(virDomainPtr domain,
                         unsigned int nvcpus);

int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 4)
    setVirError(err, "Function virDomainSetVcpus compiled out (from 0.1.4)");
    return ret;
#else
    static virDomainSetVcpusType virDomainSetVcpusSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetVcpusSymbol = libvirtSymbol(libvirt,
                                                    "virDomainSetVcpus",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetVcpus");
        return ret;
    }
#  else
    virDomainSetVcpusSymbol = &virDomainSetVcpus;
#  endif

    ret = virDomainSetVcpusSymbol(domain,
                                  nvcpus);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainSetVcpusFlags compiled out (from 0.8.5)");
    return ret;
#else
    static virDomainSetVcpusFlagsType virDomainSetVcpusFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSetVcpusFlagsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainSetVcpusFlags",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSetVcpusFlags");
        return ret;
    }
#  else
    virDomainSetVcpusFlagsSymbol = &virDomainSetVcpusFlags;
#  endif

    ret = virDomainSetVcpusFlagsSymbol(domain,
                                       nvcpus,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainShutdownType)(virDomainPtr domain);

int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainShutdown compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainShutdownType virDomainShutdownSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainShutdownSymbol = libvirtSymbol(libvirt,
                                                    "virDomainShutdown",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainShutdown");
        return ret;
    }
#  else
    virDomainShutdownSymbol = &virDomainShutdown;
#  endif

    ret = virDomainShutdownSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainShutdownFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainShutdownFlags compiled out (from 0.9.10)");
    return ret;
#else
    static virDomainShutdownFlagsType virDomainShutdownFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainShutdownFlagsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainShutdownFlags",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainShutdownFlags");
        return ret;
    }
#  else
    virDomainShutdownFlagsSymbol = &virDomainShutdownFlags;
#  endif

    ret = virDomainShutdownFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainSnapshotPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotCreateXML compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotCreateXMLType virDomainSnapshotCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotCreateXMLSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSnapshotCreateXML",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotCreateXML");
        return ret;
    }
#  else
    virDomainSnapshotCreateXMLSymbol = &virDomainSnapshotCreateXML;
#  endif

    ret = virDomainSnapshotCreateXMLSymbol(domain,
                                           xmlDesc,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotCurrentType)(virDomainPtr domain,
                                unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotCurrentWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotCurrent compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotCurrentType virDomainSnapshotCurrentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotCurrentSymbol = libvirtSymbol(libvirt,
                                                           "virDomainSnapshotCurrent",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotCurrent");
        return ret;
    }
#  else
    virDomainSnapshotCurrentSymbol = &virDomainSnapshotCurrent;
#  endif

    ret = virDomainSnapshotCurrentSymbol(domain,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotDeleteType)(virDomainSnapshotPtr snapshot,
                               unsigned int flags);

int
virDomainSnapshotDeleteWrapper(virDomainSnapshotPtr snapshot,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotDelete compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotDeleteType virDomainSnapshotDeleteSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotDeleteSymbol = libvirtSymbol(libvirt,
                                                          "virDomainSnapshotDelete",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotDelete");
        return ret;
    }
#  else
    virDomainSnapshotDeleteSymbol = &virDomainSnapshotDelete;
#  endif

    ret = virDomainSnapshotDeleteSymbol(snapshot,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotFreeType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotFreeWrapper(virDomainSnapshotPtr snapshot,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotFree compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotFreeType virDomainSnapshotFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotFreeSymbol = libvirtSymbol(libvirt,
                                                        "virDomainSnapshotFree",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotFree");
        return ret;
    }
#  else
    virDomainSnapshotFreeSymbol = &virDomainSnapshotFree;
#  endif

    ret = virDomainSnapshotFreeSymbol(snapshot);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virDomainSnapshotGetConnectType)(virDomainSnapshotPtr snapshot);

virConnectPtr
virDomainSnapshotGetConnectWrapper(virDomainSnapshotPtr snapshot,
                                   virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainSnapshotGetConnect compiled out (from 0.9.5)");
    return ret;
#else
    static virDomainSnapshotGetConnectType virDomainSnapshotGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotGetConnectSymbol = libvirtSymbol(libvirt,
                                                              "virDomainSnapshotGetConnect",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotGetConnect");
        return ret;
    }
#  else
    virDomainSnapshotGetConnectSymbol = &virDomainSnapshotGetConnect;
#  endif

    ret = virDomainSnapshotGetConnectSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainPtr
(*virDomainSnapshotGetDomainType)(virDomainSnapshotPtr snapshot);

virDomainPtr
virDomainSnapshotGetDomainWrapper(virDomainSnapshotPtr snapshot,
                                  virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainSnapshotGetDomain compiled out (from 0.9.5)");
    return ret;
#else
    static virDomainSnapshotGetDomainType virDomainSnapshotGetDomainSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotGetDomainSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSnapshotGetDomain",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotGetDomain");
        return ret;
    }
#  else
    virDomainSnapshotGetDomainSymbol = &virDomainSnapshotGetDomain;
#  endif

    ret = virDomainSnapshotGetDomainSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virDomainSnapshotGetNameType)(virDomainSnapshotPtr snapshot);

const char *
virDomainSnapshotGetNameWrapper(virDomainSnapshotPtr snapshot,
                                virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainSnapshotGetName compiled out (from 0.9.5)");
    return ret;
#else
    static virDomainSnapshotGetNameType virDomainSnapshotGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotGetNameSymbol = libvirtSymbol(libvirt,
                                                           "virDomainSnapshotGetName",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotGetName");
        return ret;
    }
#  else
    virDomainSnapshotGetNameSymbol = &virDomainSnapshotGetName;
#  endif

    ret = virDomainSnapshotGetNameSymbol(snapshot);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virDomainSnapshotPtr
(*virDomainSnapshotGetParentType)(virDomainSnapshotPtr snapshot,
                                  unsigned int flags);

virDomainSnapshotPtr
virDomainSnapshotGetParentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    virDomainSnapshotPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainSnapshotGetParent compiled out (from 0.9.7)");
    return ret;
#else
    static virDomainSnapshotGetParentType virDomainSnapshotGetParentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotGetParentSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSnapshotGetParent",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotGetParent");
        return ret;
    }
#  else
    virDomainSnapshotGetParentSymbol = &virDomainSnapshotGetParent;
#  endif

    ret = virDomainSnapshotGetParentSymbol(snapshot,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virDomainSnapshotGetXMLDescType)(virDomainSnapshotPtr snapshot,
                                   unsigned int flags);

char *
virDomainSnapshotGetXMLDescWrapper(virDomainSnapshotPtr snapshot,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotGetXMLDesc compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotGetXMLDescType virDomainSnapshotGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                              "virDomainSnapshotGetXMLDesc",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotGetXMLDesc");
        return ret;
    }
#  else
    virDomainSnapshotGetXMLDescSymbol = &virDomainSnapshotGetXMLDesc;
#  endif

    ret = virDomainSnapshotGetXMLDescSymbol(snapshot,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotHasMetadataType)(virDomainSnapshotPtr snapshot,
                                    unsigned int flags);

int
virDomainSnapshotHasMetadataWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotHasMetadata compiled out (from 0.9.13)");
    return ret;
#else
    static virDomainSnapshotHasMetadataType virDomainSnapshotHasMetadataSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotHasMetadataSymbol = libvirtSymbol(libvirt,
                                                               "virDomainSnapshotHasMetadata",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotHasMetadata");
        return ret;
    }
#  else
    virDomainSnapshotHasMetadataSymbol = &virDomainSnapshotHasMetadata;
#  endif

    ret = virDomainSnapshotHasMetadataSymbol(snapshot,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotIsCurrentType)(virDomainSnapshotPtr snapshot,
                                  unsigned int flags);

int
virDomainSnapshotIsCurrentWrapper(virDomainSnapshotPtr snapshot,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotIsCurrent compiled out (from 0.9.13)");
    return ret;
#else
    static virDomainSnapshotIsCurrentType virDomainSnapshotIsCurrentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotIsCurrentSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSnapshotIsCurrent",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotIsCurrent");
        return ret;
    }
#  else
    virDomainSnapshotIsCurrentSymbol = &virDomainSnapshotIsCurrent;
#  endif

    ret = virDomainSnapshotIsCurrentSymbol(snapshot,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotListAllChildren compiled out (from 0.9.13)");
    return ret;
#else
    static virDomainSnapshotListAllChildrenType virDomainSnapshotListAllChildrenSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotListAllChildrenSymbol = libvirtSymbol(libvirt,
                                                                   "virDomainSnapshotListAllChildren",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotListAllChildren");
        return ret;
    }
#  else
    virDomainSnapshotListAllChildrenSymbol = &virDomainSnapshotListAllChildren;
#  endif

    ret = virDomainSnapshotListAllChildrenSymbol(snapshot,
                                                 snaps,
                                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainSnapshotListChildrenNames compiled out (from 0.9.7)");
    return ret;
#else
    static virDomainSnapshotListChildrenNamesType virDomainSnapshotListChildrenNamesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotListChildrenNamesSymbol = libvirtSymbol(libvirt,
                                                                     "virDomainSnapshotListChildrenNames",
                                                                     &success,
                                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotListChildrenNames");
        return ret;
    }
#  else
    virDomainSnapshotListChildrenNamesSymbol = &virDomainSnapshotListChildrenNames;
#  endif

    ret = virDomainSnapshotListChildrenNamesSymbol(snapshot,
                                                   names,
                                                   nameslen,
                                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotListNames compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotListNamesType virDomainSnapshotListNamesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotListNamesSymbol = libvirtSymbol(libvirt,
                                                             "virDomainSnapshotListNames",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotListNames");
        return ret;
    }
#  else
    virDomainSnapshotListNamesSymbol = &virDomainSnapshotListNames;
#  endif

    ret = virDomainSnapshotListNamesSymbol(domain,
                                           names,
                                           nameslen,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainSnapshotPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotLookupByName compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotLookupByNameType virDomainSnapshotLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotLookupByNameSymbol = libvirtSymbol(libvirt,
                                                                "virDomainSnapshotLookupByName",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotLookupByName");
        return ret;
    }
#  else
    virDomainSnapshotLookupByNameSymbol = &virDomainSnapshotLookupByName;
#  endif

    ret = virDomainSnapshotLookupByNameSymbol(domain,
                                              name,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotNumType)(virDomainPtr domain,
                            unsigned int flags);

int
virDomainSnapshotNumWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainSnapshotNum compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainSnapshotNumType virDomainSnapshotNumSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotNumSymbol = libvirtSymbol(libvirt,
                                                       "virDomainSnapshotNum",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotNum");
        return ret;
    }
#  else
    virDomainSnapshotNumSymbol = &virDomainSnapshotNum;
#  endif

    ret = virDomainSnapshotNumSymbol(domain,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotNumChildrenType)(virDomainSnapshotPtr snapshot,
                                    unsigned int flags);

int
virDomainSnapshotNumChildrenWrapper(virDomainSnapshotPtr snapshot,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainSnapshotNumChildren compiled out (from 0.9.7)");
    return ret;
#else
    static virDomainSnapshotNumChildrenType virDomainSnapshotNumChildrenSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotNumChildrenSymbol = libvirtSymbol(libvirt,
                                                               "virDomainSnapshotNumChildren",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotNumChildren");
        return ret;
    }
#  else
    virDomainSnapshotNumChildrenSymbol = &virDomainSnapshotNumChildren;
#  endif

    ret = virDomainSnapshotNumChildrenSymbol(snapshot,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainSnapshotRefType)(virDomainSnapshotPtr snapshot);

int
virDomainSnapshotRefWrapper(virDomainSnapshotPtr snapshot,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virDomainSnapshotRef compiled out (from 0.9.13)");
    return ret;
#else
    static virDomainSnapshotRefType virDomainSnapshotRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSnapshotRefSymbol = libvirtSymbol(libvirt,
                                                       "virDomainSnapshotRef",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSnapshotRef");
        return ret;
    }
#  else
    virDomainSnapshotRefSymbol = &virDomainSnapshotRef;
#  endif

    ret = virDomainSnapshotRefSymbol(snapshot);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 2, 0)
    setVirError(err, "Function virDomainStartDirtyRateCalc compiled out (from 7.2.0)");
    return ret;
#else
    static virDomainStartDirtyRateCalcType virDomainStartDirtyRateCalcSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainStartDirtyRateCalcSymbol = libvirtSymbol(libvirt,
                                                              "virDomainStartDirtyRateCalc",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainStartDirtyRateCalc");
        return ret;
    }
#  else
    virDomainStartDirtyRateCalcSymbol = &virDomainStartDirtyRateCalc;
#  endif

    ret = virDomainStartDirtyRateCalcSymbol(domain,
                                            seconds,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virDomainStatsRecordListFreeType)(virDomainStatsRecordPtr * stats);

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(NULL, "Function virDomainStatsRecordListFree compiled out (from 1.2.8)");
    return;
#else
    static virDomainStatsRecordListFreeType virDomainStatsRecordListFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virDomainStatsRecordListFreeSymbol = libvirtSymbol(libvirt,
                                                               "virDomainStatsRecordListFree",
                                                               &success,
                                                               NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virDomainStatsRecordListFree");
        return;
    }
#  else
    virDomainStatsRecordListFreeSymbol = &virDomainStatsRecordListFree;
#  endif

    virDomainStatsRecordListFreeSymbol(stats);
#endif
}

typedef int
(*virDomainSuspendType)(virDomainPtr domain);

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainSuspend compiled out (from 0.0.3)");
    return ret;
#else
    static virDomainSuspendType virDomainSuspendSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainSuspendSymbol = libvirtSymbol(libvirt,
                                                   "virDomainSuspend",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainSuspend");
        return ret;
    }
#  else
    virDomainSuspendSymbol = &virDomainSuspend;
#  endif

    ret = virDomainSuspendSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainUndefineType)(virDomainPtr domain);

int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainUndefine compiled out (from 0.1.1)");
    return ret;
#else
    static virDomainUndefineType virDomainUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainUndefineSymbol = libvirtSymbol(libvirt,
                                                    "virDomainUndefine",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainUndefine");
        return ret;
    }
#  else
    virDomainUndefineSymbol = &virDomainUndefine;
#  endif

    ret = virDomainUndefineSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainUndefineFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainUndefineFlags compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainUndefineFlagsType virDomainUndefineFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainUndefineFlagsSymbol = libvirtSymbol(libvirt,
                                                         "virDomainUndefineFlags",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainUndefineFlags");
        return ret;
    }
#  else
    virDomainUndefineFlagsSymbol = &virDomainUndefineFlags;
#  endif

    ret = virDomainUndefineFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainUpdateDeviceFlags compiled out (from 0.8.0)");
    return ret;
#else
    static virDomainUpdateDeviceFlagsType virDomainUpdateDeviceFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainUpdateDeviceFlagsSymbol = libvirtSymbol(libvirt,
                                                             "virDomainUpdateDeviceFlags",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainUpdateDeviceFlags");
        return ret;
    }
#  else
    virDomainUpdateDeviceFlagsSymbol = &virDomainUpdateDeviceFlags;
#  endif

    ret = virDomainUpdateDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventAddHandle compiled out (from 0.9.3)");
    return ret;
#else
    static virEventAddHandleType virEventAddHandleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virEventAddHandleSymbol = libvirtSymbol(libvirt,
                                                    "virEventAddHandle",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virEventAddHandle");
        return ret;
    }
#  else
    virEventAddHandleSymbol = &virEventAddHandle;
#  endif

    ret = virEventAddHandleSymbol(fd,
                                  events,
                                  cb,
                                  opaque,
                                  ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventAddTimeout compiled out (from 0.9.3)");
    return ret;
#else
    static virEventAddTimeoutType virEventAddTimeoutSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virEventAddTimeoutSymbol = libvirtSymbol(libvirt,
                                                     "virEventAddTimeout",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virEventAddTimeout");
        return ret;
    }
#  else
    virEventAddTimeoutSymbol = &virEventAddTimeout;
#  endif

    ret = virEventAddTimeoutSymbol(timeout,
                                   cb,
                                   opaque,
                                   ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virEventRegisterDefaultImplType)(void);

int
virEventRegisterDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virEventRegisterDefaultImpl compiled out (from 0.9.0)");
    return ret;
#else
    static virEventRegisterDefaultImplType virEventRegisterDefaultImplSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virEventRegisterDefaultImplSymbol = libvirtSymbol(libvirt,
                                                              "virEventRegisterDefaultImpl",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virEventRegisterDefaultImpl");
        return ret;
    }
#  else
    virEventRegisterDefaultImplSymbol = &virEventRegisterDefaultImpl;
#  endif

    ret = virEventRegisterDefaultImplSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(NULL, "Function virEventRegisterImpl compiled out (from 0.5.0)");
    return;
#else
    static virEventRegisterImplType virEventRegisterImplSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virEventRegisterImplSymbol = libvirtSymbol(libvirt,
                                                       "virEventRegisterImpl",
                                                       &success,
                                                       NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virEventRegisterImpl");
        return;
    }
#  else
    virEventRegisterImplSymbol = &virEventRegisterImpl;
#  endif

    virEventRegisterImplSymbol(addHandle,
                               updateHandle,
                               removeHandle,
                               addTimeout,
                               updateTimeout,
                               removeTimeout);
#endif
}

typedef int
(*virEventRemoveHandleType)(int watch);

int
virEventRemoveHandleWrapper(int watch,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventRemoveHandle compiled out (from 0.9.3)");
    return ret;
#else
    static virEventRemoveHandleType virEventRemoveHandleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virEventRemoveHandleSymbol = libvirtSymbol(libvirt,
                                                       "virEventRemoveHandle",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virEventRemoveHandle");
        return ret;
    }
#  else
    virEventRemoveHandleSymbol = &virEventRemoveHandle;
#  endif

    ret = virEventRemoveHandleSymbol(watch);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virEventRemoveTimeoutType)(int timer);

int
virEventRemoveTimeoutWrapper(int timer,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventRemoveTimeout compiled out (from 0.9.3)");
    return ret;
#else
    static virEventRemoveTimeoutType virEventRemoveTimeoutSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virEventRemoveTimeoutSymbol = libvirtSymbol(libvirt,
                                                        "virEventRemoveTimeout",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virEventRemoveTimeout");
        return ret;
    }
#  else
    virEventRemoveTimeoutSymbol = &virEventRemoveTimeout;
#  endif

    ret = virEventRemoveTimeoutSymbol(timer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virEventRunDefaultImplType)(void);

int
virEventRunDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virEventRunDefaultImpl compiled out (from 0.9.0)");
    return ret;
#else
    static virEventRunDefaultImplType virEventRunDefaultImplSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virEventRunDefaultImplSymbol = libvirtSymbol(libvirt,
                                                         "virEventRunDefaultImpl",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virEventRunDefaultImpl");
        return ret;
    }
#  else
    virEventRunDefaultImplSymbol = &virEventRunDefaultImpl;
#  endif

    ret = virEventRunDefaultImplSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virEventUpdateHandleType)(int watch,
                            int events);

void
virEventUpdateHandleWrapper(int watch,
                            int events)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(NULL, "Function virEventUpdateHandle compiled out (from 0.9.3)");
    return;
#else
    static virEventUpdateHandleType virEventUpdateHandleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virEventUpdateHandleSymbol = libvirtSymbol(libvirt,
                                                       "virEventUpdateHandle",
                                                       &success,
                                                       NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virEventUpdateHandle");
        return;
    }
#  else
    virEventUpdateHandleSymbol = &virEventUpdateHandle;
#  endif

    virEventUpdateHandleSymbol(watch,
                               events);
#endif
}

typedef void
(*virEventUpdateTimeoutType)(int timer,
                             int timeout);

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(NULL, "Function virEventUpdateTimeout compiled out (from 0.9.3)");
    return;
#else
    static virEventUpdateTimeoutType virEventUpdateTimeoutSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virEventUpdateTimeoutSymbol = libvirtSymbol(libvirt,
                                                        "virEventUpdateTimeout",
                                                        &success,
                                                        NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virEventUpdateTimeout");
        return;
    }
#  else
    virEventUpdateTimeoutSymbol = &virEventUpdateTimeout;
#  endif

    virEventUpdateTimeoutSymbol(timer,
                                timeout);
#endif
}

typedef void
(*virFreeErrorType)(virErrorPtr err);

void
virFreeErrorWrapper(virErrorPtr err)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(NULL, "Function virFreeError compiled out (from 0.6.1)");
    return;
#else
    static virFreeErrorType virFreeErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virFreeErrorSymbol = libvirtSymbol(libvirt,
                                               "virFreeError",
                                               &success,
                                               NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virFreeError");
        return;
    }
#  else
    virFreeErrorSymbol = &virFreeError;
#  endif

    virFreeErrorSymbol(err);
#endif
}

typedef virErrorPtr
(*virGetLastErrorType)(void);

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virGetLastError compiled out (from 0.1.0)");
    return ret;
#else
    static virGetLastErrorType virGetLastErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virGetLastErrorSymbol = libvirtSymbol(libvirt,
                                                  "virGetLastError",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virGetLastError");
        return ret;
    }
#  else
    virGetLastErrorSymbol = &virGetLastError;
#  endif

    ret = virGetLastErrorSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virGetLastErrorCodeType)(void);

int
virGetLastErrorCodeWrapper(virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virGetLastErrorCode compiled out (from 4.5.0)");
    return ret;
#else
    static virGetLastErrorCodeType virGetLastErrorCodeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virGetLastErrorCodeSymbol = libvirtSymbol(libvirt,
                                                      "virGetLastErrorCode",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virGetLastErrorCode");
        return ret;
    }
#  else
    virGetLastErrorCodeSymbol = &virGetLastErrorCode;
#  endif

    ret = virGetLastErrorCodeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virGetLastErrorDomainType)(void);

int
virGetLastErrorDomainWrapper(virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virGetLastErrorDomain compiled out (from 4.5.0)");
    return ret;
#else
    static virGetLastErrorDomainType virGetLastErrorDomainSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virGetLastErrorDomainSymbol = libvirtSymbol(libvirt,
                                                        "virGetLastErrorDomain",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virGetLastErrorDomain");
        return ret;
    }
#  else
    virGetLastErrorDomainSymbol = &virGetLastErrorDomain;
#  endif

    ret = virGetLastErrorDomainSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virGetLastErrorMessageType)(void);

const char *
virGetLastErrorMessageWrapper(virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 6)
    setVirError(err, "Function virGetLastErrorMessage compiled out (from 1.0.6)");
    return ret;
#else
    static virGetLastErrorMessageType virGetLastErrorMessageSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virGetLastErrorMessageSymbol = libvirtSymbol(libvirt,
                                                         "virGetLastErrorMessage",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virGetLastErrorMessage");
        return ret;
    }
#  else
    virGetLastErrorMessageSymbol = &virGetLastErrorMessage;
#  endif

    ret = virGetLastErrorMessageSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virGetVersion compiled out (from 0.0.3)");
    return ret;
#else
    static virGetVersionType virGetVersionSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virGetVersionSymbol = libvirtSymbol(libvirt,
                                                "virGetVersion",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virGetVersion");
        return ret;
    }
#  else
    virGetVersionSymbol = &virGetVersion;
#  endif

    ret = virGetVersionSymbol(libVer,
                              type,
                              typeVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInitializeType)(void);

int
virInitializeWrapper(virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virInitialize compiled out (from 0.1.0)");
    return ret;
#else
    static virInitializeType virInitializeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInitializeSymbol = libvirtSymbol(libvirt,
                                                "virInitialize",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInitialize");
        return ret;
    }
#  else
    virInitializeSymbol = &virInitialize;
    libvirtLoadLibvirtVariables();
#  endif

    ret = virInitializeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceChangeBeginType)(virConnectPtr conn,
                               unsigned int flags);

int
virInterfaceChangeBeginWrapper(virConnectPtr conn,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virInterfaceChangeBegin compiled out (from 0.9.2)");
    return ret;
#else
    static virInterfaceChangeBeginType virInterfaceChangeBeginSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceChangeBeginSymbol = libvirtSymbol(libvirt,
                                                          "virInterfaceChangeBegin",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceChangeBegin");
        return ret;
    }
#  else
    virInterfaceChangeBeginSymbol = &virInterfaceChangeBegin;
#  endif

    ret = virInterfaceChangeBeginSymbol(conn,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceChangeCommitType)(virConnectPtr conn,
                                unsigned int flags);

int
virInterfaceChangeCommitWrapper(virConnectPtr conn,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virInterfaceChangeCommit compiled out (from 0.9.2)");
    return ret;
#else
    static virInterfaceChangeCommitType virInterfaceChangeCommitSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceChangeCommitSymbol = libvirtSymbol(libvirt,
                                                           "virInterfaceChangeCommit",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceChangeCommit");
        return ret;
    }
#  else
    virInterfaceChangeCommitSymbol = &virInterfaceChangeCommit;
#  endif

    ret = virInterfaceChangeCommitSymbol(conn,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceChangeRollbackType)(virConnectPtr conn,
                                  unsigned int flags);

int
virInterfaceChangeRollbackWrapper(virConnectPtr conn,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virInterfaceChangeRollback compiled out (from 0.9.2)");
    return ret;
#else
    static virInterfaceChangeRollbackType virInterfaceChangeRollbackSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceChangeRollbackSymbol = libvirtSymbol(libvirt,
                                                             "virInterfaceChangeRollback",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceChangeRollback");
        return ret;
    }
#  else
    virInterfaceChangeRollbackSymbol = &virInterfaceChangeRollback;
#  endif

    ret = virInterfaceChangeRollbackSymbol(conn,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceCreateType)(virInterfacePtr iface,
                          unsigned int flags);

int
virInterfaceCreateWrapper(virInterfacePtr iface,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceCreate compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceCreateType virInterfaceCreateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceCreateSymbol = libvirtSymbol(libvirt,
                                                     "virInterfaceCreate",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceCreate");
        return ret;
    }
#  else
    virInterfaceCreateSymbol = &virInterfaceCreate;
#  endif

    ret = virInterfaceCreateSymbol(iface,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virInterfacePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceDefineXML compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceDefineXMLType virInterfaceDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceDefineXMLSymbol = libvirtSymbol(libvirt,
                                                        "virInterfaceDefineXML",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceDefineXML");
        return ret;
    }
#  else
    virInterfaceDefineXMLSymbol = &virInterfaceDefineXML;
#  endif

    ret = virInterfaceDefineXMLSymbol(conn,
                                      xml,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceDestroyType)(virInterfacePtr iface,
                           unsigned int flags);

int
virInterfaceDestroyWrapper(virInterfacePtr iface,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceDestroy compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceDestroyType virInterfaceDestroySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceDestroySymbol = libvirtSymbol(libvirt,
                                                      "virInterfaceDestroy",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceDestroy");
        return ret;
    }
#  else
    virInterfaceDestroySymbol = &virInterfaceDestroy;
#  endif

    ret = virInterfaceDestroySymbol(iface,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceFreeType)(virInterfacePtr iface);

int
virInterfaceFreeWrapper(virInterfacePtr iface,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceFree compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceFreeType virInterfaceFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceFreeSymbol = libvirtSymbol(libvirt,
                                                   "virInterfaceFree",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceFree");
        return ret;
    }
#  else
    virInterfaceFreeSymbol = &virInterfaceFree;
#  endif

    ret = virInterfaceFreeSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virInterfaceGetConnectType)(virInterfacePtr iface);

virConnectPtr
virInterfaceGetConnectWrapper(virInterfacePtr iface,
                              virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetConnect compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceGetConnectType virInterfaceGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceGetConnectSymbol = libvirtSymbol(libvirt,
                                                         "virInterfaceGetConnect",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceGetConnect");
        return ret;
    }
#  else
    virInterfaceGetConnectSymbol = &virInterfaceGetConnect;
#  endif

    ret = virInterfaceGetConnectSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virInterfaceGetMACStringType)(virInterfacePtr iface);

const char *
virInterfaceGetMACStringWrapper(virInterfacePtr iface,
                                virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetMACString compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceGetMACStringType virInterfaceGetMACStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceGetMACStringSymbol = libvirtSymbol(libvirt,
                                                           "virInterfaceGetMACString",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceGetMACString");
        return ret;
    }
#  else
    virInterfaceGetMACStringSymbol = &virInterfaceGetMACString;
#  endif

    ret = virInterfaceGetMACStringSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virInterfaceGetNameType)(virInterfacePtr iface);

const char *
virInterfaceGetNameWrapper(virInterfacePtr iface,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetName compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceGetNameType virInterfaceGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceGetNameSymbol = libvirtSymbol(libvirt,
                                                      "virInterfaceGetName",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceGetName");
        return ret;
    }
#  else
    virInterfaceGetNameSymbol = &virInterfaceGetName;
#  endif

    ret = virInterfaceGetNameSymbol(iface);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virInterfaceGetXMLDescType)(virInterfacePtr iface,
                              unsigned int flags);

char *
virInterfaceGetXMLDescWrapper(virInterfacePtr iface,
                              unsigned int flags,
                              virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceGetXMLDesc compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceGetXMLDescType virInterfaceGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                         "virInterfaceGetXMLDesc",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceGetXMLDesc");
        return ret;
    }
#  else
    virInterfaceGetXMLDescSymbol = &virInterfaceGetXMLDesc;
#  endif

    ret = virInterfaceGetXMLDescSymbol(iface,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceIsActiveType)(virInterfacePtr iface);

int
virInterfaceIsActiveWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virInterfaceIsActive compiled out (from 0.7.3)");
    return ret;
#else
    static virInterfaceIsActiveType virInterfaceIsActiveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceIsActiveSymbol = libvirtSymbol(libvirt,
                                                       "virInterfaceIsActive",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceIsActive");
        return ret;
    }
#  else
    virInterfaceIsActiveSymbol = &virInterfaceIsActive;
#  endif

    ret = virInterfaceIsActiveSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virInterfacePtr
(*virInterfaceLookupByMACStringType)(virConnectPtr conn,
                                     const char * macstr);

virInterfacePtr
virInterfaceLookupByMACStringWrapper(virConnectPtr conn,
                                     const char * macstr,
                                     virErrorPtr err)
{
    virInterfacePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceLookupByMACString compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceLookupByMACStringType virInterfaceLookupByMACStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceLookupByMACStringSymbol = libvirtSymbol(libvirt,
                                                                "virInterfaceLookupByMACString",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceLookupByMACString");
        return ret;
    }
#  else
    virInterfaceLookupByMACStringSymbol = &virInterfaceLookupByMACString;
#  endif

    ret = virInterfaceLookupByMACStringSymbol(conn,
                                              macstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virInterfacePtr
(*virInterfaceLookupByNameType)(virConnectPtr conn,
                                const char * name);

virInterfacePtr
virInterfaceLookupByNameWrapper(virConnectPtr conn,
                                const char * name,
                                virErrorPtr err)
{
    virInterfacePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceLookupByName compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceLookupByNameType virInterfaceLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceLookupByNameSymbol = libvirtSymbol(libvirt,
                                                           "virInterfaceLookupByName",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceLookupByName");
        return ret;
    }
#  else
    virInterfaceLookupByNameSymbol = &virInterfaceLookupByName;
#  endif

    ret = virInterfaceLookupByNameSymbol(conn,
                                         name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceRefType)(virInterfacePtr iface);

int
virInterfaceRefWrapper(virInterfacePtr iface,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceRef compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceRefType virInterfaceRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceRefSymbol = libvirtSymbol(libvirt,
                                                  "virInterfaceRef",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceRef");
        return ret;
    }
#  else
    virInterfaceRefSymbol = &virInterfaceRef;
#  endif

    ret = virInterfaceRefSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virInterfaceUndefineType)(virInterfacePtr iface);

int
virInterfaceUndefineWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virInterfaceUndefine compiled out (from 0.6.4)");
    return ret;
#else
    static virInterfaceUndefineType virInterfaceUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virInterfaceUndefineSymbol = libvirtSymbol(libvirt,
                                                       "virInterfaceUndefine",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virInterfaceUndefine");
        return ret;
    }
#  else
    virInterfaceUndefineSymbol = &virInterfaceUndefine;
#  endif

    ret = virInterfaceUndefineSymbol(iface);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingCreateXML compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingCreateXMLType virNWFilterBindingCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingCreateXMLSymbol = libvirtSymbol(libvirt,
                                                              "virNWFilterBindingCreateXML",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingCreateXML");
        return ret;
    }
#  else
    virNWFilterBindingCreateXMLSymbol = &virNWFilterBindingCreateXML;
#  endif

    ret = virNWFilterBindingCreateXMLSymbol(conn,
                                            xml,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNWFilterBindingDeleteType)(virNWFilterBindingPtr binding);

int
virNWFilterBindingDeleteWrapper(virNWFilterBindingPtr binding,
                                virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingDelete compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingDeleteType virNWFilterBindingDeleteSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingDeleteSymbol = libvirtSymbol(libvirt,
                                                           "virNWFilterBindingDelete",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingDelete");
        return ret;
    }
#  else
    virNWFilterBindingDeleteSymbol = &virNWFilterBindingDelete;
#  endif

    ret = virNWFilterBindingDeleteSymbol(binding);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNWFilterBindingFreeType)(virNWFilterBindingPtr binding);

int
virNWFilterBindingFreeWrapper(virNWFilterBindingPtr binding,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingFree compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingFreeType virNWFilterBindingFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingFreeSymbol = libvirtSymbol(libvirt,
                                                         "virNWFilterBindingFree",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingFree");
        return ret;
    }
#  else
    virNWFilterBindingFreeSymbol = &virNWFilterBindingFree;
#  endif

    ret = virNWFilterBindingFreeSymbol(binding);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virNWFilterBindingGetFilterNameType)(virNWFilterBindingPtr binding);

const char *
virNWFilterBindingGetFilterNameWrapper(virNWFilterBindingPtr binding,
                                       virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingGetFilterName compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingGetFilterNameType virNWFilterBindingGetFilterNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingGetFilterNameSymbol = libvirtSymbol(libvirt,
                                                                  "virNWFilterBindingGetFilterName",
                                                                  &success,
                                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingGetFilterName");
        return ret;
    }
#  else
    virNWFilterBindingGetFilterNameSymbol = &virNWFilterBindingGetFilterName;
#  endif

    ret = virNWFilterBindingGetFilterNameSymbol(binding);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virNWFilterBindingGetPortDevType)(virNWFilterBindingPtr binding);

const char *
virNWFilterBindingGetPortDevWrapper(virNWFilterBindingPtr binding,
                                    virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingGetPortDev compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingGetPortDevType virNWFilterBindingGetPortDevSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingGetPortDevSymbol = libvirtSymbol(libvirt,
                                                               "virNWFilterBindingGetPortDev",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingGetPortDev");
        return ret;
    }
#  else
    virNWFilterBindingGetPortDevSymbol = &virNWFilterBindingGetPortDev;
#  endif

    ret = virNWFilterBindingGetPortDevSymbol(binding);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingGetXMLDesc compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingGetXMLDescType virNWFilterBindingGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                               "virNWFilterBindingGetXMLDesc",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingGetXMLDesc");
        return ret;
    }
#  else
    virNWFilterBindingGetXMLDescSymbol = &virNWFilterBindingGetXMLDesc;
#  endif

    ret = virNWFilterBindingGetXMLDescSymbol(binding,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingLookupByPortDev compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingLookupByPortDevType virNWFilterBindingLookupByPortDevSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingLookupByPortDevSymbol = libvirtSymbol(libvirt,
                                                                    "virNWFilterBindingLookupByPortDev",
                                                                    &success,
                                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingLookupByPortDev");
        return ret;
    }
#  else
    virNWFilterBindingLookupByPortDevSymbol = &virNWFilterBindingLookupByPortDev;
#  endif

    ret = virNWFilterBindingLookupByPortDevSymbol(conn,
                                                  portdev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNWFilterBindingRefType)(virNWFilterBindingPtr binding);

int
virNWFilterBindingRefWrapper(virNWFilterBindingPtr binding,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNWFilterBindingRef compiled out (from 4.5.0)");
    return ret;
#else
    static virNWFilterBindingRefType virNWFilterBindingRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterBindingRefSymbol = libvirtSymbol(libvirt,
                                                        "virNWFilterBindingRef",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterBindingRef");
        return ret;
    }
#  else
    virNWFilterBindingRefSymbol = &virNWFilterBindingRef;
#  endif

    ret = virNWFilterBindingRefSymbol(binding);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterDefineXML compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterDefineXMLType virNWFilterDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterDefineXMLSymbol = libvirtSymbol(libvirt,
                                                       "virNWFilterDefineXML",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterDefineXML");
        return ret;
    }
#  else
    virNWFilterDefineXMLSymbol = &virNWFilterDefineXML;
#  endif

    ret = virNWFilterDefineXMLSymbol(conn,
                                     xmlDesc);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 7, 0)
    setVirError(err, "Function virNWFilterDefineXMLFlags compiled out (from 7.7.0)");
    return ret;
#else
    static virNWFilterDefineXMLFlagsType virNWFilterDefineXMLFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterDefineXMLFlagsSymbol = libvirtSymbol(libvirt,
                                                            "virNWFilterDefineXMLFlags",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterDefineXMLFlags");
        return ret;
    }
#  else
    virNWFilterDefineXMLFlagsSymbol = &virNWFilterDefineXMLFlags;
#  endif

    ret = virNWFilterDefineXMLFlagsSymbol(conn,
                                          xmlDesc,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNWFilterFreeType)(virNWFilterPtr nwfilter);

int
virNWFilterFreeWrapper(virNWFilterPtr nwfilter,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterFree compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterFreeType virNWFilterFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterFreeSymbol = libvirtSymbol(libvirt,
                                                  "virNWFilterFree",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterFree");
        return ret;
    }
#  else
    virNWFilterFreeSymbol = &virNWFilterFree;
#  endif

    ret = virNWFilterFreeSymbol(nwfilter);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virNWFilterGetNameType)(virNWFilterPtr nwfilter);

const char *
virNWFilterGetNameWrapper(virNWFilterPtr nwfilter,
                          virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetName compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterGetNameType virNWFilterGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterGetNameSymbol = libvirtSymbol(libvirt,
                                                     "virNWFilterGetName",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterGetName");
        return ret;
    }
#  else
    virNWFilterGetNameSymbol = &virNWFilterGetName;
#  endif

    ret = virNWFilterGetNameSymbol(nwfilter);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetUUID compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterGetUUIDType virNWFilterGetUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterGetUUIDSymbol = libvirtSymbol(libvirt,
                                                     "virNWFilterGetUUID",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterGetUUID");
        return ret;
    }
#  else
    virNWFilterGetUUIDSymbol = &virNWFilterGetUUID;
#  endif

    ret = virNWFilterGetUUIDSymbol(nwfilter,
                                   uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetUUIDString compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterGetUUIDStringType virNWFilterGetUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterGetUUIDStringSymbol = libvirtSymbol(libvirt,
                                                           "virNWFilterGetUUIDString",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterGetUUIDString");
        return ret;
    }
#  else
    virNWFilterGetUUIDStringSymbol = &virNWFilterGetUUIDString;
#  endif

    ret = virNWFilterGetUUIDStringSymbol(nwfilter,
                                         buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterGetXMLDesc compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterGetXMLDescType virNWFilterGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                        "virNWFilterGetXMLDesc",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterGetXMLDesc");
        return ret;
    }
#  else
    virNWFilterGetXMLDescSymbol = &virNWFilterGetXMLDesc;
#  endif

    ret = virNWFilterGetXMLDescSymbol(nwfilter,
                                      flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterLookupByName compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterLookupByNameType virNWFilterLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterLookupByNameSymbol = libvirtSymbol(libvirt,
                                                          "virNWFilterLookupByName",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterLookupByName");
        return ret;
    }
#  else
    virNWFilterLookupByNameSymbol = &virNWFilterLookupByName;
#  endif

    ret = virNWFilterLookupByNameSymbol(conn,
                                        name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterLookupByUUID compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterLookupByUUIDType virNWFilterLookupByUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterLookupByUUIDSymbol = libvirtSymbol(libvirt,
                                                          "virNWFilterLookupByUUID",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterLookupByUUID");
        return ret;
    }
#  else
    virNWFilterLookupByUUIDSymbol = &virNWFilterLookupByUUID;
#  endif

    ret = virNWFilterLookupByUUIDSymbol(conn,
                                        uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterLookupByUUIDString compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterLookupByUUIDStringType virNWFilterLookupByUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterLookupByUUIDStringSymbol = libvirtSymbol(libvirt,
                                                                "virNWFilterLookupByUUIDString",
                                                                &success,
                                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterLookupByUUIDString");
        return ret;
    }
#  else
    virNWFilterLookupByUUIDStringSymbol = &virNWFilterLookupByUUIDString;
#  endif

    ret = virNWFilterLookupByUUIDStringSymbol(conn,
                                              uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNWFilterRefType)(virNWFilterPtr nwfilter);

int
virNWFilterRefWrapper(virNWFilterPtr nwfilter,
                      virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterRef compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterRefType virNWFilterRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterRefSymbol = libvirtSymbol(libvirt,
                                                 "virNWFilterRef",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterRef");
        return ret;
    }
#  else
    virNWFilterRefSymbol = &virNWFilterRef;
#  endif

    ret = virNWFilterRefSymbol(nwfilter);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNWFilterUndefineType)(virNWFilterPtr nwfilter);

int
virNWFilterUndefineWrapper(virNWFilterPtr nwfilter,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virNWFilterUndefine compiled out (from 0.8.0)");
    return ret;
#else
    static virNWFilterUndefineType virNWFilterUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNWFilterUndefineSymbol = libvirtSymbol(libvirt,
                                                      "virNWFilterUndefine",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNWFilterUndefine");
        return ret;
    }
#  else
    virNWFilterUndefineSymbol = &virNWFilterUndefine;
#  endif

    ret = virNWFilterUndefineSymbol(nwfilter);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkCreateType)(virNetworkPtr network);

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkCreate compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkCreateType virNetworkCreateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkCreateSymbol = libvirtSymbol(libvirt,
                                                   "virNetworkCreate",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkCreate");
        return ret;
    }
#  else
    virNetworkCreateSymbol = &virNetworkCreate;
#  endif

    ret = virNetworkCreateSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkCreateXML compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkCreateXMLType virNetworkCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkCreateXMLSymbol = libvirtSymbol(libvirt,
                                                      "virNetworkCreateXML",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkCreateXML");
        return ret;
    }
#  else
    virNetworkCreateXMLSymbol = &virNetworkCreateXML;
#  endif

    ret = virNetworkCreateXMLSymbol(conn,
                                    xmlDesc);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNetworkCreateXMLFlags compiled out (from 7.8.0)");
    return ret;
#else
    static virNetworkCreateXMLFlagsType virNetworkCreateXMLFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkCreateXMLFlagsSymbol = libvirtSymbol(libvirt,
                                                           "virNetworkCreateXMLFlags",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkCreateXMLFlags");
        return ret;
    }
#  else
    virNetworkCreateXMLFlagsSymbol = &virNetworkCreateXMLFlags;
#  endif

    ret = virNetworkCreateXMLFlagsSymbol(conn,
                                         xmlDesc,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virNetworkDHCPLeaseFreeType)(virNetworkDHCPLeasePtr lease);

void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 6)
    setVirError(NULL, "Function virNetworkDHCPLeaseFree compiled out (from 1.2.6)");
    return;
#else
    static virNetworkDHCPLeaseFreeType virNetworkDHCPLeaseFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virNetworkDHCPLeaseFreeSymbol = libvirtSymbol(libvirt,
                                                          "virNetworkDHCPLeaseFree",
                                                          &success,
                                                          NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virNetworkDHCPLeaseFree");
        return;
    }
#  else
    virNetworkDHCPLeaseFreeSymbol = &virNetworkDHCPLeaseFree;
#  endif

    virNetworkDHCPLeaseFreeSymbol(lease);
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkDefineXML compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkDefineXMLType virNetworkDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkDefineXMLSymbol = libvirtSymbol(libvirt,
                                                      "virNetworkDefineXML",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkDefineXML");
        return ret;
    }
#  else
    virNetworkDefineXMLSymbol = &virNetworkDefineXML;
#  endif

    ret = virNetworkDefineXMLSymbol(conn,
                                    xml);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 7, 0)
    setVirError(err, "Function virNetworkDefineXMLFlags compiled out (from 7.7.0)");
    return ret;
#else
    static virNetworkDefineXMLFlagsType virNetworkDefineXMLFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkDefineXMLFlagsSymbol = libvirtSymbol(libvirt,
                                                           "virNetworkDefineXMLFlags",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkDefineXMLFlags");
        return ret;
    }
#  else
    virNetworkDefineXMLFlagsSymbol = &virNetworkDefineXMLFlags;
#  endif

    ret = virNetworkDefineXMLFlagsSymbol(conn,
                                         xml,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkDestroyType)(virNetworkPtr network);

int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkDestroy compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkDestroyType virNetworkDestroySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkDestroySymbol = libvirtSymbol(libvirt,
                                                    "virNetworkDestroy",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkDestroy");
        return ret;
    }
#  else
    virNetworkDestroySymbol = &virNetworkDestroy;
#  endif

    ret = virNetworkDestroySymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkFreeType)(virNetworkPtr network);

int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkFree compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkFreeType virNetworkFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkFreeSymbol = libvirtSymbol(libvirt,
                                                 "virNetworkFree",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkFree");
        return ret;
    }
#  else
    virNetworkFreeSymbol = &virNetworkFree;
#  endif

    ret = virNetworkFreeSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virNetworkGetAutostart compiled out (from 0.2.1)");
    return ret;
#else
    static virNetworkGetAutostartType virNetworkGetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetAutostartSymbol = libvirtSymbol(libvirt,
                                                         "virNetworkGetAutostart",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetAutostart");
        return ret;
    }
#  else
    virNetworkGetAutostartSymbol = &virNetworkGetAutostart;
#  endif

    ret = virNetworkGetAutostartSymbol(network,
                                       autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virNetworkGetBridgeNameType)(virNetworkPtr network);

char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetBridgeName compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkGetBridgeNameType virNetworkGetBridgeNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetBridgeNameSymbol = libvirtSymbol(libvirt,
                                                          "virNetworkGetBridgeName",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetBridgeName");
        return ret;
    }
#  else
    virNetworkGetBridgeNameSymbol = &virNetworkGetBridgeName;
#  endif

    ret = virNetworkGetBridgeNameSymbol(network);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virNetworkGetConnectType)(virNetworkPtr net);

virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virNetworkGetConnect compiled out (from 0.3.0)");
    return ret;
#else
    static virNetworkGetConnectType virNetworkGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetConnectSymbol = libvirtSymbol(libvirt,
                                                       "virNetworkGetConnect",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetConnect");
        return ret;
    }
#  else
    virNetworkGetConnectSymbol = &virNetworkGetConnect;
#  endif

    ret = virNetworkGetConnectSymbol(net);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 6)
    setVirError(err, "Function virNetworkGetDHCPLeases compiled out (from 1.2.6)");
    return ret;
#else
    static virNetworkGetDHCPLeasesType virNetworkGetDHCPLeasesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetDHCPLeasesSymbol = libvirtSymbol(libvirt,
                                                          "virNetworkGetDHCPLeases",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetDHCPLeases");
        return ret;
    }
#  else
    virNetworkGetDHCPLeasesSymbol = &virNetworkGetDHCPLeases;
#  endif

    ret = virNetworkGetDHCPLeasesSymbol(network,
                                        mac,
                                        leases,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virNetworkGetNameType)(virNetworkPtr network);

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetName compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkGetNameType virNetworkGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetNameSymbol = libvirtSymbol(libvirt,
                                                    "virNetworkGetName",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetName");
        return ret;
    }
#  else
    virNetworkGetNameSymbol = &virNetworkGetName;
#  endif

    ret = virNetworkGetNameSymbol(network);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetUUID compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkGetUUIDType virNetworkGetUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetUUIDSymbol = libvirtSymbol(libvirt,
                                                    "virNetworkGetUUID",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetUUID");
        return ret;
    }
#  else
    virNetworkGetUUIDSymbol = &virNetworkGetUUID;
#  endif

    ret = virNetworkGetUUIDSymbol(network,
                                  uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetUUIDString compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkGetUUIDStringType virNetworkGetUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetUUIDStringSymbol = libvirtSymbol(libvirt,
                                                          "virNetworkGetUUIDString",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetUUIDString");
        return ret;
    }
#  else
    virNetworkGetUUIDStringSymbol = &virNetworkGetUUIDString;
#  endif

    ret = virNetworkGetUUIDStringSymbol(network,
                                        buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkGetXMLDesc compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkGetXMLDescType virNetworkGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                       "virNetworkGetXMLDesc",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkGetXMLDesc");
        return ret;
    }
#  else
    virNetworkGetXMLDescSymbol = &virNetworkGetXMLDesc;
#  endif

    ret = virNetworkGetXMLDescSymbol(network,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkIsActiveType)(virNetworkPtr net);

int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virNetworkIsActive compiled out (from 0.7.3)");
    return ret;
#else
    static virNetworkIsActiveType virNetworkIsActiveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkIsActiveSymbol = libvirtSymbol(libvirt,
                                                     "virNetworkIsActive",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkIsActive");
        return ret;
    }
#  else
    virNetworkIsActiveSymbol = &virNetworkIsActive;
#  endif

    ret = virNetworkIsActiveSymbol(net);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkIsPersistentType)(virNetworkPtr net);

int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virNetworkIsPersistent compiled out (from 0.7.3)");
    return ret;
#else
    static virNetworkIsPersistentType virNetworkIsPersistentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkIsPersistentSymbol = libvirtSymbol(libvirt,
                                                         "virNetworkIsPersistent",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkIsPersistent");
        return ret;
    }
#  else
    virNetworkIsPersistentSymbol = &virNetworkIsPersistent;
#  endif

    ret = virNetworkIsPersistentSymbol(net);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkListAllPorts compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkListAllPortsType virNetworkListAllPortsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkListAllPortsSymbol = libvirtSymbol(libvirt,
                                                         "virNetworkListAllPorts",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkListAllPorts");
        return ret;
    }
#  else
    virNetworkListAllPortsSymbol = &virNetworkListAllPorts;
#  endif

    ret = virNetworkListAllPortsSymbol(network,
                                       ports,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkLookupByName compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkLookupByNameType virNetworkLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkLookupByNameSymbol = libvirtSymbol(libvirt,
                                                         "virNetworkLookupByName",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkLookupByName");
        return ret;
    }
#  else
    virNetworkLookupByNameSymbol = &virNetworkLookupByName;
#  endif

    ret = virNetworkLookupByNameSymbol(conn,
                                       name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkLookupByUUID compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkLookupByUUIDType virNetworkLookupByUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkLookupByUUIDSymbol = libvirtSymbol(libvirt,
                                                         "virNetworkLookupByUUID",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkLookupByUUID");
        return ret;
    }
#  else
    virNetworkLookupByUUIDSymbol = &virNetworkLookupByUUID;
#  endif

    ret = virNetworkLookupByUUIDSymbol(conn,
                                       uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkLookupByUUIDString compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkLookupByUUIDStringType virNetworkLookupByUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkLookupByUUIDStringSymbol = libvirtSymbol(libvirt,
                                                               "virNetworkLookupByUUIDString",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkLookupByUUIDString");
        return ret;
    }
#  else
    virNetworkLookupByUUIDStringSymbol = &virNetworkLookupByUUIDString;
#  endif

    ret = virNetworkLookupByUUIDStringSymbol(conn,
                                             uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortCreateXML compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortCreateXMLType virNetworkPortCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortCreateXMLSymbol = libvirtSymbol(libvirt,
                                                          "virNetworkPortCreateXML",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortCreateXML");
        return ret;
    }
#  else
    virNetworkPortCreateXMLSymbol = &virNetworkPortCreateXML;
#  endif

    ret = virNetworkPortCreateXMLSymbol(net,
                                        xmldesc,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortDelete compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortDeleteType virNetworkPortDeleteSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortDeleteSymbol = libvirtSymbol(libvirt,
                                                       "virNetworkPortDelete",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortDelete");
        return ret;
    }
#  else
    virNetworkPortDeleteSymbol = &virNetworkPortDelete;
#  endif

    ret = virNetworkPortDeleteSymbol(port,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkPortFreeType)(virNetworkPortPtr port);

int
virNetworkPortFreeWrapper(virNetworkPortPtr port,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortFree compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortFreeType virNetworkPortFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortFreeSymbol = libvirtSymbol(libvirt,
                                                     "virNetworkPortFree",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortFree");
        return ret;
    }
#  else
    virNetworkPortFreeSymbol = &virNetworkPortFree;
#  endif

    ret = virNetworkPortFreeSymbol(port);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virNetworkPtr
(*virNetworkPortGetNetworkType)(virNetworkPortPtr port);

virNetworkPtr
virNetworkPortGetNetworkWrapper(virNetworkPortPtr port,
                                virErrorPtr err)
{
    virNetworkPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetNetwork compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortGetNetworkType virNetworkPortGetNetworkSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortGetNetworkSymbol = libvirtSymbol(libvirt,
                                                           "virNetworkPortGetNetwork",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortGetNetwork");
        return ret;
    }
#  else
    virNetworkPortGetNetworkSymbol = &virNetworkPortGetNetwork;
#  endif

    ret = virNetworkPortGetNetworkSymbol(port);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetParameters compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortGetParametersType virNetworkPortGetParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortGetParametersSymbol = libvirtSymbol(libvirt,
                                                              "virNetworkPortGetParameters",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortGetParameters");
        return ret;
    }
#  else
    virNetworkPortGetParametersSymbol = &virNetworkPortGetParameters;
#  endif

    ret = virNetworkPortGetParametersSymbol(port,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetUUID compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortGetUUIDType virNetworkPortGetUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortGetUUIDSymbol = libvirtSymbol(libvirt,
                                                        "virNetworkPortGetUUID",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortGetUUID");
        return ret;
    }
#  else
    virNetworkPortGetUUIDSymbol = &virNetworkPortGetUUID;
#  endif

    ret = virNetworkPortGetUUIDSymbol(port,
                                      uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetUUIDString compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortGetUUIDStringType virNetworkPortGetUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortGetUUIDStringSymbol = libvirtSymbol(libvirt,
                                                              "virNetworkPortGetUUIDString",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortGetUUIDString");
        return ret;
    }
#  else
    virNetworkPortGetUUIDStringSymbol = &virNetworkPortGetUUIDString;
#  endif

    ret = virNetworkPortGetUUIDStringSymbol(port,
                                            buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortGetXMLDesc compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortGetXMLDescType virNetworkPortGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                           "virNetworkPortGetXMLDesc",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortGetXMLDesc");
        return ret;
    }
#  else
    virNetworkPortGetXMLDescSymbol = &virNetworkPortGetXMLDesc;
#  endif

    ret = virNetworkPortGetXMLDescSymbol(port,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortLookupByUUID compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortLookupByUUIDType virNetworkPortLookupByUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortLookupByUUIDSymbol = libvirtSymbol(libvirt,
                                                             "virNetworkPortLookupByUUID",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortLookupByUUID");
        return ret;
    }
#  else
    virNetworkPortLookupByUUIDSymbol = &virNetworkPortLookupByUUID;
#  endif

    ret = virNetworkPortLookupByUUIDSymbol(net,
                                           uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortLookupByUUIDString compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortLookupByUUIDStringType virNetworkPortLookupByUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortLookupByUUIDStringSymbol = libvirtSymbol(libvirt,
                                                                   "virNetworkPortLookupByUUIDString",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortLookupByUUIDString");
        return ret;
    }
#  else
    virNetworkPortLookupByUUIDStringSymbol = &virNetworkPortLookupByUUIDString;
#  endif

    ret = virNetworkPortLookupByUUIDStringSymbol(net,
                                                 uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkPortRefType)(virNetworkPortPtr port);

int
virNetworkPortRefWrapper(virNetworkPortPtr port,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortRef compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortRefType virNetworkPortRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortRefSymbol = libvirtSymbol(libvirt,
                                                    "virNetworkPortRef",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortRef");
        return ret;
    }
#  else
    virNetworkPortRefSymbol = &virNetworkPortRef;
#  endif

    ret = virNetworkPortRefSymbol(port);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(5, 5, 0)
    setVirError(err, "Function virNetworkPortSetParameters compiled out (from 5.5.0)");
    return ret;
#else
    static virNetworkPortSetParametersType virNetworkPortSetParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkPortSetParametersSymbol = libvirtSymbol(libvirt,
                                                              "virNetworkPortSetParameters",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkPortSetParameters");
        return ret;
    }
#  else
    virNetworkPortSetParametersSymbol = &virNetworkPortSetParameters;
#  endif

    ret = virNetworkPortSetParametersSymbol(port,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkRefType)(virNetworkPtr network);

int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virNetworkRef compiled out (from 0.6.0)");
    return ret;
#else
    static virNetworkRefType virNetworkRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkRefSymbol = libvirtSymbol(libvirt,
                                                "virNetworkRef",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkRef");
        return ret;
    }
#  else
    virNetworkRefSymbol = &virNetworkRef;
#  endif

    ret = virNetworkRefSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virNetworkSetAutostart compiled out (from 0.2.1)");
    return ret;
#else
    static virNetworkSetAutostartType virNetworkSetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkSetAutostartSymbol = libvirtSymbol(libvirt,
                                                         "virNetworkSetAutostart",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkSetAutostart");
        return ret;
    }
#  else
    virNetworkSetAutostartSymbol = &virNetworkSetAutostart;
#  endif

    ret = virNetworkSetAutostartSymbol(network,
                                       autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNetworkUndefineType)(virNetworkPtr network);

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 2, 0)
    setVirError(err, "Function virNetworkUndefine compiled out (from 0.2.0)");
    return ret;
#else
    static virNetworkUndefineType virNetworkUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkUndefineSymbol = libvirtSymbol(libvirt,
                                                     "virNetworkUndefine",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkUndefine");
        return ret;
    }
#  else
    virNetworkUndefineSymbol = &virNetworkUndefine;
#  endif

    ret = virNetworkUndefineSymbol(network);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virNetworkUpdate compiled out (from 0.10.2)");
    return ret;
#else
    static virNetworkUpdateType virNetworkUpdateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNetworkUpdateSymbol = libvirtSymbol(libvirt,
                                                   "virNetworkUpdate",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNetworkUpdate");
        return ret;
    }
#  else
    virNetworkUpdateSymbol = &virNetworkUpdate;
#  endif

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
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 9)
    setVirError(err, "Function virNodeAllocPages compiled out (from 1.2.9)");
    return ret;
#else
    static virNodeAllocPagesType virNodeAllocPagesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeAllocPagesSymbol = libvirtSymbol(libvirt,
                                                    "virNodeAllocPages",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeAllocPages");
        return ret;
    }
#  else
    virNodeAllocPagesSymbol = &virNodeAllocPages;
#  endif

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
#endif
}

typedef int
(*virNodeDeviceCreateType)(virNodeDevicePtr dev,
                           unsigned int flags);

int
virNodeDeviceCreateWrapper(virNodeDevicePtr dev,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 3, 0)
    setVirError(err, "Function virNodeDeviceCreate compiled out (from 7.3.0)");
    return ret;
#else
    static virNodeDeviceCreateType virNodeDeviceCreateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceCreateSymbol = libvirtSymbol(libvirt,
                                                      "virNodeDeviceCreate",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceCreate");
        return ret;
    }
#  else
    virNodeDeviceCreateSymbol = &virNodeDeviceCreate;
#  endif

    ret = virNodeDeviceCreateSymbol(dev,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virNodeDevicePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 3)
    setVirError(err, "Function virNodeDeviceCreateXML compiled out (from 0.6.3)");
    return ret;
#else
    static virNodeDeviceCreateXMLType virNodeDeviceCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceCreateXMLSymbol = libvirtSymbol(libvirt,
                                                         "virNodeDeviceCreateXML",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceCreateXML");
        return ret;
    }
#  else
    virNodeDeviceCreateXMLSymbol = &virNodeDeviceCreateXML;
#  endif

    ret = virNodeDeviceCreateXMLSymbol(conn,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virNodeDevicePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 3, 0)
    setVirError(err, "Function virNodeDeviceDefineXML compiled out (from 7.3.0)");
    return ret;
#else
    static virNodeDeviceDefineXMLType virNodeDeviceDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceDefineXMLSymbol = libvirtSymbol(libvirt,
                                                         "virNodeDeviceDefineXML",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceDefineXML");
        return ret;
    }
#  else
    virNodeDeviceDefineXMLSymbol = &virNodeDeviceDefineXML;
#  endif

    ret = virNodeDeviceDefineXMLSymbol(conn,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceDestroyType)(virNodeDevicePtr dev);

int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 3)
    setVirError(err, "Function virNodeDeviceDestroy compiled out (from 0.6.3)");
    return ret;
#else
    static virNodeDeviceDestroyType virNodeDeviceDestroySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceDestroySymbol = libvirtSymbol(libvirt,
                                                       "virNodeDeviceDestroy",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceDestroy");
        return ret;
    }
#  else
    virNodeDeviceDestroySymbol = &virNodeDeviceDestroy;
#  endif

    ret = virNodeDeviceDestroySymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 5)
    setVirError(err, "Function virNodeDeviceDetachFlags compiled out (from 1.0.5)");
    return ret;
#else
    static virNodeDeviceDetachFlagsType virNodeDeviceDetachFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceDetachFlagsSymbol = libvirtSymbol(libvirt,
                                                           "virNodeDeviceDetachFlags",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceDetachFlags");
        return ret;
    }
#  else
    virNodeDeviceDetachFlagsSymbol = &virNodeDeviceDetachFlags;
#  endif

    ret = virNodeDeviceDetachFlagsSymbol(dev,
                                         driverName,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceDettachType)(virNodeDevicePtr dev);

int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeDeviceDettach compiled out (from 0.6.1)");
    return ret;
#else
    static virNodeDeviceDettachType virNodeDeviceDettachSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceDettachSymbol = libvirtSymbol(libvirt,
                                                       "virNodeDeviceDettach",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceDettach");
        return ret;
    }
#  else
    virNodeDeviceDettachSymbol = &virNodeDeviceDettach;
#  endif

    ret = virNodeDeviceDettachSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceFreeType)(virNodeDevicePtr dev);

int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceFree compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceFreeType virNodeDeviceFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceFreeSymbol = libvirtSymbol(libvirt,
                                                    "virNodeDeviceFree",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceFree");
        return ret;
    }
#  else
    virNodeDeviceFreeSymbol = &virNodeDeviceFree;
#  endif

    ret = virNodeDeviceFreeSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceGetAutostartType)(virNodeDevicePtr dev,
                                 int * autostart);

int
virNodeDeviceGetAutostartWrapper(virNodeDevicePtr dev,
                                 int * autostart,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceGetAutostart compiled out (from 7.8.0)");
    return ret;
#else
    static virNodeDeviceGetAutostartType virNodeDeviceGetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceGetAutostartSymbol = libvirtSymbol(libvirt,
                                                            "virNodeDeviceGetAutostart",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceGetAutostart");
        return ret;
    }
#  else
    virNodeDeviceGetAutostartSymbol = &virNodeDeviceGetAutostart;
#  endif

    ret = virNodeDeviceGetAutostartSymbol(dev,
                                          autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virNodeDeviceGetNameType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceGetName compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceGetNameType virNodeDeviceGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceGetNameSymbol = libvirtSymbol(libvirt,
                                                       "virNodeDeviceGetName",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceGetName");
        return ret;
    }
#  else
    virNodeDeviceGetNameSymbol = &virNodeDeviceGetName;
#  endif

    ret = virNodeDeviceGetNameSymbol(dev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virNodeDeviceGetParentType)(virNodeDevicePtr dev);

const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceGetParent compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceGetParentType virNodeDeviceGetParentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceGetParentSymbol = libvirtSymbol(libvirt,
                                                         "virNodeDeviceGetParent",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceGetParent");
        return ret;
    }
#  else
    virNodeDeviceGetParentSymbol = &virNodeDeviceGetParent;
#  endif

    ret = virNodeDeviceGetParentSymbol(dev);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virNodeDeviceGetXMLDescType)(virNodeDevicePtr dev,
                               unsigned int flags);

char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceGetXMLDesc compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceGetXMLDescType virNodeDeviceGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                          "virNodeDeviceGetXMLDesc",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceGetXMLDesc");
        return ret;
    }
#  else
    virNodeDeviceGetXMLDescSymbol = &virNodeDeviceGetXMLDesc;
#  endif

    ret = virNodeDeviceGetXMLDescSymbol(dev,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceIsActiveType)(virNodeDevicePtr dev);

int
virNodeDeviceIsActiveWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceIsActive compiled out (from 7.8.0)");
    return ret;
#else
    static virNodeDeviceIsActiveType virNodeDeviceIsActiveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceIsActiveSymbol = libvirtSymbol(libvirt,
                                                        "virNodeDeviceIsActive",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceIsActive");
        return ret;
    }
#  else
    virNodeDeviceIsActiveSymbol = &virNodeDeviceIsActive;
#  endif

    ret = virNodeDeviceIsActiveSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceIsPersistentType)(virNodeDevicePtr dev);

int
virNodeDeviceIsPersistentWrapper(virNodeDevicePtr dev,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceIsPersistent compiled out (from 7.8.0)");
    return ret;
#else
    static virNodeDeviceIsPersistentType virNodeDeviceIsPersistentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceIsPersistentSymbol = libvirtSymbol(libvirt,
                                                            "virNodeDeviceIsPersistent",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceIsPersistent");
        return ret;
    }
#  else
    virNodeDeviceIsPersistentSymbol = &virNodeDeviceIsPersistent;
#  endif

    ret = virNodeDeviceIsPersistentSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceListCaps compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceListCapsType virNodeDeviceListCapsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceListCapsSymbol = libvirtSymbol(libvirt,
                                                        "virNodeDeviceListCaps",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceListCaps");
        return ret;
    }
#  else
    virNodeDeviceListCapsSymbol = &virNodeDeviceListCaps;
#  endif

    ret = virNodeDeviceListCapsSymbol(dev,
                                      names,
                                      maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virNodeDevicePtr
(*virNodeDeviceLookupByNameType)(virConnectPtr conn,
                                 const char * name);

virNodeDevicePtr
virNodeDeviceLookupByNameWrapper(virConnectPtr conn,
                                 const char * name,
                                 virErrorPtr err)
{
    virNodeDevicePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceLookupByName compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceLookupByNameType virNodeDeviceLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceLookupByNameSymbol = libvirtSymbol(libvirt,
                                                            "virNodeDeviceLookupByName",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceLookupByName");
        return ret;
    }
#  else
    virNodeDeviceLookupByNameSymbol = &virNodeDeviceLookupByName;
#  endif

    ret = virNodeDeviceLookupByNameSymbol(conn,
                                          name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virNodeDevicePtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virNodeDeviceLookupSCSIHostByWWN compiled out (from 1.0.3)");
    return ret;
#else
    static virNodeDeviceLookupSCSIHostByWWNType virNodeDeviceLookupSCSIHostByWWNSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceLookupSCSIHostByWWNSymbol = libvirtSymbol(libvirt,
                                                                   "virNodeDeviceLookupSCSIHostByWWN",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceLookupSCSIHostByWWN");
        return ret;
    }
#  else
    virNodeDeviceLookupSCSIHostByWWNSymbol = &virNodeDeviceLookupSCSIHostByWWN;
#  endif

    ret = virNodeDeviceLookupSCSIHostByWWNSymbol(conn,
                                                 wwnn,
                                                 wwpn,
                                                 flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceNumOfCapsType)(virNodeDevicePtr dev);

int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeDeviceNumOfCaps compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeDeviceNumOfCapsType virNodeDeviceNumOfCapsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceNumOfCapsSymbol = libvirtSymbol(libvirt,
                                                         "virNodeDeviceNumOfCaps",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceNumOfCaps");
        return ret;
    }
#  else
    virNodeDeviceNumOfCapsSymbol = &virNodeDeviceNumOfCaps;
#  endif

    ret = virNodeDeviceNumOfCapsSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceReAttachType)(virNodeDevicePtr dev);

int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeDeviceReAttach compiled out (from 0.6.1)");
    return ret;
#else
    static virNodeDeviceReAttachType virNodeDeviceReAttachSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceReAttachSymbol = libvirtSymbol(libvirt,
                                                        "virNodeDeviceReAttach",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceReAttach");
        return ret;
    }
#  else
    virNodeDeviceReAttachSymbol = &virNodeDeviceReAttach;
#  endif

    ret = virNodeDeviceReAttachSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceRefType)(virNodeDevicePtr dev);

int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virNodeDeviceRef compiled out (from 0.6.0)");
    return ret;
#else
    static virNodeDeviceRefType virNodeDeviceRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceRefSymbol = libvirtSymbol(libvirt,
                                                   "virNodeDeviceRef",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceRef");
        return ret;
    }
#  else
    virNodeDeviceRefSymbol = &virNodeDeviceRef;
#  endif

    ret = virNodeDeviceRefSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceResetType)(virNodeDevicePtr dev);

int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeDeviceReset compiled out (from 0.6.1)");
    return ret;
#else
    static virNodeDeviceResetType virNodeDeviceResetSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceResetSymbol = libvirtSymbol(libvirt,
                                                     "virNodeDeviceReset",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceReset");
        return ret;
    }
#  else
    virNodeDeviceResetSymbol = &virNodeDeviceReset;
#  endif

    ret = virNodeDeviceResetSymbol(dev);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceSetAutostartType)(virNodeDevicePtr dev,
                                 int autostart);

int
virNodeDeviceSetAutostartWrapper(virNodeDevicePtr dev,
                                 int autostart,
                                 virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 8, 0)
    setVirError(err, "Function virNodeDeviceSetAutostart compiled out (from 7.8.0)");
    return ret;
#else
    static virNodeDeviceSetAutostartType virNodeDeviceSetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceSetAutostartSymbol = libvirtSymbol(libvirt,
                                                            "virNodeDeviceSetAutostart",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceSetAutostart");
        return ret;
    }
#  else
    virNodeDeviceSetAutostartSymbol = &virNodeDeviceSetAutostart;
#  endif

    ret = virNodeDeviceSetAutostartSymbol(dev,
                                          autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virNodeDeviceUndefineType)(virNodeDevicePtr dev,
                             unsigned int flags);

int
virNodeDeviceUndefineWrapper(virNodeDevicePtr dev,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(7, 3, 0)
    setVirError(err, "Function virNodeDeviceUndefine compiled out (from 7.3.0)");
    return ret;
#else
    static virNodeDeviceUndefineType virNodeDeviceUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeDeviceUndefineSymbol = libvirtSymbol(libvirt,
                                                        "virNodeDeviceUndefine",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeDeviceUndefine");
        return ret;
    }
#  else
    virNodeDeviceUndefineSymbol = &virNodeDeviceUndefine;
#  endif

    ret = virNodeDeviceUndefineSymbol(dev,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 0)
    setVirError(err, "Function virNodeGetCPUMap compiled out (from 1.0.0)");
    return ret;
#else
    static virNodeGetCPUMapType virNodeGetCPUMapSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetCPUMapSymbol = libvirtSymbol(libvirt,
                                                   "virNodeGetCPUMap",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetCPUMap");
        return ret;
    }
#  else
    virNodeGetCPUMapSymbol = &virNodeGetCPUMap;
#  endif

    ret = virNodeGetCPUMapSymbol(conn,
                                 cpumap,
                                 online,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virNodeGetCPUStats compiled out (from 0.9.3)");
    return ret;
#else
    static virNodeGetCPUStatsType virNodeGetCPUStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetCPUStatsSymbol = libvirtSymbol(libvirt,
                                                     "virNodeGetCPUStats",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetCPUStats");
        return ret;
    }
#  else
    virNodeGetCPUStatsSymbol = &virNodeGetCPUStats;
#  endif

    ret = virNodeGetCPUStatsSymbol(conn,
                                   cpuNum,
                                   params,
                                   nparams,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 3)
    setVirError(err, "Function virNodeGetCellsFreeMemory compiled out (from 0.3.3)");
    return ret;
#else
    static virNodeGetCellsFreeMemoryType virNodeGetCellsFreeMemorySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetCellsFreeMemorySymbol = libvirtSymbol(libvirt,
                                                            "virNodeGetCellsFreeMemory",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetCellsFreeMemory");
        return ret;
    }
#  else
    virNodeGetCellsFreeMemorySymbol = &virNodeGetCellsFreeMemory;
#  endif

    ret = virNodeGetCellsFreeMemorySymbol(conn,
                                          freeMems,
                                          startCell,
                                          maxCells);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef unsigned long long
(*virNodeGetFreeMemoryType)(virConnectPtr conn);

unsigned long long
virNodeGetFreeMemoryWrapper(virConnectPtr conn,
                            virErrorPtr err)
{
    unsigned long long ret = 0;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 3, 3)
    setVirError(err, "Function virNodeGetFreeMemory compiled out (from 0.3.3)");
    return ret;
#else
    static virNodeGetFreeMemoryType virNodeGetFreeMemorySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetFreeMemorySymbol = libvirtSymbol(libvirt,
                                                       "virNodeGetFreeMemory",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetFreeMemory");
        return ret;
    }
#  else
    virNodeGetFreeMemorySymbol = &virNodeGetFreeMemory;
#  endif

    ret = virNodeGetFreeMemorySymbol(conn);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 6)
    setVirError(err, "Function virNodeGetFreePages compiled out (from 1.2.6)");
    return ret;
#else
    static virNodeGetFreePagesType virNodeGetFreePagesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetFreePagesSymbol = libvirtSymbol(libvirt,
                                                      "virNodeGetFreePages",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetFreePages");
        return ret;
    }
#  else
    virNodeGetFreePagesSymbol = &virNodeGetFreePages;
#  endif

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
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virNodeGetInfo compiled out (from 0.1.0)");
    return ret;
#else
    static virNodeGetInfoType virNodeGetInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetInfoSymbol = libvirtSymbol(libvirt,
                                                 "virNodeGetInfo",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetInfo");
        return ret;
    }
#  else
    virNodeGetInfoSymbol = &virNodeGetInfo;
#  endif

    ret = virNodeGetInfoSymbol(conn,
                               info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virNodeGetMemoryParameters compiled out (from 0.10.2)");
    return ret;
#else
    static virNodeGetMemoryParametersType virNodeGetMemoryParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetMemoryParametersSymbol = libvirtSymbol(libvirt,
                                                             "virNodeGetMemoryParameters",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetMemoryParameters");
        return ret;
    }
#  else
    virNodeGetMemoryParametersSymbol = &virNodeGetMemoryParameters;
#  endif

    ret = virNodeGetMemoryParametersSymbol(conn,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virNodeGetMemoryStats compiled out (from 0.9.3)");
    return ret;
#else
    static virNodeGetMemoryStatsType virNodeGetMemoryStatsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetMemoryStatsSymbol = libvirtSymbol(libvirt,
                                                        "virNodeGetMemoryStats",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetMemoryStats");
        return ret;
    }
#  else
    virNodeGetMemoryStatsSymbol = &virNodeGetMemoryStats;
#  endif

    ret = virNodeGetMemoryStatsSymbol(conn,
                                      cellNum,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virNodeGetSEVInfo compiled out (from 4.5.0)");
    return ret;
#else
    static virNodeGetSEVInfoType virNodeGetSEVInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetSEVInfoSymbol = libvirtSymbol(libvirt,
                                                    "virNodeGetSEVInfo",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetSEVInfo");
        return ret;
    }
#  else
    virNodeGetSEVInfoSymbol = &virNodeGetSEVInfo;
#  endif

    ret = virNodeGetSEVInfoSymbol(conn,
                                  params,
                                  nparams,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virNodeGetSecurityModel compiled out (from 0.6.1)");
    return ret;
#else
    static virNodeGetSecurityModelType virNodeGetSecurityModelSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeGetSecurityModelSymbol = libvirtSymbol(libvirt,
                                                          "virNodeGetSecurityModel",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeGetSecurityModel");
        return ret;
    }
#  else
    virNodeGetSecurityModelSymbol = &virNodeGetSecurityModel;
#  endif

    ret = virNodeGetSecurityModelSymbol(conn,
                                        secmodel);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeListDevices compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeListDevicesType virNodeListDevicesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeListDevicesSymbol = libvirtSymbol(libvirt,
                                                     "virNodeListDevices",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeListDevices");
        return ret;
    }
#  else
    virNodeListDevicesSymbol = &virNodeListDevices;
#  endif

    ret = virNodeListDevicesSymbol(conn,
                                   cap,
                                   names,
                                   maxnames,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virNodeNumOfDevices compiled out (from 0.5.0)");
    return ret;
#else
    static virNodeNumOfDevicesType virNodeNumOfDevicesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeNumOfDevicesSymbol = libvirtSymbol(libvirt,
                                                      "virNodeNumOfDevices",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeNumOfDevices");
        return ret;
    }
#  else
    virNodeNumOfDevicesSymbol = &virNodeNumOfDevices;
#  endif

    ret = virNodeNumOfDevicesSymbol(conn,
                                    cap,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virNodeSetMemoryParameters compiled out (from 0.10.2)");
    return ret;
#else
    static virNodeSetMemoryParametersType virNodeSetMemoryParametersSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeSetMemoryParametersSymbol = libvirtSymbol(libvirt,
                                                             "virNodeSetMemoryParameters",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeSetMemoryParameters");
        return ret;
    }
#  else
    virNodeSetMemoryParametersSymbol = &virNodeSetMemoryParameters;
#  endif

    ret = virNodeSetMemoryParametersSymbol(conn,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virNodeSuspendForDuration compiled out (from 0.9.8)");
    return ret;
#else
    static virNodeSuspendForDurationType virNodeSuspendForDurationSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virNodeSuspendForDurationSymbol = libvirtSymbol(libvirt,
                                                            "virNodeSuspendForDuration",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virNodeSuspendForDuration");
        return ret;
    }
#  else
    virNodeSuspendForDurationSymbol = &virNodeSuspendForDuration;
#  endif

    ret = virNodeSuspendForDurationSymbol(conn,
                                          target,
                                          duration,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virResetErrorType)(virErrorPtr err);

void
virResetErrorWrapper(virErrorPtr err)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virResetError compiled out (from 0.1.0)");
    return;
#else
    static virResetErrorType virResetErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virResetErrorSymbol = libvirtSymbol(libvirt,
                                                "virResetError",
                                                &success,
                                                NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virResetError");
        return;
    }
#  else
    virResetErrorSymbol = &virResetError;
#  endif

    virResetErrorSymbol(err);
#endif
}

typedef void
(*virResetLastErrorType)(void);

void
virResetLastErrorWrapper(void)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virResetLastError compiled out (from 0.1.0)");
    return;
#else
    static virResetLastErrorType virResetLastErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virResetLastErrorSymbol = libvirtSymbol(libvirt,
                                                    "virResetLastError",
                                                    &success,
                                                    NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virResetLastError");
        return;
    }
#  else
    virResetLastErrorSymbol = &virResetLastError;
#  endif

    virResetLastErrorSymbol();
#endif
}

typedef virErrorPtr
(*virSaveLastErrorType)(void);

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virSaveLastError compiled out (from 0.6.1)");
    return ret;
#else
    static virSaveLastErrorType virSaveLastErrorSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSaveLastErrorSymbol = libvirtSymbol(libvirt,
                                                   "virSaveLastError",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSaveLastError");
        return ret;
    }
#  else
    virSaveLastErrorSymbol = &virSaveLastError;
#  endif

    ret = virSaveLastErrorSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretDefineXML compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretDefineXMLType virSecretDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretDefineXMLSymbol = libvirtSymbol(libvirt,
                                                     "virSecretDefineXML",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretDefineXML");
        return ret;
    }
#  else
    virSecretDefineXMLSymbol = &virSecretDefineXML;
#  endif

    ret = virSecretDefineXMLSymbol(conn,
                                   xml,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virSecretFreeType)(virSecretPtr secret);

int
virSecretFreeWrapper(virSecretPtr secret,
                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretFree compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretFreeType virSecretFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretFreeSymbol = libvirtSymbol(libvirt,
                                                "virSecretFree",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretFree");
        return ret;
    }
#  else
    virSecretFreeSymbol = &virSecretFree;
#  endif

    ret = virSecretFreeSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virSecretGetConnectType)(virSecretPtr secret);

virConnectPtr
virSecretGetConnectWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetConnect compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetConnectType virSecretGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetConnectSymbol = libvirtSymbol(libvirt,
                                                      "virSecretGetConnect",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetConnect");
        return ret;
    }
#  else
    virSecretGetConnectSymbol = &virSecretGetConnect;
#  endif

    ret = virSecretGetConnectSymbol(secret);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUUID compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetUUIDType virSecretGetUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetUUIDSymbol = libvirtSymbol(libvirt,
                                                   "virSecretGetUUID",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetUUID");
        return ret;
    }
#  else
    virSecretGetUUIDSymbol = &virSecretGetUUID;
#  endif

    ret = virSecretGetUUIDSymbol(secret,
                                 uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUUIDString compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetUUIDStringType virSecretGetUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetUUIDStringSymbol = libvirtSymbol(libvirt,
                                                         "virSecretGetUUIDString",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetUUIDString");
        return ret;
    }
#  else
    virSecretGetUUIDStringSymbol = &virSecretGetUUIDString;
#  endif

    ret = virSecretGetUUIDStringSymbol(secret,
                                       buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virSecretGetUsageIDType)(virSecretPtr secret);

const char *
virSecretGetUsageIDWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUsageID compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetUsageIDType virSecretGetUsageIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetUsageIDSymbol = libvirtSymbol(libvirt,
                                                      "virSecretGetUsageID",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetUsageID");
        return ret;
    }
#  else
    virSecretGetUsageIDSymbol = &virSecretGetUsageID;
#  endif

    ret = virSecretGetUsageIDSymbol(secret);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virSecretGetUsageTypeType)(virSecretPtr secret);

int
virSecretGetUsageTypeWrapper(virSecretPtr secret,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetUsageType compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetUsageTypeType virSecretGetUsageTypeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetUsageTypeSymbol = libvirtSymbol(libvirt,
                                                        "virSecretGetUsageType",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetUsageType");
        return ret;
    }
#  else
    virSecretGetUsageTypeSymbol = &virSecretGetUsageType;
#  endif

    ret = virSecretGetUsageTypeSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetValue compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetValueType virSecretGetValueSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetValueSymbol = libvirtSymbol(libvirt,
                                                    "virSecretGetValue",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetValue");
        return ret;
    }
#  else
    virSecretGetValueSymbol = &virSecretGetValue;
#  endif

    ret = virSecretGetValueSymbol(secret,
                                  value_size,
                                  flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretGetXMLDesc compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretGetXMLDescType virSecretGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                      "virSecretGetXMLDesc",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretGetXMLDesc");
        return ret;
    }
#  else
    virSecretGetXMLDescSymbol = &virSecretGetXMLDesc;
#  endif

    ret = virSecretGetXMLDescSymbol(secret,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretLookupByUUID compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretLookupByUUIDType virSecretLookupByUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretLookupByUUIDSymbol = libvirtSymbol(libvirt,
                                                        "virSecretLookupByUUID",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretLookupByUUID");
        return ret;
    }
#  else
    virSecretLookupByUUIDSymbol = &virSecretLookupByUUID;
#  endif

    ret = virSecretLookupByUUIDSymbol(conn,
                                      uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretLookupByUUIDString compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretLookupByUUIDStringType virSecretLookupByUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretLookupByUUIDStringSymbol = libvirtSymbol(libvirt,
                                                              "virSecretLookupByUUIDString",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretLookupByUUIDString");
        return ret;
    }
#  else
    virSecretLookupByUUIDStringSymbol = &virSecretLookupByUUIDString;
#  endif

    ret = virSecretLookupByUUIDStringSymbol(conn,
                                            uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretLookupByUsage compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretLookupByUsageType virSecretLookupByUsageSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretLookupByUsageSymbol = libvirtSymbol(libvirt,
                                                         "virSecretLookupByUsage",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretLookupByUsage");
        return ret;
    }
#  else
    virSecretLookupByUsageSymbol = &virSecretLookupByUsage;
#  endif

    ret = virSecretLookupByUsageSymbol(conn,
                                       usageType,
                                       usageID);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virSecretRefType)(virSecretPtr secret);

int
virSecretRefWrapper(virSecretPtr secret,
                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretRef compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretRefType virSecretRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretRefSymbol = libvirtSymbol(libvirt,
                                               "virSecretRef",
                                               &success,
                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretRef");
        return ret;
    }
#  else
    virSecretRefSymbol = &virSecretRef;
#  endif

    ret = virSecretRefSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretSetValue compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretSetValueType virSecretSetValueSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretSetValueSymbol = libvirtSymbol(libvirt,
                                                    "virSecretSetValue",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretSetValue");
        return ret;
    }
#  else
    virSecretSetValueSymbol = &virSecretSetValue;
#  endif

    ret = virSecretSetValueSymbol(secret,
                                  value,
                                  value_size,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virSecretUndefineType)(virSecretPtr secret);

int
virSecretUndefineWrapper(virSecretPtr secret,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 1)
    setVirError(err, "Function virSecretUndefine compiled out (from 0.7.1)");
    return ret;
#else
    static virSecretUndefineType virSecretUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virSecretUndefineSymbol = libvirtSymbol(libvirt,
                                                    "virSecretUndefine",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virSecretUndefine");
        return ret;
    }
#  else
    virSecretUndefineSymbol = &virSecretUndefine;
#  endif

    ret = virSecretUndefineSymbol(secret);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virSetErrorFuncType)(void * userData,
                       virErrorFunc handler);

void
virSetErrorFuncWrapper(void * userData,
                       virErrorFunc handler)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virSetErrorFunc compiled out (from 0.1.0)");
    return;
#else
    static virSetErrorFuncType virSetErrorFuncSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virSetErrorFuncSymbol = libvirtSymbol(libvirt,
                                                  "virSetErrorFunc",
                                                  &success,
                                                  NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virSetErrorFunc");
        return;
    }
#  else
    virSetErrorFuncSymbol = &virSetErrorFunc;
#  endif

    virSetErrorFuncSymbol(userData,
                          handler);
#endif
}

typedef int
(*virStoragePoolBuildType)(virStoragePoolPtr pool,
                           unsigned int flags);

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolBuild compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolBuildType virStoragePoolBuildSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolBuildSymbol = libvirtSymbol(libvirt,
                                                      "virStoragePoolBuild",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolBuild");
        return ret;
    }
#  else
    virStoragePoolBuildSymbol = &virStoragePoolBuild;
#  endif

    ret = virStoragePoolBuildSymbol(pool,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolCreateType)(virStoragePoolPtr pool,
                            unsigned int flags);

int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolCreate compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolCreateType virStoragePoolCreateSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolCreateSymbol = libvirtSymbol(libvirt,
                                                       "virStoragePoolCreate",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolCreate");
        return ret;
    }
#  else
    virStoragePoolCreateSymbol = &virStoragePoolCreate;
#  endif

    ret = virStoragePoolCreateSymbol(pool,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolCreateXML compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolCreateXMLType virStoragePoolCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolCreateXMLSymbol = libvirtSymbol(libvirt,
                                                          "virStoragePoolCreateXML",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolCreateXML");
        return ret;
    }
#  else
    virStoragePoolCreateXMLSymbol = &virStoragePoolCreateXML;
#  endif

    ret = virStoragePoolCreateXMLSymbol(conn,
                                        xmlDesc,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolDefineXML compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolDefineXMLType virStoragePoolDefineXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolDefineXMLSymbol = libvirtSymbol(libvirt,
                                                          "virStoragePoolDefineXML",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolDefineXML");
        return ret;
    }
#  else
    virStoragePoolDefineXMLSymbol = &virStoragePoolDefineXML;
#  endif

    ret = virStoragePoolDefineXMLSymbol(conn,
                                        xml,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolDeleteType)(virStoragePoolPtr pool,
                            unsigned int flags);

int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolDelete compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolDeleteType virStoragePoolDeleteSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolDeleteSymbol = libvirtSymbol(libvirt,
                                                       "virStoragePoolDelete",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolDelete");
        return ret;
    }
#  else
    virStoragePoolDeleteSymbol = &virStoragePoolDelete;
#  endif

    ret = virStoragePoolDeleteSymbol(pool,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolDestroyType)(virStoragePoolPtr pool);

int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolDestroy compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolDestroyType virStoragePoolDestroySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolDestroySymbol = libvirtSymbol(libvirt,
                                                        "virStoragePoolDestroy",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolDestroy");
        return ret;
    }
#  else
    virStoragePoolDestroySymbol = &virStoragePoolDestroy;
#  endif

    ret = virStoragePoolDestroySymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolFreeType)(virStoragePoolPtr pool);

int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolFree compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolFreeType virStoragePoolFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolFreeSymbol = libvirtSymbol(libvirt,
                                                     "virStoragePoolFree",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolFree");
        return ret;
    }
#  else
    virStoragePoolFreeSymbol = &virStoragePoolFree;
#  endif

    ret = virStoragePoolFreeSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolGetAutostartType)(virStoragePoolPtr pool,
                                  int * autostart);

int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int * autostart,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetAutostart compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetAutostartType virStoragePoolGetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetAutostartSymbol = libvirtSymbol(libvirt,
                                                             "virStoragePoolGetAutostart",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetAutostart");
        return ret;
    }
#  else
    virStoragePoolGetAutostartSymbol = &virStoragePoolGetAutostart;
#  endif

    ret = virStoragePoolGetAutostartSymbol(pool,
                                           autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virStoragePoolGetConnectType)(virStoragePoolPtr pool);

virConnectPtr
virStoragePoolGetConnectWrapper(virStoragePoolPtr pool,
                                virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetConnect compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetConnectType virStoragePoolGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetConnectSymbol = libvirtSymbol(libvirt,
                                                           "virStoragePoolGetConnect",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetConnect");
        return ret;
    }
#  else
    virStoragePoolGetConnectSymbol = &virStoragePoolGetConnect;
#  endif

    ret = virStoragePoolGetConnectSymbol(pool);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolGetInfoType)(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info);

int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetInfo compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetInfoType virStoragePoolGetInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetInfoSymbol = libvirtSymbol(libvirt,
                                                        "virStoragePoolGetInfo",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetInfo");
        return ret;
    }
#  else
    virStoragePoolGetInfoSymbol = &virStoragePoolGetInfo;
#  endif

    ret = virStoragePoolGetInfoSymbol(pool,
                                      info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virStoragePoolGetNameType)(virStoragePoolPtr pool);

const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetName compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetNameType virStoragePoolGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetNameSymbol = libvirtSymbol(libvirt,
                                                        "virStoragePoolGetName",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetName");
        return ret;
    }
#  else
    virStoragePoolGetNameSymbol = &virStoragePoolGetName;
#  endif

    ret = virStoragePoolGetNameSymbol(pool);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolGetUUIDType)(virStoragePoolPtr pool,
                             unsigned char * uuid);

int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char * uuid,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetUUID compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetUUIDType virStoragePoolGetUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetUUIDSymbol = libvirtSymbol(libvirt,
                                                        "virStoragePoolGetUUID",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetUUID");
        return ret;
    }
#  else
    virStoragePoolGetUUIDSymbol = &virStoragePoolGetUUID;
#  endif

    ret = virStoragePoolGetUUIDSymbol(pool,
                                      uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolGetUUIDStringType)(virStoragePoolPtr pool,
                                   char * buf);

int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char * buf,
                                   virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetUUIDString compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetUUIDStringType virStoragePoolGetUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetUUIDStringSymbol = libvirtSymbol(libvirt,
                                                              "virStoragePoolGetUUIDString",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetUUIDString");
        return ret;
    }
#  else
    virStoragePoolGetUUIDStringSymbol = &virStoragePoolGetUUIDString;
#  endif

    ret = virStoragePoolGetUUIDStringSymbol(pool,
                                            buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virStoragePoolGetXMLDescType)(virStoragePoolPtr pool,
                                unsigned int flags);

char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolGetXMLDesc compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolGetXMLDescType virStoragePoolGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                           "virStoragePoolGetXMLDesc",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolGetXMLDesc");
        return ret;
    }
#  else
    virStoragePoolGetXMLDescSymbol = &virStoragePoolGetXMLDesc;
#  endif

    ret = virStoragePoolGetXMLDescSymbol(pool,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolIsActiveType)(virStoragePoolPtr pool);

int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virStoragePoolIsActive compiled out (from 0.7.3)");
    return ret;
#else
    static virStoragePoolIsActiveType virStoragePoolIsActiveSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolIsActiveSymbol = libvirtSymbol(libvirt,
                                                         "virStoragePoolIsActive",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolIsActive");
        return ret;
    }
#  else
    virStoragePoolIsActiveSymbol = &virStoragePoolIsActive;
#  endif

    ret = virStoragePoolIsActiveSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolIsPersistentType)(virStoragePoolPtr pool);

int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virStoragePoolIsPersistent compiled out (from 0.7.3)");
    return ret;
#else
    static virStoragePoolIsPersistentType virStoragePoolIsPersistentSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolIsPersistentSymbol = libvirtSymbol(libvirt,
                                                             "virStoragePoolIsPersistent",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolIsPersistent");
        return ret;
    }
#  else
    virStoragePoolIsPersistentSymbol = &virStoragePoolIsPersistent;
#  endif

    ret = virStoragePoolIsPersistentSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virStoragePoolListAllVolumes compiled out (from 0.10.2)");
    return ret;
#else
    static virStoragePoolListAllVolumesType virStoragePoolListAllVolumesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolListAllVolumesSymbol = libvirtSymbol(libvirt,
                                                               "virStoragePoolListAllVolumes",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolListAllVolumes");
        return ret;
    }
#  else
    virStoragePoolListAllVolumesSymbol = &virStoragePoolListAllVolumes;
#  endif

    ret = virStoragePoolListAllVolumesSymbol(pool,
                                             vols,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolListVolumes compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolListVolumesType virStoragePoolListVolumesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolListVolumesSymbol = libvirtSymbol(libvirt,
                                                            "virStoragePoolListVolumes",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolListVolumes");
        return ret;
    }
#  else
    virStoragePoolListVolumesSymbol = &virStoragePoolListVolumes;
#  endif

    ret = virStoragePoolListVolumesSymbol(pool,
                                          names,
                                          maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByNameType)(virConnectPtr conn,
                                  const char * name);

virStoragePoolPtr
virStoragePoolLookupByNameWrapper(virConnectPtr conn,
                                  const char * name,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByName compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolLookupByNameType virStoragePoolLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolLookupByNameSymbol = libvirtSymbol(libvirt,
                                                             "virStoragePoolLookupByName",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolLookupByName");
        return ret;
    }
#  else
    virStoragePoolLookupByNameSymbol = &virStoragePoolLookupByName;
#  endif

    ret = virStoragePoolLookupByNameSymbol(conn,
                                           name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByTargetPathType)(virConnectPtr conn,
                                        const char * path);

virStoragePoolPtr
virStoragePoolLookupByTargetPathWrapper(virConnectPtr conn,
                                        const char * path,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(4, 1, 0)
    setVirError(err, "Function virStoragePoolLookupByTargetPath compiled out (from 4.1.0)");
    return ret;
#else
    static virStoragePoolLookupByTargetPathType virStoragePoolLookupByTargetPathSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolLookupByTargetPathSymbol = libvirtSymbol(libvirt,
                                                                   "virStoragePoolLookupByTargetPath",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolLookupByTargetPath");
        return ret;
    }
#  else
    virStoragePoolLookupByTargetPathSymbol = &virStoragePoolLookupByTargetPath;
#  endif

    ret = virStoragePoolLookupByTargetPathSymbol(conn,
                                                 path);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByUUIDType)(virConnectPtr conn,
                                  const unsigned char * uuid);

virStoragePoolPtr
virStoragePoolLookupByUUIDWrapper(virConnectPtr conn,
                                  const unsigned char * uuid,
                                  virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByUUID compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolLookupByUUIDType virStoragePoolLookupByUUIDSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolLookupByUUIDSymbol = libvirtSymbol(libvirt,
                                                             "virStoragePoolLookupByUUID",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolLookupByUUID");
        return ret;
    }
#  else
    virStoragePoolLookupByUUIDSymbol = &virStoragePoolLookupByUUID;
#  endif

    ret = virStoragePoolLookupByUUIDSymbol(conn,
                                           uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByUUIDStringType)(virConnectPtr conn,
                                        const char * uuidstr);

virStoragePoolPtr
virStoragePoolLookupByUUIDStringWrapper(virConnectPtr conn,
                                        const char * uuidstr,
                                        virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByUUIDString compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolLookupByUUIDStringType virStoragePoolLookupByUUIDStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolLookupByUUIDStringSymbol = libvirtSymbol(libvirt,
                                                                   "virStoragePoolLookupByUUIDString",
                                                                   &success,
                                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolLookupByUUIDString");
        return ret;
    }
#  else
    virStoragePoolLookupByUUIDStringSymbol = &virStoragePoolLookupByUUIDString;
#  endif

    ret = virStoragePoolLookupByUUIDStringSymbol(conn,
                                                 uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStoragePoolPtr
(*virStoragePoolLookupByVolumeType)(virStorageVolPtr vol);

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err)
{
    virStoragePoolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolLookupByVolume compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolLookupByVolumeType virStoragePoolLookupByVolumeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolLookupByVolumeSymbol = libvirtSymbol(libvirt,
                                                               "virStoragePoolLookupByVolume",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolLookupByVolume");
        return ret;
    }
#  else
    virStoragePoolLookupByVolumeSymbol = &virStoragePoolLookupByVolume;
#  endif

    ret = virStoragePoolLookupByVolumeSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolNumOfVolumesType)(virStoragePoolPtr pool);

int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolNumOfVolumes compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolNumOfVolumesType virStoragePoolNumOfVolumesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolNumOfVolumesSymbol = libvirtSymbol(libvirt,
                                                             "virStoragePoolNumOfVolumes",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolNumOfVolumes");
        return ret;
    }
#  else
    virStoragePoolNumOfVolumesSymbol = &virStoragePoolNumOfVolumes;
#  endif

    ret = virStoragePoolNumOfVolumesSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolRefType)(virStoragePoolPtr pool);

int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virStoragePoolRef compiled out (from 0.6.0)");
    return ret;
#else
    static virStoragePoolRefType virStoragePoolRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolRefSymbol = libvirtSymbol(libvirt,
                                                    "virStoragePoolRef",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolRef");
        return ret;
    }
#  else
    virStoragePoolRefSymbol = &virStoragePoolRef;
#  endif

    ret = virStoragePoolRefSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolRefreshType)(virStoragePoolPtr pool,
                             unsigned int flags);

int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolRefresh compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolRefreshType virStoragePoolRefreshSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolRefreshSymbol = libvirtSymbol(libvirt,
                                                        "virStoragePoolRefresh",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolRefresh");
        return ret;
    }
#  else
    virStoragePoolRefreshSymbol = &virStoragePoolRefresh;
#  endif

    ret = virStoragePoolRefreshSymbol(pool,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolSetAutostartType)(virStoragePoolPtr pool,
                                  int autostart);

int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolSetAutostart compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolSetAutostartType virStoragePoolSetAutostartSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolSetAutostartSymbol = libvirtSymbol(libvirt,
                                                             "virStoragePoolSetAutostart",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolSetAutostart");
        return ret;
    }
#  else
    virStoragePoolSetAutostartSymbol = &virStoragePoolSetAutostart;
#  endif

    ret = virStoragePoolSetAutostartSymbol(pool,
                                           autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStoragePoolUndefineType)(virStoragePoolPtr pool);

int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStoragePoolUndefine compiled out (from 0.4.1)");
    return ret;
#else
    static virStoragePoolUndefineType virStoragePoolUndefineSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStoragePoolUndefineSymbol = libvirtSymbol(libvirt,
                                                         "virStoragePoolUndefine",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStoragePoolUndefine");
        return ret;
    }
#  else
    virStoragePoolUndefineSymbol = &virStoragePoolUndefine;
#  endif

    ret = virStoragePoolUndefineSymbol(pool);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virStorageVolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolCreateXML compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolCreateXMLType virStorageVolCreateXMLSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolCreateXMLSymbol = libvirtSymbol(libvirt,
                                                         "virStorageVolCreateXML",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolCreateXML");
        return ret;
    }
#  else
    virStorageVolCreateXMLSymbol = &virStorageVolCreateXML;
#  endif

    ret = virStorageVolCreateXMLSymbol(pool,
                                       xmlDesc,
                                       flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virStorageVolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virStorageVolCreateXMLFrom compiled out (from 0.6.4)");
    return ret;
#else
    static virStorageVolCreateXMLFromType virStorageVolCreateXMLFromSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolCreateXMLFromSymbol = libvirtSymbol(libvirt,
                                                             "virStorageVolCreateXMLFrom",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolCreateXMLFrom");
        return ret;
    }
#  else
    virStorageVolCreateXMLFromSymbol = &virStorageVolCreateXMLFrom;
#  endif

    ret = virStorageVolCreateXMLFromSymbol(pool,
                                           xmlDesc,
                                           clonevol,
                                           flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStorageVolDeleteType)(virStorageVolPtr vol,
                           unsigned int flags);

int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolDelete compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolDeleteType virStorageVolDeleteSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolDeleteSymbol = libvirtSymbol(libvirt,
                                                      "virStorageVolDelete",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolDelete");
        return ret;
    }
#  else
    virStorageVolDeleteSymbol = &virStorageVolDelete;
#  endif

    ret = virStorageVolDeleteSymbol(vol,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virStorageVolDownload compiled out (from 0.9.0)");
    return ret;
#else
    static virStorageVolDownloadType virStorageVolDownloadSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolDownloadSymbol = libvirtSymbol(libvirt,
                                                        "virStorageVolDownload",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolDownload");
        return ret;
    }
#  else
    virStorageVolDownloadSymbol = &virStorageVolDownload;
#  endif

    ret = virStorageVolDownloadSymbol(vol,
                                      stream,
                                      offset,
                                      length,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStorageVolFreeType)(virStorageVolPtr vol);

int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolFree compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolFreeType virStorageVolFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolFreeSymbol = libvirtSymbol(libvirt,
                                                    "virStorageVolFree",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolFree");
        return ret;
    }
#  else
    virStorageVolFreeSymbol = &virStorageVolFree;
#  endif

    ret = virStorageVolFreeSymbol(vol);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virConnectPtr
(*virStorageVolGetConnectType)(virStorageVolPtr vol);

virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetConnect compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolGetConnectType virStorageVolGetConnectSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetConnectSymbol = libvirtSymbol(libvirt,
                                                          "virStorageVolGetConnect",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetConnect");
        return ret;
    }
#  else
    virStorageVolGetConnectSymbol = &virStorageVolGetConnect;
#  endif

    ret = virStorageVolGetConnectSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStorageVolGetInfoType)(virStorageVolPtr vol,
                            virStorageVolInfoPtr info);

int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetInfo compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolGetInfoType virStorageVolGetInfoSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetInfoSymbol = libvirtSymbol(libvirt,
                                                       "virStorageVolGetInfo",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetInfo");
        return ret;
    }
#  else
    virStorageVolGetInfoSymbol = &virStorageVolGetInfo;
#  endif

    ret = virStorageVolGetInfoSymbol(vol,
                                     info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virStorageVolGetInfoFlags compiled out (from 3.0.0)");
    return ret;
#else
    static virStorageVolGetInfoFlagsType virStorageVolGetInfoFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetInfoFlagsSymbol = libvirtSymbol(libvirt,
                                                            "virStorageVolGetInfoFlags",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetInfoFlags");
        return ret;
    }
#  else
    virStorageVolGetInfoFlagsSymbol = &virStorageVolGetInfoFlags;
#  endif

    ret = virStorageVolGetInfoFlagsSymbol(vol,
                                          info,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virStorageVolGetKeyType)(virStorageVolPtr vol);

const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetKey compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolGetKeyType virStorageVolGetKeySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetKeySymbol = libvirtSymbol(libvirt,
                                                      "virStorageVolGetKey",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetKey");
        return ret;
    }
#  else
    virStorageVolGetKeySymbol = &virStorageVolGetKey;
#  endif

    ret = virStorageVolGetKeySymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef const char *
(*virStorageVolGetNameType)(virStorageVolPtr vol);

const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    const char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetName compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolGetNameType virStorageVolGetNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetNameSymbol = libvirtSymbol(libvirt,
                                                       "virStorageVolGetName",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetName");
        return ret;
    }
#  else
    virStorageVolGetNameSymbol = &virStorageVolGetName;
#  endif

    ret = virStorageVolGetNameSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virStorageVolGetPathType)(virStorageVolPtr vol);

char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetPath compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolGetPathType virStorageVolGetPathSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetPathSymbol = libvirtSymbol(libvirt,
                                                       "virStorageVolGetPath",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetPath");
        return ret;
    }
#  else
    virStorageVolGetPathSymbol = &virStorageVolGetPath;
#  endif

    ret = virStorageVolGetPathSymbol(vol);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef char *
(*virStorageVolGetXMLDescType)(virStorageVolPtr vol,
                               unsigned int flags);

char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolGetXMLDesc compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolGetXMLDescType virStorageVolGetXMLDescSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolGetXMLDescSymbol = libvirtSymbol(libvirt,
                                                          "virStorageVolGetXMLDesc",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolGetXMLDesc");
        return ret;
    }
#  else
    virStorageVolGetXMLDescSymbol = &virStorageVolGetXMLDesc;
#  endif

    ret = virStorageVolGetXMLDescSymbol(vol,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStorageVolPtr
(*virStorageVolLookupByKeyType)(virConnectPtr conn,
                                const char * key);

virStorageVolPtr
virStorageVolLookupByKeyWrapper(virConnectPtr conn,
                                const char * key,
                                virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolLookupByKey compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolLookupByKeyType virStorageVolLookupByKeySymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolLookupByKeySymbol = libvirtSymbol(libvirt,
                                                           "virStorageVolLookupByKey",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolLookupByKey");
        return ret;
    }
#  else
    virStorageVolLookupByKeySymbol = &virStorageVolLookupByKey;
#  endif

    ret = virStorageVolLookupByKeySymbol(conn,
                                         key);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStorageVolPtr
(*virStorageVolLookupByNameType)(virStoragePoolPtr pool,
                                 const char * name);

virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char * name,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolLookupByName compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolLookupByNameType virStorageVolLookupByNameSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolLookupByNameSymbol = libvirtSymbol(libvirt,
                                                            "virStorageVolLookupByName",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolLookupByName");
        return ret;
    }
#  else
    virStorageVolLookupByNameSymbol = &virStorageVolLookupByName;
#  endif

    ret = virStorageVolLookupByNameSymbol(pool,
                                          name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStorageVolPtr
(*virStorageVolLookupByPathType)(virConnectPtr conn,
                                 const char * path);

virStorageVolPtr
virStorageVolLookupByPathWrapper(virConnectPtr conn,
                                 const char * path,
                                 virErrorPtr err)
{
    virStorageVolPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 4, 1)
    setVirError(err, "Function virStorageVolLookupByPath compiled out (from 0.4.1)");
    return ret;
#else
    static virStorageVolLookupByPathType virStorageVolLookupByPathSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolLookupByPathSymbol = libvirtSymbol(libvirt,
                                                            "virStorageVolLookupByPath",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolLookupByPath");
        return ret;
    }
#  else
    virStorageVolLookupByPathSymbol = &virStorageVolLookupByPath;
#  endif

    ret = virStorageVolLookupByPathSymbol(conn,
                                          path);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStorageVolRefType)(virStorageVolPtr vol);

int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virStorageVolRef compiled out (from 0.6.0)");
    return ret;
#else
    static virStorageVolRefType virStorageVolRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolRefSymbol = libvirtSymbol(libvirt,
                                                   "virStorageVolRef",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolRef");
        return ret;
    }
#  else
    virStorageVolRefSymbol = &virStorageVolRef;
#  endif

    ret = virStorageVolRefSymbol(vol);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virStorageVolResize compiled out (from 0.9.10)");
    return ret;
#else
    static virStorageVolResizeType virStorageVolResizeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolResizeSymbol = libvirtSymbol(libvirt,
                                                      "virStorageVolResize",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolResize");
        return ret;
    }
#  else
    virStorageVolResizeSymbol = &virStorageVolResize;
#  endif

    ret = virStorageVolResizeSymbol(vol,
                                    capacity,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virStorageVolUpload compiled out (from 0.9.0)");
    return ret;
#else
    static virStorageVolUploadType virStorageVolUploadSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolUploadSymbol = libvirtSymbol(libvirt,
                                                      "virStorageVolUpload",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolUpload");
        return ret;
    }
#  else
    virStorageVolUploadSymbol = &virStorageVolUpload;
#  endif

    ret = virStorageVolUploadSymbol(vol,
                                    stream,
                                    offset,
                                    length,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStorageVolWipeType)(virStorageVolPtr vol,
                         unsigned int flags);

int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virStorageVolWipe compiled out (from 0.8.0)");
    return ret;
#else
    static virStorageVolWipeType virStorageVolWipeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolWipeSymbol = libvirtSymbol(libvirt,
                                                    "virStorageVolWipe",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolWipe");
        return ret;
    }
#  else
    virStorageVolWipeSymbol = &virStorageVolWipe;
#  endif

    ret = virStorageVolWipeSymbol(vol,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virStorageVolWipePattern compiled out (from 0.9.10)");
    return ret;
#else
    static virStorageVolWipePatternType virStorageVolWipePatternSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStorageVolWipePatternSymbol = libvirtSymbol(libvirt,
                                                           "virStorageVolWipePattern",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStorageVolWipePattern");
        return ret;
    }
#  else
    virStorageVolWipePatternSymbol = &virStorageVolWipePattern;
#  endif

    ret = virStorageVolWipePatternSymbol(vol,
                                         algorithm,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStreamAbortType)(virStreamPtr stream);

int
virStreamAbortWrapper(virStreamPtr stream,
                      virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamAbort compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamAbortType virStreamAbortSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamAbortSymbol = libvirtSymbol(libvirt,
                                                 "virStreamAbort",
                                                 &success,
                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamAbort");
        return ret;
    }
#  else
    virStreamAbortSymbol = &virStreamAbort;
#  endif

    ret = virStreamAbortSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamEventAddCallback compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamEventAddCallbackType virStreamEventAddCallbackSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamEventAddCallbackSymbol = libvirtSymbol(libvirt,
                                                            "virStreamEventAddCallback",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamEventAddCallback");
        return ret;
    }
#  else
    virStreamEventAddCallbackSymbol = &virStreamEventAddCallback;
#  endif

    ret = virStreamEventAddCallbackSymbol(stream,
                                          events,
                                          cb,
                                          opaque,
                                          ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStreamEventRemoveCallbackType)(virStreamPtr stream);

int
virStreamEventRemoveCallbackWrapper(virStreamPtr stream,
                                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamEventRemoveCallback compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamEventRemoveCallbackType virStreamEventRemoveCallbackSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamEventRemoveCallbackSymbol = libvirtSymbol(libvirt,
                                                               "virStreamEventRemoveCallback",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamEventRemoveCallback");
        return ret;
    }
#  else
    virStreamEventRemoveCallbackSymbol = &virStreamEventRemoveCallback;
#  endif

    ret = virStreamEventRemoveCallbackSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStreamEventUpdateCallbackType)(virStreamPtr stream,
                                    int events);

int
virStreamEventUpdateCallbackWrapper(virStreamPtr stream,
                                    int events,
                                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamEventUpdateCallback compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamEventUpdateCallbackType virStreamEventUpdateCallbackSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamEventUpdateCallbackSymbol = libvirtSymbol(libvirt,
                                                               "virStreamEventUpdateCallback",
                                                               &success,
                                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamEventUpdateCallback");
        return ret;
    }
#  else
    virStreamEventUpdateCallbackSymbol = &virStreamEventUpdateCallback;
#  endif

    ret = virStreamEventUpdateCallbackSymbol(stream,
                                             events);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStreamFinishType)(virStreamPtr stream);

int
virStreamFinishWrapper(virStreamPtr stream,
                       virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamFinish compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamFinishType virStreamFinishSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamFinishSymbol = libvirtSymbol(libvirt,
                                                  "virStreamFinish",
                                                  &success,
                                                  err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamFinish");
        return ret;
    }
#  else
    virStreamFinishSymbol = &virStreamFinish;
#  endif

    ret = virStreamFinishSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStreamFreeType)(virStreamPtr stream);

int
virStreamFreeWrapper(virStreamPtr stream,
                     virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamFree compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamFreeType virStreamFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamFreeSymbol = libvirtSymbol(libvirt,
                                                "virStreamFree",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamFree");
        return ret;
    }
#  else
    virStreamFreeSymbol = &virStreamFree;
#  endif

    ret = virStreamFreeSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef virStreamPtr
(*virStreamNewType)(virConnectPtr conn,
                    unsigned int flags);

virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err)
{
    virStreamPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamNew compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamNewType virStreamNewSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamNewSymbol = libvirtSymbol(libvirt,
                                               "virStreamNew",
                                               &success,
                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamNew");
        return ret;
    }
#  else
    virStreamNewSymbol = &virStreamNew;
#  endif

    ret = virStreamNewSymbol(conn,
                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamRecv compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamRecvType virStreamRecvSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamRecvSymbol = libvirtSymbol(libvirt,
                                                "virStreamRecv",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamRecv");
        return ret;
    }
#  else
    virStreamRecvSymbol = &virStreamRecv;
#  endif

    ret = virStreamRecvSymbol(stream,
                              data,
                              nbytes);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamRecvAll compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamRecvAllType virStreamRecvAllSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamRecvAllSymbol = libvirtSymbol(libvirt,
                                                   "virStreamRecvAll",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamRecvAll");
        return ret;
    }
#  else
    virStreamRecvAllSymbol = &virStreamRecvAll;
#  endif

    ret = virStreamRecvAllSymbol(stream,
                                 handler,
                                 opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamRecvFlags compiled out (from 3.4.0)");
    return ret;
#else
    static virStreamRecvFlagsType virStreamRecvFlagsSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamRecvFlagsSymbol = libvirtSymbol(libvirt,
                                                     "virStreamRecvFlags",
                                                     &success,
                                                     err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamRecvFlags");
        return ret;
    }
#  else
    virStreamRecvFlagsSymbol = &virStreamRecvFlags;
#  endif

    ret = virStreamRecvFlagsSymbol(stream,
                                   data,
                                   nbytes,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamRecvHole compiled out (from 3.4.0)");
    return ret;
#else
    static virStreamRecvHoleType virStreamRecvHoleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamRecvHoleSymbol = libvirtSymbol(libvirt,
                                                    "virStreamRecvHole",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamRecvHole");
        return ret;
    }
#  else
    virStreamRecvHoleSymbol = &virStreamRecvHole;
#  endif

    ret = virStreamRecvHoleSymbol(stream,
                                  length,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virStreamRefType)(virStreamPtr stream);

int
virStreamRefWrapper(virStreamPtr stream,
                    virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamRef compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamRefType virStreamRefSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamRefSymbol = libvirtSymbol(libvirt,
                                               "virStreamRef",
                                               &success,
                                               err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamRef");
        return ret;
    }
#  else
    virStreamRefSymbol = &virStreamRef;
#  endif

    ret = virStreamRefSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamSend compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamSendType virStreamSendSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamSendSymbol = libvirtSymbol(libvirt,
                                                "virStreamSend",
                                                &success,
                                                err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamSend");
        return ret;
    }
#  else
    virStreamSendSymbol = &virStreamSend;
#  endif

    ret = virStreamSendSymbol(stream,
                              data,
                              nbytes);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamSendAll compiled out (from 0.7.2)");
    return ret;
#else
    static virStreamSendAllType virStreamSendAllSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamSendAllSymbol = libvirtSymbol(libvirt,
                                                   "virStreamSendAll",
                                                   &success,
                                                   err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamSendAll");
        return ret;
    }
#  else
    virStreamSendAllSymbol = &virStreamSendAll;
#  endif

    ret = virStreamSendAllSymbol(stream,
                                 handler,
                                 opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamSendHole compiled out (from 3.4.0)");
    return ret;
#else
    static virStreamSendHoleType virStreamSendHoleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamSendHoleSymbol = libvirtSymbol(libvirt,
                                                    "virStreamSendHole",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamSendHole");
        return ret;
    }
#  else
    virStreamSendHoleSymbol = &virStreamSendHole;
#  endif

    ret = virStreamSendHoleSymbol(stream,
                                  length,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamSparseRecvAll compiled out (from 3.4.0)");
    return ret;
#else
    static virStreamSparseRecvAllType virStreamSparseRecvAllSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamSparseRecvAllSymbol = libvirtSymbol(libvirt,
                                                         "virStreamSparseRecvAll",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamSparseRecvAll");
        return ret;
    }
#  else
    virStreamSparseRecvAllSymbol = &virStreamSparseRecvAll;
#  endif

    ret = virStreamSparseRecvAllSymbol(stream,
                                       handler,
                                       holeHandler,
                                       opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamSparseSendAll compiled out (from 3.4.0)");
    return ret;
#else
    static virStreamSparseSendAllType virStreamSparseSendAllSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virStreamSparseSendAllSymbol = libvirtSymbol(libvirt,
                                                         "virStreamSparseSendAll",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virStreamSparseSendAll");
        return ret;
    }
#  else
    virStreamSparseSendAllSymbol = &virStreamSparseSendAll;
#  endif

    ret = virStreamSparseSendAllSymbol(stream,
                                       handler,
                                       holeHandler,
                                       skipHandler,
                                       opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddBoolean compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddBooleanType virTypedParamsAddBooleanSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddBooleanSymbol = libvirtSymbol(libvirt,
                                                           "virTypedParamsAddBoolean",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddBoolean");
        return ret;
    }
#  else
    virTypedParamsAddBooleanSymbol = &virTypedParamsAddBoolean;
#  endif

    ret = virTypedParamsAddBooleanSymbol(params,
                                         nparams,
                                         maxparams,
                                         name,
                                         value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddDouble compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddDoubleType virTypedParamsAddDoubleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddDoubleSymbol = libvirtSymbol(libvirt,
                                                          "virTypedParamsAddDouble",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddDouble");
        return ret;
    }
#  else
    virTypedParamsAddDoubleSymbol = &virTypedParamsAddDouble;
#  endif

    ret = virTypedParamsAddDoubleSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddFromString compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddFromStringType virTypedParamsAddFromStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddFromStringSymbol = libvirtSymbol(libvirt,
                                                              "virTypedParamsAddFromString",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddFromString");
        return ret;
    }
#  else
    virTypedParamsAddFromStringSymbol = &virTypedParamsAddFromString;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddInt compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddIntType virTypedParamsAddIntSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddIntSymbol = libvirtSymbol(libvirt,
                                                       "virTypedParamsAddInt",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddInt");
        return ret;
    }
#  else
    virTypedParamsAddIntSymbol = &virTypedParamsAddInt;
#  endif

    ret = virTypedParamsAddIntSymbol(params,
                                     nparams,
                                     maxparams,
                                     name,
                                     value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddLLong compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddLLongType virTypedParamsAddLLongSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddLLongSymbol = libvirtSymbol(libvirt,
                                                         "virTypedParamsAddLLong",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddLLong");
        return ret;
    }
#  else
    virTypedParamsAddLLongSymbol = &virTypedParamsAddLLong;
#  endif

    ret = virTypedParamsAddLLongSymbol(params,
                                       nparams,
                                       maxparams,
                                       name,
                                       value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddString compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddStringType virTypedParamsAddStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddStringSymbol = libvirtSymbol(libvirt,
                                                          "virTypedParamsAddString",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddString");
        return ret;
    }
#  else
    virTypedParamsAddStringSymbol = &virTypedParamsAddString;
#  endif

    ret = virTypedParamsAddStringSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 17)
    setVirError(err, "Function virTypedParamsAddStringList compiled out (from 1.2.17)");
    return ret;
#else
    static virTypedParamsAddStringListType virTypedParamsAddStringListSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddStringListSymbol = libvirtSymbol(libvirt,
                                                              "virTypedParamsAddStringList",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddStringList");
        return ret;
    }
#  else
    virTypedParamsAddStringListSymbol = &virTypedParamsAddStringList;
#  endif

    ret = virTypedParamsAddStringListSymbol(params,
                                            nparams,
                                            maxparams,
                                            name,
                                            values);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddUInt compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddUIntType virTypedParamsAddUIntSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddUIntSymbol = libvirtSymbol(libvirt,
                                                        "virTypedParamsAddUInt",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddUInt");
        return ret;
    }
#  else
    virTypedParamsAddUIntSymbol = &virTypedParamsAddUInt;
#  endif

    ret = virTypedParamsAddUIntSymbol(params,
                                      nparams,
                                      maxparams,
                                      name,
                                      value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddULLong compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsAddULLongType virTypedParamsAddULLongSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsAddULLongSymbol = libvirtSymbol(libvirt,
                                                          "virTypedParamsAddULLong",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsAddULLong");
        return ret;
    }
#  else
    virTypedParamsAddULLongSymbol = &virTypedParamsAddULLong;
#  endif

    ret = virTypedParamsAddULLongSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef void
(*virTypedParamsClearType)(virTypedParameterPtr params,
                           int nparams);

void
virTypedParamsClearWrapper(virTypedParameterPtr params,
                           int nparams)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(NULL, "Function virTypedParamsClear compiled out (from 1.0.2)");
    return;
#else
    static virTypedParamsClearType virTypedParamsClearSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virTypedParamsClearSymbol = libvirtSymbol(libvirt,
                                                      "virTypedParamsClear",
                                                      &success,
                                                      NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virTypedParamsClear");
        return;
    }
#  else
    virTypedParamsClearSymbol = &virTypedParamsClear;
#  endif

    virTypedParamsClearSymbol(params,
                              nparams);
#endif
}

typedef void
(*virTypedParamsFreeType)(virTypedParameterPtr params,
                          int nparams);

void
virTypedParamsFreeWrapper(virTypedParameterPtr params,
                          int nparams)
{

#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(NULL, "Function virTypedParamsFree compiled out (from 1.0.2)");
    return;
#else
    static virTypedParamsFreeType virTypedParamsFreeSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(NULL);
        if (success) {
            virTypedParamsFreeSymbol = libvirtSymbol(libvirt,
                                                     "virTypedParamsFree",
                                                     &success,
                                                     NULL);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return;
        }
    }

    if (!success) {
        setVirError(NULL, "Failed to load virTypedParamsFree");
        return;
    }
#  else
    virTypedParamsFreeSymbol = &virTypedParamsFree;
#  endif

    virTypedParamsFreeSymbol(params,
                             nparams);
#endif
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
    virTypedParameterPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGet compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetType virTypedParamsGetSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetSymbol = libvirtSymbol(libvirt,
                                                    "virTypedParamsGet",
                                                    &success,
                                                    err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGet");
        return ret;
    }
#  else
    virTypedParamsGetSymbol = &virTypedParamsGet;
#  endif

    ret = virTypedParamsGetSymbol(params,
                                  nparams,
                                  name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetBoolean compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetBooleanType virTypedParamsGetBooleanSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetBooleanSymbol = libvirtSymbol(libvirt,
                                                           "virTypedParamsGetBoolean",
                                                           &success,
                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetBoolean");
        return ret;
    }
#  else
    virTypedParamsGetBooleanSymbol = &virTypedParamsGetBoolean;
#  endif

    ret = virTypedParamsGetBooleanSymbol(params,
                                         nparams,
                                         name,
                                         value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetDouble compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetDoubleType virTypedParamsGetDoubleSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetDoubleSymbol = libvirtSymbol(libvirt,
                                                          "virTypedParamsGetDouble",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetDouble");
        return ret;
    }
#  else
    virTypedParamsGetDoubleSymbol = &virTypedParamsGetDouble;
#  endif

    ret = virTypedParamsGetDoubleSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetInt compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetIntType virTypedParamsGetIntSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetIntSymbol = libvirtSymbol(libvirt,
                                                       "virTypedParamsGetInt",
                                                       &success,
                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetInt");
        return ret;
    }
#  else
    virTypedParamsGetIntSymbol = &virTypedParamsGetInt;
#  endif

    ret = virTypedParamsGetIntSymbol(params,
                                     nparams,
                                     name,
                                     value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetLLong compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetLLongType virTypedParamsGetLLongSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetLLongSymbol = libvirtSymbol(libvirt,
                                                         "virTypedParamsGetLLong",
                                                         &success,
                                                         err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetLLong");
        return ret;
    }
#  else
    virTypedParamsGetLLongSymbol = &virTypedParamsGetLLong;
#  endif

    ret = virTypedParamsGetLLongSymbol(params,
                                       nparams,
                                       name,
                                       value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetString compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetStringType virTypedParamsGetStringSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetStringSymbol = libvirtSymbol(libvirt,
                                                          "virTypedParamsGetString",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetString");
        return ret;
    }
#  else
    virTypedParamsGetStringSymbol = &virTypedParamsGetString;
#  endif

    ret = virTypedParamsGetStringSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetUInt compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetUIntType virTypedParamsGetUIntSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetUIntSymbol = libvirtSymbol(libvirt,
                                                        "virTypedParamsGetUInt",
                                                        &success,
                                                        err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetUInt");
        return ret;
    }
#  else
    virTypedParamsGetUIntSymbol = &virTypedParamsGetUInt;
#  endif

    ret = virTypedParamsGetUIntSymbol(params,
                                      nparams,
                                      name,
                                      value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetULLong compiled out (from 1.0.2)");
    return ret;
#else
    static virTypedParamsGetULLongType virTypedParamsGetULLongSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virTypedParamsGetULLongSymbol = libvirtSymbol(libvirt,
                                                          "virTypedParamsGetULLong",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virTypedParamsGetULLong");
        return ret;
    }
#  else
    virTypedParamsGetULLongSymbol = &virTypedParamsGetULLong;
#  endif

    ret = virTypedParamsGetULLongSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainLxcEnterCGroupType)(virDomainPtr domain,
                               unsigned int flags);

int
virDomainLxcEnterCGroupWrapper(virDomainPtr domain,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virDomainLxcEnterCGroup compiled out (from 2.0.0)");
    return ret;
#else
    static virDomainLxcEnterCGroupType virDomainLxcEnterCGroupSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLxcEnterCGroupSymbol = libvirtSymbol(lxc,
                                                          "virDomainLxcEnterCGroup",
                                                          &success,
                                                          err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLxcEnterCGroup");
        return ret;
    }
#  else
    virDomainLxcEnterCGroupSymbol = &virDomainLxcEnterCGroup;
#  endif

    ret = virDomainLxcEnterCGroupSymbol(domain,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virDomainLxcEnterNamespace compiled out (from 1.0.2)");
    return ret;
#else
    static virDomainLxcEnterNamespaceType virDomainLxcEnterNamespaceSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLxcEnterNamespaceSymbol = libvirtSymbol(lxc,
                                                             "virDomainLxcEnterNamespace",
                                                             &success,
                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLxcEnterNamespace");
        return ret;
    }
#  else
    virDomainLxcEnterNamespaceSymbol = &virDomainLxcEnterNamespace;
#  endif

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
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 4)
    setVirError(err, "Function virDomainLxcEnterSecurityLabel compiled out (from 1.0.4)");
    return ret;
#else
    static virDomainLxcEnterSecurityLabelType virDomainLxcEnterSecurityLabelSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLxcEnterSecurityLabelSymbol = libvirtSymbol(lxc,
                                                                 "virDomainLxcEnterSecurityLabel",
                                                                 &success,
                                                                 err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLxcEnterSecurityLabel");
        return ret;
    }
#  else
    virDomainLxcEnterSecurityLabelSymbol = &virDomainLxcEnterSecurityLabel;
#  endif

    ret = virDomainLxcEnterSecurityLabelSymbol(model,
                                               label,
                                               oldlabel,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virDomainLxcOpenNamespace compiled out (from 1.0.2)");
    return ret;
#else
    static virDomainLxcOpenNamespaceType virDomainLxcOpenNamespaceSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainLxcOpenNamespaceSymbol = libvirtSymbol(lxc,
                                                            "virDomainLxcOpenNamespace",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainLxcOpenNamespace");
        return ret;
    }
#  else
    virDomainLxcOpenNamespaceSymbol = &virDomainLxcOpenNamespace;
#  endif

    ret = virDomainLxcOpenNamespaceSymbol(domain,
                                          fdlist,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virConnectDomainQemuMonitorEventDeregisterType)(virConnectPtr conn,
                                                  int callbackID);

int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 3)
    setVirError(err, "Function virConnectDomainQemuMonitorEventDeregister compiled out (from 1.2.3)");
    return ret;
#else
    static virConnectDomainQemuMonitorEventDeregisterType virConnectDomainQemuMonitorEventDeregisterSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainQemuMonitorEventDeregisterSymbol = libvirtSymbol(qemu,
                                                                             "virConnectDomainQemuMonitorEventDeregister",
                                                                             &success,
                                                                             err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainQemuMonitorEventDeregister");
        return ret;
    }
#  else
    virConnectDomainQemuMonitorEventDeregisterSymbol = &virConnectDomainQemuMonitorEventDeregister;
#  endif

    ret = virConnectDomainQemuMonitorEventDeregisterSymbol(conn,
                                                           callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(1, 2, 3)
    setVirError(err, "Function virConnectDomainQemuMonitorEventRegister compiled out (from 1.2.3)");
    return ret;
#else
    static virConnectDomainQemuMonitorEventRegisterType virConnectDomainQemuMonitorEventRegisterSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virConnectDomainQemuMonitorEventRegisterSymbol = libvirtSymbol(qemu,
                                                                           "virConnectDomainQemuMonitorEventRegister",
                                                                           &success,
                                                                           err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virConnectDomainQemuMonitorEventRegister");
        return ret;
    }
#  else
    virConnectDomainQemuMonitorEventRegisterSymbol = &virConnectDomainQemuMonitorEventRegister;
#  endif

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
#endif
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
    char * ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainQemuAgentCommand compiled out (from 0.10.0)");
    return ret;
#else
    static virDomainQemuAgentCommandType virDomainQemuAgentCommandSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainQemuAgentCommandSymbol = libvirtSymbol(qemu,
                                                            "virDomainQemuAgentCommand",
                                                            &success,
                                                            err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainQemuAgentCommand");
        return ret;
    }
#  else
    virDomainQemuAgentCommandSymbol = &virDomainQemuAgentCommand;
#  endif

    ret = virDomainQemuAgentCommandSymbol(domain,
                                          cmd,
                                          timeout,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    virDomainPtr ret = NULL;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainQemuAttach compiled out (from 0.9.4)");
    return ret;
#else
    static virDomainQemuAttachType virDomainQemuAttachSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainQemuAttachSymbol = libvirtSymbol(qemu,
                                                      "virDomainQemuAttach",
                                                      &success,
                                                      err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainQemuAttach");
        return ret;
    }
#  else
    virDomainQemuAttachSymbol = &virDomainQemuAttach;
#  endif

    ret = virDomainQemuAttachSymbol(conn,
                                    pid_value,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
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
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(0, 8, 3)
    setVirError(err, "Function virDomainQemuMonitorCommand compiled out (from 0.8.3)");
    return ret;
#else
    static virDomainQemuMonitorCommandType virDomainQemuMonitorCommandSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainQemuMonitorCommandSymbol = libvirtSymbol(qemu,
                                                              "virDomainQemuMonitorCommand",
                                                              &success,
                                                              err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainQemuMonitorCommand");
        return ret;
    }
#  else
    virDomainQemuMonitorCommandSymbol = &virDomainQemuMonitorCommand;
#  endif

    ret = virDomainQemuMonitorCommandSymbol(domain,
                                            cmd,
                                            result,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}

typedef int
(*virDomainQemuMonitorCommandWithFilesType)(virDomainPtr domain,
                                            const char * cmd,
                                            unsigned int ninfiles,
                                            int * infiles,
                                            unsigned int * noutfiles,
                                            int ** outfiles,
                                            char ** result,
                                            unsigned int flags);

int
virDomainQemuMonitorCommandWithFilesWrapper(virDomainPtr domain,
                                            const char * cmd,
                                            unsigned int ninfiles,
                                            int * infiles,
                                            unsigned int * noutfiles,
                                            int ** outfiles,
                                            char ** result,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = -1;
#if !USE_DLOPEN && !LIBVIR_CHECK_VERSION(8, 2, 0)
    setVirError(err, "Function virDomainQemuMonitorCommandWithFiles compiled out (from 8.2.0)");
    return ret;
#else
    static virDomainQemuMonitorCommandWithFilesType virDomainQemuMonitorCommandWithFilesSymbol;
#  ifdef USE_DLOPEN
    static bool once;
    static bool success;

    if (!once) {
        once = true;
        success = libvirtLoadOnceMain(err);
        if (success) {
            virDomainQemuMonitorCommandWithFilesSymbol = libvirtSymbol(qemu,
                                                                       "virDomainQemuMonitorCommandWithFiles",
                                                                       &success,
                                                                       err);
        }
        if (!success) {
            // return dlopen or dlsym dlerror failure
            return ret;
        }
    }

    if (!success) {
        setVirError(err, "Failed to load virDomainQemuMonitorCommandWithFiles");
        return ret;
    }
#  else
    virDomainQemuMonitorCommandWithFilesSymbol = &virDomainQemuMonitorCommandWithFiles;
#  endif

    ret = virDomainQemuMonitorCommandWithFilesSymbol(domain,
                                                     cmd,
                                                     ninfiles,
                                                     infiles,
                                                     noutfiles,
                                                     outfiles,
                                                     result,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
#endif
}



