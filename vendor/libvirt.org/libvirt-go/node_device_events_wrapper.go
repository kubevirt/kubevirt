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
#include <assert.h>
#include <stdint.h>
#include "node_device_events_wrapper.h"
#include "callbacks_wrapper.h"

extern void nodeDeviceEventLifecycleCallback(virConnectPtr, virNodeDevicePtr, int, int, int);
void nodeDeviceEventLifecycleCallbackHelper(virConnectPtr c, virNodeDevicePtr d,
                                           int event, int detail, void *data)
{
    nodeDeviceEventLifecycleCallback(c, d, event, detail, (int)(intptr_t)data);
}

extern void nodeDeviceEventGenericCallback(virConnectPtr, virNodeDevicePtr, int);
void nodeDeviceEventGenericCallbackHelper(virConnectPtr c, virNodeDevicePtr d, void *data)
{
    nodeDeviceEventGenericCallback(c, d, (int)(intptr_t)data);
}


int
virConnectNodeDeviceEventRegisterAnyWrapper(virConnectPtr c,
                                            virNodeDevicePtr d,
                                            int eventID,
                                            virConnectNodeDeviceEventGenericCallback cb,
                                            long goCallbackId,
                                            virErrorPtr err)
{
    void* id = (void*)goCallbackId;
#if LIBVIR_VERSION_NUMBER < 2002000
    assert(0); // Caller should have checked version
#else
    int ret = virConnectNodeDeviceEventRegisterAny(c, d, eventID, cb, id, freeGoCallbackHelper);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virConnectNodeDeviceEventDeregisterAnyWrapper(virConnectPtr conn,
                                              int callbackID,
                                              virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 2002000
    assert(0); // Caller should have checked version
#else
    int ret = virConnectNodeDeviceEventDeregisterAny(conn, callbackID);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}

*/
import "C"
