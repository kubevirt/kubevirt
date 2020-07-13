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
#include "secret_wrapper.h"


int
virSecretFreeWrapper(virSecretPtr secret,
                     virErrorPtr err)
{
    int ret = virSecretFree(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virSecretGetUUIDWrapper(virSecretPtr secret,
                        unsigned char *uuid,
                        virErrorPtr err)
{
    int ret = virSecretGetUUID(secret, uuid);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virSecretGetUUIDStringWrapper(virSecretPtr secret,
                              char *buf,
                              virErrorPtr err)
{
    int ret = virSecretGetUUIDString(secret, buf);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


const char *
virSecretGetUsageIDWrapper(virSecretPtr secret,
                           virErrorPtr err)
{
    const char * ret = virSecretGetUsageID(secret);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virSecretGetUsageTypeWrapper(virSecretPtr secret,
                             virErrorPtr err)
{
    int ret = virSecretGetUsageType(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


unsigned char *
virSecretGetValueWrapper(virSecretPtr secret,
                         size_t *value_size,
                         unsigned int flags,
                         virErrorPtr err)
{
    unsigned char * ret = virSecretGetValue(secret, value_size, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


char *
virSecretGetXMLDescWrapper(virSecretPtr secret,
                           unsigned int flags,
                           virErrorPtr err)
{
    char * ret = virSecretGetXMLDesc(secret, flags);
    if (!ret) {
        virCopyLastError(err);
    }
    return ret;
}


int
virSecretRefWrapper(virSecretPtr secret,
                    virErrorPtr err)
{
    int ret = virSecretRef(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virSecretSetValueWrapper(virSecretPtr secret,
                         const unsigned char *value,
                         size_t value_size,
                         unsigned int flags,
                         virErrorPtr err)
{
    int ret = virSecretSetValue(secret, value, value_size, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virSecretUndefineWrapper(virSecretPtr secret,
                         virErrorPtr err)
{
    int ret = virSecretUndefine(secret);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
