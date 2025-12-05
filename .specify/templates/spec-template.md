# Feature Specification: [FEATURE NAME]

## Metadata

- **Feature ID**: [###]
- **Feature Name**: [FEATURE NAME]
- **Author(s)**: [AUTHOR_NAMES]
- **Created**: [DATE]
- **Status**: Draft
- **KubeVirt Version**: [TARGET_VERSION]
- **Feature Gate**: [FEATURE_GATE_NAME]

**Feature Branch**: `[###-feature-name]`  
**Input**: User description: "$ARGUMENTS"

## Execution Flow (main)

```text
1. Parse user description from Input
   → If empty: ERROR "No feature description provided"
2. Extract key concepts from description
   → Identify: actors, actions, data, constraints
3. For each unclear aspect:
   → Mark with [NEEDS CLARIFICATION: specific question]
4. Fill User Scenarios & Testing section
   → If no clear user flow: ERROR "Cannot determine user scenarios"
5. Generate Functional Requirements
   → Each requirement must be testable
   → Mark ambiguous requirements
6. Identify Key Entities (if data involved)
7. Run Review Checklist
   → If any [NEEDS CLARIFICATION]: WARN "Spec has uncertainties"
   → If implementation details found: ERROR "Remove tech details"
8. Return: SUCCESS (spec ready for planning)
```

## Executive Summary

**One-sentence description**: [BRIEF_DESCRIPTION]

**Business justification**: [WHY_THIS_FEATURE_MATTERS_FOR_KUBEVIRT]

**User impact**: [WHO_BENEFITS_AND_HOW]

## Problem Statement

### Current State

- Describe current KubeVirt limitations or gaps
- Include relevant background about existing virtualization capabilities
- Reference existing APIs, components, or workflows that are insufficient

### Desired State

- What should be possible after this feature is implemented?
- How will this integrate with existing KubeVirt patterns?
- What new capabilities will users have?

### Success Criteria

- **Functional**: What must work for this feature to be considered successful?
- **Non-functional**: Performance, security, reliability, compatibility requirements
- **User Experience**: How should this feel to cluster operators and VM users?

---

## ⚡ Quick Guidelines

- ✅ Focus on WHAT users need and WHY (business value for KubeVirt)
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for KubeVirt stakeholders and business users
- 🎯 Consider KubeVirt's architectural principles and patterns

### Section Requirements

- **Mandatory sections**: Must be completed for every KubeVirt feature
- **Optional sections**: Include only when relevant to the feature
- When a section doesn't apply, remove it entirely (don't leave as "N/A")

### For AI Generation

When creating this spec from a user prompt:

1. **Mark all ambiguities**: Use [NEEDS CLARIFICATION: specific question] for any assumption you'd need to make
2. **Don't guess**: If the prompt doesn't specify something, mark it clearly
3. **Think like a tester**: Every vague requirement should fail the "testable and unambiguous" checklist item
4. **KubeVirt-specific underspecified areas**:
   - User types (cluster operators, VM users, platform developers)
   - Feature gate behavior and lifecycle
   - Component integration (virt-controller, virt-handler, virt-launcher, virt-api)
   - VM lifecycle impact and compatibility
   - Hardware requirements and host environment needs
   - Performance targets and resource overhead
   - Security/compliance considerations for virtualization
   - Kubernetes integration patterns and CRD changes

---

## User Stories *(mandatory)*

### Primary Users

Identify the main personas who will use this feature:

- **Cluster Operators**: Platform administrators managing KubeVirt infrastructure
- **VM Users**: Application teams creating and managing virtual machines
- **Platform Developers**: Teams extending KubeVirt functionality

### User Stories

Format: "As a [user type], I want [capability], so that [benefit]"

**Story 1**: As a [USER_TYPE], I want [CAPABILITY], so that [BENEFIT]

- **Acceptance Criteria**:
  - [ ] [SPECIFIC_MEASURABLE_OUTCOME_1]
  - [ ] [SPECIFIC_MEASURABLE_OUTCOME_2]
  - [ ] [SPECIFIC_MEASURABLE_OUTCOME_3]

**Story 2**: As a [USER_TYPE], I want [CAPABILITY], so that [BENEFIT]

- **Acceptance Criteria**:
  - [ ] [SPECIFIC_MEASURABLE_OUTCOME_1]
  - [ ] [SPECIFIC_MEASURABLE_OUTCOME_2]

### Edge Cases & Error Scenarios

- What happens when [boundary condition]?
- How does system handle [error scenario]?
- What are the failure modes and recovery mechanisms?

## Requirements *(mandatory)*

### Functional Requirements

1. **REQ-F-001**: KubeVirt MUST [specific virtualization capability]
2. **REQ-F-002**: Feature MUST integrate with existing VM lifecycle management (create, start, stop, delete)
3. **REQ-F-003**: Standard KubeVirt networking MUST work with [feature] VMs
4. **REQ-F-004**: Standard KubeVirt storage integrations MUST work with [feature] VMs
5. **REQ-F-005**: Users MUST be able to [key interaction without specifying implementation details]

*Example of marking unclear requirements:*

