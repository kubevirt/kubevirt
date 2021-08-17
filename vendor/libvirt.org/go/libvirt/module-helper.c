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
#include <stdbool.h>
#include <dlfcn.h>
#include "module-helper.h"

extern void domainEventLifecycleCallback(virConnectPtr, virDomainPtr, int, int, int);
void domainEventLifecycleCallbackHelper(virConnectPtr c, virDomainPtr d,
                                     int event, int detail, void *data)
{
    domainEventLifecycleCallback(c, d, event, detail, (int)(intptr_t)data);
}

extern void domainEventGenericCallback(virConnectPtr, virDomainPtr, int);
void domainEventGenericCallbackHelper(virConnectPtr c, virDomainPtr d, void *data)
{
    domainEventGenericCallback(c, d, (int)(intptr_t)data);
}

extern void domainEventRTCChangeCallback(virConnectPtr, virDomainPtr, long long, int);
void domainEventRTCChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                     long long utcoffset, void *data)
{
    domainEventRTCChangeCallback(c, d, utcoffset, (int)(intptr_t)data);
}

extern void domainEventWatchdogCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventWatchdogCallbackHelper(virConnectPtr c, virDomainPtr d,
                                    int action, void *data)
{
    domainEventWatchdogCallback(c, d, action, (int)(intptr_t)data);
}

extern void domainEventIOErrorCallback(virConnectPtr, virDomainPtr, const char *, const char *, int, int);
void domainEventIOErrorCallbackHelper(virConnectPtr c, virDomainPtr d,
                                   const char *srcPath, const char *devAlias,
                                   int action, void *data)
{
    domainEventIOErrorCallback(c, d, srcPath, devAlias, action, (int)(intptr_t)data);
}

extern void domainEventGraphicsCallback(virConnectPtr, virDomainPtr, int, const virDomainEventGraphicsAddress *,
                                        const virDomainEventGraphicsAddress *, const char *,
                                        const virDomainEventGraphicsSubject *, int);
void domainEventGraphicsCallbackHelper(virConnectPtr c, virDomainPtr d,
                                    int phase, const virDomainEventGraphicsAddress *local,
                                    const virDomainEventGraphicsAddress *remote,
                                    const char *authScheme,
                                    const virDomainEventGraphicsSubject *subject, void *data)
{
    domainEventGraphicsCallback(c, d, phase, local, remote, authScheme, subject, (int)(intptr_t)data);
}

extern void domainEventIOErrorReasonCallback(virConnectPtr, virDomainPtr, const char *, const char *,
                                             int, const char *, int);
void domainEventIOErrorReasonCallbackHelper(virConnectPtr c, virDomainPtr d,
                                         const char *srcPath, const char *devAlias,
                                         int action, const char *reason, void *data)
{
    domainEventIOErrorReasonCallback(c, d, srcPath, devAlias, action, reason, (int)(intptr_t)data);
}

extern void domainEventBlockJobCallback(virConnectPtr, virDomainPtr, const char *, int, int, int);
void domainEventBlockJobCallbackHelper(virConnectPtr c, virDomainPtr d,
                                    const char *disk, int type, int status, void *data)
{
    domainEventBlockJobCallback(c, d, disk, type, status, (int)(intptr_t)data);
}

extern void domainEventDiskChangeCallback(virConnectPtr, virDomainPtr, const char *, const char *,
                                          const char *, int, int);
void domainEventDiskChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                      const char *oldSrcPath, const char *newSrcPath,
                                      const char *devAlias, int reason, void *data)
{
    domainEventDiskChangeCallback(c, d, oldSrcPath, newSrcPath, devAlias, reason, (int)(intptr_t)data);
}

extern void domainEventTrayChangeCallback(virConnectPtr, virDomainPtr, const char *, int, int);
void domainEventTrayChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                      const char *devAlias, int reason, void *data)
{
    domainEventTrayChangeCallback(c, d, devAlias, reason, (int)(intptr_t)data);
}

extern void domainEventPMSuspendCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventPMSuspendCallbackHelper(virConnectPtr c, virDomainPtr d,
                                  int reason, void *data)
{
    domainEventPMSuspendCallback(c, d, reason, (int)(intptr_t)data);
}

