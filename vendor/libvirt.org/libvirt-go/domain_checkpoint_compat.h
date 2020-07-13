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

#ifndef LIBVIRT_GO_DOMAIN_CHECKPOINT_COMPAT_H__
#define LIBVIRT_GO_DOMAIN_CHECKPOINT_COMPAT_H__

/* 5.6.0 */

#if LIBVIR_VERSION_NUMBER < 5006000
typedef struct _virDomainCheckpoint *virDomainCheckpointPtr;
#endif


#ifndef VIR_DOMAIN_CHECKPOINT_CREATE_REDEFINE
# define VIR_DOMAIN_CHECKPOINT_CREATE_REDEFINE (1 << 0)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_CREATE_QUIESCE
# define VIR_DOMAIN_CHECKPOINT_CREATE_QUIESCE (1 << 1)
#endif



#ifndef VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN
# define VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN (1 << 0)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_DELETE_METADATA_ONLY
# define VIR_DOMAIN_CHECKPOINT_DELETE_METADATA_ONLY (1 << 1)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN_ONLY
# define VIR_DOMAIN_CHECKPOINT_DELETE_CHILDREN_ONLY (1 << 2)
#endif



#ifndef VIR_DOMAIN_CHECKPOINT_LIST_ROOTS
# define VIR_DOMAIN_CHECKPOINT_LIST_ROOTS (1 << 0)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_LIST_DESCENDANTS
# define VIR_DOMAIN_CHECKPOINT_LIST_DESCENDANTS (1 << 0)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_LIST_TOPOLOGICAL
# define VIR_DOMAIN_CHECKPOINT_LIST_TOPOLOGICAL (1 << 1)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_LIST_LEAVES
# define VIR_DOMAIN_CHECKPOINT_LIST_LEAVES (1 << 2)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_LIST_NO_LEAVES
# define VIR_DOMAIN_CHECKPOINT_LIST_NO_LEAVES (1 << 3)
#endif



#ifndef VIR_DOMAIN_CHECKPOINT_XML_SECURE
# define VIR_DOMAIN_CHECKPOINT_XML_SECURE (1 << 0)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_XML_NO_DOMAIN
# define VIR_DOMAIN_CHECKPOINT_XML_NO_DOMAIN (1 << 1)
#endif

#ifndef VIR_DOMAIN_CHECKPOINT_XML_SIZE
# define VIR_DOMAIN_CHECKPOINT_XML_SIZE (1 << 2)
#endif


#endif /* LIBVIRT_GO_DOMAIN_CHECKPOINT_COMPAT_H__ */
