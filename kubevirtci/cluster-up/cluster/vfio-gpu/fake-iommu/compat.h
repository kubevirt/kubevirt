/* SPDX-License-Identifier: GPL-2.0 */
/*
 * Kernel compatibility header for fake-iommu module.
 */

#ifndef _FAKE_IOMMU_COMPAT_H
#define _FAKE_IOMMU_COMPAT_H

#include <linux/version.h>

#if defined(__has_include)
#if __has_include(<linux/rhelversion.h>)
#include <linux/rhelversion.h>
#endif
#endif

#if LINUX_VERSION_CODE < KERNEL_VERSION(5, 14, 0)
#error "fake-iommu requires Linux kernel 5.14 or later"
#endif

#ifndef CONFIG_IOMMU_API
#error "fake-iommu requires CONFIG_IOMMU_API=y"
#endif

#if defined(RHEL_RELEASE_CODE) && defined(RHEL_RELEASE_VERSION) && \
	RHEL_RELEASE_CODE >= RHEL_RELEASE_VERSION(9, 0)
#define FAKE_IOMMU_HAS_DOMAIN_OPS 1
#define FAKE_IOMMU_CAPABLE_HAS_DEV 1
#define FAKE_IOMMU_HAS_PAGING_DOMAIN_ALLOC 1
#define FAKE_IOMMU_NO_FWSPEC_FREE 1
#define FAKE_IOMMU_FWSPEC_INIT_HAS_NO_OPS 1
#elif LINUX_VERSION_CODE < KERNEL_VERSION(6, 0, 0)
#define FAKE_IOMMU_LEGACY_OPS 1
#else
#define FAKE_IOMMU_HAS_DOMAIN_OPS 1
#define FAKE_IOMMU_CAPABLE_HAS_DEV 1
#endif

/*
 * CentOS Stream 10 / RHEL 10 carries newer IOMMU API changes on top of its
 * 6.12 kernel: paging domains are allocated through domain_alloc_paging(),
 * iommu_ops no longer exposes pgsize_bitmap, attach_dev receives the previous
 * domain, and iommu_fwspec_free() is no longer available to modules.
 */
#if LINUX_VERSION_CODE >= KERNEL_VERSION(6, 13, 0) || \
	(defined(RHEL_RELEASE_CODE) && defined(RHEL_RELEASE_VERSION) && \
	 RHEL_RELEASE_CODE >= RHEL_RELEASE_VERSION(10, 0))
#define FAKE_IOMMU_HAS_PAGING_DOMAIN_ALLOC 1
#define FAKE_IOMMU_ATTACH_DEV_HAS_OLD_DOMAIN 1
#define FAKE_IOMMU_NO_FWSPEC_FREE 1
#endif

/*
 * iommu_fwspec_init() dropped the ops argument in 6.11; the core now
 * resolves ops via iommu_ops_from_fwnode(), which requires the IOMMU to
 * already be registered for that fwnode.
 */
#if defined(FAKE_IOMMU_FWSPEC_INIT_HAS_NO_OPS) || LINUX_VERSION_CODE >= KERNEL_VERSION(6, 11, 0)
#define fake_iommu_fwspec_init(dev, fwnode, ops) \
	iommu_fwspec_init((dev), (fwnode))
#else
#define fake_iommu_fwspec_init(dev, fwnode, ops) \
	iommu_fwspec_init((dev), (fwnode), (ops))
#endif

#endif /* _FAKE_IOMMU_COMPAT_H */
