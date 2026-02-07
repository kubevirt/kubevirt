# Mesa Injector Webhook

A mutating admission webhook that injects OpenGL/mesa libraries into virt-launcher pods, enabling vGPU display support for fake vGPU testing.

## Problem

When using fake vGPU with display enabled, QEMU requires OpenGL/EGL libraries that are not present in the virt-launcher container:

```
vfio-display-dmabuf: opengl not available
```

## Solution

This webhook automatically injects mesa/OpenGL libraries from the host into virt-launcher pods via hostPath mounts, similar to how NVIDIA's GPU Operator mounts NVIDIA libraries.

## Prerequisites

Mesa packages must be installed on the Kind node. Add this to the node setup:

```bash
dnf install -y mesa-dri-drivers mesa-libEGL mesa-libGL libglvnd-egl libglvnd-gles
```

## Usage

### Deploy

```bash
./deploy.sh deploy
```

This will:
1. Build the webhook container image
2. Load it into the Kind cluster
3. Generate TLS certificates
4. Deploy the webhook

### Check Status

```bash
./deploy.sh status
```

### Remove

```bash
./deploy.sh undeploy
```

## What Gets Injected

The webhook injects these host paths into virt-launcher pods:

| Host Path | Container Path | Type |
|-----------|----------------|------|
| `/usr/lib64/libGL.so.1` | `/usr/lib64/libGL.so.1` | File |
| `/usr/lib64/libEGL.so.1` | `/usr/lib64/libEGL.so.1` | File |
| `/usr/lib64/libGLESv2.so.2` | `/usr/lib64/libGLESv2.so.2` | File |
| `/usr/lib64/libGLX.so.0` | `/usr/lib64/libGLX.so.0` | File |
| `/usr/lib64/dri/` | `/usr/lib64/dri/` | Directory |
| `/usr/share/glvnd/egl_vendor.d/` | `/usr/share/glvnd/egl_vendor.d/` | Directory |

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                      │
└──────────────────────────────────────────────────────────────┘
                              │
                              │ Pod CREATE request
                              ▼
┌──────────────────────────────────────────────────────────────┐
│              MutatingWebhookConfiguration                     │
│  - Matches pods with label: kubevirt.io=virt-launcher        │
│  - Only in namespace: kubevirt                               │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    Mesa Injector Webhook                      │
│  - Receives AdmissionReview                                  │
│  - Returns JSON patch to add hostPath volumes                │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    virt-launcher Pod                          │
│  + mesa-libgl volume mount                                   │
│  + mesa-libegl volume mount                                  │
│  + mesa-dri volume mount                                     │
│  + ...                                                       │
└──────────────────────────────────────────────────────────────┘
```

## Verification

After deploying, create a VM with vGPU display enabled and check the pod spec:

```bash
kubectl -n kubevirt get pod -l kubevirt.io=virt-launcher -o jsonpath='{.items[0].spec.volumes}' | jq .
```

You should see the mesa-* volumes added by the webhook.

## License

Apache-2.0
