// SPDX-License-Identifier: GPL-2.0
/*
 * Fake NVIDIA vGPU mediated device driver for KubeVirt testing
 *
 * This module creates fake mdev devices that simulate NVIDIA Tesla T4 vGPUs.
 * It provides the sysfs infrastructure and VFIO device emulation needed for
 * KubeVirt mdev tests to pass without real GPU hardware.
 *
 * Based on Linux kernel sample drivers (mdpy.c, mtty.c)
 *
 * The module creates:
 * - /sys/class/mdev_bus/<device>/mdev_supported_types/nvidia-222/ (GRID T4-1B)
 * - /sys/class/mdev_bus/<device>/mdev_supported_types/nvidia-223/ (GRID T4-2B)
 *
 * When mdev instances are passed to VMs, they appear as PCI devices with
 * NVIDIA vendor ID (10de) and Tesla T4 device ID (1eb8).
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/slab.h>
#include <linux/cdev.h>
#include <linux/device.h>
#include <linux/mdev.h>
#include <linux/pci.h>
#include <linux/vfio.h>
#include <linux/iommu.h>
#include <linux/uuid.h>
#include <linux/version.h>
#include <linux/vmalloc.h>
#include <linux/mm.h>

#include "compat.h"

/*
 * DRM format code for XRGB8888 (32-bit RGB with 8 bits per channel)
 * This is the standard fourcc code: fourcc_code('X', 'R', '2', '4')
 * We define it here to avoid dependency on drm_fourcc.h header
 */
#ifndef DRM_FORMAT_XRGB8888
#define DRM_FORMAT_XRGB8888 (('X') | (('R') << 8) | (('2') << 16) | (('4') << 24))
#endif

#define VERSION_STRING  "1.0"
#define DRIVER_AUTHOR   "KubeVirt Fake vGPU Driver"

/*
 * IMPORTANT: The driver name must be "nvidia" so that the mdev type
 * directories are named "nvidia-222" and "nvidia-223" (matching what
 * KubeVirt tests expect). The kernel mdev framework creates type
 * directories as <driver_name>-<type_sysfs_name>.
 */
#define FAKE_VGPU_NAME          "nvidia"
#define FAKE_VGPU_CLASS_NAME    "nvidia"

/* NVIDIA PCI IDs - Tesla T4 */
#define NVIDIA_VENDOR_ID        0x10de
#define NVIDIA_T4_DEVICE_ID     0x1eb8
#define NVIDIA_T4_SUBSYS_ID     0x12a2

/* PCI config space size */
#define FAKE_VGPU_CONFIG_SPACE_SIZE     256

/* Memory BAR configuration */
#define FAKE_VGPU_MEMORY_BAR_OFFSET     PAGE_SIZE
#define FAKE_VGPU_MEMORY_SIZE           (16 * 1024 * 1024)  /* 16MB fake VRAM */

/* Display configuration for QEMU ramfb/display support */
#define FAKE_VGPU_DISPLAY_WIDTH         1024
#define FAKE_VGPU_DISPLAY_HEIGHT        768
#define FAKE_VGPU_DISPLAY_BPP           4       /* 32-bit XRGB8888 */
#define FAKE_VGPU_DISPLAY_STRIDE        (FAKE_VGPU_DISPLAY_WIDTH * FAKE_VGPU_DISPLAY_BPP)
#define FAKE_VGPU_DISPLAY_SIZE          (FAKE_VGPU_DISPLAY_STRIDE * FAKE_VGPU_DISPLAY_HEIGHT)

/* Maximum number of mdev instances */
#define MAX_T4_1B_INSTANCES     16      /* nvidia-222: GRID T4-1B */
#define MAX_T4_2B_INSTANCES     8       /* nvidia-223: GRID T4-2B */

#define STORE_LE16(addr, val)   (*(u16 *)addr = cpu_to_le16(val))
#define STORE_LE32(addr, val)   (*(u32 *)addr = cpu_to_le32(val))

