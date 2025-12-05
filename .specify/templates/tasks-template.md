# Tasks: [FEATURE NAME]

**Feature ID**: [###]  
**Feature Name**: [FEATURE NAME]  
**Updated**: [DATE]  
**Status**: Ready for Implementation  
**Implementation Plan**: [implementation-plan.md](./implementation-plan.md)  
**Feature Specification**: [feature-spec.md](./feature-spec.md)  

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)

```text
1. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: tech stack, libraries, structure, KubeVirt components
2. Load optional design documents:
   → data-model.md: Extract entities → model tasks
   → contracts/: Each file → contract test task  
   → research.md: Extract decisions → setup tasks
3. Verify Constitutional Compliance:
   → Check KubeVirt Razor adherence
   → Validate feature gate requirements
   → Confirm security boundaries
   → Verify integration-first testing plan
4. Generate tasks by category:
   → Foundation: feature gates, detection logic, environment setup
   → Core Implementation: component logic, API changes, domain configuration
   → Integration Testing: real environment validation, E2E workflows
   → Release Preparation: documentation, security review
5. Apply KubeVirt-specific task rules:
   → Feature gates mandatory for new functionality
   → Integration tests before unit tests
   → Real environment testing over mocks
   → Component separation (virt-api, virt-controller, virt-handler, virt-launcher)
6. Number tasks sequentially (T001, T002...)
7. Generate dependency graph with constitutional gates
8. Validate task completeness:
   → Constitutional compliance verified?
   → Feature gate implemented?
   → Integration tests cover real environments?
   → Security review included?
9. Return: SUCCESS (tasks ready for implementation)
```

---

## Constitutional Compliance ✅

Based on the KubeVirt Constitution, verify:

- [ ] **KubeVirt Razor**: Leverages existing Kubernetes patterns, no VM-specific APIs
- [ ] **Feature Gate Requirements**: Alpha-first development with feature gates
- [ ] **Security-First**: No new privileges beyond Pods, proper isolation maintained
- [ ] **Integration-First Testing**: Real environment testing prioritized over mocks
- [ ] **Component Architecture**: Choreographed components acting independently
- [ ] **API Backward Compatibility**: No breaking changes to existing APIs
- [ ] **Reproducible Build System**: Uses established Make/Bazel workflows

### Compliance Gate

CONSTITUTIONAL COMPLIANCE MUST BE VERIFIED BEFORE IMPLEMENTATION

---

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in KubeVirt structure
- Follow KubeVirt component separation patterns

## KubeVirt Path Conventions

- **API Types**: `api/core/v1/`, `staging/src/kubevirt.io/api/`
- **Controllers**: `pkg/virt-controller/`
- **Node Agents**: `pkg/virt-handler/`
- **VM Execution**: `pkg/virt-launcher/`
- **API Server**: `pkg/virt-api/`
- **Tests**: `tests/` (functional), `pkg/*/` (unit)
- **Feature Gates**: `pkg/virt-config/featuregate/`

## Phase 1: Foundation

### Task 1.1: Feature Gate Setup

**Definition of Done**: Feature gate implemented and functional

**Deliverables**:

- [ ] T001 **Feature Gate Registration**
  - Configuration: Alpha stage, default false
  - File: `pkg/virt-config/featuregate/[feature_name].go`
  - Pattern: Follow existing feature gate patterns

- [ ] T002 **Basic Tests**
  - File: `pkg/virt-config/featuregate/[feature_name]_test.go`
  - Verify: Feature gate can be enabled/disabled
  - Verify: No impact when disabled

**Acceptance Criteria**: Feature can be toggled without errors

### Task 1.2: Detection Logic

**Definition of Done**: Automatic feature detection in appropriate component

**Deliverables**:

- [ ] T003 [P] **Detection Implementation**
  - File: `pkg/virt-[component]/[detection_logic].go`
  - Function: Detection logic for feature availability
  - Integration: Add to existing component flow

- [ ] T004 [P] **Integration Tests**
  - File: `tests/[feature]_test.go`
  - Test: Feature detection with real and mocked environments
  - Test: Graceful handling of unavailable feature

**Acceptance Criteria**: Feature detection works reliably

### Task 1.3: Environment Setup

**Definition of Done**: Test environment ready for integration testing

**Deliverables**:

- [ ] T005 **Test Environment**
  - Environment: Configure cluster with feature capability
  - Validation: Feature availability confirmed
  - Documentation: Setup requirements and validation steps

**Acceptance Criteria**: Test environment ready for integration testing

## Phase 2: Integration-First Tests ⚠️ MUST COMPLETE BEFORE PHASE 3

CRITICAL: Integration and E2E tests MUST be written first and MUST FAIL before implementation

- [ ] T006 [P] **Integration Test - Core Functionality**
  - File: `tests/[feature]_integration_test.go`
  - Test: Core feature workflow in real environment
  - Test: Component interaction validation

- [ ] T007 [P] **Integration Test - Error Handling**
  - File: `tests/[feature]_error_test.go`
  - Test: Graceful degradation when feature unavailable
  - Test: Fallback behavior validation

- [ ] T008 [P] **E2E Test - Complete User Journey**
  - File: `tests/[feature]_e2e_test.go`
  - Test: End-to-end user workflow
  - Test: Integration with existing KubeVirt functionality

- [ ] T009 [P] **Functional Test - API Contracts**
  - File: `tests/[feature]_functional_test.go`
  - Test: API behavior validation
  - Test: Backward compatibility verification

