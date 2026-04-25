# Upgrading Secondary Interface Naming Scheme

Release:

- v1.8: Beta

## Overview

VMs created on KubeVirt versions prior to v1.0.0 use a legacy ordinal naming
scheme (e.g., `net1`, `net2`) for secondary network interfaces attached via
[Multus](https://github.com/k8snetworkplumbingwg/multus-cni). Starting from
v1.0.0, KubeVirt switched to a hashed naming scheme (e.g., `podb1f25a43da3`)
for predictable and stable interface names.

To maintain network connectivity across live migrations and upgrades, KubeVirt
has preserved the legacy ordinal naming for VMs that were already using it.
However, the ordinal scheme is **incompatible with 
[NIC hot-unplug](https://kubevirt.io/user-guide/network/hotplug_interfaces/#removing-an-interface-from-a-running-vm)** 
and increases maintenance complexity.

Starting with v1.8, KubeVirt provides an upgrade path to migrate these VMs
from the ordinal naming scheme to the modern hashed naming scheme **without
requiring a VM restart**.

## Feature Gate

This feature requires the following
[feature gates](https://kubevirt.io/user-guide/cluster_admin/activating_feature_gates/#how-to-activate-a-feature-gate):

- `PodSecondaryInterfaceNamingUpgrade` - controls the naming scheme upgrade
  logic. Introduced in v1.8 at Beta stage, it must be explicitly enabled.
- `LibvirtHooksServerAndClient` - enables the domain mutation hook mechanism
  used to adjust tap device names during migration. Must be explicitly enabled.

## How It Works

When the feature gate is enabled, the naming scheme upgrade is performed
automatically during the next **live migration** of the VM:

1. The target virt-launcher pod is created with the hashed naming scheme,
   regardless of the source pod's naming scheme.
2. The domain XML is automatically adjusted on the target to map tap device
   names from the old ordinal format (e.g., `tap1`) to the new hashed format
   (e.g., `tapadd93534eeb`).
3. After a successful migration, the VMI status is updated to reflect the new
   interface names.

No manual intervention is required beyond triggering a live migration.

## Who Is Affected

This feature only applies to VMs that were **originally created on KubeVirt
versions prior to v1.0.0** and are still using the ordinal naming scheme for
secondary network interfaces.

VMs created on v1.0.0 or later already use the hashed naming scheme and are
not affected.

> **Note**: If a VM was created prior to v1.0.0 but has been restarted on
> v1.0.0 or above, it already uses the hashed naming scheme and does not
> require this upgrade.

## Upgrading a VM

To upgrade a VM's interface naming scheme, ensure both the
`PodSecondaryInterfaceNamingUpgrade` and `LibvirtHooksServerAndClient` feature
gates are enabled, then live migrate the VM:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstanceMigration
metadata:
  name: upgrade-naming-scheme
spec:
  vmiName: <vm-name>
EOF
```

Please refer to the [Live Migration](https://kubevirt.io/user-guide/compute/live_migration/) documentation
for more information.

Once the migration completes, the VM will use the hashed naming scheme. This
also **unblocks NIC hot-unplug** functionality for the VM.
