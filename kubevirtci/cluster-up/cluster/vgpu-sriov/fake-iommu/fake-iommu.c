// SPDX-License-Identifier: GPL-2.0
/* Software IOMMU for the synthetic PCI domain. It records VFIO mappings;
 * the fake devices cannot access memory. Load before fake-pci.
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/slab.h>
#include <linux/iommu.h>
#include <linux/pci.h>
#include <linux/device.h>
#include <linux/list.h>
#include <linux/notifier.h>
#include <linux/platform_device.h>
#include <linux/property.h>
#include <linux/sizes.h>
#include <linux/spinlock.h>

#define DRIVER_NAME "fake_iommu"
#define FAKE_PCI_DOMAIN 0xfaca

/* ---------- module state ---------- */

static struct iommu_device     fake_iommu_dev;
static struct fwnode_handle   *fake_iommu_fwnode;
static struct platform_device *fake_iommu_pdev;
static struct notifier_block   pci_bus_nb;

static const struct iommu_ops fake_iommu_ops;

/* ---------- iommu_domain_ops ---------- */

struct fake_iommu_domain {
    struct iommu_domain domain;
    struct list_head mappings;
    spinlock_t lock;
};

struct fake_iommu_mapping {
    struct list_head node;
    unsigned long iova;
    phys_addr_t paddr;
    size_t size;
};

static int fake_iommu_attach_dev(struct iommu_domain *domain,
                                 struct device *dev,
                                 struct iommu_domain *old)
{
    /* No real backing: attach always succeeds. */
    return 0;
}

static int fake_iommu_map_pages(struct iommu_domain *domain,
                unsigned long iova, phys_addr_t paddr,
                size_t pgsize, size_t pgcount, int prot,
                gfp_t gfp, size_t *mapped)
{
    struct fake_iommu_domain *d = container_of(domain, struct fake_iommu_domain, domain);
    struct fake_iommu_mapping *m, *new_mapping;
    unsigned long flags;
    size_t size;

    *mapped = 0;

    if (!pgsize || !pgcount || pgsize > SIZE_MAX / pgcount)
        return -EINVAL;

    size = pgsize * pgcount;
    if (iova + size < iova)
        return -EINVAL;

    new_mapping = kzalloc(sizeof(*new_mapping), gfp);
    if (!new_mapping)
        return -ENOMEM;

    new_mapping->iova = iova;
    new_mapping->paddr = paddr;
    new_mapping->size = size;

    spin_lock_irqsave(&d->lock, flags);
    list_for_each_entry(m, &d->mappings, node) {
        if (iova < m->iova + m->size && m->iova < iova + size) {
            spin_unlock_irqrestore(&d->lock, flags);
            kfree(new_mapping);
            return -EBUSY;
        }
    }

    list_add_tail(&new_mapping->node, &d->mappings);
    spin_unlock_irqrestore(&d->lock, flags);

    *mapped = size;
    return 0;
}

static size_t fake_iommu_unmap_pages(struct iommu_domain *domain,
                     unsigned long iova,
                     size_t pgsize, size_t pgcount,
                     struct iommu_iotlb_gather *gather)
{
    struct fake_iommu_domain *d = container_of(domain, struct fake_iommu_domain, domain);
    struct fake_iommu_mapping *m, *tmp, *split;
    LIST_HEAD(free_list);
    unsigned long flags;
    unsigned long start = iova;
    size_t size;
    unsigned long end;
    size_t unmapped = 0;

    if (!pgsize || !pgcount || pgsize > SIZE_MAX / pgcount)
        return 0;

    size = pgsize * pgcount;
    end = start + size;
    if (end < start)
        return 0;

    split = kzalloc(sizeof(*split), GFP_ATOMIC);

    spin_lock_irqsave(&d->lock, flags);
    list_for_each_entry_safe(m, tmp, &d->mappings, node) {
        unsigned long m_start = m->iova;
        unsigned long m_end = m->iova + m->size;
        unsigned long overlap_start;
        unsigned long overlap_end;
        size_t overlap;

        if (end <= m_start || m_end <= start)
            continue;

        overlap_start = max(start, m_start);
        overlap_end = min(end, m_end);
        overlap = overlap_end - overlap_start;

        if (overlap_start == m_start && overlap_end == m_end) {
            list_del(&m->node);
            list_add(&m->node, &free_list);
        } else if (overlap_start == m_start) {
            m->iova = overlap_end;
            m->paddr += overlap_end - m_start;
            m->size = m_end - overlap_end;
        } else if (overlap_end == m_end) {
            m->size = overlap_start - m_start;
        } else if (split) {
            split->iova = overlap_end;
            split->paddr = m->paddr + (overlap_end - m_start);
            split->size = m_end - overlap_end;
            m->size = overlap_start - m_start;
            list_add_tail(&split->node, &d->mappings);
            split = NULL;
        } else {
            break;
        }

        unmapped += overlap;
    }
    spin_unlock_irqrestore(&d->lock, flags);

    list_for_each_entry_safe(m, tmp, &free_list, node) {
        list_del(&m->node);
        kfree(m);
    }
    kfree(split);

    return unmapped;
}

