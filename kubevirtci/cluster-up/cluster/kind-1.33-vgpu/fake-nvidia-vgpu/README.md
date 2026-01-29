# Fake NVIDIA vGPU Kernel Module

This kernel module creates fake mediated devices (mdev) that simulate NVIDIA Tesla T4 vGPUs for KubeVirt testing **without requiring real GPU hardware**.

## Quick Start

### 1. Build the Module

```bash
cd kubevirtci/cluster-up/cluster/kind-1.33-vgpu/fake-nvidia-vgpu

# Using container (recommended - ensures kernel version match)
docker run --rm \
  -v $(pwd):/src:Z \
  quay.io/kubevirtci/bootstrap:v20251218-e7a7fc9 \
  bash -c 'dnf install -y kernel-devel >/dev/null 2>&1 && cd /src && make KDIR=/usr/src/kernels/$(ls /usr/src/kernels/ | head -1) modules'
```

### 2. Setup Fake vGPU (requires sudo)

```bash
cd kubevirtci/cluster-up/cluster/kind-1.33-vgpu
sudo ./setup-fake-vgpu-host.sh setup

# Verify mdev instances
ls /sys/bus/mdev/devices/
# Should show 4 UUIDs

# Verify hotplug control interface
cat /sys/class/nvidia/nvidia/hotplug_control
# Should show: visible
```

### 3. Bring Up the Cluster

```bash
cd /path/to/kubevirt
export KUBEVIRT_PROVIDER=kind-1.33-vgpu
make cluster-up
```

The provider will validate that the fake vGPU is properly set up before creating the cluster.

### 4. Deploy KubeVirt

```bash
make cluster-sync
```

### 5. Run the Tests

```bash
# Run MediatedDevices tests
KUBEVIRT_E2E_FOCUS="MediatedDevices" make functest

# Or run VGPU-specific tests (matching CI)
KUBEVIRT_E2E_FOCUS="VGPU" make functest
```

### 6. Cleanup

```bash
# Tear down cluster
make cluster-down

# Cleanup fake vGPU (removes mdev instances and unloads module)
cd kubevirtci/cluster-up/cluster/kind-1.33-vgpu
sudo ./setup-fake-vgpu-host.sh cleanup
```

---

## What It Provides

| mdev Type | Name | Max Instances | Simulated FB |
|-----------|------|---------------|--------------|
| `nvidia-222` | GRID T4-1B | 16 | 1GB |
| `nvidia-223` | GRID T4-2B | 8 | 2GB |

When an mdev instance is passed to a VM, the guest sees:
- **PCI Vendor ID**: `10de` (NVIDIA)
- **PCI Device ID**: `1eb8` (Tesla T4)

## Requirements

- **Linux kernel 5.16 or later** (uses new VFIO/mdev API)
- Linux kernel headers (matching your running kernel)
- Build tools: `make`, `gcc`
- Root privileges for loading the module

## Reloading the Module

If you modify the kernel module source code, rebuild and reload (cluster can stay up - module runs on host):

```bash
# 1. Rebuild (see step 1 above)
cd kubevirtci/cluster-up/cluster/kind-1.33-vgpu/fake-nvidia-vgpu
make clean
# Run the docker build command from step 1

# 2. Reload
cd ..
sudo ./setup-fake-vgpu-host.sh cleanup
sudo ./setup-fake-vgpu-host.sh setup

# 3. Verify
cat /sys/class/nvidia/nvidia/hotplug_control
# Should show: visible
```

## Hotplug Emulation

The module provides a sysfs interface to simulate device hot-plug/hot-unplug for testing virt-handler's device detection:

```bash
# Check current state
cat /sys/class/nvidia/nvidia/hotplug_control
# Shows: visible or hidden

# Hide device (simulate hot-unplug)
sudo ./setup-fake-vgpu-host.sh hide

# Show device (simulate hot-plug)
sudo ./setup-fake-vgpu-host.sh show
```

This is used by the MediatedDevices tests to verify virt-handler correctly creates mdev instances when devices appear.

## Troubleshooting

### Module fails to load with "Invalid module format"

Kernel version mismatch. Rebuild the module for your running kernel:

