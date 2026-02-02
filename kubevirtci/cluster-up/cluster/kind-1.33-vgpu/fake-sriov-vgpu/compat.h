/* SPDX-License-Identifier: GPL-2.0 */
/*
 * Kernel compatibility header for fake-sriov-vgpu module
 *
 * This header provides compatibility shims for different kernel versions.
 */

#ifndef _FAKE_SRIOV_VGPU_COMPAT_H
#define _FAKE_SRIOV_VGPU_COMPAT_H

#include <linux/version.h>

/*
 * Minimum supported kernel version
 */
#if LINUX_VERSION_CODE < KERNEL_VERSION(5, 10, 0)
#error "This module requires kernel 5.10 or later"
#endif

/*
 * class_create() signature changed in 6.4
 * Before 6.4: class_create(owner, name)
 * After 6.4: class_create(name)
 */
#if LINUX_VERSION_CODE >= KERNEL_VERSION(6, 4, 0)
#define COMPAT_CLASS_CREATE(name) class_create(name)
#else
#define COMPAT_CLASS_CREATE(name) class_create(THIS_MODULE, name)
#endif

/*
 * strscpy was introduced in 4.3, but some older distributions
 * may not have it. Use strlcpy fallback if needed.
 */
#ifndef strscpy
#define strscpy(dest, src, size) strlcpy(dest, src, size)
#endif

#endif /* _FAKE_SRIOV_VGPU_COMPAT_H */
