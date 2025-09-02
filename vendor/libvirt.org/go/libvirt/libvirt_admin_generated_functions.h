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

#pragma once

int
virAdmClientCloseWrapper(virAdmClientPtr client,
                         unsigned int flags,
                         virErrorPtr err);

int
virAdmClientFreeWrapper(virAdmClientPtr client,
                        virErrorPtr err);

unsigned long long
virAdmClientGetIDWrapper(virAdmClientPtr client,
                         virErrorPtr err);

int
virAdmClientGetInfoWrapper(virAdmClientPtr client,
                           virTypedParameterPtr * params,
                           int * nparams,
                           unsigned int flags,
                           virErrorPtr err);

long long
virAdmClientGetTimestampWrapper(virAdmClientPtr client,
                                virErrorPtr err);

int
virAdmClientGetTransportWrapper(virAdmClientPtr client,
                                virErrorPtr err);

int
virAdmConnectCloseWrapper(virAdmConnectPtr conn,
                          virErrorPtr err);

int
virAdmConnectDaemonShutdownWrapper(virAdmConnectPtr conn,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virAdmConnectGetLibVersionWrapper(virAdmConnectPtr conn,
                                  unsigned long long * libVer,
                                  virErrorPtr err);

int
virAdmConnectGetLoggingFiltersWrapper(virAdmConnectPtr conn,
                                      char ** filters,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virAdmConnectGetLoggingOutputsWrapper(virAdmConnectPtr conn,
                                      char ** outputs,
                                      unsigned int flags,
                                      virErrorPtr err);

char *
virAdmConnectGetURIWrapper(virAdmConnectPtr conn,
                           virErrorPtr err);

int
virAdmConnectIsAliveWrapper(virAdmConnectPtr conn,
                            virErrorPtr err);

int
virAdmConnectListServersWrapper(virAdmConnectPtr conn,
                                virAdmServerPtr ** servers,
                                unsigned int flags,
                                virErrorPtr err);

virAdmServerPtr
virAdmConnectLookupServerWrapper(virAdmConnectPtr conn,
                                 const char * name,
                                 unsigned int flags,
                                 virErrorPtr err);

virAdmConnectPtr
virAdmConnectOpenWrapper(const char * name,
                         unsigned int flags,
                         virErrorPtr err);

int
virAdmConnectRefWrapper(virAdmConnectPtr conn,
                        virErrorPtr err);

int
virAdmConnectRegisterCloseCallbackWrapper(virAdmConnectPtr conn,
                                          virAdmConnectCloseFunc cb,
                                          void * opaque,
                                          virFreeCallback freecb,
                                          virErrorPtr err);

int
virAdmConnectSetDaemonTimeoutWrapper(virAdmConnectPtr conn,
                                     unsigned int timeout,
                                     unsigned int flags,
                                     virErrorPtr err);

int
virAdmConnectSetLoggingFiltersWrapper(virAdmConnectPtr conn,
                                      const char * filters,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virAdmConnectSetLoggingOutputsWrapper(virAdmConnectPtr conn,
                                      const char * outputs,
                                      unsigned int flags,
                                      virErrorPtr err);

int
virAdmConnectUnregisterCloseCallbackWrapper(virAdmConnectPtr conn,
                                            virAdmConnectCloseFunc cb,
                                            virErrorPtr err);

int
virAdmGetVersionWrapper(unsigned long long * libVer,
                        virErrorPtr err);

int
virAdmInitializeWrapper(virErrorPtr err);

int
virAdmServerFreeWrapper(virAdmServerPtr srv,
                        virErrorPtr err);

int
virAdmServerGetClientLimitsWrapper(virAdmServerPtr srv,
                                   virTypedParameterPtr * params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

const char *
virAdmServerGetNameWrapper(virAdmServerPtr srv,
                           virErrorPtr err);

int
virAdmServerGetThreadPoolParametersWrapper(virAdmServerPtr srv,
                                           virTypedParameterPtr * params,
                                           int * nparams,
                                           unsigned int flags,
                                           virErrorPtr err);

int
virAdmServerListClientsWrapper(virAdmServerPtr srv,
                               virAdmClientPtr ** clients,
                               unsigned int flags,
                               virErrorPtr err);

virAdmClientPtr
virAdmServerLookupClientWrapper(virAdmServerPtr srv,
                                unsigned long long id,
                                unsigned int flags,
                                virErrorPtr err);

int
virAdmServerSetClientLimitsWrapper(virAdmServerPtr srv,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err);

int
virAdmServerSetThreadPoolParametersWrapper(virAdmServerPtr srv,
                                           virTypedParameterPtr params,
                                           int nparams,
                                           unsigned int flags,
                                           virErrorPtr err);

int
virAdmServerUpdateTlsFilesWrapper(virAdmServerPtr srv,
                                  unsigned int flags,
                                  virErrorPtr err);
