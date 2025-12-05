
# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

## Execution Flow (/plan command scope)

```text
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect KubeVirt Component Architecture (virtualization, controller, operator patterns)
   → Set Component Integration Strategy based on choreography patterns
3. Fill the Constitution Check section based on the content of the constitution document
   → Validate KubeVirt Razor compliance (Pod-VM parity principle)
   → Check Feature Gate implementation requirements
   → Verify Integration-First testing approach
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If constitutional gates fail: ERROR "Must address constitutional compliance"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md (KubeVirt patterns & dependencies)
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
   → Focus on libvirt/QEMU/Kubernetes integration points
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, component-integration.md, .github/copilot-instructions.md
   → Generate CRD API contracts following Kubernetes conventions
   → Design controller choreography patterns
   → Plan feature gate implementation strategy
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Ensure KubeVirt architectural compliance
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe KubeVirt-specific task generation approach (DO NOT create tasks.md)
   → Include component-specific implementation phases
   → Plan constitutional gate validation throughout implementation
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 8. Implementation phases are executed by other commands:

- Phase 2: /tasks command creates tasks.md with KubeVirt patterns
- Phase 3+: Implementation execution following constitutional principles

## Summary

[Extract from feature spec: primary requirement + KubeVirt architectural integration approach from research]

## Technical Context

**Language/Version**: Go [VERSION] (KubeVirt standard)  
**KubeVirt Framework**: [KUBEVIRT_VERSION or NEEDS CLARIFICATION]  
**Kubernetes API**: [API_VERSION or NEEDS CLARIFICATION]  
**Primary Dependencies**: [e.g., libvirt 9.0+, QEMU 7.2+, controller-runtime or NEEDS CLARIFICATION]  
**Host Dependencies**: [e.g., kernel modules, drivers, container runtimes or NEEDS CLARIFICATION]  
**Component Architecture**: [affected components: virt-operator/controller/handler/launcher/api or NEEDS CLARIFICATION]  
**Testing Framework**: Ginkgo/Gomega (KubeVirt standard)  
**Target Platform**: Linux Kubernetes clusters  
**Feature Gate Strategy**: [feature gate name and implementation approach or NEEDS CLARIFICATION]  
**Performance Goals**: [VM-specific, e.g., startup latency, memory overhead, I/O throughput or NEEDS CLARIFICATION]  
**Security Constraints**: [e.g., Pod security standards, capabilities, resource access or NEEDS CLARIFICATION]  
**Scale/Scope**: [e.g., VMI count, node count, resource limits or NEEDS CLARIFICATION]  
**Integration Points**: [existing KubeVirt features, K8s resources, external systems or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Implementation Gates (CRITICAL - Must pass before any implementation)

#### Simplicity Gate (Constitution Article VII)

- [ ] Using ≤3 new Go packages for initial implementation?
- [ ] No future-proofing or premature optimization?
- [ ] Minimal additional complexity to existing codebase?
- [ ] Clear justification for any new abstractions?

#### Anti-Abstraction Gate (Constitution Article VIII)

- [ ] Using existing KubeVirt and Kubernetes patterns directly?
- [ ] Not creating unnecessary wrapper layers?
- [ ] Following established libvirt/QEMU integration patterns?
- [ ] Single clear representation for feature configuration?

#### KubeVirt Architectural Gates

- [ ] Follows the KubeVirt Razor principle (Pod-VM feature parity)?
- [ ] Feature gate implemented and disabled by default?
- [ ] Integration and e2e tests planned before unit tests?
- [ ] Uses choreography pattern (components react to observed state)?
- [ ] Security considerations documented with threat model?
- [ ] Does not grant VM users capabilities beyond what Pods already have?
- [ ] Backward compatibility maintained for existing APIs?
- [ ] Component integration follows established KubeVirt patterns?

#### Dependency Gate

- [ ] All upstream dependencies confirmed available?
- [ ] Host environment requirements validated?
- [ ] libvirt/QEMU feature availability confirmed?
- [ ] Kubernetes API compatibility verified?

**If any gate fails, implementation must pause until issues are resolved.**

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md                    # This file (/plan command output)
├── research.md                # Phase 0 output (/plan command)
├── data-model.md              # Phase 1 output (/plan command)
├── quickstart.md              # Phase 1 output (/plan command)
├── component-integration.md   # Phase 1 output (/plan command)
├── contracts/                 # Phase 1 output (/plan command)
│   ├── api-schema.yaml        # CRD modifications
│   └── integration-contracts.md
├── implementation-details/    # Detailed technical specs
└── tasks.md                   # Phase 2 output (/tasks command - NOT created by /plan)
```

### KubeVirt Component Architecture

