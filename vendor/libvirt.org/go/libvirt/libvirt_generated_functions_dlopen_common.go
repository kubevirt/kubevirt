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
(*virTypedParamsAddBooleanFuncType)(virTypedParameterPtr * params,
                                    int * nparams,
                                    int * maxparams,
                                    const char * name,
                                    int value);

int
virTypedParamsAddBooleanWrapper(virTypedParameterPtr * params,
                                int * nparams,
                                int * maxparams,
                                const char * name,
                                int value,
                                virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddBooleanFuncType virTypedParamsAddBooleanSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddBoolean",
                       (void**)&virTypedParamsAddBooleanSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddBooleanSymbol(params,
                                         nparams,
                                         maxparams,
                                         name,
                                         value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddDoubleFuncType)(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   double value);

int
virTypedParamsAddDoubleWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               double value,
                               virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddDoubleFuncType virTypedParamsAddDoubleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddDouble",
                       (void**)&virTypedParamsAddDoubleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddDoubleSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddFromStringFuncType)(virTypedParameterPtr * params,
                                       int * nparams,
                                       int * maxparams,
                                       const char * name,
                                       int type,
                                       const char * value);

int
virTypedParamsAddFromStringWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   int type,
                                   const char * value,
                                   virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddFromStringFuncType virTypedParamsAddFromStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddFromString",
                       (void**)&virTypedParamsAddFromStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddFromStringSymbol(params,
                                            nparams,
                                            maxparams,
                                            name,
                                            type,
                                            value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddIntFuncType)(virTypedParameterPtr * params,
                                int * nparams,
                                int * maxparams,
                                const char * name,
                                int value);

int
virTypedParamsAddIntWrapper(virTypedParameterPtr * params,
                            int * nparams,
                            int * maxparams,
                            const char * name,
                            int value,
                            virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddIntFuncType virTypedParamsAddIntSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddInt",
                       (void**)&virTypedParamsAddIntSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddIntSymbol(params,
                                     nparams,
                                     maxparams,
                                     name,
                                     value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddLLongFuncType)(virTypedParameterPtr * params,
                                  int * nparams,
                                  int * maxparams,
                                  const char * name,
                                  long long value);

int
virTypedParamsAddLLongWrapper(virTypedParameterPtr * params,
                              int * nparams,
                              int * maxparams,
                              const char * name,
                              long long value,
                              virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddLLongFuncType virTypedParamsAddLLongSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddLLong",
                       (void**)&virTypedParamsAddLLongSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddLLongSymbol(params,
                                       nparams,
                                       maxparams,
                                       name,
                                       value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddStringFuncType)(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   const char * value);

int
virTypedParamsAddStringWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               const char * value,
                               virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddStringFuncType virTypedParamsAddStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddString",
                       (void**)&virTypedParamsAddStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddStringSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddStringListFuncType)(virTypedParameterPtr * params,
                                       int * nparams,
                                       int * maxparams,
                                       const char * name,
                                       const char ** values);

int
virTypedParamsAddStringListWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   const char ** values,
                                   virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddStringListFuncType virTypedParamsAddStringListSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddStringList",
                       (void**)&virTypedParamsAddStringListSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddStringListSymbol(params,
                                            nparams,
                                            maxparams,
                                            name,
                                            values);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddUIntFuncType)(virTypedParameterPtr * params,
                                 int * nparams,
                                 int * maxparams,
                                 const char * name,
                                 unsigned int value);

int
virTypedParamsAddUIntWrapper(virTypedParameterPtr * params,
                             int * nparams,
                             int * maxparams,
                             const char * name,
                             unsigned int value,
                             virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddUIntFuncType virTypedParamsAddUIntSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddUInt",
                       (void**)&virTypedParamsAddUIntSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddUIntSymbol(params,
                                      nparams,
                                      maxparams,
                                      name,
                                      value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsAddULLongFuncType)(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   unsigned long long value);

int
virTypedParamsAddULLongWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               unsigned long long value,
                               virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsAddULLongFuncType virTypedParamsAddULLongSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsAddULLong",
                       (void**)&virTypedParamsAddULLongSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsAddULLongSymbol(params,
                                        nparams,
                                        maxparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef void
(*virTypedParamsClearFuncType)(virTypedParameterPtr params,
                               int nparams);

void
virTypedParamsClearWrapper(virTypedParameterPtr params,
                           int nparams)
{

    static virTypedParamsClearFuncType virTypedParamsClearSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsClear",
                       (void**)&virTypedParamsClearSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virTypedParamsClearSymbol(params,
                              nparams);
}

typedef void
(*virTypedParamsFreeFuncType)(virTypedParameterPtr params,
                              int nparams);

void
virTypedParamsFreeWrapper(virTypedParameterPtr params,
                          int nparams)
{

    static virTypedParamsFreeFuncType virTypedParamsFreeSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsFree",
                       (void**)&virTypedParamsFreeSymbol,
                       &once,
                       &success,
                       NULL)) {
        return;
    }
    virTypedParamsFreeSymbol(params,
                             nparams);
}

