// SPDX-License-Identifier: GPL-2.0
/* Synthetic NVIDIA SR-IOV vGPU VFIO driver for kubevirtci tests.
 * It only binds VFs of the synthetic PF in PCI domain faca.
 */
#include <linux/module.h>
#include <linux/pci.h>
#include <linux/vfio.h>
#include <linux/vfio_pci_core.h>

#define FAKE_DOMAIN 0xfaca
#define FAKE_VENDOR 0x10de
#define FAKE_DEVICE 0x26b9
#define FAKE_PROFILE 1155

struct fake_vgpu {
	struct vfio_pci_core_device core;
	u32 profile;
};

static int fake_open(struct vfio_device *vdev)
{
	struct fake_vgpu *gpu = container_of(vdev, struct fake_vgpu, core.vdev);
	int ret = vfio_pci_core_enable(&gpu->core);
	if (ret)
		return ret;
	vfio_pci_core_finish_enable(&gpu->core);
	return 0;
}

static const struct vfio_device_ops fake_ops = {
	.name = "fake-nvidia-vgpu-sriov",
	.init = vfio_pci_core_init_dev,
	.release = vfio_pci_core_release_dev,
	.open_device = fake_open,
	.close_device = vfio_pci_core_close_device,
	.ioctl = vfio_pci_core_ioctl,
	.get_region_info_caps = vfio_pci_ioctl_get_region_info,
	.read = vfio_pci_core_read,
	.write = vfio_pci_core_write,
	.mmap = vfio_pci_core_mmap,
	.request = vfio_pci_core_request,
	.bind_iommufd = vfio_iommufd_physical_bind,
	.unbind_iommufd = vfio_iommufd_physical_unbind,
	.attach_ioas = vfio_iommufd_physical_attach_ioas,
	.detach_ioas = vfio_iommufd_physical_detach_ioas,
};

static ssize_t current_vgpu_type_show(struct device *dev,
				      struct device_attribute *attr, char *buf)
{
	struct fake_vgpu *gpu = pci_get_drvdata(to_pci_dev(dev));
	return sysfs_emit(buf, "%u\n", READ_ONCE(gpu->profile));
}

static ssize_t current_vgpu_type_store(struct device *dev,
				       struct device_attribute *attr,
				       const char *buf, size_t count)
{
	struct fake_vgpu *gpu = pci_get_drvdata(to_pci_dev(dev));
	u32 profile;
	int ret = kstrtou32(buf, 0, &profile);
	if (ret)
		return ret;
	if (profile != 0 && profile != FAKE_PROFILE)
		return -EINVAL;
	WRITE_ONCE(gpu->profile, profile);
	return count;
}
static DEVICE_ATTR_RW(current_vgpu_type);

static struct attribute *fake_attrs[] = {
	&dev_attr_current_vgpu_type.attr,
	NULL,
};
static const struct attribute_group fake_group = {
	.name = "nvidia",
	.attrs = fake_attrs,
};

static int fake_probe(struct pci_dev *pdev, const struct pci_device_id *id)
{
	struct fake_vgpu *gpu;
	int ret;

	if (pci_domain_nr(pdev->bus) != FAKE_DOMAIN || !pdev->is_virtfn)
		return -ENODEV;
	gpu = vfio_alloc_device(fake_vgpu, core.vdev, &pdev->dev, &fake_ops);
	if (IS_ERR(gpu))
		return PTR_ERR(gpu);
	gpu->core.pdev = pdev;
	pci_set_drvdata(pdev, gpu);
	ret = sysfs_create_group(&pdev->dev.kobj, &fake_group);
	if (ret)
		goto err_put;
	ret = vfio_pci_core_register_device(&gpu->core);
	if (ret)
		goto err_group;
	return 0;

err_group:
	sysfs_remove_group(&pdev->dev.kobj, &fake_group);
err_put:
	pci_set_drvdata(pdev, NULL);
	vfio_put_device(&gpu->core.vdev);
	return ret;
}

static void fake_remove(struct pci_dev *pdev)
{
	struct fake_vgpu *gpu = pci_get_drvdata(pdev);
	vfio_pci_core_unregister_device(&gpu->core);
	sysfs_remove_group(&pdev->dev.kobj, &fake_group);
	pci_set_drvdata(pdev, NULL);
	vfio_put_device(&gpu->core.vdev);
}

static const struct pci_device_id fake_ids[] = {
	{ PCI_DEVICE(FAKE_VENDOR, FAKE_DEVICE) },
	{ }
};
MODULE_DEVICE_TABLE(pci, fake_ids);

static struct pci_driver fake_driver = {
	.name = "nvidia",
	.id_table = fake_ids,
	.probe = fake_probe,
	.remove = fake_remove,
	.driver_managed_dma = true,
};
module_pci_driver(fake_driver);

MODULE_DESCRIPTION("Synthetic NVIDIA SR-IOV VFIO vGPU for KubeVirt testing");
MODULE_LICENSE("GPL");
