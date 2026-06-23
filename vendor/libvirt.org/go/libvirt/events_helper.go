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
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include <stdint.h>
#include "events_helper.h"


void virGoEventHandleCallback(int watch, int fd, int events, int callbackID);

static void virGoEventAddHandleHelper(int watch, int fd, int events, void *opaque)
{
    virGoEventHandleCallback(watch, fd, events, (int)(intptr_t)opaque);
}


void virGoEventTimeoutCallback(int timer, int callbackID);

static void virGoEventAddTimeoutHelper(int timer, void *opaque)
{
    virGoEventTimeoutCallback(timer, (int)(intptr_t)opaque);
}


int virGoEventAddHandleFunc(int fd, int event, uintptr_t callback, uintptr_t opaque, uintptr_t freecb);
void virGoEventUpdateHandleFunc(int watch, int event);
int virGoEventRemoveHandleFunc(int watch);
int virGoEventAddTimeoutFunc(int freq, uintptr_t callback, uintptr_t opaque, uintptr_t freecb);
void virGoEventUpdateTimeoutFunc(int timer, int freq);
int virGoEventRemoveTimeoutFunc(int timer);


int virGoEventAddHandleFuncHelper(int fd, int event, virEventHandleCallback callback, void *opaque, virFreeCallback freecb)
{
    return virGoEventAddHandleFunc(fd, event, (uintptr_t)callback, (uintptr_t)opaque, (uintptr_t)freecb);
}


void virGoEventUpdateHandleFuncHelper(int watch, int event)
{
    virGoEventUpdateHandleFunc(watch, event);
}


int virGoEventRemoveHandleFuncHelper(int watch)
{
    return virGoEventRemoveHandleFunc(watch);
}


int virGoEventAddTimeoutFuncHelper(int freq, virEventTimeoutCallback callback, void *opaque, virFreeCallback freecb)
{
    return virGoEventAddTimeoutFunc(freq, (uintptr_t)callback, (uintptr_t)opaque, (uintptr_t)freecb);
}


void virGoEventUpdateTimeoutFuncHelper(int timer, int freq)
{
    virGoEventUpdateTimeoutFunc(timer, freq);
}


int virGoEventRemoveTimeoutFuncHelper(int timer)
{
    return virGoEventRemoveTimeoutFunc(timer);
}


void virEventRegisterImplHelper(void)
{
    virEventRegisterImplWrapper(virGoEventAddHandleFuncHelper,
                                virGoEventUpdateHandleFuncHelper,
                                virGoEventRemoveHandleFuncHelper,
                                virGoEventAddTimeoutFuncHelper,
                                virGoEventUpdateTimeoutFuncHelper,
                                virGoEventRemoveTimeoutFuncHelper);
}


void virGoEventHandleCallbackInvoke(int watch, int fd, int events, uintptr_t callback, uintptr_t opaque)
{
    ((virEventHandleCallback)callback)(watch, fd, events, (void *)opaque);
}


void virGoEventTimeoutCallbackInvoke(int timer, uintptr_t callback, uintptr_t opaque)
{
    ((virEventTimeoutCallback)callback)(timer, (void *)opaque);
}


void virGoEventHandleCallbackFree(uintptr_t callback, uintptr_t opaque)
{
    ((virFreeCallback)callback)((void *)opaque);
}


void virGoEventTimeoutCallbackFree(uintptr_t callback, uintptr_t opaque)
{
    ((virFreeCallback)callback)((void *)opaque);
}


int
virEventAddHandleHelper(int fd,
                        int events,
                        int callbackID,
                        virErrorPtr err)
{
    return virEventAddHandleWrapper(fd, events, virGoEventAddHandleHelper,
                                    (void *)(intptr_t)callbackID, NULL, err);
}


int
virEventAddTimeoutHelper(int timeout,
                         int callbackID,
                         virErrorPtr err)
{
    return virEventAddTimeoutWrapper(timeout, virGoEventAddTimeoutHelper,
                                     (void *)(intptr_t)callbackID, NULL, err);
}


*/
import "C"