MODULE_DESCRIPTION("Fake NVIDIA vGPU driver for KubeVirt testing");
MODULE_LICENSE("GPL v2");
MODULE_VERSION(VERSION_STRING);
MODULE_AUTHOR(DRIVER_AUTHOR);

/* Global device structure */
static struct fake_vgpu_dev {
	dev_t devt;
	struct class *vgpu_class;
	struct cdev cdev;
	struct device dev;
	struct mdev_parent parent;
} fake_vgpu_dev;

/* vGPU type definitions matching NVIDIA GRID naming */
static struct fake_vgpu_type {
	struct mdev_type type;
	u32 max_instances;
	u32 fb_size;            /* Framebuffer size in MB */
	const char *profile;    /* Profile name */
} fake_vgpu_types[] = {
	{
		/*
		 * sysfs_name is "222" so the full directory becomes "nvidia-222"
		 * (kernel creates <driver_name>-<sysfs_name>)
		 */
		.type.sysfs_name = "222",
		.type.pretty_name = "GRID T4-1B",
		.max_instances = MAX_T4_1B_INSTANCES,
		.fb_size = 1024,        /* 1GB */
		.profile = "1b",
	},
	{
		.type.sysfs_name = "223",
		.type.pretty_name = "GRID T4-2B",
		.max_instances = MAX_T4_2B_INSTANCES,
		.fb_size = 2048,        /* 2GB */
		.profile = "2b",
	},
};

static struct mdev_type *fake_vgpu_mdev_types[] = {
	&fake_vgpu_types[0].type,
	&fake_vgpu_types[1].type,
};

/* Track available instances per type */
static atomic_t avail_instances[ARRAY_SIZE(fake_vgpu_types)];

/* Per-mdev device state */
struct mdev_state {
	struct vfio_device vdev;
	struct mdev_device *mdev;
	const struct fake_vgpu_type *type;
	int type_index;

	/* PCI config space */
	u8 *vconfig;
	u32 bar_mask;

	/* Fake VRAM */
	void *memblk;
	u32 memsize;

	struct mutex ops_lock;
	struct vfio_device_info dev_info;
};

static const struct vfio_device_ops fake_vgpu_dev_ops;

/*
 * Create PCI config space that presents as NVIDIA Tesla T4
 */
static void fake_vgpu_create_config_space(struct mdev_state *mdev_state)
{
	u8 *vconfig = mdev_state->vconfig;

	/* PCI header */
	STORE_LE16(&vconfig[PCI_VENDOR_ID], NVIDIA_VENDOR_ID);
	STORE_LE16(&vconfig[PCI_DEVICE_ID], NVIDIA_T4_DEVICE_ID);

	/* Command: Memory space enabled */
	STORE_LE16(&vconfig[PCI_COMMAND], PCI_COMMAND_MEMORY);

	/* Status: Capabilities list present */
	STORE_LE16(&vconfig[PCI_STATUS], PCI_STATUS_CAP_LIST);

	/* Revision ID */
	vconfig[PCI_REVISION_ID] = 0xa1;

	/* Class code: Display controller / VGA compatible / VGA */
	vconfig[PCI_CLASS_PROG] = 0x00;
	STORE_LE16(&vconfig[PCI_CLASS_DEVICE], PCI_CLASS_DISPLAY_VGA);

	/* Header type: Normal */
	vconfig[PCI_HEADER_TYPE] = PCI_HEADER_TYPE_NORMAL;

	/* BAR0: Memory, 32-bit, prefetchable */
	STORE_LE32(&vconfig[PCI_BASE_ADDRESS_0],
		   PCI_BASE_ADDRESS_SPACE_MEMORY |
		   PCI_BASE_ADDRESS_MEM_TYPE_32 |
		   PCI_BASE_ADDRESS_MEM_PREFETCH);
	mdev_state->bar_mask = ~(mdev_state->memsize - 1);

	/* Subsystem IDs */
	STORE_LE16(&vconfig[PCI_SUBSYSTEM_VENDOR_ID], NVIDIA_VENDOR_ID);
	STORE_LE16(&vconfig[PCI_SUBSYSTEM_ID], NVIDIA_T4_SUBSYS_ID);

	/* Capabilities pointer */
	vconfig[PCI_CAPABILITY_LIST] = 0x60;

	/* Interrupt pin */
	vconfig[PCI_INTERRUPT_PIN] = 0x01;

	/* Power Management capability at 0x60 */
	vconfig[0x60] = PCI_CAP_ID_PM;          /* PM capability */
	vconfig[0x61] = 0x68;                   /* Next: MSI at 0x68 */
	STORE_LE16(&vconfig[0x62], 0x0003);     /* PM capabilities */
	STORE_LE16(&vconfig[0x64], 0x0000);     /* PM control/status */

	/* MSI capability at 0x68 */
	vconfig[0x68] = PCI_CAP_ID_MSI;         /* MSI capability */
	vconfig[0x69] = 0x78;                   /* Next: PCIe at 0x78 */
	STORE_LE16(&vconfig[0x6a], 0x0080);     /* MSI control */

	/* PCI Express capability at 0x78 */
	vconfig[0x78] = PCI_CAP_ID_EXP;         /* PCIe capability */
	vconfig[0x79] = 0x00;                   /* End of list */
	STORE_LE16(&vconfig[0x7a], 0x0002);     /* PCIe capabilities */
	STORE_LE32(&vconfig[0x7c], 0x00000010); /* Device capabilities */
}

