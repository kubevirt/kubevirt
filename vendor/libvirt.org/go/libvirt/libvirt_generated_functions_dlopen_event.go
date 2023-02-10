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
(*virEventAddHandleType)(int fd,
                         int events,
                         virEventHandleCallback cb,
                         void * opaque,
                         virFreeCallback ff);

int
virEventAddHandleWrapper(int fd,
                         int events,
                         virEventHandleCallback cb,
                         void * opaque,
                         virFreeCallback ff,
                         virErrorPtr err)
{
    int ret = -1;
    static virEventAddHandleType virEventAddHandleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventAddHandle",
                       (void**)&virEventAddHandleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virEventAddHandleSymbol(fd,
                                  events,
                                  cb,
                                  opaque,
                                  ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventAddTimeoutType)(int timeout,
                          virEventTimeoutCallback cb,
                          void * opaque,
                          virFreeCallback ff);

int
virEventAddTimeoutWrapper(int timeout,
                          virEventTimeoutCallback cb,
                          void * opaque,
                          virFreeCallback ff,
                          virErrorPtr err)
{
    int ret = -1;
    static virEventAddTimeoutType virEventAddTimeoutSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventAddTimeout",
                       (void**)&virEventAddTimeoutSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virEventAddTimeoutSymbol(timeout,
                                   cb,
                                   opaque,
                                   ff);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventRegisterDefaultImplType)(void);

int
virEventRegisterDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
    static virEventRegisterDefaultImplType virEventRegisterDefaultImplSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventRegisterDefaultImpl",
                       (void**)&virEventRegisterDefaultImplSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virEventRegisterDefaultImplSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virEventRegisterImplType)(virEventAddHandleFunc addHandle,
                            virEventUpdateHandleFunc updateHandle,
                            virEventRemoveHandleFunc removeHandle,
                            virEventAddTimeoutFunc addTimeout,
                            virEventUpdateTimeoutFunc updateTimeout,
                            virEventRemoveTimeoutFunc removeTimeout);

void
virEventRegisterImplWrapper(virEventAddHandleFunc addHandle,
                            virEventUpdateHandleFunc updateHandle,
                            virEventRemoveHandleFunc removeHandle,
                            virEventAddTimeoutFunc addTimeout,
                            virEventUpdateTimeoutFunc updateTimeout,
                            virEventRemoveTimeoutFunc removeTimeout)
{

    static virEventRegisterImplType virEventRegisterImplSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventRegisterImpl",
                       (void**)&virEventRegisterImplSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virEventRegisterImplSymbol(addHandle,
                               updateHandle,
                               removeHandle,
                               addTimeout,
                               updateTimeout,
                               removeTimeout);
}

typedef int
(*virEventRemoveHandleType)(int watch);

int
virEventRemoveHandleWrapper(int watch,
                            virErrorPtr err)
{
    int ret = -1;
    static virEventRemoveHandleType virEventRemoveHandleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventRemoveHandle",
                       (void**)&virEventRemoveHandleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virEventRemoveHandleSymbol(watch);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventRemoveTimeoutType)(int timer);

int
virEventRemoveTimeoutWrapper(int timer,
                             virErrorPtr err)
{
    int ret = -1;
    static virEventRemoveTimeoutType virEventRemoveTimeoutSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventRemoveTimeout",
                       (void**)&virEventRemoveTimeoutSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virEventRemoveTimeoutSymbol(timer);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virEventRunDefaultImplType)(void);

int
virEventRunDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
    static virEventRunDefaultImplType virEventRunDefaultImplSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventRunDefaultImpl",
                       (void**)&virEventRunDefaultImplSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virEventRunDefaultImplSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virEventUpdateHandleType)(int watch,
                            int events);

void
virEventUpdateHandleWrapper(int watch,
                            int events)
{

    static virEventUpdateHandleType virEventUpdateHandleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventUpdateHandle",
                       (void**)&virEventUpdateHandleSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virEventUpdateHandleSymbol(watch,
                               events);
}

typedef void
(*virEventUpdateTimeoutType)(int timer,
                             int timeout);

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout)
{

    static virEventUpdateTimeoutType virEventUpdateTimeoutSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virEventUpdateTimeout",
                       (void**)&virEventUpdateTimeoutSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virEventUpdateTimeoutSymbol(timer,
                                timeout);
}

*/
import "C"
