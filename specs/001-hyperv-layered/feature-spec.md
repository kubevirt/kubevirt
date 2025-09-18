# Feature Specification: Hyper-V HyperVLayered Support

## Metadata
- **Feature ID**: 001
- **Feature Name**: Hyper-V Layered Support
- **Author(s)**: ISE KubeVirt Development Team
- **Created**: 2025-09-03
- **Status**: Draft
- **KubeVirt Version**: v1.5.0+
- **Feature Gate**: HyperVLayered

## Executive Summary

**One-sentence description**: Enable KubeVirt to spawn HyperV Layered virtual machines using Microsoft's Hyper-V through the `/dev/mshv` kernel driver, improving peformance of virtualization capabilities on Linux Kubernetes nodes.

**Business justification**: Nested virtualization introduces performance degradation and prevents child VMs from accessing hardware directly (e.g. GPUs, storage, networking). HyperV Layered allows a host hypervisor to directly create and manage child VMs inside a parent VM. These child VMs can be assigned hardware directly, eliminating performance bottlenecks and enabling advanced use cases.

**User impact**: KubeVirt VMs spawned from clusters running in Azure will benefit from improved performance and compatibility.

## Problem Statement

### Current State
KubeVirt currently relies exclusively on the QEMU/KVM stack for virtualization, which presents several limitations for workloads:

- **Performance Gap**: Nested virtualization lacks the optimizations available in native Hyper-V environments due to overhead
- **Integration Challenges**: Organizations using Hyper-V in their existing infrastructure face friction when adopting KubeVirt

The emergence of Microsoft's HyperV Layered architecture provides a new pathway for Linux systems to leverage Hyper-V capabilities through the `/dev/mshv` kernel driver, creating an opportunity to bridge this gap.

### Desired State
After implementation, KubeVirt users should be able to:

- Deploy HyperV Layered VMs and experience performance benefits
- Migrate workloads between QEMU/KVM and Hyper-V backends without changing Kubernetes resource definitions

### Success Criteria

**Functional**:
- VMs boot successfully using Hyper-V Layered backend when feature gate is enabled
- VM lifecycle operations (start, stop, restart, delete) work identically to QEMU/KVM VMs
- Live migration between nodes functions correctly (all nodes are HyperV Layered capable)
- Standard KubeVirt features (hotplug volumes, network interfaces) work with Hyper-V VMs

**Non-functional**:
- Support for direct hardware assignment (GPU, storage, networking) to VMs
- Cluster-wide HyperV Layered readiness assumption simplifies deployment and operation

**User Experience**:
- kubectl commands work identically for HyperV Layered and QEMU/KVM VMs
- No additional Kubernetes RBAC permissions required beyond existing KubeVirt usage
- Clear documentation on HyperV Layered cluster requirements and setup
- Documentation provides clear migration guidance from nested to to HyperV Layered

## User Stories

### Primary Users
- **Cluster Operators**: Platform administrators managing Kubernetes infrastructure
- **Platform Engineers**: Teams building internal platforms on KubeVirt
- **Application Teams**: Developers deploying VM-based applications in Kubernetes

### User Stories

**Story 1**: As a cluster operator running an HyperV Layered-capable Azure cluster, I want to enable Hyper-V Layered support cluster-wide, so that all VMs automatically benefit from near-native performance and direct hardware access.
- **Acceptance Criteria**:
  - [ ] Feature gate can be enabled cluster-wide knowing all nodes support HyperV Layered
  - [ ] All nodes in the cluster are automatically ready for HyperV Layered workloads
  - [ ] Documentation provides clear HyperV Layered cluster setup requirements

**Story 2**: As a platform engineer, I want VMs to automatically use optimal performance, so that workloads run with the best available infrastructure without complex scheduling decisions.
- **Acceptance Criteria**:
  - [ ] All VMs automatically benefit from HyperV Layered when feature gate is enabled
  - [ ] No node affinity or scheduling complexity required
  - [ ] Resource requests and limits work consistently across the cluster
  - [ ] Workload placement is simplified and predictable

