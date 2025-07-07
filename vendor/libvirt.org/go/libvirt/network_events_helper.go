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
#include "network_events_helper.h"
#include "callbacks_helper.h"

extern void networkEventLifecycleCallback(virConnectPtr, virNetworkPtr, int, int, int);
void networkEventLifecycleCallbackHelper(virConnectPtr conn, virNetworkPtr net,
                                     int event, int detail, void *data)
{
    networkEventLifecycleCallback(conn, net, event, detail, (int)(intptr_t)data);
}


extern void networkEventMetadataChangeCallback(virConnectPtr, virNetworkPtr, int, const char *, int);
void networkEventMetadataChangeCallbackHelper(virConnectPtr conn,
                        virNetworkPtr net,
                        int type,
                        const char *nsuri,
                        void *opaque)
{
    networkEventMetadataChangeCallback(conn, net, type, nsuri, (int)(intptr_t)opaque);
}


int
virConnectNetworkEventRegisterAnyHelper(virConnectPtr conn,
                                        virNetworkPtr net,
                                        int eventID,
                                        virConnectNetworkEventGenericCallback cb,
                                        long goCallbackId,
                                        virErrorPtr err)
{
    void *id = (void *)goCallbackId;
    return virConnectNetworkEventRegisterAnyWrapper(conn, net, eventID, cb, id,
                                                    freeGoCallbackHelper, err);
}


*/
import "C"
