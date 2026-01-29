# VEP: Additional Virt-Handlers for Heterogeneous Node Pools

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue created, which links to VEP dir in [kubevirt/enhancements] (not the initial VEP PR)
- [ ] (R) Target version is explicitly mentioned and approved
- [ ] (R) Graduation criteria filled

## Overview

This enhancement introduces support for deploying multiple virt-handler DaemonSets through virt-operator, each targeting different node pools with potentially different virt-handler and virt-launcher images. This enables heterogeneous cluster configurations where specialized nodes (e.g., GPU nodes, FPGA nodes) can run customized KubeVirt components.

## Motivation

In heterogeneous clusters, different node pools may require specialized virt-handler and virt-launcher images. For example:

- **GPU nodes**: May need virt-launcher images with GPU drivers and libraries pre-installed
- **FPGA nodes**: May require specialized virt-handler/launcher with FPGA support
- **Secure enclaves**: May need hardened images with specific security configurations
- **Different architectures**: ARM vs x86 nodes may need different images

Currently, KubeVirt deploys a single virt-handler DaemonSet that runs the same image across all nodes. This limitation prevents operators from customizing KubeVirt components for specific node pools without maintaining separate cluster installations.

## Goals

1. Allow operators to configure additional virt-handler DaemonSets targeting specific node pools via node selectors
2. Support custom virt-handler images per node pool
3. Support custom virt-launcher images per node pool, used when creating VMI pods on matching nodes
4. Ensure VMIs scheduled on nodes served by additional handlers use the appropriate virt-launcher image
5. Maintain backward compatibility - existing single virt-handler deployments continue to work unchanged
6. Protect the feature behind a feature gate during alpha/beta phases

## Non Goals

1. Automatic image selection based on node capabilities (requires explicit configuration)
2. Support for different KubeVirt versions across node pools (all handlers must be from the same KubeVirt release)
3. Live migration between pools with different virt-launcher images (VMIs retain their original launcher image)
4. Automatic node labeling or pool detection

## Definition of Users

- **Cluster Administrators**: Configure additional virt-handlers in the KubeVirt CR to support heterogeneous node pools
- **VM Owners**: Create VMIs with node selectors to target specific node pools

## User Stories

### Story 1: GPU Node Pool

As a cluster administrator, I want to deploy a specialized virt-launcher image on GPU nodes that includes NVIDIA drivers, so that VMs running on those nodes can access GPU resources without additional configuration.

### Story 2: Secure Node Pool

As a security administrator, I want to run hardened virt-handler and virt-launcher images on nodes in a secure enclave, while using standard images on regular nodes.

### Story 3: Multi-Architecture Cluster

As a cluster administrator managing a mixed ARM/x86 cluster, I want to deploy architecture-specific virt-handler images to each node type.

## Repos

- kubevirt/kubevirt (primary)
- kubevirt/api (API types)

## Design

### API Changes

A new `AdditionalVirtHandlerConfig` type is added to the KubeVirt API:

```go
type AdditionalVirtHandlerConfig struct {
    // Name is a unique identifier for this additional handler configuration.
    // It will be used as a suffix for the DaemonSet name (virt-handler-<name>).
    Name string `json:"name"`

    // VirtHandlerImage is the container image to use for virt-handler.
    // If not specified, the default virt-handler image is used.
    VirtHandlerImage string `json:"virtHandlerImage,omitempty"`

    // VirtLauncherImage is the container image to use for virt-launcher
    // when creating VMI pods on nodes served by this handler.
    // If not specified, the default virt-launcher image is used.
    VirtLauncherImage string `json:"virtLauncherImage,omitempty"`

    // NodeSelector specifies labels that must match a node's labels for
    // this DaemonSet's pods to be scheduled on that node. This is also
    // used to match VMIs to determine which virt-launcher image to use.
    NodeSelector map[string]string `json:"nodeSelector"`
}
```

The `KubeVirtSpec` is extended with:

```go
type KubeVirtSpec struct {
    // ... existing fields ...

    // AdditionalVirtHandlers defines additional virt-handler DaemonSets
    // for heterogeneous node pools.
    AdditionalVirtHandlers []AdditionalVirtHandlerConfig `json:"additionalVirtHandlers,omitempty"`
}
```

### Feature Gate

The feature is protected by the `AdditionalVirtHandlers` feature gate, initially in Alpha state.

### Virt-Operator Changes

When the feature gate is enabled and additional handlers are configured:

1. **DaemonSet Generation**: For each entry in `AdditionalVirtHandlers`, virt-operator creates a new DaemonSet named `virt-handler-<name>` with:
   - The specified `VirtHandlerImage` (or default if not specified)
   - The specified `VirtLauncherImage` passed via `--launcher-image` flag
   - Node placement from the configuration
   - A unique pool label (`kubevirt.io/handler-pool: <name>`) to distinguish from the primary handler

2. **Anti-Affinity**: The primary virt-handler DaemonSet is configured with anti-affinity to avoid scheduling on nodes targeted by additional handlers, preventing duplicate handlers on the same node.

