# Fake vGPU Provider - Open Issues

This document tracks known issues and limitations with the fake vGPU provider.

---

## Issue #1: vGPU Display Support Requires OpenGL

### Status: **Workaround Applied**

### Problem

When running MediatedDevices tests with the fake vGPU provider, VMs fail to start with:

```
vfio-display-dmabuf: opengl not available
```

### Root Cause

1. KubeVirt enables `display=on` and `ramfb=on` **by default** for vGPU devices (when no explicit `VirtualGPUOptions` is configured)
2. When `display=on` is set for vfio-pci, QEMU tries to use **DMABUF display mode**
3. DMABUF requires **OpenGL/EGL libraries** which are **not present** in the virt-launcher container
4. QEMU fails immediately before even probing the fake vGPU's display capabilities

The fake vGPU driver correctly advertises region-based display support (`VFIO_GFX_PLANE_TYPE_REGION`), but QEMU tries to initialize OpenGL context first and fails.

### Why It Works With Real NVIDIA vGPU

With **real NVIDIA vGPU**, display works because:

1. **The NVIDIA driver provides OpenGL/EGL support** through NVIDIA proprietary libraries (libEGL_nvidia.so, libGLESv2_nvidia.so, etc.)
2. **These libraries are mounted into the virt-launcher container** via device plugin mechanisms or host mounts configured by the NVIDIA GPU Operator
3. **DMABUF display mode works** because NVIDIA provides the OpenGL context that QEMU needs

With **fake vGPU**:
- No NVIDIA driver = no OpenGL libraries
- virt-launcher container has no software OpenGL fallback
- QEMU's DMABUF initialization fails immediately

### Possible Solutions

#### Option 1: Disable display for fake vGPU testing (Current Workaround)

Configure VMIs to explicitly disable display when using fake vGPU:

```go
vGPUs := []v1.GPU{
    {
        Name:       "gpu1",
        DeviceName: deviceName,
        VirtualGPUOptions: &v1.VirtualGPUOptions{
            Display: &v1.VGPUDisplayOptions{
                Enabled: pointer.Bool(false),  // Disable display
            },
        },
    },
}
```

**Pros:**
- Quick fix, no infrastructure changes needed
- Tests mdev allocation/configuration functionality

**Cons:**
- Cannot test display-related functionality
- Requires modifying tests or skipping display assertions

#### Option 2: Mount OpenGL libraries into virt-launcher (like NVIDIA driver does)

Mount mesa/OpenGL libraries into the virt-launcher container, similar to how the NVIDIA GPU Operator mounts NVIDIA libraries for real vGPUs.

**Prerequisites (common to all sub-options):**
1. Install mesa-dri-drivers on the kind node during cluster setup:
   ```bash
   dnf install -y mesa-dri-drivers mesa-libEGL mesa-libGL libglvnd-egl libglvnd-gles
   ```

**General Pros:**
- Would enable full display support including DMABUF
- Matches how real NVIDIA vGPU works (library mounting)
- No changes to upstream virt-launcher image
- Test-specific, doesn't affect production deployments

**General Cons:**
- Requires mesa packages on the kind node
- Adds complexity to the test infrastructure

---

##### Option 2a: Mutating Admission Webhook

Deploy a simple mutating webhook that injects mesa library mounts into virt-launcher pods.

**How it works:**
1. Webhook watches for pods with label `kubevirt.io=virt-launcher`
2. Injects hostPath volume mounts for mesa libraries
3. virt-launcher gets OpenGL support transparently

**Implementation sketch:**
```yaml
# Webhook injects this into virt-launcher pod spec
volumeMounts:
- name: mesa-egl
  mountPath: /usr/lib64/libEGL.so.1
  readOnly: true
- name: mesa-dri
  mountPath: /usr/lib64/dri
  readOnly: true
volumes:
- name: mesa-egl
  hostPath:
    path: /usr/lib64/libEGL.so.1
    type: File
- name: mesa-dri
  hostPath:
    path: /usr/lib64/dri
    type: Directory
```

**Pros:**
- Simple to implement (a few hundred lines of Go)
- Transparent to KubeVirt - no KubeVirt code changes needed
- Easy to enable/disable for testing

**Cons:**
- Need to deploy and manage a webhook
- TLS certificate management for webhook

---

