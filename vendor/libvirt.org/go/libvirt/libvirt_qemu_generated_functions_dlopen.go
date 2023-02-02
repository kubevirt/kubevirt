//go:build !libvirt_without_qemu && libvirt_dlopen
// +build !libvirt_without_qemu,libvirt_dlopen

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
#include "libvirt_qemu_generated_dlopen.h"
#include "error_helper.h"


typedef int
(*virConnectDomainQemuMonitorEventDeregisterType)(virConnectPtr conn,
                                                  int callbackID);

int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainQemuMonitorEventDeregisterType virConnectDomainQemuMonitorEventDeregisterSymbol;
    static bool once;
    static bool success;

    if (!libvirtQemuSymbol("virConnectDomainQemuMonitorEventDeregister",
                           (void**)&virConnectDomainQemuMonitorEventDeregisterSymbol,
                           &once,
                           &success,
                           err)) {
        return ret;
    }
    ret = virConnectDomainQemuMonitorEventDeregisterSymbol(conn,
                                                           callbackID);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virConnectDomainQemuMonitorEventRegisterType)(virConnectPtr conn,
                                                virDomainPtr dom,
                                                const char * event,
                                                virConnectDomainQemuMonitorEventCallback cb,
                                                void * opaque,
                                                virFreeCallback freecb,
                                                unsigned int flags);

int
virConnectDomainQemuMonitorEventRegisterWrapper(virConnectPtr conn,
                                                virDomainPtr dom,
                                                const char * event,
                                                virConnectDomainQemuMonitorEventCallback cb,
                                                void * opaque,
                                                virFreeCallback freecb,
                                                unsigned int flags,
                                                virErrorPtr err)
{
    int ret = -1;
    static virConnectDomainQemuMonitorEventRegisterType virConnectDomainQemuMonitorEventRegisterSymbol;
    static bool once;
    static bool success;

    if (!libvirtQemuSymbol("virConnectDomainQemuMonitorEventRegister",
                           (void**)&virConnectDomainQemuMonitorEventRegisterSymbol,
                           &once,
                           &success,
                           err)) {
        return ret;
    }
    ret = virConnectDomainQemuMonitorEventRegisterSymbol(conn,
                                                         dom,
                                                         event,
                                                         cb,
                                                         opaque,
                                                         freecb,
                                                         flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef char *
(*virDomainQemuAgentCommandType)(virDomainPtr domain,
                                 const char * cmd,
                                 int timeout,
                                 unsigned int flags);

char *
virDomainQemuAgentCommandWrapper(virDomainPtr domain,
                                 const char * cmd,
                                 int timeout,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = NULL;
    static virDomainQemuAgentCommandType virDomainQemuAgentCommandSymbol;
    static bool once;
    static bool success;

    if (!libvirtQemuSymbol("virDomainQemuAgentCommand",
                           (void**)&virDomainQemuAgentCommandSymbol,
                           &once,
                           &success,
                           err)) {
        return ret;
    }
    ret = virDomainQemuAgentCommandSymbol(domain,
                                          cmd,
                                          timeout,
                                          flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virDomainPtr
(*virDomainQemuAttachType)(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags);

virDomainPtr
virDomainQemuAttachWrapper(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags,
                           virErrorPtr err)
{
    virDomainPtr ret = NULL;
    static virDomainQemuAttachType virDomainQemuAttachSymbol;
    static bool once;
    static bool success;

    if (!libvirtQemuSymbol("virDomainQemuAttach",
                           (void**)&virDomainQemuAttachSymbol,
                           &once,
                           &success,
                           err)) {
        return ret;
    }
    ret = virDomainQemuAttachSymbol(conn,
                                    pid_value,
                                    flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainQemuMonitorCommandType)(virDomainPtr domain,
                                   const char * cmd,
                                   char ** result,
                                   unsigned int flags);

int
virDomainQemuMonitorCommandWrapper(virDomainPtr domain,
                                   const char * cmd,
                                   char ** result,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
    static virDomainQemuMonitorCommandType virDomainQemuMonitorCommandSymbol;
    static bool once;
    static bool success;

    if (!libvirtQemuSymbol("virDomainQemuMonitorCommand",
                           (void**)&virDomainQemuMonitorCommandSymbol,
                           &once,
                           &success,
                           err)) {
        return ret;
    }
    ret = virDomainQemuMonitorCommandSymbol(domain,
                                            cmd,
                                            result,
                                            flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainQemuMonitorCommandWithFilesType)(virDomainPtr domain,
                                            const char * cmd,
                                            unsigned int ninfiles,
                                            int * infiles,
                                            unsigned int * noutfiles,
                                            int ** outfiles,
                                            char ** result,
                                            unsigned int flags);

int
virDomainQemuMonitorCommandWithFilesWrapper(virDomainPtr domain,
                                            const char * cmd,
                                            unsigned int ninfiles,
                                            int * infiles,
                                            unsigned int * noutfiles,
                                            int ** outfiles,
                                            char ** result,
                                            unsigned int flags,
                                            virErrorPtr err)
{
    int ret = -1;
    static virDomainQemuMonitorCommandWithFilesType virDomainQemuMonitorCommandWithFilesSymbol;
    static bool once;
    static bool success;

    if (!libvirtQemuSymbol("virDomainQemuMonitorCommandWithFiles",
                           (void**)&virDomainQemuMonitorCommandWithFilesSymbol,
                           &once,
                           &success,
                           err)) {
        return ret;
    }
    ret = virDomainQemuMonitorCommandWithFilesSymbol(domain,
                                                     cmd,
                                                     ninfiles,
                                                     infiles,
                                                     noutfiles,
                                                     outfiles,
                                                     result,
                                                     flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
