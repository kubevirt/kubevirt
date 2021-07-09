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
#include "nwfilter_wrapper.h"


int
virNWFilterFreeWrapper(virNWFilterPtr nwfilter,
                       virErrorPtr err)
{
    int ret = virNWFilterFree(nwfilter);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virNWFilterGetNameWrapper(virNWFilterPtr nwfilter,
                          virErrorPtr err)
{
    const char * ret = virNWFilterGetName(nwfilter);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNWFilterGetUUIDWrapper(virNWFilterPtr nwfilter,
                          unsigned char *uuid,
                          virErrorPtr err)
{
    int ret = virNWFilterGetUUID(nwfilter, uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNWFilterGetUUIDStringWrapper(virNWFilterPtr nwfilter,
                                char *buf,
                                virErrorPtr err)
{
    int ret = virNWFilterGetUUIDString(nwfilter, buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virNWFilterGetXMLDescWrapper(virNWFilterPtr nwfilter,
                             unsigned int flags,
                             virErrorPtr err)
{
    char * ret = virNWFilterGetXMLDesc(nwfilter, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNWFilterRefWrapper(virNWFilterPtr nwfilter,
                      virErrorPtr err)
{
    int ret = virNWFilterRef(nwfilter);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virNWFilterUndefineWrapper(virNWFilterPtr nwfilter,
                           virErrorPtr err)
{
    int ret = virNWFilterUndefine(nwfilter);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
