<!--
Sync Impact Report:
- Version change: new → 1.0.0
- Initial constitution creation for KubeVirt project
- Added core principles aligned with Kubernetes virtualization patterns
- Added The KubeVirt Razor as foundational principle
- Enhanced Testing Discipline with integration-first approach
- Added Feature Gate Requirements and Security-First principles
- Templates requiring updates: ✅ all consistent
- Follow-up TODOs: none
-->

# KubeVirt Constitution

## Core Principles

### The KubeVirt Razor (FOUNDATIONAL)

**"If something is useful for Pods, we should not implement it only for VMs"**

This is the fundamental decision-making framework that guides all feature development. Every feature MUST leverage existing Kubernetes patterns and APIs rather than creating VM-specific alternatives. Features MUST NOT grant VM users capabilities beyond what Kubernetes already provides for container workloads. Installing KubeVirt must never grant users permissions they don't already have for native workloads.

**Rationale**: Ensures consistent user experience, reduced cognitive load, natural integration with existing tooling, and reduced maintenance burden while maintaining security boundaries.

### I. Kubernetes-Native Architecture

All KubeVirt features MUST extend Kubernetes through CRDs and controllers following cloud-native patterns. Components run as pods with clear separation: virt-api (admission/validation), virt-controller (orchestration), virt-handler (node agent), virt-launcher (VMI execution). Choreography pattern is mandatory - controllers react to CR changes rather than centralized reconciliation. No direct host system modifications outside containerized environments.

**Rationale**: Ensures KubeVirt remains a true Kubernetes add-on, maintainable through standard K8s tooling and lifecycle management.

### II. Feature Gate Requirements (NON-NEGOTIABLE)

All new features MUST begin life disabled by default behind a feature gate. Feature lifecycle: Alpha (disabled) → Beta (enabled in dev) → GA (enabled in prod) → Deprecated → Discontinued. No exceptions for any new functionality regardless of perceived stability or simplicity.

**Rationale**: Enables safe progressive rollout, allows for community feedback, and provides escape hatch for problematic features without breaking existing deployments.

### III. Reproducible Build System (NON-NEGOTIABLE)

All builds MUST use `make bazel-build-images` through containerized environment (`hack/dockerized`). Image content derived from RPM dependency trees via `make rpm-deps` + `hack/rpm-deps.sh`. NO direct package installation in Dockerfiles - extend RPM lists instead. Generated code changes require `make generate` + `make generate-verify`. Multi-architecture support mandatory without architecture-specific conditionals.

**Rationale**: Guarantees consistent, auditable builds across environments and prevents supply chain vulnerabilities through controlled dependency management.

### IV. API Backward Compatibility

All API changes MUST be backward compatible. Add new optional fields only - never rename or remove existing fields. Changes to `api/` types require: edit types → `make generate` → commit all generated files + OpenAPI schema. Validation/defaulting logic added to `pkg/virt-api/webhooks/`. Document changes in `docs/` and update CRD schemas. Feature gating through config CRs preferred over environment variables.

**Rationale**: Protects existing KubeVirt deployments and ensures smooth upgrade paths for users managing production VM workloads.

### V. Integration-First Testing (NON-NEGOTIABLE)

All implementation MUST prioritize integration and end-to-end testing over unit tests: (1) Integration tests validating component interactions in real environments, (2) End-to-end tests verifying complete user workflows, (3) Unit tests for internal logic and edge cases. Use actual libvirt/QEMU/Kubernetes over mocks and simulators. Functional tests using Ginkgo framework are mandatory for all behavior changes.

**Rationale**: VM management demands high reliability in real environments - comprehensive integration testing prevents regression in critical virtualization workflows where mocked components hide real-world issues.

### VI. Security-First Development

Security MUST be designed into features from specification, not added as afterthought. Requirements: threat modeling during feature design, security review before implementation, principle of least privilege, input validation and sanitization. All features must maintain existing security boundaries and never expand attack surface without explicit justification.

**Rationale**: Virtualization introduces additional attack vectors and privilege escalation risks that require proactive security measures rather than reactive patches.

## Component Architecture

Components MUST respect established responsibilities and use choreography patterns: virt-operator (installation/cluster management), virt-controller (VM resource reconciliation), virt-handler (node-level operations), virt-launcher (VM process isolation), virt-api (validation/admission). Communication primarily through Kubernetes API resources and events. Direct communication permitted only for real-time operations (gRPC between virt-handler ↔ virt-launcher). Prefer reusing existing utilities in `pkg/` over creating new abstractions.

**Rationale**: Clear boundaries prevent component coupling while enabling necessary real-time communication for VM lifecycle operations.

## Development Workflow

All changes follow Make-based workflow with Bazel backend. Code generation and dependency management through established tooling prevents manual errors. Pull requests require functional test coverage and generated file consistency. Architecture changes require design proposals following community template. Performance-critical paths (launch flows, monitoring) follow established caching/timing patterns.

**Workflow Requirements**: Use `make` targets exclusively, never raw `bazel` commands. Validate via `make bazel-build-verify`. Complex changes start with community design proposal. Generated files committed alongside source changes.

## Constitutional Gates

Before implementing any feature, verify compliance with these mandatory checks:

- [ ] Follows the KubeVirt Razor principle (leverages existing K8s patterns)?
- [ ] Feature gate implemented and disabled by default?
- [ ] Integration and e2e tests planned before unit tests?
- [ ] Uses existing frameworks directly (Kubernetes, libvirt, QEMU)?
- [ ] Security considerations documented with threat model?
- [ ] Does not grant VM users capabilities beyond what Pods already have?
- [ ] Backward compatibility maintained for existing APIs?

**Enforcement**: All pull requests must demonstrate constitutional gate compliance before merge approval.

## Governance

This constitution supersedes all other development practices. All pull requests and code reviews MUST verify compliance with these principles and constitutional gates. Complexity additions require justification against simplicity principle. Changes affecting multiple components need architectural review. Use `.github/copilot-instructions.md` for runtime development guidance and established patterns.

**Amendment Process**: Constitution changes require community discussion (minimum 14 days), maintainer consensus, and documented rationale. Version increments follow semantic versioning: MAJOR for incompatible governance changes, MINOR for new principles/sections, PATCH for clarifications.

**Version**: 1.0.0 | **Ratified**: 2025-09-25 | **Last Amended**: 2025-09-25