static void handle_pci_cfg_write(struct mdev_state *mdev_state, u16 offset,
				 const char *buf, u32 count)
{
	u32 cfg_addr;

	switch (offset) {
	case PCI_BASE_ADDRESS_0:
		cfg_addr = *(u32 *)buf;

		if (cfg_addr == 0xffffffff) {
			cfg_addr = (cfg_addr & mdev_state->bar_mask);
		} else {
			cfg_addr &= PCI_BASE_ADDRESS_MEM_MASK;
			if (cfg_addr)
				dev_dbg(mdev_state->vdev.dev, "BAR0 @ 0x%x\n",
					cfg_addr);
		}

		cfg_addr |= (mdev_state->vconfig[offset] &
			     ~PCI_BASE_ADDRESS_MEM_MASK);
		STORE_LE32(&mdev_state->vconfig[offset], cfg_addr);
		break;

	case PCI_COMMAND:
		/* Allow command register writes */
		STORE_LE16(&mdev_state->vconfig[offset], *(u16 *)buf);
		break;
	}
}

static ssize_t mdev_access(struct mdev_state *mdev_state, char *buf,
			   size_t count, loff_t pos, bool is_write)
{
	int ret = 0;

	mutex_lock(&mdev_state->ops_lock);

	if (pos < FAKE_VGPU_CONFIG_SPACE_SIZE) {
		/* PCI config space access */
		if (is_write)
			handle_pci_cfg_write(mdev_state, pos, buf, count);
		else
			memcpy(buf, mdev_state->vconfig + pos, count);
	} else if (pos >= FAKE_VGPU_MEMORY_BAR_OFFSET &&
		   pos + count <= FAKE_VGPU_MEMORY_BAR_OFFSET +
				  mdev_state->memsize) {
		/* Memory BAR access */
		pos -= FAKE_VGPU_MEMORY_BAR_OFFSET;
		if (is_write)
			memcpy(mdev_state->memblk + pos, buf, count);
		else
			memcpy(buf, mdev_state->memblk + pos, count);
	} else {
		dev_dbg(mdev_state->vdev.dev, "%s: %s @0x%llx (unhandled)\n",
			__func__, is_write ? "WR" : "RD", pos);
		ret = -EINVAL;
		goto out;
	}

	ret = count;

out:
	mutex_unlock(&mdev_state->ops_lock);
	return ret;
}

static int fake_vgpu_reset(struct mdev_state *mdev_state)
{
	/* Clear fake VRAM */
	if (mdev_state->memblk)
		memset(mdev_state->memblk, 0, mdev_state->memsize);
	return 0;
}

