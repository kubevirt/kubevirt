// SPDX-License-Identifier: GPL-2.0
/*
 * Synthetic NVIDIA SR-IOV PCI PF and four VFs for KubeVirt testing.
 * Adapted from kubevirtci's fake-pci module (commit c3d11ec0).
 * BARs have no backing and the device cannot perform GPU work.
 *
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/pci.h>
#include <linux/numa.h>
#include <linux/string.h>
#include <linux/unaligned.h>
#include <linux/irqdomain.h>
#include <linux/msi.h>


/*
 * On x86 builds without CONFIG_PCI_DOMAINS_GENERIC the PCI domain is read
 * from ((struct pci_sysdata *)bus->sysdata)->domain rather than from
 * bridge->domain_nr. Include the arch header so we can stamp our own
 * domain into a struct pci_sysdata that we attach to the bridge.
 */
#ifndef CONFIG_PCI_DOMAINS_GENERIC
#include <asm/pci.h>
#define FAKE_PCI_USE_X86_SYSDATA 1
#endif

#define DRIVER_NAME             "fake_pci"

#define FAKE_PCI_VENDOR_ID      0x10de
#define FAKE_PCI_DEVICE_ID      0x26b9
#define FAKE_PCI_SUBSYS_ID      0x16a1

#define FAKE_PCI_DEFAULT_DOMAIN   0xfaca

#define FAKE_PCI_CONFIG_SIZE      4096
#define FAKE_VF_COUNT             4
#define FAKE_BAR0_SIZE            (16 * 1024 * 1024)

#define STORE_LE16(addr, val)   put_unaligned_le16((val), (addr))
#define STORE_LE32(addr, val)   put_unaligned_le32((val), (addr))

/* ---------- module state ---------- */

static struct fake_pci_state {
    u8 config[FAKE_VF_COUNT + 1][FAKE_PCI_CONFIG_SIZE];

    bool vfs_enabled;

    struct pci_bus *bus;
    struct pci_dev *pf;
    struct irq_domain *msi_domain;

#ifdef FAKE_PCI_USE_X86_SYSDATA
    struct pci_sysdata sysdata;
#endif
} fpci;

/* Bus number window: a single bus, number 0, on our private domain. */
static struct resource fake_pci_busn_res = {
    .name  = "fake-pci-busn",
    .start = 0,
    .end   = 0,
    .flags = IORESOURCE_BUS,
};

static struct resource fake_pci_mem_res = {
    .name = "fake-pci-mem",
    .start = 0x10000000,
    .end = 0x14ffffff,
    .flags = IORESOURCE_MEM,
};

/* ---------- config space synthesis ---------- */

