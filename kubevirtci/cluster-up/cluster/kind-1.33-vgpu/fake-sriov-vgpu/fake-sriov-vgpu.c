// SPDX-License-Identifier: GPL-2.0
/*
 * Fake SR-IOV vGPU PCI device driver for KubeVirt testing
 *
 * This module creates fake PCI devices that appear in /sys/bus/pci/devices/
 * to simulate NVIDIA SR-IOV Virtual Functions with vGPU profiles assigned.
 *
 * It creates a virtual PCI bus and registers fake devices on it, making them
 * visible to the standard PCI device discovery mechanisms.
 *
 * Copyright the KubeVirt Authors.
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/slab.h>
#include <linux/pci.h>
#include <linux/list.h>
#include <linux/mutex.h>
#include <linux/version.h>
#include <linux/device.h>
#include <linux/ioport.h>

#include "compat.h"

/* Fake resources for our virtual PCI bus */
static struct resource fake_pci_mem = {
	.name	= "fake-sriov-vgpu PCI mem",
	.start	= 0x80000000,
	.end	= 0x8fffffff,
	.flags	= IORESOURCE_MEM,
};

static struct resource fake_pci_io = {
	.name	= "fake-sriov-vgpu PCI I/O",
	.start	= 0x1000,
	.end	= 0x1fff,
	.flags	= IORESOURCE_IO,
};

#define VERSION_STRING  "1.0"
#define DRIVER_AUTHOR   "KubeVirt Fake SR-IOV vGPU Driver"

#define FAKE_SRIOV_NAME         "fake-sriov-vgpu"

/* NVIDIA PCI IDs - Tesla T4 */
#define NVIDIA_VENDOR_ID        0x10de
#define NVIDIA_T4_DEVICE_ID     0x1eb8
#define NVIDIA_T4_SUBSYS_ID     0x12a2

/* Our fake PCI domain/segment */
#define FAKE_PCI_DOMAIN         0x0001  /* Use domain 1 to avoid conflicts */
#define FAKE_PCI_BUS            0x00

/* Maximum number of fake VFs */
#define MAX_FAKE_VFS            32

/* PCI config space size */
#define PCI_CONFIG_SPACE_SIZE   256

MODULE_DESCRIPTION("Fake SR-IOV vGPU PCI driver for KubeVirt testing");
MODULE_LICENSE("GPL v2");
MODULE_VERSION(VERSION_STRING);
MODULE_AUTHOR(DRIVER_AUTHOR);

/* Per-VF device state */
struct fake_vf_state {
	struct list_head list;
	struct pci_dev *pdev;
	u8 config_space[PCI_CONFIG_SPACE_SIZE];
	u32 vgpu_type;
	int slot;
	int func;
	struct kobject *nvidia_kobj;
};

/* Global state */
static struct {
	struct pci_bus *bus;
	struct pci_host_bridge *bridge;
	struct list_head vf_list;
	struct mutex lock;
	int vf_count;
	struct class *ctrl_class;
	struct device *ctrl_dev;
} fake_sriov;

/* Forward declarations */
static void destroy_fake_vf(struct fake_vf_state *vf);

/*
 * PCI config space operations for our fake bus
 */
static struct fake_vf_state *find_vf_by_devfn(unsigned int devfn)
{
	struct fake_vf_state *vf;
	int slot = PCI_SLOT(devfn);
	int func = PCI_FUNC(devfn);

	list_for_each_entry(vf, &fake_sriov.vf_list, list) {
		if (vf->slot == slot && vf->func == func)
			return vf;
	}
	return NULL;
}

static int fake_pci_read(struct pci_bus *bus, unsigned int devfn,
			 int where, int size, u32 *val)
{
	struct fake_vf_state *vf;

	if (where >= PCI_CONFIG_SPACE_SIZE)
		return PCIBIOS_BAD_REGISTER_NUMBER;

	vf = find_vf_by_devfn(devfn);
	if (!vf) {
		*val = ~0;
		return PCIBIOS_DEVICE_NOT_FOUND;
	}

	switch (size) {
	case 1:
		*val = vf->config_space[where];
		break;
	case 2:
		*val = *(u16 *)&vf->config_space[where];
		break;
	case 4:
		*val = *(u32 *)&vf->config_space[where];
		break;
	default:
		return PCIBIOS_BAD_REGISTER_NUMBER;
	}

	return PCIBIOS_SUCCESSFUL;
}