**Story 3**: As an application team, I want to leverage hardware acceleration and HyperV Layered-specific features, so that I can achieve optimal performance for GPU workloads and hardware-dependent applications.
- **Acceptance Criteria**:
  - [ ] Direct GPU assignment to Hyper-V VMs is supported
  - [ ] High-performance storage access bypasses nested virtualization overhead

## Requirements

### Functional Requirements
1. **REQ-F-001**: KubeVirt MUST support spawning VMs using Hyper-V (Layered) when the HyperVLayered feature gate is enabled on HyperVLayered-capable clusters
2. **REQ-F-002**: Hyper-V VMs MUST integrate with existing KubeVirt VM lifecycle management (create, start, stop, delete, migrate)
3. **REQ-F-003**: Standard KubeVirt networking MUST work with Hyper-V VMs (services, ingress, network policies)
4. **REQ-F-004**: Standard KubeVirt storage integrations MUST work with Hyper-V VMs (PVCs, storage classes)
5. **REQ-F-005**: Feature MUST assume all nodes in the cluster are HyperVLayered-capable when enabled
6. **REQ-F-006**: Users MUST be able to create VMs without needing to specify hypervisor preference through VirtualMachine specification

### Non-Functional Requirements
1. **REQ-NF-001**: **Hardware Access**: Hyper-V (Layered) VMs MUST support direct hardware assignment for GPU, storage, and networking devices
2. **REQ-NF-002**: **Compatibility**: Implementation MUST NOT break existing QEMU/KVM VM functionality on non-HyperVLayered clusters
3. **REQ-NF-003**: **Cluster Assumption**: Feature MUST assume all nodes are HyperVLayered-capable when feature gate is enabled
4. **REQ-NF-004**: **Simplified Operation**: No node-level readiness detection or heterogeneous scheduling required

### Kubernetes Integration Requirements
1. **REQ-K8S-001**: MUST follow Kubernetes API conventions for all new fields and resources
2. **REQ-K8S-002**: MUST integrate with VirtualMachine and VirtualMachineInstance CRDs without breaking changes
3. **REQ-K8S-003**: MUST support standard Kubernetes features (RBAC, namespaces, resource quotas, limits)
4. **REQ-K8S-004**: MUST be compatible with KubeVirt's operator pattern and choreographed architecture

### Feature Gate Requirements
1. **REQ-FG-001**: Feature MUST be disabled by default during Alpha stage
2. **REQ-FG-002**: ALL Hyper-V functionality MUST be controlled by the HyperVLayered feature gate
3. **REQ-FG-003**: System MUST function identically to pre-feature state when gate is disabled
4. **REQ-FG-004**: Feature gate MUST follow KubeVirt's established feature gate lifecycle patterns

## API Design (High-Level)

**NOTE**: This feature requires **NO API changes** - existing VirtualMachine resources work transparently.

### No API Extensions Required
```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachine
spec:
  template:
    spec:
      domain:
        # Standard KubeVirt VM specification - no changes needed
        # HyperVLayered optimization happens automatically when feature gate is enabled on HyperVLayered cluster
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
        devices:
          disks:
          - name: containerdisk
            disk:
              bus: virtio
          # All existing fields work identically
```

### Transparent Status Reporting
```yaml
status:
  # Standard status fields show actual runtime information
  # Users see optimization happening transparently
  phase: "Running"
  interfaces: [...]
  # Internal field for observability (optional visibility)
  conditions:
  - type: "HypervisorOptimized"
    status: "True"
    reason: "HyperVLayeredEnabled"
    message: "VM is running with HyperVLayered performance optimization"
```

### Cluster-Wide Behavior (No User Configuration)
- **Cluster-Wide Assumption**: All VMs automatically benefit from HyperVLayered when feature gate is enabled on HyperVLayered-capable clusters
- **Simplified Operation**: No node readiness detection or heterogeneous scheduling required
- **Seamless Experience**: VMs automatically use HyperVLayered when feature gate is enabled, QEMU/KVM otherwise
- **Zero Configuration**: No new fields, no decisions required - existing VM specifications work optimally

## Dependencies and Prerequisites

