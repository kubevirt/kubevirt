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
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

#ifndef LIBVIRT_GO_DOMAIN_EVENTS_HELPER_H__
#define LIBVIRT_GO_DOMAIN_EVENTS_HELPER_H__

#include "libvirt_generated.h"


void
domainEventLifecycleCallbackHelper(virConnectPtr conn,
                                   virDomainPtr dom,
                                   int event,
                                   int detail,
                                   void *data);


void
domainEventGenericCallbackHelper(virConnectPtr conn,
                                 virDomainPtr dom,
                                 void *data);


void
domainEventRTCChangeCallbackHelper(virConnectPtr conn,
                                   virDomainPtr dom,
                                   long long utcoffset,
                                   void *data);


void
domainEventWatchdogCallbackHelper(virConnectPtr conn,
                                  virDomainPtr dom,
                                  int action,
                                  void *data);


void
domainEventIOErrorCallbackHelper(virConnectPtr conn,
                                 virDomainPtr dom,
                                 const char *srcPath,
                                 const char *devAlias,
                                 int action,
                                 void *data);


void
domainEventGraphicsCallbackHelper(virConnectPtr conn,
                                  virDomainPtr dom,
                                  int phase,
                                  const virDomainEventGraphicsAddress *local,
                                  const virDomainEventGraphicsAddress *remote,
                                  const char *authScheme,
                                  const virDomainEventGraphicsSubject *subject,
                                  void *data);


void
domainEventIOErrorReasonCallbackHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       const char *srcPath,
                                       const char *devAlias,
                                       int action,
                                       const char *reason,
                                       void *data);


void
domainEventBlockJobCallbackHelper(virConnectPtr conn,
                                  virDomainPtr dom,
                                  const char *disk,
                                  int type,
                                  int status,
                                  void *data);


void
domainEventDiskChangeCallbackHelper(virConnectPtr conn,
                                    virDomainPtr dom,
                                    const char *oldSrcPath,
                                    const char *newSrcPath,
                                    const char *devAlias,
                                    int reason,
                                    void *data);


void
domainEventTrayChangeCallbackHelper(virConnectPtr conn,
                                    virDomainPtr dom,
                                    const char *devAlias,
                                    int reason,
                                    void *data);


void
domainEventPMSuspendCallbackHelper(virConnectPtr conn,
                                   virDomainPtr dom,
                                   int reason,
                                   void *data);


void
domainEventPMWakeupCallbackHelper(virConnectPtr conn,
                                  virDomainPtr dom,
                                  int reason,
                                  void *data);


void
domainEventPMSuspendDiskCallbackHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       int reason,
                                       void *data);


void
domainEventBalloonChangeCallbackHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       unsigned long long actual,
                                       void *data);


void
domainEventDeviceRemovedCallbackHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       const char *devAlias,
                                       void *data);


void
domainEventTunableCallbackHelper(virConnectPtr conn,
                                 virDomainPtr dom,
                                 virTypedParameterPtr params,
                                 int nparams,
                                 void *opaque);


void
domainEventAgentLifecycleCallbackHelper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        int state,
                                        int reason,
                                        void *opaque);


void
domainEventDeviceAddedCallbackHelper(virConnectPtr conn,
                                     virDomainPtr dom,
                                     const char *devAlias,
                                     void *opaque);


void
domainEventMigrationIterationCallbackHelper(virConnectPtr conn,
                                            virDomainPtr dom,
                                            int iteration,
                                            void *opaque);


void
domainEventJobCompletedCallbackHelper(virConnectPtr conn,
                                      virDomainPtr dom,
                                      virTypedParameterPtr params,
                                      int nparams,
                                      void *opaque);


void
domainEventDeviceRemovalFailedCallbackHelper(virConnectPtr conn,
                                             virDomainPtr dom,
                                             const char *devAlias,
                                             void *opaque);


void
domainEventMetadataChangeCallbackHelper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        int type,
                                        const char *nsuri,
                                        void *opaque);


void
domainEventBlockThresholdCallbackHelper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        const char *dev,
                                        const char *path,
                                        unsigned long long threshold,
                                        unsigned long long excess,
                                        void *opaque);


void
domainEventMemoryFailureCallbackHelper(virConnectPtr conn,
				       virDomainPtr dom,
				       int recipient,
				       int action,
				       unsigned int flags,
				       void *opaque);


void
domainEventMemoryDeviceSizeChangeCallbackHelper(virConnectPtr conn,
						virDomainPtr dom,
						const char *alias,
						unsigned long long size,
						void *opaque);

void
domainEventNICMACChangeCallbackHelper(virConnectPtr conn,
                                      virDomainPtr dom,
                                      const char *alias,
                                      const char *oldMAC,
                                      const char *newMAC,
                                      void *opaque);

int
virConnectDomainEventRegisterAnyHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       int eventID,
                                       virConnectDomainEventGenericCallback cb,
                                       long goCallbackId,
                                       virErrorPtr err);


#endif /* LIBVIRT_GO_DOMAIN_EVENTS_HELPER_H__ */
