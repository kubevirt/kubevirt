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
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virConnCopyLastError not available prior to libvirt version 0.1.0");
#else
    ret = virConnCopyLastError(conn,
                               to);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

virErrorPtr
virConnGetLastErrorWrapper(virConnectPtr conn,
                           virErrorPtr err)
{
    virErrorPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virConnGetLastError not available prior to libvirt version 0.1.0");
#else
    ret = virConnGetLastError(conn);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virConnResetLastErrorWrapper(virConnectPtr conn)
{

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virConnResetLastError not available prior to libvirt version 0.1.0");
#else
    virConnResetLastError(conn);
#endif
    return;
}

void
virConnSetErrorFuncWrapper(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler)
{

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virConnSetErrorFunc not available prior to libvirt version 0.1.0");
#else
    virConnSetErrorFunc(conn,
                        userData,
                        handler);
#endif
    return;
}

void
virDefaultErrorFuncWrapper(virErrorPtr err)
{

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virDefaultErrorFunc not available prior to libvirt version 0.1.0");
#else
    virDefaultErrorFunc(err);
#endif
    return;
}

void
virFreeErrorWrapper(virErrorPtr err)
{

#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(NULL, "Function virFreeError not available prior to libvirt version 0.6.1");
#else
    virFreeError(err);
#endif
    return;
}

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(err, "Function virGetLastError not available prior to libvirt version 0.1.0");
#else
    ret = virGetLastError();
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virGetLastErrorCodeWrapper(virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virGetLastErrorCode not available prior to libvirt version 4.5.0");
#else
    ret = virGetLastErrorCode();
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virGetLastErrorDomainWrapper(virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(4, 5, 0)
    setVirError(err, "Function virGetLastErrorDomain not available prior to libvirt version 4.5.0");
#else
    ret = virGetLastErrorDomain();
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

const char *
virGetLastErrorMessageWrapper(virErrorPtr err)
{
    const char * ret = NULL;
#if !LIBVIR_CHECK_VERSION(1, 0, 6)
    setVirError(err, "Function virGetLastErrorMessage not available prior to libvirt version 1.0.6");
#else
    ret = virGetLastErrorMessage();
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virResetErrorWrapper(virErrorPtr err)
{

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virResetError not available prior to libvirt version 0.1.0");
#else
    virResetError(err);
#endif
    return;
}

void
virResetLastErrorWrapper(void)
{

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virResetLastError not available prior to libvirt version 0.1.0");
#else
    virResetLastError();
#endif
    return;
}

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(0, 6, 1)
    setVirError(err, "Function virSaveLastError not available prior to libvirt version 0.6.1");
#else
    ret = virSaveLastError();
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virSetErrorFuncWrapper(void * userData,
                       virErrorFunc handler)
{

#if !LIBVIR_CHECK_VERSION(0, 1, 0)
    setVirError(NULL, "Function virSetErrorFunc not available prior to libvirt version 0.1.0");
#else
    virSetErrorFunc(userData,
                    handler);
#endif
    return;
}

*/
import "C"