static int fake_pci_write(struct pci_bus *bus, unsigned int devfn,
			  int where, int size, u32 val)
{
	struct fake_vf_state *vf;

	if (where >= PCI_CONFIG_SPACE_SIZE)
		return PCIBIOS_BAD_REGISTER_NUMBER;

	vf = find_vf_by_devfn(devfn);
	if (!vf)
		return PCIBIOS_DEVICE_NOT_FOUND;

	switch (size) {
	case 1:
		vf->config_space[where] = val & 0xff;
		break;
	case 2:
		*(u16 *)&vf->config_space[where] = val & 0xffff;
		break;
	case 4:
		*(u32 *)&vf->config_space[where] = val;
		break;
	default:
		return PCIBIOS_BAD_REGISTER_NUMBER;
	}

	return PCIBIOS_SUCCESSFUL;
}

static struct pci_ops fake_pci_ops = {
	.read = fake_pci_read,
	.write = fake_pci_write,
};

/*
 * Initialize PCI config space to look like NVIDIA Tesla T4 VF
 */
static void init_config_space(struct fake_vf_state *vf)
{
	u8 *cfg = vf->config_space;

	memset(cfg, 0, PCI_CONFIG_SPACE_SIZE);

	/* Vendor and Device ID */
	*(u16 *)&cfg[PCI_VENDOR_ID] = NVIDIA_VENDOR_ID;
	*(u16 *)&cfg[PCI_DEVICE_ID] = NVIDIA_T4_DEVICE_ID;

	/* Command: Memory space enable */
	*(u16 *)&cfg[PCI_COMMAND] = PCI_COMMAND_MEMORY;

	/* Status: Capabilities list */
	*(u16 *)&cfg[PCI_STATUS] = PCI_STATUS_CAP_LIST;

	/* Revision */
	cfg[PCI_REVISION_ID] = 0xa1;

	/* Class: Display controller / VGA compatible */
	cfg[PCI_CLASS_PROG] = 0x00;
	*(u16 *)&cfg[PCI_CLASS_DEVICE] = 0x0300;  /* VGA compatible */

	/* Header type: Normal */
	cfg[PCI_HEADER_TYPE] = PCI_HEADER_TYPE_NORMAL;

	/* Subsystem IDs */
	*(u16 *)&cfg[PCI_SUBSYSTEM_VENDOR_ID] = NVIDIA_VENDOR_ID;
	*(u16 *)&cfg[PCI_SUBSYSTEM_ID] = NVIDIA_T4_SUBSYS_ID;

	/* BAR0: Memory, 64-bit, prefetchable (minimal - just for structure) */
	*(u32 *)&cfg[PCI_BASE_ADDRESS_0] = PCI_BASE_ADDRESS_MEM_TYPE_64 |
					   PCI_BASE_ADDRESS_MEM_PREFETCH;

	/* Capabilities pointer */
	cfg[PCI_CAPABILITY_LIST] = 0x60;

	/* Interrupt pin */
	cfg[PCI_INTERRUPT_PIN] = 0x01;

	/* Power Management capability at 0x60 */
	cfg[0x60] = PCI_CAP_ID_PM;
	cfg[0x61] = 0x00;  /* End of caps */
	*(u16 *)&cfg[0x62] = 0x0003;  /* PM capabilities */
}

/*
 * nvidia/current_vgpu_type sysfs attribute
 */
static ssize_t current_vgpu_type_show(struct kobject *kobj,
				      struct kobj_attribute *attr, char *buf)
{
	struct fake_vf_state *vf;

	mutex_lock(&fake_sriov.lock);
	list_for_each_entry(vf, &fake_sriov.vf_list, list) {
		if (vf->nvidia_kobj == kobj) {
			mutex_unlock(&fake_sriov.lock);
			return sprintf(buf, "%u\n", vf->vgpu_type);
		}
	}
	mutex_unlock(&fake_sriov.lock);
	return -ENODEV;
}

static ssize_t current_vgpu_type_store(struct kobject *kobj,
				       struct kobj_attribute *attr,
				       const char *buf, size_t count)
{
	struct fake_vf_state *vf;
	u32 type;
	int ret;

	ret = kstrtou32(buf, 0, &type);
	if (ret)
		return ret;

	mutex_lock(&fake_sriov.lock);
	list_for_each_entry(vf, &fake_sriov.vf_list, list) {
		if (vf->nvidia_kobj == kobj) {
			vf->vgpu_type = type;
			mutex_unlock(&fake_sriov.lock);
			return count;
		}
	}
	mutex_unlock(&fake_sriov.lock);
	return -ENODEV;
}

