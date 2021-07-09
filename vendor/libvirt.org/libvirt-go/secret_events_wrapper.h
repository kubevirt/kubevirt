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

#ifndef LIBVIRT_GO_SECRET_EVENTS_WRAPPER_H__
#define LIBVIRT_GO_SECRET_EVENTS_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "secret_compat.h"

void
secretEventLifecycleCallbackHelper(virConnectPtr c,
                                   virSecretPtr d,
                                   int event,
                                   int detail,
                                   void* data);

void secretEventGenericCallbackHelper(virConnectPtr c,
                                      virSecretPtr d,
                                      void* data);

int
virConnectSecretEventRegisterAnyWrapper(virConnectPtr c,
                                        virSecretPtr d,
                                        int eventID,
                                        virConnectSecretEventGenericCallback cb,
                                        long goCallbackId,
					virErrorPtr err);

int
virConnectSecretEventDeregisterAnyWrapper(virConnectPtr conn,
                                          int callbackID,
					  virErrorPtr err);

#endif /* LIBVIRT_GO_SECRET_EVENTS_WRAPPER_H__ */
