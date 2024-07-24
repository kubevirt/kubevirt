//go:build !libvirt_without_lxc && libvirt_dlopen
// +build !libvirt_without_lxc,libvirt_dlopen

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
#cgo LDFLAGS: -ldl
#cgo CFLAGS: -DLIBVIRT_DLOPEN
#include <assert.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include "libvirt_lxc_generated_dlopen.h"
#include "error_helper.h"


typedef int
(*virDomainLxcEnterCGroupFuncType)(virDomainPtr domain,
                                   unsigned int flags);

int
virDomainLxcEnterCGroupWrapper(virDomainPtr domain,
                               unsigned int flags,
                               virErrorPtr err)
{
    int ret = -1;
    static virDomainLxcEnterCGroupFuncType virDomainLxcEnterCGroupSymbol;
    static bool once;
    static bool success;

    if (!libvirtLxcSymbol("virDomainLxcEnterCGroup",
                          (void**)&virDomainLxcEnterCGroupSymbol,
                          &once,
                          &success,
                          err)) {
        return ret;
    }
    ret = virDomainLxcEnterCGroupSymbol(domain,
                                        flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcEnterNamespaceFuncType)(virDomainPtr domain,
                                      unsigned int nfdlist,
                                      int * fdlist,
                                      unsigned int * noldfdlist,
                                      int ** oldfdlist,
                                      unsigned int flags);

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
    static virDomainLxcEnterNamespaceFuncType virDomainLxcEnterNamespaceSymbol;
    static bool once;
    static bool success;

    if (!libvirtLxcSymbol("virDomainLxcEnterNamespace",
                          (void**)&virDomainLxcEnterNamespaceSymbol,
                          &once,
                          &success,
                          err)) {
        return ret;
    }
    ret = virDomainLxcEnterNamespaceSymbol(domain,
                                           nfdlist,
                                           fdlist,
                                           noldfdlist,
                                           oldfdlist,
                                           flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcEnterSecurityLabelFuncType)(virSecurityModelPtr model,
                                          virSecurityLabelPtr label,
                                          virSecurityLabelPtr oldlabel,
                                          unsigned int flags);

int
virDomainLxcEnterSecurityLabelWrapper(virSecurityModelPtr model,
                                      virSecurityLabelPtr label,
                                      virSecurityLabelPtr oldlabel,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = -1;
    static virDomainLxcEnterSecurityLabelFuncType virDomainLxcEnterSecurityLabelSymbol;
    static bool once;
    static bool success;

    if (!libvirtLxcSymbol("virDomainLxcEnterSecurityLabel",
                          (void**)&virDomainLxcEnterSecurityLabelSymbol,
                          &once,
                          &success,
                          err)) {
        return ret;
    }
    ret = virDomainLxcEnterSecurityLabelSymbol(model,
                                               label,
                                               oldlabel,
                                               flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

typedef int
(*virDomainLxcOpenNamespaceFuncType)(virDomainPtr domain,
                                     int ** fdlist,
                                     unsigned int flags);

int
virDomainLxcOpenNamespaceWrapper(virDomainPtr domain,
                                 int ** fdlist,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = -1;
    static virDomainLxcOpenNamespaceFuncType virDomainLxcOpenNamespaceSymbol;
    static bool once;
    static bool success;

    if (!libvirtLxcSymbol("virDomainLxcOpenNamespace",
                          (void**)&virDomainLxcOpenNamespaceSymbol,
                          &once,
                          &success,
                          err)) {
        return ret;
    }
    ret = virDomainLxcOpenNamespaceSymbol(domain,
                                          fdlist,
                                          flags);
    if (ret < 0) {
        virCopyLastErrorWrapper(err);
    }
    return ret;
}

*/
import "C"
