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

package libvirt

/*
#cgo pkg-config: libvirt
// Can't rely on pkg-config for libvirt-lxc since it was not
// installed until 2.6.0 onwards
#cgo LDFLAGS: -lvirt-lxc
#include <assert.h>
#include "lxc_wrapper.h"

int
virDomainLxcEnterCGroupWrapper(virDomainPtr domain,
                               unsigned int flags,
                               virErrorPtr err)
{
#if LIBVIR_VERSION_NUMBER < 2000000
    assert(0); // Caller should have checked version
#else
    int ret = virDomainLxcEnterCGroup(domain, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
#endif
}


int
virDomainLxcEnterNamespaceWrapper(virDomainPtr domain,
                                  unsigned int nfdlist,
                                  int *fdlist,
                                  unsigned int *noldfdlist,
                                  int **oldfdlist,
                                  unsigned int flags,
                                  virErrorPtr err)
{
    int ret = virDomainLxcEnterNamespace(domain, nfdlist, fdlist, noldfdlist, oldfdlist, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainLxcEnterSecurityLabelWrapper(virSecurityModelPtr model,
                                      virSecurityLabelPtr label,
                                      virSecurityLabelPtr oldlabel,
                                      unsigned int flags,
                                      virErrorPtr err)
{
    int ret = virDomainLxcEnterSecurityLabel(model, label, oldlabel, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


int
virDomainLxcOpenNamespaceWrapper(virDomainPtr domain,
                                 int **fdlist,
                                 unsigned int flags,
                                 virErrorPtr err)
{
    int ret = virDomainLxcOpenNamespace(domain, fdlist, flags);
    if (ret < 0) {
        virCopyLastError(err);
    }
    return ret;
}


*/
import "C"
