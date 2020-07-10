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
#include "node_device_wrapper.h"


int
virNodeDeviceDestroyWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = virNodeDeviceDestroy(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceDetachFlagsWrapper(virNodeDevicePtr dev,
                                const char *driverName,
                                unsigned int flags,
                                virErrorPtr err)
{
    int ret = virNodeDeviceDetachFlags(dev, driverName, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceDettachWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    int ret = virNodeDeviceDettach(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceFreeWrapper(virNodeDevicePtr dev,
                         virErrorPtr err)
{
    int ret = virNodeDeviceFree(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virNodeDeviceGetNameWrapper(virNodeDevicePtr dev,
                            virErrorPtr err)
{
    const char * ret = virNodeDeviceGetName(dev);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virNodeDeviceGetParentWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    const char * ret = virNodeDeviceGetParent(dev);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virNodeDeviceGetXMLDescWrapper(virNodeDevicePtr dev,
                               unsigned int flags,
                               virErrorPtr err)
{
    char * ret = virNodeDeviceGetXMLDesc(dev, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceListCapsWrapper(virNodeDevicePtr dev,
                             char ** const names,
                             int maxnames,
                             virErrorPtr err)
{
    int ret = virNodeDeviceListCaps(dev, names, maxnames);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceNumOfCapsWrapper(virNodeDevicePtr dev,
                              virErrorPtr err)
{
    int ret = virNodeDeviceNumOfCaps(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceReAttachWrapper(virNodeDevicePtr dev,
                             virErrorPtr err)
{
    int ret = virNodeDeviceReAttach(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceRefWrapper(virNodeDevicePtr dev,
                        virErrorPtr err)
{
    int ret = virNodeDeviceRef(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNodeDeviceResetWrapper(virNodeDevicePtr dev,
                          virErrorPtr err)
{
    int ret = virNodeDeviceReset(dev);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
