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

#include <stdint.h>
#include "module-generated.h"

void domainEventLifecycleCallbackHelper(virConnectPtr c, virDomainPtr d,
                                     int event, int detail, void *data);

void domainEventGenericCallbackHelper(virConnectPtr c, virDomainPtr d, void *data);

void domainEventRTCChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                     long long utcoffset, void *data);

void domainEventWatchdogCallbackHelper(virConnectPtr c, virDomainPtr d,
                                    int action, void *data);

void domainEventIOErrorCallbackHelper(virConnectPtr c, virDomainPtr d,
                                   const char *srcPath, const char *devAlias,
                                   int action, void *data);

void domainEventGraphicsCallbackHelper(virConnectPtr c, virDomainPtr d,
                                    int phase, const virDomainEventGraphicsAddress *local,
                                    const virDomainEventGraphicsAddress *remote,
                                    const char *authScheme,
                                    const virDomainEventGraphicsSubject *subject, void *data);

void domainEventIOErrorReasonCallbackHelper(virConnectPtr c, virDomainPtr d,
                                         const char *srcPath, const char *devAlias,
                                         int action, const char *reason, void *data);

void domainEventBlockJobCallbackHelper(virConnectPtr c, virDomainPtr d,
                                    const char *disk, int type, int status, void *data);

void domainEventDiskChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                      const char *oldSrcPath, const char *newSrcPath,
                                      const char *devAlias, int reason, void *data);

void domainEventTrayChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                      const char *devAlias, int reason, void *data);

void domainEventPMSuspendCallbackHelper(virConnectPtr c, virDomainPtr d,
                                  int reason, void *data);

void domainEventPMWakeupCallbackHelper(virConnectPtr c, virDomainPtr d,
                                     int reason, void *data);

void domainEventPMSuspendDiskCallbackHelper(virConnectPtr c, virDomainPtr d,
                                          int reason, void *data);

void domainEventBalloonChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                         unsigned long long actual, void *data);

void domainEventDeviceRemovedCallbackHelper(virConnectPtr c, virDomainPtr d,
                                         const char *devAlias, void *data);

void domainEventTunableCallbackHelper(virConnectPtr conn,
                                   virDomainPtr dom,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   void *opaque);

void domainEventAgentLifecycleCallbackHelper(virConnectPtr conn,
                                          virDomainPtr dom,
                                          int state,
                                          int reason,
                                          void *opaque);

void domainEventDeviceAddedCallbackHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       const char *devAlias,
                                       void *opaque);

void domainEventMigrationIterationCallbackHelper(virConnectPtr conn,
                                              virDomainPtr dom,
                                              int iteration,
                                              void *opaque);

void domainEventJobCompletedCallbackHelper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        virTypedParameterPtr params,
                                        int nparams,
                                        void *opaque);

void domainEventDeviceRemovalFailedCallbackHelper(virConnectPtr conn,
                                               virDomainPtr dom,
                                               const char *devAlias,
                                               void *opaque);

void domainEventMetadataChangeCallbackHelper(virConnectPtr conn,
                                          virDomainPtr dom,
                                          int type,
                                          const char *nsuri,
                                          void *opaque);

void domainEventBlockThresholdCallbackHelper(virConnectPtr conn,
                                          virDomainPtr dom,
                                          const char *dev,
                                          const char *path,
                                          unsigned long long threshold,
                                          unsigned long long excess,
                                          void *opaque);

void domainEventMemoryFailureCallbackHelper(virConnectPtr conn,
                                            virDomainPtr dom,
                                            int recipient,
                                            int action,
                                            unsigned int flags,
                                            void *opaque);

void eventHandleCallbackInvoke(int watch,
                               int fd,
                               int events,
                               uintptr_t callback,
                               uintptr_t opaque);

void eventTimeoutCallbackInvoke(int timer,
                                uintptr_t callback,
                                uintptr_t opaque);

void eventHandleCallbackFree(uintptr_t callback,
                             uintptr_t opaque);

void eventTimeoutCallbackFree(uintptr_t callback,
                              uintptr_t opaque);