static int fake_vgpu_init_dev(struct vfio_device *vdev)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);
	struct mdev_device *mdev = to_mdev_device(vdev->dev);
	const struct fake_vgpu_type *type;
	int i, ret = -ENOMEM;

	/* Find which type this mdev belongs to */
	for (i = 0; i < ARRAY_SIZE(fake_vgpu_types); i++) {
		if (mdev->type == &fake_vgpu_types[i].type) {
			type = &fake_vgpu_types[i];
			mdev_state->type_index = i;
			break;
		}
	}
	if (i == ARRAY_SIZE(fake_vgpu_types))
		return -EINVAL;

	mdev_state->type = type;
	mdev_state->mdev = mdev;

	/* Allocate PCI config space */
	mdev_state->vconfig = kzalloc(FAKE_VGPU_CONFIG_SPACE_SIZE, GFP_KERNEL);
	if (!mdev_state->vconfig)
		return ret;

	/* Allocate fake VRAM (use a reasonable fixed size) */
	mdev_state->memsize = FAKE_VGPU_MEMORY_SIZE;
	mdev_state->memblk = vzalloc(mdev_state->memsize);
	if (!mdev_state->memblk)
		goto err_vconfig;

	mutex_init(&mdev_state->ops_lock);
	fake_vgpu_create_config_space(mdev_state);
	fake_vgpu_reset(mdev_state);

	dev_info(vdev->dev, "Created fake vGPU: %s (%s)\n",
		 type->type.sysfs_name, type->type.pretty_name);

	return 0;

err_vconfig:
	kfree(mdev_state->vconfig);
	return ret;
}

static int fake_vgpu_probe(struct mdev_device *mdev)
{
	struct mdev_state *mdev_state;
	const struct fake_vgpu_type *type;
	int i, ret;

	/* Find the type and check availability */
	for (i = 0; i < ARRAY_SIZE(fake_vgpu_types); i++) {
		if (mdev->type == &fake_vgpu_types[i].type) {
			type = &fake_vgpu_types[i];
			break;
		}
	}
	if (i == ARRAY_SIZE(fake_vgpu_types))
		return -EINVAL;

	if (atomic_dec_return(&avail_instances[i]) < 0) {
		atomic_inc(&avail_instances[i]);
		return -ENOSPC;
	}

	mdev_state = vfio_alloc_device(mdev_state, vdev, &mdev->dev,
				       &fake_vgpu_dev_ops);
	if (IS_ERR(mdev_state)) {
		atomic_inc(&avail_instances[i]);
		return PTR_ERR(mdev_state);
	}

	ret = vfio_register_emulated_iommu_dev(&mdev_state->vdev);
	if (ret) {
		vfio_put_device(&mdev_state->vdev);
		atomic_inc(&avail_instances[i]);
		return ret;
	}

	dev_set_drvdata(&mdev->dev, mdev_state);
	return 0;
}

static void fake_vgpu_release_dev(struct vfio_device *vdev)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);

	vfree(mdev_state->memblk);
	kfree(mdev_state->vconfig);
	mutex_destroy(&mdev_state->ops_lock);
}

static void fake_vgpu_remove(struct mdev_device *mdev)
{
	struct mdev_state *mdev_state = dev_get_drvdata(&mdev->dev);

	dev_info(&mdev->dev, "Removing fake vGPU: %s\n",
		 mdev_state->type->type.sysfs_name);

	vfio_unregister_group_dev(&mdev_state->vdev);
	vfio_put_device(&mdev_state->vdev);
	atomic_inc(&avail_instances[mdev_state->type_index]);
}

