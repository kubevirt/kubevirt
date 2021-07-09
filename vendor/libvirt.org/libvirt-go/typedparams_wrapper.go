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
 * Copyright (C) 2019 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <assert.h>
#include "typedparams_wrapper.h"

int
virTypedParamsAddIntWrapper(virTypedParameterPtr *params,
			    int *nparams,
			    int *maxparams,
			    const char *name,
			    int value,
			    virErrorPtr err)
{
    int ret = virTypedParamsAddInt(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsAddUIntWrapper(virTypedParameterPtr *params,
			     int *nparams,
			     int *maxparams,
			     const char *name,
			     unsigned int value,
			     virErrorPtr err)
{
    int ret = virTypedParamsAddUInt(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsAddLLongWrapper(virTypedParameterPtr *params,
			      int *nparams,
			      int *maxparams,
			      const char *name,
			      long long value,
			      virErrorPtr err)
{
    int ret = virTypedParamsAddLLong(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsAddULLongWrapper(virTypedParameterPtr *params,
			       int *nparams,
			       int *maxparams,
			       const char *name,
			       unsigned long long value,
			       virErrorPtr err)
{
    int ret = virTypedParamsAddULLong(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsAddDoubleWrapper(virTypedParameterPtr *params,
			       int *nparams,
			       int *maxparams,
			       const char *name,
			       double value,
			       virErrorPtr err)
{
    int ret = virTypedParamsAddDouble(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsAddBooleanWrapper(virTypedParameterPtr *params,
				int *nparams,
				int *maxparams,
				const char *name,
				int value,
				virErrorPtr err)
{
    int ret = virTypedParamsAddBoolean(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsAddStringWrapper(virTypedParameterPtr *params,
			       int *nparams,
			       int *maxparams,
			       const char *name,
			       const char *value,
			       virErrorPtr err)
{
    int ret = virTypedParamsAddString(params, nparams, maxparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virTypedParamsGetIntWrapper(virTypedParameterPtr params,
			    int nparams,
			    const char *name,
			    int *value,
			    virErrorPtr err)
{
    int ret = virTypedParamsGetInt(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsGetUIntWrapper(virTypedParameterPtr params,
			     int nparams,
			     const char *name,
			     unsigned int *value,
			     virErrorPtr err)
{
    int ret = virTypedParamsGetUInt(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsGetLLongWrapper(virTypedParameterPtr params,
			      int nparams,
			      const char *name,
			      long long *value,
			      virErrorPtr err)
{
    int ret = virTypedParamsGetLLong(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsGetULLongWrapper(virTypedParameterPtr params,
			       int nparams,
			       const char *name,
			       unsigned long long *value,
			       virErrorPtr err)
{
    int ret = virTypedParamsGetULLong(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsGetDoubleWrapper(virTypedParameterPtr params,
			       int nparams,
			       const char *name,
			       double *value,
			       virErrorPtr err)
{
    int ret = virTypedParamsGetDouble(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsGetBooleanWrapper(virTypedParameterPtr params,
				int nparams,
				const char *name,
				int *value,
				virErrorPtr err)
{
    int ret = virTypedParamsGetBoolean(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}

int
virTypedParamsGetStringWrapper(virTypedParameterPtr params,
			       int nparams,
			       const char *name,
			       const char **value,
			       virErrorPtr err)
{
    int ret = virTypedParamsGetString(params, nparams, name, value);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}



*/
import "C"
