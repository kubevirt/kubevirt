//go:build !libvirt_without_admin && libvirt_dlopen
// +build !libvirt_without_admin,libvirt_dlopen

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
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include <dlfcn.h>
#include <stdbool.h>
#include <stdio.h>
#include "libvirt_admin_generated_dlopen.h"
#include "error_helper.h"

static void *handle;
static bool once;

static void *
libvirtAdminLoad(virErrorPtr err)
{
    char *errMsg;

    if (once) {
        if (handle == NULL) {
            setVirError(err, "Failed to open libvirt-admin.so.0");
        }
        return handle;
    }
    handle = dlopen("libvirt-admin.so.0", RTLD_NOW|RTLD_LOCAL);
    once = true;
    if (handle == NULL) {
        setVirError(err, dlerror());
        return handle;
    }
    return handle;
}


bool
libvirtAdminSymbol(const char *name,
                   void **symbol,
                   bool *once,
                   bool *success,
                   virErrorPtr err)
{
    char *errMsg;

    if (!libvirtAdminLoad(err)) {
        return *success;
    }

    if (*once) {
        if (!*success) {
            // Set error for successive calls
            char msg[100];
            snprintf(msg, 100, "Failed to load %s", name);
            setVirError(err, msg);
        }
        return *success;
    }

    // Documentation of dlsym says we should use dlerror() to check for failure
    // in dlsym() as a NULL might be the right address for a given symbol.
    // This is also the reason to have the @success argument.
    *symbol = dlsym(handle, name);
    if ((errMsg = dlerror()) != NULL) {
        setVirError(err, errMsg);
        *once = true;
        return *success;
    }
    *once = true;
    *success = true;
    return *success;
}

*/
import "C"
