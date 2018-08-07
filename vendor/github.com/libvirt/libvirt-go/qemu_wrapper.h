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

#ifndef LIBVIRT_GO_DOMAIN_EVENTS_WRAPPER_H__
#define LIBVIRT_GO_DOMAIN_EVENTS_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/libvirt-qemu.h>
#include <libvirt/virterror.h>
#include "qemu_compat.h"

void
domainQemuMonitorEventCallbackHelper(virConnectPtr c,
                                     virDomainPtr d,
                                     const char *event,
                                     long long secs,
                                     unsigned int micros,
                                     const char *details,
                                     void *data);

int
virConnectDomainQemuMonitorEventDeregisterWrapper(virConnectPtr conn,
                                                  int callbackID,
                                                  virErrorPtr err);

int
virConnectDomainQemuMonitorEventRegisterWrapper(virConnectPtr conn,
                                                virDomainPtr dom,
                                                const char *event,
                                                long goCallbackId,
                                                unsigned int flags,
                                                virErrorPtr err);

char *
virDomainQemuAgentCommandWrapper(virDomainPtr domain,
                                 const char *cmd,
                                 int timeout,
                                 unsigned int flags,
                                 virErrorPtr err);

virDomainPtr
virDomainQemuAttachWrapper(virConnectPtr conn,
                           unsigned int pid_value,
                           unsigned int flags,
                           virErrorPtr err);

int
virDomainQemuMonitorCommandWrapper(virDomainPtr domain,
                                   const char *cmd,
                                   char **result,
                                   unsigned int flags,
                                   virErrorPtr err);


#endif /* LIBVIRT_GO_DOMAIN_EVENTS_WRAPPER_H__ */