extern void domainEventPMWakeupCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventPMWakeupCallbackHelper(virConnectPtr c, virDomainPtr d,
                                     int reason, void *data)
{
    domainEventPMWakeupCallback(c, d, reason, (int)(intptr_t)data);
}

extern void domainEventPMSuspendDiskCallback(virConnectPtr, virDomainPtr, int, int);
void domainEventPMSuspendDiskCallbackHelper(virConnectPtr c, virDomainPtr d,
                                          int reason, void *data)
{
    domainEventPMSuspendDiskCallback(c, d, reason, (int)(intptr_t)data);
}

extern void domainEventBalloonChangeCallback(virConnectPtr, virDomainPtr, unsigned long long, int);
void domainEventBalloonChangeCallbackHelper(virConnectPtr c, virDomainPtr d,
                                         unsigned long long actual, void *data)
{
    domainEventBalloonChangeCallback(c, d, actual, (int)(intptr_t)data);
}

extern void domainEventDeviceRemovedCallback(virConnectPtr, virDomainPtr, const char *, int);
void domainEventDeviceRemovedCallbackHelper(virConnectPtr c, virDomainPtr d,
                                         const char *devAlias, void *data)
{
    domainEventDeviceRemovedCallback(c, d, devAlias, (int)(intptr_t)data);
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

void eventHandleCallbackInvoke(int watch, int fd, int events, uintptr_t callback, uintptr_t opaque)
{
    ((virEventHandleCallback)callback)(watch, fd, events, (void *)opaque);
}

void eventTimeoutCallbackInvoke(int timer, uintptr_t callback, uintptr_t opaque)
{
    ((virEventTimeoutCallback)callback)(timer, (void *)opaque);
}


void eventHandleCallbackFree(uintptr_t callback, uintptr_t opaque)
{
    ((virFreeCallback)callback)((void *)opaque);
}

void eventTimeoutCallbackFree(uintptr_t callback, uintptr_t opaque)
{
    ((virFreeCallback)callback)((void *)opaque);
}

extern void networkEventLifecycleCallback(virConnectPtr, virNetworkPtr, int, int, int);
void networkEventLifecycleCallbackHelper(virConnectPtr c, virNetworkPtr d,
                                         int event, int detail, void *data)
{
    networkEventLifecycleCallback(c, d, event, detail, (int)(intptr_t)data);
}

extern void nodeDeviceEventGenericCallback(virConnectPtr, virNodeDevicePtr, int);
void nodeDeviceEventGenericCallbackHelper(virConnectPtr c, virNodeDevicePtr d, void *data)
{
    nodeDeviceEventGenericCallback(c, d, (int)(intptr_t)data);
}

extern void nodeDeviceEventLifecycleCallback(virConnectPtr, virNodeDevicePtr, int, int, int);
void nodeDeviceEventLifecycleCallbackHelper(virConnectPtr c, virNodeDevicePtr d,
                                           int event, int detail, void *data)
{
    nodeDeviceEventLifecycleCallback(c, d, event, detail, (int)(intptr_t)data);
}

extern void secretEventLifecycleCallback(virConnectPtr, virSecretPtr, int, int, int);
void secretEventLifecycleCallbackHelper(virConnectPtr c, virSecretPtr d,
                                        int event, int detail, void *data)
{
    secretEventLifecycleCallback(c, d, event, detail, (int)(intptr_t)data);
}

extern void secretEventGenericCallback(virConnectPtr, virSecretPtr, int);
void secretEventGenericCallbackHelper(virConnectPtr c, virSecretPtr d,
                                      void *data)
{
    secretEventGenericCallback(c, d, (int)(intptr_t)data);
}

extern void storagePoolEventLifecycleCallback(virConnectPtr, virStoragePoolPtr, int, int, int);
void storagePoolEventLifecycleCallbackHelper(virConnectPtr c, virStoragePoolPtr d,
                                             int event, int detail, void *data)
{
    storagePoolEventLifecycleCallback(c, d, event, detail, (int)(intptr_t)data);
}

extern void storagePoolEventGenericCallback(virConnectPtr, virStoragePoolPtr, int);
void storagePoolEventGenericCallbackHelper(virConnectPtr c, virStoragePoolPtr d,
                                           void *data)
{
    storagePoolEventGenericCallback(c, d, (int)(intptr_t)data);
}

extern int connectAuthCallback(virConnectCredentialPtr, unsigned int, int);
static int connectAuthCallbackHelper(virConnectCredentialPtr cred, unsigned int ncred, void *cbdata)
{
    int *callbackID = cbdata;

    return connectAuthCallback(cred, ncred, *callbackID);
}

extern void closeCallback(virConnectPtr, int, long);
static void closeCallbackHelper(virConnectPtr conn, int reason, void *opaque)
{
    closeCallback(conn, reason, (long)opaque);
}

extern void freeCallbackId(long);
static void freeGoCallbackHelper(void* goCallbackId) {
   freeCallbackId((long)goCallbackId);
}

extern void eventHandleCallback(int watch, int fd, int events, int callbackID);
static void eventAddHandleHelper(int watch, int fd, int events, void *opaque)
{
    eventHandleCallback(watch, fd, events, (int)(intptr_t)opaque);
}

extern void eventTimeoutCallback(int timer, int callbackID);
static void eventAddTimeoutHelper(int timer, void *opaque)
{
    eventTimeoutCallback(timer, (int)(intptr_t)opaque);
}

extern int eventAddHandleFunc(int fd, int event, uintptr_t callback, uintptr_t opaque, uintptr_t freecb);
static int eventAddHandleFuncHelper(int fd, int event, virEventHandleCallback callback, void *opaque, virFreeCallback freecb)
{
    return eventAddHandleFunc(fd, event, (uintptr_t)callback, (uintptr_t)opaque, (uintptr_t)freecb);
}

extern void eventUpdateHandleFunc(int watch, int event);
static void eventUpdateHandleFuncHelper(int watch, int event)
{
    eventUpdateHandleFunc(watch, event);
}

extern int eventRemoveHandleFunc(int watch);
static int eventRemoveHandleFuncHelper(int watch)
{
    return eventRemoveHandleFunc(watch);
}

extern int eventAddTimeoutFunc(int freq, uintptr_t callback, uintptr_t opaque, uintptr_t freecb);
static int eventAddTimeoutFuncHelper(int freq, virEventTimeoutCallback callback, void *opaque, virFreeCallback freecb)
{
    return eventAddTimeoutFunc(freq, (uintptr_t)callback, (uintptr_t)opaque, (uintptr_t)freecb);
}

extern void eventUpdateTimeoutFunc(int timer, int freq);
static void eventUpdateTimeoutFuncHelper(int timer, int freq)
{
    eventUpdateTimeoutFunc(timer, freq);
}

extern int eventRemoveTimeoutFunc(int timer);
static int eventRemoveTimeoutFuncHelper(int timer)
{
    return eventRemoveTimeoutFunc(timer);
}

extern void domainQemuMonitorEventCallback(virConnectPtr, virDomainPtr, const char *, long long, unsigned int, const char *, int);
static void domainQemuMonitorEventCallbackHelper(virConnectPtr c, virDomainPtr d,
                                       const char *event, long long secs,
                                       unsigned int micros, const char *details, void *data)
{
    domainQemuMonitorEventCallback(c, d, event, secs, micros, details, (int)(intptr_t)data);
}


struct StreamCallbackHelper {
    int callbackID;
    int holeCallbackID;
    int skipCallbackID;
};

extern int streamSourceCallback(virStreamPtr st, char *cdata, size_t nbytes, int callbackID);
static int streamSourceCallbackHelper(virStreamPtr st, char *data, size_t nbytes, void *opaque)
{
    struct StreamCallbackHelper *cbdata = opaque;
    return streamSourceCallback(st, data, nbytes, cbdata->callbackID);
}

extern int streamSourceHoleCallback(virStreamPtr st, int *inData, long long *length, int callbackID);
static int streamSourceHoleCallbackHelper(virStreamPtr st, int *inData, long long *length, void *opaque)
{
    struct StreamCallbackHelper *cbdata = opaque;
    return streamSourceHoleCallback(st, inData, length, cbdata->holeCallbackID);
}

extern int streamSourceSkipCallback(virStreamPtr st, long long length, int callbackID);
static int streamSourceSkipCallbackHelper(virStreamPtr st, long long length, void *opaque)
{
    struct StreamCallbackHelper *cbdata = opaque;
    return streamSourceSkipCallback(st, length, cbdata->skipCallbackID);
}

extern int streamSinkCallback(virStreamPtr st, const char *cdata, size_t nbytes, int callbackID);
static int streamSinkCallbackHelper(virStreamPtr st, const char *data, size_t nbytes, void *opaque)
{
    struct StreamCallbackHelper *cbdata = opaque;
    return streamSinkCallback(st, data, nbytes, cbdata->callbackID);
}

extern int streamSinkHoleCallback(virStreamPtr st, long long length, int callbackID);
static int streamSinkHoleCallbackHelper(virStreamPtr st, long long length, void *opaque)
{
    struct StreamCallbackHelper *cbdata = opaque;
    return streamSinkHoleCallback(st, length, cbdata->holeCallbackID);
}

extern void streamEventCallback(virStreamPtr st, int events, int callbackID);
static void streamEventCallbackHelper(virStreamPtr st, int events, void *opaque)
{
    streamEventCallback(st, events, (int)(intptr_t)opaque);
}

virConnectPtr
virConnectOpenAuthHelper(const char *name,
                         int *credtype,
                         unsigned int ncredtype,
                         int callbackID,
                         unsigned int flags,
                         virErrorPtr err)
{
    virConnectAuth auth = {
       .credtype = credtype,
       .ncredtype = ncredtype,
       .cb = connectAuthCallbackHelper,
       .cbdata = &callbackID,
    };

    return virConnectOpenAuthWrapper(name, &auth, flags, err);
}

virConnectPtr
virConnectOpenAuthDefaultHelper(const char *name,
                                unsigned int flags,
                                virErrorPtr err)
{
    return virConnectOpenAuthWrapper(name, virConnectAuthPtrDefaultVar, flags, err);
}

int
virConnectRegisterCloseCallbackHelper(virConnectPtr conn,
                                      long goCallbackId,
                                      virErrorPtr err)
{
    return virConnectRegisterCloseCallbackWrapper(conn,
                                                  closeCallbackHelper,
                                                  (void*)goCallbackId,
                                                  freeGoCallbackHelper,
                                                  err);
}

int
virConnectUnregisterCloseCallbackHelper(virConnectPtr conn,
                                        virErrorPtr err)
{
    return virConnectUnregisterCloseCallbackWrapper(conn, closeCallbackHelper, err);
}

int
virConnectDomainEventRegisterAnyHelper(virConnectPtr c,
                                       virDomainPtr d,
                                       int eventID,
                                       virConnectDomainEventGenericCallback cb,
                                       long goCallbackId,
                                       virErrorPtr err)
{
    return virConnectDomainEventRegisterAnyWrapper(c, d, eventID, cb,
                                                   (void*)goCallbackId,
                                                   freeGoCallbackHelper,
                                                   err);
}

int
virEventAddHandleHelper(int fd,
                        int events,
                        int callbackID,
                        virErrorPtr err)
{
    return virEventAddHandleWrapper(fd, events,
                                    eventAddHandleHelper,
                                    (void *)(intptr_t)callbackID,
                                    NULL,
                                    err);
}

int
virEventAddTimeoutHelper(int timeout,
                         int callbackID,
                         virErrorPtr err)
{
    return virEventAddTimeoutWrapper(timeout,
                                     eventAddTimeoutHelper,
                                     (void *)(intptr_t)callbackID,
                                     NULL,
                                     err);
}

void virEventRegisterImplHelper(void)
{
    virEventRegisterImplWrapper(eventAddHandleFuncHelper,
                                eventUpdateHandleFuncHelper,
                                eventRemoveHandleFuncHelper,
                                eventAddTimeoutFuncHelper,
                                eventUpdateTimeoutFuncHelper,
                                eventRemoveTimeoutFuncHelper);
}

int
virConnectNetworkEventRegisterAnyHelper(virConnectPtr c,
                                        virNetworkPtr d,
                                        int eventID,
                                        virConnectNetworkEventGenericCallback cb,
                                        long goCallbackId,
                                        virErrorPtr err)
{
    return virConnectNetworkEventRegisterAnyWrapper(c, d, eventID, cb,
                                                    (void*)goCallbackId,
                                                    freeGoCallbackHelper,
                                                    err);
}

int
virConnectNodeDeviceEventRegisterAnyHelper(virConnectPtr c,
                                           virNodeDevicePtr d,
                                           int eventID,
                                           virConnectNodeDeviceEventGenericCallback cb,
                                           long goCallbackId,
                                           virErrorPtr err)
{
    return virConnectNodeDeviceEventRegisterAnyWrapper(c, d, eventID, cb,
                                                       (void*)goCallbackId,
                                                       freeGoCallbackHelper,
                                                       err);
}

int
virConnectDomainQemuMonitorEventRegisterHelper(virConnectPtr conn,
                                               virDomainPtr dom,
                                               const char *event,
                                               long goCallbackId,
                                               unsigned int flags,
                                               virErrorPtr err)
{
    return virConnectDomainQemuMonitorEventRegisterWrapper(conn, dom, event,
                                                           domainQemuMonitorEventCallbackHelper,
                                                           (void*)goCallbackId,
                                                           freeGoCallbackHelper,
                                                           flags,
                                                           err);
}

int
virConnectSecretEventRegisterAnyHelper(virConnectPtr c,
                                       virSecretPtr d,
                                       int eventID,
                                       virConnectSecretEventGenericCallback cb,
                                       long goCallbackId,
                                       virErrorPtr err)
{
    return virConnectSecretEventRegisterAnyWrapper(c, d, eventID, cb,
                                                   (void*)goCallbackId,
                                                   freeGoCallbackHelper,
                                                   err);
}

int
virConnectStoragePoolEventRegisterAnyHelper(virConnectPtr c,
                                             virStoragePoolPtr d,
                                             int eventID,
                                             virConnectStoragePoolEventGenericCallback cb,
                                             long goCallbackId,
                                             virErrorPtr err)
{
    return virConnectStoragePoolEventRegisterAnyWrapper(c, d, eventID, cb,
                                                        (void*)goCallbackId,
                                                        freeGoCallbackHelper,
                                                        err);
}

int
virStreamRecvAllHelper(virStreamPtr stream,
                       int callbackID,
                       virErrorPtr err)
{
    struct StreamCallbackHelper cbdata = { .callbackID = callbackID };
    return virStreamRecvAllWrapper(stream, streamSinkCallbackHelper, &cbdata, err);
}

int
virStreamSparseRecvAllHelper(virStreamPtr stream,
                             int callbackID,
                             int holeCallbackID,
                             virErrorPtr err)
{
    struct StreamCallbackHelper cbdata = {
        .callbackID = callbackID,
        .holeCallbackID = holeCallbackID
    };
    return virStreamSparseRecvAllWrapper(stream,
                                         streamSinkCallbackHelper,
                                         streamSinkHoleCallbackHelper,
                                         &cbdata,
                                         err);
}

int
virStreamSendAllHelper(virStreamPtr stream,
                       int callbackID,
                       virErrorPtr err)
{
    struct StreamCallbackHelper cbdata = { .callbackID = callbackID };
    return virStreamSendAllWrapper(stream, streamSourceCallbackHelper, &cbdata, err);
}

int
virStreamSparseSendAllHelper(virStreamPtr stream,
                             int callbackID,
                             int holeCallbackID,
                             int skipCallbackID,
                             virErrorPtr err)
{
    struct StreamCallbackHelper cbdata = {
        .callbackID = callbackID,
        .holeCallbackID = holeCallbackID,
        .skipCallbackID = skipCallbackID
    };
    return virStreamSparseSendAllWrapper(stream,
                                         streamSourceCallbackHelper,
                                         streamSourceHoleCallbackHelper,
                                         streamSourceSkipCallbackHelper,
                                         &cbdata,
                                         err);
}

int
virStreamEventAddCallbackHelper(virStreamPtr stream,
                                int events,
                                int callbackID,
                                virErrorPtr err)
{
    return virStreamEventAddCallbackWrapper(stream,
                                            events,
                                            streamEventCallbackHelper,
                                            (void *)(intptr_t)callbackID,
                                            NULL,
                                            err);
}
