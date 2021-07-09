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

#ifndef LIBVIRT_GO_TYPEDPARAMS_WRAPPER_H__
#define LIBVIRT_GO_TYPEDPARAMS_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>

int
virTypedParamsAddIntWrapper(virTypedParameterPtr *params,
			    int *nparams,
			    int *maxparams,
			    const char *name,
			    int value,
			    virErrorPtr err);
int
virTypedParamsAddUIntWrapper(virTypedParameterPtr *params,
			     int *nparams,
			     int *maxparams,
			     const char *name,
			     unsigned int value,
			     virErrorPtr err);
int
virTypedParamsAddLLongWrapper(virTypedParameterPtr *params,
			      int *nparams,
			      int *maxparams,
			      const char *name,
			      long long value,
			      virErrorPtr err);
int
virTypedParamsAddULLongWrapper(virTypedParameterPtr *params,
			       int *nparams,
			       int *maxparams,
			       const char *name,
			       unsigned long long value,
			       virErrorPtr err);
int
virTypedParamsAddDoubleWrapper(virTypedParameterPtr *params,
			       int *nparams,
			       int *maxparams,
			       const char *name,
			       double value,
			       virErrorPtr err);
int
virTypedParamsAddBooleanWrapper(virTypedParameterPtr *params,
				int *nparams,
				int *maxparams,
				const char *name,
				int value,
				virErrorPtr err);
int
virTypedParamsAddStringWrapper(virTypedParameterPtr *params,
			       int *nparams,
			       int *maxparams,
			       const char *name,
			       const char *value,
			       virErrorPtr err);

int
virTypedParamsGetIntWrapper(virTypedParameterPtr params,
			    int nparams,
			    const char *name,
			    int *value,
			    virErrorPtr err);
int
virTypedParamsGetUIntWrapper(virTypedParameterPtr params,
			     int nparams,
			     const char *name,
			     unsigned int *value,
			     virErrorPtr err);
int
virTypedParamsGetLLongWrapper(virTypedParameterPtr params,
			      int nparams,
			      const char *name,
			      long long *value,
			      virErrorPtr err);
int
virTypedParamsGetULLongWrapper(virTypedParameterPtr params,
			       int nparams,
			       const char *name,
			       unsigned long long *value,
			       virErrorPtr err);
int
virTypedParamsGetDoubleWrapper(virTypedParameterPtr params,
			       int nparams,
			       const char *name,
			       double *value,
			       virErrorPtr err);
int
virTypedParamsGetBooleanWrapper(virTypedParameterPtr params,
				int nparams,
				const char *name,
				int *value,
				virErrorPtr err);
int
virTypedParamsGetStringWrapper(virTypedParameterPtr params,
			       int nparams,
			       const char *name,
			       const char **value,
			       virErrorPtr err);


#endif /* LIBVIRT_GO_TYPEDPARAMS_WRAPPER_H__ */