static ssize_t fake_vgpu_read(struct vfio_device *vdev, char __user *buf,
			      size_t count, loff_t *ppos)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);
	unsigned int done = 0;
	int ret;

	while (count) {
		size_t filled;

		if (count >= 4 && !(*ppos % 4)) {
			u32 val;

			ret = mdev_access(mdev_state, (char *)&val, sizeof(val),
					  *ppos, false);
			if (ret <= 0)
				goto read_err;

			if (copy_to_user(buf, &val, sizeof(val)))
				goto read_err;

			filled = 4;
		} else if (count >= 2 && !(*ppos % 2)) {
			u16 val;

			ret = mdev_access(mdev_state, (char *)&val, sizeof(val),
					  *ppos, false);
			if (ret <= 0)
				goto read_err;

			if (copy_to_user(buf, &val, sizeof(val)))
				goto read_err;

			filled = 2;
		} else {
			u8 val;

			ret = mdev_access(mdev_state, (char *)&val, sizeof(val),
					  *ppos, false);
			if (ret <= 0)
				goto read_err;

			if (copy_to_user(buf, &val, sizeof(val)))
				goto read_err;

			filled = 1;
		}

		count -= filled;
		done += filled;
		*ppos += filled;
		buf += filled;
	}

	return done;

read_err:
	return -EFAULT;
}

static ssize_t fake_vgpu_write(struct vfio_device *vdev, const char __user *buf,
			       size_t count, loff_t *ppos)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);
	unsigned int done = 0;
	int ret;

	while (count) {
		size_t filled;

		if (count >= 4 && !(*ppos % 4)) {
			u32 val;

			if (copy_from_user(&val, buf, sizeof(val)))
				goto write_err;

			ret = mdev_access(mdev_state, (char *)&val, sizeof(val),
					  *ppos, true);
			if (ret <= 0)
				goto write_err;

			filled = 4;
		} else if (count >= 2 && !(*ppos % 2)) {
			u16 val;

			if (copy_from_user(&val, buf, sizeof(val)))
				goto write_err;

			ret = mdev_access(mdev_state, (char *)&val, sizeof(val),
					  *ppos, true);
			if (ret <= 0)
				goto write_err;

			filled = 2;
		} else {
			u8 val;

			if (copy_from_user(&val, buf, sizeof(val)))
				goto write_err;

			ret = mdev_access(mdev_state, (char *)&val, sizeof(val),
					  *ppos, true);
			if (ret <= 0)
				goto write_err;

			filled = 1;
		}

		count -= filled;
		done += filled;
		*ppos += filled;
		buf += filled;
	}

	return done;

write_err:
	return -EFAULT;
}

static int fake_vgpu_mmap(struct vfio_device *vdev, struct vm_area_struct *vma)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);

	if (vma->vm_pgoff != FAKE_VGPU_MEMORY_BAR_OFFSET >> PAGE_SHIFT)
		return -EINVAL;
	if (vma->vm_end < vma->vm_start)
		return -EINVAL;
	if (vma->vm_end - vma->vm_start > mdev_state->memsize)
		return -EINVAL;
	if ((vma->vm_flags & VM_SHARED) == 0)
		return -EINVAL;

	return remap_vmalloc_range(vma, mdev_state->memblk, 0);
}

static int fake_vgpu_get_region_info(struct vfio_device *vdev,
				     struct vfio_region_info *region_info,
				     struct vfio_info_cap *caps)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);

	if (region_info->index >= VFIO_PCI_NUM_REGIONS)
		return -EINVAL;

	switch (region_info->index) {
	case VFIO_PCI_CONFIG_REGION_INDEX:
		region_info->offset = 0;
		region_info->size = FAKE_VGPU_CONFIG_SPACE_SIZE;
		region_info->flags = VFIO_REGION_INFO_FLAG_READ |
				     VFIO_REGION_INFO_FLAG_WRITE;
		break;

	case VFIO_PCI_BAR0_REGION_INDEX:
		region_info->offset = FAKE_VGPU_MEMORY_BAR_OFFSET;
		region_info->size = mdev_state->memsize;
		region_info->flags = VFIO_REGION_INFO_FLAG_READ |
				     VFIO_REGION_INFO_FLAG_WRITE |
				     VFIO_REGION_INFO_FLAG_MMAP;
		break;

	default:
		region_info->size = 0;
		region_info->offset = 0;
		region_info->flags = 0;
		break;
	}

	return 0;
}

