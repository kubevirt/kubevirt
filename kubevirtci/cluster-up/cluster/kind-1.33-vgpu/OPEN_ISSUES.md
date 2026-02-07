# Fake vGPU Provider - Open Issues

This document tracks known issues and limitations with the fake vGPU provider.

---

## Issue #1: vGPU Display Support Requires OpenGL

### Status: **Workaround Applied** (Full solution available via mesa-injector webhook)

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

##### Option 2a: Mutating Admission Webhook (IMPLEMENTED)

A mutating admission webhook (`mesa-injector`) is now available that injects mesa library mounts into virt-launcher pods.

**Location:** `mesa-injector/` directory

**How it works:**
1. Webhook watches for pods with label `kubevirt.io=virt-launcher`
2. Injects hostPath volume mounts for mesa libraries
3. virt-launcher gets OpenGL support transparently

**Usage:**
```bash
# 1. Bring up cluster with mesa support
VGPU_DISPLAY=true make cluster-up KUBEVIRT_PROVIDER=kind-1.33-vgpu

# 2. Deploy the webhook
./mesa-injector/deploy.sh deploy

# 3. Now VMs can use vGPU display without disabling it
```

**What gets injected:**
- `/usr/lib64/libGL.so.1`
- `/usr/lib64/libEGL.so.1`
- `/usr/lib64/libGLESv2.so.2`
- `/usr/lib64/libGLX.so.0`
- `/usr/lib64/dri/`
- `/usr/share/glvnd/egl_vendor.d/`

**Pros:**
- Transparent to KubeVirt - no KubeVirt code changes needed
- Easy to enable/disable for testing
- Automatic TLS certificate generation

**Cons:**
- Need to deploy and manage a webhook
- Adds complexity to the test setup

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