##### Option 2b: Custom Device Plugin

Create a custom Kubernetes device plugin that advertises a fake resource and mounts mesa libraries when allocated.

**How it works:**
1. Device plugin advertises resource `fakevgpu.io/opengl-libs` (or similar)
2. When a pod requests this resource, the device plugin mounts mesa libraries
3. Test VMI would request this resource to get OpenGL support

**Implementation:**
```go
// Device plugin advertises:
// - Resource: fakevgpu.io/opengl-libs
// - Quantity: 100 (or unlimited)
// 
// When allocated, mounts:
// - /usr/lib64/libEGL.so.1
// - /usr/lib64/libGL.so.1  
// - /usr/lib64/dri/
```

**Pros:**
- Kubernetes-native resource allocation
- Uses existing device plugin framework
- Could be combined with fake vGPU device plugin in the future

**Cons:**
- More complex to implement (gRPC device plugin interface)
- Need to register with kubelet
- More moving parts than webhook approach

---

#### Option 3: Document as known limitation

Accept that display tests won't pass with fake vGPU and skip them.

**Pros:**
- No code changes needed

**Cons:**
- Reduces test coverage

### Current Resolution

**Option 1 is applied and verified** - Tests using fake vGPU should explicitly disable display in the VMI spec, and display-related assertions should be skipped.

#### Verification

Tested by creating a VMI with `virtualGPUOptions.display.enabled: false`:

```yaml
gpus:
- deviceName: nvidia.com/GRID_T4-1B
  name: gpu1
  virtualGPUOptions:
    display:
      enabled: false
```

Result: VM starts successfully. QEMU command line shows `"display":"off"` and the vGPU device is passed through correctly.

**Note**: There are harmless warnings in the logs:
- `"Could not enable error recovery for the device"` - Expected for fake vGPU
- `"Failed to mmap BAR 0. Performance may be slow"` - BAR emulation limitation, but VM still works

#### Why not Option 2 (Add OpenGL to virt-launcher)?

We decided against Option 2 because:
- It would require upstream changes to the virt-launcher container image
- Adding mesa/OpenGL packages just for fake vGPU testing is not production-relevant
- If needed in the future, it could be wrapped behind a feature gate, but currently not justified

#### Implementation

The test `mdev_configuration_allocation_test.go` has been updated to:

1. **Detect fake vGPU**: Added `isFakeVGPU()` helper that checks if `/sys/class/mdev_bus/nvidia` exists (fake vGPU uses this path, real GPUs use PCI addresses like `/sys/class/mdev_bus/0000:XX:XX.X/`)

2. **Disable display for fake vGPU**: When fake vGPU is detected, the test configures the VMI with:
   ```go
   vGPUs[0].VirtualGPUOptions = &v1.VGPUOptions{
       Display: &v1.VGPUDisplayOptions{
           Enabled: pointer.Bool(false),
       },
   }
   ```

3. **Skip display assertions**: The display/ramfb assertions are only run for real vGPU. For fake vGPU, it verifies `display=off` instead

---

## Note: Manual VM Testing with Fake vGPU

When manually creating VMs with fake vGPU (outside of the automated tests), you must first configure the KubeVirt CR to recognize the mdev devices:

```bash
kubectl patch kubevirt kubevirt -n kubevirt --type=merge -p '
{
  "spec": {
    "configuration": {
      "mediatedDevicesConfiguration": {
        "mediatedDeviceTypes": ["nvidia-222"],
        "nodeMediatedDeviceTypes": null
      },
      "permittedHostDevices": {
        "mediatedDevices": [
          {
            "mdevNameSelector": "GRID T4-1B",
            "resourceName": "nvidia.com/GRID_T4-1B"
          }
        ]
      }
    }
  }
}'
```

This configuration:
1. Tells virt-handler to create mdev instances of type `nvidia-222`
2. Permits the `GRID T4-1B` mdev to be used by VMs with resource name `nvidia.com/GRID_T4-1B`

**Important**: Without this configuration, virt-handler will remove any manually created mdev instances during its reconciliation loop.

---

## Issue #2: PCI Hot-Plug Test Skipped

### Status: **Skipped (by design)**

### Problem

The test `"should create mdevs on devices that appear after CR configuration"` is skipped when running with fake vGPU.

### What the Test Does

