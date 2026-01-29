# VEP: Additional Virt-Handlers

## Release Signoff Checklist

- [x] Enhancement issue opened in kubevirt/enhancements repo
- [x] Design document approved
- [x] Feature gate created (`AdditionalVirtHandlers`)
- [x] Test plan documented
- [x] User documentation added

## Summary

This VEP proposes adding support for deploying multiple virt-handler DaemonSets to serve heterogeneous node pools with different virt-handler and virt-launcher container images.

## Motivation

Many organizations operate heterogeneous Kubernetes clusters with specialized node pools for different workloads. Examples include:

- **GPU nodes**: Require virt-launcher images with GPU drivers and libraries
- **FPGA nodes**: Need specialized images with FPGA support libraries
- **Secure enclaves**: Run hardened images with additional security configurations
- **Multi-architecture clusters**: ARM vs x86 nodes requiring architecture-specific images

Currently, KubeVirt deploys a single virt-handler DaemonSet that runs the same images across all nodes. This forces operators to either:

1. Run separate KubeVirt installations for each node pool (operational overhead)
2. Build a single image containing all specialized components (image bloat)
3. Use external webhooks to mutate VMI pods (fragile, external dependency)

### Goals

- Enable operators to deploy additional virt-handler DaemonSets targeting specific node pools
- Allow custom virt-handler and virt-launcher images per node pool
- Automatically select the appropriate virt-launcher image for VMIs based on their node selector
- Maintain backward compatibility with existing single-handler deployments

### Non-Goals

- Runtime image switching for running VMIs
- Automatic detection of node capabilities
- Cross-pool live migration with image transformation

## Proposal

### API Changes

Add a new field `additionalVirtHandlers` to `KubeVirtSpec`:

```go
// KubeVirtSpec (in types.go)
type KubeVirtSpec struct {
    // ... existing fields ...

    // additionalVirtHandlers configures additional virt-handler DaemonSets
    // targeting specific nodes with custom images.
    // +optional
    AdditionalVirtHandlers []AdditionalVirtHandlerConfig `json:"additionalVirtHandlers,omitempty"`
}
```

Add a new type `AdditionalVirtHandlerConfig` to `componentconfig.go`:

```go
// AdditionalVirtHandlerConfig defines configuration for an additional virt-handler DaemonSet
// that targets specific nodes with custom images.
type AdditionalVirtHandlerConfig struct {
    // name is a unique identifier appended to "virt-handler" to form the DaemonSet name.
    // For example, "gpu" results in a DaemonSet named "virt-handler-gpu".
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
    // +kubebuilder:validation:MaxLength=48
    Name string `json:"name"`

    // virtHandlerImage overrides the virt-handler container image for this DaemonSet.
    // If not specified, the default virt-handler image is used.
    // +optional
    VirtHandlerImage string `json:"virtHandlerImage,omitempty"`

    // virtLauncherImage overrides the virt-launcher image used by this virt-handler
    // for the init container and passed to virt-launcher pods on nodes served by this handler.
    // If not specified, the default virt-launcher image is used.
    // +optional
    VirtLauncherImage string `json:"virtLauncherImage,omitempty"`

    // nodeSelector specifies labels that must match a node's labels for this DaemonSet's pods
    // to be scheduled on that node. This is also used to match VMIs to determine which
    // virt-launcher image to use.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinProperties=1
    NodeSelector map[string]string `json:"nodeSelector"`
}
```

### User Stories

#### Story 1: GPU Node Pool