static void fake_pci_init_config(u8 *c, bool pf)
{

    STORE_LE16(&c[PCI_VENDOR_ID], FAKE_PCI_VENDOR_ID);
    STORE_LE16(&c[PCI_DEVICE_ID], FAKE_PCI_DEVICE_ID);

    /* Command: memory space + bus master enabled */
    STORE_LE16(&c[PCI_COMMAND], PCI_COMMAND_MEMORY | PCI_COMMAND_MASTER);
    /* Status: capabilities list present */
    STORE_LE16(&c[PCI_STATUS], PCI_STATUS_CAP_LIST);

    c[PCI_REVISION_ID]  = 0xa1;
    c[PCI_CLASS_PROG]   = 0x00;
    /* 0x0302 = Display controller / 3D controller */
    STORE_LE16(&c[PCI_CLASS_DEVICE], 0x0302);

    c[PCI_CACHE_LINE_SIZE] = 0x10;
    c[PCI_LATENCY_TIMER]   = 0x00;
    c[PCI_HEADER_TYPE]     = PCI_HEADER_TYPE_NORMAL;

    /* BAR0 is a synthetic 16 MiB resource for VFIO; it has no backing. */
    STORE_LE16(&c[PCI_SUBSYSTEM_VENDOR_ID], FAKE_PCI_VENDOR_ID);
    STORE_LE16(&c[PCI_SUBSYSTEM_ID], FAKE_PCI_SUBSYS_ID);
    STORE_LE32(&c[PCI_BASE_ADDRESS_0], PCI_BASE_ADDRESS_SPACE_MEMORY |
               PCI_BASE_ADDRESS_MEM_TYPE_32 | PCI_BASE_ADDRESS_MEM_PREFETCH);

    /* Capabilities pointer */
    c[PCI_CAPABILITY_LIST] = 0x50;

    /* No IRQ source exists in this software-only device. */
    c[PCI_INTERRUPT_LINE] = 0;
    c[PCI_INTERRUPT_PIN]  = 0;

    /* PM capability @ 0x60 -> PCIe capability @ 0x78. */
    c[0x50] = PCI_CAP_ID_MSI;
    c[0x51] = 0x60;
    STORE_LE16(&c[0x52], 0x0000);
    STORE_LE32(&c[0x54], 0x00000000);
    STORE_LE16(&c[0x58], 0x0000);
    c[0x60] = PCI_CAP_ID_PM;
    c[0x61] = 0x78;
    STORE_LE16(&c[0x62], 0x0003);  /* PMC */
    STORE_LE16(&c[0x64], 0x0000);  /* PMCSR */

    /* PCI Express capability @ 0x78 -> end of list */
    c[0x78] = PCI_CAP_ID_EXP;
    c[0x79] = 0x00;
    STORE_LE16(&c[0x7a], 0x0002);
    STORE_LE32(&c[0x7c], 0x00000010);

    if (pf) {
        /* PCIe SR-IOV extended capability: four VFs at functions 1-4. */
        STORE_LE32(&c[0x100], 0x00010010);
        STORE_LE16(&c[0x10c], FAKE_VF_COUNT);
        STORE_LE16(&c[0x10e], FAKE_VF_COUNT);
        STORE_LE16(&c[0x114], 1);
        STORE_LE16(&c[0x116], 1);
        STORE_LE16(&c[0x11a], FAKE_PCI_DEVICE_ID);
        STORE_LE32(&c[0x11c], 1);
        STORE_LE32(&c[0x120], 1);
    }
}

/* ---------- pci_ops ---------- */

static int fake_pci_dev_index(unsigned int devfn)
{
    unsigned int slot = PCI_SLOT(devfn);
    unsigned int func = PCI_FUNC(devfn);

    if (slot != 0 || func > FAKE_VF_COUNT)
        return -1;
    if (func && !fpci.vfs_enabled)
        return -1;
    return (int)func;
}

static int fake_pci_read(struct pci_bus *bus, unsigned int devfn,
             int where, int size, u32 *val)
{
    int idx;
    const u8 *cfg;

    /* Only bus 0 on our domain hosts devices */
    if (bus->number != 0) {
        *val = ~0U;
        return PCIBIOS_SUCCESSFUL;
    }

    idx = fake_pci_dev_index(devfn);
    if (idx < 0) {
        *val = ~0U;
        return PCIBIOS_SUCCESSFUL;
    }

    if (where < 0 || where + size > FAKE_PCI_CONFIG_SIZE) {
        *val = 0;
        return PCIBIOS_BAD_REGISTER_NUMBER;
    }

    cfg = fpci.config[idx];
    switch (size) {
    case 1:
        *val = cfg[where];
        break;
    case 2:
        *val = get_unaligned_le16(&cfg[where]);
        break;
    case 4:
        *val = get_unaligned_le32(&cfg[where]);
        break;
    default:
        return PCIBIOS_BAD_REGISTER_NUMBER;
    }

    return PCIBIOS_SUCCESSFUL;
}

