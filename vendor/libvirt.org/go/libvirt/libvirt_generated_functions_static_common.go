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
virTypedParamsAddBooleanWrapper(virTypedParameterPtr * params,
                                int * nparams,
                                int * maxparams,
                                const char * name,
                                int value,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddBoolean not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddBoolean(params,
                                   nparams,
                                   maxparams,
                                   name,
                                   value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddDoubleWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               double value,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddDouble not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddDouble(params,
                                  nparams,
                                  maxparams,
                                  name,
                                  value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

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
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddFromString not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddFromString(params,
                                      nparams,
                                      maxparams,
                                      name,
                                      type,
                                      value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddIntWrapper(virTypedParameterPtr * params,
                            int * nparams,
                            int * maxparams,
                            const char * name,
                            int value,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddInt not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddInt(params,
                               nparams,
                               maxparams,
                               name,
                               value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddLLongWrapper(virTypedParameterPtr * params,
                              int * nparams,
                              int * maxparams,
                              const char * name,
                              long long value,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddLLong not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddLLong(params,
                                 nparams,
                                 maxparams,
                                 name,
                                 value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddStringWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               const char * value,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddString not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddString(params,
                                  nparams,
                                  maxparams,
                                  name,
                                  value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddStringListWrapper(virTypedParameterPtr * params,
                                   int * nparams,
                                   int * maxparams,
                                   const char * name,
                                   const char ** values,
                                   virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 2, 17)
    setVirError(err, "Function virTypedParamsAddStringList not available prior to libvirt version 1.2.17");
#else
    ret = virTypedParamsAddStringList(params,
                                      nparams,
                                      maxparams,
                                      name,
                                      values);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddUIntWrapper(virTypedParameterPtr * params,
                             int * nparams,
                             int * maxparams,
                             const char * name,
                             unsigned int value,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddUInt not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddUInt(params,
                                nparams,
                                maxparams,
                                name,
                                value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsAddULLongWrapper(virTypedParameterPtr * params,
                               int * nparams,
                               int * maxparams,
                               const char * name,
                               unsigned long long value,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsAddULLong not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsAddULLong(params,
                                  nparams,
                                  maxparams,
                                  name,
                                  value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

void
virTypedParamsClearWrapper(virTypedParameterPtr params,
                           int nparams)
{

#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(NULL, "Function virTypedParamsClear not available prior to libvirt version 1.0.2");
#else
    virTypedParamsClear(params,
                        nparams);
#endif
    return;
}

void
virTypedParamsFreeWrapper(virTypedParameterPtr params,
                          int nparams)
{

#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(NULL, "Function virTypedParamsFree not available prior to libvirt version 1.0.2");
#else
    virTypedParamsFree(params,
                       nparams);
#endif
    return;
}

virTypedParameterPtr
virTypedParamsGetWrapper(virTypedParameterPtr params,
                         int nparams,
                         const char * name,
                         virErrorPtr err)
{
    virTypedParameterPtr ret = NULL;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGet not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGet(params,
                            nparams,
                            name);
    if (!ret) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetBooleanWrapper(virTypedParameterPtr params,
                                int nparams,
                                const char * name,
                                int * value,
                                virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetBoolean not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetBoolean(params,
                                   nparams,
                                   name,
                                   value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetDoubleWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               double * value,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetDouble not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetDouble(params,
                                  nparams,
                                  name,
                                  value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetIntWrapper(virTypedParameterPtr params,
                            int nparams,
                            const char * name,
                            int * value,
                            virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetInt not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetInt(params,
                               nparams,
                               name,
                               value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetLLongWrapper(virTypedParameterPtr params,
                              int nparams,
                              const char * name,
                              long long * value,
                              virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetLLong not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetLLong(params,
                                 nparams,
                                 name,
                                 value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetStringWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               const char ** value,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetString not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetString(params,
                                  nparams,
                                  name,
                                  value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetUIntWrapper(virTypedParameterPtr params,
                             int nparams,
                             const char * name,
                             unsigned int * value,
                             virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetUInt not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetUInt(params,
                                nparams,
                                name,
                                value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virTypedParamsGetULLongWrapper(virTypedParameterPtr params,
                               int nparams,
                               const char * name,
                               unsigned long long * value,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virTypedParamsGetULLong not available prior to libvirt version 1.0.2");
#else
    ret = virTypedParamsGetULLong(params,
                                  nparams,
                                  name,
                                  value);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
