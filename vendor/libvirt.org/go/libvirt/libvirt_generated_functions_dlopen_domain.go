//go:build libvirt_dlopen
// +build libvirt_dlopen

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
#include "libvirt_generated_dlopen.h"
#include "error_helper.h"


typedef int
(*virConnectDomainEventDeregisterFuncType)(virConnectPtr conn,
                                           virConnectDomainEventCallback cb);

int
virConnectDomainEventDeregisterWrapper(virConnectPtr conn,
                                       virConnectDomainEventCallback cb,
                                       virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainEventDeregisterFuncType virConnectDomainEventDeregisterSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectDomainEventDeregister",
                       (void**)&virConnectDomainEventDeregisterSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectDomainEventDeregisterSymbol(conn,
                                                cb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainEventDeregisterAnyFuncType)(virConnectPtr conn,
                                              int callbackID);

int
virConnectDomainEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainEventDeregisterAnyFuncType virConnectDomainEventDeregisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectDomainEventDeregisterAny",
                       (void**)&virConnectDomainEventDeregisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectDomainEventDeregisterAnySymbol(conn,
                                                   callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainEventRegisterFuncType)(virConnectPtr conn,
                                         virConnectDomainEventCallback cb,
                                         void * opaque,
                                         virFreeCallback freecb);

int
virConnectDomainEventRegisterWrapper(virConnectPtr conn,
                                     virConnectDomainEventCallback cb,
                                     void * opaque,
                                     virFreeCallback freecb,
                                     virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainEventRegisterFuncType virConnectDomainEventRegisterSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectDomainEventRegister",
                       (void**)&virConnectDomainEventRegisterSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectDomainEventRegisterSymbol(conn,
                                              cb,
                                              opaque,
                                              freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainEventRegisterAnyFuncType)(virConnectPtr conn,
                                            virDomainPtr dom,
                                            int eventID,
                                            virConnectDomainEventGenericCallback cb,
                                            void * opaque,
                                            virFreeCallback freecb);

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
    static virConnectDomainEventRegisterAnyFuncType virConnectDomainEventRegisterAnySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectDomainEventRegisterAny",
                       (void**)&virConnectDomainEventRegisterAnySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectDomainEventRegisterAnySymbol(conn,
                                                 dom,
                                                 eventID,
                                                 cb,
                                                 opaque,
                                                 freecb);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectDomainXMLFromNativeFuncType)(virConnectPtr conn,
                                         const char * nativeFormat,
                                         const char * nativeConfig,
                                         unsigned int flags);

char *
virConnectDomainXMLFromNativeWrapper(virConnectPtr conn,
                                     const char * nativeFormat,
                                     const char * nativeConfig,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    char * ret = NULL;
    static virConnectDomainXMLFromNativeFuncType virConnectDomainXMLFromNativeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectDomainXMLFromNative",
                       (void**)&virConnectDomainXMLFromNativeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectDomainXMLFromNativeSymbol(conn,
                                              nativeFormat,
                                              nativeConfig,
                                              flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectDomainXMLToNativeFuncType)(virConnectPtr conn,
                                       const char * nativeFormat,
                                       const char * domainXml,
                                       unsigned int flags);

char *
virConnectDomainXMLToNativeWrapper(virConnectPtr conn,
                                   const char * nativeFormat,
                                   const char * domainXml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    char * ret = NULL;
    static virConnectDomainXMLToNativeFuncType virConnectDomainXMLToNativeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectDomainXMLToNative",
                       (void**)&virConnectDomainXMLToNativeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectDomainXMLToNativeSymbol(conn,
                                            nativeFormat,
                                            domainXml,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectGetAllDomainStatsFuncType)(virConnectPtr conn,
                                       unsigned int stats,
                                       virDomainStatsRecordPtr ** retStats,
                                       unsigned int flags);

int
virConnectGetAllDomainStatsWrapper(virConnectPtr conn,
                                   unsigned int stats,
                                   virDomainStatsRecordPtr ** retStats,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virConnectGetAllDomainStatsFuncType virConnectGetAllDomainStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetAllDomainStats",
                       (void**)&virConnectGetAllDomainStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetAllDomainStatsSymbol(conn,
                                            stats,
                                            retStats,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virConnectGetDomainCapabilitiesFuncType)(virConnectPtr conn,
                                           const char * emulatorbin,
                                           const char * arch,
                                           const char * machine,
                                           const char * virttype,
                                           unsigned int flags);

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
    static virConnectGetDomainCapabilitiesFuncType virConnectGetDomainCapabilitiesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectGetDomainCapabilities",
                       (void**)&virConnectGetDomainCapabilitiesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectGetDomainCapabilitiesSymbol(conn,
                                                emulatorbin,
                                                arch,
                                                machine,
                                                virttype,
                                                flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListAllDomainsFuncType)(virConnectPtr conn,
                                    virDomainPtr ** domains,
                                    unsigned int flags);

int
virConnectListAllDomainsWrapper(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllDomainsFuncType virConnectListAllDomainsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListAllDomains",
                       (void**)&virConnectListAllDomainsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListAllDomainsSymbol(conn,
                                         domains,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDefinedDomainsFuncType)(virConnectPtr conn,
                                        char ** const names,
                                        int maxnames);

int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames,
                                    virErrorPtr err)
{
    int ret = -1;
    static virConnectListDefinedDomainsFuncType virConnectListDefinedDomainsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListDefinedDomains",
                       (void**)&virConnectListDefinedDomainsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListDefinedDomainsSymbol(conn,
                                             names,
                                             maxnames);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectListDomainsFuncType)(virConnectPtr conn,
                                 int * ids,
                                 int maxids);

int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int * ids,
                             int maxids,
                             virErrorPtr err)
{
    int ret = -1;
    static virConnectListDomainsFuncType virConnectListDomainsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectListDomains",
                       (void**)&virConnectListDomainsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectListDomainsSymbol(conn,
                                      ids,
                                      maxids);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDefinedDomainsFuncType)(virConnectPtr conn);

int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDefinedDomainsFuncType virConnectNumOfDefinedDomainsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfDefinedDomains",
                       (void**)&virConnectNumOfDefinedDomainsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfDefinedDomainsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectNumOfDomainsFuncType)(virConnectPtr conn);

int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDomainsFuncType virConnectNumOfDomainsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnectNumOfDomains",
                       (void**)&virConnectNumOfDomainsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnectNumOfDomainsSymbol(conn);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAbortJobFuncType)(virDomainPtr domain);

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainAbortJobFuncType virDomainAbortJobSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAbortJob",
                       (void**)&virDomainAbortJobSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAbortJobSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAbortJobFlagsFuncType)(virDomainPtr domain,
                                  unsigned int flags);

int
virDomainAbortJobFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainAbortJobFlagsFuncType virDomainAbortJobFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAbortJobFlags",
                       (void**)&virDomainAbortJobFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAbortJobFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAddIOThreadFuncType)(virDomainPtr domain,
                                unsigned int iothread_id,
                                unsigned int flags);

int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainAddIOThreadFuncType virDomainAddIOThreadSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAddIOThread",
                       (void**)&virDomainAddIOThreadSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAddIOThreadSymbol(domain,
                                     iothread_id,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAgentSetResponseTimeoutFuncType)(virDomainPtr domain,
                                            int timeout,
                                            unsigned int flags);

int
virDomainAgentSetResponseTimeoutWrapper(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = -1;
    static virDomainAgentSetResponseTimeoutFuncType virDomainAgentSetResponseTimeoutSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAgentSetResponseTimeout",
                       (void**)&virDomainAgentSetResponseTimeoutSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAgentSetResponseTimeoutSymbol(domain,
                                                 timeout,
                                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAttachDeviceFuncType)(virDomainPtr domain,
                                 const char * xml);

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainAttachDeviceFuncType virDomainAttachDeviceSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAttachDevice",
                       (void**)&virDomainAttachDeviceSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAttachDeviceSymbol(domain,
                                      xml);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAttachDeviceFlagsFuncType)(virDomainPtr domain,
                                      const char * xml,
                                      unsigned int flags);

int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainAttachDeviceFlagsFuncType virDomainAttachDeviceFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAttachDeviceFlags",
                       (void**)&virDomainAttachDeviceFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAttachDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAuthorizedSSHKeysGetFuncType)(virDomainPtr domain,
                                         const char * user,
                                         char *** keys,
                                         unsigned int flags);

