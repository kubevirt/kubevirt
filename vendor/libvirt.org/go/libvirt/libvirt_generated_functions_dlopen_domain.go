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
(*virConnectDomainEventDeregisterType)(virConnectPtr conn,
                                       virConnectDomainEventCallback cb);

int
virConnectDomainEventDeregisterWrapper(virConnectPtr conn,
                                       virConnectDomainEventCallback cb,
                                       virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainEventDeregisterType virConnectDomainEventDeregisterSymbol;
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
(*virConnectDomainEventDeregisterAnyType)(virConnectPtr conn,
                                          int callbackID);

int
virConnectDomainEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
                                          virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainEventDeregisterAnyType virConnectDomainEventDeregisterAnySymbol;
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
(*virConnectDomainEventRegisterType)(virConnectPtr conn,
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
    static virConnectDomainEventRegisterType virConnectDomainEventRegisterSymbol;
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
(*virConnectDomainEventRegisterAnyType)(virConnectPtr conn,
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
    static virConnectDomainEventRegisterAnyType virConnectDomainEventRegisterAnySymbol;
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
(*virConnectDomainXMLFromNativeType)(virConnectPtr conn,
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
    static virConnectDomainXMLFromNativeType virConnectDomainXMLFromNativeSymbol;
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
(*virConnectDomainXMLToNativeType)(virConnectPtr conn,
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
    static virConnectDomainXMLToNativeType virConnectDomainXMLToNativeSymbol;
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
(*virConnectGetAllDomainStatsType)(virConnectPtr conn,
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
    static virConnectGetAllDomainStatsType virConnectGetAllDomainStatsSymbol;
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
(*virConnectGetDomainCapabilitiesType)(virConnectPtr conn,
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
    static virConnectGetDomainCapabilitiesType virConnectGetDomainCapabilitiesSymbol;
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
(*virConnectListAllDomainsType)(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags);

int
virConnectListAllDomainsWrapper(virConnectPtr conn,
                                virDomainPtr ** domains,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virConnectListAllDomainsType virConnectListAllDomainsSymbol;
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
(*virConnectListDefinedDomainsType)(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames);

int
virConnectListDefinedDomainsWrapper(virConnectPtr conn,
                                    char ** const names,
                                    int maxnames,
                                    virErrorPtr err)
{
    int ret = -1;
    static virConnectListDefinedDomainsType virConnectListDefinedDomainsSymbol;
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
(*virConnectListDomainsType)(virConnectPtr conn,
                             int * ids,
                             int maxids);

int
virConnectListDomainsWrapper(virConnectPtr conn,
                             int * ids,
                             int maxids,
                             virErrorPtr err)
{
    int ret = -1;
    static virConnectListDomainsType virConnectListDomainsSymbol;
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
(*virConnectNumOfDefinedDomainsType)(virConnectPtr conn);

int
virConnectNumOfDefinedDomainsWrapper(virConnectPtr conn,
                                     virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDefinedDomainsType virConnectNumOfDefinedDomainsSymbol;
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
(*virConnectNumOfDomainsType)(virConnectPtr conn);

int
virConnectNumOfDomainsWrapper(virConnectPtr conn,
                              virErrorPtr err)
{
    int ret = -1;
    static virConnectNumOfDomainsType virConnectNumOfDomainsSymbol;
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
(*virDomainAbortJobType)(virDomainPtr domain);

int
virDomainAbortJobWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainAbortJobType virDomainAbortJobSymbol;
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
(*virDomainAbortJobFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainAbortJobFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainAbortJobFlagsType virDomainAbortJobFlagsSymbol;
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
(*virDomainAddIOThreadType)(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags);

int
virDomainAddIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainAddIOThreadType virDomainAddIOThreadSymbol;
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
(*virDomainAgentSetResponseTimeoutType)(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags);

int
virDomainAgentSetResponseTimeoutWrapper(virDomainPtr domain,
                                        int timeout,
                                        unsigned int flags,
                                        virErrorPtr err)
{
    int ret = -1;
    static virDomainAgentSetResponseTimeoutType virDomainAgentSetResponseTimeoutSymbol;
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
(*virDomainAttachDeviceType)(virDomainPtr domain,
                             const char * xml);

int
virDomainAttachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainAttachDeviceType virDomainAttachDeviceSymbol;
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
(*virDomainAttachDeviceFlagsType)(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags);

int
virDomainAttachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainAttachDeviceFlagsType virDomainAttachDeviceFlagsSymbol;
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
(*virDomainAuthorizedSSHKeysGetType)(virDomainPtr domain,
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
    static virDomainAuthorizedSSHKeysGetType virDomainAuthorizedSSHKeysGetSymbol;
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
(*virDomainAuthorizedSSHKeysSetType)(virDomainPtr domain,
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
    static virDomainAuthorizedSSHKeysSetType virDomainAuthorizedSSHKeysSetSymbol;
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
(*virDomainBackupBeginType)(virDomainPtr domain,
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
    static virDomainBackupBeginType virDomainBackupBeginSymbol;
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
(*virDomainBackupGetXMLDescType)(virDomainPtr domain,
                                 unsigned int flags);

char *
virDomainBackupGetXMLDescWrapper(virDomainPtr domain,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = NULL;
    static virDomainBackupGetXMLDescType virDomainBackupGetXMLDescSymbol;
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
(*virDomainBlockCommitType)(virDomainPtr dom,
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
    static virDomainBlockCommitType virDomainBlockCommitSymbol;
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
(*virDomainBlockCopyType)(virDomainPtr dom,
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
    static virDomainBlockCopyType virDomainBlockCopySymbol;
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
(*virDomainBlockJobAbortType)(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags);

int
virDomainBlockJobAbortWrapper(virDomainPtr dom,
                              const char * disk,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainBlockJobAbortType virDomainBlockJobAbortSymbol;
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
(*virDomainBlockJobSetSpeedType)(virDomainPtr dom,
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
    static virDomainBlockJobSetSpeedType virDomainBlockJobSetSpeedSymbol;
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
(*virDomainBlockPeekType)(virDomainPtr dom,
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
    static virDomainBlockPeekType virDomainBlockPeekSymbol;
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
(*virDomainBlockPullType)(virDomainPtr dom,
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
    static virDomainBlockPullType virDomainBlockPullSymbol;
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
(*virDomainBlockRebaseType)(virDomainPtr dom,
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
    static virDomainBlockRebaseType virDomainBlockRebaseSymbol;
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
(*virDomainBlockResizeType)(virDomainPtr dom,
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
    static virDomainBlockResizeType virDomainBlockResizeSymbol;
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
(*virDomainBlockStatsType)(virDomainPtr dom,
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
    static virDomainBlockStatsType virDomainBlockStatsSymbol;
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
(*virDomainBlockStatsFlagsType)(virDomainPtr dom,
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
    static virDomainBlockStatsFlagsType virDomainBlockStatsFlagsSymbol;
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
(*virDomainCoreDumpType)(virDomainPtr domain,
                         const char * to,
                         unsigned int flags);

int
virDomainCoreDumpWrapper(virDomainPtr domain,
                         const char * to,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainCoreDumpType virDomainCoreDumpSymbol;
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
(*virDomainCoreDumpWithFormatType)(virDomainPtr domain,
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
    static virDomainCoreDumpWithFormatType virDomainCoreDumpWithFormatSymbol;
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
(*virDomainCreateType)(virDomainPtr domain);

int
virDomainCreateWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainCreateType virDomainCreateSymbol;
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
(*virDomainCreateLinuxType)(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags);

virDomainPtr
virDomainCreateLinuxWrapper(virConnectPtr conn,
                            const char * xmlDesc,
                            unsigned int flags,
                            virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCreateLinuxType virDomainCreateLinuxSymbol;
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
(*virDomainCreateWithFilesType)(virDomainPtr domain,
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
    static virDomainCreateWithFilesType virDomainCreateWithFilesSymbol;
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
(*virDomainCreateWithFlagsType)(virDomainPtr domain,
                                unsigned int flags);

int
virDomainCreateWithFlagsWrapper(virDomainPtr domain,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainCreateWithFlagsType virDomainCreateWithFlagsSymbol;
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
(*virDomainCreateXMLType)(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags);

virDomainPtr
virDomainCreateXMLWrapper(virConnectPtr conn,
                          const char * xmlDesc,
                          unsigned int flags,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainCreateXMLType virDomainCreateXMLSymbol;
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
(*virDomainCreateXMLWithFilesType)(virConnectPtr conn,
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
    static virDomainCreateXMLWithFilesType virDomainCreateXMLWithFilesSymbol;
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
(*virDomainDefineXMLType)(virConnectPtr conn,
                          const char * xml);

virDomainPtr
virDomainDefineXMLWrapper(virConnectPtr conn,
                          const char * xml,
                          virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainDefineXMLType virDomainDefineXMLSymbol;
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
(*virDomainDefineXMLFlagsType)(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags);

virDomainPtr
virDomainDefineXMLFlagsWrapper(virConnectPtr conn,
                               const char * xml,
                               unsigned int flags,
                               virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainDefineXMLFlagsType virDomainDefineXMLFlagsSymbol;
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
(*virDomainDelIOThreadType)(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags);

int
virDomainDelIOThreadWrapper(virDomainPtr domain,
                            unsigned int iothread_id,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainDelIOThreadType virDomainDelIOThreadSymbol;
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
(*virDomainDestroyType)(virDomainPtr domain);

int
virDomainDestroyWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainDestroyType virDomainDestroySymbol;
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
(*virDomainDestroyFlagsType)(virDomainPtr domain,
                             unsigned int flags);

int
virDomainDestroyFlagsWrapper(virDomainPtr domain,
                             unsigned int flags,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainDestroyFlagsType virDomainDestroyFlagsSymbol;
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
(*virDomainDetachDeviceType)(virDomainPtr domain,
                             const char * xml);

int
virDomainDetachDeviceWrapper(virDomainPtr domain,
                             const char * xml,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainDetachDeviceType virDomainDetachDeviceSymbol;
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
(*virDomainDetachDeviceAliasType)(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags);

int
virDomainDetachDeviceAliasWrapper(virDomainPtr domain,
                                  const char * alias,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainDetachDeviceAliasType virDomainDetachDeviceAliasSymbol;
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
(*virDomainDetachDeviceFlagsType)(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags);

int
virDomainDetachDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainDetachDeviceFlagsType virDomainDetachDeviceFlagsSymbol;
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
(*virDomainFDAssociateType)(virDomainPtr domain,
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
    static virDomainFDAssociateType virDomainFDAssociateSymbol;
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
(*virDomainFSFreezeType)(virDomainPtr dom,
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
    static virDomainFSFreezeType virDomainFSFreezeSymbol;
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
(*virDomainFSInfoFreeType)(virDomainFSInfoPtr info);

void
virDomainFSInfoFreeWrapper(virDomainFSInfoPtr info)
{

    static virDomainFSInfoFreeType virDomainFSInfoFreeSymbol;
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
(*virDomainFSThawType)(virDomainPtr dom,
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
    static virDomainFSThawType virDomainFSThawSymbol;
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
(*virDomainFSTrimType)(virDomainPtr dom,
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
    static virDomainFSTrimType virDomainFSTrimSymbol;
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
(*virDomainFreeType)(virDomainPtr domain);

int
virDomainFreeWrapper(virDomainPtr domain,
                     virErrorPtr err)
{
    int ret = -1;
    static virDomainFreeType virDomainFreeSymbol;
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
(*virDomainGetAutostartType)(virDomainPtr domain,
                             int * autostart);

int
virDomainGetAutostartWrapper(virDomainPtr domain,
                             int * autostart,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainGetAutostartType virDomainGetAutostartSymbol;
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
(*virDomainGetBlkioParametersType)(virDomainPtr domain,
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
    static virDomainGetBlkioParametersType virDomainGetBlkioParametersSymbol;
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
(*virDomainGetBlockInfoType)(virDomainPtr domain,
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
    static virDomainGetBlockInfoType virDomainGetBlockInfoSymbol;
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
(*virDomainGetBlockIoTuneType)(virDomainPtr dom,
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
    static virDomainGetBlockIoTuneType virDomainGetBlockIoTuneSymbol;
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
(*virDomainGetBlockJobInfoType)(virDomainPtr dom,
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
    static virDomainGetBlockJobInfoType virDomainGetBlockJobInfoSymbol;
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
(*virDomainGetCPUStatsType)(virDomainPtr domain,
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
    static virDomainGetCPUStatsType virDomainGetCPUStatsSymbol;
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
(*virDomainGetConnectType)(virDomainPtr dom);

virConnectPtr
virDomainGetConnectWrapper(virDomainPtr dom,
                           virErrorPtr err)
{
    virConnectPtr ret = NULL;
    static virDomainGetConnectType virDomainGetConnectSymbol;
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
(*virDomainGetControlInfoType)(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags);

int
virDomainGetControlInfoWrapper(virDomainPtr domain,
                               virDomainControlInfoPtr info,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainGetControlInfoType virDomainGetControlInfoSymbol;
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
(*virDomainGetDiskErrorsType)(virDomainPtr dom,
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
    static virDomainGetDiskErrorsType virDomainGetDiskErrorsSymbol;
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
(*virDomainGetEmulatorPinInfoType)(virDomainPtr domain,
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
    static virDomainGetEmulatorPinInfoType virDomainGetEmulatorPinInfoSymbol;
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
(*virDomainGetFSInfoType)(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags);

int
virDomainGetFSInfoWrapper(virDomainPtr dom,
                          virDomainFSInfoPtr ** info,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainGetFSInfoType virDomainGetFSInfoSymbol;
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
(*virDomainGetGuestInfoType)(virDomainPtr domain,
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
    static virDomainGetGuestInfoType virDomainGetGuestInfoSymbol;
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
(*virDomainGetGuestVcpusType)(virDomainPtr domain,
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
    static virDomainGetGuestVcpusType virDomainGetGuestVcpusSymbol;
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
(*virDomainGetHostnameType)(virDomainPtr domain,
                            unsigned int flags);

char *
virDomainGetHostnameWrapper(virDomainPtr domain,
                            unsigned int flags,
                            virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetHostnameType virDomainGetHostnameSymbol;
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
(*virDomainGetIDType)(virDomainPtr domain);

unsigned int
virDomainGetIDWrapper(virDomainPtr domain,
                      virErrorPtr err)
{
    unsigned int ret = 0;
    static virDomainGetIDType virDomainGetIDSymbol;
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
(*virDomainGetIOThreadInfoType)(virDomainPtr dom,
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags);

int
virDomainGetIOThreadInfoWrapper(virDomainPtr dom,
                                virDomainIOThreadInfoPtr ** info,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = -1;
    static virDomainGetIOThreadInfoType virDomainGetIOThreadInfoSymbol;
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
(*virDomainGetInfoType)(virDomainPtr domain,
                        virDomainInfoPtr info);

int
virDomainGetInfoWrapper(virDomainPtr domain,
                        virDomainInfoPtr info,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainGetInfoType virDomainGetInfoSymbol;
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
(*virDomainGetInterfaceParametersType)(virDomainPtr domain,
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
    static virDomainGetInterfaceParametersType virDomainGetInterfaceParametersSymbol;
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
(*virDomainGetJobInfoType)(virDomainPtr domain,
                           virDomainJobInfoPtr info);

int
virDomainGetJobInfoWrapper(virDomainPtr domain,
                           virDomainJobInfoPtr info,
                           virErrorPtr err)
{
    int ret = -1;
    static virDomainGetJobInfoType virDomainGetJobInfoSymbol;
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
(*virDomainGetJobStatsType)(virDomainPtr domain,
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
    static virDomainGetJobStatsType virDomainGetJobStatsSymbol;
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
(*virDomainGetLaunchSecurityInfoType)(virDomainPtr domain,
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
    static virDomainGetLaunchSecurityInfoType virDomainGetLaunchSecurityInfoSymbol;
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
(*virDomainGetMaxMemoryType)(virDomainPtr domain);

unsigned long
virDomainGetMaxMemoryWrapper(virDomainPtr domain,
                             virErrorPtr err)
{
    unsigned long ret = 0;
    static virDomainGetMaxMemoryType virDomainGetMaxMemorySymbol;
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
(*virDomainGetMaxVcpusType)(virDomainPtr domain);

int
virDomainGetMaxVcpusWrapper(virDomainPtr domain,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainGetMaxVcpusType virDomainGetMaxVcpusSymbol;
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
(*virDomainGetMemoryParametersType)(virDomainPtr domain,
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
    static virDomainGetMemoryParametersType virDomainGetMemoryParametersSymbol;
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
(*virDomainGetMessagesType)(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags);

int
virDomainGetMessagesWrapper(virDomainPtr domain,
                            char *** msgs,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainGetMessagesType virDomainGetMessagesSymbol;
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
(*virDomainGetMetadataType)(virDomainPtr domain,
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
    static virDomainGetMetadataType virDomainGetMetadataSymbol;
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
(*virDomainGetNameType)(virDomainPtr domain);

const char *
virDomainGetNameWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    const char * ret = NULL;
    static virDomainGetNameType virDomainGetNameSymbol;
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
(*virDomainGetNumaParametersType)(virDomainPtr domain,
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
    static virDomainGetNumaParametersType virDomainGetNumaParametersSymbol;
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
(*virDomainGetOSTypeType)(virDomainPtr domain);

char *
virDomainGetOSTypeWrapper(virDomainPtr domain,
                          virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetOSTypeType virDomainGetOSTypeSymbol;
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
(*virDomainGetPerfEventsType)(virDomainPtr domain,
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
    static virDomainGetPerfEventsType virDomainGetPerfEventsSymbol;
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
(*virDomainGetSchedulerParametersType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams);

int
virDomainGetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int * nparams,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSchedulerParametersType virDomainGetSchedulerParametersSymbol;
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
(*virDomainGetSchedulerParametersFlagsType)(virDomainPtr domain,
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
    static virDomainGetSchedulerParametersFlagsType virDomainGetSchedulerParametersFlagsSymbol;
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
(*virDomainGetSchedulerTypeType)(virDomainPtr domain,
                                 int * nparams);

char *
virDomainGetSchedulerTypeWrapper(virDomainPtr domain,
                                 int * nparams,
                                 virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetSchedulerTypeType virDomainGetSchedulerTypeSymbol;
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
(*virDomainGetSecurityLabelType)(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel);

int
virDomainGetSecurityLabelWrapper(virDomainPtr domain,
                                 virSecurityLabelPtr seclabel,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSecurityLabelType virDomainGetSecurityLabelSymbol;
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
(*virDomainGetSecurityLabelListType)(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels);

int
virDomainGetSecurityLabelListWrapper(virDomainPtr domain,
                                     virSecurityLabelPtr * seclabels,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainGetSecurityLabelListType virDomainGetSecurityLabelListSymbol;
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
(*virDomainGetStateType)(virDomainPtr domain,
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
    static virDomainGetStateType virDomainGetStateSymbol;
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
(*virDomainGetTimeType)(virDomainPtr dom,
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
    static virDomainGetTimeType virDomainGetTimeSymbol;
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
(*virDomainGetUUIDType)(virDomainPtr domain,
                        unsigned char * uuid);

int
virDomainGetUUIDWrapper(virDomainPtr domain,
                        unsigned char * uuid,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainGetUUIDType virDomainGetUUIDSymbol;
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
(*virDomainGetUUIDStringType)(virDomainPtr domain,
                              char * buf);

int
virDomainGetUUIDStringWrapper(virDomainPtr domain,
                              char * buf,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetUUIDStringType virDomainGetUUIDStringSymbol;
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
(*virDomainGetVcpuPinInfoType)(virDomainPtr domain,
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
    static virDomainGetVcpuPinInfoType virDomainGetVcpuPinInfoSymbol;
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
(*virDomainGetVcpusType)(virDomainPtr domain,
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
    static virDomainGetVcpusType virDomainGetVcpusSymbol;
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
(*virDomainGetVcpusFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainGetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainGetVcpusFlagsType virDomainGetVcpusFlagsSymbol;
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
(*virDomainGetXMLDescType)(virDomainPtr domain,
                           unsigned int flags);

char *
virDomainGetXMLDescWrapper(virDomainPtr domain,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = NULL;
    static virDomainGetXMLDescType virDomainGetXMLDescSymbol;
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
(*virDomainHasManagedSaveImageType)(virDomainPtr dom,
                                    unsigned int flags);

int
virDomainHasManagedSaveImageWrapper(virDomainPtr dom,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    int ret = -1;
    static virDomainHasManagedSaveImageType virDomainHasManagedSaveImageSymbol;
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
(*virDomainIOThreadInfoFreeType)(virDomainIOThreadInfoPtr info);

void
virDomainIOThreadInfoFreeWrapper(virDomainIOThreadInfoPtr info)
{

    static virDomainIOThreadInfoFreeType virDomainIOThreadInfoFreeSymbol;
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
(*virDomainInjectNMIType)(virDomainPtr domain,
                          unsigned int flags);

int
virDomainInjectNMIWrapper(virDomainPtr domain,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainInjectNMIType virDomainInjectNMISymbol;
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
(*virDomainInterfaceAddressesType)(virDomainPtr dom,
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
    static virDomainInterfaceAddressesType virDomainInterfaceAddressesSymbol;
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
(*virDomainInterfaceFreeType)(virDomainInterfacePtr iface);

void
virDomainInterfaceFreeWrapper(virDomainInterfacePtr iface)
{

    static virDomainInterfaceFreeType virDomainInterfaceFreeSymbol;
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
(*virDomainInterfaceStatsType)(virDomainPtr dom,
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
    static virDomainInterfaceStatsType virDomainInterfaceStatsSymbol;
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
(*virDomainIsActiveType)(virDomainPtr dom);

int
virDomainIsActiveWrapper(virDomainPtr dom,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainIsActiveType virDomainIsActiveSymbol;
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
(*virDomainIsPersistentType)(virDomainPtr dom);

int
virDomainIsPersistentWrapper(virDomainPtr dom,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainIsPersistentType virDomainIsPersistentSymbol;
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
(*virDomainIsUpdatedType)(virDomainPtr dom);

int
virDomainIsUpdatedWrapper(virDomainPtr dom,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainIsUpdatedType virDomainIsUpdatedSymbol;
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
(*virDomainListGetStatsType)(virDomainPtr * doms,
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
    static virDomainListGetStatsType virDomainListGetStatsSymbol;
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
(*virDomainLookupByIDType)(virConnectPtr conn,
                           int id);

virDomainPtr
virDomainLookupByIDWrapper(virConnectPtr conn,
                           int id,
                           virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByIDType virDomainLookupByIDSymbol;
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
(*virDomainLookupByNameType)(virConnectPtr conn,
                             const char * name);

virDomainPtr
virDomainLookupByNameWrapper(virConnectPtr conn,
                             const char * name,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByNameType virDomainLookupByNameSymbol;
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
(*virDomainLookupByUUIDType)(virConnectPtr conn,
                             const unsigned char * uuid);

virDomainPtr
virDomainLookupByUUIDWrapper(virConnectPtr conn,
                             const unsigned char * uuid,
                             virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByUUIDType virDomainLookupByUUIDSymbol;
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
(*virDomainLookupByUUIDStringType)(virConnectPtr conn,
                                   const char * uuidstr);

virDomainPtr
virDomainLookupByUUIDStringWrapper(virConnectPtr conn,
                                   const char * uuidstr,
                                   virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainLookupByUUIDStringType virDomainLookupByUUIDStringSymbol;
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
(*virDomainManagedSaveType)(virDomainPtr dom,
                            unsigned int flags);

int
virDomainManagedSaveWrapper(virDomainPtr dom,
                            unsigned int flags,
                            virErrorPtr err)
{
    int ret = -1;
    static virDomainManagedSaveType virDomainManagedSaveSymbol;
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
(*virDomainManagedSaveDefineXMLType)(virDomainPtr domain,
                                     const char * dxml,
                                     unsigned int flags);

int
virDomainManagedSaveDefineXMLWrapper(virDomainPtr domain,
                                     const char * dxml,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainManagedSaveDefineXMLType virDomainManagedSaveDefineXMLSymbol;
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
(*virDomainManagedSaveGetXMLDescType)(virDomainPtr domain,
                                      unsigned int flags);

char *
virDomainManagedSaveGetXMLDescWrapper(virDomainPtr domain,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    char * ret = NULL;
    static virDomainManagedSaveGetXMLDescType virDomainManagedSaveGetXMLDescSymbol;
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
(*virDomainManagedSaveRemoveType)(virDomainPtr dom,
                                  unsigned int flags);

int
virDomainManagedSaveRemoveWrapper(virDomainPtr dom,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainManagedSaveRemoveType virDomainManagedSaveRemoveSymbol;
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
(*virDomainMemoryPeekType)(virDomainPtr dom,
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
    static virDomainMemoryPeekType virDomainMemoryPeekSymbol;
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
(*virDomainMemoryStatsType)(virDomainPtr dom,
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
    static virDomainMemoryStatsType virDomainMemoryStatsSymbol;
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
(*virDomainMigrateType)(virDomainPtr domain,
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
    static virDomainMigrateType virDomainMigrateSymbol;
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
(*virDomainMigrate2Type)(virDomainPtr domain,
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
    static virDomainMigrate2Type virDomainMigrate2Symbol;
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
(*virDomainMigrate3Type)(virDomainPtr domain,
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
    static virDomainMigrate3Type virDomainMigrate3Symbol;
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
(*virDomainMigrateGetCompressionCacheType)(virDomainPtr domain,
                                           unsigned long long * cacheSize,
                                           unsigned int flags);

int
virDomainMigrateGetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long * cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateGetCompressionCacheType virDomainMigrateGetCompressionCacheSymbol;
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
(*virDomainMigrateGetMaxDowntimeType)(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags);

int
virDomainMigrateGetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long * downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateGetMaxDowntimeType virDomainMigrateGetMaxDowntimeSymbol;
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
(*virDomainMigrateGetMaxSpeedType)(virDomainPtr domain,
                                   unsigned long * bandwidth,
                                   unsigned int flags);

int
virDomainMigrateGetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long * bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateGetMaxSpeedType virDomainMigrateGetMaxSpeedSymbol;
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
(*virDomainMigrateSetCompressionCacheType)(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags);

int
virDomainMigrateSetCompressionCacheWrapper(virDomainPtr domain,
                                           unsigned long long cacheSize,
                                           unsigned int flags,
                                           virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateSetCompressionCacheType virDomainMigrateSetCompressionCacheSymbol;
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
(*virDomainMigrateSetMaxDowntimeType)(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags);

int
virDomainMigrateSetMaxDowntimeWrapper(virDomainPtr domain,
                                      unsigned long long downtime,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateSetMaxDowntimeType virDomainMigrateSetMaxDowntimeSymbol;
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
(*virDomainMigrateSetMaxSpeedType)(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags);

int
virDomainMigrateSetMaxSpeedWrapper(virDomainPtr domain,
                                   unsigned long bandwidth,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateSetMaxSpeedType virDomainMigrateSetMaxSpeedSymbol;
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
(*virDomainMigrateStartPostCopyType)(virDomainPtr domain,
                                     unsigned int flags);

int
virDomainMigrateStartPostCopyWrapper(virDomainPtr domain,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainMigrateStartPostCopyType virDomainMigrateStartPostCopySymbol;
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
(*virDomainMigrateToURIType)(virDomainPtr domain,
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
    static virDomainMigrateToURIType virDomainMigrateToURISymbol;
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
(*virDomainMigrateToURI2Type)(virDomainPtr domain,
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
    static virDomainMigrateToURI2Type virDomainMigrateToURI2Symbol;
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
(*virDomainMigrateToURI3Type)(virDomainPtr domain,
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
    static virDomainMigrateToURI3Type virDomainMigrateToURI3Symbol;
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
(*virDomainOpenChannelType)(virDomainPtr dom,
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
    static virDomainOpenChannelType virDomainOpenChannelSymbol;
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
(*virDomainOpenConsoleType)(virDomainPtr dom,
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
    static virDomainOpenConsoleType virDomainOpenConsoleSymbol;
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
(*virDomainOpenGraphicsType)(virDomainPtr dom,
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
    static virDomainOpenGraphicsType virDomainOpenGraphicsSymbol;
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
(*virDomainOpenGraphicsFDType)(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags);

int
virDomainOpenGraphicsFDWrapper(virDomainPtr dom,
                               unsigned int idx,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainOpenGraphicsFDType virDomainOpenGraphicsFDSymbol;
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
(*virDomainPMSuspendForDurationType)(virDomainPtr dom,
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
    static virDomainPMSuspendForDurationType virDomainPMSuspendForDurationSymbol;
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
(*virDomainPMWakeupType)(virDomainPtr dom,
                         unsigned int flags);

int
virDomainPMWakeupWrapper(virDomainPtr dom,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainPMWakeupType virDomainPMWakeupSymbol;
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
(*virDomainPinEmulatorType)(virDomainPtr domain,
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
    static virDomainPinEmulatorType virDomainPinEmulatorSymbol;
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
(*virDomainPinIOThreadType)(virDomainPtr domain,
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
    static virDomainPinIOThreadType virDomainPinIOThreadSymbol;
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
(*virDomainPinVcpuType)(virDomainPtr domain,
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
    static virDomainPinVcpuType virDomainPinVcpuSymbol;
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
(*virDomainPinVcpuFlagsType)(virDomainPtr domain,
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
    static virDomainPinVcpuFlagsType virDomainPinVcpuFlagsSymbol;
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
(*virDomainRebootType)(virDomainPtr domain,
                       unsigned int flags);

int
virDomainRebootWrapper(virDomainPtr domain,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainRebootType virDomainRebootSymbol;
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
(*virDomainRefType)(virDomainPtr domain);

int
virDomainRefWrapper(virDomainPtr domain,
                    virErrorPtr err)
{
    int ret = -1;
    static virDomainRefType virDomainRefSymbol;
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
(*virDomainRenameType)(virDomainPtr dom,
                       const char * new_name,
                       unsigned int flags);

int
virDomainRenameWrapper(virDomainPtr dom,
                       const char * new_name,
                       unsigned int flags,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainRenameType virDomainRenameSymbol;
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
(*virDomainResetType)(virDomainPtr domain,
                      unsigned int flags);

int
virDomainResetWrapper(virDomainPtr domain,
                      unsigned int flags,
                      virErrorPtr err)
{
    int ret = -1;
    static virDomainResetType virDomainResetSymbol;
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
(*virDomainRestoreType)(virConnectPtr conn,
                        const char * from);

int
virDomainRestoreWrapper(virConnectPtr conn,
                        const char * from,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainRestoreType virDomainRestoreSymbol;
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
(*virDomainRestoreFlagsType)(virConnectPtr conn,
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
    static virDomainRestoreFlagsType virDomainRestoreFlagsSymbol;
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
(*virDomainRestoreParamsType)(virConnectPtr conn,
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
    static virDomainRestoreParamsType virDomainRestoreParamsSymbol;
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
(*virDomainResumeType)(virDomainPtr domain);

int
virDomainResumeWrapper(virDomainPtr domain,
                       virErrorPtr err)
{
    int ret = -1;
    static virDomainResumeType virDomainResumeSymbol;
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
(*virDomainSaveType)(virDomainPtr domain,
                     const char * to);

int
virDomainSaveWrapper(virDomainPtr domain,
                     const char * to,
                     virErrorPtr err)
{
    int ret = -1;
    static virDomainSaveType virDomainSaveSymbol;
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
(*virDomainSaveFlagsType)(virDomainPtr domain,
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
    static virDomainSaveFlagsType virDomainSaveFlagsSymbol;
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
(*virDomainSaveImageDefineXMLType)(virConnectPtr conn,
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
    static virDomainSaveImageDefineXMLType virDomainSaveImageDefineXMLSymbol;
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
(*virDomainSaveImageGetXMLDescType)(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags);

char *
virDomainSaveImageGetXMLDescWrapper(virConnectPtr conn,
                                    const char * file,
                                    unsigned int flags,
                                    virErrorPtr err)
{
    char * ret = NULL;
    static virDomainSaveImageGetXMLDescType virDomainSaveImageGetXMLDescSymbol;
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
(*virDomainSaveParamsType)(virDomainPtr domain,
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
    static virDomainSaveParamsType virDomainSaveParamsSymbol;
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
(*virDomainScreenshotType)(virDomainPtr domain,
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
    static virDomainScreenshotType virDomainScreenshotSymbol;
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
(*virDomainSendKeyType)(virDomainPtr domain,
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
    static virDomainSendKeyType virDomainSendKeySymbol;
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
(*virDomainSendProcessSignalType)(virDomainPtr domain,
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
    static virDomainSendProcessSignalType virDomainSendProcessSignalSymbol;
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
(*virDomainSetAutostartType)(virDomainPtr domain,
                             int autostart);

int
virDomainSetAutostartWrapper(virDomainPtr domain,
                             int autostart,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainSetAutostartType virDomainSetAutostartSymbol;
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
(*virDomainSetBlkioParametersType)(virDomainPtr domain,
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
    static virDomainSetBlkioParametersType virDomainSetBlkioParametersSymbol;
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
(*virDomainSetBlockIoTuneType)(virDomainPtr dom,
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
    static virDomainSetBlockIoTuneType virDomainSetBlockIoTuneSymbol;
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
(*virDomainSetBlockThresholdType)(virDomainPtr domain,
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
    static virDomainSetBlockThresholdType virDomainSetBlockThresholdSymbol;
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
(*virDomainSetGuestVcpusType)(virDomainPtr domain,
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
    static virDomainSetGuestVcpusType virDomainSetGuestVcpusSymbol;
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
(*virDomainSetIOThreadParamsType)(virDomainPtr domain,
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
    static virDomainSetIOThreadParamsType virDomainSetIOThreadParamsSymbol;
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
(*virDomainSetInterfaceParametersType)(virDomainPtr domain,
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
    static virDomainSetInterfaceParametersType virDomainSetInterfaceParametersSymbol;
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
(*virDomainSetLaunchSecurityStateType)(virDomainPtr domain,
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
    static virDomainSetLaunchSecurityStateType virDomainSetLaunchSecurityStateSymbol;
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
(*virDomainSetLifecycleActionType)(virDomainPtr domain,
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
    static virDomainSetLifecycleActionType virDomainSetLifecycleActionSymbol;
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
(*virDomainSetMaxMemoryType)(virDomainPtr domain,
                             unsigned long memory);

int
virDomainSetMaxMemoryWrapper(virDomainPtr domain,
                             unsigned long memory,
                             virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMaxMemoryType virDomainSetMaxMemorySymbol;
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
(*virDomainSetMemoryType)(virDomainPtr domain,
                          unsigned long memory);

int
virDomainSetMemoryWrapper(virDomainPtr domain,
                          unsigned long memory,
                          virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryType virDomainSetMemorySymbol;
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
(*virDomainSetMemoryFlagsType)(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags);

int
virDomainSetMemoryFlagsWrapper(virDomainPtr domain,
                               unsigned long memory,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryFlagsType virDomainSetMemoryFlagsSymbol;
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
(*virDomainSetMemoryParametersType)(virDomainPtr domain,
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
    static virDomainSetMemoryParametersType virDomainSetMemoryParametersSymbol;
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
(*virDomainSetMemoryStatsPeriodType)(virDomainPtr domain,
                                     int period,
                                     unsigned int flags);

int
virDomainSetMemoryStatsPeriodWrapper(virDomainPtr domain,
                                     int period,
                                     unsigned int flags,
                                     virErrorPtr err)
{
    int ret = -1;
    static virDomainSetMemoryStatsPeriodType virDomainSetMemoryStatsPeriodSymbol;
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
(*virDomainSetMetadataType)(virDomainPtr domain,
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
    static virDomainSetMetadataType virDomainSetMetadataSymbol;
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
(*virDomainSetNumaParametersType)(virDomainPtr domain,
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
    static virDomainSetNumaParametersType virDomainSetNumaParametersSymbol;
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
(*virDomainSetPerfEventsType)(virDomainPtr domain,
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
    static virDomainSetPerfEventsType virDomainSetPerfEventsSymbol;
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
(*virDomainSetSchedulerParametersType)(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams);

int
virDomainSetSchedulerParametersWrapper(virDomainPtr domain,
                                       virTypedParameterPtr params,
                                       int nparams,
                                       virErrorPtr err)
{
    int ret = -1;
    static virDomainSetSchedulerParametersType virDomainSetSchedulerParametersSymbol;
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
(*virDomainSetSchedulerParametersFlagsType)(virDomainPtr domain,
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
    static virDomainSetSchedulerParametersFlagsType virDomainSetSchedulerParametersFlagsSymbol;
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
(*virDomainSetTimeType)(virDomainPtr dom,
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
    static virDomainSetTimeType virDomainSetTimeSymbol;
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
(*virDomainSetUserPasswordType)(virDomainPtr dom,
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
    static virDomainSetUserPasswordType virDomainSetUserPasswordSymbol;
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
(*virDomainSetVcpuType)(virDomainPtr domain,
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
    static virDomainSetVcpuType virDomainSetVcpuSymbol;
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
(*virDomainSetVcpusType)(virDomainPtr domain,
                         unsigned int nvcpus);

int
virDomainSetVcpusWrapper(virDomainPtr domain,
                         unsigned int nvcpus,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainSetVcpusType virDomainSetVcpusSymbol;
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
(*virDomainSetVcpusFlagsType)(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags);

int
virDomainSetVcpusFlagsWrapper(virDomainPtr domain,
                              unsigned int nvcpus,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainSetVcpusFlagsType virDomainSetVcpusFlagsSymbol;
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
(*virDomainShutdownType)(virDomainPtr domain);

int
virDomainShutdownWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainShutdownType virDomainShutdownSymbol;
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
(*virDomainShutdownFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainShutdownFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainShutdownFlagsType virDomainShutdownFlagsSymbol;
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
(*virDomainStartDirtyRateCalcType)(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags);

int
virDomainStartDirtyRateCalcWrapper(virDomainPtr domain,
                                   int seconds,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainStartDirtyRateCalcType virDomainStartDirtyRateCalcSymbol;
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
(*virDomainStatsRecordListFreeType)(virDomainStatsRecordPtr * stats);

void
virDomainStatsRecordListFreeWrapper(virDomainStatsRecordPtr * stats)
{

    static virDomainStatsRecordListFreeType virDomainStatsRecordListFreeSymbol;
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
(*virDomainSuspendType)(virDomainPtr domain);

int
virDomainSuspendWrapper(virDomainPtr domain,
                        virErrorPtr err)
{
    int ret = -1;
    static virDomainSuspendType virDomainSuspendSymbol;
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
(*virDomainUndefineType)(virDomainPtr domain);

int
virDomainUndefineWrapper(virDomainPtr domain,
                         virErrorPtr err)
{
    int ret = -1;
    static virDomainUndefineType virDomainUndefineSymbol;
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
(*virDomainUndefineFlagsType)(virDomainPtr domain,
                              unsigned int flags);

int
virDomainUndefineFlagsWrapper(virDomainPtr domain,
                              unsigned int flags,
                              virErrorPtr err)
{
    int ret = -1;
    static virDomainUndefineFlagsType virDomainUndefineFlagsSymbol;
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
(*virDomainUpdateDeviceFlagsType)(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags);

int
virDomainUpdateDeviceFlagsWrapper(virDomainPtr domain,
                                  const char * xml,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
    static virDomainUpdateDeviceFlagsType virDomainUpdateDeviceFlagsSymbol;
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