As a cluster administrator, I want to run virt-launcher pods with pre-installed NVIDIA drivers on GPU nodes so that VMs can access GPU hardware without additional setup.

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
```

#### Story 2: Multi-Pool Configuration

As a cluster administrator, I want to run different virt-launcher images on GPU nodes and FPGA nodes in the same cluster.

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

#### Story 3: VMI Targeting a Pool

As a VM user, I want my VMI to automatically use the GPU-optimized virt-launcher when I target GPU nodes.

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

## Design Details

### VMI to Handler Matching

VMIs are matched to additional handlers based on their `spec.nodeSelector`:

1. A VMI matches an additional handler if the VMI's nodeSelector contains **all** key-value pairs from the handler's nodeSelector (superset matching)
2. The first matching handler in the list is used
3. If no handler matches, the default virt-launcher image is used

**Matching examples:**

| VMI nodeSelector | Handler nodeSelector | Match? |
|------------------|---------------------|--------|
| `{gpu: "true"}` | `{gpu: "true"}` | Yes |
| `{gpu: "true", zone: "us-west"}` | `{gpu: "true"}` | Yes |
| `{gpu: "true"}` | `{gpu: "true", zone: "us-west"}` | No |
| `{cpu: "intel"}` | `{gpu: "true"}` | No |
| `{}` (empty) | `{gpu: "true"}` | No |

**Rationale for nodeSelector matching over node detection:**
- Allows image selection before pod scheduling
- Provides predictable behavior - users know which image will be used by looking at the VMI spec
- Avoids race conditions during scheduling

### Component Changes

#### virt-operator

- Creates additional DaemonSets (`virt-handler-<name>`) for each entry in `additionalVirtHandlers`
- Applies the handler's `nodeSelector` as pod scheduling constraints
- Adds `kubevirt.io/handler-pool: <name>` label to additional DaemonSets
- Injects `RequiredDuringSchedulingIgnoredDuringExecution` node affinity on the primary virt-handler with `NotIn` expressions to avoid nodes matching any additional handler's nodeSelector
- Deletes additional DaemonSets when removed from configuration

#### virt-controller (TemplateService)

- Uses `handlermatcher.MatchVMIToAdditionalHandler()` to find matching handler
- Uses `handlermatcher.GetLauncherImageForVMI()` to select virt-launcher image
- Adds `kubevirt.io/handler-pool` annotation to virt-launcher pods identifying the handler pool

#### virt-controller (workload-updater)

- Uses `GetLauncherImageForVMI()` to determine expected launcher image per VMI
- Correctly identifies outdated VMIs when handler configurations change

### Test Plan

#### Unit Tests

`pkg/virt-controller/services/handlermatcher_test.go`:
- VMI with no nodeSelector returns nil
- Exact nodeSelector match returns handler
- VMI nodeSelector superset returns handler
- VMI nodeSelector subset returns nil
- Value mismatch returns nil
- Multiple handlers returns first match
- Launcher image selection with and without custom images

#### Functional Tests

`tests/operator/operator.go` (Context: "with AdditionalVirtHandlers feature gate"):
- Creates additional virt-handler DaemonSet when feature gate is enabled
- Verifies DaemonSet has correct handler-pool label
- Verifies DaemonSet has configured nodeSelector
- Additional virt-handler pod runs on labeled nodes
- Deletes additional DaemonSet when removed from configuration
- Uses custom images when specified
- Matches VMI to additional handler and sets handler pool annotation
- Configures anti-affinity on primary virt-handler to avoid additional handler nodes

### Graduation Criteria

#### Alpha (v1.8.0)

- Feature gate `AdditionalVirtHandlers` (disabled by default)
- API type `AdditionalVirtHandlerConfig` with all fields
- DaemonSet creation and deletion
- VMI matching and image selection
- Anti-affinity on primary handler
- Unit and functional tests
- User documentation

#### Beta (target: v1.10.0)

- Feature gate enabled by default
- Validation webhook for configuration conflicts
- Metrics for handler pool utilization
- E2E tests for multi-pool scenarios

#### GA

- Feature gate removed (always enabled)
- Stable API with no breaking changes for 2+ releases

### Upgrade / Downgrade Strategy

**Upgrade:**
- Existing deployments continue to work with single virt-handler
- Additional handlers can be added incrementally
- Running VMIs are not affected until restart

**Downgrade:**
- Additional DaemonSets are deleted when feature gate is disabled
- Running VMIs served by additional handlers continue running
- New VMIs use default images

### Version Skew Strategy

All virt-handler images must be from the same KubeVirt version. Using mismatched versions is unsupported and may cause undefined behavior.

## Alternatives Considered

### Namespace-based Separation

Run separate KubeVirt installations in different namespaces for each node pool.

**Rejected because:**
- Increases operational complexity
- Prevents resource sharing between pools
- Complicates upgrades

### Per-VMI Image Override Annotations

Allow users to specify virt-launcher image via VMI annotations.

**Rejected because:**
- Security concerns - arbitrary image injection
- No virt-handler customization
- Harder to audit/govern

### Webhook-based Image Mutation

Use external mutating webhook to inject images based on node targeting.

**Rejected because:**
- External dependency
- Doesn't address virt-handler customization
- Fragile - webhook failures block VMI creation

## Implementation History

- 2025-01: VEP proposed
- 2025-01: Implementation PR opened
- v1.8.0: Alpha release target
