# vGPU Display Support Issue

## Problem Statement

When using fake vGPU with `display=on`, QEMU requires OpenGL/EGL for DMABUF framebuffer rendering. This fails in container-based environments (Kind) with:

```
vfio-display-dmabuf: opengl not available
```

## Root Cause

QEMU's vfio-display-dmabuf requires:
1. OpenGL/EGL libraries (mesa)
2. A DRI render node (`/dev/dri/renderD*`)
3. DMA-BUF kernel support

Container-based environments (Kind) cannot safely provide a render node because containers share the host kernel directly.

## Attempted Solutions on Kind (Container-based)

| Approach | Outcome |
|----------|---------|
| Mesa library injection via webhook | Libraries mounted successfully |
| Resolved all library dependencies | `ldd` shows no missing libs |
| Host GPU passthrough (`/dev/dri`) | **KERNEL PANIC** - Unsafe, crashes host |
| `vgem` kernel module | **SYSTEM HANG** - Unsafe, freezes host |

### Why It Fails

- Containers share the host kernel directly
- No isolation layer for GPU device access
- Host's `/dev/dri` is the actual GPU - accessing it from nested containers destabilizes the system
- No safe way to create an isolated virtual render node in containers

## Current Solution

**Disable display for fake vGPU in tests:**

```bash
# Run without VGPU_DISPLAY - tests auto-disable display for fake vGPU
KUBEVIRT_E2E_FOCUS="MediatedDevices" make functest
```

The `shouldDisableDisplay()` function in `mdev_configuration_allocation_test.go` handles this automatically.

## Alternative: VM-Based Providers

Moving to VM-based providers (kcli, kubevirtci VM providers) would solve this:

### Why VMs Work

```
Host GPU → Hypervisor → virtio-gpu (in VM) → /dev/dri → virt-launcher pod
```

- **virtio-gpu**: Virtual GPU designed for VMs, provides safe `/dev/dri`
- **Isolation**: Hypervisor fully isolates VM from host hardware
- **DMA-BUF support**: virtio-gpu supports DMABUF for framebuffer sharing
- **No host impact**: VM crashes don't affect host

### Implementation for VM Providers

1. Enable `virtio-gpu` device in VM configuration
2. VM gets `/dev/dri/card*` and `/dev/dri/renderD*`
3. Pass `/dev/dri` to pods via hostPath mount
4. Deploy mesa-injector webhook for library injection
5. EGL initializes successfully, QEMU display works

## Other Alternatives (Not Recommended)

| Alternative | Status | Notes |
|-------------|--------|-------|
| Mesa drm-shim | Untested | Userspace DRM interception, complex setup |
| EGL_PLATFORM=surfaceless | Unlikely to work | QEMU needs real DMABUF support |
| udmabuf | Risky | Kernel module, similar risks to vgem |

## Summary

| Environment | Recommendation |
|-------------|----------------|
| **Kind (containers)** | Use display-disable workaround |
| **VM-based providers** | Enable virtio-gpu, use mesa-injector |
| **Real vGPU hardware** | Mesa-injector webhook works |

## Related Files

- `mesa-injector/` - Webhook for injecting mesa libraries (works with real vGPU/VMs)
- `vgpu-node/node.sh` - `node::install_mesa()` function
- `tests/mdev_configuration_allocation_test.go` - `shouldDisableDisplay()` function

## References

- QEMU vfio-display: https://www.qemu.org/docs/master/system/devices/vfio.html
- Mesa EGL: https://docs.mesa3d.org/egl.html
- virtio-gpu: https://www.kraxel.org/blog/2016/09/using-virtio-gpu-with-libvirt-and-spice/