static struct kobj_attribute vgpu_type_attr =
	__ATTR(current_vgpu_type, 0644, current_vgpu_type_show, current_vgpu_type_store);

static struct attribute *nvidia_attrs[] = {
	&vgpu_type_attr.attr,
	NULL,
};

static const struct attribute_group nvidia_attr_group = {
	.attrs = nvidia_attrs,
};

/*
 * Create a fake VF PCI device
 */
static struct fake_vf_state *create_fake_vf(int slot, int func, u32 vgpu_type)
{
	struct fake_vf_state *vf;
	struct pci_dev *pdev;
	unsigned int devfn;
	int ret;

	if (fake_sriov.vf_count >= MAX_FAKE_VFS) {
		pr_err("fake_sriov_vgpu: maximum VF count reached\n");
		return ERR_PTR(-ENOSPC);
	}

	/* Check slot/func not already used */
	devfn = PCI_DEVFN(slot, func);
	if (find_vf_by_devfn(devfn)) {
		pr_err("fake_sriov_vgpu: slot %d func %d already exists\n", slot, func);
		return ERR_PTR(-EEXIST);
	}

	vf = kzalloc(sizeof(*vf), GFP_KERNEL);
	if (!vf)
		return ERR_PTR(-ENOMEM);

	vf->slot = slot;
	vf->func = func;
	vf->vgpu_type = vgpu_type;

	/* Initialize config space */
	init_config_space(vf);

	/* Add to list first so config space reads work during scan */
	list_add_tail(&vf->list, &fake_sriov.vf_list);
	fake_sriov.vf_count++;

	/* Scan the device - this creates the pci_dev and adds it to sysfs */
	pdev = pci_scan_single_device(fake_sriov.bus, devfn);
	if (!pdev) {
		pr_err("fake_sriov_vgpu: failed to scan device %02x:%02x.%d\n",
		       FAKE_PCI_BUS, slot, func);
		ret = -ENODEV;
		goto err_list;
	}

	vf->pdev = pdev;

	/* Add the device to the bus */
	pci_bus_add_device(pdev);

	/* Create nvidia/ subdirectory with current_vgpu_type */
	vf->nvidia_kobj = kobject_create_and_add("nvidia", &pdev->dev.kobj);
	if (!vf->nvidia_kobj) {
		pr_err("fake_sriov_vgpu: failed to create nvidia kobj\n");
		ret = -ENOMEM;
		goto err_pdev;
	}

	ret = sysfs_create_group(vf->nvidia_kobj, &nvidia_attr_group);
	if (ret) {
		pr_err("fake_sriov_vgpu: failed to create nvidia attrs: %d\n", ret);
		goto err_nvidia_kobj;
	}

	pr_info("fake_sriov_vgpu: created VF %04x:%02x:%02x.%d (vgpu_type=%u)\n",
		FAKE_PCI_DOMAIN, FAKE_PCI_BUS, slot, func, vgpu_type);

	return vf;

err_nvidia_kobj:
	kobject_put(vf->nvidia_kobj);
err_pdev:
	pci_stop_and_remove_bus_device(pdev);
err_list:
	list_del(&vf->list);
	fake_sriov.vf_count--;
	kfree(vf);
	return ERR_PTR(ret);
}

static void destroy_fake_vf(struct fake_vf_state *vf)
{
	pr_info("fake_sriov_vgpu: destroying VF %04x:%02x:%02x.%d\n",
		FAKE_PCI_DOMAIN, FAKE_PCI_BUS, vf->slot, vf->func);

	sysfs_remove_group(vf->nvidia_kobj, &nvidia_attr_group);
	kobject_put(vf->nvidia_kobj);

	if (vf->pdev)
		pci_stop_and_remove_bus_device(vf->pdev);

	list_del(&vf->list);
	fake_sriov.vf_count--;
	kfree(vf);
}

/*
 * Control interface sysfs attributes
 */

