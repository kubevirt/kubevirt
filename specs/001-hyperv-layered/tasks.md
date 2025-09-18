# Tasks: HyperV Layered Support

**Feature ID**: 001  
**Feature Name**: Hyper-V Layered Support  
**Updated**: 2025-09-17  
**Status**: Ready for Implementation  
**Implementation Plan**: [implementation-plan.md](./implementation-plan.md)  
**Feature Specification**: [feature-spec.md](./feature-spec.md)  

---

## Constitutional Compliance ✅

Based on the simplified KubeVirt Constitution, verify:
- [x] **KubeVirt Razor**: Transparent hypervisor optimization, no VM-specific APIs
- [x] **Simplicity**: ≤3 Go packages, direct framework usage, no abstractions
- [x] **Security**: No new privileges beyond Pods, cluster-wide feature gate control
- [x] **Integration-First Testing**: Real environment testing prioritized
- [x] **Service Architecture**: Choreographed components acting independently

**🚨 CONSTITUTIONAL COMPLIANCE VERIFIED - PROCEED TO IMPLEMENTATION**

---

## Implementation Tasks (Simplified)

### Phase 1: Foundation (Week 1-2)

#### Task 1.1: Feature Gate Setup 
**Definition of Done**: `HyperVLayered` feature gate implemented and functional

**Deliverables**:
- [ ] **Feature Gate Registration**
  - Configuration: Alpha stage, default false
  - Pattern: Follow existing feature gate patterns

- [ ] **Basic Tests**
  - File: `pkg/virt-config/featuregate/hyperv_layered_test.go`
  - Verify: Feature gate can be enabled/disabled
  - Verify: No impact when disabled

**Acceptance Criteria**: Feature can be toggled without errors

#### Task 1.2: HyperV Layered Detection Logic
**Definition of Done**: Automatic HyperV Layered device detection in virt-launcher

**Deliverables**:
- [ ] **Device Detection**
  - File: `pkg/virt-launcher/virtwrap/converter/hyperv.go`
  - Function: `hasHyperVLayeredSupport()` - check `/dev/mshv` device exists
  - Integration: Add to existing converter flow

- [ ] **Integration Tests**
  - File: `tests/hyperv_layered_test.go`
  - Test: HyperV Layered detection with real and mocked environments
  - Test: Graceful handling of missing device

**Acceptance Criteria**: HyperV Layered device detection works reliably

#### Task 1.3: Environment Setup 
**Definition of Done**: HyperV Layered test environment ready

**Deliverables**:
- [ ] **Azure HyperV Layered Test Cluster**
  - Environment: Azure VMs with HyperV Layered capability
  - Validation: `/dev/mshv` device available
  - Documentation: Setup requirements and validation steps

**Acceptance Criteria**: Test environment ready for integration testing

### Phase 2: Core Implementation 

#### Task 2.1: Hypervisor Selection
**Definition of Done**: Transparent hypervisor selection in converter

**Research First** (from code analysis):
- **Question**: Do memory overhead calculations in `GetMemoryOverhead()` apply to mshv?
- **Location**: `pkg/virt-controller/services/renderresources.go:393-490`
- **Impact**: Critical for accurate resource allocation

**Deliverables**:
- [ ] **Memory Overhead Research**
  - Task: Benchmark HyperV Layered vs QEMU/KVM memory overhead
  - Task: Validate existing calculations work for mshv
  - Documentation: Findings and any needed adjustments

- [ ] **Hypervisor Selection Logic**
  - File: `pkg/virt-launcher/virtwrap/converter/converter.go`
  - Location: Extend existing KVM detection (lines 1466-1473)
  - Logic: Use mshv when feature gate enabled + device available

- [ ] **Memory Overhead Updates** (if needed)
  - File: `pkg/virt-controller/services/renderresources.go`
  - Task: Adjust calculations if HyperV Layered differs from KVM

**Acceptance Criteria**: VMs automatically use HyperV Layered when available, accurate resource allocation

