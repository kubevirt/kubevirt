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