## Phase 3: Core Implementation (ONLY after tests are failing)

### Task 3.1: Component Logic

**Definition of Done**: Core feature logic implemented

**Deliverables**:

- [ ] T010 [P] **Primary Component Implementation**
  - File: `pkg/virt-[component]/[feature].go`
  - Task: Implement core feature logic
  - Pattern: Follow existing component patterns

- [ ] T011 [P] **Secondary Component Updates**
  - File: `pkg/virt-[component2]/[integration].go`
  - Task: Update related components for feature support
  - Integration: Maintain choreography patterns

### Task 3.2: API Integration

**Definition of Done**: Feature integrated with KubeVirt APIs

**Deliverables**:

- [ ] T012 **API Type Extensions** (if needed)
  - File: `api/core/v1/types.go`
  - Task: Add new optional fields for feature
  - Command: Run `make generate` after changes

- [ ] T013 **Validation Logic**
  - File: `pkg/virt-api/webhooks/[feature]_webhook.go`
  - Task: Add validation for new feature fields
  - Pattern: Follow existing webhook patterns

## Phase 4: Release Preparation

### Task 4.1: Documentation

**Definition of Done**: Complete user and operational documentation

**Deliverables**:

- [ ] T014 [P] **User Guide**
  - File: `docs/[feature]-user-guide.md`
  - Content: Feature usage and configuration
  - Content: Integration with existing workflows

- [ ] T015 [P] **Troubleshooting Guide**
  - File: `docs/[feature]-troubleshooting.md`
  - Content: Common issues and resolution
  - Content: Feature detection and fallback debugging

### Task 4.2: Security Review

**Definition of Done**: Security review completed with no issues

**Deliverables**:

- [ ] T016 **Security Analysis**
  - Review: No new privilege escalation paths
  - Review: Proper isolation maintained
  - Review: Feature gate security controls

**Acceptance Criteria**: Security review passed, ready for Alpha release

## Dependencies

- Constitutional compliance verification before Phase 1
- Feature gate setup (T001-T002) before detection logic (T003-T004)
- Integration tests (T006-T009) before implementation (T010-T013)
- Core implementation before release preparation (T014-T016)
- Security review (T016) required for Alpha release

## Critical Research Questions

### High Priority ⚠️

1. **Component Integration**: How does feature interact with existing component boundaries?
2. **API Compatibility**: Do changes maintain backward compatibility requirements?
3. **Performance Impact**: What are the resource overhead implications?

### Medium Priority

1. **Feature Detection**: How reliable is feature availability detection?
2. **Fallback Behavior**: What happens when feature is unavailable?
3. **Security Boundaries**: Are existing security constraints maintained?

## Parallel Example

```text
# Launch integration tests together:
Task: "Integration test core functionality in tests/[feature]_integration_test.go"
Task: "Integration test error handling in tests/[feature]_error_test.go"
Task: "E2E test complete user journey in tests/[feature]_e2e_test.go"
Task: "Functional test API contracts in tests/[feature]_functional_test.go"
```

## Success Criteria - Alpha Release

### Functional ✅

- [ ] Feature automatically activates when gate enabled and capability available
- [ ] Seamless fallback when feature unavailable
- [ ] Standard VM specifications work without modification
- [ ] All existing integration tests pass identically

### Technical ✅

- [ ] Performance benefits demonstrated
- [ ] Zero regressions in existing functionality
- [ ] Security review completed
- [ ] Constitutional compliance verified

### Documentation ✅

- [ ] User guide complete
- [ ] Setup requirements documented
- [ ] Troubleshooting guide available

## KubeVirt-Specific Notes

- [P] tasks = different components/files, no dependencies
- Feature gates mandatory for all new functionality
- Integration tests in real environments over mocks
- Follow Make/Bazel workflow: `make generate` after API changes
- Component choreography: avoid direct inter-component dependencies
- Commit after each task with constitutional gate verification

## KubeVirt Task Generation Rules

Applied during main() execution

1. **From Constitutional Requirements**:
   - Feature gate implementation → mandatory foundation task
   - Security considerations → security review task
   - Integration-first testing → integration test tasks before implementation

2. **From API Changes**:
   - CRD modifications → API type extension tasks
   - New fields → validation webhook tasks
   - Schema changes → `make generate` tasks

3. **From Component Architecture**:
   - virt-controller logic → controller reconciliation tasks
   - virt-handler logic → node agent tasks  
   - virt-launcher logic → VM execution tasks
   - virt-api logic → admission webhook tasks

4. **From User Stories**:
   - Each user workflow → E2E test task [P]
   - Feature detection → integration test task [P]
   - Error scenarios → fallback test task [P]

5. **KubeVirt-Specific Ordering**:
   - Constitutional compliance → Feature gates → Detection logic → Integration tests → Implementation → Documentation → Security review
   - Real environment testing before unit testing
   - Component choreography maintained (no tight coupling)

## Validation Checklist

GATE: Checked by main() before returning

- [ ] Constitutional compliance verified
- [ ] Feature gate implemented for new functionality
- [ ] Integration tests prioritized over unit tests  
- [ ] Real environment testing planned
- [ ] Component boundaries respected
- [ ] API backward compatibility maintained
- [ ] Security review included for Alpha release
- [ ] Each task specifies exact KubeVirt file path
- [ ] No task violates component separation principles
