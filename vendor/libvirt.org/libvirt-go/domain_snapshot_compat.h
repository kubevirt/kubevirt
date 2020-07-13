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
 * Copyright (C) 2019 Red Hat, Inc.
 *
 */

#ifndef LIBVIRT_GO_DOMAIN_SNAPSHOT_COMPAT_H__
#define LIBVIRT_GO_DOMAIN_SNAPSHOT_COMPAT_H__

/* 5.2.0 */
#ifndef VIR_DOMAIN_SNAPSHOT_LIST_TOPOLOGICAL
# define VIR_DOMAIN_SNAPSHOT_LIST_TOPOLOGICAL (1 << 10)
#endif

/* 5.6.0 */

#ifndef VIR_DOMAIN_SNAPSHOT_CREATE_VALIDATE
# define VIR_DOMAIN_SNAPSHOT_CREATE_VALIDATE (1 << 9)
#endif

#endif /* LIBVIRT_GO_DOMAIN_SNAPSHOT_COMPAT_H__ */
