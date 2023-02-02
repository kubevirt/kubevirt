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
(*virConnCopyLastErrorType)(virConnectPtr conn,
                            virErrorPtr to);

int
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err)
{
    int ret = -1;
    static virConnCopyLastErrorType virConnCopyLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnCopyLastError",
                       (void**)&virConnCopyLastErrorSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnCopyLastErrorSymbol(conn,
                                     to);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef virErrorPtr
(*virConnGetLastErrorType)(virConnectPtr conn);

virErrorPtr
virConnGetLastErrorWrapper(virConnectPtr conn,
                           virErrorPtr err)
{
    virErrorPtr ret = NULL;
    static virConnGetLastErrorType virConnGetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnGetLastError",
                       (void**)&virConnGetLastErrorSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virConnGetLastErrorSymbol(conn);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virConnResetLastErrorType)(virConnectPtr conn);

void
virConnResetLastErrorWrapper(virConnectPtr conn)
{

    static virConnResetLastErrorType virConnResetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnResetLastError",
                       (void**)&virConnResetLastErrorSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virConnResetLastErrorSymbol(conn);
}

typedef void
(*virConnSetErrorFuncType)(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler);

void
virConnSetErrorFuncWrapper(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler)
{

    static virConnSetErrorFuncType virConnSetErrorFuncSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virConnSetErrorFunc",
                       (void**)&virConnSetErrorFuncSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virConnSetErrorFuncSymbol(conn,
                              userData,
                              handler);
}

typedef int
(*virCopyLastErrorType)(virErrorPtr to);

int
virCopyLastErrorWrapper(virErrorPtr to)
{
    int ret = -1;
    static virCopyLastErrorType virCopyLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virCopyLastError",
                       (void**)&virCopyLastErrorSymbol,
                       &once,
                       &success,
                       NULL)) {
        return ret;
    }
    ret = virCopyLastErrorSymbol(to);
    if (ret < 0) {
	setVirError(to, "Failed to copy last error");
    }
    return ret;
}

typedef void
(*virDefaultErrorFuncType)(virErrorPtr err);

void
virDefaultErrorFuncWrapper(virErrorPtr err)
{

    static virDefaultErrorFuncType virDefaultErrorFuncSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virDefaultErrorFunc",
                       (void**)&virDefaultErrorFuncSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virDefaultErrorFuncSymbol(err);
}

typedef void
(*virFreeErrorType)(virErrorPtr err);

void
virFreeErrorWrapper(virErrorPtr err)
{

    static virFreeErrorType virFreeErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virFreeError",
                       (void**)&virFreeErrorSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virFreeErrorSymbol(err);
}

typedef virErrorPtr
(*virGetLastErrorType)(void);

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
    static virGetLastErrorType virGetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virGetLastError",
                       (void**)&virGetLastErrorSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virGetLastErrorSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virGetLastErrorCodeType)(void);

int
virGetLastErrorCodeWrapper(virErrorPtr err)
{
    int ret = -1;
    static virGetLastErrorCodeType virGetLastErrorCodeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virGetLastErrorCode",
                       (void**)&virGetLastErrorCodeSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virGetLastErrorCodeSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virGetLastErrorDomainType)(void);

int
virGetLastErrorDomainWrapper(virErrorPtr err)
{
    int ret = -1;
    static virGetLastErrorDomainType virGetLastErrorDomainSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virGetLastErrorDomain",
                       (void**)&virGetLastErrorDomainSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virGetLastErrorDomainSymbol();
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef const char *
(*virGetLastErrorMessageType)(void);

const char *
virGetLastErrorMessageWrapper(virErrorPtr err)
{
    const char * ret = NULL;
    static virGetLastErrorMessageType virGetLastErrorMessageSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virGetLastErrorMessage",
                       (void**)&virGetLastErrorMessageSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virGetLastErrorMessageSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virResetErrorType)(virErrorPtr err);

void
virResetErrorWrapper(virErrorPtr err)
{

    static virResetErrorType virResetErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virResetError",
                       (void**)&virResetErrorSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virResetErrorSymbol(err);
}

typedef void
(*virResetLastErrorType)(void);

void
virResetLastErrorWrapper(void)
{

    static virResetLastErrorType virResetLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virResetLastError",
                       (void**)&virResetLastErrorSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virResetLastErrorSymbol();
}

typedef virErrorPtr
(*virSaveLastErrorType)(void);

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
    static virSaveLastErrorType virSaveLastErrorSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSaveLastError",
                       (void**)&virSaveLastErrorSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virSaveLastErrorSymbol();
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virSetErrorFuncType)(void * userData,
                       virErrorFunc handler);

void
virSetErrorFuncWrapper(void * userData,
                       virErrorFunc handler)
{

    static virSetErrorFuncType virSetErrorFuncSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virSetErrorFunc",
                       (void**)&virSetErrorFuncSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virSetErrorFuncSymbol(userData,
                          handler);
}

*/
import "C"
