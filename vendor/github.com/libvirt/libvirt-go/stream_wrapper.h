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

#ifndef LIBVIRT_GO_STREAM_WRAPPER_H__
#define LIBVIRT_GO_STREAM_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "stream_compat.h"

int
virStreamAbortWrapper(virStreamPtr stream,
                      virErrorPtr err);

int
virStreamEventAddCallbackWrapper(virStreamPtr st,
                                 int events,
                                 int callbackID,
                                 virErrorPtr err);

int
virStreamEventRemoveCallbackWrapper(virStreamPtr stream,
                                    virErrorPtr err);

int
virStreamEventUpdateCallbackWrapper(virStreamPtr stream,
                                    int events,
                                    virErrorPtr err);

int
virStreamFinishWrapper(virStreamPtr stream,
                       virErrorPtr err);

int
virStreamFreeWrapper(virStreamPtr stream,
                     virErrorPtr err);

int
virStreamRecvWrapper(virStreamPtr stream,
                     char *data,
                     size_t nbytes,
                     virErrorPtr err);

int
virStreamRecvAllWrapper(virStreamPtr st,
                        int callbackID,
                        virErrorPtr err);

int
virStreamRecvFlagsWrapper(virStreamPtr st,
                          char *data,
                          size_t nbytes,
                          unsigned int flags,
                          virErrorPtr err);

int
virStreamRecvHoleWrapper(virStreamPtr,
                         long long *length,
                         unsigned int flags,
                         virErrorPtr err);

int
virStreamRefWrapper(virStreamPtr stream,
                    virErrorPtr err);

int
virStreamSendWrapper(virStreamPtr stream,
                     const char *data,
                     size_t nbytes,
                     virErrorPtr err);

int
virStreamSendAllWrapper(virStreamPtr st,
                        int callbackID,
                        virErrorPtr err);

int
virStreamSendHoleWrapper(virStreamPtr st,
                         long long length,
                         unsigned int flags,
                         virErrorPtr err);

int
virStreamSparseRecvAllWrapper(virStreamPtr st,
                              int callbackID,
                              int holeCallbackID,
                              virErrorPtr err);

int
virStreamSparseSendAllWrapper(virStreamPtr st,
                              int callbackID,
                              int holeCallbackID,
                              int skipCallbackID,
                              virErrorPtr err);


#endif /* LIBVIRT_GO_STREAM_WRAPPER_H__ */
