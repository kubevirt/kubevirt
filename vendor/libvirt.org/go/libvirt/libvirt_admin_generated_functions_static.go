//go:build !libvirt_without_admin && !libvirt_dlopen
// +build !libvirt_without_admin,!libvirt_dlopen

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
#cgo pkg-config: libvirt-admin
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_admin_generated.h"
#include "error_helper.h"


int
virAdmClientCloseWrapper(virAdmClientPtr client,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmClientClose not available prior to libvirt version 2.0.0");
#else
    ret = virAdmClientClose(client,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmClientFreeWrapper(virAdmClientPtr client,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmClientFree not available prior to libvirt version 2.0.0");
#else
    ret = virAdmClientFree(client);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

unsigned long long
virAdmClientGetIDWrapper(virAdmClientPtr client,
                         virErrorPtr err)
{
    unsigned long long ret = 0;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmClientGetID not available prior to libvirt version 2.0.0");
#else
    ret = virAdmClientGetID(client);
    if (ret == 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmClientGetInfoWrapper(virAdmClientPtr client,
                           virTypedParameterPtr * params,
                           int * nparams,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmClientGetInfo not available prior to libvirt version 2.0.0");
#else
    ret = virAdmClientGetInfo(client,
                              params,
                              nparams,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

long long
virAdmClientGetTimestampWrapper(virAdmClientPtr client,
                                virErrorPtr err)
{
    long long ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmClientGetTimestamp not available prior to libvirt version 2.0.0");
#else
    ret = virAdmClientGetTimestamp(client);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmClientGetTransportWrapper(virAdmClientPtr client,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmClientGetTransport not available prior to libvirt version 2.0.0");
#else
    ret = virAdmClientGetTransport(client);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectCloseWrapper(virAdmConnectPtr conn,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectClose not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectClose(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectDaemonShutdownWrapper(virAdmConnectPtr conn,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(11, 2, 0)
    setVirError(err, "Function virAdmConnectDaemonShutdown not available prior to libvirt version 11.2.0");
#else
    ret = virAdmConnectDaemonShutdown(conn,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectGetLibVersionWrapper(virAdmConnectPtr conn,
                                  unsigned long long * libVer,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectGetLibVersion not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectGetLibVersion(conn,
                                     libVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectGetLoggingFiltersWrapper(virAdmConnectPtr conn,
                                      char ** filters,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virAdmConnectGetLoggingFilters not available prior to libvirt version 3.0.0");
#else
    ret = virAdmConnectGetLoggingFilters(conn,
                                         filters,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectGetLoggingOutputsWrapper(virAdmConnectPtr conn,
                                      char ** outputs,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virAdmConnectGetLoggingOutputs not available prior to libvirt version 3.0.0");
#else
    ret = virAdmConnectGetLoggingOutputs(conn,
                                         outputs,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virAdmConnectGetURIWrapper(virAdmConnectPtr conn,
                           virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectGetURI not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectGetURI(conn);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectIsAliveWrapper(virAdmConnectPtr conn,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectIsAlive not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectIsAlive(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectListServersWrapper(virAdmConnectPtr conn,
                                virAdmServerPtr ** servers,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectListServers not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectListServers(conn,
                                   servers,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virAdmServerPtr
virAdmConnectLookupServerWrapper(virAdmConnectPtr conn,
                                 const char * name,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    virAdmServerPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectLookupServer not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectLookupServer(conn,
                                    name,
                                    flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virAdmConnectPtr
virAdmConnectOpenWrapper(const char * name,
                         unsigned int flags,
                         virErrorPtr err)
{
    virAdmConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectOpen not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectOpen(name,
                            flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectRefWrapper(virAdmConnectPtr conn,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectRef not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectRef(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectRegisterCloseCallbackWrapper(virAdmConnectPtr conn,
                                          virAdmConnectCloseFunc cb,
                                          void * opaque,
                                          virFreeCallback freecb,
                                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectRegisterCloseCallback not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectRegisterCloseCallback(conn,
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
virAdmConnectSetDaemonTimeoutWrapper(virAdmConnectPtr conn,
                                     unsigned int timeout,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(8, 6, 0)
    setVirError(err, "Function virAdmConnectSetDaemonTimeout not available prior to libvirt version 8.6.0");
#else
    ret = virAdmConnectSetDaemonTimeout(conn,
                                        timeout,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectSetLoggingFiltersWrapper(virAdmConnectPtr conn,
                                      const char * filters,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virAdmConnectSetLoggingFilters not available prior to libvirt version 3.0.0");
#else
    ret = virAdmConnectSetLoggingFilters(conn,
                                         filters,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectSetLoggingOutputsWrapper(virAdmConnectPtr conn,
                                      const char * outputs,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 0, 0)
    setVirError(err, "Function virAdmConnectSetLoggingOutputs not available prior to libvirt version 3.0.0");
#else
    ret = virAdmConnectSetLoggingOutputs(conn,
                                         outputs,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmConnectUnregisterCloseCallbackWrapper(virAdmConnectPtr conn,
                                            virAdmConnectCloseFunc cb,
                                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmConnectUnregisterCloseCallback not available prior to libvirt version 2.0.0");
#else
    ret = virAdmConnectUnregisterCloseCallback(conn,
                                               cb);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmGetVersionWrapper(unsigned long long * libVer,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmGetVersion not available prior to libvirt version 2.0.0");
#else
    ret = virAdmGetVersion(libVer);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmInitializeWrapper(virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmInitialize not available prior to libvirt version 2.0.0");
#else
    ret = virAdmInitialize();
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmServerFreeWrapper(virAdmServerPtr srv,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerFree not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerFree(srv);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmServerGetClientLimitsWrapper(virAdmServerPtr srv,
                                   virTypedParameterPtr * params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerGetClientLimits not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerGetClientLimits(srv,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virAdmServerGetNameWrapper(virAdmServerPtr srv,
                           virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerGetName not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerGetName(srv);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmServerGetThreadPoolParametersWrapper(virAdmServerPtr srv,
                                           virTypedParameterPtr * params,
                                           int * nparams,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerGetThreadPoolParameters not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerGetThreadPoolParameters(srv,
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
virAdmServerListClientsWrapper(virAdmServerPtr srv,
                               virAdmClientPtr ** clients,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerListClients not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerListClients(srv,
                                  clients,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virAdmClientPtr
virAdmServerLookupClientWrapper(virAdmServerPtr srv,
                                unsigned long long id,
                                unsigned int flags,
                                virErrorPtr err)
{
    virAdmClientPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerLookupClient not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerLookupClient(srv,
                                   id,
                                   flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virAdmServerSetClientLimitsWrapper(virAdmServerPtr srv,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerSetClientLimits not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerSetClientLimits(srv,
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
virAdmServerSetThreadPoolParametersWrapper(virAdmServerPtr srv,
                                           virTypedParameterPtr params,
                                           int nparams,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerSetThreadPoolParameters not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerSetThreadPoolParameters(srv,
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
virAdmServerUpdateTlsFilesWrapper(virAdmServerPtr srv,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virAdmServerUpdateTlsFiles not available prior to libvirt version 2.0.0");
#else
    ret = virAdmServerUpdateTlsFiles(srv,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
