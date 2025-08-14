//go:build !libvirt_without_admin && libvirt_dlopen
// +build !libvirt_without_admin,libvirt_dlopen

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
#include "libvirt_admin_generated_dlopen.h"
#include "error_helper.h"


typedef int
(*virAdmClientCloseFuncType)(virAdmClientPtr client,
                             unsigned int flags);

int
virAdmClientCloseWrapper(virAdmClientPtr client,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virAdmClientCloseFuncType virAdmClientCloseSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmClientClose",
                            (void**)&virAdmClientCloseSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmClientCloseSymbol(client,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmClientFreeFuncType)(virAdmClientPtr client);

int
virAdmClientFreeWrapper(virAdmClientPtr client,
                        virErrorPtr err)
{
    int ret = -1;
    static virAdmClientFreeFuncType virAdmClientFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmClientFree",
                            (void**)&virAdmClientFreeSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmClientFreeSymbol(client);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned long long
(*virAdmClientGetIDFuncType)(virAdmClientPtr client);

unsigned long long
virAdmClientGetIDWrapper(virAdmClientPtr client,
                         virErrorPtr err)
{
    unsigned long long ret = 0;
    static virAdmClientGetIDFuncType virAdmClientGetIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmClientGetID",
                            (void**)&virAdmClientGetIDSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmClientGetIDSymbol(client);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmClientGetInfoFuncType)(virAdmClientPtr client,
                               virTypedParameterPtr * params,
                               int * nparams,
                               unsigned int flags);

int
virAdmClientGetInfoWrapper(virAdmClientPtr client,
                           virTypedParameterPtr * params,
                           int * nparams,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virAdmClientGetInfoFuncType virAdmClientGetInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmClientGetInfo",
                            (void**)&virAdmClientGetInfoSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmClientGetInfoSymbol(client,
                                    params,
                                    nparams,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef long long
(*virAdmClientGetTimestampFuncType)(virAdmClientPtr client);

long long
virAdmClientGetTimestampWrapper(virAdmClientPtr client,
                                virErrorPtr err)
{
    long long ret = -1;
    static virAdmClientGetTimestampFuncType virAdmClientGetTimestampSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmClientGetTimestamp",
                            (void**)&virAdmClientGetTimestampSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmClientGetTimestampSymbol(client);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmClientGetTransportFuncType)(virAdmClientPtr client);

int
virAdmClientGetTransportWrapper(virAdmClientPtr client,
                                virErrorPtr err)
{
    int ret = -1;
    static virAdmClientGetTransportFuncType virAdmClientGetTransportSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmClientGetTransport",
                            (void**)&virAdmClientGetTransportSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmClientGetTransportSymbol(client);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectCloseFuncType)(virAdmConnectPtr conn);

int
virAdmConnectCloseWrapper(virAdmConnectPtr conn,
                          virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectCloseFuncType virAdmConnectCloseSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectClose",
                            (void**)&virAdmConnectCloseSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectCloseSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectDaemonShutdownFuncType)(virAdmConnectPtr conn,
                                       unsigned int flags);

int
virAdmConnectDaemonShutdownWrapper(virAdmConnectPtr conn,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectDaemonShutdownFuncType virAdmConnectDaemonShutdownSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectDaemonShutdown",
                            (void**)&virAdmConnectDaemonShutdownSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectDaemonShutdownSymbol(conn,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectGetLibVersionFuncType)(virAdmConnectPtr conn,
                                      unsigned long long * libVer);

int
virAdmConnectGetLibVersionWrapper(virAdmConnectPtr conn,
                                  unsigned long long * libVer,
                                  virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectGetLibVersionFuncType virAdmConnectGetLibVersionSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectGetLibVersion",
                            (void**)&virAdmConnectGetLibVersionSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectGetLibVersionSymbol(conn,
                                           libVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectGetLoggingFiltersFuncType)(virAdmConnectPtr conn,
                                          char ** filters,
                                          unsigned int flags);

int
virAdmConnectGetLoggingFiltersWrapper(virAdmConnectPtr conn,
                                      char ** filters,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectGetLoggingFiltersFuncType virAdmConnectGetLoggingFiltersSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectGetLoggingFilters",
                            (void**)&virAdmConnectGetLoggingFiltersSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectGetLoggingFiltersSymbol(conn,
                                               filters,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectGetLoggingOutputsFuncType)(virAdmConnectPtr conn,
                                          char ** outputs,
                                          unsigned int flags);

int
virAdmConnectGetLoggingOutputsWrapper(virAdmConnectPtr conn,
                                      char ** outputs,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectGetLoggingOutputsFuncType virAdmConnectGetLoggingOutputsSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectGetLoggingOutputs",
                            (void**)&virAdmConnectGetLoggingOutputsSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectGetLoggingOutputsSymbol(conn,
                                               outputs,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virAdmConnectGetURIFuncType)(virAdmConnectPtr conn);

char *
virAdmConnectGetURIWrapper(virAdmConnectPtr conn,
                           virErrorPtr err)
{
    char * ret = NULL;
    static virAdmConnectGetURIFuncType virAdmConnectGetURISymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectGetURI",
                            (void**)&virAdmConnectGetURISymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectGetURISymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectIsAliveFuncType)(virAdmConnectPtr conn);

int
virAdmConnectIsAliveWrapper(virAdmConnectPtr conn,
                            virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectIsAliveFuncType virAdmConnectIsAliveSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectIsAlive",
                            (void**)&virAdmConnectIsAliveSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectIsAliveSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectListServersFuncType)(virAdmConnectPtr conn,
                                    virAdmServerPtr ** servers,
                                    unsigned int flags);

int
virAdmConnectListServersWrapper(virAdmConnectPtr conn,
                                virAdmServerPtr ** servers,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectListServersFuncType virAdmConnectListServersSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectListServers",
                            (void**)&virAdmConnectListServersSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectListServersSymbol(conn,
                                         servers,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virAdmServerPtr
(*virAdmConnectLookupServerFuncType)(virAdmConnectPtr conn,
                                     const char * name,
                                     unsigned int flags);

virAdmServerPtr
virAdmConnectLookupServerWrapper(virAdmConnectPtr conn,
                                 const char * name,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    virAdmServerPtr ret = NULL;
    static virAdmConnectLookupServerFuncType virAdmConnectLookupServerSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectLookupServer",
                            (void**)&virAdmConnectLookupServerSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectLookupServerSymbol(conn,
                                          name,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virAdmConnectPtr
(*virAdmConnectOpenFuncType)(const char * name,
                             unsigned int flags);

virAdmConnectPtr
virAdmConnectOpenWrapper(const char * name,
                         unsigned int flags,
                         virErrorPtr err)
{
    virAdmConnectPtr ret = NULL;
    static virAdmConnectOpenFuncType virAdmConnectOpenSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectOpen",
                            (void**)&virAdmConnectOpenSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectOpenSymbol(name,
                                  flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectRefFuncType)(virAdmConnectPtr conn);

int
virAdmConnectRefWrapper(virAdmConnectPtr conn,
                        virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectRefFuncType virAdmConnectRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectRef",
                            (void**)&virAdmConnectRefSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectRefSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectRegisterCloseCallbackFuncType)(virAdmConnectPtr conn,
                                              virAdmConnectCloseFunc cb,
                                              void * opaque,
                                              virFreeCallback freecb);

int
virAdmConnectRegisterCloseCallbackWrapper(virAdmConnectPtr conn,
                                          virAdmConnectCloseFunc cb,
                                          void * opaque,
                                          virFreeCallback freecb,
                                          virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectRegisterCloseCallbackFuncType virAdmConnectRegisterCloseCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectRegisterCloseCallback",
                            (void**)&virAdmConnectRegisterCloseCallbackSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectRegisterCloseCallbackSymbol(conn,
                                                   cb,
                                                   opaque,
                                                   freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectSetDaemonTimeoutFuncType)(virAdmConnectPtr conn,
                                         unsigned int timeout,
                                         unsigned int flags);

int
virAdmConnectSetDaemonTimeoutWrapper(virAdmConnectPtr conn,
                                     unsigned int timeout,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectSetDaemonTimeoutFuncType virAdmConnectSetDaemonTimeoutSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectSetDaemonTimeout",
                            (void**)&virAdmConnectSetDaemonTimeoutSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectSetDaemonTimeoutSymbol(conn,
                                              timeout,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectSetLoggingFiltersFuncType)(virAdmConnectPtr conn,
                                          const char * filters,
                                          unsigned int flags);

int
virAdmConnectSetLoggingFiltersWrapper(virAdmConnectPtr conn,
                                      const char * filters,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectSetLoggingFiltersFuncType virAdmConnectSetLoggingFiltersSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectSetLoggingFilters",
                            (void**)&virAdmConnectSetLoggingFiltersSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectSetLoggingFiltersSymbol(conn,
                                               filters,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectSetLoggingOutputsFuncType)(virAdmConnectPtr conn,
                                          const char * outputs,
                                          unsigned int flags);

int
virAdmConnectSetLoggingOutputsWrapper(virAdmConnectPtr conn,
                                      const char * outputs,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectSetLoggingOutputsFuncType virAdmConnectSetLoggingOutputsSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectSetLoggingOutputs",
                            (void**)&virAdmConnectSetLoggingOutputsSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectSetLoggingOutputsSymbol(conn,
                                               outputs,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmConnectUnregisterCloseCallbackFuncType)(virAdmConnectPtr conn,
                                                virAdmConnectCloseFunc cb);

int
virAdmConnectUnregisterCloseCallbackWrapper(virAdmConnectPtr conn,
                                            virAdmConnectCloseFunc cb,
                                            virErrorPtr err)
{
    int ret = -1;
    static virAdmConnectUnregisterCloseCallbackFuncType virAdmConnectUnregisterCloseCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmConnectUnregisterCloseCallback",
                            (void**)&virAdmConnectUnregisterCloseCallbackSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmConnectUnregisterCloseCallbackSymbol(conn,
                                                     cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmGetVersionFuncType)(unsigned long long * libVer);

int
virAdmGetVersionWrapper(unsigned long long * libVer,
                        virErrorPtr err)
{
    int ret = -1;
    static virAdmGetVersionFuncType virAdmGetVersionSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmGetVersion",
                            (void**)&virAdmGetVersionSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmGetVersionSymbol(libVer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmInitializeFuncType)(void);

int
virAdmInitializeWrapper(virErrorPtr err)
{
    int ret = -1;
    static virAdmInitializeFuncType virAdmInitializeSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmInitialize",
                            (void**)&virAdmInitializeSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmInitializeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerFreeFuncType)(virAdmServerPtr srv);

int
virAdmServerFreeWrapper(virAdmServerPtr srv,
                        virErrorPtr err)
{
    int ret = -1;
    static virAdmServerFreeFuncType virAdmServerFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerFree",
                            (void**)&virAdmServerFreeSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerFreeSymbol(srv);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerGetClientLimitsFuncType)(virAdmServerPtr srv,
                                       virTypedParameterPtr * params,
                                       int * nparams,
                                       unsigned int flags);

int
virAdmServerGetClientLimitsWrapper(virAdmServerPtr srv,
                                   virTypedParameterPtr * params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virAdmServerGetClientLimitsFuncType virAdmServerGetClientLimitsSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerGetClientLimits",
                            (void**)&virAdmServerGetClientLimitsSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerGetClientLimitsSymbol(srv,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virAdmServerGetNameFuncType)(virAdmServerPtr srv);

const char *
virAdmServerGetNameWrapper(virAdmServerPtr srv,
                           virErrorPtr err)
{
    const char * ret = NULL;
    static virAdmServerGetNameFuncType virAdmServerGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerGetName",
                            (void**)&virAdmServerGetNameSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerGetNameSymbol(srv);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerGetThreadPoolParametersFuncType)(virAdmServerPtr srv,
                                               virTypedParameterPtr * params,
                                               int * nparams,
                                               unsigned int flags);

int
virAdmServerGetThreadPoolParametersWrapper(virAdmServerPtr srv,
                                           virTypedParameterPtr * params,
                                           int * nparams,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
    static virAdmServerGetThreadPoolParametersFuncType virAdmServerGetThreadPoolParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerGetThreadPoolParameters",
                            (void**)&virAdmServerGetThreadPoolParametersSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerGetThreadPoolParametersSymbol(srv,
                                                    params,
                                                    nparams,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerListClientsFuncType)(virAdmServerPtr srv,
                                   virAdmClientPtr ** clients,
                                   unsigned int flags);

int
virAdmServerListClientsWrapper(virAdmServerPtr srv,
                               virAdmClientPtr ** clients,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virAdmServerListClientsFuncType virAdmServerListClientsSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerListClients",
                            (void**)&virAdmServerListClientsSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerListClientsSymbol(srv,
                                        clients,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virAdmClientPtr
(*virAdmServerLookupClientFuncType)(virAdmServerPtr srv,
                                    unsigned long long id,
                                    unsigned int flags);

virAdmClientPtr
virAdmServerLookupClientWrapper(virAdmServerPtr srv,
                                unsigned long long id,
                                unsigned int flags,
                                virErrorPtr err)
{
    virAdmClientPtr ret = NULL;
    static virAdmServerLookupClientFuncType virAdmServerLookupClientSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerLookupClient",
                            (void**)&virAdmServerLookupClientSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerLookupClientSymbol(srv,
                                         id,
                                         flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerSetClientLimitsFuncType)(virAdmServerPtr srv,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags);

int
virAdmServerSetClientLimitsWrapper(virAdmServerPtr srv,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virAdmServerSetClientLimitsFuncType virAdmServerSetClientLimitsSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerSetClientLimits",
                            (void**)&virAdmServerSetClientLimitsSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerSetClientLimitsSymbol(srv,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerSetThreadPoolParametersFuncType)(virAdmServerPtr srv,
                                               virTypedParameterPtr params,
                                               int nparams,
                                               unsigned int flags);

int
virAdmServerSetThreadPoolParametersWrapper(virAdmServerPtr srv,
                                           virTypedParameterPtr params,
                                           int nparams,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
    static virAdmServerSetThreadPoolParametersFuncType virAdmServerSetThreadPoolParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerSetThreadPoolParameters",
                            (void**)&virAdmServerSetThreadPoolParametersSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerSetThreadPoolParametersSymbol(srv,
                                                    params,
                                                    nparams,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virAdmServerUpdateTlsFilesFuncType)(virAdmServerPtr srv,
                                      unsigned int flags);

int
virAdmServerUpdateTlsFilesWrapper(virAdmServerPtr srv,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virAdmServerUpdateTlsFilesFuncType virAdmServerUpdateTlsFilesSymbol;
    static bool once;
    static bool success;

    if (!libvirtAdminSymbol("virAdmServerUpdateTlsFiles",
                            (void**)&virAdmServerUpdateTlsFilesSymbol,
                            &once,
                            &success,
                            err)) {
        return ret;
    }
    ret = virAdmServerUpdateTlsFilesSymbol(srv,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
