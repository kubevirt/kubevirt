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

#ifndef LIBVIRT_GO_STORAGE_POOL_WRAPPER_H__
#define LIBVIRT_GO_STORAGE_POOL_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "storage_pool_compat.h"

int
virStoragePoolBuildWrapper(virStoragePoolPtr pool,
                           unsigned int flags,
                           virErrorPtr err);

int
virStoragePoolCreateWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err);

int
virStoragePoolDeleteWrapper(virStoragePoolPtr pool,
                            unsigned int flags,
                            virErrorPtr err);

int
virStoragePoolDestroyWrapper(virStoragePoolPtr pool,
                             virErrorPtr err);

int
virStoragePoolFreeWrapper(virStoragePoolPtr pool,
                          virErrorPtr err);

int
virStoragePoolGetAutostartWrapper(virStoragePoolPtr pool,
                                  int *autostart,
                                  virErrorPtr err);

int
virStoragePoolGetInfoWrapper(virStoragePoolPtr pool,
                             virStoragePoolInfoPtr info,
                             virErrorPtr err);

const char *
virStoragePoolGetNameWrapper(virStoragePoolPtr pool,
                             virErrorPtr err);

int
virStoragePoolGetUUIDWrapper(virStoragePoolPtr pool,
                             unsigned char *uuid,
                             virErrorPtr err);

int
virStoragePoolGetUUIDStringWrapper(virStoragePoolPtr pool,
                                   char *buf,
                                   virErrorPtr err);

char *
virStoragePoolGetXMLDescWrapper(virStoragePoolPtr pool,
                                unsigned int flags,
                                virErrorPtr err);

int
virStoragePoolIsActiveWrapper(virStoragePoolPtr pool,
                              virErrorPtr err);

int
virStoragePoolIsPersistentWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err);

int
virStoragePoolListAllVolumesWrapper(virStoragePoolPtr pool,
                                    virStorageVolPtr **vols,
                                    unsigned int flags,
                                    virErrorPtr err);

int
virStoragePoolListVolumesWrapper(virStoragePoolPtr pool,
                                 char **const names,
                                 int maxnames,
                                 virErrorPtr err);

int
virStoragePoolNumOfVolumesWrapper(virStoragePoolPtr pool,
                                  virErrorPtr err);

int
virStoragePoolRefWrapper(virStoragePoolPtr pool,
                         virErrorPtr err);

int
virStoragePoolRefreshWrapper(virStoragePoolPtr pool,
                             unsigned int flags,
                             virErrorPtr err);

int
virStoragePoolSetAutostartWrapper(virStoragePoolPtr pool,
                                  int autostart,
                                  virErrorPtr err);

int
virStoragePoolUndefineWrapper(virStoragePoolPtr pool,
                              virErrorPtr err);

virStorageVolPtr
virStorageVolCreateXMLWrapper(virStoragePoolPtr pool,
                              const char *xmlDesc,
                              unsigned int flags,
                              virErrorPtr err);

virStorageVolPtr
virStorageVolCreateXMLFromWrapper(virStoragePoolPtr pool,
                                  const char *xmlDesc,
                                  virStorageVolPtr clonevol,
                                  unsigned int flags,
                                  virErrorPtr err);

virStorageVolPtr
virStorageVolLookupByNameWrapper(virStoragePoolPtr pool,
                                 const char *name,
                                 virErrorPtr err);


#endif /* LIBVIRT_GO_STORAGE_POOL_WRAPPER_H__ */