/* Create: echo "slot func vgpu_type" > create */
static ssize_t create_store(struct device *dev,
			    struct device_attribute *attr,
			    const char *buf, size_t count)
{
	int slot, func;
	u32 vgpu_type = 256;
	struct fake_vf_state *vf;
	int ret;

	ret = sscanf(buf, "%d %d %u", &slot, &func, &vgpu_type);
	if (ret < 2) {
		pr_err("fake_sriov_vgpu: usage: echo 'slot func [vgpu_type]' > create\n");
		return -EINVAL;
	}

	if (slot < 0 || slot > 31 || func < 0 || func > 7) {
		pr_err("fake_sriov_vgpu: invalid slot/func\n");
		return -EINVAL;
	}

	mutex_lock(&fake_sriov.lock);
	vf = create_fake_vf(slot, func, vgpu_type);
	mutex_unlock(&fake_sriov.lock);

	if (IS_ERR(vf))
		return PTR_ERR(vf);

	return count;
}
static DEVICE_ATTR_WO(create);

/* Remove: echo "slot func" > remove */
static ssize_t remove_store(struct device *dev,
			    struct device_attribute *attr,
			    const char *buf, size_t count)
{
	int slot, func;
	struct fake_vf_state *vf;
	unsigned int devfn;
	int ret;

	ret = sscanf(buf, "%d %d", &slot, &func);
	if (ret != 2) {
		pr_err("fake_sriov_vgpu: usage: echo 'slot func' > remove\n");
		return -EINVAL;
	}

	devfn = PCI_DEVFN(slot, func);

	mutex_lock(&fake_sriov.lock);
	vf = find_vf_by_devfn(devfn);
	if (!vf) {
		mutex_unlock(&fake_sriov.lock);
		pr_err("fake_sriov_vgpu: VF slot %d func %d not found\n", slot, func);
		return -ENOENT;
	}
	destroy_fake_vf(vf);
	mutex_unlock(&fake_sriov.lock);

	return count;
}
static DEVICE_ATTR_WO(remove);

/* List all VFs */
static ssize_t list_show(struct device *dev,
			 struct device_attribute *attr, char *buf)
{
	struct fake_vf_state *vf;
	ssize_t len = 0;

	mutex_lock(&fake_sriov.lock);
	list_for_each_entry(vf, &fake_sriov.vf_list, list) {
		len += scnprintf(buf + len, PAGE_SIZE - len,
				 "%04x:%02x:%02x.%d vgpu_type=%u\n",
				 FAKE_PCI_DOMAIN, FAKE_PCI_BUS,
				 vf->slot, vf->func, vf->vgpu_type);
		if (len >= PAGE_SIZE - 1)
			break;
	}
	mutex_unlock(&fake_sriov.lock);

	if (len == 0)
		len = scnprintf(buf, PAGE_SIZE, "(no VFs created)\n");

	return len;
}
static DEVICE_ATTR_RO(list);

/* Clear all VFs */
static ssize_t clear_store(struct device *dev,
			   struct device_attribute *attr,
			   const char *buf, size_t count)
{
	struct fake_vf_state *vf, *tmp;

	mutex_lock(&fake_sriov.lock);
	list_for_each_entry_safe(vf, tmp, &fake_sriov.vf_list, list) {
		destroy_fake_vf(vf);
	}
	mutex_unlock(&fake_sriov.lock);

	pr_info("fake_sriov_vgpu: all VFs cleared\n");
	return count;
}
static DEVICE_ATTR_WO(clear);

static struct attribute *ctrl_dev_attrs[] = {
	&dev_attr_create.attr,
	&dev_attr_remove.attr,
	&dev_attr_list.attr,
	&dev_attr_clear.attr,
	NULL,
};

static const struct attribute_group ctrl_dev_attr_group = {
	.attrs = ctrl_dev_attrs,
};

static const struct attribute_group *ctrl_dev_attr_groups[] = {
	&ctrl_dev_attr_group,
	NULL,
};

static void ctrl_device_release(struct device *dev)
{
	pr_debug("fake_sriov_vgpu: control device released\n");
}

/* Sysdata structure for our fake bus */
static struct {
	int domain;
} fake_sysdata = {
	.domain = FAKE_PCI_DOMAIN,
};

/*
 * Create the virtual PCI bus
 */
