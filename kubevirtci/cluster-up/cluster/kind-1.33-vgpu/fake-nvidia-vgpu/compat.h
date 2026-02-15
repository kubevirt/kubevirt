/* SPDX-License-Identifier: GPL-2.0 */
/*
 * Kernel compatibility header for fake-nvidia-vgpu module
 *
 * This header provides compatibility shims for different kernel versions.
 * The mdev/VFIO API has changed significantly across kernel versions:
 *
 * - Kernel 5.16+: New vfio_device based API
 * - Kernel 5.11-5.15: Transitional API  
 * - Kernel 5.10 and earlier: Legacy mdev API
 *
 * This module targets kernel 5.16+ for simplicity.
 */

#ifndef _FAKE_VGPU_COMPAT_H
#define _FAKE_VGPU_COMPAT_H

#include <linux/version.h>

/*
 * Minimum supported kernel version
 * The new vfio_alloc_device API was introduced in 5.16
 */
#if LINUX_VERSION_CODE < KERNEL_VERSION(5, 16, 0)
#error "This module requires kernel 5.16 or later"
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
 * PCI class defines location changed
 */
#ifndef PCI_CLASS_DISPLAY_VGA
#define PCI_CLASS_DISPLAY_VGA 0x0300
#endif

#ifndef PCI_CLASS_DISPLAY_OTHER
#define PCI_CLASS_DISPLAY_OTHER 0x0380
#endif

/*
 * vfio_iommufd helpers - available since 6.2
 */
#if LINUX_VERSION_CODE < KERNEL_VERSION(6, 2, 0)
/* Older kernels use different bind functions */
#define vfio_iommufd_emulated_bind NULL
#define vfio_iommufd_emulated_unbind NULL
#define vfio_iommufd_emulated_attach_ioas NULL
#define vfio_iommufd_emulated_detach_ioas NULL
#endif

/*
 * VFIO GFX plane support - added in kernel 4.16
 * Define fallback structures/ioctls if not present
 *
 * Note: VFIO_TYPE and VFIO_BASE are defined in linux/vfio.h
 * VFIO_TYPE = ';' (0x3B)
 * VFIO_BASE = 100
 */
#include <linux/vfio.h>

#ifndef VFIO_GFX_PLANE_TYPE_PROBE
#define VFIO_GFX_PLANE_TYPE_PROBE       (1 << 0)
#define VFIO_GFX_PLANE_TYPE_DMABUF      (1 << 1)
#define VFIO_GFX_PLANE_TYPE_REGION      (1 << 2)
#endif

#ifndef VFIO_DEVICE_QUERY_GFX_PLANE

struct vfio_device_gfx_plane_info {
	__u32 argsz;
	__u32 flags;
	__u32 drm_format;
	__u64 drm_format_mod;
	__u32 width;
	__u32 height;
	__u32 stride;
	__u32 size;
	__u32 x_pos;
	__u32 y_pos;
	__u32 x_hot;
	__u32 y_hot;
	union {
		__u32 region_index;
		__u32 dmabuf_id;
	};
};

#define VFIO_DEVICE_QUERY_GFX_PLANE     _IO(VFIO_TYPE, VFIO_BASE + 14)

#endif /* VFIO_DEVICE_QUERY_GFX_PLANE */

#ifndef VFIO_DEVICE_GET_GFX_DMABUF
#define VFIO_DEVICE_GET_GFX_DMABUF      _IO(VFIO_TYPE, VFIO_BASE + 15)
#endif

#endif /* _FAKE_VGPU_COMPAT_H */
