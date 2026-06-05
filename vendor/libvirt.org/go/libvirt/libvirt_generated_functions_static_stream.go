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
virStreamAbortWrapper(virStreamPtr stream,
                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamAbort not available prior to libvirt version 0.7.2");
#else
    ret = virStreamAbort(stream);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamEventAddCallbackWrapper(virStreamPtr stream,
                                 int events,
                                 virStreamEventCallback cb,
                                 void * opaque,
                                 virFreeCallback ff,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamEventAddCallback not available prior to libvirt version 0.7.2");
#else
    ret = virStreamEventAddCallback(stream,
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
virStreamEventRemoveCallbackWrapper(virStreamPtr stream,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamEventRemoveCallback not available prior to libvirt version 0.7.2");
#else
    ret = virStreamEventRemoveCallback(stream);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamEventUpdateCallbackWrapper(virStreamPtr stream,
                                    int events,
                                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamEventUpdateCallback not available prior to libvirt version 0.7.2");
#else
    ret = virStreamEventUpdateCallback(stream,
                                       events);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamFinishWrapper(virStreamPtr stream,
                       virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamFinish not available prior to libvirt version 0.7.2");
#else
    ret = virStreamFinish(stream);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamFreeWrapper(virStreamPtr stream,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamFree not available prior to libvirt version 0.7.2");
#else
    ret = virStreamFree(stream);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err)
{
    virStreamPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamNew not available prior to libvirt version 0.7.2");
#else
    ret = virStreamNew(conn,
                       flags);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamRecvWrapper(virStreamPtr stream,
                     char * data,
                     size_t nbytes,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamRecv not available prior to libvirt version 0.7.2");
#else
    ret = virStreamRecv(stream,
                        data,
                        nbytes);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamRecvAllWrapper(virStreamPtr stream,
                        virStreamSinkFunc handler,
                        void * opaque,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamRecvAll not available prior to libvirt version 0.7.2");
#else
    ret = virStreamRecvAll(stream,
                           handler,
                           opaque);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamRecvFlagsWrapper(virStreamPtr stream,
                          char * data,
                          size_t nbytes,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamRecvFlags not available prior to libvirt version 3.4.0");
#else
    ret = virStreamRecvFlags(stream,
                             data,
                             nbytes,
                             flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamRecvHoleWrapper(virStreamPtr stream,
                         long long * length,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamRecvHole not available prior to libvirt version 3.4.0");
#else
    ret = virStreamRecvHole(stream,
                            length,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamRefWrapper(virStreamPtr stream,
                    virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamRef not available prior to libvirt version 0.7.2");
#else
    ret = virStreamRef(stream);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamSendWrapper(virStreamPtr stream,
                     const char * data,
                     size_t nbytes,
                     virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamSend not available prior to libvirt version 0.7.2");
#else
    ret = virStreamSend(stream,
                        data,
                        nbytes);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamSendAllWrapper(virStreamPtr stream,
                        virStreamSourceFunc handler,
                        void * opaque,
                        virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 7, 2)
    setVirError(err, "Function virStreamSendAll not available prior to libvirt version 0.7.2");
#else
    ret = virStreamSendAll(stream,
                           handler,
                           opaque);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamSendHoleWrapper(virStreamPtr stream,
                         long long length,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamSendHole not available prior to libvirt version 3.4.0");
#else
    ret = virStreamSendHole(stream,
                            length,
                            flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamSparseRecvAllWrapper(virStreamPtr stream,
                              virStreamSinkFunc handler,
                              virStreamSinkHoleFunc holeHandler,
                              void * opaque,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamSparseRecvAll not available prior to libvirt version 3.4.0");
#else
    ret = virStreamSparseRecvAll(stream,
                                 handler,
                                 holeHandler,
                                 opaque);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virStreamSparseSendAllWrapper(virStreamPtr stream,
                              virStreamSourceFunc handler,
                              virStreamSourceHoleFunc holeHandler,
                              virStreamSourceSkipFunc skipHandler,
                              void * opaque,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(3, 4, 0)
    setVirError(err, "Function virStreamSparseSendAll not available prior to libvirt version 3.4.0");
#else
    ret = virStreamSparseSendAll(stream,
                                 handler,
                                 holeHandler,
                                 skipHandler,
                                 opaque);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