static int fake_vgpu_get_irq_info(struct vfio_irq_info *irq_info)
{
	switch (irq_info->index) {
	case VFIO_PCI_INTX_IRQ_INDEX:
	case VFIO_PCI_MSI_IRQ_INDEX:
		irq_info->flags = VFIO_IRQ_INFO_EVENTFD;
		irq_info->count = 1;
		break;
	default:
		irq_info->flags = 0;
		irq_info->count = 0;
		break;
	}
	return 0;
}

static int fake_vgpu_get_device_info(struct vfio_device_info *dev_info)
{
	dev_info->flags = VFIO_DEVICE_FLAGS_PCI;
	dev_info->num_regions = VFIO_PCI_NUM_REGIONS;
	dev_info->num_irqs = VFIO_PCI_NUM_IRQS;
	return 0;
}

static long fake_vgpu_ioctl(struct vfio_device *vdev, unsigned int cmd,
			    unsigned long arg)
{
	struct mdev_state *mdev_state =
		container_of(vdev, struct mdev_state, vdev);
	unsigned long minsz;
	int ret = 0;

	switch (cmd) {
	case VFIO_DEVICE_GET_INFO:
	{
		struct vfio_device_info info;

		minsz = offsetofend(struct vfio_device_info, num_irqs);

		if (copy_from_user(&info, (void __user *)arg, minsz))
			return -EFAULT;

		if (info.argsz < minsz)
			return -EINVAL;

		ret = fake_vgpu_get_device_info(&info);
		if (ret)
			return ret;

		memcpy(&mdev_state->dev_info, &info, sizeof(info));

		if (copy_to_user((void __user *)arg, &info, minsz))
			return -EFAULT;

		return 0;
	}

	case VFIO_DEVICE_GET_REGION_INFO:
	{
		struct vfio_region_info info;
		struct vfio_info_cap caps = { .buf = NULL, .size = 0 };

		minsz = offsetofend(struct vfio_region_info, offset);

		if (copy_from_user(&info, (void __user *)arg, minsz))
			return -EFAULT;

		if (info.argsz < minsz)
			return -EINVAL;

		ret = fake_vgpu_get_region_info(vdev, &info, &caps);
		if (ret)
			return ret;

		if (copy_to_user((void __user *)arg, &info, minsz))
			return -EFAULT;

		return 0;
	}

	case VFIO_DEVICE_GET_IRQ_INFO:
	{
		struct vfio_irq_info info;

		minsz = offsetofend(struct vfio_irq_info, count);

		if (copy_from_user(&info, (void __user *)arg, minsz))
			return -EFAULT;

		if (info.argsz < minsz ||
		    info.index >= mdev_state->dev_info.num_irqs)
			return -EINVAL;

		ret = fake_vgpu_get_irq_info(&info);
		if (ret)
			return ret;

		if (copy_to_user((void __user *)arg, &info, minsz))
			return -EFAULT;

		return 0;
	}

	case VFIO_DEVICE_SET_IRQS:
		/* Accept but ignore IRQ setup */
		return 0;

	case VFIO_DEVICE_RESET:
		return fake_vgpu_reset(mdev_state);

	case VFIO_DEVICE_QUERY_GFX_PLANE:
	{
		struct vfio_device_gfx_plane_info plane;

		minsz = offsetofend(struct vfio_device_gfx_plane_info,
				    region_index);

		if (copy_from_user(&plane, (void __user *)arg, minsz))
			return -EFAULT;

		if (plane.argsz < minsz)
			return -EINVAL;

		/*
		 * If this is a probe request, report what we support.
		 * We support region-based display (framebuffer in BAR0).
		 */
		if (plane.flags & VFIO_GFX_PLANE_TYPE_PROBE) {
			plane.flags = VFIO_GFX_PLANE_TYPE_REGION;
			goto plane_reply;
		}

		/*
		 * QEMU requests the primary plane. We provide a simple
		 * framebuffer in BAR0 that QEMU can use for ramfb display.
		 */
		if (plane.flags != VFIO_GFX_PLANE_TYPE_REGION)
			return -EINVAL;

		/* Fill in framebuffer details */
		plane.drm_format = DRM_FORMAT_XRGB8888;
		plane.drm_format_mod = 0;
		plane.width = FAKE_VGPU_DISPLAY_WIDTH;
		plane.height = FAKE_VGPU_DISPLAY_HEIGHT;
		plane.stride = FAKE_VGPU_DISPLAY_STRIDE;
		plane.size = FAKE_VGPU_DISPLAY_SIZE;
		plane.x_pos = 0;
		plane.y_pos = 0;
		plane.x_hot = 0;
		plane.y_hot = 0;
		plane.region_index = VFIO_PCI_BAR0_REGION_INDEX;

plane_reply:
		if (copy_to_user((void __user *)arg, &plane, minsz))
			return -EFAULT;

		return 0;
	}

	case VFIO_DEVICE_GET_GFX_DMABUF:
		/* We don't support dma-buf export, only region-based display */
		return -EINVAL;
	}

	return -ENOTTY;
}

