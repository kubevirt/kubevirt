# Secondary Interface Naming Scheme Upgrade

Release:

- v1.8: Beta
- v1.10: GA

## Overview

VMs created on KubeVirt versions prior to v1.0.0 use a legacy ordinal naming
scheme (e.g., `net1`, `net2`) for secondary network interfaces attached via
[Multus](https://github.com/k8snetworkplumbingwg/multus-cni). Starting from
v1.0.0, KubeVirt switched to a hashed naming scheme (e.g., `podb1f25a43da3`)
for predictable and stable interface names.

To maintain network connectivity across live migrations and upgrades, KubeVirt
has preserved the legacy ordinal naming for VMs that were already using it.
However, the ordinal scheme is **incompatible with NIC hot-unplug** and
increases maintenance complexity.

Starting with v1.10, KubeVirt automatically upgrades affected VMs from the
ordinal naming scheme to the modern hashed naming scheme **without requiring
a VM restart**. The upgrade is a one-time process that also **unblocks
[NIC hot-unplug](https://kubevirt.io/user-guide/network/hotplug_interfaces/#removing-an-interface-from-a-running-vm)**
functionality for the VM.

## Who Is Affected

This upgrade only applies to VMs that were **originally created on KubeVirt
versions prior to v1.0.0** and are still using the ordinal naming scheme for
secondary network interfaces.

VMs created or restarted on v1.0.0 or later already use the hashed naming
scheme and are not affected.

## How It Works

When a VM with ordinal-named interfaces is live migrated (e.g., as part of a
KubeVirt update), the following happens:

1. The target virt-launcher pod is created with the hashed naming scheme,
   regardless of the source pod's naming scheme.
2. The domain XML is automatically adjusted on the target to map tap device
   names from the old ordinal format (e.g., `tap1`) to the new hashed format
   (e.g., `tapadd93534eeb`).
3. After a successful migration, the VMI status is updated to reflect the new
   interface names.

No manual intervention is required. Once a VM has been upgraded, subsequent
migrations proceed normally.
