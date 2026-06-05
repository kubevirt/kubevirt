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
#cgo !libvirt_dlopen pkg-config: libvirt-admin
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include "libvirt_admin_generated.h"
#include "callbacks_helper.h"

extern void admCloseCallback(virAdmConnectPtr, int, long);
void admCloseCallbackHelper(virAdmConnectPtr conn, int reason, void *opaque)
{
    admCloseCallback(conn, reason, (long)opaque);
}


int
virAdmConnectRegisterCloseCallbackHelper(virAdmConnectPtr conn,
                                         long goCallbackId,
                                         virErrorPtr err)
{
    void *id = (void *)goCallbackId;
    return virAdmConnectRegisterCloseCallbackWrapper(conn, admCloseCallbackHelper, id,
                                                     freeGoCallbackHelper, err);
}


int
virAdmConnectUnregisterCloseCallbackHelper(virAdmConnectPtr conn,
                                           virErrorPtr err)
{
    return virAdmConnectUnregisterCloseCallbackWrapper(conn, admCloseCallbackHelper, err);
}

*/
import "C"
