# Fake SR-IOV vGPU Kernel Module

This kernel module creates fake PCI devices that appear in `/sys/bus/pci/devices/` to simulate NVIDIA SR-IOV Virtual Functions (VFs) with vGPU profiles assigned. It is designed to test KubeVirt's vGPU VF discovery (PR #16710) **without requiring real GPU hardware**.

## Purpose

PR #16710 adds support for discovering NVIDIA vGPU SR-IOV Virtual Functions that:
- Don't have the `vfio-pci` driver bound yet
- Have vGPU profiles assigned (detected via `nvidia/current_vgpu_type` sysfs file)
- Are not Physical Functions (no `virtfn*` subdirectories)

This module creates a **real virtual PCI bus** and registers fake devices on it, making them visible to standard PCI device discovery mechanisms including KubeVirt's virt-handler.

## Quick Start

### 1. Build the Module

```bash
cd kubevirtci/cluster-up/cluster/kind-1.33-vgpu/fake-sriov-vgpu

# Build using container (ensures kernel version match)
docker run --rm \
  -v $(pwd):/src:Z \
  quay.io/kubevirtci/bootstrap:v20251218-e7a7fc9 \
  bash -c 'dnf install -y kernel-devel kernel-headers >/dev/null 2>&1 && cd /src && make KDIR=/usr/src/kernels/$(ls /usr/src/kernels/ | head -1) modules'
```

### 2. Setup Fake VFs (requires sudo)

```bash
sudo ./setup-fake-sriov.sh setup

# Output shows:
# - Module loaded
# - 4 VF devices created
# - Devices appear in /sys/bus/pci/devices/0001:00:XX.X
```

### 3. Verify PCI Devices

```bash
# List fake PCI devices (domain 0001)
ls -la /sys/bus/pci/devices/0001:*

# Check device attributes
cat /sys/bus/pci/devices/0001:00:00.0/vendor    # 0x10de (NVIDIA)
cat /sys/bus/pci/devices/0001:00:00.0/device    # 0x1eb8 (Tesla T4)
cat /sys/bus/pci/devices/0001:00:00.0/nvidia/current_vgpu_type  # 256
```

### 4. Cleanup

```bash
sudo ./setup-fake-sriov.sh cleanup
```

## How It Works

The module creates a **virtual PCI bus** at domain `0001` and registers fake PCI devices on it:

1. **Virtual PCI Bus**: Created using `pci_create_root_bus()` with fake I/O and memory resources
2. **Fake PCI Devices**: Registered using `pci_scan_single_device()` and `pci_bus_add_device()`
3. **Config Space**: Each device has a PCI config space that reports NVIDIA Tesla T4 IDs
4. **nvidia/ sysfs**: Each device has `nvidia/current_vgpu_type` attribute for vGPU profile detection

### Device Structure

```
/sys/bus/pci/devices/0001:00:00.0/
├── vendor              # 0x10de (NVIDIA)
├── device              # 0x1eb8 (Tesla T4)
├── class               # 0x030000 (Display controller)
├── subsystem_vendor    # 0x10de
├── subsystem_device    # 0x12a2
└── nvidia/
    └── current_vgpu_type  # vGPU profile (0 = none, non-zero = assigned)
```

## Testing PR #16710

### KubeVirt Configuration

Add the fake devices to the permitted host devices:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
spec:
  configuration:
    permittedHostDevices:
      pciHostDevices:
      - pciVendorSelector: "10de:1eb8"
        resourceName: "nvidia.com/VGPU_T4"
```

### Test Scenarios

| Scenario | VF Config | Expected Result |
|----------|-----------|-----------------|
| VF with vGPU profile | `vgpu_type=256` | **Discovered** (new PR #16710 behavior) |
| VF without vGPU profile | `vgpu_type=0` | Skipped (no profile assigned) |

### Manual VF Management

```bash
# Create a VF with vGPU profile (slot 5, func 0, vgpu_type 256)
sudo ./setup-fake-sriov.sh create-vf 5 0 256

# Create a VF without vGPU profile (should be skipped by discovery)
sudo ./setup-fake-sriov.sh create-vf 6 0 0

# Remove a VF
sudo ./setup-fake-sriov.sh remove-vf 5 0

# Update vGPU type on existing device
echo 512 > /sys/bus/pci/devices/0001:00:00.0/nvidia/current_vgpu_type
```

## Requirements

- Linux kernel 5.10 or later
- Linux kernel headers (matching your running kernel)
- Build tools: `make`, `gcc`
- Root privileges for loading the module

## Troubleshooting

### Module fails to load with "Invalid module format"

Kernel version mismatch. Rebuild the module for your running kernel:

```bash
uname -r  # Check your kernel version
make clean && make
```

### Module crashes during load

Check dmesg for errors. The module may need architecture-specific adjustments for your kernel.

### VFs not being discovered by KubeVirt

1. Check that the `permittedHostDevices` CR includes the NVIDIA T4 PCI ID (`10de:1eb8`)
2. Verify the VF has a vGPU profile: `cat /sys/bus/pci/devices/0001:00:00.0/nvidia/current_vgpu_type`
3. The value must be non-zero for the new PR #16710 logic to discover it

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FAKE_SRIOV_VFS` | 4 | Number of VF devices to create |
| `FAKE_SRIOV_VGPU_TYPE` | 256 | vGPU type value (0 = no profile) |

## License

### Kernel Module Files (GPL-2.0-only)

| File | License | Notes |
|------|---------|-------|
| `fake-sriov-vgpu.c` | GPL-2.0-only | Kernel module source |
| `compat.h` | GPL-2.0-only | Compatibility header |
| `Makefile` | GPL-2.0-only | Build configuration |
| `dkms.conf` | GPL-2.0-only | DKMS configuration |

### Infrastructure Files (Apache-2.0)

| File | License | Notes |
|------|---------|-------|
| `setup-fake-sriov.sh` | Apache-2.0 | KubeVirt test infrastructure |
| `README.md` | Apache-2.0 | Documentation |

## References

- [KubeVirt PR #16710](https://github.com/kubevirt/kubevirt/pull/16710) - vGPU SR-IOV support
- [NVIDIA vGPU Documentation](https://docs.nvidia.com/grid/)
- [Linux PCI Driver API](https://docs.kernel.org/PCI/pci.html)
