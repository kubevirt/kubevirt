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
#include "stream_helper.h"


int virGoStreamSourceCallback(virStreamPtr st, char *cdata, size_t nbytes, int callbackID);
int virGoStreamSourceHoleCallback(virStreamPtr st, int *inData, long long *length, int callbackID);
int virGoStreamSourceSkipCallback(virStreamPtr st, long long length, int callbackID);

int virGoStreamSinkCallback(virStreamPtr st, const char *cdata, size_t nbytes, int callbackID);
int virGoStreamSinkHoleCallback(virStreamPtr st, long long length, int callbackID);

struct CallbackData {
    int callbackID;
    int holeCallbackID;
    int skipCallbackID;
};


static int virGoStreamSourceCallbackHelper(virStreamPtr st, char *data, size_t nbytes, void *opaque)
{
    struct CallbackData *cbdata = opaque;

    return virGoStreamSourceCallback(st, data, nbytes, cbdata->callbackID);
}


static int virGoStreamSourceHoleCallbackHelper(virStreamPtr st, int *inData, long long *length, void *opaque)
{
    struct CallbackData *cbdata = opaque;

    return virGoStreamSourceHoleCallback(st, inData, length, cbdata->holeCallbackID);
}


static int virGoStreamSourceSkipCallbackHelper(virStreamPtr st, long long length, void *opaque)
{
    struct CallbackData *cbdata = opaque;

    return virGoStreamSourceSkipCallback(st, length, cbdata->skipCallbackID);
}


static int virGoStreamSinkCallbackHelper(virStreamPtr st, const char *data, size_t nbytes, void *opaque)
{
    struct CallbackData *cbdata = opaque;

    return virGoStreamSinkCallback(st, data, nbytes, cbdata->callbackID);
}


static int virGoStreamSinkHoleCallbackHelper(virStreamPtr st, long long length, void *opaque)
{
    struct CallbackData *cbdata = opaque;

    return virGoStreamSinkHoleCallback(st, length, cbdata->holeCallbackID);
}


void
virGoStreamEventCallback(virStreamPtr st, int events, int callbackID);

static void
virGoStreamEventCallbackHelper(virStreamPtr st, int events, void *opaque)
{
    virGoStreamEventCallback(st, events, (int)(intptr_t)opaque);
}


int
virStreamEventAddCallbackHelper(virStreamPtr stream,
                                int events,
                                int callbackID,
                                virErrorPtr err)
{
    return virStreamEventAddCallbackWrapper(stream, events, virGoStreamEventCallbackHelper,
                                            (void *)(intptr_t)callbackID, NULL, err);
}


int
virStreamRecvAllHelper(virStreamPtr stream,
                       int callbackID,
                       virErrorPtr err)
{
    struct CallbackData cbdata = { .callbackID = callbackID };
    return virStreamRecvAllWrapper(stream, virGoStreamSinkCallbackHelper, &cbdata, err);
}


int
virStreamSendAllHelper(virStreamPtr stream,
                       int callbackID,
                       virErrorPtr err)
{
    struct CallbackData cbdata = { .callbackID = callbackID };
    return virStreamSendAllWrapper(stream, virGoStreamSourceCallbackHelper, &cbdata, err);
}


int
virStreamSparseRecvAllHelper(virStreamPtr stream,
                             int callbackID,
                             int holeCallbackID,
                             virErrorPtr err)
{
    struct CallbackData cbdata = { .callbackID = callbackID,
                                   .holeCallbackID = holeCallbackID };
    return virStreamSparseRecvAllWrapper(stream, virGoStreamSinkCallbackHelper,
                                         virGoStreamSinkHoleCallbackHelper,
                                         &cbdata, err);
}


int
virStreamSparseSendAllHelper(virStreamPtr stream,
                             int callbackID,
                             int holeCallbackID,
                             int skipCallbackID,
                             virErrorPtr err)
{
    struct CallbackData cbdata = { .callbackID = callbackID,
                                   .holeCallbackID = holeCallbackID,
                                   .skipCallbackID = skipCallbackID };
    return virStreamSparseSendAllWrapper(stream, virGoStreamSourceCallbackHelper,
                                         virGoStreamSourceHoleCallbackHelper,
                                         virGoStreamSourceSkipCallbackHelper,
                                         &cbdata, err);
}


*/
import "C"