### Upstream Dependencies
- **Kubernetes Version**: 1.31+ - Required for stable CRD features and resource management
- **libvirt Version**: latest - Needed for Hyper-V integration capabilities
- **QEMU Version**: latest - Required for hybrid hypervisor support
- **Microsoft mshv Driver**: Latest stable - Core dependency for Hyper-V Layered functionality
- **Linux Kernel**: 6.6 - Needed for mshv driver compatibility

### Host Environment Requirements
- **HyperVLayered Cluster Assumption**: ALL nodes in the cluster MUST be HyperVLayered-capable when feature gate is enabled
- **Azure Environment**: 
  - Azure HyperVLayered Capable VMs
  - Cluster with HyperVLayered-capable VM sizes for WORKER nodes
  - Hyper-V role enabled on ALL host Azure VMs
- **Kernel Requirements**: 
  - Linux kernel 6.6+ with `mshv` driver support on ALL nodes
  - `CONFIG_HYPERV=y` kernel configuration on ALL nodes
- **Hardware Requirements**:
  - Minimum 16GB RAM for meaningful HyperVLayered VM deployment per node
  - Hardware virtualization extensions accessible from guest on ALL nodes
  - [NEEDS CLARIFICATION] GPU requirement
- **Driver Requirements**:
  - `/dev/mshv` device accessible to KubeVirt components on ALL nodes
  - Proper permissions and SELinux contexts for mshv access on ALL nodes
- **Security Requirements**:
  - KubeVirt service account must have access to /dev/mshv on ALL nodes
  - SELinux policies updated for Hyper-V device access on ALL nodes

### KubeVirt Component Impact
- **virt-operator**: Must validate HyperVLayered cluster-wide readiness during deployment and feature gate enablement
- **virt-controller**: **MODERATE CHANGES REQUIRED** - Memory overhead calculations for QEMU/KVM vs Hyper-V will require non-trivial code changes to accurately account for different hypervisor memory requirements and resource planning
- **virt-handler**: **MINOR CHANGES REQUIRED** - May need updates for HyperVLayered-specific node validation and hypervisor capability detection, though cluster-wide assumption simplifies most logic
- **virt-launcher**: Must support seamless execution on HyperVLayered when feature gate is enabled, QEMU/KVM otherwise
- **virt-api**: **NO CHANGES REQUIRED** - No complex validation or scheduling logic required

## Compatibility and Migration

### Backward Compatibility
- Existing VirtualMachine and VirtualMachineInstance resources continue working unchanged
- QEMU/KVM VMs are unaffected by HyperVLayered feature introduction and continue to use existing behavior
- kubectl commands maintain identical syntax and behavior regardless of underlying hypervisor
- No breaking changes to existing KubeVirt APIs - all existing workloads continue to work as expected

### Upgrade/Downgrade Scenarios
- **Upgrade**: Feature gate remains disabled, preserving existing behavior until explicitly enabled
- **Downgrade**: Running HyperVLayered VMs should be gracefully stopped before KubeVirt downgrade
- **Configuration Migration**: Existing VM configurations work with HyperVLayered when feature gate is enabled

### Multi-Version Support
- Same VM specification works across KubeVirt versions (with and without HyperVLayered support)
- Feature gate controls cluster-wide HyperVLayered behavior - when enabled, all VMs use HyperVLayered; when disabled, all VMs use QEMU/KVM
- Rolling updates can proceed safely with homogeneous HyperVLayered clusters

## Security Considerations

### Compliance
- Feature must maintain KubeVirt's existing security certifications and compliance posture
- No additional privileged access requirements beyond current KubeVirt operations
- Audit logging for Hyper-V specific operations must integrate with Kubernetes audit mechanisms

## Testing Strategy

### Unit Testing
- HyperVLayered feature gate enable/disable logic
- HyperVLayered driver integration modules
- API validation and defaulting logic

### Integration Testing
- HyperVLayered VM lifecycle operations
- HyperVLayered cluster-wide behavior with feature gate
- Storage and networking integration scenarios with HyperVLayered

### End-to-End Testing
- Complete VM deployment workflows on HyperVLayered clusters
- Multi-node HyperVLayered cluster scenarios
- Feature gate enable/disable with active workloads
- Performance benchmarking with HyperVLayered optimization