#### Task 2.2: Domain XML Configuration
**Definition of Done**: libvirt domain properly configured for HyperV Layered

**Deliverables**:
- [ ] **Research libvirt mshv Requirements**
  - Task: Investigate domain XML requirements for mshv
  - Documentation: Required configurations and differences from KVM

- [ ] **Domain Configuration Implementation**
  - File: `pkg/virt-launcher/virtwrap/converter/converter.go`
  - Task: Configure domain for mshv hypervisor
  - Pattern: Follow existing hypervisor-specific patterns

**Acceptance Criteria**: VMs boot successfully with mshv hypervisor

#### Task 2.3: Integration Testing
**Definition of Done**: Comprehensive integration test coverage

**Strategy**: Use existing KubeVirt tests on HyperV Layered cluster to validate transparency

**Deliverables**:
- [ ] **Existing Test Validation**
  - Task: Run existing integration tests on HyperV Layered cluster
  - Verify: Tests pass identically with HyperV Layered enabled/disabled
  - Coverage: VM lifecycle, networking, storage

- [ ] **HyperV Layered-Specific Tests**
  - Test: Hypervisor selection logic
  - Test: Fallback behavior
  - Test: Memory overhead accuracy

- [ ] **GPU Passthrough Validation**
  - Test: GPU passthrough with HyperV Layered

**Acceptance Criteria**: All tests pass, transparent operation validated

### Phase 3: Release Preparation

#### Task 3.1: Documentation
**Definition of Done**: Complete user and operational documentation

**Deliverables**:
- [ ] **User Guide**
  - File: `docs/hyperv-layered-user-guide.md`
  - Content: Transparent operation explanation
  - Content: HyperV Layered cluster setup requirements

- [ ] **Troubleshooting Guide**
  - File: `docs/hyperv-layered-troubleshooting.md`
  - Content: Common issues and resolution
  - Content: HyperV Layered detection and fallback debugging

**Acceptance Criteria**: Documentation enables successful HyperV Layered adoption

#### Task 3.2: Security Review
**Definition of Done**: Security review completed with no issues

**Deliverables**:
- [ ] **Security Analysis**
  - Review: No new privilege escalation paths
  - Review: Proper isolation maintained
  - Review: Feature gate security controls

**Acceptance Criteria**: Security review passed, ready for Alpha release
---


## Critical Research Questions (From Code Analysis)

### High Priority ⚠️
1. **Memory Overhead**: Do extensive QEMU/KVM calculations in `GetMemoryOverhead()` apply to mshv?
2. **Resource Management**: Does KVM device handling work with mshv or need adaptation?

### Medium Priority
1. **Process Memory Limits**: Does `AdjustQemuProcessMemoryLimits()` work with mshv?
2. **Hardware Passthrough**: Different overhead calculations for L1VH vs VFIO?

---

## Success Criteria - Alpha Release

### Functional ✅
- [ ] VMs automatically use HyperV Layered when feature gate enabled and HyperV Layered available
- [ ] Seamless fallback to QEMU/KVM when HyperV Layered unavailable
- [ ] Standard VM specifications work without modification
- [ ] All existing integration tests pass identically

### Technical ✅
- [ ] Memory overhead calculations accurate for HyperV Layered
- [ ] Performance benefits demonstrated
- [ ] Zero regressions in existing functionality
- [ ] Security review completed

### Documentation ✅
- [ ] User guide complete
- [ ] Setup requirements documented
- [ ] Troubleshooting guide available

---

## Risk Mitigation

**Key Risks**:
1. **Memory Overhead Differences** → Research early in Phase 2
2. **HyperV Layered Environment Availability** → Parallel setup in Phase 1
3. **libvirt/QEMU Compatibility** → Version validation upfront
4. **Performance Validation** → Continuous testing throughout

---

*This simplified task breakdown focuses on essential implementation elements while maintaining KubeVirt's architectural principles and quality standards.*
