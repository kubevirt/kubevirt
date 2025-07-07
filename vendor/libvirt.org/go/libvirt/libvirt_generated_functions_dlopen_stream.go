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
(*virStreamAbortFuncType)(virStreamPtr stream);

int
virStreamAbortWrapper(virStreamPtr stream,
                      virErrorPtr err)
{
    int ret = -1;
    static virStreamAbortFuncType virStreamAbortSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamAbort",
                       (void**)&virStreamAbortSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamAbortSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamEventAddCallbackFuncType)(virStreamPtr stream,
                                     int events,
                                     virStreamEventCallback cb,
                                     void * opaque,
                                     virFreeCallback ff);

int
virStreamEventAddCallbackWrapper(virStreamPtr stream,
                                 int events,
                                 virStreamEventCallback cb,
                                 void * opaque,
                                 virFreeCallback ff,
                                 virErrorPtr err)
{
    int ret = -1;
    static virStreamEventAddCallbackFuncType virStreamEventAddCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamEventAddCallback",
                       (void**)&virStreamEventAddCallbackSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamEventAddCallbackSymbol(stream,
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
(*virStreamEventRemoveCallbackFuncType)(virStreamPtr stream);

int
virStreamEventRemoveCallbackWrapper(virStreamPtr stream,
                                    virErrorPtr err)
{
    int ret = -1;
    static virStreamEventRemoveCallbackFuncType virStreamEventRemoveCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamEventRemoveCallback",
                       (void**)&virStreamEventRemoveCallbackSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamEventRemoveCallbackSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamEventUpdateCallbackFuncType)(virStreamPtr stream,
                                        int events);

int
virStreamEventUpdateCallbackWrapper(virStreamPtr stream,
                                    int events,
                                    virErrorPtr err)
{
    int ret = -1;
    static virStreamEventUpdateCallbackFuncType virStreamEventUpdateCallbackSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamEventUpdateCallback",
                       (void**)&virStreamEventUpdateCallbackSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamEventUpdateCallbackSymbol(stream,
                                             events);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamFinishFuncType)(virStreamPtr stream);

int
virStreamFinishWrapper(virStreamPtr stream,
                       virErrorPtr err)
{
    int ret = -1;
    static virStreamFinishFuncType virStreamFinishSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamFinish",
                       (void**)&virStreamFinishSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamFinishSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamFreeFuncType)(virStreamPtr stream);

int
virStreamFreeWrapper(virStreamPtr stream,
                     virErrorPtr err)
{
    int ret = -1;
    static virStreamFreeFuncType virStreamFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamFree",
                       (void**)&virStreamFreeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamFreeSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virStreamPtr
(*virStreamNewFuncType)(virConnectPtr conn,
                        unsigned int flags);

virStreamPtr
virStreamNewWrapper(virConnectPtr conn,
                    unsigned int flags,
                    virErrorPtr err)
{
    virStreamPtr ret = NULL;
    static virStreamNewFuncType virStreamNewSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamNew",
                       (void**)&virStreamNewSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamNewSymbol(conn,
                             flags);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvFuncType)(virStreamPtr stream,
                         char * data,
                         size_t nbytes);

int
virStreamRecvWrapper(virStreamPtr stream,
                     char * data,
                     size_t nbytes,
                     virErrorPtr err)
{
    int ret = -1;
    static virStreamRecvFuncType virStreamRecvSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamRecv",
                       (void**)&virStreamRecvSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamRecvSymbol(stream,
                              data,
                              nbytes);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvAllFuncType)(virStreamPtr stream,
                            virStreamSinkFunc handler,
                            void * opaque);

int
virStreamRecvAllWrapper(virStreamPtr stream,
                        virStreamSinkFunc handler,
                        void * opaque,
                        virErrorPtr err)
{
    int ret = -1;
    static virStreamRecvAllFuncType virStreamRecvAllSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamRecvAll",
                       (void**)&virStreamRecvAllSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamRecvAllSymbol(stream,
                                 handler,
                                 opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvFlagsFuncType)(virStreamPtr stream,
                              char * data,
                              size_t nbytes,
                              unsigned int flags);

int
virStreamRecvFlagsWrapper(virStreamPtr stream,
                          char * data,
                          size_t nbytes,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = -1;
    static virStreamRecvFlagsFuncType virStreamRecvFlagsSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamRecvFlags",
                       (void**)&virStreamRecvFlagsSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamRecvFlagsSymbol(stream,
                                   data,
                                   nbytes,
                                   flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRecvHoleFuncType)(virStreamPtr stream,
                             long long * length,
                             unsigned int flags);

int
virStreamRecvHoleWrapper(virStreamPtr stream,
                         long long * length,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virStreamRecvHoleFuncType virStreamRecvHoleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamRecvHole",
                       (void**)&virStreamRecvHoleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamRecvHoleSymbol(stream,
                                  length,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamRefFuncType)(virStreamPtr stream);

int
virStreamRefWrapper(virStreamPtr stream,
                    virErrorPtr err)
{
    int ret = -1;
    static virStreamRefFuncType virStreamRefSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamRef",
                       (void**)&virStreamRefSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamRefSymbol(stream);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSendFuncType)(virStreamPtr stream,
                         const char * data,
                         size_t nbytes);

int
virStreamSendWrapper(virStreamPtr stream,
                     const char * data,
                     size_t nbytes,
                     virErrorPtr err)
{
    int ret = -1;
    static virStreamSendFuncType virStreamSendSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamSend",
                       (void**)&virStreamSendSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamSendSymbol(stream,
                              data,
                              nbytes);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSendAllFuncType)(virStreamPtr stream,
                            virStreamSourceFunc handler,
                            void * opaque);

int
virStreamSendAllWrapper(virStreamPtr stream,
                        virStreamSourceFunc handler,
                        void * opaque,
                        virErrorPtr err)
{
    int ret = -1;
    static virStreamSendAllFuncType virStreamSendAllSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamSendAll",
                       (void**)&virStreamSendAllSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamSendAllSymbol(stream,
                                 handler,
                                 opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSendHoleFuncType)(virStreamPtr stream,
                             long long length,
                             unsigned int flags);

int
virStreamSendHoleWrapper(virStreamPtr stream,
                         long long length,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = -1;
    static virStreamSendHoleFuncType virStreamSendHoleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamSendHole",
                       (void**)&virStreamSendHoleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamSendHoleSymbol(stream,
                                  length,
                                  flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSparseRecvAllFuncType)(virStreamPtr stream,
                                  virStreamSinkFunc handler,
                                  virStreamSinkHoleFunc holeHandler,
                                  void * opaque);

int
virStreamSparseRecvAllWrapper(virStreamPtr stream,
                              virStreamSinkFunc handler,
                              virStreamSinkHoleFunc holeHandler,
                              void * opaque,
                              virErrorPtr err)
{
    int ret = -1;
    static virStreamSparseRecvAllFuncType virStreamSparseRecvAllSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamSparseRecvAll",
                       (void**)&virStreamSparseRecvAllSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamSparseRecvAllSymbol(stream,
                                       handler,
                                       holeHandler,
                                       opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virStreamSparseSendAllFuncType)(virStreamPtr stream,
                                  virStreamSourceFunc handler,
                                  virStreamSourceHoleFunc holeHandler,
                                  virStreamSourceSkipFunc skipHandler,
                                  void * opaque);

int
virStreamSparseSendAllWrapper(virStreamPtr stream,
                              virStreamSourceFunc handler,
                              virStreamSourceHoleFunc holeHandler,
                              virStreamSourceSkipFunc skipHandler,
                              void * opaque,
                              virErrorPtr err)
{
    int ret = -1;
    static virStreamSparseSendAllFuncType virStreamSparseSendAllSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virStreamSparseSendAll",
                       (void**)&virStreamSparseSendAllSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virStreamSparseSendAllSymbol(stream,
                                       handler,
                                       holeHandler,
                                       skipHandler,
                                       opaque);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
