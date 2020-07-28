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
#include "network_port_wrapper.h"

virNetworkPtr
virNetworkPortGetNetworkWrapper(virNetworkPortPtr port,
				virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    virNetworkPtr ret;
    ret = virNetworkPortGetNetwork(port);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


char *
virNetworkPortGetXMLDescWrapper(virNetworkPortPtr port,
				unsigned int flags,
				virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    char *ret;
    ret = virNetworkPortGetXMLDesc(port, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortGetUUIDWrapper(virNetworkPortPtr port,
			     unsigned char *uuid,
			     virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortGetUUID(port, uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortGetUUIDStringWrapper(virNetworkPortPtr port,
				   char *buf,
				   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortGetUUIDString(port, buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortSetParametersWrapper(virNetworkPortPtr port,
				   virTypedParameterPtr params,
				   int nparams,
				   unsigned int flags,
				   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortSetParameters(port, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortGetParametersWrapper(virNetworkPortPtr port,
				   virTypedParameterPtr *params,
				   int *nparams,
				   unsigned int flags,
				   virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortGetParameters(port, params, nparams, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortDeleteWrapper(virNetworkPortPtr port,
			    unsigned int flags,
			    virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortDelete(port, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortFreeWrapper(virNetworkPortPtr port,
			  virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortFree(port);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virNetworkPortRefWrapper(virNetworkPortPtr port,
			 virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 5005000
    assert(0); // Caller should have checked version
#else
    int ret;
    ret = virNetworkPortRef(port);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


*/
import "C"
