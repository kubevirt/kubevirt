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
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

#ifndef LIBVIRT_GO_STORAGE_VOLUME_WRAPPER_H__
#define LIBVIRT_GO_STORAGE_VOLUME_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "storage_volume_compat.h"

virStoragePoolPtr
virStoragePoolLookupByVolumeWrapper(virStorageVolPtr vol,
                                    virErrorPtr err);

int
virStorageVolDeleteWrapper(virStorageVolPtr vol,
                           unsigned int flags,
                           virErrorPtr err);

int
virStorageVolDownloadWrapper(virStorageVolPtr vol,
                             virStreamPtr stream,
                             unsigned long long offset,
                             unsigned long long length,
                             unsigned int flags,
                             virErrorPtr err);

int
virStorageVolFreeWrapper(virStorageVolPtr vol,
                         virErrorPtr err);

virConnectPtr
virStorageVolGetConnectWrapper(virStorageVolPtr vol,
                               virErrorPtr err);

int
virStorageVolGetInfoWrapper(virStorageVolPtr vol,
                            virStorageVolInfoPtr info,
                            virErrorPtr err);

int
virStorageVolGetInfoFlagsWrapper(virStorageVolPtr vol,
                                 virStorageVolInfoPtr info,
                                 unsigned int flags,
                                 virErrorPtr err);

const char *
virStorageVolGetKeyWrapper(virStorageVolPtr vol,
                           virErrorPtr err);

const char *
virStorageVolGetNameWrapper(virStorageVolPtr vol,
                            virErrorPtr err);

char *
virStorageVolGetPathWrapper(virStorageVolPtr vol,
                            virErrorPtr err);

char *
virStorageVolGetXMLDescWrapper(virStorageVolPtr vol,
                               unsigned int flags,
                               virErrorPtr err);

int
virStorageVolRefWrapper(virStorageVolPtr vol,
                        virErrorPtr err);

int
virStorageVolResizeWrapper(virStorageVolPtr vol,
                           unsigned long long capacity,
                           unsigned int flags,
                           virErrorPtr err);

int
virStorageVolUploadWrapper(virStorageVolPtr vol,
                           virStreamPtr stream,
                           unsigned long long offset,
                           unsigned long long length,
                           unsigned int flags,
                           virErrorPtr err);

int
virStorageVolWipeWrapper(virStorageVolPtr vol,
                         unsigned int flags,
                         virErrorPtr err);

int
virStorageVolWipePatternWrapper(virStorageVolPtr vol,
                                unsigned int algorithm,
                                unsigned int flags,
                                virErrorPtr err);

#endif /* LIBVIRT_GO_STORAGE_VOLUME_WRAPPER_H__ */