void networkEventLifecycleCallbackHelper(virConnectPtr c, virNetworkPtr d,
                                         int event, int detail, void *data);

void nodeDeviceEventGenericCallbackHelper(virConnectPtr c, virNodeDevicePtr d, void *data);

void nodeDeviceEventLifecycleCallbackHelper(virConnectPtr c, virNodeDevicePtr d,
                                            int event, int detail, void *data);

void secretEventLifecycleCallbackHelper(virConnectPtr c, virSecretPtr d,
                                        int event, int detail, void *data);

void secretEventGenericCallbackHelper(virConnectPtr c, virSecretPtr d,
                                      void *data);

void storagePoolEventLifecycleCallbackHelper(virConnectPtr c, virStoragePoolPtr d,
                                             int event, int detail, void *data);

void storagePoolEventGenericCallbackHelper(virConnectPtr c, virStoragePoolPtr d,
                                           void *data);


virConnectPtr virConnectOpenAuthHelper(const char *name,
                                       int *credtype,
                                       unsigned int ncredtype,
                                       int callbackID,
                                       unsigned int flags,
                                       virErrorPtr err);

virConnectPtr virConnectOpenAuthDefaultHelper(const char *name,
                                              unsigned int flags,
                                              virErrorPtr err);

int virConnectRegisterCloseCallbackHelper(virConnectPtr conn,
                                          long goCallbackId,
                                          virErrorPtr err);

int virConnectUnregisterCloseCallbackHelper(virConnectPtr conn,
                                            virErrorPtr err);

int
virConnectDomainEventRegisterAnyHelper(virConnectPtr c,
                                       virDomainPtr d,
                                       int eventID,
                                       virConnectDomainEventGenericCallback cb,
                                       long goCallbackId,
                                       virErrorPtr err);

int virEventAddHandleHelper(int fd,
                            int events,
                            int callbackID,
                            virErrorPtr err);

int
virEventAddTimeoutHelper(int timeout,
                         int callbackID,
                         virErrorPtr err);

void virEventRegisterImplHelper(void);

int virConnectNetworkEventRegisterAnyHelper(virConnectPtr c,
                                            virNetworkPtr d,
                                            int eventID,
                                            virConnectNetworkEventGenericCallback cb,
                                            long goCallbackId,
                                            virErrorPtr err);

int virConnectNodeDeviceEventRegisterAnyHelper(virConnectPtr c,
                                               virNodeDevicePtr d,
                                               int eventID,
                                               virConnectNodeDeviceEventGenericCallback cb,
                                               long goCallbackId,
                                               virErrorPtr err);

int
virConnectDomainQemuMonitorEventRegisterHelper(virConnectPtr conn,
                                               virDomainPtr dom,
                                               const char *event,
                                               long goCallbackId,
                                               unsigned int flags,
                                               virErrorPtr err);

int
virConnectSecretEventRegisterAnyHelper(virConnectPtr c,
                                       virSecretPtr d,
                                       int eventID,
                                       virConnectSecretEventGenericCallback cb,
                                       long goCallbackId,
                                       virErrorPtr err);

int
virConnectStoragePoolEventRegisterAnyHelper(virConnectPtr c,
                                             virStoragePoolPtr d,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             long goCallbackId,
                                             virErrorPtr err);

int virStreamRecvAllHelper(virStreamPtr stream,
                           int callbackID,
                           virErrorPtr err);

int virStreamSparseRecvAllHelper(virStreamPtr stream,
                                 int callbackID,
                                 int holeCallbackID,
                                 virErrorPtr err);

int virStreamSendAllHelper(virStreamPtr stream,
                           int callbackID,
                           virErrorPtr err);

int virStreamSparseSendAllHelper(virStreamPtr stream,
                                 int callbackID,
                                 int holeCallbackID,
                                 int skipCallbackID,
                                 virErrorPtr err);

int virStreamEventAddCallbackHelper(virStreamPtr stream,
                                    int events,
                                    int callbackID,
                                    virErrorPtr err);