static phys_addr_t fake_iommu_iova_to_phys(struct iommu_domain *domain,
                       dma_addr_t iova)
{
    struct fake_iommu_domain *d = container_of(domain, struct fake_iommu_domain, domain);
    struct fake_iommu_mapping *m;
    unsigned long flags;
    phys_addr_t paddr = 0;

    spin_lock_irqsave(&d->lock, flags);
    list_for_each_entry(m, &d->mappings, node) {
        if (iova >= m->iova && iova < m->iova + m->size) {
            paddr = m->paddr + (iova - m->iova);
            break;
        }
    }
    spin_unlock_irqrestore(&d->lock, flags);

    return paddr;
}

static void fake_iommu_domain_free(struct iommu_domain *domain)
{
    struct fake_iommu_domain *d = container_of(domain, struct fake_iommu_domain, domain);
    struct fake_iommu_mapping *m, *tmp;
    unsigned long flags;
    LIST_HEAD(free_list);

    spin_lock_irqsave(&d->lock, flags);
    list_splice_init(&d->mappings, &free_list);
    spin_unlock_irqrestore(&d->lock, flags);

    list_for_each_entry_safe(m, tmp, &free_list, node) {
        list_del(&m->node);
        kfree(m);
    }

    kfree(d);
}

static const struct iommu_domain_ops fake_iommu_domain_ops = {
    .attach_dev    = fake_iommu_attach_dev,
    .map_pages     = fake_iommu_map_pages,
    .unmap_pages   = fake_iommu_unmap_pages,
    .iova_to_phys  = fake_iommu_iova_to_phys,
    .free          = fake_iommu_domain_free,
};

/*
 * Static identity and blocked domains.
 */
static void fake_iommu_static_domain_free(struct iommu_domain *domain)
{
    /* no-op: identity / blocked domains are statically allocated */
}

static const struct iommu_domain_ops fake_iommu_static_domain_ops = {
    .attach_dev = fake_iommu_attach_dev,
    .free       = fake_iommu_static_domain_free,
};

static struct iommu_domain fake_iommu_identity_domain = {
    .type = IOMMU_DOMAIN_IDENTITY,
    .ops  = &fake_iommu_static_domain_ops,
};

static struct iommu_domain fake_iommu_blocked_domain = {
    .type = IOMMU_DOMAIN_BLOCKED,
    .ops  = &fake_iommu_static_domain_ops,
};

static struct iommu_domain *fake_iommu_domain_alloc_paging(struct device *dev)
{
    struct fake_iommu_domain *d;

    d = kzalloc(sizeof(*d), GFP_KERNEL);
    if (!d)
        return NULL;

    d->domain.type = IOMMU_DOMAIN_DMA;
    d->domain.geometry.aperture_start = 0;
    d->domain.geometry.aperture_end   = ~0ULL;
    d->domain.geometry.force_aperture = true;
    d->domain.pgsize_bitmap           = SZ_4K;
    INIT_LIST_HEAD(&d->mappings);
    spin_lock_init(&d->lock);

    return &d->domain;
}

/* ---------- iommu_ops ---------- */

static bool fake_iommu_device_on_target_domain(struct device *dev)
{
    struct pci_dev *pdev;

    if (!dev_is_pci(dev))
        return false;
    pdev = to_pci_dev(dev);
    return pci_domain_nr(pdev->bus) == FAKE_PCI_DOMAIN;
}

static struct iommu_device *fake_iommu_probe_device(struct device *dev)
{
    if (!fake_iommu_device_on_target_domain(dev))
        return ERR_PTR(-ENODEV);

    return &fake_iommu_dev;
}

static void fake_iommu_release_device(struct device *dev)
{
}

static struct iommu_group *fake_iommu_device_group(struct device *dev)
{
    return generic_device_group(dev);
}

/* VFIO requires cache coherent IOMMU mappings. */
static bool fake_iommu_capable(struct device *dev, enum iommu_cap cap)
{
    switch (cap) {
    case IOMMU_CAP_CACHE_COHERENCY:
        return true;
    default:
        return false;
    }
}

static const struct iommu_ops fake_iommu_ops = {
    .capable             = fake_iommu_capable,
    .domain_alloc_paging = fake_iommu_domain_alloc_paging,
    .probe_device        = fake_iommu_probe_device,
    .release_device      = fake_iommu_release_device,
    .device_group        = fake_iommu_device_group,
    .identity_domain     = &fake_iommu_identity_domain,
    .blocked_domain      = &fake_iommu_blocked_domain,
    .default_domain_ops  = &fake_iommu_domain_ops,
    .owner               = THIS_MODULE,
};

/* ---------- claiming devices via fwspec + PCI bus notifier ---------- */

static int fake_iommu_set_fwspec(struct device *dev)
{
    int ret;

    if (!fake_iommu_device_on_target_domain(dev))
        return 0;

    if (dev_iommu_fwspec_get(dev)) {
        dev_dbg(dev, "%s: device already has an iommu fwspec\n",
            DRIVER_NAME);
        return 0;
    }

    ret = iommu_fwspec_init(dev, fake_iommu_fwnode);
    if (ret) {
        dev_err(dev, "%s: iommu_fwspec_init failed: %d\n",
            DRIVER_NAME, ret);
        return ret;
    }

    return 0;
}

