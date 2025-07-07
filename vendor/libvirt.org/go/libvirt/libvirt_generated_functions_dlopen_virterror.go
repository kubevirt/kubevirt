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
(*virConnCopyLastErrorFuncType)(virConnectPtr conn,
                                virErrorPtr to);

int
virConnCopyLastErrorWrapper(virConnectPtr conn,
                            virErrorPtr to,
                            virErrorPtr err)
{
    int ret = -1;
    static virConnCopyLastErrorFuncType virConnCopyLastErrorSymbol;
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
(*virConnGetLastErrorFuncType)(virConnectPtr conn);

virErrorPtr
virConnGetLastErrorWrapper(virConnectPtr conn,
                           virErrorPtr err)
{
    virErrorPtr ret = NULL;
    static virConnGetLastErrorFuncType virConnGetLastErrorSymbol;
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
(*virConnResetLastErrorFuncType)(virConnectPtr conn);

void
virConnResetLastErrorWrapper(virConnectPtr conn)
{

    static virConnResetLastErrorFuncType virConnResetLastErrorSymbol;
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
(*virConnSetErrorFuncFuncType)(virConnectPtr conn,
                               void * userData,
                               virErrorFunc handler);

void
virConnSetErrorFuncWrapper(virConnectPtr conn,
                           void * userData,
                           virErrorFunc handler)
{

    static virConnSetErrorFuncFuncType virConnSetErrorFuncSymbol;
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
(*virCopyLastErrorFuncType)(virErrorPtr to);

int
virCopyLastErrorWrapper(virErrorPtr to)
{
    int ret = -1;
    static virCopyLastErrorFuncType virCopyLastErrorSymbol;
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
(*virDefaultErrorFuncFuncType)(virErrorPtr err);

void
virDefaultErrorFuncWrapper(virErrorPtr err)
{

    static virDefaultErrorFuncFuncType virDefaultErrorFuncSymbol;
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
(*virFreeErrorFuncType)(virErrorPtr err);

void
virFreeErrorWrapper(virErrorPtr err)
{

    static virFreeErrorFuncType virFreeErrorSymbol;
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
(*virGetLastErrorFuncType)(void);

virErrorPtr
virGetLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
    static virGetLastErrorFuncType virGetLastErrorSymbol;
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
(*virGetLastErrorCodeFuncType)(void);

int
virGetLastErrorCodeWrapper(virErrorPtr err)
{
    int ret = -1;
    static virGetLastErrorCodeFuncType virGetLastErrorCodeSymbol;
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
(*virGetLastErrorDomainFuncType)(void);

int
virGetLastErrorDomainWrapper(virErrorPtr err)
{
    int ret = -1;
    static virGetLastErrorDomainFuncType virGetLastErrorDomainSymbol;
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
(*virGetLastErrorMessageFuncType)(void);

const char *
virGetLastErrorMessageWrapper(virErrorPtr err)
{
    const char * ret = NULL;
    static virGetLastErrorMessageFuncType virGetLastErrorMessageSymbol;
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
(*virResetErrorFuncType)(virErrorPtr err);

void
virResetErrorWrapper(virErrorPtr err)
{

    static virResetErrorFuncType virResetErrorSymbol;
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
(*virResetLastErrorFuncType)(void);

void
virResetLastErrorWrapper(void)
{

    static virResetLastErrorFuncType virResetLastErrorSymbol;
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
(*virSaveLastErrorFuncType)(void);

virErrorPtr
virSaveLastErrorWrapper(virErrorPtr err)
{
    virErrorPtr ret = NULL;
    static virSaveLastErrorFuncType virSaveLastErrorSymbol;
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
(*virSetErrorFuncFuncType)(void * userData,
                           virErrorFunc handler);

void
virSetErrorFuncWrapper(void * userData,
                       virErrorFunc handler)
{

    static virSetErrorFuncFuncType virSetErrorFuncSymbol;
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
