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

extern void domainEventLifecycleCallback(virConnectPtr, virDomainPtr, int, int, int);
void domainEventLifecycleCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                     int event, int detail, void *data)
{
    domainEventLifecycleCallback(conn, dom, event, detail, (int)(intptr_t)data);
}


extern void domainEventGenericCallback(virConnectPtr, virDomainPtr, int);
void domainEventGenericCallbackHelper(virConnectPtr conn, virDomainPtr dom, void *data)
{
    domainEventGenericCallback(conn, dom, (int)(intptr_t)data);
}


extern void domainEventRTCChangeCallback(virConnectPtr, virDomainPtr, long long, int);
void domainEventRTCChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                     long long utcoffset, void *data)
{
    domainEventRTCChangeCallback(conn, dom, utcoffset, (int)(intptr_t)data);
}


extern void domainEventWatchdogCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventWatchdogCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                    int action, void *data)
{
    domainEventWatchdogCallback(conn, dom, action, (int)(intptr_t)data);
}


extern void domainEventIOErrorCallback(virConnectPtr, virDomainPtr, const char *, const char *, int, int);
void domainEventIOErrorCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                   const char *srcPath, const char *devAlias,
                                   int action, void *data)
{
    domainEventIOErrorCallback(conn, dom, srcPath, devAlias, action, (int)(intptr_t)data);
}


extern void domainEventGraphicsCallback(virConnectPtr, virDomainPtr, int, const virDomainEventGraphicsAddress *,
                                        const virDomainEventGraphicsAddress *, const char *,
                                        const virDomainEventGraphicsSubject *, int);
void domainEventGraphicsCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                    int phase, const virDomainEventGraphicsAddress *local,
                                    const virDomainEventGraphicsAddress *remote,
                                    const char *authScheme,
                                    const virDomainEventGraphicsSubject *subject, void *data)
{
    domainEventGraphicsCallback(conn, dom, phase, local, remote, authScheme, subject, (int)(intptr_t)data);
}


extern void domainEventIOErrorReasonCallback(virConnectPtr, virDomainPtr, const char *, const char *,
                                             int, const char *, int);
void domainEventIOErrorReasonCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                         const char *srcPath, const char *devAlias,
                                         int action, const char *reason, void *data)
{
    domainEventIOErrorReasonCallback(conn, dom, srcPath, devAlias, action, reason, (int)(intptr_t)data);
}


extern void domainEventBlockJobCallback(virConnectPtr, virDomainPtr, const char *, int, int, int);
void domainEventBlockJobCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                    const char *disk, int type, int status, void *data)
{
    domainEventBlockJobCallback(conn, dom, disk, type, status, (int)(intptr_t)data);
}


extern void domainEventDiskChangeCallback(virConnectPtr, virDomainPtr, const char *, const char *,
                                          const char *, int, int);
void domainEventDiskChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                      const char *oldSrcPath, const char *newSrcPath,
                                      const char *devAlias, int reason, void *data)
{
    domainEventDiskChangeCallback(conn, dom, oldSrcPath, newSrcPath, devAlias, reason, (int)(intptr_t)data);
}


extern void domainEventTrayChangeCallback(virConnectPtr, virDomainPtr, const char *, int, int);
void domainEventTrayChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                      const char *devAlias, int reason, void *data)
{
    domainEventTrayChangeCallback(conn, dom, devAlias, reason, (int)(intptr_t)data);
}


extern void domainEventPMSuspendCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventPMSuspendCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                  int reason, void *data)
{
    domainEventPMSuspendCallback(conn, dom, reason, (int)(intptr_t)data);
}


extern void domainEventPMWakeupCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventPMWakeupCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                     int reason, void *data)
{
    domainEventPMWakeupCallback(conn, dom, reason, (int)(intptr_t)data);
}


extern void domainEventPMSuspendDiskCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventPMSuspendDiskCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                          int reason, void *data)
{
    domainEventPMSuspendDiskCallback(conn, dom, reason, (int)(intptr_t)data);
}


extern void domainEventBalloonChangeCallback(virConnectPtr, virDomainPtr, unsigned long long, int);
void domainEventBalloonChangeCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                         unsigned long long actual, void *data)
{
    domainEventBalloonChangeCallback(conn, dom, actual, (int)(intptr_t)data);
}


extern void domainEventDeviceRemovedCallback(virConnectPtr, virDomainPtr, const char *, int);
void domainEventDeviceRemovedCallbackHelper(virConnectPtr conn, virDomainPtr dom,
                                         const char *devAlias, void *data)
{
    domainEventDeviceRemovedCallback(conn, dom, devAlias, (int)(intptr_t)data);
}


