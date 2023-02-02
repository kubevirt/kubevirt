//go:build !libvirt_without_qemu && !libvirt_dlopen
// +build !libvirt_without_qemu,!libvirt_dlopen

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
#cgo pkg-config: libvirt-qemu
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_qemu_generated.h"
#include "error_helper.h"


int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 3)
    setVirError(err, "Function virConnectDomainQemuMonitorEventDeregister not available prior to libvirt version 1.2.3");
#else
    ret = virConnectDomainQemuMonitorEventDeregister(conn,
                                                     callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(1, 2, 3)
    setVirError(err, "Function virConnectDomainQemuMonitorEventRegister not available prior to libvirt version 1.2.3");
#else
    ret = virConnectDomainQemuMonitorEventRegister(conn,
                                                   dom,
                                                   event,
                                                   cb,
                                                   opaque,
                                                   freecb,
                                                   flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

char *
virDomainQemuAgentCommandWrapper(virDomainPtr domain,
                                 const char * cmd,
                                 int timeout,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 10, 0)
    setVirError(err, "Function virDomainQemuAgentCommand not available prior to libvirt version 0.10.0");
#else
    ret = virDomainQemuAgentCommand(domain,
                                    cmd,
                                    timeout,
                                    flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virDomainPtr
virDomainQemuAttachWrapper(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags,
                           virErrorPtr err)
{
    virDomainPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 9, 4)
    setVirError(err, "Function virDomainQemuAttach not available prior to libvirt version 0.9.4");
#else
    ret = virDomainQemuAttach(conn,
                              pid_value,
                              flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainQemuMonitorCommandWrapper(virDomainPtr domain,
                                   const char * cmd,
                                   char ** result,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 8, 3)
    setVirError(err, "Function virDomainQemuMonitorCommand not available prior to libvirt version 0.8.3");
#else
    ret = virDomainQemuMonitorCommand(domain,
                                      cmd,
                                      result,
                                      flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(8, 2, 0)
    setVirError(err, "Function virDomainQemuMonitorCommandWithFiles not available prior to libvirt version 8.2.0");
#else
    ret = virDomainQemuMonitorCommandWithFiles(domain,
                                               cmd,
                                               ninfiles,
                                               infiles,
                                               noutfiles,
                                               outfiles,
                                               result,
                                               flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