static unsigned int fake_vgpu_get_available(struct mdev_type *mtype)
{
	int i;

	for (i = 0; i < ARRAY_SIZE(fake_vgpu_types); i++) {
		if (mtype == &fake_vgpu_types[i].type)
			return atomic_read(&avail_instances[i]);
	}
	return 0;
}

static ssize_t fake_vgpu_show_description(struct mdev_type *mtype, char *buf)
{
	const struct fake_vgpu_type *type =
		container_of(mtype, struct fake_vgpu_type, type);

	return sprintf(buf, "NVIDIA GRID vGPU (%s), %dMB framebuffer\n",
		       type->profile, type->fb_size);
}

/* Sysfs attributes for mdev devices */
static ssize_t gpu_type_show(struct device *dev, struct device_attribute *attr,
			     char *buf)
{
	struct mdev_state *mdev_state = dev_get_drvdata(dev);

	return sprintf(buf, "%s\n", mdev_state->type->type.pretty_name);
}
static DEVICE_ATTR_RO(gpu_type);

static ssize_t fb_size_show(struct device *dev, struct device_attribute *attr,
			    char *buf)
{
	struct mdev_state *mdev_state = dev_get_drvdata(dev);

	return sprintf(buf, "%d MB\n", mdev_state->type->fb_size);
}
static DEVICE_ATTR_RO(fb_size);

static struct attribute *mdev_dev_attrs[] = {
	&dev_attr_gpu_type.attr,
	&dev_attr_fb_size.attr,
	NULL,
};

static const struct attribute_group mdev_dev_group = {
	.name = "nvidia",
	.attrs = mdev_dev_attrs,
};

static const struct attribute_group *mdev_dev_groups[] = {
	&mdev_dev_group,
	NULL,
};

static const struct vfio_device_ops fake_vgpu_dev_ops = {
	.name = "fake-nvidia-vgpu",
	.init = fake_vgpu_init_dev,
	.release = fake_vgpu_release_dev,
	.read = fake_vgpu_read,
	.write = fake_vgpu_write,
	.ioctl = fake_vgpu_ioctl,
	.mmap = fake_vgpu_mmap,
	.bind_iommufd = vfio_iommufd_emulated_bind,
	.unbind_iommufd = vfio_iommufd_emulated_unbind,
	.attach_ioas = vfio_iommufd_emulated_attach_ioas,
	.detach_ioas = vfio_iommufd_emulated_detach_ioas,
};

static struct mdev_driver fake_vgpu_driver = {
	.device_api = VFIO_DEVICE_API_PCI_STRING,
	.driver = {
		.name = "nvidia",
		.owner = THIS_MODULE,
		.mod_name = KBUILD_MODNAME,
		.dev_groups = mdev_dev_groups,
	},
	.probe = fake_vgpu_probe,
	.remove = fake_vgpu_remove,
	.get_available = fake_vgpu_get_available,
	.show_description = fake_vgpu_show_description,
};

