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

#ifndef LIBVIRT_GO_NETWORK_WRAPPER_H__
#define LIBVIRT_GO_NETWORK_WRAPPER_H__

#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include "network_compat.h"

int
virNetworkCreateWrapper(virNetworkPtr network,
                        virErrorPtr err);

void
virNetworkDHCPLeaseFreeWrapper(virNetworkDHCPLeasePtr lease);

int
virNetworkDestroyWrapper(virNetworkPtr network,
                         virErrorPtr err);

int
virNetworkFreeWrapper(virNetworkPtr network,
                      virErrorPtr err);

int
virNetworkGetAutostartWrapper(virNetworkPtr network,
                              int *autostart,
                              virErrorPtr err);

char *
virNetworkGetBridgeNameWrapper(virNetworkPtr network,
                               virErrorPtr err);

virConnectPtr
virNetworkGetConnectWrapper(virNetworkPtr net,
                            virErrorPtr err);

int
virNetworkGetDHCPLeasesWrapper(virNetworkPtr network,
                               const char *mac,
                               virNetworkDHCPLeasePtr **leases,
                               unsigned int flags,
                               virErrorPtr err);

const char *
virNetworkGetNameWrapper(virNetworkPtr network,
                         virErrorPtr err);

int
virNetworkGetUUIDWrapper(virNetworkPtr network,
                         unsigned char *uuid,
                         virErrorPtr err);

int
virNetworkGetUUIDStringWrapper(virNetworkPtr network,
                               char *buf,
                               virErrorPtr err);

char *
virNetworkGetXMLDescWrapper(virNetworkPtr network,
                            unsigned int flags,
                            virErrorPtr err);

int
virNetworkIsActiveWrapper(virNetworkPtr net,
                          virErrorPtr err);

int
virNetworkIsPersistentWrapper(virNetworkPtr net,
                              virErrorPtr err);

int
virNetworkRefWrapper(virNetworkPtr network,
                     virErrorPtr err);

int
virNetworkSetAutostartWrapper(virNetworkPtr network,
                              int autostart,
                              virErrorPtr err);

int
virNetworkUndefineWrapper(virNetworkPtr network,
                          virErrorPtr err);

int
virNetworkUpdateWrapper(virNetworkPtr network,
                        unsigned int command,
                        unsigned int section,
                        int parentIndex,
                        const char *xml,
                        unsigned int flags,
                        virErrorPtr err);


#endif /* LIBVIRT_GO_NETWORK_WRAPPER_H__ */