### Performance Testing
- VM startup time comparisons
- Resource utilization profiling
- Concurrent VM scalability testing
- Network and storage I/O performance validation

### Compatibility Testing
- Various Kubernetes distributions and versions

## Documentation Requirements

### User Documentation
- **Getting Started Guide**: Setting up HyperVLayered support in cluster environments
- **API Reference**: Complete documentation of HyperVLayered behavior and feature gate usage
- **Migration Guide**: Understanding HyperVLayered cluster requirements and benefits
- **Troubleshooting Guide**: Common issues and resolution procedures
- **Best Practices**: Recommendations for HyperVLayered cluster optimization

### Developer Documentation
- **Architecture Overview**: How HyperVLayered integration fits into KubeVirt architecture
- **Integration Guide**: Understanding HyperVLayered operation in existing KubeVirt workflows
- **Testing Framework**: How to run and extend HyperVLayered integration tests

### Operations Documentation
- **Installation Procedures**: Setting up HyperVLayered cluster configuration
- **Monitoring Setup**: Configuring observability for HyperVLayered operation

## Observability and Monitoring

### Metrics
- `kubevirt_hyperv_HyperVLayered_vms_total` - Count of running HyperVLayered VMs
- `kubevirt_hyperv_HyperVLayered_vm_startup_duration_seconds` - VM startup time metrics for HyperVLayered VMs
- `kubevirt_hyperv_HyperVLayered_device_errors_total` - `/dev/mshv` interaction error counts
- `kubevirt_HyperVLayered_hardware_passthrough_total` - Count of VMs using direct hardware access

### Logging
- HyperVLayered VM lifecycle events (start, stop, migration)
- `/dev/mshv` driver interaction events and hardware passthrough operations
- Hardware assignment and deassignment events
- HyperVLayered specific error conditions and recovery actions

### Events
- Kubernetes events for HyperVLayered VM state changes
- Events for HyperVLayered cluster enablement and hardware passthrough availability
- Alerts for `/dev/mshv` driver availability issues
- Events for feature gate state changes affecting workload execution
- Hardware assignment/deassignment event notifications

## Open Questions and Risks

### Dependencies on External Projects
- **Microsoft Patch Development**: Dependency on patches to QEMU/libvirt for Hyper-V support
  - **Mitigation**: Active engagement with Microsoft teams, version pinning, compatibility testing

## Future Considerations

### Evolution Path
- **Multiple Hypervisor Support**: Extending support to other hypervisors (e.g., VMware, Xen) through similar abstraction layers
- **Hybrid Clusters**: Future support for mixed HyperVLayered and QEMU/KVM clusters with intelligent scheduling
- **Enhanced Hardware Integration**: Broader support for GPU types, advanced storage, and high-performance networking
- **Performance Optimization**: HyperVLayered specific performance tuning and advanced hardware acceleration features

### Related Features
- **Hardware Passthrough Orchestration**: Advanced scheduling and management of hardware-accelerated workloads
- **GPU Workload Optimization**: Integration with NVIDIA, AMD, and Intel GPU acceleration for AI/ML workloads
- **High-Performance Storage**: HyperVLayered optimized storage performance for database and analytics workloads

## Success Metrics

### Definition of Done
- [ ] VMs successfully boot and run using HyperVLayered with near-native performance
- [ ] Hardware passthrough (GPU, storage, networking) is functional and provides measurable performance benefits
- [ ] All functional requirements are implemented and tested with transparent user experience
- [ ] Performance requirements are met with validated HyperVLayered operation
- [ ] Documentation is complete and published
- [ ] Feature is stable and ready for Alpha release with seamless operation

### Key Performance Indicators
1. **User Experience**: Transparency metrics showing users are unaware of underlying hypervisor complexity
2. **Reliability**: HyperVLayered VM failure rate compared to nested baseline
3. **User Satisfaction**: Community feedback on seamless Azure integration and performance benefits

---

This specification provides the foundation for implementing Hyper-V HyperVLayered support in KubeVirt while maintaining the project's architectural principles and user experience standards. The feature will enable organizations to leverage HyperVLayered capabilities within Azure, eliminating nested virtualization performance penalties and enabling direct hardware access for high-performance workloads.