- **REQ-F-006**: Feature MUST [NEEDS CLARIFICATION: specific behavior not defined]
- **REQ-F-007**: System MUST support [NEEDS CLARIFICATION: scope or scale not specified]

### Non-Functional Requirements

1. **REQ-NF-001**: **Performance**: [PERFORMANCE_REQUIREMENT with measurable criteria]
2. **REQ-NF-002**: **Security**: [SECURITY_REQUIREMENT maintaining KubeVirt security model]
3. **REQ-NF-003**: **Compatibility**: Implementation MUST NOT break existing QEMU/KVM functionality
4. **REQ-NF-004**: **Reliability**: [RELIABILITY_REQUIREMENT for VM operations]
5. **REQ-NF-005**: **Scalability**: [SCALABILITY_REQUIREMENT for cluster operations]

### Kubernetes Integration Requirements

1. **REQ-K8S-001**: MUST follow Kubernetes API conventions for all new fields and resources
2. **REQ-K8S-002**: MUST integrate with VirtualMachine and VirtualMachineInstance CRDs without breaking changes
3. **REQ-K8S-003**: MUST support standard Kubernetes features (RBAC, namespaces, resource quotas, limits)
4. **REQ-K8S-004**: MUST be compatible with KubeVirt's operator pattern and choreographed architecture

### Feature Gate Requirements

1. **REQ-FG-001**: Feature MUST be disabled by default during Alpha stage
2. **REQ-FG-002**: ALL new functionality MUST be controlled by the [FEATURE_GATE_NAME] feature gate
3. **REQ-FG-003**: System MUST function identically to pre-feature state when gate is disabled
4. **REQ-FG-004**: Feature gate MUST follow KubeVirt's established feature gate lifecycle patterns

### Key Entities *(include if feature involves data or new resources)*

- **[Entity 1]**: [What it represents in KubeVirt context, key attributes without implementation]
- **[Entity 2]**: [What it represents, relationships to VMs or other KubeVirt resources]

## Dependencies and Prerequisites *(include if applicable)*

### Upstream Dependencies

- **Kubernetes Version**: [MIN_K8S_VERSION] - [REASON]
- **libvirt Version**: [MIN_LIBVIRT_VERSION] - [REASON]
- **QEMU Version**: [MIN_QEMU_VERSION] - [REASON]
- **Other Dependencies**: [LIST_OTHER_DEPS]

### Host Environment Requirements

- **Kernel Requirements**: [KERNEL_VERSIONS_AND_FEATURES]
- **Hardware Requirements**: [HARDWARE_SPECS]
- **Driver Requirements**: [DRIVER_DEPENDENCIES]
- **Security Requirements**: [SECURITY_CONTEXTS_PERMISSIONS]

### KubeVirt Component Impact

- **virt-operator**: [IMPACT_DESCRIPTION]
- **virt-controller**: [IMPACT_DESCRIPTION]
- **virt-handler**: [IMPACT_DESCRIPTION]
- **virt-launcher**: [IMPACT_DESCRIPTION]
- **virt-api**: [IMPACT_DESCRIPTION]

## API Design (High-Level) *(include if API changes needed)*

**NOTE**: Focus on WHAT the API should do, not HOW it's implemented.

### API Changes

- Describe what new fields might be added to existing CRDs
- Outline any new custom resources that might be needed
- Specify the data these fields/resources should contain

### API Interactions

- How will users interact with this feature through the Kubernetes API?
- What kubectl commands should work?
- How does this integrate with existing VirtualMachine and VirtualMachineInstance resources?

## Security Considerations *(include if applicable)*

### Security Impact

- What new security surfaces does this feature introduce?
- How does this affect KubeVirt's security posture?
- What are the potential attack vectors?

### Security Controls

- What security measures need to be implemented?
- How will access be controlled?
- What audit/logging requirements exist?

---

## Review & Acceptance Checklist

GATE: Automated checks run during main() execution

### Content Quality

- [ ] No implementation details (languages, frameworks, APIs, code structure)
- [ ] Focused on user value and KubeVirt business needs
- [ ] Written for KubeVirt stakeholders and business users
- [ ] All mandatory sections completed
- [ ] KubeVirt architectural principles respected

### Requirement Completeness

- [ ] No [NEEDS CLARIFICATION] markers remain
- [ ] Requirements are testable and unambiguous  
- [ ] Success criteria are measurable
- [ ] Scope is clearly bounded
- [ ] Dependencies and assumptions identified
- [ ] Feature gate requirements specified
- [ ] Component impact assessed
- [ ] Kubernetes integration patterns defined

### KubeVirt-Specific Validation

- [ ] User stories cover cluster operators, VM users, and platform developers
- [ ] VM lifecycle impact considered
- [ ] Compatibility with existing QEMU/KVM functionality addressed
- [ ] Performance and resource overhead implications identified
- [ ] Security implications for virtualization workloads assessed

---

## Execution Status

Updated by main() during processing

- [ ] User description parsed
- [ ] Key concepts extracted
- [ ] Ambiguities marked
- [ ] User scenarios defined
- [ ] Requirements generated
- [ ] Entities identified
- [ ] Review checklist passed

---