```text
# KubeVirt Component Structure (choose applicable components)
pkg/
├── virt-config/
│   └── featuregate/          # Feature gate implementation
├── virt-operator/            # [IF_OPERATOR_CHANGES]
│   ├── resources/
│   └── [FEATURE_COMPONENTS]
├── virt-controller/          # [IF_CONTROLLER_CHANGES]
│   ├── watch/
│   └── [FEATURE_CONTROLLERS]
├── virt-handler/             # [IF_HANDLER_CHANGES]
│   ├── node/
│   └── [FEATURE_HANDLERS]
├── virt-launcher/            # [IF_LAUNCHER_CHANGES]
│   ├── domain/
│   └── [FEATURE_DOMAIN_LOGIC]
├── virt-api/                 # [IF_API_CHANGES]
│   ├── webhooks/
│   │   ├── validating-webhook/
│   │   └── mutating-webhook/
│   └── [FEATURE_VALIDATIONS]
└── [OTHER_SHARED_PACKAGES]

api/
├── core/v1/                  # [IF_CRD_CHANGES]
│   ├── types.go             # VMI/VM spec modifications
│   └── [FEATURE_TYPES]
└── openapi-spec/            # Generated API documentation

tests/
├── [FEATURE_UNIT_TESTS]     # Component-specific unit tests
├── [FEATURE_INTEGRATION_TESTS] # KubeVirt integration tests
└── [FEATURE_E2E_TESTS]      # End-to-end user workflows

cmd/
└── [IF_NEW_BINARIES_NEEDED] # New command-line tools (rare)
```

### Component Integration Map

```text
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  virt-operator  │────│ virt-controller │────│   virt-handler  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
[OPERATOR_ROLE]         [CONTROLLER_ROLE]         [HANDLER_ROLE]
                                 │                       │
                                 ▼                       ▼
                        ┌─────────────────┐    ┌─────────────────┐
                        │   virt-api      │    │ virt-launcher   │
                        └─────────────────┘    └─────────────────┘
                                 │                       │
                                 ▼                       ▼
                        [API_VALIDATION_ROLE]    [VM_EXECUTION_ROLE]
```

**Structure Decision**: [Document which KubeVirt components are affected and how they integrate with the choreography pattern]

## Phase 0: KubeVirt Architecture Research

1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → KubeVirt-specific research task
   - For each component dependency → choreography patterns task
   - For each integration point → existing KubeVirt patterns task
   - For each host dependency → libvirt/QEMU capability research

2. **Generate and dispatch KubeVirt research agents**:

   ```text
   For each unknown in Technical Context:
     Task: "Research {unknown} for KubeVirt {component} integration"
   For each component interaction:
     Task: "Find KubeVirt choreography patterns for {interaction}"
   For each virtualization feature:
     Task: "Research libvirt/QEMU capabilities for {feature}"
   For each API change:
     Task: "Research Kubernetes CRD best practices for {change}"
   ```

3. **Consolidate findings** in `research.md` using KubeVirt format:
   - **KubeVirt Razor Analysis**: How feature maintains Pod-VM parity
   - **Component Choreography**: How components will coordinate
   - **Technology Decisions**: libvirt/QEMU/K8s API choices with rationale
   - **Feature Gate Strategy**: Implementation approach and backward compatibility
   - **Security Model**: How feature maintains Pod security equivalence
   - **Integration Points**: Existing KubeVirt features that interact
   - **Alternatives Considered**: What approaches were evaluated and rejected

**Output**: research.md with all NEEDS CLARIFICATION resolved and KubeVirt architectural decisions documented

## Phase 1: KubeVirt Design & Integration Contracts

Prerequisites: research.md complete with all constitutional gates passed

1. **Extract KubeVirt entities from feature spec** → `data-model.md`:
   - **CRD Modifications**: VMI/VM spec and status field additions
   - **Controller State Models**: State machines for reconciliation loops
   - **API Validation Rules**: Webhook validation requirements
   - **Feature Gate Integration**: How feature toggles affect API behavior
   - **Component Data Flow**: How data flows between virt-* components

2. **Generate KubeVirt API contracts** from functional requirements:
   - **CRD Schema Contracts**: OpenAPI schema for new API fields
   - **Controller Reconciliation Contracts**: Expected state transitions
   - **Component Integration Contracts**: How virt-* components coordinate
   - **Feature Gate Contracts**: API behavior with feature enabled/disabled
   - Output Kubernetes-compliant schemas to `/contracts/`

3. **Generate KubeVirt integration contracts** from choreography patterns:
   - **Watch Contracts**: What resources each controller observes
   - **Event Contracts**: What events trigger reconciliation
   - **Status Update Contracts**: How components communicate via status
   - **Error Handling Contracts**: How failures propagate through system

