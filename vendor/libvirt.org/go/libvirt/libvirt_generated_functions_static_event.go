//go:build !libvirt_dlopen
// +build !libvirt_dlopen

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
#cgo pkg-config: libvirt
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_generated.h"
#include "error_helper.h"


int
virEventAddHandleWrapper(int fd,
                         int events,
                         virEventHandleCallback cb,
                         void * opaque,
                         virFreeCallback ff,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventAddHandle not available prior to libvirt version 0.9.3");
#else
    ret = virEventAddHandle(fd,
                            events,
                            cb,
                            opaque,
                            ff);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virEventAddTimeoutWrapper(int timeout,
                          virEventTimeoutCallback cb,
                          void * opaque,
                          virFreeCallback ff,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventAddTimeout not available prior to libvirt version 0.9.3");
#else
    ret = virEventAddTimeout(timeout,
                             cb,
                             opaque,
                             ff);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virEventRegisterDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virEventRegisterDefaultImpl not available prior to libvirt version 0.9.0");
#else
    ret = virEventRegisterDefaultImpl();
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virEventRegisterImplWrapper(virEventAddHandleFunc addHandle,
                            virEventUpdateHandleFunc updateHandle,
                            virEventRemoveHandleFunc removeHandle,
                            virEventAddTimeoutFunc addTimeout,
                            virEventUpdateTimeoutFunc updateTimeout,
                            virEventRemoveTimeoutFunc removeTimeout)
{

#if !LIBVIR_CHECK_VERSION(0, 5, 0)
    setVirError(NULL, "Function virEventRegisterImpl not available prior to libvirt version 0.5.0");
#else
    virEventRegisterImpl(addHandle,
                         updateHandle,
                         removeHandle,
                         addTimeout,
                         updateTimeout,
                         removeTimeout);
#endif
    return;
}

int
virEventRemoveHandleWrapper(int watch,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventRemoveHandle not available prior to libvirt version 0.9.3");
#else
    ret = virEventRemoveHandle(watch);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virEventRemoveTimeoutWrapper(int timer,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(err, "Function virEventRemoveTimeout not available prior to libvirt version 0.9.3");
#else
    ret = virEventRemoveTimeout(timer);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virEventRunDefaultImplWrapper(virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 9, 0)
    setVirError(err, "Function virEventRunDefaultImpl not available prior to libvirt version 0.9.0");
#else
    ret = virEventRunDefaultImpl();
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virEventUpdateHandleWrapper(int watch,
                            int events)
{

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(NULL, "Function virEventUpdateHandle not available prior to libvirt version 0.9.3");
#else
    virEventUpdateHandle(watch,
                         events);
#endif
    return;
}

void
virEventUpdateTimeoutWrapper(int timer,
                             int timeout)
{

#if !LIBVIR_CHECK_VERSION(0, 9, 3)
    setVirError(NULL, "Function virEventUpdateTimeout not available prior to libvirt version 0.9.3");
#else
    virEventUpdateTimeout(timer,
                          timeout);
#endif
    return;
}

*/
import "C"
