/*
 * This file is part of the libvirt-go project
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
#cgo pkg-config: libvirt
// Can't rely on pkg-config for libvirt-qemu since it was not
// installed until 2.6.0 onwards
#cgo LDFLAGS: -lvirt-qemu
#include <assert.h>
#include <stdint.h>
#include "qemu_wrapper.h"
#include "callbacks_wrapper.h"


extern void domainQemuMonitorEventCallback(virConnectPtr, virDomainPtr, const char *, long long, unsigned int, const char *, int);
void domainQemuMonitorEventCallbackHelper(virConnectPtr c, virDomainPtr d,
					const char *event, long long secs,
					unsigned int micros, const char *details, void *data)
{
    domainQemuMonitorEventCallback(c, d, event, secs, micros, details, (int)(intptr_t)data);
}


int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002003
    assert(0); // Caller should have checked version
#else
    int ret = virConnectDomainQemuMonitorEventDeregister(conn, callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virConnectDomainQemuMonitorEventRegisterWrapper(virConnectPtr conn,
                                                virDomainPtr dom,
                                                const char *event,
                                                long goCallbackId,
                                                unsigned int flags,
                                                virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 1002003
    assert(0); // Caller should have checked version
#else
    void *id = (void*)goCallbackId;
    int ret = virConnectDomainQemuMonitorEventRegister(conn, dom, event, domainQemuMonitorEventCallbackHelper,
                                                       id, freeGoCallbackHelper, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


char *
virDomainQemuAgentCommandWrapper(virDomainPtr domain,
                                 const char *cmd,
                                 int timeout,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    char * ret = virDomainQemuAgentCommand(domain, cmd, timeout, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


virDomainPtr
virDomainQemuAttachWrapper(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags,
                           virErrorPtr err)
{
    virDomainPtr ret = virDomainQemuAttach(conn, pid_value, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainQemuMonitorCommandWrapper(virDomainPtr domain,
                                   const char *cmd,
                                   char **result,
                                   unsigned int flags,
                                   virErrorPtr err)
{
    int ret = virDomainQemuMonitorCommand(domain, cmd, result, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
