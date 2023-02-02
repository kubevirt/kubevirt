//go:build !libvirt_without_qemu
// +build !libvirt_without_qemu

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
// Can't rely on pkg-config for libvirt-qemu since it was not
// installed until 2.6.0 onwards
#cgo !libvirt_dlopen LDFLAGS: -lvirt-qemu
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include <stdint.h>
#include "qemu_helper.h"
#include "callbacks_helper.h"


extern void domainQemuMonitorEventCallback(virConnectPtr, virDomainPtr, const char *, long long, unsigned int, const char *, int);
void domainQemuMonitorEventCallbackHelper(virConnectPtr conn, virDomainPtr dom,
					const char *event, long long secs,
					unsigned int micros, const char *details, void *data)
{
    domainQemuMonitorEventCallback(conn, dom, event, secs, micros, details, (int)(intptr_t)data);
}


int
virConnectDomainQemuMonitorEventRegisterHelper(virConnectPtr conn,
                                               virDomainPtr dom,
                                               const char *event,
                                               long goCallbackId,
                                               unsigned int flags,
                                               virErrorPtr err)
{
    void *id = (void *)goCallbackId;
    return virConnectDomainQemuMonitorEventRegisterWrapper(conn, dom, event,
                                                           domainQemuMonitorEventCallbackHelper,
                                                           id, freeGoCallbackHelper, flags, err);
}


*/
import "C"
