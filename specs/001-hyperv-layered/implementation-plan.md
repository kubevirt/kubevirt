# Implementation Plan: Hyper-V HyperVLayered Support

**Branch**: `spec/hyperv-layered` | **Date**: 2025-09-04 | **Spec**: [feature-spec.md](./feature-spec.md)
**Input**: Feature specification from `/specs/001-hyperv-layered/feature-spec.md`

## Prerequisites

Before implementation begins, ensure:
- [x] Feature specification is approved and stable
- [x] All dependencies are available in target versions  
- [x] Development environment supports required tools and libraries
- [x] Required approvals and permissions are obtained

## Architecture Overview

### Component Integration Map
**IMPORTANT**: This implementation plan should remain high-level and readable. Any code samples, detailed algorithms, or extensive technical specifications must be placed in the appropriate `implementation-details/` file.

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  virt-operator  │────│ virt-controller │────│   virt-handler  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
   [FEATURE GATE]         [DEVICE REGISTRATION]      [MSHV DRIVER]
                                 │                       │
                                 ▼                       ▼
                        ┌─────────────────┐    ┌─────────────────┐
                        │   virt-api      │    │ virt-launcher   │
                        └─────────────────┘    └─────────────────┘
                                 │                       │
                                 ▼                       ▼
                            [NO CHANGES]        [DOMAIN XML CONFIG]                                               