static int fake_pci_write(struct pci_bus *bus, unsigned int devfn,
              int where, int size, u32 val)
{
    int idx;
    u8 *cfg;

    if (bus->number != 0)
        return PCIBIOS_SUCCESSFUL;

    idx = fake_pci_dev_index(devfn);
    if (idx < 0)
        return PCIBIOS_SUCCESSFUL;

    if (where < 0 || where + size > FAKE_PCI_CONFIG_SIZE)
        return PCIBIOS_BAD_REGISTER_NUMBER;

    cfg = fpci.config[idx];

    /* Only state used by PCI enumeration, SR-IOV and VFIO is writable. */
    switch (where) {
    case 0x108: /* SR-IOV control: VF enable bit */
        if (idx == 0 && size == 2) {
            STORE_LE16(&cfg[where], val);
            fpci.vfs_enabled = !!(val & PCI_SRIOV_CTRL_VFE);
        }
        break;
    case 0x110: /* Number of VFs */
        if (idx == 0 && size == 2)
            STORE_LE16(&cfg[where], val);
        break;
    case 0x120: /* System page size */
        if (idx == 0 && size == 4)
            STORE_LE32(&cfg[where], val);
        break;
    case PCI_COMMAND:
        if (size == 2)
            STORE_LE16(&cfg[where], (u16)val);
        break;

    case PCI_BASE_ADDRESS_0:
        if (size == 4) {
            u32 bar = val == 0xffffffff ? ~(FAKE_BAR0_SIZE - 1) :
                      val & PCI_BASE_ADDRESS_MEM_MASK;

            STORE_LE32(&cfg[where], bar | PCI_BASE_ADDRESS_MEM_PREFETCH);
        }
        break;

    case 0x64:    /* PM cap base (0x60) + PCI_PM_CTRL (0x04) = PMCSR */
        /* VFIO expects D3hot writes to be visible in PMCSR. */
        if (size == 2) {
            u16 cur = get_unaligned_le16(&cfg[where]);

            cur = (cur & ~0x0003u) | ((u16)val & 0x0003u);
            STORE_LE16(&cfg[where], cur);
        }
        break;

    default:
        break;
    }

    return PCIBIOS_SUCCESSFUL;
}

static struct pci_ops fake_pci_ops = {
    .read  = fake_pci_read,
    .write = fake_pci_write,
};

/* ---------- bus bring-up / tear-down ---------- */

