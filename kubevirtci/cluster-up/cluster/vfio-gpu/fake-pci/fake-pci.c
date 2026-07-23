// SPDX-License-Identifier: GPL-2.0
/*
 * Example DRA PCI device emulation for KubeVirt testing
 *
 * This module registers a synthetic PCI host bridge in a dedicated PCI domain
 * (default 0xfaca) and populates it with N example PCI devices using
 * synthetic vendor/device/subsystem IDs (default 0xe1a5:0xd0c5/0xd0c5). The
 * devices appear under /sys/bus/pci/devices/<domain>:00:XX.0/ with proper
 * vendor/device/class/subsystem fields, so they are discoverable by:
 *
 *   - lspci (with -D); the default IDs show up as unregistered synthetic
 *     vendor/device IDs unless overridden by module parameters
 *   - KubeVirt's PermittedHostDevices listing
 *   - A DRA driver that scans /sys/bus/pci/devices/ to publish a ResourceSlice
 *   - The pciBusID metadata path in pkg/virt-launcher/virtwrap/device/
 *     hostdevice/dra/gpu_hostdev.go
 *
 * Scope and limitations:
 *
 *   - BARs are advertised as "no resource" (size 0). The PCI core does not
 *     allocate iomem windows for these devices. There is no real DMA backing;
 *     the demo targets DRA discovery, vfio-pci binding, and VMI lifecycle
 *     (with fake-iommu), not guest-visible GPU functionality.
 *   - All synthesized devices are identical by default. Override the
 *     vendor_id, device_id, and subsys_id module parameters to model the
 *     specific hardware your tests need (e.g. mimic a known vendor/SKU
 *     pair to exercise a driver that filters on vendor ID).
 *   - Devices appear in their own PCI domain (default 0xfaca) to avoid
 *     colliding with the real PCI hierarchy at domain 0x0000. Requires a
 *     kernel built with CONFIG_PCI_DOMAINS=y (true for x86_64 / arm64).
 *     The private domain is set via bridge->domain_nr on kernels with
 *     CONFIG_PCI_DOMAINS_GENERIC=y, or via an attached struct pci_sysdata
 *     on x86 builds where GENERIC is off (Ubuntu's default). See compat.h.
 *
 * This module is not a direct copy of one kernel driver; it follows patterns
 * from samples/vfio-mdev/mdpy.c and mtty.c (https://github.com/torvalds/linux/tree/master/samples/vfio-mdev/mdpy.c and https://github.com/torvalds/linux/tree/master/samples/vfio-mdev/mtty.c) and 
 * standard pci_host_bridge + pci_ops dummy-bus bring-up (https://github.com/torvalds/linux/tree/master/drivers/pci/host-bridge.c). 
 * Unlike those mdev samples, this publishes real pci_dev nodes for vfio-pci/DRA discovery.
 *
 */

#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/slab.h>
#include <linux/pci.h>
#include <linux/numa.h>
#include <linux/string.h>
#include <linux/version.h>

#include "compat.h"

#ifdef FAKE_PCI_HAS_LINUX_UNALIGNED
#include <linux/unaligned.h>
#else
#include <asm/unaligned.h>
#endif

/*
 * On x86 builds without CONFIG_PCI_DOMAINS_GENERIC the PCI domain is read
 * from ((struct pci_sysdata *)bus->sysdata)->domain rather than from
 * bridge->domain_nr. Include the arch header so we can stamp our own
 * domain into a struct pci_sysdata that we attach to the bridge.
 */
#if !defined(CONFIG_PCI_DOMAINS_GENERIC) && \
    (defined(CONFIG_X86) || defined(CONFIG_X86_64))
#include <asm/pci.h>
#define FAKE_PCI_USE_X86_SYSDATA 1
#endif

#define DRIVER_NAME             "fake_pci"
#define DRIVER_VERSION          "1.0"