4. **Generate contract tests** following KubeVirt patterns:
   - **API Validation Tests**: CRD schema validation (must fail initially)
   - **Controller Integration Tests**: Component interaction scenarios
   - **Feature Gate Tests**: Behavior verification with feature on/off
   - **End-to-End Tests**: User workflow validation scenarios
   - Use Ginkgo/Gomega framework following KubeVirt conventions

5. **Create component integration documentation** → `component-integration.md`:
   - **Architecture Diagram**: Component interaction visualization
   - **Choreography Flows**: How components react to state changes
   - **Security Boundaries**: How feature maintains Pod security equivalence
   - **Performance Characteristics**: Expected resource usage and scaling

6. **Extract test scenarios** from user stories:
   - Each story → KubeVirt E2E test scenario
   - Quickstart test = VMI lifecycle validation steps
   - Integration test = component coordination validation

7. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh copilot`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW KubeVirt patterns from current plan
   - Include component integration context
   - Update constitutional compliance markers
   - Keep under 150 lines for token efficiency
   - Output to repository root (`.github/copilot-instructions.md`)

**Output**: data-model.md, component-integration.md, /contracts/*, failing KubeVirt integration tests, quickstart.md, .github/copilot-instructions.md

## Phase 2: KubeVirt Task Planning Approach

This section describes what the /tasks command will do - DO NOT execute during /plan

**KubeVirt Task Generation Strategy**:

- Load `.specify/templates/tasks-template.md` as base with KubeVirt patterns
- Generate tasks from Phase 1 KubeVirt design docs (component-integration, contracts, data-model)
- **Foundation Tasks**: Feature gate implementation, CRD schema updates [P]
- **Component Tasks**: Each virt-* component modification with constitutional gates
- **Integration Tasks**: Controller choreography, API validation, status updates
- **Testing Tasks**: Integration-first approach (E2E before unit tests)
- **Validation Tasks**: Constitutional compliance verification throughout

**KubeVirt Ordering Strategy**:

- **Integration-First Order**: E2E tests → Integration tests → Unit tests
- **Component Dependency Order**: APIs → Controllers → Handlers → Launchers
- **Constitutional Gates**: Validate compliance at each major milestone
- **Feature Gate Strategy**: Disabled-by-default implementation with toggle validation
- Mark [P] for parallel execution (independent components/files)

**KubeVirt-Specific Task Categories**:

- **Pre-Implementation Gates**: Constitutional compliance validation
- **Foundation Phase**: Feature gates, API schema, basic infrastructure
- **Core Implementation Phase**: Component integration, controller logic
- **Integration Testing Phase**: Cross-component validation, choreography testing
- **Production Readiness Phase**: Security review, performance validation, documentation

**Estimated Output**: 35-45 numbered, ordered tasks in tasks.md with constitutional gate checkpoints

**IMPORTANT**: This phase is executed by the /tasks command following KubeVirt constitutional principles, NOT by /plan

## Phase 3+: Future Implementation

These phases are beyond the scope of the /plan command

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking

Fill ONLY if Constitution Check has violations that must be justified

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |

## Progress Tracking

This checklist is updated during execution flow

**Phase Status**:

- [ ] Phase 0: Research complete (/plan command)
- [ ] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:

- [ ] Initial Constitution Check: PASS
- [ ] Post-Design Constitution Check: PASS
- [ ] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---

## KubeVirt Implementation Guidelines

### Constitutional Compliance Throughout Implementation

- **KubeVirt Razor Principle**: Continuously validate Pod-VM feature parity
- **Feature Gate Discipline**: All new functionality behind disabled-by-default gates
- **Integration-First Testing**: E2E and integration tests before unit tests
- **Component Choreography**: Independent component reactions to observed state
- **Security Equivalence**: VM users cannot exceed Pod user capabilities
- **Simplicity Maintenance**: Resist abstraction layers and complexity

### Quality Standards for KubeVirt Features

- **Code Quality**: Follow established KubeVirt patterns and conventions
- **Test Coverage**: Comprehensive integration and E2E test coverage
- **Documentation**: User guides, API docs, and troubleshooting information
- **Security Review**: Threat model analysis and security boundary validation
- **Performance Validation**: Resource usage and scaling characteristics
- **Backward Compatibility**: Maintain API compatibility and migration paths

### Implementation Execution Guidelines

- **Follow Constitutional Gates**: Validate compliance at each phase
- **Use Established Patterns**: Leverage existing KubeVirt architectural patterns
- **Integration-First Development**: Build and test component interactions early
- **Feature Gate Compliance**: Ensure feature can be safely disabled
- **Security-First Approach**: Maintain Pod security model equivalence

---
*Based on KubeVirt Constitution v1.0.0 - See `.specify/memory/constitution.md`*
