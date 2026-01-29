# Additional Virt-Handlers

## Overview

KubeVirt supports deploying multiple virt-handler DaemonSets to serve
heterogeneous node pools. This allows cluster administrators to run
specialized virt-handler and virt-launcher images on specific nodes,
such as GPU nodes, FPGA nodes, or nodes in secure enclaves.

## Use Cases

- **GPU Nodes**: Deploy virt-launcher images with pre-installed GPU drivers
- **FPGA Nodes**: Use specialized images with FPGA support libraries
- **Secure Enclaves**: Run hardened images on high-security nodes
- **Multi-Architecture**: Deploy architecture-specific images to ARM vs x86 nodes

## Prerequisites

This feature requires enabling the `AdditionalVirtHandlers` feature gate:

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    developerConfiguration:
      featureGates:
        - AdditionalVirtHandlers
```

## Configuration

Additional virt-handlers are configured in the KubeVirt CR under
`spec.additionalVirtHandlers`. Each entry creates a separate virt-handler
DaemonSet targeting specific nodes.

### Basic Example

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    developerConfiguration:
      featureGates:
        - AdditionalVirtHandlers
  additionalVirtHandlers:
    - name: gpu-pool
      virtHandlerImage: registry.example.com/kubevirt/virt-handler:v1.0.0-gpu
      virtLauncherImage: registry.example.com/kubevirt/virt-launcher:v1.0.0-gpu
      nodeSelector:
        node.kubernetes.io/gpu: "true"
```

### Configuration Fields

| Field | Description | Required |
|-------|-------------|----------|
| `name` | Unique identifier for the handler pool. Used as suffix for DaemonSet name (`virt-handler-<name>`). | Yes |
| `virtHandlerImage` | Container image for virt-handler. Defaults to the standard virt-handler image if not specified. | No |
| `virtLauncherImage` | Container image for virt-launcher pods created on matching nodes. Defaults to the standard virt-launcher image if not specified. | No |
| `nodeSelector` | Labels that must match a node's labels for this DaemonSet's pods to be scheduled on that node. Also used to match VMIs to determine which virt-launcher image to use. | Yes |

### Multiple Pools Example

```yaml
apiVersion: kubevirt.io/v1
kind: KubeVirt
metadata:
  name: kubevirt
  namespace: kubevirt
spec:
  configuration:
    developerConfiguration:
      featureGates:
        - AdditionalVirtHandlers
  additionalVirtHandlers:
    - name: gpu-pool
      virtLauncherImage: registry.example.com/kubevirt/virt-launcher:v1.0.0-gpu
      nodeSelector:
        accelerator: nvidia-gpu
    - name: fpga-pool
      virtLauncherImage: registry.example.com/kubevirt/virt-launcher:v1.0.0-fpga
      nodeSelector:
        accelerator: intel-fpga
```

## How VMIs Use Custom Images

When a VirtualMachineInstance (VMI) is created, KubeVirt determines which
virt-launcher image to use based on the VMI's node selector:

1. If the VMI's `spec.nodeSelector` contains all key-value pairs from an
   additional handler's `nodeSelector`, that handler's
   `virtLauncherImage` is used.

2. If no additional handler matches, the default virt-launcher image is used.

### Example: VMI Targeting GPU Pool

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: gpu-vm
spec:
  nodeSelector:
    accelerator: nvidia-gpu
  domain:
    resources:
      requests:
        memory: 4Gi
    devices:
      gpus:
        - name: gpu1
          deviceName: nvidia.com/GPU
      disks:
        - name: containerdisk
          disk:
            bus: virtio
  volumes:
    - name: containerdisk
      containerDisk:
        image: registry.example.com/my-gpu-vm:latest
```

This VMI will use the `virt-launcher:v1.0.0-gpu` image because its node
selector (`accelerator: nvidia-gpu`) matches the gpu-pool handler's placement.

### Example: VMI Using Default Image

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: standard-vm
spec:
  domain:
    resources:
      requests:
        memory: 1Gi
    devices:
      disks:
        - name: containerdisk
          disk:
            bus: virtio
  volumes:
    - name: containerdisk
      containerDisk:
        image: registry.example.com/my-vm:latest
```

This VMI has no node selector, so it will use the default virt-launcher image.

## Matching Logic

A VMI matches an additional handler when the VMI's node selector is a
**superset** of the handler's node selector. This means:

- All key-value pairs in the handler's `nodeSelector` must
  exist in the VMI's `spec.nodeSelector`
- The VMI may have additional node selectors beyond those in the handler

### Examples

| VMI Node Selector | Handler Node Selector | Match? |
|-------------------|----------------------|--------|
| `{gpu: "true"}` | `{gpu: "true"}` | Yes |
| `{gpu: "true", zone: "us-west"}` | `{gpu: "true"}` | Yes |
| `{gpu: "true"}` | `{gpu: "true", zone: "us-west"}` | No |
| `{cpu: "intel"}` | `{gpu: "true"}` | No |
| `{}` (empty) | `{gpu: "true"}` | No |

## Architecture

When additional handlers are configured:

1. **virt-operator** creates a DaemonSet for each additional handler
   (`virt-handler-<name>`) with the specified placement and images.

2. The **primary virt-handler** DaemonSet is configured with anti-affinity
   to avoid running on nodes targeted by additional handlers.

3. **virt-controller** selects the appropriate virt-launcher image when
   creating VMI pods based on node selector matching.

4. **workload-updater** correctly identifies outdated VMIs by comparing
   against the expected per-pool launcher image.

## Limitations

- All virt-handler images must be from the same KubeVirt version
- Live migration between pools with different virt-launcher images preserves
  the source image (the target pod uses the same image as the source)
- Changing handler configurations does not affect running VMIs
- The feature gate must be enabled before configuring additional handlers

## Troubleshooting

### Additional DaemonSet Not Created

1. Verify the feature gate is enabled:
   ```bash
   kubectl get kubevirt kubevirt -n kubevirt -o jsonpath='{.spec.configuration.developerConfiguration.featureGates}'
   ```

2. Check virt-operator logs for errors:
   ```bash
   kubectl logs -n kubevirt -l kubevirt.io=virt-operator
   ```

### VMI Using Wrong Launcher Image

1. Verify the VMI's node selector matches the handler's placement:
   ```bash
   kubectl get vmi <name> -o jsonpath='{.spec.nodeSelector}'
   ```

2. Check the handler configuration:
   ```bash
   kubectl get kubevirt kubevirt -n kubevirt -o jsonpath='{.spec.additionalVirtHandlers}'
   ```

3. Inspect the virt-launcher pod's image:
   ```bash
   kubectl get pod -l kubevirt.io/domain=<vmi-name> -o jsonpath='{.items[0].spec.containers[?(@.name=="compute")].image}'
   ```
