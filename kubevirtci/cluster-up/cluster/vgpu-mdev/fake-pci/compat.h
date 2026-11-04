/* SPDX-License-Identifier: GPL-2.0 */
/*
 * Kernel compatibility header for fake-pci module.
 *
 * The module is built against the running kernel and targets 5.14+. The PCI
 * host bridge APIs used here (pci_alloc_host_bridge, pci_host_probe,
 * pci_remove_root_bus) have been stable since well before 5.14, so only a
 * couple of small shims are needed.
 */

#ifndef _FAKE_PCI_COMPAT_H
#define _FAKE_PCI_COMPAT_H

#include <linux/version.h>

#if LINUX_VERSION_CODE < KERNEL_VERSION(5, 14, 0)
#error "This module requires kernel 5.14 or later"
#endif

#if LINUX_VERSION_CODE >= KERNEL_VERSION(6, 12, 0)
#define FAKE_PCI_HAS_LINUX_UNALIGNED 1
#endif

#ifndef CONFIG_PCI_DOMAINS
#error "This module requires CONFIG_PCI_DOMAINS=y (private PCI domain needed to avoid host bus collision)"
#endif

/*
 * Two paths get us a private PCI domain at register time:
 *
 *   1. CONFIG_PCI_DOMAINS_GENERIC=y - pci_register_host_bridge() honors
 *      pci_host_bridge->domain_nr directly. Fedora 39+, RHEL 9, arm64
 *      distros, and most non-x86 builds use this.
 *
 *   2. x86 (CONFIG_X86, CONFIG_X86_64) - even with GENERIC=n (Ubuntu's
 *      default for x86_64), the kernel reads the bus's domain from
 *      ((struct pci_sysdata *)bus->sysdata)->domain. We attach our own
 *      struct pci_sysdata to bridge->sysdata in fake-pci.c.
 *
 * If neither path is available (e.g. an arm32 build with GENERIC=n) the
 * module would land on domain 0 and collide with the real hierarchy, so
 * we hard-fail at build time.
 */
#if !defined(CONFIG_PCI_DOMAINS_GENERIC) && \
	!defined(CONFIG_X86) && !defined(CONFIG_X86_64)
#error "This module requires either CONFIG_PCI_DOMAINS_GENERIC=y or an x86 build"
#endif

#endif /* _FAKE_PCI_COMPAT_H */
