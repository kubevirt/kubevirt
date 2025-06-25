//go:build !libvirt_dlopen
// +build !libvirt_dlopen

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
#cgo pkg-config: libvirt
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_generated.h"
#include "error_helper.h"


int
virConnectDomainEventDeregisterWrapper(virConnectPtr conn,
                                       virConnectDomainEventCallback cb,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virConnectDomainEventDeregister not available prior to libvirt version 0.5.0");
#else
    ret = virConnectDomainEventDeregister(conn,
                                          cb);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectDomainEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectDomainEventDeregisterAny not available prior to libvirt version 0.8.0");
#else
    ret = virConnectDomainEventDeregisterAny(conn,
                                             callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectDomainEventRegisterWrapper(virConnectPtr conn,
                                     virConnectDomainEventCallback cb,
                                     void * opaque,
                                     virFreeCallback freecb,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virConnectDomainEventRegister not available prior to libvirt version 0.5.0");
#else
    ret = virConnectDomainEventRegister(conn,
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
virConnectDomainEventRegisterAnyWrapper(virConnectPtr conn,
                                        virDomainPtr dom,
                                        int eventID,
                                        virConnectDomainEventGenericCallback cb,
                                        void * opaque,
                                        virFreeCallback freecb,
                                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virConnectDomainEventRegisterAny not available prior to libvirt version 0.8.0");
#else
    ret = virConnectDomainEventRegisterAny(conn,
                                           dom,
                                           eventID,
                                           cb,
                                           opaque,
                                           freecb);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectDomainXMLFromNativeWrapper(virConnectPtr conn,
                                     const char * nativeFormat,
                                     const char * nativeConfig,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectDomainXMLFromNative not available prior to libvirt version 0.6.4");
#else
    ret = virConnectDomainXMLFromNative(conn,
                                        nativeFormat,
                                        nativeConfig,
                                        flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virConnectDomainXMLToNativeWrapper(virConnectPtr conn,
                                   const char * nativeFormat,
                                   const char * domainXml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 4)
    setVirError(err, "Function virConnectDomainXMLToNative not available prior to libvirt version 0.6.4");
#else
    ret = virConnectDomainXMLToNative(conn,
                                      nativeFormat,
                                      domainXml,
                                      flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectGetAllDomainStatsWrapper(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr ** retStats,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virConnectGetAllDomainStats not available prior to libvirt version 1.2.8");
#else
    ret = virConnectGetAllDomainStats(conn,
                                      stats,
                                      retStats,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(1, 2, 7)
    setVirError(err, "Function virConnectGetDomainCapabilities not available prior to libvirt version 1.2.7");
#else
    ret = virConnectGetDomainCapabilities(conn,
                                          emulatorbin,
                                          arch,
                                          machine,
                                          virttype,
                                          flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListAllDomainsWrapper(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 13)
    setVirError(err, "Function virConnectListAllDomains not available prior to libvirt version 0.9.13");
#else
    ret = virConnectListAllDomains(conn,
                                   domains,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virConnectListDefinedDomains not available prior to libvirt version 0.1.1");
#else
    ret = virConnectListDefinedDomains(conn,
                                       names,
                                       maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int * ids,
                             int maxids,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectListDomains not available prior to libvirt version 0.0.3");
#else
    ret = virConnectListDomains(conn,
                                ids,
                                maxids);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 5)
    setVirError(err, "Function virConnectNumOfDefinedDomains not available prior to libvirt version 0.1.5");
#else
    ret = virConnectNumOfDefinedDomains(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virConnectNumOfDomains not available prior to libvirt version 0.0.3");
#else
    ret = virConnectNumOfDomains(conn);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainAbortJob not available prior to libvirt version 0.7.7");
#else
    ret = virDomainAbortJob(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAbortJobFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(8, 5, 0)
    setVirError(err, "Function virDomainAbortJobFlags not available prior to libvirt version 8.5.0");
#else
    ret = virDomainAbortJobFlags(domain,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 15)
    setVirError(err, "Function virDomainAddIOThread not available prior to libvirt version 1.2.15");
#else
    ret = virDomainAddIOThread(domain,
                               iothread_id,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAgentSetResponseTimeoutWrapper(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 10, 0)
    setVirError(err, "Function virDomainAgentSetResponseTimeout not available prior to libvirt version 5.10.0");
#else
    ret = virDomainAgentSetResponseTimeout(domain,
                                           timeout,
                                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 9)
    setVirError(err, "Function virDomainAttachDevice not available prior to libvirt version 0.1.9");
#else
    ret = virDomainAttachDevice(domain,
                                xml);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainAttachDeviceFlags not available prior to libvirt version 0.7.7");
#else
    ret = virDomainAttachDeviceFlags(domain,
                                     xml,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAuthorizedSSHKeysGetWrapper(virDomainPtr domain,
                                     const char * user,
                                     char *** keys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(6, 10, 0)
    setVirError(err, "Function virDomainAuthorizedSSHKeysGet not available prior to libvirt version 6.10.0");
#else
    ret = virDomainAuthorizedSSHKeysGet(domain,
                                        user,
                                        keys,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainAuthorizedSSHKeysSetWrapper(virDomainPtr domain,
                                     const char * user,
                                     const char ** keys,
                                     unsigned int nkeys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(6, 10, 0)
    setVirError(err, "Function virDomainAuthorizedSSHKeysSet not available prior to libvirt version 6.10.0");
#else
    ret = virDomainAuthorizedSSHKeysSet(domain,
                                        user,
                                        keys,
                                        nkeys,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBackupBeginWrapper(virDomainPtr domain,
                            const char * backupXML,
                            const char * checkpointXML,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(6, 0, 0)
    setVirError(err, "Function virDomainBackupBegin not available prior to libvirt version 6.0.0");
#else
    ret = virDomainBackupBegin(domain,
                               backupXML,
                               checkpointXML,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainBackupGetXMLDescWrapper(virDomainPtr domain,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(6, 0, 0)
    setVirError(err, "Function virDomainBackupGetXMLDesc not available prior to libvirt version 6.0.0");
#else
    ret = virDomainBackupGetXMLDesc(domain,
                                    flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 10, 2)
    setVirError(err, "Function virDomainBlockCommit not available prior to libvirt version 0.10.2");
#else
    ret = virDomainBlockCommit(dom,
                               disk,
                               base,
                               top,
                               bandwidth,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virDomainBlockCopy not available prior to libvirt version 1.2.8");
#else
    ret = virDomainBlockCopy(dom,
                             disk,
                             destxml,
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
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainBlockJobAbort not available prior to libvirt version 0.9.4");
#else
    ret = virDomainBlockJobAbort(dom,
                                 disk,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBlockJobSetSpeedWrapper(virDomainPtr dom,
                                 const char * disk,
                                 unsigned long bandwidth,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainBlockJobSetSpeed not available prior to libvirt version 0.9.4");
#else
    ret = virDomainBlockJobSetSpeed(dom,
                                    disk,
                                    bandwidth,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 4, 2)
    setVirError(err, "Function virDomainBlockPeek not available prior to libvirt version 0.4.2");
#else
    ret = virDomainBlockPeek(dom,
                             disk,
                             offset,
                             size,
                             buffer,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBlockPullWrapper(virDomainPtr dom,
                          const char * disk,
                          unsigned long bandwidth,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainBlockPull not available prior to libvirt version 0.9.4");
#else
    ret = virDomainBlockPull(dom,
                             disk,
                             bandwidth,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBlockRebaseWrapper(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainBlockRebase not available prior to libvirt version 0.9.10");
#else
    ret = virDomainBlockRebase(dom,
                               disk,
                               base,
                               bandwidth,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBlockResizeWrapper(virDomainPtr dom,
                            const char * disk,
                            unsigned long long size,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virDomainBlockResize not available prior to libvirt version 0.9.8");
#else
    ret = virDomainBlockResize(dom,
                               disk,
                               size,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBlockStatsWrapper(virDomainPtr dom,
                           const char * disk,
                           virDomainBlockStatsPtr stats,
                           size_t size,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 3, 2)
    setVirError(err, "Function virDomainBlockStats not available prior to libvirt version 0.3.2");
#else
    ret = virDomainBlockStats(dom,
                              disk,
                              stats,
                              size);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainBlockStatsFlagsWrapper(virDomainPtr dom,
                                const char * disk,
                                virTypedParameterPtr params,
                                int * nparams,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainBlockStatsFlags not available prior to libvirt version 0.9.5");
#else
    ret = virDomainBlockStatsFlags(dom,
                                   disk,
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
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char * to,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 9)
    setVirError(err, "Function virDomainCoreDump not available prior to libvirt version 0.1.9");
#else
    ret = virDomainCoreDump(domain,
                            to,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCoreDumpWithFormatWrapper(virDomainPtr domain,
                                   const char * to,
                                   unsigned int dumpformat,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 3)
    setVirError(err, "Function virDomainCoreDumpWithFormat not available prior to libvirt version 1.2.3");
#else
    ret = virDomainCoreDumpWithFormat(domain,
                                      to,
                                      dumpformat,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainCreate not available prior to libvirt version 0.1.1");
#else
    ret = virDomainCreate(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainCreateLinux not available prior to libvirt version 0.0.3");
#else
    ret = virDomainCreateLinux(conn,
                               xmlDesc,
                               flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCreateWithFilesWrapper(virDomainPtr domain,
                                unsigned int nfiles,
                                int * files,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 1, 1)
    setVirError(err, "Function virDomainCreateWithFiles not available prior to libvirt version 1.1.1");
#else
    ret = virDomainCreateWithFiles(domain,
                                   nfiles,
                                   files,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 2)
    setVirError(err, "Function virDomainCreateWithFlags not available prior to libvirt version 0.8.2");
#else
    ret = virDomainCreateWithFlags(domain,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(err, "Function virDomainCreateXML not available prior to libvirt version 0.5.0");
#else
    ret = virDomainCreateXML(conn,
                             xmlDesc,
                             flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainCreateXMLWithFilesWrapper(virConnectPtr conn,
                                   const char * xmlDesc,
                                   unsigned int nfiles,
                                   int * files,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(1, 1, 1)
    setVirError(err, "Function virDomainCreateXMLWithFiles not available prior to libvirt version 1.1.1");
#else
    ret = virDomainCreateXMLWithFiles(conn,
                                      xmlDesc,
                                      nfiles,
                                      files,
                                      flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainDefineXML not available prior to libvirt version 0.1.1");
#else
    ret = virDomainDefineXML(conn,
                             xml);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(1, 2, 12)
    setVirError(err, "Function virDomainDefineXMLFlags not available prior to libvirt version 1.2.12");
#else
    ret = virDomainDefineXMLFlags(conn,
                                  xml,
                                  flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 15)
    setVirError(err, "Function virDomainDelIOThread not available prior to libvirt version 1.2.15");
#else
    ret = virDomainDelIOThread(domain,
                               iothread_id,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDelThrottleGroupWrapper(virDomainPtr dom,
                                 const char * group,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(11, 2, 0)
    setVirError(err, "Function virDomainDelThrottleGroup not available prior to libvirt version 11.2.0");
#else
    ret = virDomainDelThrottleGroup(dom,
                                    group,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainDestroy not available prior to libvirt version 0.0.3");
#else
    ret = virDomainDestroy(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainDestroyFlags not available prior to libvirt version 0.9.4");
#else
    ret = virDomainDestroyFlags(domain,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 9)
    setVirError(err, "Function virDomainDetachDevice not available prior to libvirt version 0.1.9");
#else
    ret = virDomainDetachDevice(domain,
                                xml);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 4, 0)
    setVirError(err, "Function virDomainDetachDeviceAlias not available prior to libvirt version 4.4.0");
#else
    ret = virDomainDetachDeviceAlias(domain,
                                     alias,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainDetachDeviceFlags not available prior to libvirt version 0.7.7");
#else
    ret = virDomainDetachDeviceFlags(domain,
                                     xml,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainFDAssociateWrapper(virDomainPtr domain,
                            const char * name,
                            unsigned int nfds,
                            int * fds,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(9, 0, 0)
    setVirError(err, "Function virDomainFDAssociate not available prior to libvirt version 9.0.0");
#else
    ret = virDomainFDAssociate(domain,
                               name,
                               nfds,
                               fds,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainFSFreezeWrapper(virDomainPtr dom,
                         const char ** mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainFSFreeze not available prior to libvirt version 1.2.5");
#else
    ret = virDomainFSFreeze(dom,
                            mountpoints,
                            nmountpoints,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info)
{

#if !LIBVIR_CHECK_VERSION(1, 2, 11)
    setVirError(NULL, "Function virDomainFSInfoFree not available prior to libvirt version 1.2.11");
#else
    virDomainFSInfoFree(info);
#endif
    return;
}

int
virDomainFSThawWrapper(virDomainPtr dom,
                       const char ** mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainFSThaw not available prior to libvirt version 1.2.5");
#else
    ret = virDomainFSThaw(dom,
                          mountpoints,
                          nmountpoints,
                          flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainFSTrimWrapper(virDomainPtr dom,
                       const char * mountPoint,
                       unsigned long long minimum,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 1)
    setVirError(err, "Function virDomainFSTrim not available prior to libvirt version 1.0.1");
#else
    ret = virDomainFSTrim(dom,
                          mountPoint,
                          minimum,
                          flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainFree not available prior to libvirt version 0.0.3");
#else
    ret = virDomainFree(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int * autostart,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virDomainGetAutostart not available prior to libvirt version 0.2.1");
#else
    ret = virDomainGetAutostart(domain,
                                autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetAutostartOnceWrapper(virDomainPtr domain,
                                 int * autostart,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(11, 2, 0)
    setVirError(err, "Function virDomainGetAutostartOnce not available prior to libvirt version 11.2.0");
#else
    ret = virDomainGetAutostartOnce(domain,
                                    autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainGetBlkioParameters not available prior to libvirt version 0.9.0");
#else
    ret = virDomainGetBlkioParameters(domain,
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
virDomainGetBlockInfoWrapper(virDomainPtr domain,
                             const char * disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 1)
    setVirError(err, "Function virDomainGetBlockInfo not available prior to libvirt version 0.8.1");
#else
    ret = virDomainGetBlockInfo(domain,
                                disk,
                                info,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int * nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virDomainGetBlockIoTune not available prior to libvirt version 0.9.8");
#else
    ret = virDomainGetBlockIoTune(dom,
                                  disk,
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
virDomainGetBlockJobInfoWrapper(virDomainPtr dom,
                                const char * disk,
                                virDomainBlockJobInfoPtr info,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainGetBlockJobInfo not available prior to libvirt version 0.9.4");
#else
    ret = virDomainGetBlockJobInfo(dom,
                                   disk,
                                   info,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainGetCPUStats not available prior to libvirt version 0.9.10");
#else
    ret = virDomainGetCPUStats(domain,
                               params,
                               nparams,
                               start_cpu,
                               ncpus,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 3, 0)
    setVirError(err, "Function virDomainGetConnect not available prior to libvirt version 0.3.0");
#else
    ret = virDomainGetConnect(dom);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetControlInfoWrapper(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainGetControlInfo not available prior to libvirt version 0.9.3");
#else
    ret = virDomainGetControlInfo(domain,
                                  info,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetDiskErrorsWrapper(virDomainPtr dom,
                              virDomainDiskErrorPtr errors,
                              unsigned int maxerrors,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainGetDiskErrors not available prior to libvirt version 0.9.10");
#else
    ret = virDomainGetDiskErrors(dom,
                                 errors,
                                 maxerrors,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetEmulatorPinInfoWrapper(virDomainPtr domain,
                                   unsigned char * cpumap,
                                   int maplen,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainGetEmulatorPinInfo not available prior to libvirt version 0.10.0");
#else
    ret = virDomainGetEmulatorPinInfo(domain,
                                      cpumap,
                                      maplen,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 11)
    setVirError(err, "Function virDomainGetFSInfo not available prior to libvirt version 1.2.11");
#else
    ret = virDomainGetFSInfo(dom,
                             info,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetGuestInfoWrapper(virDomainPtr domain,
                             unsigned int types,
                             virTypedParameterPtr * params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(5, 7, 0)
    setVirError(err, "Function virDomainGetGuestInfo not available prior to libvirt version 5.7.0");
#else
    ret = virDomainGetGuestInfo(domain,
                                types,
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
virDomainGetGuestVcpusWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              unsigned int * nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virDomainGetGuestVcpus not available prior to libvirt version 2.0.0");
#else
    ret = virDomainGetGuestVcpus(domain,
                                 params,
                                 nparams,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainGetHostname not available prior to libvirt version 0.10.0");
#else
    ret = virDomainGetHostname(domain,
                               flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err)
{
    unsigned int ret = 0;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetID not available prior to libvirt version 0.0.3");
#else
    ret = virDomainGetID(domain);
    if (ret == 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetIOThreadInfoWrapper(virDomainPtr dom,
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(err, "Function virDomainGetIOThreadInfo not available prior to libvirt version 1.2.14");
#else
    ret = virDomainGetIOThreadInfo(dom,
                                   info,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetInfo not available prior to libvirt version 0.0.3");
#else
    ret = virDomainGetInfo(domain,
                           info);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainGetInterfaceParameters not available prior to libvirt version 0.9.9");
#else
    ret = virDomainGetInterfaceParameters(domain,
                                          device,
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
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 7)
    setVirError(err, "Function virDomainGetJobInfo not available prior to libvirt version 0.7.7");
#else
    ret = virDomainGetJobInfo(domain,
                              info);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetJobStatsWrapper(virDomainPtr domain,
                            int * type,
                            virTypedParameterPtr * params,
                            int * nparams,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virDomainGetJobStats not available prior to libvirt version 1.0.3");
#else
    ret = virDomainGetJobStats(domain,
                               type,
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
virDomainGetLaunchSecurityInfoWrapper(virDomainPtr domain,
                                      virTypedParameterPtr * params,
                                      int * nparams,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virDomainGetLaunchSecurityInfo not available prior to libvirt version 4.5.0");
#else
    ret = virDomainGetLaunchSecurityInfo(domain,
                                         params,
                                         nparams,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err)
{
    unsigned long ret = 0;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetMaxMemory not available prior to libvirt version 0.0.3");
#else
    ret = virDomainGetMaxMemory(domain);
    if (ret == 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virDomainGetMaxVcpus not available prior to libvirt version 0.2.1");
#else
    ret = virDomainGetMaxVcpus(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int * nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainGetMemoryParameters not available prior to libvirt version 0.8.5");
#else
    ret = virDomainGetMemoryParameters(domain,
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
virDomainGetMessagesWrapper(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 1, 0)
    setVirError(err, "Function virDomainGetMessages not available prior to libvirt version 7.1.0");
#else
    ret = virDomainGetMessages(domain,
                               msgs,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainGetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char * uri,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainGetMetadata not available prior to libvirt version 0.9.10");
#else
    ret = virDomainGetMetadata(domain,
                               type,
                               uri,
                               flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetName not available prior to libvirt version 0.0.3");
#else
    ret = virDomainGetName(domain);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainGetNumaParameters not available prior to libvirt version 0.9.9");
#else
    ret = virDomainGetNumaParameters(domain,
                                     params,
                                     nparams,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetOSType not available prior to libvirt version 0.0.3");
#else
    ret = virDomainGetOSType(domain);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              int * nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 3, 3)
    setVirError(err, "Function virDomainGetPerfEvents not available prior to libvirt version 1.3.3");
#else
    ret = virDomainGetPerfEvents(domain,
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
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 3)
    setVirError(err, "Function virDomainGetSchedulerParameters not available prior to libvirt version 0.2.3");
#else
    ret = virDomainGetSchedulerParameters(domain,
                                          params,
                                          nparams);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int * nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainGetSchedulerParametersFlags not available prior to libvirt version 0.9.2");
#else
    ret = virDomainGetSchedulerParametersFlags(domain,
                                               params,
                                               nparams,
                                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int * nparams,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 2, 3)
    setVirError(err, "Function virDomainGetSchedulerType not available prior to libvirt version 0.2.3");
#else
    ret = virDomainGetSchedulerType(domain,
                                    nparams);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virDomainGetSecurityLabel not available prior to libvirt version 0.6.1");
#else
    ret = virDomainGetSecurityLabel(domain,
                                    seclabel);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainGetSecurityLabelList not available prior to libvirt version 0.10.0");
#else
    ret = virDomainGetSecurityLabelList(domain,
                                        seclabels);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetStateWrapper(virDomainPtr domain,
                         int * state,
                         int * reason,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainGetState not available prior to libvirt version 0.9.2");
#else
    ret = virDomainGetState(domain,
                            state,
                            reason,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetTimeWrapper(virDomainPtr dom,
                        long long * seconds,
                        unsigned int * nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainGetTime not available prior to libvirt version 1.2.5");
#else
    ret = virDomainGetTime(dom,
                           seconds,
                           nseconds,
                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 5)
    setVirError(err, "Function virDomainGetUUID not available prior to libvirt version 0.0.5");
#else
    ret = virDomainGetUUID(domain,
                           uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char * buf,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainGetUUIDString not available prior to libvirt version 0.1.1");
#else
    ret = virDomainGetUUIDString(domain,
                                 buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetVcpuPinInfoWrapper(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char * cpumaps,
                               int maplen,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainGetVcpuPinInfo not available prior to libvirt version 0.9.3");
#else
    ret = virDomainGetVcpuPinInfo(domain,
                                  ncpumaps,
                                  cpumaps,
                                  maplen,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetVcpusWrapper(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char * cpumaps,
                         int maplen,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 4)
    setVirError(err, "Function virDomainGetVcpus not available prior to libvirt version 0.1.4");
#else
    ret = virDomainGetVcpus(domain,
                            info,
                            maxinfo,
                            cpumaps,
                            maplen);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainGetVcpusFlags not available prior to libvirt version 0.8.5");
#else
    ret = virDomainGetVcpusFlags(domain,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainGetXMLDesc not available prior to libvirt version 0.0.3");
#else
    ret = virDomainGetXMLDesc(domain,
                              flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainGraphicsReloadWrapper(virDomainPtr domain,
                               unsigned int type,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(10, 2, 0)
    setVirError(err, "Function virDomainGraphicsReload not available prior to libvirt version 10.2.0");
#else
    ret = virDomainGraphicsReload(domain,
                                  type,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainHasManagedSaveImage not available prior to libvirt version 0.8.0");
#else
    ret = virDomainHasManagedSaveImage(dom,
                                       flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info)
{

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(NULL, "Function virDomainIOThreadInfoFree not available prior to libvirt version 1.2.14");
#else
    virDomainIOThreadInfoFree(info);
#endif
    return;
}

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainInjectNMI not available prior to libvirt version 0.9.2");
#else
    ret = virDomainInjectNMI(domain,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainInterfaceAddressesWrapper(virDomainPtr dom,
                                   virDomainInterfacePtr ** ifaces,
                                   unsigned int source,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(err, "Function virDomainInterfaceAddresses not available prior to libvirt version 1.2.14");
#else
    ret = virDomainInterfaceAddresses(dom,
                                      ifaces,
                                      source,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface)
{

#if !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(NULL, "Function virDomainInterfaceFree not available prior to libvirt version 1.2.14");
#else
    virDomainInterfaceFree(iface);
#endif
    return;
}

int
virDomainInterfaceStatsWrapper(virDomainPtr dom,
                               const char * device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 3, 2)
    setVirError(err, "Function virDomainInterfaceStats not available prior to libvirt version 0.3.2");
#else
    ret = virDomainInterfaceStats(dom,
                                  device,
                                  stats,
                                  size);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virDomainIsActive not available prior to libvirt version 0.7.3");
#else
    ret = virDomainIsActive(dom);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 3)
    setVirError(err, "Function virDomainIsPersistent not available prior to libvirt version 0.7.3");
#else
    ret = virDomainIsPersistent(dom);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 6)
    setVirError(err, "Function virDomainIsUpdated not available prior to libvirt version 0.8.6");
#else
    ret = virDomainIsUpdated(dom);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainListGetStatsWrapper(virDomainPtr * doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr ** retStats,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virDomainListGetStats not available prior to libvirt version 1.2.8");
#else
    ret = virDomainListGetStats(doms,
                                stats,
                                retStats,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainLookupByID not available prior to libvirt version 0.0.3");
#else
    ret = virDomainLookupByID(conn,
                              id);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char * name,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainLookupByName not available prior to libvirt version 0.0.3");
#else
    ret = virDomainLookupByName(conn,
                                name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 0, 5)
    setVirError(err, "Function virDomainLookupByUUID not available prior to libvirt version 0.0.5");
#else
    ret = virDomainLookupByUUID(conn,
                                uuid);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainLookupByUUIDString not available prior to libvirt version 0.1.1");
#else
    ret = virDomainLookupByUUIDString(conn,
                                      uuidstr);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainManagedSave not available prior to libvirt version 0.8.0");
#else
    ret = virDomainManagedSave(dom,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char * dxml,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 7, 0)
    setVirError(err, "Function virDomainManagedSaveDefineXML not available prior to libvirt version 3.7.0");
#else
    ret = virDomainManagedSaveDefineXML(domain,
                                        dxml,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(3, 7, 0)
    setVirError(err, "Function virDomainManagedSaveGetXMLDesc not available prior to libvirt version 3.7.0");
#else
    ret = virDomainManagedSaveGetXMLDesc(domain,
                                         flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainManagedSaveRemove not available prior to libvirt version 0.8.0");
#else
    ret = virDomainManagedSaveRemove(dom,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMemoryPeekWrapper(virDomainPtr dom,
                           unsigned long long start,
                           size_t size,
                           void * buffer,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 4, 2)
    setVirError(err, "Function virDomainMemoryPeek not available prior to libvirt version 0.4.2");
#else
    ret = virDomainMemoryPeek(dom,
                              start,
                              size,
                              buffer,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMemoryStatsWrapper(virDomainPtr dom,
                            virDomainMemoryStatPtr stats,
                            unsigned int nr_stats,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 5)
    setVirError(err, "Function virDomainMemoryStats not available prior to libvirt version 0.7.5");
#else
    ret = virDomainMemoryStats(dom,
                               stats,
                               nr_stats,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 3, 2)
    setVirError(err, "Function virDomainMigrate not available prior to libvirt version 0.3.2");
#else
    ret = virDomainMigrate(domain,
                           dconn,
                           flags,
                           dname,
                           uri,
                           bandwidth);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainMigrate2 not available prior to libvirt version 0.9.2");
#else
    ret = virDomainMigrate2(domain,
                            dconn,
                            dxml,
                            flags,
                            dname,
                            uri,
                            bandwidth);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainMigrate3Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         virTypedParameterPtr params,
                         unsigned int nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(1, 1, 0)
    setVirError(err, "Function virDomainMigrate3 not available prior to libvirt version 1.1.0");
#else
    ret = virDomainMigrate3(domain,
                            dconn,
                            params,
                            nparams,
                            flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateGetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long * cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virDomainMigrateGetCompressionCache not available prior to libvirt version 1.0.3");
#else
    ret = virDomainMigrateGetCompressionCache(domain,
                                              cacheSize,
                                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 7, 0)
    setVirError(err, "Function virDomainMigrateGetMaxDowntime not available prior to libvirt version 3.7.0");
#else
    ret = virDomainMigrateGetMaxDowntime(domain,
                                         downtime,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long * bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 5)
    setVirError(err, "Function virDomainMigrateGetMaxSpeed not available prior to libvirt version 0.9.5");
#else
    ret = virDomainMigrateGetMaxSpeed(domain,
                                      bandwidth,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateSetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 3)
    setVirError(err, "Function virDomainMigrateSetCompressionCache not available prior to libvirt version 1.0.3");
#else
    ret = virDomainMigrateSetCompressionCache(domain,
                                              cacheSize,
                                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateSetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainMigrateSetMaxDowntime not available prior to libvirt version 0.8.0");
#else
    ret = virDomainMigrateSetMaxDowntime(domain,
                                         downtime,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateSetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainMigrateSetMaxSpeed not available prior to libvirt version 0.9.0");
#else
    ret = virDomainMigrateSetMaxSpeed(domain,
                                      bandwidth,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 3, 3)
    setVirError(err, "Function virDomainMigrateStartPostCopy not available prior to libvirt version 1.3.3");
#else
    ret = virDomainMigrateStartPostCopy(domain,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateToURIWrapper(virDomainPtr domain,
                             const char * duri,
                             unsigned long flags,
                             const char * dname,
                             unsigned long bandwidth,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virDomainMigrateToURI not available prior to libvirt version 0.7.2");
#else
    ret = virDomainMigrateToURI(domain,
                                duri,
                                flags,
                                dname,
                                bandwidth);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainMigrateToURI2 not available prior to libvirt version 0.9.2");
#else
    ret = virDomainMigrateToURI2(domain,
                                 dconnuri,
                                 miguri,
                                 dxml,
                                 flags,
                                 dname,
                                 bandwidth);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainMigrateToURI3Wrapper(virDomainPtr domain,
                              const char * dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 1, 0)
    setVirError(err, "Function virDomainMigrateToURI3 not available prior to libvirt version 1.1.0");
#else
    ret = virDomainMigrateToURI3(domain,
                                 dconnuri,
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
virDomainOpenChannelWrapper(virDomainPtr dom,
                            const char * name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virDomainOpenChannel not available prior to libvirt version 1.0.2");
#else
    ret = virDomainOpenChannel(dom,
                               name,
                               st,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainOpenConsoleWrapper(virDomainPtr dom,
                            const char * dev_name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 6)
    setVirError(err, "Function virDomainOpenConsole not available prior to libvirt version 0.8.6");
#else
    ret = virDomainOpenConsole(dom,
                               dev_name,
                               st,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainOpenGraphicsWrapper(virDomainPtr dom,
                             unsigned int idx,
                             int fd,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainOpenGraphics not available prior to libvirt version 0.9.7");
#else
    ret = virDomainOpenGraphics(dom,
                                idx,
                                fd,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainOpenGraphicsFDWrapper(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(err, "Function virDomainOpenGraphicsFD not available prior to libvirt version 1.2.8");
#else
    ret = virDomainOpenGraphicsFD(dom,
                                  idx,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainPMSuspendForDurationWrapper(virDomainPtr dom,
                                     unsigned int target,
                                     unsigned long long duration,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainPMSuspendForDuration not available prior to libvirt version 0.9.10");
#else
    ret = virDomainPMSuspendForDuration(dom,
                                        target,
                                        duration,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 11)
    setVirError(err, "Function virDomainPMWakeup not available prior to libvirt version 0.9.11");
#else
    ret = virDomainPMWakeup(dom,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainPinEmulatorWrapper(virDomainPtr domain,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainPinEmulator not available prior to libvirt version 0.10.0");
#else
    ret = virDomainPinEmulator(domain,
                               cpumap,
                               maplen,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainPinIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 14)
    setVirError(err, "Function virDomainPinIOThread not available prior to libvirt version 1.2.14");
#else
    ret = virDomainPinIOThread(domain,
                               iothread_id,
                               cpumap,
                               maplen,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainPinVcpuWrapper(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char * cpumap,
                        int maplen,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 4)
    setVirError(err, "Function virDomainPinVcpu not available prior to libvirt version 0.1.4");
#else
    ret = virDomainPinVcpu(domain,
                           vcpu,
                           cpumap,
                           maplen);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainPinVcpuFlagsWrapper(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char * cpumap,
                             int maplen,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainPinVcpuFlags not available prior to libvirt version 0.9.3");
#else
    ret = virDomainPinVcpuFlags(domain,
                                vcpu,
                                cpumap,
                                maplen,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virDomainReboot not available prior to libvirt version 0.1.0");
#else
    ret = virDomainReboot(domain,
                          flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 6, 0)
    setVirError(err, "Function virDomainRef not available prior to libvirt version 0.6.0");
#else
    ret = virDomainRef(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRenameWrapper(virDomainPtr dom,
                       const char * new_name,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 19)
    setVirError(err, "Function virDomainRename not available prior to libvirt version 1.2.19");
#else
    ret = virDomainRename(dom,
                          new_name,
                          flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 7)
    setVirError(err, "Function virDomainReset not available prior to libvirt version 0.9.7");
#else
    ret = virDomainReset(domain,
                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char * from,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainRestore not available prior to libvirt version 0.0.3");
#else
    ret = virDomainRestore(conn,
                           from);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRestoreFlagsWrapper(virConnectPtr conn,
                             const char * from,
                             const char * dxml,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainRestoreFlags not available prior to libvirt version 0.9.4");
#else
    ret = virDomainRestoreFlags(conn,
                                from,
                                dxml,
                                flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainRestoreParamsWrapper(virConnectPtr conn,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(8, 4, 0)
    setVirError(err, "Function virDomainRestoreParams not available prior to libvirt version 8.4.0");
#else
    ret = virDomainRestoreParams(conn,
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
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainResume not available prior to libvirt version 0.0.3");
#else
    ret = virDomainResume(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char * to,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainSave not available prior to libvirt version 0.0.3");
#else
    ret = virDomainSave(domain,
                        to);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSaveFlagsWrapper(virDomainPtr domain,
                          const char * to,
                          const char * dxml,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainSaveFlags not available prior to libvirt version 0.9.4");
#else
    ret = virDomainSaveFlags(domain,
                             to,
                             dxml,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSaveImageDefineXMLWrapper(virConnectPtr conn,
                                   const char * file,
                                   const char * dxml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainSaveImageDefineXML not available prior to libvirt version 0.9.4");
#else
    ret = virDomainSaveImageDefineXML(conn,
                                      file,
                                      dxml,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainSaveImageGetXMLDesc not available prior to libvirt version 0.9.4");
#else
    ret = virDomainSaveImageGetXMLDesc(conn,
                                       file,
                                       flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSaveParamsWrapper(virDomainPtr domain,
                           virTypedParameterPtr params,
                           int nparams,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(8, 4, 0)
    setVirError(err, "Function virDomainSaveParams not available prior to libvirt version 8.4.0");
#else
    ret = virDomainSaveParams(domain,
                              params,
                              nparams,
                              flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainScreenshotWrapper(virDomainPtr domain,
                           virStreamPtr stream,
                           unsigned int screen,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainScreenshot not available prior to libvirt version 0.9.2");
#else
    ret = virDomainScreenshot(domain,
                              stream,
                              screen,
                              flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virDomainSendKey not available prior to libvirt version 0.9.3");
#else
    ret = virDomainSendKey(domain,
                           codeset,
                           holdtime,
                           keycodes,
                           nkeycodes,
                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSendProcessSignalWrapper(virDomainPtr domain,
                                  long long pid_value,
                                  unsigned int signum,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 1)
    setVirError(err, "Function virDomainSendProcessSignal not available prior to libvirt version 1.0.1");
#else
    ret = virDomainSendProcessSignal(domain,
                                     pid_value,
                                     signum,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 1)
    setVirError(err, "Function virDomainSetAutostart not available prior to libvirt version 0.2.1");
#else
    ret = virDomainSetAutostart(domain,
                                autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetAutostartOnceWrapper(virDomainPtr domain,
                                 int autostart,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(11, 2, 0)
    setVirError(err, "Function virDomainSetAutostartOnce not available prior to libvirt version 11.2.0");
#else
    ret = virDomainSetAutostartOnce(domain,
                                    autostart);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainSetBlkioParameters not available prior to libvirt version 0.9.0");
#else
    ret = virDomainSetBlkioParameters(domain,
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
virDomainSetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 8)
    setVirError(err, "Function virDomainSetBlockIoTune not available prior to libvirt version 0.9.8");
#else
    ret = virDomainSetBlockIoTune(dom,
                                  disk,
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
virDomainSetBlockThresholdWrapper(virDomainPtr domain,
                                  const char * dev,
                                  unsigned long long threshold,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 1, 0)
    setVirError(err, "Function virDomainSetBlockThreshold not available prior to libvirt version 3.1.0");
#else
    ret = virDomainSetBlockThreshold(domain,
                                     dev,
                                     threshold,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetGuestVcpusWrapper(virDomainPtr domain,
                              const char * cpumap,
                              int state,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virDomainSetGuestVcpus not available prior to libvirt version 2.0.0");
#else
    ret = virDomainSetGuestVcpus(domain,
                                 cpumap,
                                 state,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetIOThreadParamsWrapper(virDomainPtr domain,
                                  unsigned int iothread_id,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 10, 0)
    setVirError(err, "Function virDomainSetIOThreadParams not available prior to libvirt version 4.10.0");
#else
    ret = virDomainSetIOThreadParams(domain,
                                     iothread_id,
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
virDomainSetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainSetInterfaceParameters not available prior to libvirt version 0.9.9");
#else
    ret = virDomainSetInterfaceParameters(domain,
                                          device,
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
virDomainSetLaunchSecurityStateWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(8, 0, 0)
    setVirError(err, "Function virDomainSetLaunchSecurityState not available prior to libvirt version 8.0.0");
#else
    ret = virDomainSetLaunchSecurityState(domain,
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
virDomainSetLifecycleActionWrapper(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int action,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 9, 0)
    setVirError(err, "Function virDomainSetLifecycleAction not available prior to libvirt version 3.9.0");
#else
    ret = virDomainSetLifecycleAction(domain,
                                      type,
                                      action,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainSetMaxMemory not available prior to libvirt version 0.0.3");
#else
    ret = virDomainSetMaxMemory(domain,
                                memory);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainSetMemory not available prior to libvirt version 0.1.1");
#else
    ret = virDomainSetMemory(domain,
                             memory);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetMemoryFlagsWrapper(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virDomainSetMemoryFlags not available prior to libvirt version 0.9.0");
#else
    ret = virDomainSetMemoryFlags(domain,
                                  memory,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainSetMemoryParameters not available prior to libvirt version 0.8.5");
#else
    ret = virDomainSetMemoryParameters(domain,
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
virDomainSetMemoryStatsPeriodWrapper(virDomainPtr domain,
                                     int period,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 1, 1)
    setVirError(err, "Function virDomainSetMemoryStatsPeriod not available prior to libvirt version 1.1.1");
#else
    ret = virDomainSetMemoryStatsPeriod(domain,
                                        period,
                                        flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainSetMetadata not available prior to libvirt version 0.9.10");
#else
    ret = virDomainSetMetadata(domain,
                               type,
                               metadata,
                               key,
                               uri,
                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 9)
    setVirError(err, "Function virDomainSetNumaParameters not available prior to libvirt version 0.9.9");
#else
    ret = virDomainSetNumaParameters(domain,
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
virDomainSetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 3, 3)
    setVirError(err, "Function virDomainSetPerfEvents not available prior to libvirt version 1.3.3");
#else
    ret = virDomainSetPerfEvents(domain,
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
virDomainSetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 2, 3)
    setVirError(err, "Function virDomainSetSchedulerParameters not available prior to libvirt version 0.2.3");
#else
    ret = virDomainSetSchedulerParameters(domain,
                                          params,
                                          nparams);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 2)
    setVirError(err, "Function virDomainSetSchedulerParametersFlags not available prior to libvirt version 0.9.2");
#else
    ret = virDomainSetSchedulerParametersFlags(domain,
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
virDomainSetThrottleGroupWrapper(virDomainPtr dom,
                                 const char * group,
                                 virTypedParameterPtr params,
                                 int nparams,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(11, 2, 0)
    setVirError(err, "Function virDomainSetThrottleGroup not available prior to libvirt version 11.2.0");
#else
    ret = virDomainSetThrottleGroup(dom,
                                    group,
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
virDomainSetTimeWrapper(virDomainPtr dom,
                        long long seconds,
                        unsigned int nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 5)
    setVirError(err, "Function virDomainSetTime not available prior to libvirt version 1.2.5");
#else
    ret = virDomainSetTime(dom,
                           seconds,
                           nseconds,
                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetUserPasswordWrapper(virDomainPtr dom,
                                const char * user,
                                const char * password,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 16)
    setVirError(err, "Function virDomainSetUserPassword not available prior to libvirt version 1.2.16");
#else
    ret = virDomainSetUserPassword(dom,
                                   user,
                                   password,
                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetVcpuWrapper(virDomainPtr domain,
                        const char * vcpumap,
                        int state,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 1, 0)
    setVirError(err, "Function virDomainSetVcpu not available prior to libvirt version 3.1.0");
#else
    ret = virDomainSetVcpu(domain,
                           vcpumap,
                           state,
                           flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 4)
    setVirError(err, "Function virDomainSetVcpus not available prior to libvirt version 0.1.4");
#else
    ret = virDomainSetVcpus(domain,
                            nvcpus);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainSetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 5)
    setVirError(err, "Function virDomainSetVcpusFlags not available prior to libvirt version 0.8.5");
#else
    ret = virDomainSetVcpusFlags(domain,
                                 nvcpus,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainShutdown not available prior to libvirt version 0.0.3");
#else
    ret = virDomainShutdown(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 10)
    setVirError(err, "Function virDomainShutdownFlags not available prior to libvirt version 0.9.10");
#else
    ret = virDomainShutdownFlags(domain,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainStartDirtyRateCalcWrapper(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(7, 2, 0)
    setVirError(err, "Function virDomainStartDirtyRateCalc not available prior to libvirt version 7.2.0");
#else
    ret = virDomainStartDirtyRateCalc(domain,
                                      seconds,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats)
{

#if !LIBVIR_CHECK_VERSION(1, 2, 8)
    setVirError(NULL, "Function virDomainStatsRecordListFree not available prior to libvirt version 1.2.8");
#else
    virDomainStatsRecordListFree(stats);
#endif
    return;
}

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 0, 3)
    setVirError(err, "Function virDomainSuspend not available prior to libvirt version 0.0.3");
#else
    ret = virDomainSuspend(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 1)
    setVirError(err, "Function virDomainUndefine not available prior to libvirt version 0.1.1");
#else
    ret = virDomainUndefine(domain);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainUndefineFlags not available prior to libvirt version 0.9.4");
#else
    ret = virDomainUndefineFlags(domain,
                                 flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainUpdateDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 0)
    setVirError(err, "Function virDomainUpdateDeviceFlags not available prior to libvirt version 0.8.0");
#else
    ret = virDomainUpdateDeviceFlags(domain,
                                     xml,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