static const struct file_operations fake_vgpu_fops = {
	.owner = THIS_MODULE,
};

static void fake_vgpu_device_release(struct device *dev)
{
	dev_dbg(dev, "fake_vgpu: device released\n");
}

static int __init fake_vgpu_init(void)
{
	int ret, i;

	pr_info("fake_nvidia_vgpu: initializing\n");

	/* Initialize available instance counters */
	for (i = 0; i < ARRAY_SIZE(fake_vgpu_types); i++)
		atomic_set(&avail_instances[i], fake_vgpu_types[i].max_instances);

	memset(&fake_vgpu_dev, 0, sizeof(fake_vgpu_dev));

	ret = alloc_chrdev_region(&fake_vgpu_dev.devt, 0, MINORMASK + 1,
				  FAKE_VGPU_NAME);
	if (ret < 0) {
		pr_err("fake_nvidia_vgpu: failed to allocate chrdev region\n");
		return ret;
	}

	cdev_init(&fake_vgpu_dev.cdev, &fake_vgpu_fops);
	ret = cdev_add(&fake_vgpu_dev.cdev, fake_vgpu_dev.devt, MINORMASK + 1);
	if (ret < 0)
		goto err_chrdev;

	pr_info("fake_nvidia_vgpu: registered with major %d\n",
		MAJOR(fake_vgpu_dev.devt));

	ret = mdev_register_driver(&fake_vgpu_driver);
	if (ret)
		goto err_cdev;

	fake_vgpu_dev.vgpu_class = COMPAT_CLASS_CREATE(FAKE_VGPU_CLASS_NAME);
	if (IS_ERR(fake_vgpu_dev.vgpu_class)) {
		ret = PTR_ERR(fake_vgpu_dev.vgpu_class);
		goto err_driver;
	}

	fake_vgpu_dev.dev.class = fake_vgpu_dev.vgpu_class;
	fake_vgpu_dev.dev.release = fake_vgpu_device_release;
	dev_set_name(&fake_vgpu_dev.dev, "%s", FAKE_VGPU_NAME);

	ret = device_register(&fake_vgpu_dev.dev);
	if (ret)
		goto err_class;

	ret = mdev_register_parent(&fake_vgpu_dev.parent, &fake_vgpu_dev.dev,
				   &fake_vgpu_driver, fake_vgpu_mdev_types,
				   ARRAY_SIZE(fake_vgpu_mdev_types));
	if (ret)
		goto err_device;

	pr_info("fake_nvidia_vgpu: ready, providing %d GRID T4-1B and %d GRID T4-2B instances\n",
		MAX_T4_1B_INSTANCES, MAX_T4_2B_INSTANCES);

	return 0;

err_device:
	device_del(&fake_vgpu_dev.dev);
	put_device(&fake_vgpu_dev.dev);
err_class:
	class_destroy(fake_vgpu_dev.vgpu_class);
err_driver:
	mdev_unregister_driver(&fake_vgpu_driver);
err_cdev:
	cdev_del(&fake_vgpu_dev.cdev);
err_chrdev:
	unregister_chrdev_region(fake_vgpu_dev.devt, MINORMASK + 1);
	return ret;
}

static void __exit fake_vgpu_exit(void)
{
	fake_vgpu_dev.dev.bus = NULL;
	mdev_unregister_parent(&fake_vgpu_dev.parent);
	device_unregister(&fake_vgpu_dev.dev);
	mdev_unregister_driver(&fake_vgpu_driver);
	cdev_del(&fake_vgpu_dev.cdev);
	unregister_chrdev_region(fake_vgpu_dev.devt, MINORMASK + 1);
	class_destroy(fake_vgpu_dev.vgpu_class);
	fake_vgpu_dev.vgpu_class = NULL;

	pr_info("fake_nvidia_vgpu: unloaded\n");
}

module_init(fake_vgpu_init)
module_exit(fake_vgpu_exit)
