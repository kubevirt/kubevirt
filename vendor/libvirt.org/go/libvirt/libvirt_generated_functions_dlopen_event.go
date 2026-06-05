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
(*virEventAddHandleFuncType)(int fd,
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
    static virEventAddHandleFuncType virEventAddHandleSymbol;
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
(*virEventAddTimeoutFuncType)(int timeout,
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
    static virEventAddTimeoutFuncType virEventAddTimeoutSymbol;
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
(*virEventRegisterDefaultImplFuncType)(void);

int
virEventRegisterDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
    static virEventRegisterDefaultImplFuncType virEventRegisterDefaultImplSymbol;
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
(*virEventRegisterImplFuncType)(virEventAddHandleFunc addHandle,
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

    static virEventRegisterImplFuncType virEventRegisterImplSymbol;
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
(*virEventRemoveHandleFuncType)(int watch);

int
virEventRemoveHandleWrapper(int watch,
                            virErrorPtr err)
{
    int ret = -1;
    static virEventRemoveHandleFuncType virEventRemoveHandleSymbol;
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
(*virEventRemoveTimeoutFuncType)(int timer);

int
virEventRemoveTimeoutWrapper(int timer,
                             virErrorPtr err)
{
    int ret = -1;
    static virEventRemoveTimeoutFuncType virEventRemoveTimeoutSymbol;
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
(*virEventRunDefaultImplFuncType)(void);

int
virEventRunDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
    static virEventRunDefaultImplFuncType virEventRunDefaultImplSymbol;
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
(*virEventUpdateHandleFuncType)(int watch,
                                int events);

void
virEventUpdateHandleWrapper(int watch,
                            int events)
{

    static virEventUpdateHandleFuncType virEventUpdateHandleSymbol;
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
(*virEventUpdateTimeoutFuncType)(int timer,
                                 int timeout);

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout)
{

    static virEventUpdateTimeoutFuncType virEventUpdateTimeoutSymbol;
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