static int create_fake_pci_bus(void)
{
	struct pci_host_bridge *bridge;
	struct pci_bus *bus;
	LIST_HEAD(resources);

	/* Add resource windows */
	pci_add_resource(&resources, &fake_pci_io);
	pci_add_resource(&resources, &fake_pci_mem);

	/* Create root bus directly - simpler than host bridge for virtual devices */
	bus = pci_create_root_bus(NULL, FAKE_PCI_BUS, &fake_pci_ops,
				  &fake_sysdata, &resources);
	if (!bus) {
		pr_err("fake_sriov_vgpu: failed to create root bus\n");
		pci_free_resource_list(&resources);
		return -ENOMEM;
	}

	fake_sriov.bus = bus;
	
	/* Get the bridge from the bus */
	bridge = to_pci_host_bridge(bus->bridge);
	fake_sriov.bridge = bridge;

	pr_info("fake_sriov_vgpu: created PCI bus %04x:%02x\n",
		pci_domain_nr(fake_sriov.bus), FAKE_PCI_BUS);

	return 0;
}

static void destroy_fake_pci_bus(void)
{
	if (fake_sriov.bridge) {
		pci_remove_root_bus(fake_sriov.bus);
		fake_sriov.bus = NULL;
		fake_sriov.bridge = NULL;
	}
}

static int __init fake_sriov_init(void)
{
	int ret;

	pr_info("fake_sriov_vgpu: initializing\n");

	INIT_LIST_HEAD(&fake_sriov.vf_list);
	mutex_init(&fake_sriov.lock);
	fake_sriov.vf_count = 0;

	/* Create control class and device */
	fake_sriov.ctrl_class = COMPAT_CLASS_CREATE(FAKE_SRIOV_NAME);
	if (IS_ERR(fake_sriov.ctrl_class)) {
		ret = PTR_ERR(fake_sriov.ctrl_class);
		pr_err("fake_sriov_vgpu: failed to create class: %d\n", ret);
		goto err_class;
	}

	fake_sriov.ctrl_dev = kzalloc(sizeof(*fake_sriov.ctrl_dev), GFP_KERNEL);
	if (!fake_sriov.ctrl_dev) {
		ret = -ENOMEM;
		goto err_ctrl_alloc;
	}

	device_initialize(fake_sriov.ctrl_dev);
	fake_sriov.ctrl_dev->class = fake_sriov.ctrl_class;
	fake_sriov.ctrl_dev->release = ctrl_device_release;
	fake_sriov.ctrl_dev->groups = ctrl_dev_attr_groups;
	dev_set_name(fake_sriov.ctrl_dev, "control");

	ret = device_add(fake_sriov.ctrl_dev);
	if (ret) {
		pr_err("fake_sriov_vgpu: failed to add control device: %d\n", ret);
		goto err_ctrl_add;
	}

	/* Create the virtual PCI bus */
	ret = create_fake_pci_bus();
	if (ret)
		goto err_bus;

	pr_info("fake_sriov_vgpu: ready\n");
	pr_info("fake_sriov_vgpu: control at /sys/class/%s/control/\n", FAKE_SRIOV_NAME);
	pr_info("fake_sriov_vgpu: create VF: echo 'slot func [vgpu_type]' > create\n");
	pr_info("fake_sriov_vgpu: devices appear in /sys/bus/pci/devices/\n");

	return 0;

err_bus:
	device_del(fake_sriov.ctrl_dev);
err_ctrl_add:
	put_device(fake_sriov.ctrl_dev);
err_ctrl_alloc:
	class_destroy(fake_sriov.ctrl_class);
err_class:
	mutex_destroy(&fake_sriov.lock);
	return ret;
}

static void __exit fake_sriov_exit(void)
{
	struct fake_vf_state *vf, *tmp;

	pr_info("fake_sriov_vgpu: cleaning up\n");

	/* Remove all VFs */
	mutex_lock(&fake_sriov.lock);
	list_for_each_entry_safe(vf, tmp, &fake_sriov.vf_list, list) {
		destroy_fake_vf(vf);
	}
	mutex_unlock(&fake_sriov.lock);

	/* Remove PCI bus */
	destroy_fake_pci_bus();

	/* Remove control interface */
	device_del(fake_sriov.ctrl_dev);
	put_device(fake_sriov.ctrl_dev);
	class_destroy(fake_sriov.ctrl_class);
	mutex_destroy(&fake_sriov.lock);

	pr_info("fake_sriov_vgpu: unloaded\n");
}

module_init(fake_sriov_init)
module_exit(fake_sriov_exit)