typedef virTypedParameterPtr
(*virTypedParamsGetFuncType)(virTypedParameterPtr params,
                             int nparams,
                             const char * name);

virTypedParameterPtr
virTypedParamsGetWrapper(virTypedParameterPtr params,
                         int nparams,
                         const char * name,
                         virErrorPtr err)
{
    virTypedParameterPtr ret = NULL;
    static virTypedParamsGetFuncType virTypedParamsGetSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGet",
                       (void**)&virTypedParamsGetSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetSymbol(params,
                                  nparams,
                                  name);
    if (!ret) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetBooleanFuncType)(virTypedParameterPtr params,
                                    int nparams,
                                    const char * name,
                                    int * value);

int
virTypedParamsGetBooleanWrapper(virTypedParameterPtr params,
                                int nparams,
                                const char * name,
                                int * value,
                                virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetBooleanFuncType virTypedParamsGetBooleanSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetBoolean",
                       (void**)&virTypedParamsGetBooleanSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetBooleanSymbol(params,
                                         nparams,
                                         name,
                                         value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetDoubleFuncType)(virTypedParameterPtr params,
                                   int nparams,
                                   const char * name,
                                   double * value);

int
virTypedParamsGetDoubleWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               double * value,
                               virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetDoubleFuncType virTypedParamsGetDoubleSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetDouble",
                       (void**)&virTypedParamsGetDoubleSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetDoubleSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetIntFuncType)(virTypedParameterPtr params,
                                int nparams,
                                const char * name,
                                int * value);

int
virTypedParamsGetIntWrapper(virTypedParameterPtr params,
                            int nparams,
                            const char * name,
                            int * value,
                            virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetIntFuncType virTypedParamsGetIntSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetInt",
                       (void**)&virTypedParamsGetIntSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetIntSymbol(params,
                                     nparams,
                                     name,
                                     value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetLLongFuncType)(virTypedParameterPtr params,
                                  int nparams,
                                  const char * name,
                                  long long * value);

int
virTypedParamsGetLLongWrapper(virTypedParameterPtr params,
                              int nparams,
                              const char * name,
                              long long * value,
                              virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetLLongFuncType virTypedParamsGetLLongSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetLLong",
                       (void**)&virTypedParamsGetLLongSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetLLongSymbol(params,
                                       nparams,
                                       name,
                                       value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetStringFuncType)(virTypedParameterPtr params,
                                   int nparams,
                                   const char * name,
                                   const char ** value);

int
virTypedParamsGetStringWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               const char ** value,
                               virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetStringFuncType virTypedParamsGetStringSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetString",
                       (void**)&virTypedParamsGetStringSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetStringSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetUIntFuncType)(virTypedParameterPtr params,
                                 int nparams,
                                 const char * name,
                                 unsigned int * value);

int
virTypedParamsGetUIntWrapper(virTypedParameterPtr params,
                             int nparams,
                             const char * name,
                             unsigned int * value,
                             virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetUIntFuncType virTypedParamsGetUIntSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetUInt",
                       (void**)&virTypedParamsGetUIntSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetUIntSymbol(params,
                                      nparams,
                                      name,
                                      value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virTypedParamsGetULLongFuncType)(virTypedParameterPtr params,
                                   int nparams,
                                   const char * name,
                                   unsigned long long * value);

int
virTypedParamsGetULLongWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               unsigned long long * value,
                               virErrorPtr err)
{
    int ret = -1;
    static virTypedParamsGetULLongFuncType virTypedParamsGetULLongSymbol;
    static bool once;
    static bool success;

    if (!libvirtSymbol("virTypedParamsGetULLong",
                       (void**)&virTypedParamsGetULLongSymbol,
                       &once,
                       &success,
                       err)) {
        return ret;
    }
    ret = virTypedParamsGetULLongSymbol(params,
                                        nparams,
                                        name,
                                        value);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