3. **Reconciliation**: The operator reconciles additional DaemonSets like any other managed resource, handling creates, updates, and deletes.

### Virt-Controller Changes

When creating virt-launcher pods for VMIs:

1. **Image Selection**: The TemplateService determines the appropriate virt-launcher image by matching the VMI's node selector against additional handler configurations:
   - If the VMI's node selector contains all key-value pairs from an additional handler's node placement, that handler's `VirtLauncherImage` is used
   - Otherwise, the default virt-launcher image is used

2. **Workload Updater**: The WorkloadUpdateController's outdated VMI detection is updated to compare against the expected per-pool launcher image rather than a single global image.

### Matching Logic

A VMI matches an additional handler if the VMI's `spec.nodeSelector` contains all key-value pairs from the handler's `nodeSelector`. This is a superset match - the VMI may have additional selectors beyond those in the handler configuration.

## API Examples

### Basic Configuration

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

### VMI Targeting GPU Pool

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: gpu-vm
spec:
  nodeSelector:
    node.kubernetes.io/gpu: "true"
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
        image: registry.example.com/my-gpu-vm:latest
```

This VMI will use the `virt-launcher:v1.0.0-gpu` image because its node selector matches the gpu-pool handler's placement.

## Alternatives

### Alternative 1: Namespace-Based Separation

Deploy separate KubeVirt installations in different namespaces, each targeting specific node pools.

**Rejected because**: Increases operational complexity, prevents resource sharing, and complicates cluster-wide VM management.

### Alternative 2: Image Override Annotations

Allow per-VMI image overrides via annotations.

**Rejected because**: Security concern - allows arbitrary image injection; doesn't provide centralized management of approved images.

### Alternative 3: Webhook-Based Image Mutation

Use a mutating webhook to modify virt-launcher images based on node selection.

**Rejected because**: Adds external dependency; harder to manage and debug; doesn't address virt-handler customization.

## Scalability

- **DaemonSet Overhead**: Each additional handler configuration creates one DaemonSet. The number of pods scales with nodes matching the selector, not with total cluster size.
- **Memory/CPU**: Additional virt-handler pods consume resources only on targeted nodes.
- **API Objects**: Minimal increase - one DaemonSet per additional handler configuration.
- **Matching Logic**: O(n*m) where n is number of VMIs created and m is number of additional handlers. Both are expected to be small numbers.

## Update/Rollback Compatibility

### Upgrades

- Existing clusters without additional handlers configured continue to work unchanged
- The feature gate must be enabled before configuring additional handlers
- Adding additional handlers is non-disruptive to existing VMIs

### Rollbacks

- Disabling the feature gate removes additional DaemonSets
- Existing VMIs continue running with their original launcher images
- New VMIs will use the default launcher image

### VMI Considerations

- VMIs retain their original virt-launcher image throughout their lifecycle
- Migrations preserve the source pod's launcher image
- Changing handler configurations does not affect running VMIs

## Functional Testing Approach

### Unit Tests

- Handler matching logic (node selector superset matching)
- Image selection in TemplateService
- Workload updater outdated detection with per-pool images

### Functional Tests

1. **DaemonSet Creation**: Verify additional DaemonSets are created when feature gate is enabled and handlers are configured
2. **DaemonSet Deletion**: Verify DaemonSets are removed when handlers are removed from configuration
3. **Feature Gate Protection**: Verify handlers are not created when feature gate is disabled
4. **Custom Images**: Verify DaemonSets use specified custom images
5. **VMI Image Selection**: Verify VMIs with matching selectors use custom launcher images
6. **VMI Default Image**: Verify VMIs without matching selectors use default launcher images

## Implementation History

- Initial implementation of AdditionalVirtHandlerConfig API type
- Added AdditionalVirtHandlers feature gate (Alpha)
- Implemented virt-operator support for additional DaemonSets with placement and anti-affinity
- Added virt-controller support for per-pool virt-launcher image selection
- Updated workload-updater for per-pool image awareness
- Added unit and functional tests

## Graduation Requirements

### Alpha

- [x] Feature gate (`AdditionalVirtHandlers`) guards all code changes
- [x] API types for additional handler configuration
- [x] virt-operator creates/manages additional DaemonSets
- [x] virt-controller selects per-pool virt-launcher images
- [x] Workload updater handles per-pool images correctly
- [x] Unit tests for matching logic and image selection
- [x] Functional tests for DaemonSet lifecycle and VMI image selection
- [ ] Documentation for feature configuration

### Beta

- [ ] User feedback incorporated from alpha usage
- [ ] E2E tests with actual heterogeneous node pools
- [ ] Support for tolerations in additional handler placement
- [ ] Metrics for additional handler health and VMI distribution

### GA

- [ ] Feature enabled by default (no feature gate required)
- [ ] Proven stability in production environments
- [ ] Complete documentation and examples
- [ ] Migration guide for users enabling the feature
