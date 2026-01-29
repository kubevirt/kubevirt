### What this PR does

#### Before this PR:

KubeVirt deploys a single virt-handler DaemonSet that runs the same virt-handler and virt-launcher images across all nodes. This prevents operators from customizing KubeVirt components for specific node pools (e.g., GPU nodes, FPGA nodes, secure enclaves) without maintaining separate cluster installations.

#### After this PR:

Operators can configure additional virt-handler DaemonSets targeting specific node pools via the KubeVirt CR. Each additional handler can specify:
- A custom `virtHandlerImage` for the virt-handler pods
- A custom `virtLauncherImage` for VMI pods scheduled on matching nodes
- A `nodeSelector` to target specific nodes

VMIs with node selectors matching an additional handler's nodeSelector will automatically use that handler's custom virt-launcher image.

**Example KubeVirt CR with additional handler for GPU nodes:**

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

**Example VMI targeting the GPU pool (will use the custom virt-launcher image):**

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

### References

- VEP tracking issue: https://github.com/kubevirt/enhancements/issues/<vep_tracking_issue_number>
- VEP design document: [vep.md](./vep.md)

### Why we need it and why it was done in this way

The following tradeoffs were made:
- **Node selector matching over node detection**: VMIs are matched to handlers based on their `spec.nodeSelector` rather than the actual scheduled node. This allows image selection before scheduling and provides predictable behavior.
- **Superset matching**: A VMI matches a handler if its nodeSelector contains all key-value pairs from the handler's nodeSelector. This allows VMIs to have additional constraints while still matching a handler.
- **Feature gate protection**: The feature is behind an Alpha feature gate (`AdditionalVirtHandlers`) to allow for API/behavior changes based on user feedback.

The following alternatives were considered:
- **Namespace-based separation**: Rejected because it increases operational complexity and prevents resource sharing.
- **Per-VMI image override annotations**: Rejected due to security concerns (arbitrary image injection).
- **Webhook-based image mutation**: Rejected because it adds external dependencies and doesn't address virt-handler customization.

### Implementation details

Key files:

| Component | File | Description |
|-----------|------|-------------|
| API Type | `staging/src/kubevirt.io/api/core/v1/componentconfig.go` | `AdditionalVirtHandlerConfig` struct with `name`, `virtHandlerImage`, `virtLauncherImage`, `nodeSelector` fields |
| Feature Gate | `pkg/virt-config/featuregate/active.go` | `AdditionalVirtHandlersGate` (Alpha v1.8.0) |
| Handler Matching | `pkg/virt-controller/services/handlermatcher.go` | `MatchVMIToAdditionalHandler()` and `GetLauncherImageForVMI()` functions |
| DaemonSet Creation | `pkg/virt-operator/resource/generate/install/strategy.go` | Creates additional DaemonSets via `NewHandlerDaemonSetWithConfig()` |
| Anti-Affinity | `pkg/virt-operator/resource/apply/apps.go` | `injectAdditionalHandlerAntiAffinity()` configures primary handler to avoid additional handler nodes |
| Template Service | `pkg/virt-controller/services/template.go` | Selects launcher image and sets `HandlerPoolAnnotation` on pods |
| Workload Updater | `pkg/virt-controller/watch/workload-updater/workload-updater.go` | Detects outdated VMIs based on per-pool launcher image |
| User Docs | `docs/additional-virt-handlers.md` | Configuration guide and troubleshooting |

### Special notes for your reviewer

- The API type `AdditionalVirtHandlerConfig` is added to `staging/src/kubevirt.io/api/core/v1/componentconfig.go` with validation:
  - `name`: Required, pattern `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, max length 48
  - `nodeSelector`: Required, minimum 1 property
  - `virtHandlerImage`, `virtLauncherImage`: Optional
- The primary virt-handler DaemonSet gets `RequiredDuringSchedulingIgnoredDuringExecution` node affinity with `NotIn` expressions to avoid nodes targeted by additional handlers
- The matching logic in `pkg/virt-controller/services/handlermatcher.go` is shared between TemplateService and workload-updater
- Virt-launcher pods get a `kubevirt.io/handler-pool` annotation identifying which handler pool they belong to
- All existing tests pass; new unit and functional tests are included

### Test coverage

Functional tests (`tests/operator/operator.go`):
- Creates additional virt-handler DaemonSet when feature gate is enabled
- Verifies DaemonSet has correct `kubevirt.io/handler-pool` label
- Verifies DaemonSet has configured nodeSelector
- Additional virt-handler pod runs on labeled nodes
- Deletes additional DaemonSet when removed from configuration
- Uses custom images when specified
- Matches VMI to additional handler and sets handler pool annotation on virt-launcher pod
- Configures anti-affinity on primary virt-handler to avoid additional handler nodes

Unit tests (`pkg/virt-controller/services/handlermatcher_test.go`):
- VMI with no nodeSelector returns nil
- Exact nodeSelector match returns handler
- VMI nodeSelector superset returns handler
- VMI nodeSelector subset returns nil
- Value mismatch returns nil
- Multiple handlers returns first match
- Launcher image selection scenarios

### Checklist

- [x] Design: A design document was considered and is present (vep.md in this PR)
- [x] PR: The PR description is expressive enough and will help future contributors
- [x] Code: Write code that humans can understand and Keep it simple
- [x] Refactor: You have left the code cleaner than you found it (Boy Scout Rule)
- [x] Upgrade: Impact of this change on upgrade flows was considered - feature is additive and behind feature gate
- [x] Testing: New unit tests for handler matching logic, functional tests for DaemonSet lifecycle and VMI image selection
- [x] Documentation: User documentation added at `docs/additional-virt-handlers.md`
- [ ] Community: Announcement to kubevirt-dev was considered

### Release note

```release-note
Added support for additional virt-handler DaemonSets to serve heterogeneous node pools. Operators can now configure custom virt-handler and virt-launcher images for specific nodes (e.g., GPU nodes) via `spec.additionalVirtHandlers` in the KubeVirt CR. VMIs with node selectors matching an additional handler's nodeSelector will automatically use that handler's custom virt-launcher image. This feature requires enabling the `AdditionalVirtHandlers` feature gate.
```