int
virDomainAuthorizedSSHKeysGetWrapper(virDomainPtr domain,
                                     const char * user,
                                     char *** keys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainAuthorizedSSHKeysGetFuncType virDomainAuthorizedSSHKeysGetSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAuthorizedSSHKeysGet",
                       (void**)&virDomainAuthorizedSSHKeysGetSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAuthorizedSSHKeysGetSymbol(domain,
                                              user,
                                              keys,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainAuthorizedSSHKeysSetFuncType)(virDomainPtr domain,
                                         const char * user,
                                         const char ** keys,
                                         unsigned int nkeys,
                                         unsigned int flags);

int
virDomainAuthorizedSSHKeysSetWrapper(virDomainPtr domain,
                                     const char * user,
                                     const char ** keys,
                                     unsigned int nkeys,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainAuthorizedSSHKeysSetFuncType virDomainAuthorizedSSHKeysSetSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainAuthorizedSSHKeysSet",
                       (void**)&virDomainAuthorizedSSHKeysSetSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainAuthorizedSSHKeysSetSymbol(domain,
                                              user,
                                              keys,
                                              nkeys,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBackupBeginFuncType)(virDomainPtr domain,
                                const char * backupXML,
                                const char * checkpointXML,
                                unsigned int flags);

int
virDomainBackupBeginWrapper(virDomainPtr domain,
                            const char * backupXML,
                            const char * checkpointXML,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainBackupBeginFuncType virDomainBackupBeginSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBackupBegin",
                       (void**)&virDomainBackupBeginSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBackupBeginSymbol(domain,
                                     backupXML,
                                     checkpointXML,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainBackupGetXMLDescFuncType)(virDomainPtr domain,
                                     unsigned int flags);

char *
virDomainBackupGetXMLDescWrapper(virDomainPtr domain,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = NULL;
    static virDomainBackupGetXMLDescFuncType virDomainBackupGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBackupGetXMLDesc",
                       (void**)&virDomainBackupGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBackupGetXMLDescSymbol(domain,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockCommitFuncType)(virDomainPtr dom,
                                const char * disk,
                                const char * base,
                                const char * top,
                                unsigned long bandwidth,
                                unsigned int flags);

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
    static virDomainBlockCommitFuncType virDomainBlockCommitSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockCommit",
                       (void**)&virDomainBlockCommitSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockCommitSymbol(dom,
                                     disk,
                                     base,
                                     top,
                                     bandwidth,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockCopyFuncType)(virDomainPtr dom,
                              const char * disk,
                              const char * destxml,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags);

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
    static virDomainBlockCopyFuncType virDomainBlockCopySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockCopy",
                       (void**)&virDomainBlockCopySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockCopySymbol(dom,
                                   disk,
                                   destxml,
                                   params,
                                   nparams,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockJobAbortFuncType)(virDomainPtr dom,
                                  const char * disk,
                                  unsigned int flags);

int
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockJobAbortFuncType virDomainBlockJobAbortSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockJobAbort",
                       (void**)&virDomainBlockJobAbortSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockJobAbortSymbol(dom,
                                       disk,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockJobSetSpeedFuncType)(virDomainPtr dom,
                                     const char * disk,
                                     unsigned long bandwidth,
                                     unsigned int flags);

int
virDomainBlockJobSetSpeedWrapper(virDomainPtr dom,
                                 const char * disk,
                                 unsigned long bandwidth,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockJobSetSpeedFuncType virDomainBlockJobSetSpeedSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockJobSetSpeed",
                       (void**)&virDomainBlockJobSetSpeedSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockJobSetSpeedSymbol(dom,
                                          disk,
                                          bandwidth,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockPeekFuncType)(virDomainPtr dom,
                              const char * disk,
                              unsigned long long offset,
                              size_t size,
                              void * buffer,
                              unsigned int flags);

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
    static virDomainBlockPeekFuncType virDomainBlockPeekSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockPeek",
                       (void**)&virDomainBlockPeekSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockPeekSymbol(dom,
                                   disk,
                                   offset,
                                   size,
                                   buffer,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockPullFuncType)(virDomainPtr dom,
                              const char * disk,
                              unsigned long bandwidth,
                              unsigned int flags);

int
virDomainBlockPullWrapper(virDomainPtr dom,
                          const char * disk,
                          unsigned long bandwidth,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockPullFuncType virDomainBlockPullSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockPull",
                       (void**)&virDomainBlockPullSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockPullSymbol(dom,
                                   disk,
                                   bandwidth,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockRebaseFuncType)(virDomainPtr dom,
                                const char * disk,
                                const char * base,
                                unsigned long bandwidth,
                                unsigned int flags);

int
virDomainBlockRebaseWrapper(virDomainPtr dom,
                            const char * disk,
                            const char * base,
                            unsigned long bandwidth,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockRebaseFuncType virDomainBlockRebaseSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockRebase",
                       (void**)&virDomainBlockRebaseSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockRebaseSymbol(dom,
                                     disk,
                                     base,
                                     bandwidth,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockResizeFuncType)(virDomainPtr dom,
                                const char * disk,
                                unsigned long long size,
                                unsigned int flags);

int
virDomainBlockResizeWrapper(virDomainPtr dom,
                            const char * disk,
                            unsigned long long size,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockResizeFuncType virDomainBlockResizeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockResize",
                       (void**)&virDomainBlockResizeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockResizeSymbol(dom,
                                     disk,
                                     size,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockStatsFuncType)(virDomainPtr dom,
                               const char * disk,
                               virDomainBlockStatsPtr stats,
                               size_t size);

int
virDomainBlockStatsWrapper(virDomainPtr dom,
                           const char * disk,
                           virDomainBlockStatsPtr stats,
                           size_t size,
                           virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockStatsFuncType virDomainBlockStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockStats",
                       (void**)&virDomainBlockStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockStatsSymbol(dom,
                                    disk,
                                    stats,
                                    size);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainBlockStatsFlagsFuncType)(virDomainPtr dom,
                                    const char * disk,
                                    virTypedParameterPtr params,
                                    int * nparams,
                                    unsigned int flags);

int
virDomainBlockStatsFlagsWrapper(virDomainPtr dom,
                                const char * disk,
                                virTypedParameterPtr params,
                                int * nparams,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockStatsFlagsFuncType virDomainBlockStatsFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainBlockStatsFlags",
                       (void**)&virDomainBlockStatsFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainBlockStatsFlagsSymbol(dom,
                                         disk,
                                         params,
                                         nparams,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCoreDumpFuncType)(virDomainPtr domain,
                             const char * to,
                             unsigned int flags);

int
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char * to,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainCoreDumpFuncType virDomainCoreDumpSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCoreDump",
                       (void**)&virDomainCoreDumpSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCoreDumpSymbol(domain,
                                  to,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCoreDumpWithFormatFuncType)(virDomainPtr domain,
                                       const char * to,
                                       unsigned int dumpformat,
                                       unsigned int flags);

int
virDomainCoreDumpWithFormatWrapper(virDomainPtr domain,
                                   const char * to,
                                   unsigned int dumpformat,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainCoreDumpWithFormatFuncType virDomainCoreDumpWithFormatSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCoreDumpWithFormat",
                       (void**)&virDomainCoreDumpWithFormatSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCoreDumpWithFormatSymbol(domain,
                                            to,
                                            dumpformat,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCreateFuncType)(virDomainPtr domain);

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainCreateFuncType virDomainCreateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCreate",
                       (void**)&virDomainCreateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCreateSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCreateLinuxFuncType)(virConnectPtr conn,
                                const char * xmlDesc,
                                unsigned int flags);

virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCreateLinuxFuncType virDomainCreateLinuxSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCreateLinux",
                       (void**)&virDomainCreateLinuxSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCreateLinuxSymbol(conn,
                                     xmlDesc,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCreateWithFilesFuncType)(virDomainPtr domain,
                                    unsigned int nfiles,
                                    int * files,
                                    unsigned int flags);

int
virDomainCreateWithFilesWrapper(virDomainPtr domain,
                                unsigned int nfiles,
                                int * files,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainCreateWithFilesFuncType virDomainCreateWithFilesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCreateWithFiles",
                       (void**)&virDomainCreateWithFilesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCreateWithFilesSymbol(domain,
                                         nfiles,
                                         files,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainCreateWithFlagsFuncType)(virDomainPtr domain,
                                    unsigned int flags);

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainCreateWithFlagsFuncType virDomainCreateWithFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCreateWithFlags",
                       (void**)&virDomainCreateWithFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCreateWithFlagsSymbol(domain,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCreateXMLFuncType)(virConnectPtr conn,
                              const char * xmlDesc,
                              unsigned int flags);

virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCreateXMLFuncType virDomainCreateXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCreateXML",
                       (void**)&virDomainCreateXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCreateXMLSymbol(conn,
                                   xmlDesc,
                                   flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainCreateXMLWithFilesFuncType)(virConnectPtr conn,
                                       const char * xmlDesc,
                                       unsigned int nfiles,
                                       int * files,
                                       unsigned int flags);

virDomainPtr
virDomainCreateXMLWithFilesWrapper(virConnectPtr conn,
                                   const char * xmlDesc,
                                   unsigned int nfiles,
                                   int * files,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCreateXMLWithFilesFuncType virDomainCreateXMLWithFilesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainCreateXMLWithFiles",
                       (void**)&virDomainCreateXMLWithFilesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainCreateXMLWithFilesSymbol(conn,
                                            xmlDesc,
                                            nfiles,
                                            files,
                                            flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainDefineXMLFuncType)(virConnectPtr conn,
                              const char * xml);

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainDefineXMLFuncType virDomainDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDefineXML",
                       (void**)&virDomainDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDefineXMLSymbol(conn,
                                   xml);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainDefineXMLFlagsFuncType)(virConnectPtr conn,
                                   const char * xml,
                                   unsigned int flags);

virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainDefineXMLFlagsFuncType virDomainDefineXMLFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDefineXMLFlags",
                       (void**)&virDomainDefineXMLFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDefineXMLFlagsSymbol(conn,
                                        xml,
                                        flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDelIOThreadFuncType)(virDomainPtr domain,
                                unsigned int iothread_id,
                                unsigned int flags);

int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainDelIOThreadFuncType virDomainDelIOThreadSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDelIOThread",
                       (void**)&virDomainDelIOThreadSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDelIOThreadSymbol(domain,
                                     iothread_id,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDelThrottleGroupFuncType)(virDomainPtr dom,
                                     const char * group,
                                     unsigned int flags);

int
virDomainDelThrottleGroupWrapper(virDomainPtr dom,
                                 const char * group,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainDelThrottleGroupFuncType virDomainDelThrottleGroupSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDelThrottleGroup",
                       (void**)&virDomainDelThrottleGroupSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDelThrottleGroupSymbol(dom,
                                          group,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDestroyFuncType)(virDomainPtr domain);

int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainDestroyFuncType virDomainDestroySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDestroy",
                       (void**)&virDomainDestroySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDestroySymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDestroyFlagsFuncType)(virDomainPtr domain,
                                 unsigned int flags);

int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainDestroyFlagsFuncType virDomainDestroyFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDestroyFlags",
                       (void**)&virDomainDestroyFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDestroyFlagsSymbol(domain,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDetachDeviceFuncType)(virDomainPtr domain,
                                 const char * xml);

int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainDetachDeviceFuncType virDomainDetachDeviceSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDetachDevice",
                       (void**)&virDomainDetachDeviceSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDetachDeviceSymbol(domain,
                                      xml);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDetachDeviceAliasFuncType)(virDomainPtr domain,
                                      const char * alias,
                                      unsigned int flags);

int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainDetachDeviceAliasFuncType virDomainDetachDeviceAliasSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDetachDeviceAlias",
                       (void**)&virDomainDetachDeviceAliasSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDetachDeviceAliasSymbol(domain,
                                           alias,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainDetachDeviceFlagsFuncType)(virDomainPtr domain,
                                      const char * xml,
                                      unsigned int flags);

int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainDetachDeviceFlagsFuncType virDomainDetachDeviceFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainDetachDeviceFlags",
                       (void**)&virDomainDetachDeviceFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainDetachDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFDAssociateFuncType)(virDomainPtr domain,
                                const char * name,
                                unsigned int nfds,
                                int * fds,
                                unsigned int flags);

int
virDomainFDAssociateWrapper(virDomainPtr domain,
                            const char * name,
                            unsigned int nfds,
                            int * fds,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainFDAssociateFuncType virDomainFDAssociateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainFDAssociate",
                       (void**)&virDomainFDAssociateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainFDAssociateSymbol(domain,
                                     name,
                                     nfds,
                                     fds,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFSFreezeFuncType)(virDomainPtr dom,
                             const char ** mountpoints,
                             unsigned int nmountpoints,
                             unsigned int flags);

int
virDomainFSFreezeWrapper(virDomainPtr dom,
                         const char ** mountpoints,
                         unsigned int nmountpoints,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainFSFreezeFuncType virDomainFSFreezeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainFSFreeze",
                       (void**)&virDomainFSFreezeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainFSFreezeSymbol(dom,
                                  mountpoints,
                                  nmountpoints,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainFSInfoFreeFuncType)(virDomainFSInfoPtr info);

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info)
{

    static virDomainFSInfoFreeFuncType virDomainFSInfoFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainFSInfoFree",
                       (void**)&virDomainFSInfoFreeSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virDomainFSInfoFreeSymbol(info);
}

typedef int
(*virDomainFSThawFuncType)(virDomainPtr dom,
                           const char ** mountpoints,
                           unsigned int nmountpoints,
                           unsigned int flags);

int
virDomainFSThawWrapper(virDomainPtr dom,
                       const char ** mountpoints,
                       unsigned int nmountpoints,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainFSThawFuncType virDomainFSThawSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainFSThaw",
                       (void**)&virDomainFSThawSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainFSThawSymbol(dom,
                                mountpoints,
                                nmountpoints,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFSTrimFuncType)(virDomainPtr dom,
                           const char * mountPoint,
                           unsigned long long minimum,
                           unsigned int flags);

int
virDomainFSTrimWrapper(virDomainPtr dom,
                       const char * mountPoint,
                       unsigned long long minimum,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainFSTrimFuncType virDomainFSTrimSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainFSTrim",
                       (void**)&virDomainFSTrimSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainFSTrimSymbol(dom,
                                mountPoint,
                                minimum,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainFreeFuncType)(virDomainPtr domain);

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err)
{
    int ret = -1;
    static virDomainFreeFuncType virDomainFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainFree",
                       (void**)&virDomainFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainFreeSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetAutostartFuncType)(virDomainPtr domain,
                                 int * autostart);

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int * autostart,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainGetAutostartFuncType virDomainGetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetAutostart",
                       (void**)&virDomainGetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetAutostartSymbol(domain,
                                      autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetAutostartOnceFuncType)(virDomainPtr domain,
                                     int * autostart);

int
virDomainGetAutostartOnceWrapper(virDomainPtr domain,
                                 int * autostart,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainGetAutostartOnceFuncType virDomainGetAutostartOnceSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetAutostartOnce",
                       (void**)&virDomainGetAutostartOnceSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetAutostartOnceSymbol(domain,
                                          autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlkioParametersFuncType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       unsigned int flags);

int
virDomainGetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int * nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainGetBlkioParametersFuncType virDomainGetBlkioParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetBlkioParameters",
                       (void**)&virDomainGetBlkioParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetBlkioParametersSymbol(domain,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlockInfoFuncType)(virDomainPtr domain,
                                 const char * disk,
                                 virDomainBlockInfoPtr info,
                                 unsigned int flags);

int
virDomainGetBlockInfoWrapper(virDomainPtr domain,
                             const char * disk,
                             virDomainBlockInfoPtr info,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainGetBlockInfoFuncType virDomainGetBlockInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetBlockInfo",
                       (void**)&virDomainGetBlockInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetBlockInfoSymbol(domain,
                                      disk,
                                      info,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlockIoTuneFuncType)(virDomainPtr dom,
                                   const char * disk,
                                   virTypedParameterPtr params,
                                   int * nparams,
                                   unsigned int flags);

int
virDomainGetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int * nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainGetBlockIoTuneFuncType virDomainGetBlockIoTuneSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetBlockIoTune",
                       (void**)&virDomainGetBlockIoTuneSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetBlockIoTuneSymbol(dom,
                                        disk,
                                        params,
                                        nparams,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetBlockJobInfoFuncType)(virDomainPtr dom,
                                    const char * disk,
                                    virDomainBlockJobInfoPtr info,
                                    unsigned int flags);

int
virDomainGetBlockJobInfoWrapper(virDomainPtr dom,
                                const char * disk,
                                virDomainBlockJobInfoPtr info,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainGetBlockJobInfoFuncType virDomainGetBlockJobInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetBlockJobInfo",
                       (void**)&virDomainGetBlockJobInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetBlockJobInfoSymbol(dom,
                                         disk,
                                         info,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetCPUStatsFuncType)(virDomainPtr domain,
                                virTypedParameterPtr params,
                                unsigned int nparams,
                                int start_cpu,
                                unsigned int ncpus,
                                unsigned int flags);

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
    static virDomainGetCPUStatsFuncType virDomainGetCPUStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetCPUStats",
                       (void**)&virDomainGetCPUStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetCPUStatsSymbol(domain,
                                     params,
                                     nparams,
                                     start_cpu,
                                     ncpus,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virConnectPtr
(*virDomainGetConnectFuncType)(virDomainPtr dom);

virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virDomainGetConnectFuncType virDomainGetConnectSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetConnect",
                       (void**)&virDomainGetConnectSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetConnectSymbol(dom);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetControlInfoFuncType)(virDomainPtr domain,
                                   virDomainControlInfoPtr info,
                                   unsigned int flags);

int
virDomainGetControlInfoWrapper(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainGetControlInfoFuncType virDomainGetControlInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetControlInfo",
                       (void**)&virDomainGetControlInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetControlInfoSymbol(domain,
                                        info,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetDiskErrorsFuncType)(virDomainPtr dom,
                                  virDomainDiskErrorPtr errors,
                                  unsigned int maxerrors,
                                  unsigned int flags);

int
virDomainGetDiskErrorsWrapper(virDomainPtr dom,
                              virDomainDiskErrorPtr errors,
                              unsigned int maxerrors,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetDiskErrorsFuncType virDomainGetDiskErrorsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetDiskErrors",
                       (void**)&virDomainGetDiskErrorsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetDiskErrorsSymbol(dom,
                                       errors,
                                       maxerrors,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetEmulatorPinInfoFuncType)(virDomainPtr domain,
                                       unsigned char * cpumap,
                                       int maplen,
                                       unsigned int flags);

int
virDomainGetEmulatorPinInfoWrapper(virDomainPtr domain,
                                   unsigned char * cpumap,
                                   int maplen,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainGetEmulatorPinInfoFuncType virDomainGetEmulatorPinInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetEmulatorPinInfo",
                       (void**)&virDomainGetEmulatorPinInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetEmulatorPinInfoSymbol(domain,
                                            cpumap,
                                            maplen,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetFSInfoFuncType)(virDomainPtr dom,
                              virDomainFSInfoPtr ** info,
                              unsigned int flags);

int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainGetFSInfoFuncType virDomainGetFSInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetFSInfo",
                       (void**)&virDomainGetFSInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetFSInfoSymbol(dom,
                                   info,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetGuestInfoFuncType)(virDomainPtr domain,
                                 unsigned int types,
                                 virTypedParameterPtr * params,
                                 int * nparams,
                                 unsigned int flags);

int
virDomainGetGuestInfoWrapper(virDomainPtr domain,
                             unsigned int types,
                             virTypedParameterPtr * params,
                             int * nparams,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainGetGuestInfoFuncType virDomainGetGuestInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetGuestInfo",
                       (void**)&virDomainGetGuestInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetGuestInfoSymbol(domain,
                                      types,
                                      params,
                                      nparams,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetGuestVcpusFuncType)(virDomainPtr domain,
                                  virTypedParameterPtr * params,
                                  unsigned int * nparams,
                                  unsigned int flags);

int
virDomainGetGuestVcpusWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              unsigned int * nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetGuestVcpusFuncType virDomainGetGuestVcpusSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetGuestVcpus",
                       (void**)&virDomainGetGuestVcpusSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetGuestVcpusSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetHostnameFuncType)(virDomainPtr domain,
                                unsigned int flags);

char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetHostnameFuncType virDomainGetHostnameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetHostname",
                       (void**)&virDomainGetHostnameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetHostnameSymbol(domain,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned int
(*virDomainGetIDFuncType)(virDomainPtr domain);

unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err)
{
    unsigned int ret = 0;
    static virDomainGetIDFuncType virDomainGetIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetID",
                       (void**)&virDomainGetIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetIDSymbol(domain);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetIOThreadInfoFuncType)(virDomainPtr dom,
                                    virDomainIOThreadInfoPtr ** info,
                                    unsigned int flags);

int
virDomainGetIOThreadInfoWrapper(virDomainPtr dom,
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainGetIOThreadInfoFuncType virDomainGetIOThreadInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetIOThreadInfo",
                       (void**)&virDomainGetIOThreadInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetIOThreadInfoSymbol(dom,
                                         info,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetInfoFuncType)(virDomainPtr domain,
                            virDomainInfoPtr info);

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainGetInfoFuncType virDomainGetInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetInfo",
                       (void**)&virDomainGetInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetInfoSymbol(domain,
                                 info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetInterfaceParametersFuncType)(virDomainPtr domain,
                                           const char * device,
                                           virTypedParameterPtr params,
                                           int * nparams,
                                           unsigned int flags);

int
virDomainGetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainGetInterfaceParametersFuncType virDomainGetInterfaceParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetInterfaceParameters",
                       (void**)&virDomainGetInterfaceParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetInterfaceParametersSymbol(domain,
                                                device,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetJobInfoFuncType)(virDomainPtr domain,
                               virDomainJobInfoPtr info);

int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err)
{
    int ret = -1;
    static virDomainGetJobInfoFuncType virDomainGetJobInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetJobInfo",
                       (void**)&virDomainGetJobInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetJobInfoSymbol(domain,
                                    info);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetJobStatsFuncType)(virDomainPtr domain,
                                int * type,
                                virTypedParameterPtr * params,
                                int * nparams,
                                unsigned int flags);

int
virDomainGetJobStatsWrapper(virDomainPtr domain,
                            int * type,
                            virTypedParameterPtr * params,
                            int * nparams,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainGetJobStatsFuncType virDomainGetJobStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetJobStats",
                       (void**)&virDomainGetJobStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetJobStatsSymbol(domain,
                                     type,
                                     params,
                                     nparams,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetLaunchSecurityInfoFuncType)(virDomainPtr domain,
                                          virTypedParameterPtr * params,
                                          int * nparams,
                                          unsigned int flags);

int
virDomainGetLaunchSecurityInfoWrapper(virDomainPtr domain,
                                      virTypedParameterPtr * params,
                                      int * nparams,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virDomainGetLaunchSecurityInfoFuncType virDomainGetLaunchSecurityInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetLaunchSecurityInfo",
                       (void**)&virDomainGetLaunchSecurityInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetLaunchSecurityInfoSymbol(domain,
                                               params,
                                               nparams,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef unsigned long
(*virDomainGetMaxMemoryFuncType)(virDomainPtr domain);

unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err)
{
    unsigned long ret = 0;
    static virDomainGetMaxMemoryFuncType virDomainGetMaxMemorySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetMaxMemory",
                       (void**)&virDomainGetMaxMemorySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetMaxMemorySymbol(domain);
    if (ret == 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetMaxVcpusFuncType)(virDomainPtr domain);

int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainGetMaxVcpusFuncType virDomainGetMaxVcpusSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetMaxVcpus",
                       (void**)&virDomainGetMaxVcpusSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetMaxVcpusSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetMemoryParametersFuncType)(virDomainPtr domain,
                                        virTypedParameterPtr params,
                                        int * nparams,
                                        unsigned int flags);

int
virDomainGetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int * nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainGetMemoryParametersFuncType virDomainGetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetMemoryParameters",
                       (void**)&virDomainGetMemoryParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetMemoryParametersSymbol(domain,
                                             params,
                                             nparams,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetMessagesFuncType)(virDomainPtr domain,
                                char *** msgs,
                                unsigned int flags);

int
virDomainGetMessagesWrapper(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainGetMessagesFuncType virDomainGetMessagesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetMessages",
                       (void**)&virDomainGetMessagesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetMessagesSymbol(domain,
                                     msgs,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetMetadataFuncType)(virDomainPtr domain,
                                int type,
                                const char * uri,
                                unsigned int flags);

char *
virDomainGetMetadataWrapper(virDomainPtr domain,
                            int type,
                            const char * uri,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetMetadataFuncType virDomainGetMetadataSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetMetadata",
                       (void**)&virDomainGetMetadataSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetMetadataSymbol(domain,
                                     type,
                                     uri,
                                     flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virDomainGetNameFuncType)(virDomainPtr domain);

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    const char * ret = NULL;
    static virDomainGetNameFuncType virDomainGetNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetName",
                       (void**)&virDomainGetNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetNameSymbol(domain);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetNumaParametersFuncType)(virDomainPtr domain,
                                      virTypedParameterPtr params,
                                      int * nparams,
                                      unsigned int flags);

int
virDomainGetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int * nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainGetNumaParametersFuncType virDomainGetNumaParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetNumaParameters",
                       (void**)&virDomainGetNumaParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetNumaParametersSymbol(domain,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetOSTypeFuncType)(virDomainPtr domain);

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetOSTypeFuncType virDomainGetOSTypeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetOSType",
                       (void**)&virDomainGetOSTypeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetOSTypeSymbol(domain);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetPerfEventsFuncType)(virDomainPtr domain,
                                  virTypedParameterPtr * params,
                                  int * nparams,
                                  unsigned int flags);

int
virDomainGetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr * params,
                              int * nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetPerfEventsFuncType virDomainGetPerfEventsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetPerfEvents",
                       (void**)&virDomainGetPerfEventsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetPerfEventsSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSchedulerParametersFuncType)(virDomainPtr domain,
                                           virTypedParameterPtr params,
                                           int * nparams);

int
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSchedulerParametersFuncType virDomainGetSchedulerParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetSchedulerParameters",
                       (void**)&virDomainGetSchedulerParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetSchedulerParametersSymbol(domain,
                                                params,
                                                nparams);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSchedulerParametersFlagsFuncType)(virDomainPtr domain,
                                                virTypedParameterPtr params,
                                                int * nparams,
                                                unsigned int flags);

int
virDomainGetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int * nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSchedulerParametersFlagsFuncType virDomainGetSchedulerParametersFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetSchedulerParametersFlags",
                       (void**)&virDomainGetSchedulerParametersFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetSchedulerParametersFlagsSymbol(domain,
                                                     params,
                                                     nparams,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetSchedulerTypeFuncType)(virDomainPtr domain,
                                     int * nparams);

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int * nparams,
                                 virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetSchedulerTypeFuncType virDomainGetSchedulerTypeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetSchedulerType",
                       (void**)&virDomainGetSchedulerTypeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetSchedulerTypeSymbol(domain,
                                          nparams);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSecurityLabelFuncType)(virDomainPtr domain,
                                     virSecurityLabelPtr seclabel);

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSecurityLabelFuncType virDomainGetSecurityLabelSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetSecurityLabel",
                       (void**)&virDomainGetSecurityLabelSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetSecurityLabelSymbol(domain,
                                          seclabel);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetSecurityLabelListFuncType)(virDomainPtr domain,
                                         virSecurityLabelPtr * seclabels);

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSecurityLabelListFuncType virDomainGetSecurityLabelListSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetSecurityLabelList",
                       (void**)&virDomainGetSecurityLabelListSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetSecurityLabelListSymbol(domain,
                                              seclabels);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetStateFuncType)(virDomainPtr domain,
                             int * state,
                             int * reason,
                             unsigned int flags);

int
virDomainGetStateWrapper(virDomainPtr domain,
                         int * state,
                         int * reason,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainGetStateFuncType virDomainGetStateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetState",
                       (void**)&virDomainGetStateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetStateSymbol(domain,
                                  state,
                                  reason,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetTimeFuncType)(virDomainPtr dom,
                            long long * seconds,
                            unsigned int * nseconds,
                            unsigned int flags);

int
virDomainGetTimeWrapper(virDomainPtr dom,
                        long long * seconds,
                        unsigned int * nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainGetTimeFuncType virDomainGetTimeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetTime",
                       (void**)&virDomainGetTimeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetTimeSymbol(dom,
                                 seconds,
                                 nseconds,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetUUIDFuncType)(virDomainPtr domain,
                            unsigned char * uuid);

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainGetUUIDFuncType virDomainGetUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetUUID",
                       (void**)&virDomainGetUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetUUIDSymbol(domain,
                                 uuid);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetUUIDStringFuncType)(virDomainPtr domain,
                                  char * buf);

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char * buf,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetUUIDStringFuncType virDomainGetUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetUUIDString",
                       (void**)&virDomainGetUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetUUIDStringSymbol(domain,
                                       buf);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetVcpuPinInfoFuncType)(virDomainPtr domain,
                                   int ncpumaps,
                                   unsigned char * cpumaps,
                                   int maplen,
                                   unsigned int flags);

int
virDomainGetVcpuPinInfoWrapper(virDomainPtr domain,
                               int ncpumaps,
                               unsigned char * cpumaps,
                               int maplen,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainGetVcpuPinInfoFuncType virDomainGetVcpuPinInfoSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetVcpuPinInfo",
                       (void**)&virDomainGetVcpuPinInfoSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetVcpuPinInfoSymbol(domain,
                                        ncpumaps,
                                        cpumaps,
                                        maplen,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetVcpusFuncType)(virDomainPtr domain,
                             virVcpuInfoPtr info,
                             int maxinfo,
                             unsigned char * cpumaps,
                             int maplen);

int
virDomainGetVcpusWrapper(virDomainPtr domain,
                         virVcpuInfoPtr info,
                         int maxinfo,
                         unsigned char * cpumaps,
                         int maplen,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainGetVcpusFuncType virDomainGetVcpusSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetVcpus",
                       (void**)&virDomainGetVcpusSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetVcpusSymbol(domain,
                                  info,
                                  maxinfo,
                                  cpumaps,
                                  maplen);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGetVcpusFlagsFuncType)(virDomainPtr domain,
                                  unsigned int flags);

int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetVcpusFlagsFuncType virDomainGetVcpusFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetVcpusFlags",
                       (void**)&virDomainGetVcpusFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetVcpusFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainGetXMLDescFuncType)(virDomainPtr domain,
                               unsigned int flags);

char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetXMLDescFuncType virDomainGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGetXMLDesc",
                       (void**)&virDomainGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGetXMLDescSymbol(domain,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainGraphicsReloadFuncType)(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int flags);

int
virDomainGraphicsReloadWrapper(virDomainPtr domain,
                               unsigned int type,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainGraphicsReloadFuncType virDomainGraphicsReloadSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainGraphicsReload",
                       (void**)&virDomainGraphicsReloadSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainGraphicsReloadSymbol(domain,
                                        type,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainHasManagedSaveImageFuncType)(virDomainPtr dom,
                                        unsigned int flags);

int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainHasManagedSaveImageFuncType virDomainHasManagedSaveImageSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainHasManagedSaveImage",
                       (void**)&virDomainHasManagedSaveImageSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainHasManagedSaveImageSymbol(dom,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainIOThreadInfoFreeFuncType)(virDomainIOThreadInfoPtr info);

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info)
{

    static virDomainIOThreadInfoFreeFuncType virDomainIOThreadInfoFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainIOThreadInfoFree",
                       (void**)&virDomainIOThreadInfoFreeSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virDomainIOThreadInfoFreeSymbol(info);
}

typedef int
(*virDomainInjectNMIFuncType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainInjectNMIFuncType virDomainInjectNMISymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainInjectNMI",
                       (void**)&virDomainInjectNMISymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainInjectNMISymbol(domain,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainInterfaceAddressesFuncType)(virDomainPtr dom,
                                       virDomainInterfacePtr ** ifaces,
                                       unsigned int source,
                                       unsigned int flags);

int
virDomainInterfaceAddressesWrapper(virDomainPtr dom,
                                   virDomainInterfacePtr ** ifaces,
                                   unsigned int source,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainInterfaceAddressesFuncType virDomainInterfaceAddressesSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainInterfaceAddresses",
                       (void**)&virDomainInterfaceAddressesSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainInterfaceAddressesSymbol(dom,
                                            ifaces,
                                            source,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainInterfaceFreeFuncType)(virDomainInterfacePtr iface);

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface)
{

    static virDomainInterfaceFreeFuncType virDomainInterfaceFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainInterfaceFree",
                       (void**)&virDomainInterfaceFreeSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virDomainInterfaceFreeSymbol(iface);
}

typedef int
(*virDomainInterfaceStatsFuncType)(virDomainPtr dom,
                                   const char * device,
                                   virDomainInterfaceStatsPtr stats,
                                   size_t size);

int
virDomainInterfaceStatsWrapper(virDomainPtr dom,
                               const char * device,
                               virDomainInterfaceStatsPtr stats,
                               size_t size,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainInterfaceStatsFuncType virDomainInterfaceStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainInterfaceStats",
                       (void**)&virDomainInterfaceStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainInterfaceStatsSymbol(dom,
                                        device,
                                        stats,
                                        size);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainIsActiveFuncType)(virDomainPtr dom);

int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainIsActiveFuncType virDomainIsActiveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainIsActive",
                       (void**)&virDomainIsActiveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainIsActiveSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainIsPersistentFuncType)(virDomainPtr dom);

int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainIsPersistentFuncType virDomainIsPersistentSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainIsPersistent",
                       (void**)&virDomainIsPersistentSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainIsPersistentSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainIsUpdatedFuncType)(virDomainPtr dom);

int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainIsUpdatedFuncType virDomainIsUpdatedSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainIsUpdated",
                       (void**)&virDomainIsUpdatedSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainIsUpdatedSymbol(dom);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainListGetStatsFuncType)(virDomainPtr * doms,
                                 unsigned int stats,
                                 virDomainStatsRecordPtr ** retStats,
                                 unsigned int flags);

int
virDomainListGetStatsWrapper(virDomainPtr * doms,
                             unsigned int stats,
                             virDomainStatsRecordPtr ** retStats,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainListGetStatsFuncType virDomainListGetStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainListGetStats",
                       (void**)&virDomainListGetStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainListGetStatsSymbol(doms,
                                      stats,
                                      retStats,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByIDFuncType)(virConnectPtr conn,
                               int id);

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByIDFuncType virDomainLookupByIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainLookupByID",
                       (void**)&virDomainLookupByIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainLookupByIDSymbol(conn,
                                    id);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByNameFuncType)(virConnectPtr conn,
                                 const char * name);

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char * name,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByNameFuncType virDomainLookupByNameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainLookupByName",
                       (void**)&virDomainLookupByNameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainLookupByNameSymbol(conn,
                                      name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByUUIDFuncType)(virConnectPtr conn,
                                 const unsigned char * uuid);

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByUUIDFuncType virDomainLookupByUUIDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainLookupByUUID",
                       (void**)&virDomainLookupByUUIDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainLookupByUUIDSymbol(conn,
                                      uuid);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainLookupByUUIDStringFuncType)(virConnectPtr conn,
                                       const char * uuidstr);

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByUUIDStringFuncType virDomainLookupByUUIDStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainLookupByUUIDString",
                       (void**)&virDomainLookupByUUIDStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainLookupByUUIDStringSymbol(conn,
                                            uuidstr);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainManagedSaveFuncType)(virDomainPtr dom,
                                unsigned int flags);

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainManagedSaveFuncType virDomainManagedSaveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainManagedSave",
                       (void**)&virDomainManagedSaveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainManagedSaveSymbol(dom,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainManagedSaveDefineXMLFuncType)(virDomainPtr domain,
                                         const char * dxml,
                                         unsigned int flags);

int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char * dxml,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainManagedSaveDefineXMLFuncType virDomainManagedSaveDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainManagedSaveDefineXML",
                       (void**)&virDomainManagedSaveDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainManagedSaveDefineXMLSymbol(domain,
                                              dxml,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainManagedSaveGetXMLDescFuncType)(virDomainPtr domain,
                                          unsigned int flags);

char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    char * ret = NULL;
    static virDomainManagedSaveGetXMLDescFuncType virDomainManagedSaveGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainManagedSaveGetXMLDesc",
                       (void**)&virDomainManagedSaveGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainManagedSaveGetXMLDescSymbol(domain,
                                               flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainManagedSaveRemoveFuncType)(virDomainPtr dom,
                                      unsigned int flags);

int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainManagedSaveRemoveFuncType virDomainManagedSaveRemoveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainManagedSaveRemove",
                       (void**)&virDomainManagedSaveRemoveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainManagedSaveRemoveSymbol(dom,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMemoryPeekFuncType)(virDomainPtr dom,
                               unsigned long long start,
                               size_t size,
                               void * buffer,
                               unsigned int flags);

int
virDomainMemoryPeekWrapper(virDomainPtr dom,
                           unsigned long long start,
                           size_t size,
                           void * buffer,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virDomainMemoryPeekFuncType virDomainMemoryPeekSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMemoryPeek",
                       (void**)&virDomainMemoryPeekSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMemoryPeekSymbol(dom,
                                    start,
                                    size,
                                    buffer,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMemoryStatsFuncType)(virDomainPtr dom,
                                virDomainMemoryStatPtr stats,
                                unsigned int nr_stats,
                                unsigned int flags);

int
virDomainMemoryStatsWrapper(virDomainPtr dom,
                            virDomainMemoryStatPtr stats,
                            unsigned int nr_stats,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainMemoryStatsFuncType virDomainMemoryStatsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMemoryStats",
                       (void**)&virDomainMemoryStatsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMemoryStatsSymbol(dom,
                                     stats,
                                     nr_stats,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainMigrateFuncType)(virDomainPtr domain,
                            virConnectPtr dconn,
                            unsigned long flags,
                            const char * dname,
                            const char * uri,
                            unsigned long bandwidth);

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
    static virDomainMigrateFuncType virDomainMigrateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrate",
                       (void**)&virDomainMigrateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateSymbol(domain,
                                 dconn,
                                 flags,
                                 dname,
                                 uri,
                                 bandwidth);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainMigrate2FuncType)(virDomainPtr domain,
                             virConnectPtr dconn,
                             const char * dxml,
                             unsigned long flags,
                             const char * dname,
                             const char * uri,
                             unsigned long bandwidth);

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
    static virDomainMigrate2FuncType virDomainMigrate2Symbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrate2",
                       (void**)&virDomainMigrate2Symbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrate2Symbol(domain,
                                  dconn,
                                  dxml,
                                  flags,
                                  dname,
                                  uri,
                                  bandwidth);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainMigrate3FuncType)(virDomainPtr domain,
                             virConnectPtr dconn,
                             virTypedParameterPtr params,
                             unsigned int nparams,
                             unsigned int flags);

virDomainPtr
virDomainMigrate3Wrapper(virDomainPtr domain,
                         virConnectPtr dconn,
                         virTypedParameterPtr params,
                         unsigned int nparams,
                         unsigned int flags,
                         virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainMigrate3FuncType virDomainMigrate3Symbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrate3",
                       (void**)&virDomainMigrate3Symbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrate3Symbol(domain,
                                  dconn,
                                  params,
                                  nparams,
                                  flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateGetCompressionCacheFuncType)(virDomainPtr domain,
                                               unsigned long long * cacheSize,
                                               unsigned int flags);

int
virDomainMigrateGetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long * cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateGetCompressionCacheFuncType virDomainMigrateGetCompressionCacheSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateGetCompressionCache",
                       (void**)&virDomainMigrateGetCompressionCacheSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateGetCompressionCacheSymbol(domain,
                                                    cacheSize,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateGetMaxDowntimeFuncType)(virDomainPtr domain,
                                          unsigned long long * downtime,
                                          unsigned int flags);

int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateGetMaxDowntimeFuncType virDomainMigrateGetMaxDowntimeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateGetMaxDowntime",
                       (void**)&virDomainMigrateGetMaxDowntimeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateGetMaxDowntimeSymbol(domain,
                                               downtime,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateGetMaxSpeedFuncType)(virDomainPtr domain,
                                       unsigned long * bandwidth,
                                       unsigned int flags);

int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long * bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateGetMaxSpeedFuncType virDomainMigrateGetMaxSpeedSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateGetMaxSpeed",
                       (void**)&virDomainMigrateGetMaxSpeedSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateGetMaxSpeedSymbol(domain,
                                            bandwidth,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateSetCompressionCacheFuncType)(virDomainPtr domain,
                                               unsigned long long cacheSize,
                                               unsigned int flags);

int
virDomainMigrateSetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateSetCompressionCacheFuncType virDomainMigrateSetCompressionCacheSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateSetCompressionCache",
                       (void**)&virDomainMigrateSetCompressionCacheSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateSetCompressionCacheSymbol(domain,
                                                    cacheSize,
                                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateSetMaxDowntimeFuncType)(virDomainPtr domain,
                                          unsigned long long downtime,
                                          unsigned int flags);

int
virDomainMigrateSetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateSetMaxDowntimeFuncType virDomainMigrateSetMaxDowntimeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateSetMaxDowntime",
                       (void**)&virDomainMigrateSetMaxDowntimeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateSetMaxDowntimeSymbol(domain,
                                               downtime,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateSetMaxSpeedFuncType)(virDomainPtr domain,
                                       unsigned long bandwidth,
                                       unsigned int flags);

int
virDomainMigrateSetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateSetMaxSpeedFuncType virDomainMigrateSetMaxSpeedSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateSetMaxSpeed",
                       (void**)&virDomainMigrateSetMaxSpeedSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateSetMaxSpeedSymbol(domain,
                                            bandwidth,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateStartPostCopyFuncType)(virDomainPtr domain,
                                         unsigned int flags);

int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateStartPostCopyFuncType virDomainMigrateStartPostCopySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateStartPostCopy",
                       (void**)&virDomainMigrateStartPostCopySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateStartPostCopySymbol(domain,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateToURIFuncType)(virDomainPtr domain,
                                 const char * duri,
                                 unsigned long flags,
                                 const char * dname,
                                 unsigned long bandwidth);

int
virDomainMigrateToURIWrapper(virDomainPtr domain,
                             const char * duri,
                             unsigned long flags,
                             const char * dname,
                             unsigned long bandwidth,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateToURIFuncType virDomainMigrateToURISymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateToURI",
                       (void**)&virDomainMigrateToURISymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateToURISymbol(domain,
                                      duri,
                                      flags,
                                      dname,
                                      bandwidth);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateToURI2FuncType)(virDomainPtr domain,
                                  const char * dconnuri,
                                  const char * miguri,
                                  const char * dxml,
                                  unsigned long flags,
                                  const char * dname,
                                  unsigned long bandwidth);

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
    static virDomainMigrateToURI2FuncType virDomainMigrateToURI2Symbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateToURI2",
                       (void**)&virDomainMigrateToURI2Symbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateToURI2Symbol(domain,
                                       dconnuri,
                                       miguri,
                                       dxml,
                                       flags,
                                       dname,
                                       bandwidth);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainMigrateToURI3FuncType)(virDomainPtr domain,
                                  const char * dconnuri,
                                  virTypedParameterPtr params,
                                  unsigned int nparams,
                                  unsigned int flags);

int
virDomainMigrateToURI3Wrapper(virDomainPtr domain,
                              const char * dconnuri,
                              virTypedParameterPtr params,
                              unsigned int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateToURI3FuncType virDomainMigrateToURI3Symbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainMigrateToURI3",
                       (void**)&virDomainMigrateToURI3Symbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainMigrateToURI3Symbol(domain,
                                       dconnuri,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenChannelFuncType)(virDomainPtr dom,
                                const char * name,
                                virStreamPtr st,
                                unsigned int flags);

int
virDomainOpenChannelWrapper(virDomainPtr dom,
                            const char * name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainOpenChannelFuncType virDomainOpenChannelSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainOpenChannel",
                       (void**)&virDomainOpenChannelSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainOpenChannelSymbol(dom,
                                     name,
                                     st,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenConsoleFuncType)(virDomainPtr dom,
                                const char * dev_name,
                                virStreamPtr st,
                                unsigned int flags);

int
virDomainOpenConsoleWrapper(virDomainPtr dom,
                            const char * dev_name,
                            virStreamPtr st,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainOpenConsoleFuncType virDomainOpenConsoleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainOpenConsole",
                       (void**)&virDomainOpenConsoleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainOpenConsoleSymbol(dom,
                                     dev_name,
                                     st,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenGraphicsFuncType)(virDomainPtr dom,
                                 unsigned int idx,
                                 int fd,
                                 unsigned int flags);

int
virDomainOpenGraphicsWrapper(virDomainPtr dom,
                             unsigned int idx,
                             int fd,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainOpenGraphicsFuncType virDomainOpenGraphicsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainOpenGraphics",
                       (void**)&virDomainOpenGraphicsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainOpenGraphicsSymbol(dom,
                                      idx,
                                      fd,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainOpenGraphicsFDFuncType)(virDomainPtr dom,
                                   unsigned int idx,
                                   unsigned int flags);

int
virDomainOpenGraphicsFDWrapper(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainOpenGraphicsFDFuncType virDomainOpenGraphicsFDSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainOpenGraphicsFD",
                       (void**)&virDomainOpenGraphicsFDSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainOpenGraphicsFDSymbol(dom,
                                        idx,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPMSuspendForDurationFuncType)(virDomainPtr dom,
                                         unsigned int target,
                                         unsigned long long duration,
                                         unsigned int flags);

int
virDomainPMSuspendForDurationWrapper(virDomainPtr dom,
                                     unsigned int target,
                                     unsigned long long duration,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainPMSuspendForDurationFuncType virDomainPMSuspendForDurationSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainPMSuspendForDuration",
                       (void**)&virDomainPMSuspendForDurationSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainPMSuspendForDurationSymbol(dom,
                                              target,
                                              duration,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPMWakeupFuncType)(virDomainPtr dom,
                             unsigned int flags);

int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainPMWakeupFuncType virDomainPMWakeupSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainPMWakeup",
                       (void**)&virDomainPMWakeupSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainPMWakeupSymbol(dom,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinEmulatorFuncType)(virDomainPtr domain,
                                unsigned char * cpumap,
                                int maplen,
                                unsigned int flags);

int
virDomainPinEmulatorWrapper(virDomainPtr domain,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainPinEmulatorFuncType virDomainPinEmulatorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainPinEmulator",
                       (void**)&virDomainPinEmulatorSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainPinEmulatorSymbol(domain,
                                     cpumap,
                                     maplen,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinIOThreadFuncType)(virDomainPtr domain,
                                unsigned int iothread_id,
                                unsigned char * cpumap,
                                int maplen,
                                unsigned int flags);

int
virDomainPinIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned char * cpumap,
                            int maplen,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainPinIOThreadFuncType virDomainPinIOThreadSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainPinIOThread",
                       (void**)&virDomainPinIOThreadSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainPinIOThreadSymbol(domain,
                                     iothread_id,
                                     cpumap,
                                     maplen,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinVcpuFuncType)(virDomainPtr domain,
                            unsigned int vcpu,
                            unsigned char * cpumap,
                            int maplen);

int
virDomainPinVcpuWrapper(virDomainPtr domain,
                        unsigned int vcpu,
                        unsigned char * cpumap,
                        int maplen,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainPinVcpuFuncType virDomainPinVcpuSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainPinVcpu",
                       (void**)&virDomainPinVcpuSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainPinVcpuSymbol(domain,
                                 vcpu,
                                 cpumap,
                                 maplen);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainPinVcpuFlagsFuncType)(virDomainPtr domain,
                                 unsigned int vcpu,
                                 unsigned char * cpumap,
                                 int maplen,
                                 unsigned int flags);

int
virDomainPinVcpuFlagsWrapper(virDomainPtr domain,
                             unsigned int vcpu,
                             unsigned char * cpumap,
                             int maplen,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainPinVcpuFlagsFuncType virDomainPinVcpuFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainPinVcpuFlags",
                       (void**)&virDomainPinVcpuFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainPinVcpuFlagsSymbol(domain,
                                      vcpu,
                                      cpumap,
                                      maplen,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRebootFuncType)(virDomainPtr domain,
                           unsigned int flags);

int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainRebootFuncType virDomainRebootSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainReboot",
                       (void**)&virDomainRebootSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRebootSymbol(domain,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRefFuncType)(virDomainPtr domain);

int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err)
{
    int ret = -1;
    static virDomainRefFuncType virDomainRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainRef",
                       (void**)&virDomainRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRefSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRenameFuncType)(virDomainPtr dom,
                           const char * new_name,
                           unsigned int flags);

int
virDomainRenameWrapper(virDomainPtr dom,
                       const char * new_name,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainRenameFuncType virDomainRenameSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainRename",
                       (void**)&virDomainRenameSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRenameSymbol(dom,
                                new_name,
                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainResetFuncType)(virDomainPtr domain,
                          unsigned int flags);

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err)
{
    int ret = -1;
    static virDomainResetFuncType virDomainResetSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainReset",
                       (void**)&virDomainResetSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainResetSymbol(domain,
                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRestoreFuncType)(virConnectPtr conn,
                            const char * from);

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char * from,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainRestoreFuncType virDomainRestoreSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainRestore",
                       (void**)&virDomainRestoreSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRestoreSymbol(conn,
                                 from);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRestoreFlagsFuncType)(virConnectPtr conn,
                                 const char * from,
                                 const char * dxml,
                                 unsigned int flags);

int
virDomainRestoreFlagsWrapper(virConnectPtr conn,
                             const char * from,
                             const char * dxml,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainRestoreFlagsFuncType virDomainRestoreFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainRestoreFlags",
                       (void**)&virDomainRestoreFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRestoreFlagsSymbol(conn,
                                      from,
                                      dxml,
                                      flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainRestoreParamsFuncType)(virConnectPtr conn,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags);

int
virDomainRestoreParamsWrapper(virConnectPtr conn,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainRestoreParamsFuncType virDomainRestoreParamsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainRestoreParams",
                       (void**)&virDomainRestoreParamsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainRestoreParamsSymbol(conn,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainResumeFuncType)(virDomainPtr domain);

int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainResumeFuncType virDomainResumeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainResume",
                       (void**)&virDomainResumeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainResumeSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveFuncType)(virDomainPtr domain,
                         const char * to);

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char * to,
                     virErrorPtr err)
{
    int ret = -1;
    static virDomainSaveFuncType virDomainSaveSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSave",
                       (void**)&virDomainSaveSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSaveSymbol(domain,
                              to);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveFlagsFuncType)(virDomainPtr domain,
                              const char * to,
                              const char * dxml,
                              unsigned int flags);

int
virDomainSaveFlagsWrapper(virDomainPtr domain,
                          const char * to,
                          const char * dxml,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainSaveFlagsFuncType virDomainSaveFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSaveFlags",
                       (void**)&virDomainSaveFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSaveFlagsSymbol(domain,
                                   to,
                                   dxml,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveImageDefineXMLFuncType)(virConnectPtr conn,
                                       const char * file,
                                       const char * dxml,
                                       unsigned int flags);

int
virDomainSaveImageDefineXMLWrapper(virConnectPtr conn,
                                   const char * file,
                                   const char * dxml,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainSaveImageDefineXMLFuncType virDomainSaveImageDefineXMLSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSaveImageDefineXML",
                       (void**)&virDomainSaveImageDefineXMLSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSaveImageDefineXMLSymbol(conn,
                                            file,
                                            dxml,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainSaveImageGetXMLDescFuncType)(virConnectPtr conn,
                                        const char * file,
                                        unsigned int flags);

char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    char * ret = NULL;
    static virDomainSaveImageGetXMLDescFuncType virDomainSaveImageGetXMLDescSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSaveImageGetXMLDesc",
                       (void**)&virDomainSaveImageGetXMLDescSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSaveImageGetXMLDescSymbol(conn,
                                             file,
                                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSaveParamsFuncType)(virDomainPtr domain,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags);

int
virDomainSaveParamsWrapper(virDomainPtr domain,
                           virTypedParameterPtr params,
                           int nparams,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = -1;
    static virDomainSaveParamsFuncType virDomainSaveParamsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSaveParams",
                       (void**)&virDomainSaveParamsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSaveParamsSymbol(domain,
                                    params,
                                    nparams,
                                    flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainScreenshotFuncType)(virDomainPtr domain,
                               virStreamPtr stream,
                               unsigned int screen,
                               unsigned int flags);

char *
virDomainScreenshotWrapper(virDomainPtr domain,
                           virStreamPtr stream,
                           unsigned int screen,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
    static virDomainScreenshotFuncType virDomainScreenshotSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainScreenshot",
                       (void**)&virDomainScreenshotSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainScreenshotSymbol(domain,
                                    stream,
                                    screen,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSendKeyFuncType)(virDomainPtr domain,
                            unsigned int codeset,
                            unsigned int holdtime,
                            unsigned int * keycodes,
                            int nkeycodes,
                            unsigned int flags);

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
    static virDomainSendKeyFuncType virDomainSendKeySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSendKey",
                       (void**)&virDomainSendKeySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSendKeySymbol(domain,
                                 codeset,
                                 holdtime,
                                 keycodes,
                                 nkeycodes,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSendProcessSignalFuncType)(virDomainPtr domain,
                                      long long pid_value,
                                      unsigned int signum,
                                      unsigned int flags);

int
virDomainSendProcessSignalWrapper(virDomainPtr domain,
                                  long long pid_value,
                                  unsigned int signum,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSendProcessSignalFuncType virDomainSendProcessSignalSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSendProcessSignal",
                       (void**)&virDomainSendProcessSignalSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSendProcessSignalSymbol(domain,
                                           pid_value,
                                           signum,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetAutostartFuncType)(virDomainPtr domain,
                                 int autostart);

int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainSetAutostartFuncType virDomainSetAutostartSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetAutostart",
                       (void**)&virDomainSetAutostartSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetAutostartSymbol(domain,
                                      autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetAutostartOnceFuncType)(virDomainPtr domain,
                                     int autostart);

int
virDomainSetAutostartOnceWrapper(virDomainPtr domain,
                                 int autostart,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainSetAutostartOnceFuncType virDomainSetAutostartOnceSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetAutostartOnce",
                       (void**)&virDomainSetAutostartOnceSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetAutostartOnceSymbol(domain,
                                          autostart);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetBlkioParametersFuncType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags);

int
virDomainSetBlkioParametersWrapper(virDomainPtr domain,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainSetBlkioParametersFuncType virDomainSetBlkioParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetBlkioParameters",
                       (void**)&virDomainSetBlkioParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetBlkioParametersSymbol(domain,
                                            params,
                                            nparams,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetBlockIoTuneFuncType)(virDomainPtr dom,
                                   const char * disk,
                                   virTypedParameterPtr params,
                                   int nparams,
                                   unsigned int flags);

int
virDomainSetBlockIoTuneWrapper(virDomainPtr dom,
                               const char * disk,
                               virTypedParameterPtr params,
                               int nparams,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainSetBlockIoTuneFuncType virDomainSetBlockIoTuneSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetBlockIoTune",
                       (void**)&virDomainSetBlockIoTuneSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetBlockIoTuneSymbol(dom,
                                        disk,
                                        params,
                                        nparams,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetBlockThresholdFuncType)(virDomainPtr domain,
                                      const char * dev,
                                      unsigned long long threshold,
                                      unsigned int flags);

int
virDomainSetBlockThresholdWrapper(virDomainPtr domain,
                                  const char * dev,
                                  unsigned long long threshold,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSetBlockThresholdFuncType virDomainSetBlockThresholdSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetBlockThreshold",
                       (void**)&virDomainSetBlockThresholdSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetBlockThresholdSymbol(domain,
                                           dev,
                                           threshold,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetGuestVcpusFuncType)(virDomainPtr domain,
                                  const char * cpumap,
                                  int state,
                                  unsigned int flags);

int
virDomainSetGuestVcpusWrapper(virDomainPtr domain,
                              const char * cpumap,
                              int state,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainSetGuestVcpusFuncType virDomainSetGuestVcpusSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetGuestVcpus",
                       (void**)&virDomainSetGuestVcpusSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetGuestVcpusSymbol(domain,
                                       cpumap,
                                       state,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetIOThreadParamsFuncType)(virDomainPtr domain,
                                      unsigned int iothread_id,
                                      virTypedParameterPtr params,
                                      int nparams,
                                      unsigned int flags);

int
virDomainSetIOThreadParamsWrapper(virDomainPtr domain,
                                  unsigned int iothread_id,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSetIOThreadParamsFuncType virDomainSetIOThreadParamsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetIOThreadParams",
                       (void**)&virDomainSetIOThreadParamsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetIOThreadParamsSymbol(domain,
                                           iothread_id,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetInterfaceParametersFuncType)(virDomainPtr domain,
                                           const char * device,
                                           virTypedParameterPtr params,
                                           int nparams,
                                           unsigned int flags);

int
virDomainSetInterfaceParametersWrapper(virDomainPtr domain,
                                       const char * device,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainSetInterfaceParametersFuncType virDomainSetInterfaceParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetInterfaceParameters",
                       (void**)&virDomainSetInterfaceParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetInterfaceParametersSymbol(domain,
                                                device,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetLaunchSecurityStateFuncType)(virDomainPtr domain,
                                           virTypedParameterPtr params,
                                           int nparams,
                                           unsigned int flags);

int
virDomainSetLaunchSecurityStateWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       unsigned int flags,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainSetLaunchSecurityStateFuncType virDomainSetLaunchSecurityStateSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetLaunchSecurityState",
                       (void**)&virDomainSetLaunchSecurityStateSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetLaunchSecurityStateSymbol(domain,
                                                params,
                                                nparams,
                                                flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetLifecycleActionFuncType)(virDomainPtr domain,
                                       unsigned int type,
                                       unsigned int action,
                                       unsigned int flags);

int
virDomainSetLifecycleActionWrapper(virDomainPtr domain,
                                   unsigned int type,
                                   unsigned int action,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainSetLifecycleActionFuncType virDomainSetLifecycleActionSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetLifecycleAction",
                       (void**)&virDomainSetLifecycleActionSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetLifecycleActionSymbol(domain,
                                            type,
                                            action,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMaxMemoryFuncType)(virDomainPtr domain,
                                 unsigned long memory);

int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMaxMemoryFuncType virDomainSetMaxMemorySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetMaxMemory",
                       (void**)&virDomainSetMaxMemorySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetMaxMemorySymbol(domain,
                                      memory);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryFuncType)(virDomainPtr domain,
                              unsigned long memory);

int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryFuncType virDomainSetMemorySymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetMemory",
                       (void**)&virDomainSetMemorySymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetMemorySymbol(domain,
                                   memory);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryFlagsFuncType)(virDomainPtr domain,
                                   unsigned long memory,
                                   unsigned int flags);

int
virDomainSetMemoryFlagsWrapper(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryFlagsFuncType virDomainSetMemoryFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetMemoryFlags",
                       (void**)&virDomainSetMemoryFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetMemoryFlagsSymbol(domain,
                                        memory,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryParametersFuncType)(virDomainPtr domain,
                                        virTypedParameterPtr params,
                                        int nparams,
                                        unsigned int flags);

int
virDomainSetMemoryParametersWrapper(virDomainPtr domain,
                                    virTypedParameterPtr params,
                                    int nparams,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryParametersFuncType virDomainSetMemoryParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetMemoryParameters",
                       (void**)&virDomainSetMemoryParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetMemoryParametersSymbol(domain,
                                             params,
                                             nparams,
                                             flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMemoryStatsPeriodFuncType)(virDomainPtr domain,
                                         int period,
                                         unsigned int flags);

int
virDomainSetMemoryStatsPeriodWrapper(virDomainPtr domain,
                                     int period,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryStatsPeriodFuncType virDomainSetMemoryStatsPeriodSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetMemoryStatsPeriod",
                       (void**)&virDomainSetMemoryStatsPeriodSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetMemoryStatsPeriodSymbol(domain,
                                              period,
                                              flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetMetadataFuncType)(virDomainPtr domain,
                                int type,
                                const char * metadata,
                                const char * key,
                                const char * uri,
                                unsigned int flags);

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
    static virDomainSetMetadataFuncType virDomainSetMetadataSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetMetadata",
                       (void**)&virDomainSetMetadataSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetMetadataSymbol(domain,
                                     type,
                                     metadata,
                                     key,
                                     uri,
                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetNumaParametersFuncType)(virDomainPtr domain,
                                      virTypedParameterPtr params,
                                      int nparams,
                                      unsigned int flags);

int
virDomainSetNumaParametersWrapper(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainSetNumaParametersFuncType virDomainSetNumaParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetNumaParameters",
                       (void**)&virDomainSetNumaParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetNumaParametersSymbol(domain,
                                           params,
                                           nparams,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetPerfEventsFuncType)(virDomainPtr domain,
                                  virTypedParameterPtr params,
                                  int nparams,
                                  unsigned int flags);

int
virDomainSetPerfEventsWrapper(virDomainPtr domain,
                              virTypedParameterPtr params,
                              int nparams,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainSetPerfEventsFuncType virDomainSetPerfEventsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetPerfEvents",
                       (void**)&virDomainSetPerfEventsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetPerfEventsSymbol(domain,
                                       params,
                                       nparams,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetSchedulerParametersFuncType)(virDomainPtr domain,
                                           virTypedParameterPtr params,
                                           int nparams);

int
virDomainSetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainSetSchedulerParametersFuncType virDomainSetSchedulerParametersSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetSchedulerParameters",
                       (void**)&virDomainSetSchedulerParametersSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetSchedulerParametersSymbol(domain,
                                                params,
                                                nparams);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetSchedulerParametersFlagsFuncType)(virDomainPtr domain,
                                                virTypedParameterPtr params,
                                                int nparams,
                                                unsigned int flags);

int
virDomainSetSchedulerParametersFlagsWrapper(virDomainPtr domain,
                                            virTypedParameterPtr params,
                                            int nparams,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = -1;
    static virDomainSetSchedulerParametersFlagsFuncType virDomainSetSchedulerParametersFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetSchedulerParametersFlags",
                       (void**)&virDomainSetSchedulerParametersFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetSchedulerParametersFlagsSymbol(domain,
                                                     params,
                                                     nparams,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetThrottleGroupFuncType)(virDomainPtr dom,
                                     const char * group,
                                     virTypedParameterPtr params,
                                     int nparams,
                                     unsigned int flags);

int
virDomainSetThrottleGroupWrapper(virDomainPtr dom,
                                 const char * group,
                                 virTypedParameterPtr params,
                                 int nparams,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainSetThrottleGroupFuncType virDomainSetThrottleGroupSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetThrottleGroup",
                       (void**)&virDomainSetThrottleGroupSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetThrottleGroupSymbol(dom,
                                          group,
                                          params,
                                          nparams,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetTimeFuncType)(virDomainPtr dom,
                            long long seconds,
                            unsigned int nseconds,
                            unsigned int flags);

int
virDomainSetTimeWrapper(virDomainPtr dom,
                        long long seconds,
                        unsigned int nseconds,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainSetTimeFuncType virDomainSetTimeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetTime",
                       (void**)&virDomainSetTimeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetTimeSymbol(dom,
                                 seconds,
                                 nseconds,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetUserPasswordFuncType)(virDomainPtr dom,
                                    const char * user,
                                    const char * password,
                                    unsigned int flags);

int
virDomainSetUserPasswordWrapper(virDomainPtr dom,
                                const char * user,
                                const char * password,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainSetUserPasswordFuncType virDomainSetUserPasswordSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetUserPassword",
                       (void**)&virDomainSetUserPasswordSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetUserPasswordSymbol(dom,
                                         user,
                                         password,
                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetVcpuFuncType)(virDomainPtr domain,
                            const char * vcpumap,
                            int state,
                            unsigned int flags);

int
virDomainSetVcpuWrapper(virDomainPtr domain,
                        const char * vcpumap,
                        int state,
                        unsigned int flags,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainSetVcpuFuncType virDomainSetVcpuSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetVcpu",
                       (void**)&virDomainSetVcpuSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetVcpuSymbol(domain,
                                 vcpumap,
                                 state,
                                 flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetVcpusFuncType)(virDomainPtr domain,
                             unsigned int nvcpus);

int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainSetVcpusFuncType virDomainSetVcpusSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetVcpus",
                       (void**)&virDomainSetVcpusSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetVcpusSymbol(domain,
                                  nvcpus);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainSetVcpusFlagsFuncType)(virDomainPtr domain,
                                  unsigned int nvcpus,
                                  unsigned int flags);

int
virDomainSetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainSetVcpusFlagsFuncType virDomainSetVcpusFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSetVcpusFlags",
                       (void**)&virDomainSetVcpusFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSetVcpusFlagsSymbol(domain,
                                       nvcpus,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainShutdownFuncType)(virDomainPtr domain);

int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainShutdownFuncType virDomainShutdownSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainShutdown",
                       (void**)&virDomainShutdownSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainShutdownSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainShutdownFlagsFuncType)(virDomainPtr domain,
                                  unsigned int flags);

int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainShutdownFlagsFuncType virDomainShutdownFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainShutdownFlags",
                       (void**)&virDomainShutdownFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainShutdownFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainStartDirtyRateCalcFuncType)(virDomainPtr domain,
                                       int seconds,
                                       unsigned int flags);

int
virDomainStartDirtyRateCalcWrapper(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainStartDirtyRateCalcFuncType virDomainStartDirtyRateCalcSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainStartDirtyRateCalc",
                       (void**)&virDomainStartDirtyRateCalcSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainStartDirtyRateCalcSymbol(domain,
                                            seconds,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virDomainStatsRecordListFreeFuncType)(virDomainStatsRecordPtr * stats);

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats)
{

    static virDomainStatsRecordListFreeFuncType virDomainStatsRecordListFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainStatsRecordListFree",
                       (void**)&virDomainStatsRecordListFreeSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virDomainStatsRecordListFreeSymbol(stats);
}

typedef int
(*virDomainSuspendFuncType)(virDomainPtr domain);

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainSuspendFuncType virDomainSuspendSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainSuspend",
                       (void**)&virDomainSuspendSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainSuspendSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainUndefineFuncType)(virDomainPtr domain);

int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainUndefineFuncType virDomainUndefineSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainUndefine",
                       (void**)&virDomainUndefineSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainUndefineSymbol(domain);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainUndefineFlagsFuncType)(virDomainPtr domain,
                                  unsigned int flags);

int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainUndefineFlagsFuncType virDomainUndefineFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainUndefineFlags",
                       (void**)&virDomainUndefineFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainUndefineFlagsSymbol(domain,
                                       flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainUpdateDeviceFlagsFuncType)(virDomainPtr domain,
                                      const char * xml,
                                      unsigned int flags);

int
virDomainUpdateDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainUpdateDeviceFlagsFuncType virDomainUpdateDeviceFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDomainUpdateDeviceFlags",
                       (void**)&virDomainUpdateDeviceFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virDomainUpdateDeviceFlagsSymbol(domain,
                                           xml,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
