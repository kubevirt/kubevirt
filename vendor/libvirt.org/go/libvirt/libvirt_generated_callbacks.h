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

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef void (*virFreeCallback)(void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef void (*virEventTimeoutCallback)(int timer,
                                        void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef void (*virEventHandleCallback)(int watch,
                                       int fd,
                                       int events,
                                       void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 4, 1)
typedef int (*virConnectAuthCallbackPtr)(virConnectCredentialPtr cred,
                                         unsigned int ncred,
                                         void * cbdata);
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 0)
typedef void (*virConnectCloseFunc)(virConnectPtr conn,
                                    int reason,
                                    void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
typedef void (*virConnectDomainEventAgentLifecycleCallback)(virConnectPtr conn,
                                                            virDomainPtr dom,
                                                            int state,
                                                            int reason,
                                                            void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 10, 0)
typedef void (*virConnectDomainEventBalloonChangeCallback)(virConnectPtr conn,
                                                           virDomainPtr dom,
                                                           unsigned long long actual,
                                                           void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 4)
typedef void (*virConnectDomainEventBlockJobCallback)(virConnectPtr conn,
                                                      virDomainPtr dom,
                                                      const char * disk,
                                                      int type,
                                                      int status,
                                                      void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 2, 0)
typedef void (*virConnectDomainEventBlockThresholdCallback)(virConnectPtr conn,
                                                            virDomainPtr dom,
                                                            const char * dev,
                                                            const char * path,
                                                            unsigned long long threshold,
                                                            unsigned long long excess,
                                                            void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int (*virConnectDomainEventCallback)(virConnectPtr conn,
                                             virDomainPtr dom,
                                             int event,
                                             int detail,
                                             void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 15)
typedef void (*virConnectDomainEventDeviceAddedCallback)(virConnectPtr conn,
                                                         virDomainPtr dom,
                                                         const char * devAlias,
                                                         void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 4)
typedef void (*virConnectDomainEventDeviceRemovalFailedCallback)(virConnectPtr conn,
                                                                 virDomainPtr dom,
                                                                 const char * devAlias,
                                                                 void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 1, 1)
typedef void (*virConnectDomainEventDeviceRemovedCallback)(virConnectPtr conn,
                                                           virDomainPtr dom,
                                                           const char * devAlias,
                                                           void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 7)
typedef void (*virConnectDomainEventDiskChangeCallback)(virConnectPtr conn,
                                                        virDomainPtr dom,
                                                        const char * oldSrcPath,
                                                        const char * newSrcPath,
                                                        const char * devAlias,
                                                        int reason,
                                                        void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef void (*virConnectDomainEventGenericCallback)(virConnectPtr conn,
                                                     virDomainPtr dom,
                                                     void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef void (*virConnectDomainEventGraphicsCallback)(virConnectPtr conn,
                                                      virDomainPtr dom,
                                                      int phase,
                                                      const virDomainEventGraphicsAddress * local,
                                                      const virDomainEventGraphicsAddress * remote,
                                                      const char * authScheme,
                                                      const virDomainEventGraphicsSubject * subject,
                                                      void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef void (*virConnectDomainEventIOErrorCallback)(virConnectPtr conn,
                                                     virDomainPtr dom,
                                                     const char * srcPath,
                                                     const char * devAlias,
                                                     int action,
                                                     void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 1)
typedef void (*virConnectDomainEventIOErrorReasonCallback)(virConnectPtr conn,
                                                           virDomainPtr dom,
                                                           const char * srcPath,
                                                           const char * devAlias,
                                                           int action,
                                                           const char * reason,
                                                           void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 3)
typedef void (*virConnectDomainEventJobCompletedCallback)(virConnectPtr conn,
                                                          virDomainPtr dom,
                                                          virTypedParameterPtr params,
                                                          int nparams,
                                                          void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(7, 9, 0)
typedef void (*virConnectDomainEventMemoryDeviceSizeChangeCallback)(virConnectPtr conn,
                                                                    virDomainPtr dom,
                                                                    const char * alias,
                                                                    unsigned long long size,
                                                                    void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(6, 9, 0)
typedef void (*virConnectDomainEventMemoryFailureCallback)(virConnectPtr conn,
                                                           virDomainPtr dom,
                                                           int recipient,
                                                           int action,
                                                           unsigned int flags,
                                                           void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
typedef void (*virConnectDomainEventMetadataChangeCallback)(virConnectPtr conn,
                                                            virDomainPtr dom,
                                                            int type,
                                                            const char * nsuri,
                                                            void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 3, 2)
typedef void (*virConnectDomainEventMigrationIterationCallback)(virConnectPtr conn,
                                                                virDomainPtr dom,
                                                                int iteration,
                                                                void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(11, 2, 0)
typedef void (*virConnectDomainEventNICMACChangeCallback)(virConnectPtr conn,
                                                          virDomainPtr dom,
                                                          const char * alias,
                                                          const char * oldMAC,
                                                          const char * newMAC,
                                                          void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef void (*virConnectDomainEventPMSuspendCallback)(virConnectPtr conn,
                                                       virDomainPtr dom,
                                                       int reason,
                                                       void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 0, 0)
typedef void (*virConnectDomainEventPMSuspendDiskCallback)(virConnectPtr conn,
                                                           virDomainPtr dom,
                                                           int reason,
                                                           void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef void (*virConnectDomainEventPMWakeupCallback)(virConnectPtr conn,
                                                      virDomainPtr dom,
                                                      int reason,
                                                      void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef void (*virConnectDomainEventRTCChangeCallback)(virConnectPtr conn,
                                                       virDomainPtr dom,
                                                       long long utcoffset,
                                                       void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 9, 11)
typedef void (*virConnectDomainEventTrayChangeCallback)(virConnectPtr conn,
                                                        virDomainPtr dom,
                                                        const char * devAlias,
                                                        int reason,
                                                        void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 9)
typedef void (*virConnectDomainEventTunableCallback)(virConnectPtr conn,
                                                     virDomainPtr dom,
                                                     virTypedParameterPtr params,
                                                     int nparams,
                                                     void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 8, 0)
typedef void (*virConnectDomainEventWatchdogCallback)(virConnectPtr conn,
                                                      virDomainPtr dom,
                                                      int action,
                                                      void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 1)
typedef void (*virConnectNetworkEventGenericCallback)(virConnectPtr conn,
                                                      virNetworkPtr net,
                                                      void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(1, 2, 1)
typedef void (*virConnectNetworkEventLifecycleCallback)(virConnectPtr conn,
                                                        virNetworkPtr net,
                                                        int event,
                                                        int detail,
                                                        void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(9, 8, 0)
typedef void (*virConnectNetworkEventMetadataChangeCallback)(virConnectPtr conn,
                                                             virNetworkPtr net,
                                                             int type,
                                                             const char * nsuri,
                                                             void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
typedef void (*virConnectNodeDeviceEventGenericCallback)(virConnectPtr conn,
                                                         virNodeDevicePtr dev,
                                                         void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(2, 2, 0)
typedef void (*virConnectNodeDeviceEventLifecycleCallback)(virConnectPtr conn,
                                                           virNodeDevicePtr dev,
                                                           int event,
                                                           int detail,
                                                           void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
typedef void (*virConnectSecretEventGenericCallback)(virConnectPtr conn,
                                                     virSecretPtr secret,
                                                     void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 0, 0)
typedef void (*virConnectSecretEventLifecycleCallback)(virConnectPtr conn,
                                                       virSecretPtr secret,
                                                       int event,
                                                       int detail,
                                                       void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
typedef void (*virConnectStoragePoolEventGenericCallback)(virConnectPtr conn,
                                                          virStoragePoolPtr pool,
                                                          void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(2, 0, 0)
typedef void (*virConnectStoragePoolEventLifecycleCallback)(virConnectPtr conn,
                                                            virStoragePoolPtr pool,
                                                            int event,
                                                            int detail,
                                                            void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
typedef void (*virErrorFunc)(void * userData,
                             virErrorPtr error);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int (*virEventAddHandleFunc)(int fd,
                                     int event,
                                     virEventHandleCallback cb,
                                     void * opaque,
                                     virFreeCallback ff);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int (*virEventAddTimeoutFunc)(int timeout,
                                      virEventTimeoutCallback cb,
                                      void * opaque,
                                      virFreeCallback ff);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int (*virEventRemoveHandleFunc)(int watch);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef int (*virEventRemoveTimeoutFunc)(int timer);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef void (*virEventUpdateHandleFunc)(int watch,
                                         int event);
#endif

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
typedef void (*virEventUpdateTimeoutFunc)(int timer,
                                          int timeout);
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef void (*virStreamEventCallback)(virStreamPtr stream,
                                       int events,
                                       void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef int (*virStreamSinkFunc)(virStreamPtr st,
                                 const char * data,
                                 size_t nbytes,
                                 void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 4, 0)
typedef int (*virStreamSinkHoleFunc)(virStreamPtr st,
                                     long long length,
                                     void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(0, 7, 2)
typedef int (*virStreamSourceFunc)(virStreamPtr st,
                                   char * data,
                                   size_t nbytes,
                                   void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 4, 0)
typedef int (*virStreamSourceHoleFunc)(virStreamPtr st,
                                       int * inData,
                                       long long * length,
                                       void * opaque);
#endif

#if !LIBVIR_CHECK_VERSION(3, 4, 0)
typedef int (*virStreamSourceSkipFunc)(virStreamPtr st,
                                       long long length,
                                       void * opaque);
#endif
