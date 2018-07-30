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
 * Copyright (C) 2018 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <assert.h>
#include "interface_wrapper.h"


int
virInterfaceCreateWrapper(virInterfacePtr iface,
                          unsigned int flags,
                          virErrorPtr err)
{
    int ret = virInterfaceCreate(iface, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceDestroyWrapper(virInterfacePtr iface,
                           unsigned int flags,
                           virErrorPtr err)
{
    int ret = virInterfaceDestroy(iface, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceFreeWrapper(virInterfacePtr iface,
                        virErrorPtr err)
{
    int ret = virInterfaceFree(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


virConnectPtr
virInterfaceGetConnectWrapper(virInterfacePtr iface,
                              virErrorPtr err)
{
    virConnectPtr ret = virInterfaceGetConnect(iface);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virInterfaceGetMACStringWrapper(virInterfacePtr iface,
                                virErrorPtr err)
{
    const char * ret = virInterfaceGetMACString(iface);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virInterfaceGetNameWrapper(virInterfacePtr iface,
                           virErrorPtr err)
{
    const char * ret = virInterfaceGetName(iface);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virInterfaceGetXMLDescWrapper(virInterfacePtr iface,
                              unsigned int flags,
                              virErrorPtr err)
{
    char * ret = virInterfaceGetXMLDesc(iface, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceIsActiveWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = virInterfaceIsActive(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceRefWrapper(virInterfacePtr iface,
                       virErrorPtr err)
{
    int ret = virInterfaceRef(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virInterfaceUndefineWrapper(virInterfacePtr iface,
                            virErrorPtr err)
{
    int ret = virInterfaceUndefine(iface);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
