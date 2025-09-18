# KubeVirt Development Constitution

**Version**: 1.0
**Status**: Draft  

## Preamble

This constitution establishes the core principles that guide all feature development within the KubeVirt project. These principles ensure consistency, quality, and architectural integrity while remaining simple and actionable.

**KubeVirt's Mission**: To provide virtualization capabilities for Kubernetes through Custom Resource Definitions, enabling VM management using native Kubernetes APIs and patterns.

**Core Values**:
- **Native Integration**: VMs should feel like native Kubernetes workloads  
- **Service-Oriented Architecture**: Choreographed components that act independently based on observed state
- **Quality First**: Integration and end-to-end testing prioritized over unit tests
- **Community-Driven**: Inclusive, transparent development processes

## Article I: The KubeVirt Razor

### The Razor Principle
**"If something is useful for Pods, we should not implement it only for VMs"**

This is the fundamental decision-making framework that guides all feature development. Every feature **MUST** leverage existing Kubernetes patterns and APIs rather than creating VM-specific alternatives.

**Why**: This ensures consistent user experience, reduced cognitive load, natural integration with existing tooling, and reduced maintenance burden.

### Native Workload Parity
Features **MUST NOT** grant VM users capabilities beyond what Kubernetes already provides for container workloads. Installing KubeVirt must never grant users permissions they don't already have for native workloads.

## Article II: Feature Gates and Quality

### Alpha-First Development
All new features **MUST** begin life disabled by default behind a feature gate. No exceptions.

**States**: Alpha (disabled) → Beta (enabled in dev) → GA (enabled in prod) → Deprecated → Discontinued

*Note: This matches KubeVirt's actual feature gate implementation in `/pkg/virt-config/featuregate/`*

### Integration-First Testing
**Non-negotiable**: All implementation **MUST** prioritize integration and end-to-end testing over unit tests:
1. **Integration tests** validating component interactions in real environments
2. **End-to-end tests** verifying complete user workflows  
3. **Unit tests** for internal logic and edge cases

**Real Environment Priority**: Use actual libvirt/QEMU/Kubernetes over mocks and simulators

*Note: This reflects KubeVirt's actual testing philosophy as documented in `/tests/README.md`*

## Article III: Component Architecture

### Service Boundaries
Components **MUST** respect established responsibilities and use choreography patterns:
- **virt-operator**: Installation and cluster-wide management
- **virt-controller**: VM resource reconciliation and Pod lifecycle management
- **virt-handler**: Node-level VM operations and libvirt domain management  
- **virt-launcher**: Individual VM process isolation and management
- **virt-api**: API validation, defaulting, and admission webhooks

*Note: These match the actual component responsibilities documented in `/docs/components.md`*

### Communication Patterns
**Primary**: Kubernetes API resources, events, and controller watch/reconcile patterns

**Permitted Direct Communication** (between KubeVirt components only):
- gRPC APIs for real-time VM lifecycle operations (virt-handler ↔ virt-launcher)
- HTTP/REST APIs for essential operations that cannot be asynchronous

**Prohibited**: Shared filesystems, direct database access, unauthenticated communication

*Note: KubeVirt actually uses gRPC between virt-handler and virt-launcher as seen in `/pkg/handler-launcher-com/`*

## Article IV: Simplicity

### Minimal Viable Implementation
Features **MUST** start with the simplest implementation that satisfies user stories. Avoid:
- Premature optimization
- Speculative features
- Complex abstractions without proven need

### Trust the Frameworks
Use KubeVirt, Kubernetes, libvirt, and QEMU features directly rather than creating wrapper layers. This matches KubeVirt's approach of extending rather than replacing Kubernetes primitives.

**Integration Strategy** (from actual KubeVirt architecture):
1. **Extension over Replacement**: Extend existing Kubernetes APIs rather than replacing them
2. **Composition over Custom**: Compose existing primitives rather than building custom solutions  
3. **Standards over Proprietary**: Follow established standards (CNI, CSI, CRI) over proprietary interfaces
4. **Upstream over Fork**: Contribute improvements upstream rather than maintaining forks

*Note: This reflects principles documented in `/docs/architecture.md`*

## Article V: Security

### Security-First
Security **MUST** be designed into features from specification, not added as afterthought:
- Threat modeling during feature design
- Security review before implementation
- Principle of least privilege
- Input validation and sanitization

## Article VI: Enforcement

### Constitutional Gates
Before implementing any feature, verify:
- [ ] Follows the KubeVirt Razor principle?
- [ ] Feature gate implemented and disabled by default?
- [ ] Integration and e2e tests planned before unit tests?
- [ ] Uses existing frameworks directly (Kubernetes, libvirt, QEMU)?
- [ ] Security considerations documented?
- [ ] Does not grant VM users capabilities beyond what Pods already have?

*Note: These gates reflect actual KubeVirt development practices and architectural requirements*

### Amendment Process
This constitution may only be modified through:
- Community discussion (minimum 14 days)
- Maintainer consensus
- Documented rationale and impact analysis

---

## Philosophy

This constitution prioritizes **actionable simplicity** over comprehensive coverage. It establishes the minimum viable governance needed to maintain KubeVirt's architectural integrity while allowing teams to move quickly and make good decisions.

Additional guidance, patterns, and detailed procedures should be developed as separate documents that build upon these foundational principles.

**Remember**: When in doubt, ask "Does this make VMs feel more like native Kubernetes workloads?" If yes, you're probably on the right track.

*This constitution has been validated against actual KubeVirt project practices, codebase patterns, and architectural documentation to ensure accuracy and relevance.*