static int fake_iommu_bus_notify(struct notifier_block *nb,
                 unsigned long action, void *data)
{
    struct device *dev = data;

    if (action == BUS_NOTIFY_ADD_DEVICE)
        (void)fake_iommu_set_fwspec(dev);

    return NOTIFY_OK;
}

static int fake_iommu_register(void)
{
    int ret;

    ret = iommu_device_register(&fake_iommu_dev, &fake_iommu_ops,
                    &fake_iommu_pdev->dev);
    if (ret)
        return ret;

    /* Keep the fwnode used by the PCI bus notifier. */
    fake_iommu_dev.fwnode = fake_iommu_fwnode;
    return 0;
}

/* ---------- module init / exit ---------- */

/*
 * fwnode_create_software_node() needs at least an empty property array. We
 * don't expose any properties; the fwnode exists solely so that
 * iommu_device_register() has something to bind to.
 */
static const struct property_entry fake_iommu_props[] = {
    { }
};

static int __init fake_iommu_init(void)
{
    int ret;

    pr_info("%s: initializing for PCI domain 0x%x\n",
        DRIVER_NAME, FAKE_PCI_DOMAIN);

    /* 1. Software fwnode that the iommu_device will own. */
    fake_iommu_fwnode = fwnode_create_software_node(fake_iommu_props, NULL);
    if (IS_ERR(fake_iommu_fwnode)) {
        ret = PTR_ERR(fake_iommu_fwnode);
        fake_iommu_fwnode = NULL;
        pr_err("%s: fwnode_create_software_node failed: %d\n",
               DRIVER_NAME, ret);
        return ret;
    }

    /*
     * 2. A platform device acts as the hwdev parent for the iommu. The
     *    IOMMU subsystem expects a real struct device.
     */
    fake_iommu_pdev = platform_device_register_simple(DRIVER_NAME, -1,
                              NULL, 0);
    if (IS_ERR(fake_iommu_pdev)) {
        ret = PTR_ERR(fake_iommu_pdev);
        fake_iommu_pdev = NULL;
        pr_err("%s: platform_device_register_simple failed: %d\n",
               DRIVER_NAME, ret);
        goto err_fwnode;
    }

    /* 3. Sysfs entry under /sys/class/iommu/fake-iommu */
    ret = iommu_device_sysfs_add(&fake_iommu_dev, &fake_iommu_pdev->dev,
                     NULL, "fake-iommu");
    if (ret) {
        pr_err("%s: iommu_device_sysfs_add failed: %d\n",
               DRIVER_NAME, ret);
        goto err_pdev;
    }

    fake_iommu_dev.fwnode = fake_iommu_fwnode;

    /*
     * 4. Install our high-priority PCI bus notifier *before* registering
     *    the IOMMU. The kernel's own iommu bus notifier (priority 0) is
     *    installed inside iommu_device_register() and runs after ours,
     *    so by the time it probes a newly-added device, our notifier
     *    has already populated the fwspec.
     */
    pci_bus_nb.notifier_call = fake_iommu_bus_notify;
    pci_bus_nb.priority = 100;
    ret = bus_register_notifier(&pci_bus_type, &pci_bus_nb);
    if (ret) {
        pr_err("%s: bus_register_notifier(pci) failed: %d\n",
               DRIVER_NAME, ret);
        goto err_sysfs;
    }

    ret = fake_iommu_register();
    if (ret) {
        pr_err("%s: iommu_device_register failed: %d\n", DRIVER_NAME, ret);
        goto err_nb;
    }

    pr_info("%s: ready (claiming PCI devices on domain 0x%x)\n",
        DRIVER_NAME, FAKE_PCI_DOMAIN);
    return 0;

err_nb:
    bus_unregister_notifier(&pci_bus_type, &pci_bus_nb);
err_sysfs:
    iommu_device_sysfs_remove(&fake_iommu_dev);
err_pdev:
    platform_device_unregister(fake_iommu_pdev);
    fake_iommu_pdev = NULL;
err_fwnode:
    fwnode_remove_software_node(fake_iommu_fwnode);
    fake_iommu_fwnode = NULL;
    return ret;
}

static void __exit fake_iommu_exit(void)
{
    bus_unregister_notifier(&pci_bus_type, &pci_bus_nb);
    iommu_device_unregister(&fake_iommu_dev);
    iommu_device_sysfs_remove(&fake_iommu_dev);
    if (fake_iommu_pdev) {
        platform_device_unregister(fake_iommu_pdev);
        fake_iommu_pdev = NULL;
    }
    if (fake_iommu_fwnode) {
        fwnode_remove_software_node(fake_iommu_fwnode);
        fake_iommu_fwnode = NULL;
    }
    pr_info("%s: unloaded\n", DRIVER_NAME);
}

module_init(fake_iommu_init);
module_exit(fake_iommu_exit);

MODULE_DESCRIPTION("Software IOMMU for synthetic SR-IOV vGPU testing");
MODULE_LICENSE("GPL v2");
