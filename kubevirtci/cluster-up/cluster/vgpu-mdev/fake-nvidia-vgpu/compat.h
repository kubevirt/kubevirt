/* SPDX-License-Identifier: GPL-2.0 */
/*
 * Kernel compatibility header for fake-nvidia-vgpu module
 *
 * This header provides compatibility shims for different kernel versions.
 *
 * Note: RHEL/CentOS kernels backport newer features while keeping older
 * version numbers. CentOS 9's 5.14 kernel has APIs from upstream 5.16+.
 * We use feature detection where possible instead of version checks.
 */

#ifndef _FAKE_VGPU_COMPAT_H
#define _FAKE_VGPU_COMPAT_H

#include <linux/version.h>

/*
 * Use the new mdev_driver API
 * This is available in:
 * - Upstream kernel 5.16+
 * - RHEL/CentOS 9 (5.14 with backports)
 *
 * For simplicity, we always use the new API since this module targets
 * modern enterprise kernels that have this API.
 */
#define USE_NEW_MDEV_API 1

/*
 * class_create() signature
 *
 * Newer kernels (6.4+ upstream, but also backported to CentOS 9):
 *   class_create(name)
 *
 * Older kernels:
 *   class_create(owner, name)
 *
 * We check for the actual function signature using a config test,
 * but for CentOS 9 the new signature is used.
 */
#if LINUX_VERSION_CODE >= KERNEL_VERSION(6, 4, 0) || \
    (defined(RHEL_MAJOR) && RHEL_MAJOR >= 9)
#define COMPAT_CLASS_CREATE(name) class_create(name)
#else
#define COMPAT_CLASS_CREATE(name) class_create(THIS_MODULE, name)
#endif

/*
 * PCI class defines
 */
#ifndef PCI_CLASS_DISPLAY_VGA
#define PCI_CLASS_DISPLAY_VGA 0x0300
#endif

#ifndef PCI_CLASS_DISPLAY_OTHER
#define PCI_CLASS_DISPLAY_OTHER 0x0380
#endif

/*
 * vfio_iommufd helpers are required by vfio_register_emulated_iommu_dev().
 * Available on upstream 6.2+ and RHEL/CentOS 9 (5.14 with backports).
 */

#include <linux/vfio.h>

#endif /* _FAKE_VGPU_COMPAT_H */
