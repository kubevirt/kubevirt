//go:build !libvirt_without_lxc && !libvirt_dlopen
// +build !libvirt_without_lxc,!libvirt_dlopen

/*
 * This file is part of the libvirt-go-module project
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
 * Copyright (C) 2022 Red Hat, Inc.
 *
 */
/****************************************************************************
 * THIS CODE HAS BEEN GENERATED. DO NOT CHANGE IT DIRECTLY                  *
 ****************************************************************************/

package libvirt

/*
#cgo pkg-config: libvirt-lxc
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_lxc_generated.h"
#include "error_helper.h"


int
virDomainLxcEnterCGroupWrapper(virDomainPtr domain,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(2, 0, 0)
    setVirError(err, "Function virDomainLxcEnterCGroup not available prior to libvirt version 2.0.0");
#else
    ret = virDomainLxcEnterCGroup(domain,
                                  flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainLxcEnterNamespaceWrapper(virDomainPtr domain,
                                  unsigned int nfdlist,
                                  int * fdlist,
                                  unsigned int * noldfdlist,
                                  int ** oldfdlist,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virDomainLxcEnterNamespace not available prior to libvirt version 1.0.2");
#else
    ret = virDomainLxcEnterNamespace(domain,
                                     nfdlist,
                                     fdlist,
                                     noldfdlist,
                                     oldfdlist,
                                     flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainLxcEnterSecurityLabelWrapper(virSecurityModelPtr model,
                                      virSecurityLabelPtr label,
                                      virSecurityLabelPtr oldlabel,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 4)
    setVirError(err, "Function virDomainLxcEnterSecurityLabel not available prior to libvirt version 1.0.4");
#else
    ret = virDomainLxcEnterSecurityLabel(model,
                                         label,
                                         oldlabel,
                                         flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

int
virDomainLxcOpenNamespaceWrapper(virDomainPtr domain,
                                 int ** fdlist,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
#if !LIBVIR_CHECK_VERSION(1, 0, 2)
    setVirError(err, "Function virDomainLxcOpenNamespace not available prior to libvirt version 1.0.2");
#else
    ret = virDomainLxcOpenNamespace(domain,
                                    fdlist,
                                    flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
#endif
    return ret;
}

*/
import "C"