/*
 * Default vendor/device/subsystem IDs are intentionally synthetic.
 * 0xe1a5 / 0xd0c5 are unassigned in the public pci.ids database and are
 * used only as stable local test IDs. They should not be presented as a
 * real vendor/device pair. Override via module parameters when a specific
 * real-world vendor/device pair is needed.
 */
#define FAKE_PCI_VENDOR_ID      0xe1a5
#define FAKE_PCI_DEVICE_ID      0xd0c5
#define FAKE_PCI_SUBSYS_ID      0xd0c5

#define FAKE_PCI_DEFAULT_DOMAIN   0xfaca
#define FAKE_PCI_DEFAULT_DEVICES  4
#define FAKE_PCI_MAX_DEVICES      32

#define FAKE_PCI_CONFIG_SIZE      256

#define STORE_LE16(addr, val)   put_unaligned_le16((val), (addr))
#define STORE_LE32(addr, val)   put_unaligned_le32((val), (addr))

/* ---------- module parameters ---------- */

static unsigned int num_devices = FAKE_PCI_DEFAULT_DEVICES;
module_param(num_devices, uint, 0444);
MODULE_PARM_DESC(num_devices,
         "Number of fake PCI devices to expose (1.."
         __stringify(FAKE_PCI_MAX_DEVICES) ")");

static unsigned int pci_domain = FAKE_PCI_DEFAULT_DOMAIN;
module_param(pci_domain, uint, 0444);
MODULE_PARM_DESC(pci_domain,
         "PCI domain number for the synthetic bridge (default 0xfaca)");

static unsigned int vendor_id = FAKE_PCI_VENDOR_ID;
module_param(vendor_id, uint, 0444);
MODULE_PARM_DESC(vendor_id,
         "PCI vendor ID to emulate (default 0xe1a5, synthetic / unassigned)");

static unsigned int device_id = FAKE_PCI_DEVICE_ID;
module_param(device_id, uint, 0444);
MODULE_PARM_DESC(device_id,
         "PCI device ID to emulate (default 0xd0c5, synthetic)");

static unsigned int subsys_id = FAKE_PCI_SUBSYS_ID;
module_param(subsys_id, uint, 0444);
MODULE_PARM_DESC(subsys_id,
         "PCI subsystem ID to emulate (default 0xd0c5, synthetic)");

/* ---------- module state ---------- */

struct fake_pci_dev_state {
    u8 config[FAKE_PCI_CONFIG_SIZE];
};

static struct fake_pci_state {
    struct fake_pci_dev_state *devices; /* num_devices entries */