extern void domainEventTunableCallback(virConnectPtr, virDomainPtr, virTypedParameterPtr, int, int);
void domainEventTunableCallbackHelper(virConnectPtr conn,
				    virDomainPtr dom,
				    virTypedParameterPtr params,
				    int nparams,
				    void *opaque)
{
    domainEventTunableCallback(conn, dom, params, nparams, (int)(intptr_t)opaque);
}


extern void domainEventAgentLifecycleCallback(virConnectPtr, virDomainPtr, int, int, int);
void domainEventAgentLifecycleCallbackHelper(virConnectPtr conn,
					   virDomainPtr dom,
					   int state,
					   int reason,
					   void *opaque)
{
    domainEventAgentLifecycleCallback(conn, dom, state, reason, (int)(intptr_t)opaque);
}


extern void domainEventDeviceAddedCallback(virConnectPtr, virDomainPtr, const char *, int);
void domainEventDeviceAddedCallbackHelper(virConnectPtr conn,
					virDomainPtr dom,
					const char *devAlias,
					void *opaque)
{
    domainEventDeviceAddedCallback(conn, dom, devAlias, (int)(intptr_t)opaque);
}


extern void domainEventMigrationIterationCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventMigrationIterationCallbackHelper(virConnectPtr conn,
					       virDomainPtr dom,
					       int iteration,
					       void *opaque)
{
    domainEventMigrationIterationCallback(conn, dom, iteration, (int)(intptr_t)opaque);
}


extern void domainEventJobCompletedCallback(virConnectPtr, virDomainPtr, virTypedParameterPtr, int, int);
void domainEventJobCompletedCallbackHelper(virConnectPtr conn,
					 virDomainPtr dom,
					 virTypedParameterPtr params,
					 int nparams,
					 void *opaque)
{
    domainEventJobCompletedCallback(conn, dom, params, nparams, (int)(intptr_t)opaque);
}


extern void domainEventDeviceRemovalFailedCallback(virConnectPtr, virDomainPtr, const char *, int);
void domainEventDeviceRemovalFailedCallbackHelper(virConnectPtr conn,
						virDomainPtr dom,
						const char *devAlias,
						void *opaque)
{
    domainEventDeviceRemovalFailedCallback(conn, dom, devAlias, (int)(intptr_t)opaque);
}


extern void domainEventMetadataChangeCallback(virConnectPtr, virDomainPtr, int, const char *, int);
void domainEventMetadataChangeCallbackHelper(virConnectPtr conn,
					   virDomainPtr dom,
					   int type,
					   const char *nsuri,
					   void *opaque)
{
    domainEventMetadataChangeCallback(conn, dom, type, nsuri, (int)(intptr_t)opaque);
}


extern void domainEventBlockThresholdCallback(virConnectPtr, virDomainPtr, const char *, const char *, unsigned long long, unsigned long long, int);
void domainEventBlockThresholdCallbackHelper(virConnectPtr conn,
					   virDomainPtr dom,
					   const char *dev,
					   const char *path,
					   unsigned long long threshold,
					   unsigned long long excess,
					   void *opaque)
{
    domainEventBlockThresholdCallback(conn, dom, dev, path, threshold, excess, (int)(intptr_t)opaque);
}


extern void domainEventMemoryFailureCallback(virConnectPtr, virDomainPtr, int, int, unsigned int, int);
void domainEventMemoryFailureCallbackHelper(virConnectPtr conn,
                                            virDomainPtr dom,
                                            int recipient,
                                            int action,
                                            unsigned int flags,
                                            void *opaque)
{
    domainEventMemoryFailureCallback(conn, dom, recipient, action, flags, (int)(intptr_t)opaque);
}


extern void domainEventMemoryDeviceSizeChangeCallback(virConnectPtr, virDomainPtr, const char *, unsigned long long, int);
void domainEventMemoryDeviceSizeChangeCallbackHelper(virConnectPtr conn,
						virDomainPtr dom,
						const char *alias,
						unsigned long long size,
						void *opaque)
{
  domainEventMemoryDeviceSizeChangeCallback(conn, dom, alias, size, (int)(intptr_t)opaque);
}


extern void domainEventNICMACChangeCallback(virConnectPtr, virDomainPtr, const char *, const char *, const char *, int);
void domainEventNICMACChangeCallbackHelper(virConnectPtr conn,
                                           virDomainPtr dom,
                                           const char *alias,
                                           const char *oldMAC,
                                           const char *newMAC,
void *opaque)
{
	domainEventNICMACChangeCallback(conn, dom, alias, oldMAC, newMAC, (int)(intptr_t)opaque);
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
                                                   freeGoCallbackHelper, err);
}


*/
import "C"