1. **Unbinds** a real PCI GPU from its driver (device disappears from `/sys/class/mdev_bus/`)
2. **Configures KubeVirt CR** with mdev types while device is absent
3. **Rebinds** the device to its driver (device reappears)
4. **Verifies** virt-handler automatically creates mdev instances when device reappears

This tests KubeVirt's **virt-handler hot-plug detection** - its ability to detect newly appeared devices and create configured mdev instances.

### Root Cause

Fake vGPU cannot emulate PCI hot-plug because:

1. **No PCI address**: Fake vGPU creates `/sys/class/mdev_bus/nvidia/`, not a PCI device path like `/sys/class/mdev_bus/0000:XX:XX.X/`
2. **No real driver**: Cannot use `echo $ID > /sys/bus/pci/drivers/.../unbind` mechanism
3. **Different code path**: Module unload/reload would test different kernel code than real PCI hot-plug

### Possible Solutions

#### Option 1: Skip the test (Current)

Accept that this specific hot-plug scenario cannot be tested with fake vGPU.

**Pros:**
- Simple, no additional complexity
- Core virt-handler mdev creation is tested by other tests

**Cons:**
- Hot-plug code path not tested with fake vGPU

#### Option 2: Add module unload/reload emulation

Modify the test to unload and reload the fake-nvidia-vgpu kernel module to simulate device disappear/reappear.

**Pros:**
- Would test virt-handler's device detection

**Cons:**
- Tests a different code path than real PCI hot-plug
- Requires root privileges on the host (not inside container)
- Complex to implement safely
- May interfere with other tests running in parallel

### Current Resolution

**Option 1 applied** - Test is skipped when `isFakeVGPU()` returns true.

```go
if isFakeVGPU() {
    Skip("Test requires real PCI GPU hardware - skipping for fake vGPU")
}
```

The primary virt-handler mdev creation functionality is already covered by:
- `"Should successfully passthrough a mediated device"`
- `"Should override default mdev configuration on a specific node"`

### Potential Future Work

If hot-plug testing becomes a priority, here's a potential implementation plan:

#### Phase 1: Add sysfs control interface to fake-nvidia-vgpu module

Add a sysfs file to control device visibility without full module unload:

```c
// In fake-nvidia-vgpu.c, add:
// /sys/class/nvidia/fake_vgpu_control - write "hide"/"show" to simulate device disappear/reappear

static ssize_t control_store(struct device *dev, struct device_attribute *attr,
                             const char *buf, size_t count) {
    if (strncmp(buf, "hide", 4) == 0) {
        // Unregister mdev parent (device disappears from mdev_bus)
        mdev_unregister_parent(&fake_vgpu_dev.parent);
    } else if (strncmp(buf, "show", 4) == 0) {
        // Re-register mdev parent (device reappears)
        mdev_register_parent(&fake_vgpu_dev.parent, ...);
    }
    return count;
}
```

#### Phase 2: Add helper script for hot-plug simulation

```bash
# In setup-fake-vgpu-host.sh, add:
hotplug_hide() {
    echo "hide" > /sys/class/nvidia/fake_vgpu_control
}

hotplug_show() {
    echo "show" > /sys/class/nvidia/fake_vgpu_control
}
```

#### Phase 3: Modify test for fake vGPU path

```go
if isFakeVGPU() {
    // Use module control interface instead of PCI unbind/bind
    By("hiding fake vGPU device")
    runBashCmdRw("echo hide > /sys/class/nvidia/fake_vgpu_control")
    
    // ... configure KubeVirt CR ...
    
    By("showing fake vGPU device")
    runBashCmdRw("echo show > /sys/class/nvidia/fake_vgpu_control")
    
    // ... verify mdev creation ...
}
```

#### Estimated Effort

| Phase | Complexity | Description |
|-------|------------|-------------|
| Phase 1 | Medium | Kernel module changes, needs testing |
| Phase 2 | Low | Simple shell script additions |
| Phase 3 | Low | Test code branching |

**Total: ~2-3 days of work** if this becomes a priority.

---

## Issue #3: (Template for future issues)

### Status: Open/In Progress/Resolved

### Problem

Description of the issue.

### Root Cause

Analysis of why this happens.

### Possible Solutions

Options considered.

### Current Resolution

What was done.