```
## Constitutional Compliance

### ✅ KubeVirt Razor Check
**Principle**: "If something is useful for Pods, we should not implement it only for VMs"
- **Application**: HyperVLayered provides transparent hypervisor optimization similar to container runtime selection - operates at infrastructure level with no VM-specific APIs
- **Kubernetes Integration**: Reuses existing VirtualMachine CRDs, feature gate patterns, controller reconciliation, and resource management

### ✅ Simplicity Check  
- **New Go packages**: 2 (target: ≤3) - feature gate logic and HyperVLayered hypervisor detection
- **Framework usage**: Direct libvirt/QEMU integration through existing converter patterns, no wrapper abstractions
- **Abstractions avoided**: No hypervisor abstraction layer, no complex scheduling logic, no new CRDs

### ✅ Security Check
- **Privilege boundary**: Feature does not grant VM users capabilities beyond Pods
- **Implementation**: Zero API changes, no new RBAC requirements, cluster-wide feature gate controls behavior transparently

## Implementation Strategy

### Phase 1: Foundation
**Objective**: Basic infrastructure and feature gate setup

**Deliverables**:
- [ ] Feature gate `HyperVLayered` in `pkg/virt-config/featuregate/`
- [ ] `/dev/mshv` device detection logic in virt-launcher converter
- [ ] Basic integration tests for HyperVLayered detection and feature toggle
- [ ] Research HyperVLayered memory overhead characteristics vs QEMU/KVM

**Exit Criteria**: Feature can be toggled on/off without errors, mshv device detection works

### Phase 2: Core Implementation 
**Objective**: Transparent HyperVLayered hypervisor selection

**Deliverables**:
- [ ] Automatic hypervisor selection in `pkg/virt-launcher/virtwrap/converter/converter.go` (extends existing KVM detection at line 1466-1473)
- [ ] HyperVLayered-aware memory overhead calculations in `pkg/virt-controller/services/renderresources.go` (modifies GetMemoryOverhead function)
- [ ] libvirt domain XML configuration for mshv hypervisor backend
- [ ] mshv device resource management (extends existing KVM device handling)
- [ ] End-to-end VM lifecycle with HyperVLayered hypervisor

**Exit Criteria**: VMs automatically use HyperVLayered when `/dev/mshv` available and feature gate enabled, graceful fallback to QEMU/KVM works

### Phase 3: Release Preparation
**Objective**: Performance validation and documentation

**Deliverables**:
- [ ] Comprehensive test suite (integration + e2e) on both HyperVLayered and standard clusters
- [ ] Performance benchmarking: HyperVLayered vs nested QEMU/KVM validation
- [ ] Memory overhead validation and adjustment if needed
- [ ] Hardware (GPU) passthrough testing with HyperVLayered
- [ ] Documentation and troubleshooting guides

**Exit Criteria**: Ready for Alpha release behind feature gate with validated performance benefits

## Testing Strategy

### Integration-First Approach
Following KubeVirt's integration-first testing philosophy:

1. **Integration Tests**: HyperVLayered detection, hypervisor selection, and domain configuration with real libvirt/QEMU/mshv
2. **End-to-End Tests**: Complete VM lifecycle on HyperVLayered-capable Azure clusters and standard clusters  
3. **Unit Tests**: Hypervisor selection logic, memory overhead calculations, and mshv detection edge cases

**Test Environment**: Real Kubernetes cluster with actual HyperVLayered-capable Azure VMs and standard nodes for fallback testing

## API Design

### VMI Spec Changes
```yaml
# NO API CHANGES REQUIRED
# Existing VirtualMachine resources work transparently
# HyperVLayered optimization happens automatically when feature gate enabled
```

## File Organization

### New Files
```
tests/hyperv_layered_test.go
```

### Modified Files (Based on Code Analysis)
- `pkg/virt-launcher/virtwrap/converter/converter.go`: Add HyperVLayered hypervisor selection to existing KVM detection logic
- `pkg/virt-controller/services/renderresources.go`: Register mshv device and validate/adjust HyperVLayered memory overhead calculations in GetMemoryOverhead() 
- `pkg/virt-controller/services/template.go`: Add HyperVLayered-specific overhead constants if needed 
- `pkg/virt-handler/isolation/detector.go`: Validate AdjustQemuProcessMemoryLimits works with mshv 
- `pkg/virt-handler/controller.go`: claimDeviceOwnership updates for mshv device 
- `pkg/virt-config/featuregate/active.go` and `pkg/virt-config/feature-gates.go`: New feature gate implementation

## Success Criteria

### Functional
- [ ] All feature spec user stories implemented
- [ ] VMs automatically use HyperVLayered when feature gate enabled on HyperVLayered-capable clusters
- [ ] VMs gracefully fall back to QEMU/KVM when `/dev/mshv` unavailable
- [ ] VM lifecycle operations identical between HyperVLayered and QEMU/KVM
- [ ] No regressions in existing functionality when feature disabled

### Technical  
- [ ] Follows KubeVirt coding patterns and service-oriented architecture
- [ ] Integration and e2e tests pass on both HyperVLayered and standard clusters
- [ ] Memory overhead calculations accurate for HyperVLayered hypervisor
- [ ] Performance meets requirements (near-native HyperVLayered performance vs nested virtualization)
- [ ] Security review completed with no privilege escalation

### Documentation
- [ ] User guide updated with HyperVLayered cluster setup requirements
- [ ] Performance benchmarking documentation showing HyperVLayered benefits
- [ ] Troubleshooting guide for HyperVLayered detection and configuration issues

## Risk Mitigation

**Key Risks**:
1. **Memory Overhead Differences**: HyperVLayered may have different overhead characteristics than QEMU/KVM → **Mitigation**: Research and benchmark actual HyperVLayered overhead, adjust calculations if needed
2. **HyperVLayered Driver Availability**: Limited to specific Azure VM types → **Mitigation**: Graceful fallback to QEMU/KVM, clear documentation of requirements
3. **libvirt/QEMU HyperVLayered Support**: Dependency on upstream mshv backend maturity → **Mitigation**: Version pinning, compatibility testing, active upstream engagement
4. **Process Memory Management**: QEMU memory limit functions may not work with mshv → **Mitigation**: Validate and adapt AdjustQemuProcessMemoryLimits for mshv

## Critical Research Questions (From Code Analysis)

### High Priority
1. **Memory Overhead Validation**: Do the extensive QEMU/KVM memory calculations in `GetMemoryOverhead()` apply to mshv hypervisor?
2. **Device Resource Management**: Does existing KVM device resource handling (`devices.kubevirt.io/kvm`) work with mshv or need adaptation?

### Medium Priority  
1. **Process Memory Limits**: Does `AdjustQemuProcessMemoryLimits()` work with mshv or need modification?
2. **Hardware Passthrough Overhead**: Is the 1Gi VFIO overhead calculation accurate for HyperVLayered hardware passthrough?

---

## Implementation Checklist

Before starting implementation, verify:
- [x] Follows the KubeVirt Razor principle?
- [x] Feature gate implemented and disabled by default?
- [x] Integration and e2e tests planned before unit tests?
- [x] Uses existing frameworks directly (Kubernetes, libvirt, QEMU)?
- [x] Security considerations documented?
- [x] Does not grant VM users capabilities beyond what Pods already have?

**Remember**: When in doubt, ask "Does this make VMs feel more like native Kubernetes workloads?"

---

*This implementation plan focuses on essential elements while addressing critical technical details identified in the code analysis. HyperVLayered support provides transparent hypervisor optimization without breaking existing KubeVirt patterns.*
