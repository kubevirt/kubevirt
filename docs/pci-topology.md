# PCI Topology for Hotplug Port Reservation

## Problem

Virtual machines on q35 machine types use PCIe root ports for device attachment.
Each device (disk, network interface, controller, etc.) occupies one root port.
To support hotplugging devices after boot, empty root ports must be reserved
at VM creation time — libvirt does not allow adding root ports to a running VM.

The number and method of reserving these ports directly affects the PCI bus
addresses assigned to devices. If the reservation strategy changes across a
reboot, devices shift to different PCI addresses. This breaks:

- **Windows VMs**: Windows marks non-OS disks as offline when they appear at
  new PCI addresses (SAN policy `OfflineShared`).
- **Device identity**: Applications that reference devices by PCI address
  (udev rules, DPDK bindings) break when addresses change.

## Topology Versions

### v1: Placeholder Interfaces (Original)

Reserves ports by injecting temporary placeholder network interfaces before
the first domain definition. Libvirt assigns root ports to these placeholders,
then they are removed in a second definition pass, leaving empty ports.

**Formula**: `max(0, 4 - len(interfaces))`

- 0 interfaces → 0 placeholders (early return)
- 1 interface → 3 placeholders
- 2 interfaces → 2 placeholders
- 3 interfaces → 1 placeholder
- 4+ interfaces → 0 placeholders

**Example** (1 interface, cirros VM):

```
Bus 0x01: Network interface
Bus 0x02: (empty placeholder)
Bus 0x03: (empty placeholder)
Bus 0x04: (empty placeholder)
Bus 0x05: SCSI controller
Bus 0x06: virtio-serial controller
Bus 0x07: Root disk (vda)          ← stable address
Bus 0x08: Memory balloon
```

3 free ports for hotplug. Maximum 4 regardless of VM size.

### v2: Memory-Scaled Placeholders (PR #14754)

Increased hotplug capacity by scaling placeholder count based on VM memory
and device count.

**Formula**:
```
if memory > 2GB:
    max(16 - portsInUse, 6)
else:
    max(8 - portsInUse, 3)
```

**Example** (1 interface, >2GB memory, 7 ports in use):

```
Bus 0x01: Network interface
Bus 0x02: (empty placeholder)
Bus 0x03: (empty placeholder)
Bus 0x04: (empty placeholder)
Bus 0x05: (empty placeholder)
Bus 0x06: (empty placeholder)
Bus 0x07: (empty placeholder)
Bus 0x08: (empty placeholder)
Bus 0x09: (empty placeholder)
Bus 0x0a: (empty placeholder)
Bus 0x0b: SCSI controller
Bus 0x0c: virtio-serial controller
Bus 0x0d: Root disk (vda)          ← SHIFTED from 0x07
Bus 0x0e: Memory balloon
```

9 free ports, but disk moved from bus 0x07 to 0x0d.

**Why v2 is unstable**: The placeholder count depends on `portsInUse`, which
changes when disks or interfaces are added/removed from the VM spec. Every
spec change can shift ALL device addresses — even without an upgrade.

### v3: Placeholder Interfaces + Direct Controllers (Current)

Uses the v1 placeholder formula for address stability, then adds direct
`pcie-root-port` controllers for additional hotplug capacity. Controllers
sit on bus 0 slots and provide new buses for devices, but libvirt assigns
devices to root ports independently of how many controllers exist — so
adding controllers does not shift any device addresses.

**Placeholder formula**: Same as v1: `max(0, 4 - len(interfaces))`

**Extra controller formula**:
```
if memory > 2GB:
    extra = max(0, max(16 - portsInUse, 6) - placeholderCount)
else:
    extra = max(0, max(8 - portsInUse, 3) - placeholderCount)
```

**Three-pass domain definition**:
1. Define with placeholder interfaces → libvirt assigns root ports
2. Redefine without placeholders → leaves empty ports
3. Redefine with extra controllers appended → adds hotplug capacity

**Example** (1 interface, >2GB memory, 7 ports in use):

```
Bus 0x01: Network interface
Bus 0x02: (empty placeholder)
Bus 0x03: (empty placeholder)
Bus 0x04: (empty placeholder)
Bus 0x05: SCSI controller
Bus 0x06: virtio-serial controller
Bus 0x07: Root disk (vda)          ← SAME as v1
Bus 0x08: Memory balloon
Bus 0x09: (extra controller)
Bus 0x0a: (extra controller)
Bus 0x0b: (extra controller)
Bus 0x0c: (extra controller)
Bus 0x0d: (extra controller)
Bus 0x0e: (extra controller)
```

Same 9 free ports as v2, same device addresses as v1.

## Annotations

Two annotations control PCI topology behavior:

| Annotation | Values | Set by | Purpose |
|---|---|---|---|
| `kubevirt.io/pci-topology-version` | `v2`, `v3` | Webhook, virt-handler | Documents which topology scheme is in use |
| `kubevirt.io/pci-interface-slot-count` | Integer string (e.g. `"11"`) | virt-handler | Frozen total of placeholders + boot-time interfaces for v2 VMs |

### Who Sets What

- **VMI mutating webhook** (CREATE): Sets version to `v3` if absent
- **VM mutating webhook** (CREATE): Sets version to `v3` on template if absent
- **virt-handler**: Detects version for running VMIs without annotation (upgrade window only), sets `v2` + placeholder count or `v3`
- **virt-controller**: Propagates annotations from VMI to VM template (one-time, only when VM template has no version annotation)

### Annotation Flow

```
VM Created (webhook sets v3 on template)
    │
    ▼
VM Started (template annotations propagate to VMI)
    │
    ▼
virt-launcher reads VMI annotations
    │
    ├── v3 or absent → v1 placeholder formula + extra controllers
    └── v2 + slot count → placeholders = max(0, slotCount - interfaces) + extra controllers
```

## Upgrade Behavior

### Running v1 VM (no annotation)

1. virt-handler inspects domain XML
2. Detects placeholder count matches v1 formula → annotates VMI as `v3`
3. virt-controller propagates `v3` to VM template
4. Next boot: uses v1 placeholders + controllers → same addresses

### Running v2 VM (no annotation)

1. virt-handler inspects domain XML
2. Detects placeholder count differs from v1 formula → annotates VMI as `v2` with frozen slot count (placeholders + boot-time interfaces)
3. virt-controller propagates `v2` + slot count to VM template
4. Next boot: derives placeholders as `max(0, slotCount - currentInterfaces)` → same addresses
5. Adding/removing interfaces while stopped is absorbed: the slot count stays constant, placeholder count adjusts automatically

### Stopped v1 VM

1. Starts → webhook sets `v3` (v3 is compatible with v1)
2. Uses v1 placeholders + controllers → same addresses

### Stopped v2 VM

1. Starts → webhook sets `v3` (no running domain to detect v2)
2. Uses v1 placeholders → **one-time address shift**
3. Acceptable since v2 was already unstable across spec changes

## Detection Logic

virt-handler distinguishes v1 from v2 by comparing the detected placeholder
count in the running domain against the v1 formula result:

1. Count pcie-root-port controllers in domain XML
2. Identify which root port buses have devices attached
3. Find the highest occupied bus
4. Count empty root ports at or below that bus (ports above are libvirt spares)
5. Add back hotplugged devices that consumed root ports at runtime:
   - Hotplugged volumes are identified via `HotplugVolume` on VMI volume status
   - Hotplugged interfaces are identified by comparing current VMI interfaces
     against the boot-time VM spec in the ControllerRevision
6. Compare result to v1 formula: `max(0, 4 - len(interfaces))`
7. If different → v2. If same → v1 (annotate as v3).

For >2GB VMs this is always distinguishable: v1 gives 0-3, v2 gives ≥6.
For ≤2GB VMs both can give 3 — but that's harmless since the placeholder
count is the same either way.

## Clone and Snapshot Restore

VM clone and snapshot restore both preserve PCI topology annotations. The
annotations live on `spec.template.metadata.annotations`, which is captured
in the snapshot content and carried through to the new VM:

1. Clone snapshots the source VM, capturing its full spec including template
   annotations
2. The clone controller generates patches from the snapshot — template
   annotations are preserved unless the user specifies
   `cloneSpec.Template.AnnotationFilters` to remove them
3. When the new VM is created, the mutating webhook fires but
   `setDefaultPciTopologyVersion` checks if the annotation already exists
   on the template and returns early

**Result**: A v2 source produces a v2 clone with the same frozen placeholder
count and stable PCI addresses. A v3 source produces a v3 clone.

The only case where the clone gets a different topology is if the user
explicitly strips annotations via `cloneSpec.Template.AnnotationFilters`.
In that case the webhook sets v3 on the new VM, which may shift addresses
relative to the source. This is user-initiated and expected.

## Key Files

| File | Purpose |
|---|---|
| `staging/src/kubevirt.io/api/core/v1/types.go` | Annotation constants |
| `pkg/virt-launcher/virtwrap/manager.go` | `allocateHotplugPorts`, formulas, controller addition |
| `pkg/virt-launcher/virtwrap/network/nichotplug.go` | `WithNetworkIfacesResources` (two-pass define) |
| `pkg/virt-launcher/virtwrap/converter/pci-placement.go` | `CountPCIDevices` |
| `pkg/virt-api/webhooks/mutating-webhook/mutators/vmi-mutator.go` | VMI webhook |
| `pkg/virt-api/webhooks/mutating-webhook/mutators/vm-mutator.go` | VM webhook |
| `pkg/virt-handler/pci_topology.go` | Detection logic |
| `pkg/virt-controller/watch/vm/vm.go` | Annotation propagation |