static int fake_pci_bring_up(void)
{
    struct pci_host_bridge *bridge;
    int ret;

    bridge = pci_alloc_host_bridge(0);
    if (!bridge)
        return -ENOMEM;

    bridge->ops = &fake_pci_ops;
    bridge->busnr = 0;

    /*
     * Place our synthetic bus in its own PCI domain so the BDFs we
     * synthesize do not collide with the real PCI hierarchy at domain
     * 0x0000. The kernel configuration determines which PCI domain
     * field is used:
     *
     *   - CONFIG_PCI_DOMAINS_GENERIC=y: pci_register_host_bridge()
     *     honors bridge->domain_nr directly.
     *   - x86 with GENERIC=n: pci_domain_nr(bus)
     *     reads ((struct pci_sysdata *)bus->sysdata)->domain. We stamp
     *     our domain into a struct pci_sysdata embedded in our module
     *     state and hand a pointer to it via bridge->sysdata.
     */
#ifdef FAKE_PCI_USE_X86_SYSDATA
    memset(&fpci.sysdata, 0, sizeof(fpci.sysdata));
    fpci.sysdata.domain = (int)FAKE_PCI_DEFAULT_DOMAIN;
    fpci.sysdata.node   = NUMA_NO_NODE;
    bridge->sysdata     = &fpci.sysdata;
#else
    bridge->sysdata     = NULL;
    bridge->domain_nr   = (int)FAKE_PCI_DEFAULT_DOMAIN;
#endif

    pci_add_resource(&bridge->windows, &fake_pci_busn_res);
    pci_add_resource(&bridge->windows, &fake_pci_mem_res);

    ret = pci_host_probe(bridge);
    if (ret) {
        pr_err("%s: pci_host_probe failed: %d\n", DRIVER_NAME, ret);
        pci_free_host_bridge(bridge);
        return ret;
    }

    fpci.bus = bridge->bus;

    {
        const struct irq_domain_info info = {
            .domain_flags = IRQ_DOMAIN_FLAG_MSI | IRQ_DOMAIN_FLAG_ISOLATED_MSI,
            .size = 1, .hwirq_max = 0, .ops = &irq_domain_simple_ops,
            .dev = &bridge->dev,
        };
        fpci.msi_domain = irq_domain_instantiate(&info);
        if (IS_ERR(fpci.msi_domain)) {
            ret = PTR_ERR(fpci.msi_domain);
            fpci.msi_domain = NULL;
            goto err_bus;
        }
    }

    fpci.pf = pci_get_domain_bus_and_slot(FAKE_PCI_DEFAULT_DOMAIN, 0, 0);
    if (!fpci.pf) {
        ret = -ENODEV;
        goto err_bus;
    }
    dev_set_msi_domain(&fpci.pf->dev, fpci.msi_domain);
    ret = pci_enable_sriov(fpci.pf, FAKE_VF_COUNT);
    if (ret) {
        pr_err("%s: pci_enable_sriov failed: %d\n", DRIVER_NAME, ret);
        goto err_bus;
    }

    for (unsigned int fn = 0; fn <= FAKE_VF_COUNT; fn++) {
        struct pci_dev *pdev = pci_get_domain_bus_and_slot(FAKE_PCI_DEFAULT_DOMAIN, 0, fn);
        if (pdev) {
            pdev->resource[0].start = 0x10000000 + fn * FAKE_BAR0_SIZE;
            pdev->resource[0].end = pdev->resource[0].start + FAKE_BAR0_SIZE - 1;
            pdev->resource[0].flags = IORESOURCE_MEM;
            ret = pci_request_region(pdev, 0, DRIVER_NAME);
            /* VFIO requires the synthetic BAR to appear claimed. */
            pdev->resource[0].parent = &fake_pci_mem_res;
            pdev->resource[0].flags |= IORESOURCE_BUSY;
            pr_info("%s: %s BAR0 resource request ret=%d [%pa-%pa]\n",
                    DRIVER_NAME, pci_name(pdev), ret,
                    &pdev->resource[0].start, &pdev->resource[0].end);
            if (fn)
                dev_set_msi_domain(&pdev->dev, fpci.msi_domain);
            pci_dev_put(pdev);
        }
    }

    pr_info("%s: PF and %u VFs on domain 0x%x\n",
        DRIVER_NAME, FAKE_VF_COUNT, FAKE_PCI_DEFAULT_DOMAIN);

    return 0;

err_bus:
    if (fpci.pf) {
        dev_set_msi_domain(&fpci.pf->dev, NULL);
        pci_dev_put(fpci.pf);
        fpci.pf = NULL;
    }
    pci_lock_rescan_remove();
    pci_stop_root_bus(fpci.bus);
    pci_remove_root_bus(fpci.bus);
    pci_unlock_rescan_remove();
    if (fpci.msi_domain) {
        irq_domain_remove(fpci.msi_domain);
        fpci.msi_domain = NULL;
    }
    fpci.bus = NULL;
    return ret;
}

static void fake_pci_tear_down(void)
{
    if (!fpci.bus)
        return;

    pci_disable_sriov(fpci.pf);
    if (fpci.msi_domain)
        dev_set_msi_domain(&fpci.pf->dev, NULL);
    pci_dev_put(fpci.pf);
    fpci.pf = NULL;

    pci_lock_rescan_remove();
    pci_stop_root_bus(fpci.bus);
    pci_remove_root_bus(fpci.bus);
    pci_unlock_rescan_remove();
    if (fpci.msi_domain) {
        irq_domain_remove(fpci.msi_domain);
        fpci.msi_domain = NULL;
    }
    fpci.bus = NULL;

    pr_info("%s: bridge torn down\n", DRIVER_NAME);
}

/* ---------- module init / exit ---------- */

static int __init fake_pci_init(void)
{
    unsigned int i;

    for (i = 0; i <= FAKE_VF_COUNT; i++)
        fake_pci_init_config(fpci.config[i], i == 0);

    return fake_pci_bring_up();
}

static void __exit fake_pci_exit(void)
{
    fake_pci_tear_down();
}

module_init(fake_pci_init);
module_exit(fake_pci_exit);

MODULE_DESCRIPTION("Synthetic NVIDIA SR-IOV PF and VFs for KubeVirt testing");
MODULE_LICENSE("GPL v2");