```bash
uname -r  # Check your kernel version
# Rebuild with matching kernel-devel
```

### Module fails to load with "Unknown symbol"

Missing dependencies. Load them first:

```bash
sudo modprobe vfio vfio_iommu_type1 mdev
```

### No /sys/class/mdev_bus/nvidia directory

Module didn't load properly. Check dmesg:

```bash
dmesg | tail -30
```

---

## How It Works

1. **Module loads** → Creates a fake parent device at `/sys/class/mdev_bus/nvidia/`
2. **mdev types registered** → `nvidia-222` and `nvidia-223` appear in `mdev_supported_types/`
3. **mdev instance created** → Writing UUID to `create` file triggers probe
4. **VFIO integration** → Instance registers as VFIO device, can be assigned to VMs
5. **Guest sees NVIDIA device** → PCI config space reports vendor `10de`, device `1eb8`
6. **Display plane support** → Reports framebuffer region to QEMU for ramfb display

## Source Code Attribution

This module is based on the **official Linux kernel sample mdev drivers**:

| File | Location | Purpose |
|------|----------|---------|
| `mdpy.c` | `samples/vfio-mdev/mdpy.c` | Mediated display device sample |
| `mtty.c` | `samples/vfio-mdev/mtty.c` | Mediated TTY device sample |

**Upstream source:** https://github.com/torvalds/linux/tree/master/samples/vfio-mdev

### What Was Changed

| Component | Original (samples) | This Module |
|-----------|-------------------|-------------|
| PCI Vendor ID | Various test IDs | `10de` (NVIDIA) |
| PCI Device ID | Various test IDs | `1eb8` (Tesla T4) |
| mdev type names | `1`, `2` | `nvidia-222`, `nvidia-223` |
| Pretty names | Generic | `GRID T4-1B`, `GRID T4-2B` |
| Display support | Basic | VFIO GFX plane for QEMU ramfb |

## License

### Kernel Module Files (GPL-2.0-only)

The following files are licensed under **GPL v2** (required for kernel modules that use kernel APIs):

| File | License | Based On |
|------|---------|----------|
| `fake-nvidia-vgpu.c` | GPL-2.0-only | Linux kernel samples `mdpy.c`, `mtty.c` |
| `compat.h` | GPL-2.0-only | Original work |
| `Makefile` | GPL-2.0-only | Standard kernel module makefile pattern |
| `dkms.conf` | GPL-2.0-only | Standard DKMS configuration |

**Upstream kernel samples:**
- https://github.com/torvalds/linux/blob/master/samples/vfio-mdev/mdpy.c
- https://github.com/torvalds/linux/blob/master/samples/vfio-mdev/mtty.c

### Infrastructure Files (Apache-2.0)

The following files are part of KubeVirt infrastructure and follow the **Apache 2.0** license:

| File | License | Notes |
|------|---------|-------|
| `setup-fake-vgpu.sh` | Apache-2.0 | KubeVirt test infrastructure |
| `../setup-fake-vgpu-host.sh` | Apache-2.0 | KubeVirt test infrastructure |
| `../provider.sh` | Apache-2.0 | KubeVirt provider script |
| `../config_vgpu_cluster.sh` | Apache-2.0 | KubeVirt cluster configuration |
| `../vgpu-node/node.sh` | Apache-2.0 | KubeVirt node helpers |
| `../OPEN_ISSUES.md` | Apache-2.0 | Documentation |
| `README.md` | Apache-2.0 | Documentation |

### Test Files (Apache-2.0)

| File | License | Notes |
|------|---------|-------|
| `tests/mdev_configuration_allocation_test.go` | Apache-2.0 | KubeVirt test code |
| `tests/BUILD.bazel` | Apache-2.0 | KubeVirt build configuration |

### License Texts

- **GPL-2.0**: https://www.gnu.org/licenses/old-licenses/gpl-2.0.html
- **Apache-2.0**: https://www.apache.org/licenses/LICENSE-2.0

## References

- [Linux mdev documentation](https://docs.kernel.org/driver-api/vfio-mediated-device.html)
- [KubeVirt mdev tests](../../../tests/mdev_configuration_allocation_test.go)
