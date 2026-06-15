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

package libvirt

/*
#cgo !libvirt_dlopen pkg-config: libvirt
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include <stdint.h>
#include "domain_events_helper.h"
#include "callbacks_helper.h"

extern void virGoDomainEventLifecycleCallback(virConnectPtr, virDomainPtr, int, int, int);
void virGoDomainEventLifecycleCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                             int event, int detail, void *data)
{
    virGoDomainEventLifecycleCallback(conn, dom, event, detail, (int)(intptr_t)data);
}


extern void virGoDomainEventGenericCallback(virConnectPtr, virDomainPtr, int);
void virGoDomainEventGenericCallbackHelper(virConnectPtr conn, virDomainPtr dom, void *data)
{
    virGoDomainEventGenericCallback(conn, dom, (int)(intptr_t)data);
}


extern void virGoDomainEventRTCChangeCallback(virConnectPtr, virDomainPtr, long long, int);
void virGoDomainEventRTCChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                             long long utcoffset, void *data)
{
    virGoDomainEventRTCChangeCallback(conn, dom, utcoffset, (int)(intptr_t)data);
}


extern void virGoDomainEventWatchdogCallback(virConnectPtr, virDomainPtr, int, int);
void virGoDomainEventWatchdogCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                            int action, void *data)
{
    virGoDomainEventWatchdogCallback(conn, dom, action, (int)(intptr_t)data);
}


extern void virGoDomainEventIOErrorCallback(virConnectPtr, virDomainPtr, const char *, const char *, int, int);
void virGoDomainEventIOErrorCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                           const char *srcPath, const char *devAlias,
                                           int action, void *data)
{
    virGoDomainEventIOErrorCallback(conn, dom, srcPath, devAlias, action, (int)(intptr_t)data);
}


extern void virGoDomainEventGraphicsCallback(virConnectPtr, virDomainPtr, int, const virDomainEventGraphicsAddress *,
                                             const virDomainEventGraphicsAddress *, const char *,
                                             const virDomainEventGraphicsSubject *, int);
void virGoDomainEventGraphicsCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                            int phase, const virDomainEventGraphicsAddress *local,
                                            const virDomainEventGraphicsAddress *remote,
                                            const char *authScheme,
                                            const virDomainEventGraphicsSubject *subject, void *data)
{
    virGoDomainEventGraphicsCallback(conn, dom, phase, local, remote, authScheme, subject, (int)(intptr_t)data);
}


extern void virGoDomainEventIOErrorReasonCallback(virConnectPtr, virDomainPtr, const char *, const char *,
                                                  int, const char *, int);
void virGoDomainEventIOErrorReasonCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                                 const char *srcPath, const char *devAlias,
                                                 int action, const char *reason, void *data)
{
    virGoDomainEventIOErrorReasonCallback(conn, dom, srcPath, devAlias, action, reason, (int)(intptr_t)data);
}


extern void virGoDomainEventBlockJobCallback(virConnectPtr, virDomainPtr, const char *, int, int, int);
void virGoDomainEventBlockJobCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                            const char *disk, int type, int status, void *data)
{
    virGoDomainEventBlockJobCallback(conn, dom, disk, type, status, (int)(intptr_t)data);
}


extern void virGoDomainEventDiskChangeCallback(virConnectPtr, virDomainPtr, const char *, const char *,
                                               const char *, int, int);
void virGoDomainEventDiskChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                              const char *oldSrcPath, const char *newSrcPath,
                                              const char *devAlias, int reason, void *data)
{
    virGoDomainEventDiskChangeCallback(conn, dom, oldSrcPath, newSrcPath, devAlias, reason, (int)(intptr_t)data);
}


extern void virGoDomainEventTrayChangeCallback(virConnectPtr, virDomainPtr, const char *, int, int);
void virGoDomainEventTrayChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                              const char *devAlias, int reason, void *data)
{
    virGoDomainEventTrayChangeCallback(conn, dom, devAlias, reason, (int)(intptr_t)data);
}


extern void virGoDomainEventPMSuspendCallback(virConnectPtr, virDomainPtr, int, int);
void virGoDomainEventPMSuspendCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                             int reason, void *data)
{
    virGoDomainEventPMSuspendCallback(conn, dom, reason, (int)(intptr_t)data);
}


extern void virGoDomainEventPMWakeupCallback(virConnectPtr, virDomainPtr, int, int);
void virGoDomainEventPMWakeupCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                            int reason, void *data)
{
    virGoDomainEventPMWakeupCallback(conn, dom, reason, (int)(intptr_t)data);
}


extern void virGoDomainEventPMSuspendDiskCallback(virConnectPtr, virDomainPtr, int, int);
void virGoDomainEventPMSuspendDiskCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                                 int reason, void *data)
{
    virGoDomainEventPMSuspendDiskCallback(conn, dom, reason, (int)(intptr_t)data);
}


extern void virGoDomainEventBalloonChangeCallback(virConnectPtr, virDomainPtr, unsigned long long, int);
void virGoDomainEventBalloonChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                                 unsigned long long actual, void *data)
{
    virGoDomainEventBalloonChangeCallback(conn, dom, actual, (int)(intptr_t)data);
}


extern void virGoDomainEventDeviceRemovedCallback(virConnectPtr, virDomainPtr, const char *, int);
void virGoDomainEventDeviceRemovedCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                                 const char *devAlias, void *data)
{
    virGoDomainEventDeviceRemovedCallback(conn, dom, devAlias, (int)(intptr_t)data);
}


extern void virGoDomainEventTunableCallback(virConnectPtr, virDomainPtr, virTypedParameterPtr, int, int);
void virGoDomainEventTunableCallbackHelper(virConnectPtr conn,
                                           virDomainPtr dom,
                                           virTypedParameterPtr params,
                                           int nparams,
                                           void *opaque)
{
    virGoDomainEventTunableCallback(conn, dom, params, nparams, (int)(intptr_t)opaque);
}


extern void virGoDomainEventAgentLifecycleCallback(virConnectPtr, virDomainPtr, int, int, int);
void virGoDomainEventAgentLifecycleCallbackHelper(virConnectPtr conn,
                                                  virDomainPtr dom,
                                                  int state,
                                                  int reason,
                                                  void *opaque)
{
    virGoDomainEventAgentLifecycleCallback(conn, dom, state, reason, (int)(intptr_t)opaque);
}


extern void virGoDomainEventDeviceAddedCallback(virConnectPtr, virDomainPtr, const char *, int);
void virGoDomainEventDeviceAddedCallbackHelper(virConnectPtr conn,
                                               virDomainPtr dom,
                                               const char *devAlias,
                                               void *opaque)
{
    virGoDomainEventDeviceAddedCallback(conn, dom, devAlias, (int)(intptr_t)opaque);
}


extern void virGoDomainEventMigrationIterationCallback(virConnectPtr, virDomainPtr, int, int);
void virGoDomainEventMigrationIterationCallbackHelper(virConnectPtr conn,
                                                      virDomainPtr dom,
                                                      int iteration,
                                                      void *opaque)
{
    virGoDomainEventMigrationIterationCallback(conn, dom, iteration, (int)(intptr_t)opaque);
}


extern void virGoDomainEventJobCompletedCallback(virConnectPtr, virDomainPtr, virTypedParameterPtr, int, int);
void virGoDomainEventJobCompletedCallbackHelper(virConnectPtr conn,
                                                virDomainPtr dom,
                                                virTypedParameterPtr params,
                                                int nparams,
                                                void *opaque)
{
    virGoDomainEventJobCompletedCallback(conn, dom, params, nparams, (int)(intptr_t)opaque);
}


extern void virGoDomainEventDeviceRemovalFailedCallback(virConnectPtr, virDomainPtr, const char *, int);
void virGoDomainEventDeviceRemovalFailedCallbackHelper(virConnectPtr conn,
                                                       virDomainPtr dom,
                                                       const char *devAlias,
                                                       void *opaque)
{
    virGoDomainEventDeviceRemovalFailedCallback(conn, dom, devAlias, (int)(intptr_t)opaque);
}


extern void virGoDomainEventMetadataChangeCallback(virConnectPtr, virDomainPtr, int, const char *, int);
void virGoDomainEventMetadataChangeCallbackHelper(virConnectPtr conn,
                                                  virDomainPtr dom,
                                                  int type,
                                                  const char *nsuri,
                                                  void *opaque)
{
    virGoDomainEventMetadataChangeCallback(conn, dom, type, nsuri, (int)(intptr_t)opaque);
}


extern void virGoDomainEventBlockThresholdCallback(virConnectPtr, virDomainPtr, const char *, const char *, unsigned long long, unsigned long long, int);
void virGoDomainEventBlockThresholdCallbackHelper(virConnectPtr conn,
                                                  virDomainPtr dom,
                                                  const char *dev,
                                                  const char *path,
                                                  unsigned long long threshold,
                                                  unsigned long long excess,
						  void *opaque)
{
    virGoDomainEventBlockThresholdCallback(conn, dom, dev, path, threshold, excess, (int)(intptr_t)opaque);
}


extern void virGoDomainEventMemoryFailureCallback(virConnectPtr, virDomainPtr, int, int, unsigned int, int);
void virGoDomainEventMemoryFailureCallbackHelper(virConnectPtr conn,
                                                 virDomainPtr dom,
						 int recipient,
						 int action,
						 unsigned int flags,
						 void *opaque)
{
    virGoDomainEventMemoryFailureCallback(conn, dom, recipient, action, flags, (int)(intptr_t)opaque);
}


extern void virGoDomainEventMemoryDeviceSizeChangeCallback(virConnectPtr, virDomainPtr, const char *, unsigned long long, int);
void virGoDomainEventMemoryDeviceSizeChangeCallbackHelper(virConnectPtr conn,
                                                          virDomainPtr dom,
							  const char *alias,
							  unsigned long long size,
							  void *opaque)
{
    virGoDomainEventMemoryDeviceSizeChangeCallback(conn, dom, alias, size, (int)(intptr_t)opaque);
}


extern void virGoDomainEventNICMACChangeCallback(virConnectPtr, virDomainPtr, const char *, const char *, const char *, int);
void virGoDomainEventNICMACChangeCallbackHelper(virConnectPtr conn,
                                                virDomainPtr dom,
						const char *alias,
						const char *oldMAC,
						const char *newMAC,
						void *opaque)
{
    virGoDomainEventNICMACChangeCallback(conn, dom, alias, oldMAC, newMAC, (int)(intptr_t)opaque);
}

int
virConnectDomainEventRegisterAnyHelper(virConnectPtr conn,
                                       virDomainPtr dom,
                                       int eventID,
                                       virConnectDomainEventGenericCallback cb,
                                       long goCallbackId,
                                       virErrorPtr err)
{
    void *id = (void *)goCallbackId;
    return virConnectDomainEventRegisterAnyWrapper(conn, dom, eventID, cb, id,
                                                   virGoFreeCallbackHelper, err);
}


*/
import "C"
