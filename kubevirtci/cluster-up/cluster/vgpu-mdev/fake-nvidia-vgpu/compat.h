/* SPDX-License-Identifier: GPL-2.0 */
/*
 * Kernel compatibility header for fake-nvidia-vgpu module
 *
 * This header provides compatibility shims for different kernel versions.
 *
 * RHEL/CentOS kernels may backport newer APIs while keeping older version
 * numbers. Building the module requires both the mdev_driver API and the
 * VFIO iommufd emulated-device helpers.
 */

#ifndef _FAKE_VGPU_COMPAT_H
#define _FAKE_VGPU_COMPAT_H

#include <linux/version.h>

/*
 * Use the new mdev_driver API
 * This is available in:
 * - Upstream kernels with the mdev_driver API
 * - Enterprise kernels that backport this API
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
 * Select the signature by kernel version or RHEL major version.
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
 * The exact upstream version and enterprise backports must be checked against
 * the target kernel headers; the module uses these helpers unconditionally.
 */

#include <linux/vfio.h>

#endif /* _FAKE_VGPU_COMPAT_H */