    struct pci_host_bridge *bridge;
    struct pci_bus *bus;
    bool bus_present;

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

/* ---------- config space synthesis ---------- */

static void fake_pci_init_config(struct fake_pci_dev_state *d)
{
    u8 *c = d->config;

    memset(c, 0, FAKE_PCI_CONFIG_SIZE);

    STORE_LE16(&c[PCI_VENDOR_ID], vendor_id);
    STORE_LE16(&c[PCI_DEVICE_ID], device_id);

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

    /*
     * Leave BAR0..BAR5 at 0. Our write callback enforces that BAR sizing
     * (write 0xFFFFFFFF then read back) yields 0, meaning "BAR not
     * implemented". The PCI core will not allocate iomem for these
     * devices, which is what we want: no real backing memory.
     */

    STORE_LE16(&c[PCI_SUBSYSTEM_VENDOR_ID], vendor_id);
    STORE_LE16(&c[PCI_SUBSYSTEM_ID], subsys_id);

    /* Capabilities pointer */
    c[PCI_CAPABILITY_LIST] = 0x60;

    c[PCI_INTERRUPT_LINE] = 0xff;
    c[PCI_INTERRUPT_PIN]  = 0x01;

    /* PM capability @ 0x60 -> next 0x68 */
    c[0x60] = PCI_CAP_ID_PM;
    c[0x61] = 0x68;
    STORE_LE16(&c[0x62], 0x0003);  /* PMC */
    STORE_LE16(&c[0x64], 0x0000);  /* PMCSR */

    /* MSI capability @ 0x68 -> next 0x78 */
    c[0x68] = PCI_CAP_ID_MSI;
    c[0x69] = 0x78;
    STORE_LE16(&c[0x6a], 0x0080);  /* 64-bit capable, disabled */

    /* PCI Express capability @ 0x78 -> end of list */
    c[0x78] = PCI_CAP_ID_EXP;
    c[0x79] = 0x00;
    STORE_LE16(&c[0x7a], 0x0002);
    STORE_LE32(&c[0x7c], 0x00000010);
}

/* ---------- pci_ops ---------- */

static int fake_pci_dev_index(unsigned int devfn)
{
    unsigned int slot = PCI_SLOT(devfn);
    unsigned int func = PCI_FUNC(devfn);

    if (func != 0)
        return -1;
    if (slot >= num_devices)
        return -1;
    return (int)slot;
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

    cfg = fpci.devices[idx].config;
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

    cfg = fpci.devices[idx].config;

    /*
     * Restrict writes to fields that real PCI devices honor. Everything
     * else is silently dropped to keep our synthetic state stable.
     *
     * BAR sizing: writing 0xFFFFFFFF to a BAR is the standard "size me"
     * probe. We always store 0, so the readback is 0 and the PCI core
     * concludes the BAR is unimplemented. This avoids resource allocation
     * for memory we cannot back.
     */
    switch (where) {
    case PCI_COMMAND:
        if (size == 2)
            STORE_LE16(&cfg[where], (u16)val);
        break;

    case PCI_BASE_ADDRESS_0:
    case PCI_BASE_ADDRESS_1:
    case PCI_BASE_ADDRESS_2:
    case PCI_BASE_ADDRESS_3:
    case PCI_BASE_ADDRESS_4:
    case PCI_BASE_ADDRESS_5:
        if (size == 4)
            STORE_LE32(&cfg[where], 0);
        break;

    case 0x64:    /* PM cap base (0x60) + PCI_PM_CTRL (0x04) = PMCSR */
        /*
         * vfio-pci.probe() does pci_set_power_state(D3hot) immediately
         * after binding and reads PMCSR back to verify the transition.
         * If we drop writes, the readback returns D0, the kernel logs
         *   "Refused to change power state from D0 to D3hot"
         * and (on Ubuntu's 6.8.0-90 kernel) probe fails with -EINVAL.
         *
         * Honor 2-byte writes to PMCSR. We track only the PowerState
         * bits [1:0]; PME_En/PME_Status/Data_* are advertised as
         * unsupported in PMC (we set PMC=0x0003: version=3, no D1/D2
         * and PME_Support=0), so dev->pme_support is 0 and the kernel
         * never touches the upper bits.
         */
        if (size == 2) {
            u16 cur = get_unaligned_le16(&cfg[where]);

            cur = (cur & ~0x0003u) | ((u16)val & 0x0003u);
            STORE_LE16(&cfg[where], cur);
        }
        break;

    default:
        /* Drop other writes */
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
    unsigned int i;
    int ret;

    bridge = pci_alloc_host_bridge(0);
    if (!bridge)
        return -ENOMEM;

    bridge->ops      = &fake_pci_ops;
    bridge->busnr    = 0;
    bridge->dev.parent = NULL;

    /*
     * Place our synthetic bus in its own PCI domain so the BDFs we
     * synthesize do not collide with the real PCI hierarchy at domain
     * 0x0000. Two code paths get us there, picked at build time by
     * compat.h:
     *
     *   - CONFIG_PCI_DOMAINS_GENERIC=y: pci_register_host_bridge()
     *     honors bridge->domain_nr directly.
     *   - x86 with GENERIC=n (Ubuntu's default): pci_domain_nr(bus)
     *     reads ((struct pci_sysdata *)bus->sysdata)->domain. We stamp
     *     our domain into a struct pci_sysdata embedded in our module
     *     state and hand a pointer to it via bridge->sysdata.
     */
#ifdef FAKE_PCI_USE_X86_SYSDATA
    memset(&fpci.sysdata, 0, sizeof(fpci.sysdata));
    fpci.sysdata.domain = (int)pci_domain;
    fpci.sysdata.node   = NUMA_NO_NODE;
    bridge->sysdata     = &fpci.sysdata;
#else
    bridge->sysdata     = NULL;
    bridge->domain_nr   = (int)pci_domain;
#endif

    /* Reset list pointers so we can re-use the static resource */
    fake_pci_busn_res.parent  = NULL;
    fake_pci_busn_res.child   = NULL;
    fake_pci_busn_res.sibling = NULL;

    pci_add_resource(&bridge->windows, &fake_pci_busn_res);

    ret = pci_host_probe(bridge);
    if (ret) {
        pr_err("%s: pci_host_probe failed: %d\n", DRIVER_NAME, ret);
        pci_free_host_bridge(bridge);
        return ret;
    }

    fpci.bridge      = bridge;
    fpci.bus         = bridge->bus;
    fpci.bus_present = true;

    pr_info("%s: bridge up on domain 0x%x, %u device(s)\n",
        DRIVER_NAME, pci_domain, num_devices);
    for (i = 0; i < num_devices; i++)
        pr_info("%s:   %04x:00:%02x.0 [%04x:%04x]\n",
            DRIVER_NAME, pci_domain, i, vendor_id, device_id);

    return 0;
}

static void fake_pci_tear_down(void)
{
    if (!fpci.bus_present)
        return;

    pci_lock_rescan_remove();
    pci_stop_root_bus(fpci.bus);
    pci_remove_root_bus(fpci.bus);
    pci_unlock_rescan_remove();
    fpci.bridge      = NULL;
    fpci.bus         = NULL;
    fpci.bus_present = false;

    pr_info("%s: bridge torn down\n", DRIVER_NAME);
}

/* ---------- module init / exit ---------- */

static int __init fake_pci_init(void)
{
    unsigned int i;
    int ret;

    pr_info("%s: initializing (vendor=%#x device=%#x num=%u domain=%#x)\n",
        DRIVER_NAME, vendor_id, device_id, num_devices, pci_domain);

    if (num_devices == 0 || num_devices > FAKE_PCI_MAX_DEVICES) {
        pr_err("%s: invalid num_devices %u, must be 1..%d\n",
               DRIVER_NAME, num_devices, FAKE_PCI_MAX_DEVICES);
        return -EINVAL;
    }

    memset(&fpci, 0, sizeof(fpci));

    fpci.devices = kcalloc(num_devices, sizeof(*fpci.devices), GFP_KERNEL);
    if (!fpci.devices)
        return -ENOMEM;

    for (i = 0; i < num_devices; i++)
        fake_pci_init_config(&fpci.devices[i]);

    ret = fake_pci_bring_up();
    if (ret)
        goto err_free;

    pr_info("%s: ready (%u device(s) on domain 0x%x)\n",
        DRIVER_NAME, num_devices, pci_domain);
    return 0;

err_free:
    kfree(fpci.devices);
    fpci.devices = NULL;
    return ret;
}

static void __exit fake_pci_exit(void)
{
    fake_pci_tear_down();

    kfree(fpci.devices);
    fpci.devices = NULL;

    pr_info("%s: unloaded\n", DRIVER_NAME);
}

module_init(fake_pci_init);
module_exit(fake_pci_exit);

MODULE_DESCRIPTION("Example DRA PCI device emulation for KubeVirt testing");
MODULE_LICENSE("GPL v2");
MODULE_VERSION(DRIVER_VERSION);
MODULE_AUTHOR("KubeVirt Fake PCI Driver");
