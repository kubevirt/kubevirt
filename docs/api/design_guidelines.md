# KubeVirt API Design Guidelines & Best Practices

## Table of Contents

1. [Foundations](#1-foundations)
2. [Resource Hierarchy & Lifecycle](#2-resource-hierarchy--lifecycle)
3. [Field & Type Design](#3-field--type-design)
4. [Validation](#4-validation)
5. [Hypervisor Translation Layer](#5-hypervisor-translation-layer)
6. [Status & Conditions](#6-status--conditions)
7. [References & Sources](#7-references--sources)

---

## How to Read This Document

These guidelines describe the **target state** for KubeVirt's API — the rules
that new API work must follow. The existing VM and VMI APIs predate many of
these rules and contain acknowledged technical debt: `uint32`/`uint64` fields,
historical `+kubebuilder:validation:Enum` markers on extensible fields,
near-zero CEL adoption, custom condition structs without per-condition
`observedGeneration`, etc. Where the document calls out known gaps explicitly, those
sections note it; the absence of such a note does not imply the existing API is
compliant.

New fields, new resources, and new API surface must follow these guidelines.
Retrofitting existing violations will be addressed through individual VEPs as
the API evolves.

---

## 1. Foundations

### 1.1 Backend Agnosticism

KubeVirt's public API must remain decoupled from any specific backend
implementation — both to future-proof the project and to allow for the potential
coexistence of alternative backends. This applies at every layer of the stack:
hypervisor engines, storage drivers, network plugins, and future virtualization
backends must not leak their names, native types, or internal semantics into the
public API surface. API fields, condition types, and validation rules should use
generalized, implementation-neutral nomenclature — `hypervisor`, `domain`,
`runtime` — rather than engine-specific terms. This applies at every abstraction
layer:

- **Hypervisor:** Avoid embedding `qemu`- or `libvirt`-specific concepts. Use
  `hypervisor`, `domain`, or `runtime` instead of surfacing engine internals.
  The temptation to break this rule is highest at the virtualization boundary, which is why the
  hypervisor is the most prominent example of this principle.
- **Storage backends:** Avoid encoding a specific CSI driver's behavior in a
  field name.
- **Network plugins:** Avoid exposing plugin-specific types or identifiers in
  public API fields.

**Justified exceptions:** A field whose explicit purpose is to identify or
select a specific backend may name that backend directly.
`HypervisorConfiguration.Name` (values: `kvm`, `hyperv-direct`) is the canonical
example — the field exists precisely to declare which hypervisor is present.
Similarly, expert or debug annotations such as `kubevirt.io/libvirt-log-filters`
are deliberate escape hatches that must name the engine to be meaningful. Each
such exception must be explicitly motivated in the field's API comment or the
relevant [VEP](https://github.com/kubevirt/enhancements).

### 1.2 Kubernetes Ecosystem Alignment (The KubeVirt Razor)

KubeVirt follows the architectural rule known as the **KubeVirt Razor**: *"If
something is useful for Pods, we should not implement it only for VMs."* The
intent is to prevent KubeVirt from building a parallel ecosystem that duplicates
mechanisms Kubernetes already provides. KubeVirt's API should feel like a
natural extension of Kubernetes, not a separate system running alongside it.

When a Kubernetes standard mechanism exists, it must be used in preference to a
KubeVirt-specific equivalent:

- **Types:** Use `metav1.Condition` for conditions, `resource.Quantity` for
  sizes and capacities, `metav1.Duration` for durations. Introducing a
  KubeVirt-specific parallel requires explicit justification.
- **Patterns:** Use owner references for garbage collection and standard
  `status.observedGeneration` for controller sync signaling. [Server-side
  apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) for
  field ownership is the Kubernetes-standard target pattern; KubeVirt adoption
  is in progress via [VEP
  #389](https://github.com/kubevirt/enhancements/pull/390). The cost of
  deviating from a standard pattern is paid by every consumer of the API.
- **Conventions:** Follow Kubernetes naming, serialization, and deprecation
  conventions (documented in the [Kubernetes API
  guidelines](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md)).
  Deviations create a learning tax for users and operators already familiar with
  the Kubernetes ecosystem.

KubeVirt does deviate from Kubernetes conventions in cases where virtualization
semantics have no Pod equivalent — the VM→VMI orchestration hierarchy, the
[sticky-spec late-defaulting](#22-identity-persistence-considerations) for
hardware identity, and the hypervisor abstraction layer are several examples.
Each such deviation must be explicitly motivated in the relevant API
documentation or [VEP](https://github.com/kubevirt/enhancements); the mere fact
that something is convenient or already implemented is not sufficient
justification. For example, the [sticky-spec
late-defaulting](#22-identity-persistence-considerations) is a deliberate
deviation from the Kubernetes spec/status split — the VM controller writes
generated hardware identity values back into `VM.spec`, the same resource the
user manages, rather than into `status`. See
[§2.2.1](#221-exception-late-defaulting-into-spec) for the justification.

---

## 2. Resource Hierarchy & Lifecycle

### 2.1 VM vs VMI Roles

#### 2.1.1 VirtualMachine (VM)

The `VirtualMachine` resource represents the persistent, long-lived management
object (analogous to a specialized Kubernetes `StatefulSet`). It acts as the
definitive source of truth for the user's permanent desired state — including
boot configuration, storage topologies, and high-level power state intent (e.g.,
`spec.runStrategy: Always` or `spec.runStrategy: Halted`). A `VirtualMachine`
resource outlives any single guest operating system boot cycle.

#### 2.1.2 VirtualMachineInstance (VMI)

The `VirtualMachineInstance` represents the transient execution span of a
virtual machine runtime. Unlike standard Kubernetes `Pods` which maintain a 1:1
relationship with an active container boundary, a VMI tracks a continuous
virtualization lifecycle. It must accommodate complex runtime orchestration,
such as mapping to multiple underlying infrastructure primitives concurrently
(e.g., managing both a source and a target Pod during Live Migration).

#### 2.1.3 Unidirectional Mutation

Users mutate the `VM` specification. The VM controller reconciles this intent by
managing the creation, deletion, or highly controlled mutation of the underlying
`VMI` specification. Direct user changes to `VMI` spec are rejected at the API
boundary — only KubeVirt's internal controllers are permitted to modify it.

---

### 2.2 Identity Persistence Considerations

#### 2.2.1 Exception: Late Defaulting into Spec

This pattern is a deliberate deviation from the Kubernetes spec/status split:
the VM controller writes generated hardware identity values back into `VM.spec`
— the user-managed resource — rather than into `status`. The motivation is
below.

In standard cloud-native environments, restarts provide a completely blank
slate. In virtualized infrastructure, however, specific hardware identifiers
dictate the permanent identity of a virtual machine and must survive across hard
reboots. For example, if a guest OS unexpectedly loses its firmware UUID, SMBIOS
serial number, or persistent MAC address between boot cycles, it can trigger
guest OS license deactivations, alter network interface ordering, or cause
catastrophic system failure.

#### 2.2.2 Persistence Patterns

Some fields, such as hardware identity fields, are complex, low-level details
that users should not be expected to manage. KubeVirt does not require them to
be set at `VM` creation time, and instead takes care of generating and
persisting these values transparently — using the patterns below — so that the
VM lifecycle remains stable without user intervention.

- **Pattern 1 — Admission Webhook Seeding (primary pattern):** For new VMs, the
  mutating admission webhook generates stable identity values and writes them
  directly into `VM.spec` at creation time. The firmware UUID is the canonical
  example: if not provided by the user, the webhook seeds it before the object
  is persisted. Anchoring in `VM.spec` ensures the value survives across the
  entire VM lifecycle, including reboots and re-creations of the underlying VMI.

- **Pattern 2 — VM Controller Back-fill (migration pattern):** When a new field
  requiring lifecycle-persistent identity is introduced, existing VMs in the
  cluster will not have it set. If KubeVirt can reproduce the value
  deterministically, the VM controller can compute it and patch it into
  `VM.spec` retroactively. The firmware UUID is again the canonical example: for
  pre-existing VMs that predate the admission webhook logic, the VM controller
  computes the UUID deterministically from the VM name and writes it into
  `VM.spec`.

- **Pattern 3 — Write-Back from Runtime (future consideration):** Some identity
  values are generated by the hypervisor and are challenging to reproduce
  deterministically within KubeVirt (e.g., vNIC MAC addresses or PCI device
  addressing, which must remain stable across reboots to avoid network
  configuration changes). This pattern is not yet implemented generically in
  KubeVirt.

When a new spec field is introduced, its design needs to take into account its
persistence requirements. If it is identified as requiring one, the required
lifecycle (VM vs VMI) must be considered upfront and an explicit persistence
strategy defined.

---

### 2.3 Ownership, Garbage Collection, and Finalizers

#### 2.3.1 Owner References

Owner references express the parent-child relationship between KubeVirt
resources and enable Kubernetes to cascade-delete children when their owner is
deleted. The canonical chain is:

```
VirtualMachine → VirtualMachineInstance → virt-launcher Pod
```

The VM controller creates the VMI with an owner reference pointing to the VM.
The VMI controller creates the virt-launcher Pod with an owner reference
pointing to the VMI. When the VM is deleted, Kubernetes cascades deletion to the
VMI; when the VMI is deleted, the Pod follows.

**Namespace constraint:** Owner references must be within the same namespace.
Cross-namespace owner references are explicitly disallowed by Kubernetes and
must never be introduced.

#### 2.3.2 Finalizers

Owner references alone do not guarantee ordering. Kubernetes background deletion
(the default) removes the owner immediately and cleans up dependents
asynchronously with no ordering guarantee. For KubeVirt, where a running
hypervisor domain must be stopped and pods must be cleaned up before their
parent objects disappear, this timing is not sufficient.

Finalizers are the correct tool for sequencing: they block the object's removal
from etcd until the controller explicitly clears them, giving the controller a
guaranteed window to complete cleanup.

KubeVirt uses two finalizers in the VM→VMI lifecycle:

- **`kubevirt.io/foregroundDeleteVirtualMachine`** on VMI — placed by the
  mutating admission webhook at VMI creation time; cleared by the VMI lifecycle
  controller after the VMI has completed its lifecycle and all associated pods
  have been deleted.
- **`kubevirt.io/virtualMachineControllerFinalize`** on VM — held by the VM
  controller; cleared after the VMI has been fully removed from etcd and the VM
  controller has completed its own cleanup.

The VM controller also places `kubevirt.io/virtualMachineControllerFinalize` on
the VMI itself. This is complementary to the owner reference: the owner
reference triggers cascade deletion, while the additional finalizer gives the VM
controller a guaranteed observation window — the VMI cannot be removed from etcd
until the controller explicitly clears the finalizer. The controller uses this
window to complete any work that requires the VMI's final state to still be
present in etcd. Currently, that means reading the VMI's final status
(conditions, generation, run strategy, volume requests, etc.) and writing it
onto the VM object before the VMI disappears. This follows the guideline in
§2.3.4: *"Add a finalizer to the parent if the parent controller must observe
the child's terminal state before completing its own cleanup."*

#### 2.3.3 Finalizer Naming

All KubeVirt finalizer names must be domain-qualified with the `kubevirt.io/`
prefix. Unqualified finalizer names are rejected by the Kubernetes API server.
The deprecated finalizer `foregroundDeleteVirtualMachine` (without the prefix)
is a historical violation of this rule; it is retained solely for migration
purposes and must not be used as a model for new finalizers.

#### 2.3.4 Guidelines for New Resources

When adding a new resource to the KubeVirt hierarchy:

- Set an owner reference on the child pointing to the parent to enable cascade
  GC.
- Add a finalizer to the child if the controller must perform cleanup (external
  state, domain teardown, pod deletion) before the child object is removed.
- Add a finalizer to the parent if the parent controller must observe the
  child's terminal state before completing its own cleanup.
- Do not rely on Kubernetes background GC timing for correctness. If ordering
  matters, use finalizers.
- Never add a new finalizer to an object after its `deletionTimestamp` has been
  set — Kubernetes rejects this.

---

## 3. Field & Type Design

### 3.1 Kubernetes-Native Data Primitives

#### 3.1.1 Supported Primitives

The public API surface (spanning both `spec` and `status`) should utilize
deterministic, Kubernetes-friendly data primitives: booleans, strings, signed
integers (`int32`/`int64`), and `resource.Quantity`. Unsigned integers
(`uint32`/`uint64`) exist in the API as technical debt and must not be used in
new fields; where they appear, the mitigations in §3.1.2 apply.

#### 3.1.2 Mapping Rules

- **Capacities & Sizes:** Any field representing size, memory allocation, or
  storage limits must use `resource.Quantity`. This abstracts hypervisor
  byte-level or sector-level requirements away from the end user into clean,
  cloud-native strings (e.g., expressing memory as `2Gi` or disk size as `40Gi`
  instead of raw integers).

- **Fractional or Decimal Weights:** Raw floating-point numbers are strictly
  forbidden within `spec` and strongly discouraged in `status`. Floats introduce
  non-deterministic serialization behavior across different client architectures
  (Go, Python, JavaScript), causing values to silently change when round-tripped
  through different implementations. Floats in `status` additionally cause
  spurious etcd writes when values oscillate in their least significant bits
  between reconciliation cycles. Any field requiring fractional granularity must
  be represented as an integer using milli-units (e.g., representing a
  performance weight of `1.5` as `1500m`); the conversion to the backend's
  native float happens at the translation layer (§5.1). Consider whether a
  frequently changing float value (e.g., byte counters, utilization ratios) is
  better reported via a metrics endpoint — each `status` update is persisted to
  etcd and is not designed for high-frequency telemetry.

  > **Note:** The prohibition is stronger for `spec` than for `status` because
  > `spec` is typically written by multiple actors across different languages and client
  > implementations — a float written by one client and read back by another may
  > silently change due to serialization differences, and that corrupted value
  > then drives controller logic. `status` is written only by controllers
  > (typically a single Go binary), so the round-trip corruption risk is lower;
  > the remaining concerns are efficiency (spurious etcd writes) and client-side
  > display inconsistency.

- **Oversized Unsigned Values:** Hypervisor configurations that naturally use
  `uint64` must be mapped to safe `string` types or verified `int64` fields
  accompanied by rigid OpenAPI upper/lower boundary validation tags to prevent
  overflow exploits.

---

### 3.2 Open String Set (Avoiding Enums)

#### 3.2.1 Kubernetes Compatibility

In accordance with [Kubernetes
`api_changes.md`](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api_changes.md),
the use of hard OpenAPI `enum` arrays for schema validation must be avoided on
fields representing extensible lists of choices (e.g., disk bus types, network
binding modes, hypervisor features, or power states). Hard enums break rolling
upgrades: if a newer version of KubeVirt introduces a new string token, any
older client or older control plane component running the historical schema will
completely reject the payload at the API boundary, causing serialization and
upgrade failures.

> **Note:** Several existing fields carry `+kubebuilder:validation:Enum` markers
> in violation of this rule. The clearest cases are
> `HypervisorConfiguration.Name` (`kvm;hyperv-direct`),
> `ExperimentalMigrationOptions.Compression` (`none;zstd`), and
> `TLSConfiguration.MinTLSVersion` (`VersionTLS10`–`VersionTLS13`) — all
> enumerate technology lists where new values are a natural extension path.
> Additional violations exist; search for `+kubebuilder:validation:Enum` in
> `staging/src/kubevirt.io/api/` to identify them. None of these must be used as
> a model for new fields.

#### 3.2.2 Schema Implementation

Fields that represent a selection from a set of string constants must be defined
as open `string` primitives in the OpenAPI v3 schema.

#### 3.2.3 Graceful Rejection

Instead of hard-rejecting unrecognized values at the cluster edge, validation
and error handling are deferred to the internal controller loops and node-level
daemons:

- **Serialization Safety:** If an older KubeVirt component processes a resource
  containing a newer, unrecognized string value, it must smoothly ingest, parse,
  and pass the object without failing or crashing.
- **Soft Failure Transmission:** The reconciling controller must observe the
  unrecognized value and gracefully pause execution *for that specific object
  only*. It must signal the error by updating the object's `status.conditions`,
  e.g. `Ready=False` with a clear programmatic code (e.g.,
  `Reason=UnsupportedValue`) and an explanatory human-readable message.

---

## 4. Validation

### 4.1 Structural Invariants

#### 4.1.1 Only Absolute Invariants

OpenAPI schema-level constraints — such as `Minimum`, `Maximum`, `MaxLength`,
`MaxItems`, and `Pattern` — must **only** be enforced if they represent an
absolute physical, mathematical, or architectural invariant that is
fundamentally incapable of changing in the future. Arbitrary boundaries
established for business logic or aesthetic neatness artificially restrict
system growth and require disruptive, coordinated CRD schema rollouts to alter.

#### 4.1.2 Legitimate Structural Invariants

Schema constraints are strictly reserved for unyielding specifications,
including:

- **Mathematical Primitives:** `// +kubebuilder:validation:Minimum=1` for
  virtual CPU counts (a runtime environment cannot execute on zero or negative
  processing cores).
- **Network & Protocol Specs:** `// +kubebuilder:validation:Maximum=65535` for
  standard TCP/UDP port allocations.
- **Fixed Cryptographic/IEEE Formats:** `// +kubebuilder:validation:Pattern=...`
  for MAC addresses or standard UUID structures where the string layout is
  governed by immutable external RFC standards.

### 4.2 CEL Validation

#### 4.2.1 Prefer CEL over Webhooks

Cross-field validation — where the validity of one configuration parameter
depends directly on the state of another — must prioritize high-performance
**CEL (Common Expression Language)** rules via the
`+kubebuilder:validation:XValidation`
[marker](https://book.kubebuilder.io/reference/markers/crd-validation), which
instructs Kubebuilder to generate the corresponding CEL [validation
rules](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules)
in the CRD schema. Writing or extending imperative validating admission webhooks
should be treated as a secondary fallback, reserved for validation logic CEL
genuinely cannot express. It is worth warning that validating admission
webhooks that base their decisions on external state are subject to a
potential TOCTOU (time-of-check to time-of-use) race, and special attention is
needed when using them.

> **Note:** KubeVirt's existing validation logic lives predominantly in
> imperative admission webhooks, which predate CEL and this requirement. These must not
> be treated as a model for new validation work; new cross-field constraints
> must use CEL. Search for `+kubebuilder:validation:XValidation` in
> `staging/src/kubevirt.io/api/` to identify existing CEL adoption.

#### 4.2.2 CEL Examples

CEL rules allow immediate server-side rejection at the API boundary before etcd
persistence:

- **Exactly One:** Ensuring that a virtual disk drive slot does not
  simultaneously define conflicting storage backends (illustrative — the real
  `Disk` struct has many more backend types; a production rule must account for
  all of them). In CEL, `has(a) != has(b)` is a boolean XOR — it evaluates to
  `true` when exactly one of the two fields is present, and `false` when both or
  neither are present:
  `// +kubebuilder:validation:XValidation:rule="has(self.containerDisk) != has(self.persistentVolumeClaim)",message="A disk slot must define exactly one backend provider (ContainerDisk or PVC)"`

- **Dependency Enforcement:** Verifying that a feature flag is only set when its
  prerequisite is also configured. For example, `isolateEmulatorThread` requires
  `dedicatedCPUPlacement`:

  ```yaml
  # Valid
  spec:
    domain:
      cpu:
        dedicatedCPUPlacement: true
        isolateEmulatorThread: true

  # Rejected: isolation requires dedicated CPU placement
  spec:
    domain:
      cpu:
        dedicatedCPUPlacement: false
        isolateEmulatorThread: true
  ```

  ```go
  // +kubebuilder:validation:XValidation:rule="!has(self.isolateEmulatorThread) || !self.isolateEmulatorThread || self.dedicatedCPUPlacement",message="isolateEmulatorThread requires dedicatedCPUPlacement"
  ```

#### 4.2.3 Upgrade Safety

Because Custom Resource Definitions (CRDs) containing new CEL rules are updated
in the cluster *before* the corresponding controller binaries are fully rolled
out, developers must observe strict safety rules when introducing rules:

- **Safe CEL Usage (New Fields & Pure Invariants):** It is safe to apply CEL
  constraints to newly introduced API fields or to validate absolute, unchanging
  architectural invariants. Older active clients will not be interacting with
  these new properties and are thus insulated from validation triggers.

- **Unsafe CEL Usage (Retroactive Restrictions):** Developers must not introduce
  a CEL rule that retroactively tightens validation criteria on an *existing,
  older field* if an active legacy client or third-party operator could still be
  writing payloads that would violate the new rule — doing so breaks cluster
  state continuity and triggers abrupt, unexpected rejections during the rolling
  upgrade window. **Exception:** Kubernetes 1.28+ supports a CEL evaluation mode
  where a constraint is only checked when the field's value actually changes;
  existing objects with unchanged values continue to pass regardless of whether
  they satisfy the new rule. When the minimum supported cluster version allows
  it, this mode is the safer way to tighten validation on an existing field.

- **Unconditional Schema Change with Gated Population:** A field may be added to
  the API schema unconditionally — always present in the Go type and CRD — while
  its population by controllers is gated behind a feature flag. The pattern is
  illustrated by the planned migration of `VirtualMachineCondition` to
  `metav1.Condition` ([VEP
  #376](https://github.com/kubevirt/enhancements/pull/377)): the struct change
  would be unconditional, but controllers would only populate
  `ObservedGeneration` once a corresponding feature gate is enabled. When the
  gate is off, the field stays at its zero value, which `omitempty` serializes
  as absent.

  This pattern carries a hard design constraint: **the zero or absent value of
  the field must be a safe no-op for every consumer.** If any consumer acts on
  the zero value as though it were meaningful data, it will break before the
  gate is ever enabled — silently, with no error. For example, a consumer
  checking condition freshness with
  `condition.ObservedGeneration < vm.Generation` would evaluate `0 < 5 = true`
  and incorrectly treat every condition as stale on every VM, regardless of
  actual state. The safe consumer pattern is to treat zero/absent as
  "information not yet available" and skip the check entirely:

  ```go
  if condition.ObservedGeneration > 0 && condition.ObservedGeneration < vm.Generation {
      // treat condition as stale
  }
  ```

  To illustrate why zero-safety matters beyond initial enablement: if a feature
  gate is later disabled — for example, while investigating a regression or
  waiting for a hotfix — controllers will stop populating the field and
  consumers will again observe zero/absent. A consumer written without the `> 0`
  guard would misfire on every VM. Write the guard from day one, before the gate
  is ever enabled.

  **Note:** As of this writing, `VirtualMachineCondition` and
  `VirtualMachineInstanceCondition` are still custom structs without
  per-condition `ObservedGeneration`. [VEP
  #376](https://github.com/kubevirt/enhancements/pull/377) will introduce this
  field; this section describes the pattern to apply when it does.

For feature gate graduation criteria, alpha/beta/GA definitions, and the full
deprecation process, see
[feature-lifecycle.md](https://github.com/kubevirt/enhancements/blob/main/docs/feature-lifecycle.md).

---

## 5. Hypervisor Translation Layer

The upper layers of the KubeVirt control plane must remain entirely ignorant of
hypervisor-native configuration types, quirks, and raw data models. This section
specifies the concrete translation boundary at which the backend agnosticism
principle (§1.1) is enforced in practice.

### 5.1 Write Path: Spec → Hypervisor

The translation of safe, standardized Kubernetes API primitives into raw
hypervisor-specific types (e.g., converting a `resource.Quantity` into a raw
bit-mask, byte count, or string configuration block, or a milli-unit integer
into a floating-point number) must be deferred to the absolute edge of
execution. This translation must occur at the node level (e.g., within the
localized runtime agent container), immediately before the hypervisor's native
execution engine API or configuration writer is invoked.

### 5.2 Read Path: Hypervisor → Status

#### 5.2.1 Translate Back to Native Types

The abstraction layer must work symmetrically in reverse. Raw hardware metrics,
execution states, and telemetry data read directly from the hypervisor engine
will naturally contain floats, raw bytes, or non-standard types.

#### 5.2.2 Translate at the Node

These values must be translated *back* into safe, standardized Kubernetes-native
primitives at the node level immediately upon collection by the node daemon. The
data published to the `VMI.status` block must strictly mirror the data types,
granularities, and structures established in the `spec`.

#### 5.2.3 Hypervisor as Source of Truth

Virt-handler must be capable of querying the active hypervisor provider on the
host to rebuild the VMI `status` completely from scratch. The running hypervisor
instance is always the absolute source of truth — etcd, remote controllers, and
cached state are all potentially stale or racy relative to the current hardware
reality. The control plane must be capable of perfectly reconstructing the
object's `status` purely from that observed physical reality without relying on
cached historical flags.

## 6. Status & Conditions

### 6.1 Use Conditions, not Linear State-Machines

#### 6.1.1 Condition Types

Condition types inside the `status.conditions` array must exclusively represent
high-level, long-running, and orthogonal facets of a resource's readiness,
capability, or health (e.g., `Ready`, `LiveMigratable`, `Paused`).

#### 6.1.2 Avoid Phase Fields

Do not introduce
[`Phase`](https://github.com/kubernetes/community/blob/03ffbd014976893b33f750b469fd6843c4070275/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
string fields or linear state-machine tracking in new API resources. A single
`Phase` string forces a multi-faceted distributed system into an imperative,
edge-triggered state timeline that is prone to deadlocks and is difficult to
extend without breaking consumers.

- **The Alternative:** If sub-component states or progressive configuration
  steps must be exposed, they should be tracked via distinct, decoupled boolean
  conditions or well-scoped programmatic reasons within an existing top-level
  condition, rather than a centralized linear state machine.

**Historical note — `vmi.Status.Phase`:** `VirtualMachineInstance` carries a
`Phase` field (`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`,
`Failed`, `WaitingForSync`, `Unknown`) for the same historical reason Kubernetes
`Pod` does: conditions were adopted as the preferred pattern later, after the
problems with phase-based state machines were well understood. `WaitingForSync`
was added later to support decentralized live migration, and is a concrete
example of how phase-based models accrete values over time rather than composing
cleanly. Because `vmi.Status.Phase` is a stable `v1` field with heavy consumer
dependency, there are no plans to deprecate it. However, new consumers and new
API designs are strongly encouraged to prefer conditions over phase fields, as
conditions are orthogonal, independently observable, and more precisely reflect
the multi-controller reality of KubeVirt.

---

### 6.2 `Reason` vs `Message`

#### 6.2.1 The `Reason` Field

The `Reason` field is a machine-readable, single-word `CamelCase` token. Its
primary design audience is **other controllers or automated operators** that
must programmatically branch, switch, or execute recovery actions based on the
reported state.

- **Abstract Categorization:** `Reason` codes should be structured abstractly
  enough to group comparable operational states under broad, predictable
  categories (e.g., `ResourceAllocationError`, `NetworkBindingFailed`). This
  allows consuming controllers to make safe, generalized automation decisions.
- **Specialized Exception:** Highly focused, narrow `Reason` codes are
  acceptable *only* if a consuming controller needs to execute a distinctly
  different, automated remediation pathway for that specific sub-fault.
- **No Multi-Reason Packing:** A condition must never attempt to combine
  multiple distinct failures into a single programmatic reason string (e.g.,
  avoid `StorageAndNetworkNotReady`).

#### 6.2.2 The `Message` Field

The `Message` field is a dynamic, human-readable string intended for end-user
triage, CLI output (`kubectl describe`), and log analysis. All granular
multi-faceted error strings, raw hardware stack traces, nested error codes, or
contextual virtualization parameters must be offloaded entirely to the `Message`
field.

---

### 6.3 Level-Based Status

"Level-based" means the controller acts on the *current* observed state of the
world (the level), rather than reacting to state-change events (edge-triggered).
This is the design pattern recommended by the [Kubernetes API
Conventions](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md).

#### 6.3.1 Recomputation from Scratch

The `status` block must strictly reflect the current physical reality observed
during the active execution loop, rather than being computed as a delta,
increment, or state transition from the *previous* reconciliation cycle's
status data — see [§5.2.3](#523-hypervisor-as-source-of-truth) for how this
applies to the hypervisor translation layer specifically.

#### 6.3.2 Anti-Pattern: Historical Memory

Relying on cached values from previous status loops to determine the next status
transforms the controller into an unstable, edge-triggered state machine. If a
controller container crashes mid-reconciliation, or etcd experiences a temporary
network partition, this historical memory is permanently lost, leaving the
platform in an unrecoverable or inconsistent state. Any deviation from pure
recomputation is an architectural anomaly that requires extraordinary
justification and bulletproof fallback recovery paths.

---

### 6.4 Abstract Status Fields

#### 6.4.1 Vendor-Neutral Schemas

Data exposed inside the `status` block must be cleanly abstracted away from
hypervisor-specific or provider-specific primitives. This shields the public API
from backend implementation leaks and preserves long-term extensibility.

#### 6.4.2 Abstract Wrapping

Do not leak low-level execution engine return codes, hardware-specific layout
bitmasks, or raw virtualization driver structures directly into status fields.
Instead, wrap these details in generalized, high-level structures or abstract
classifications. This ensures that if alternative hypervisor backends or
virtualization engines are introduced to KubeVirt in the future, the public
status API remains entirely unchanged, and backend-specific differences remain
isolated within the node-agent layer.

---

### 6.5 `observedGeneration`

#### 6.5.1 Producer Responsibilities

To guarantee transparency across distributed asynchronous components, every
top-level status block must maintain an `observedGeneration` field. Whenever a
controller evaluates a resource and writes a status update, it must snapshot the
`metadata.generation` it evaluated and save it into `status.observedGeneration`.
This is a current hard requirement for all new resources.

Per-condition `observedGeneration` — where each individual entry in the
`conditions` array carries its own generation snapshot — is the target standard,
aligned with [VEP #376](https://github.com/kubevirt/enhancements/pull/377) and
the migration of KubeVirt conditions to `metav1.Condition`. Existing resources
(`VirtualMachine`, `VirtualMachineInstance`) do not yet have per-condition
`observedGeneration`; new resources should design for it from the start by
adopting `metav1.Condition` rather than a custom condition struct.

**Note on current state:** `VirtualMachineStatus` carries a top-level
`observedGeneration` field. `VirtualMachineInstanceStatus` does not — this is a
known gap. Consumers must not assume `vmi.status.observedGeneration` is present.

#### 6.5.2 Consumer Responsibilities

Any sub-component, webhook, or external operator consuming a KubeVirt resource
status is not strictly forced to block execution until the status stabilizes
(`metadata.generation == status.observedGeneration`). Instead, consumers must
execute a conscious, contextual design decision:

- **Synchronized/High-Stakes Logic:** If the consuming logic executes an
  irreversible or destructive action (e.g., triggering a hard resource eviction
  or declaring a terminal failure state), it should check for a generation
  mismatch. If `metadata.generation > status.observedGeneration`, it should
  recognize the status as stale, defer execution, and wait for the reconciling
  controller to catch up.
- **Speculative/Low-Stakes Logic:** If the consumer is executing non-destructive
  monitoring, speculative routing, or path optimization, it may choose to act
  immediately on the current data, fully aware of the active reconciliation gap.

---

### 6.6 Status Ownership

#### 6.6.1 Single Controller Owner

To eliminate etcd write conflicts, optimistic concurrency failures, and infinite
controller update cascades, the status should be owned by exactly **one**
controller binary. For new resources, this must be enforced structurally by
declaring a [`/status`
subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource)
in the CRD. With a status subresource in place, the API server silently discards
status fields included in a spec (main-resource) write — status can only be
updated via the `/status` subresource endpoint — preventing spec writes from
accidentally clobbering observed state. Designing single-ownership in from the
start is always preferable to relying on convention.

#### 6.6.2 VMI: Dual-Writer Debt

`VirtualMachineInstance` sits at the boundary between two distinct control
planes: the VMI controller (cluster-level orchestration) and `virt-handler`
(node-level domain state). Both currently write to `vmi.Status` as part of their
core responsibilities, and VMI does not have a `/status` subresource. This is a
known architectural debt that cannot be resolved without significant rework. It
is not a pattern to emulate in new resources; it is a debt inherited from the
foundational design of the platform.

#### 6.6.3 Field-Group Isolation

Where dual-write cannot be avoided, the primary mitigation is strict field-group
isolation: each component writes only to fields it exclusively owns, and those
ownership boundaries must be honored by convention and reflected in the schema
structure where possible (e.g., via [Server-Side
Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) field
ownership). The controllers operate within non-overlapping sub-structures or
independent field paths. This reduces the blast radius of a write conflict to a
single field group rather than the entire status block.

#### 6.6.4 Careful Merge (Last Resort)

Some fields cannot be partitioned cleanly because both components contribute
distinct, complementary values to the same field —
`vmi.Status.Interfaces[n].InfoSource` is the canonical example, where the VMI
controller and `virt-handler` each populate it with information the other does
not have. When this cannot be avoided, the **careful merge pattern** applies:

- **Explicit ownership documentation:** The field's API comment must state which
  component owns which portion of the value. This is a hard requirement —
  undocumented shared ownership is indistinguishable from a bug.
- **Read-before-write with preservation:** Each component reads the current
  field value, calculates only its own portion, and merges the result back
  without touching the portion owned by the other component. Neither component
  may assume the field is empty on write.
- **Single batched patch:** All status changes a component needs to make in a
  given reconcile must be collected and issued as a single patch operation.
  Issuing multiple sequential patches increases the window for the other
  component to interleave a write between them, compounding the race. Full
  object `.Update()` calls are prohibited — a full update from a stale in-memory
  copy will silently overwrite whatever the other component last wrote.

The careful merge pattern is inherently racy and error-prone. It must be treated
as a last resort, applied only when partitioning the field is not possible, and
subjected to heightened scrutiny in design review and code review.

---

### 6.7 Condition Design Guidelines

#### 6.7.1 Adding a New Condition

Before introducing a new condition type, apply these considerations in order:

1. **Can an existing condition cover it?** Check whether the scenario is
    already expressible via an existing condition's `Reason` or `Message`
    fields. Adding a new condition type is only justified when the new facet is
    independently observable, long-lived, and meaningful to automation — not
    merely a sub-state of something already tracked.

2. **Should the condition be more generic?** If a new condition is genuinely
    needed, consider whether a broader formulation would serve the same purpose
    while leaving room for future extension. A condition scoped to one specific
    operation risks proliferation: `HotVCPUChange`, `HotMemoryChange`, and
    `VolumesChange` are each specific to a single hotplug operation; a more
    generic `ResourceChange` might have unified them. Conversely, a condition so
    broad it means "something is happening" is not actionable by automation. The
    right level is the one at which a consuming controller or operator can make
    a meaningful, independent decision.

3. **Define absence semantics explicitly.** `Unknown` — and an absent
    condition, which consumers should treat as `Unknown` — legitimately covers
    two interpretations: the state has not yet been determined, or the condition
    is not relevant to this resource or its current configuration. When
    introducing a new condition, document which of these applies.

#### 6.7.2 Naming Conventions

New condition types should follow the established patterns:

- **Adjective or past-participle for persistent state:** `Ready`, `Paused`,
  `LiveMigratable`, `StorageLiveMigratable`
- **Noun phrase for a pending requirement:** `RestartRequired`,
  `MigrationRequired`, `ManualRecoveryRequired`
- **Noun phrase for an in-progress change:** `HotVCPUChange`, `HotMemoryChange`,
  `VolumesChange`

Avoid negative forms (e.g., `AgentVersionNotSupported` is a legacy outlier —
prefer the positive framing with `status: "False"` and an appropriate `Reason`
to signal the negative case).

#### 6.7.3 Existing Condition Types

The source of truth for current condition types is
[`staging/src/kubevirt.io/api/core/v1/types.go`](https://github.com/kubevirt/kubevirt/blob/v1.8.4/staging/src/kubevirt.io/api/core/v1/types.go).

---

## 7. References & Sources

This document synthesizes rules and patterns from the sources below. Inline
citations throughout link to specific sections where applicable; this list
provides the complete set of sources consulted.

> **Codebase alignment note:** The implementation patterns, examples, and known
> architectural constraints in this document are also grounded in the [KubeVirt
> codebase](https://github.com/kubevirt/kubevirt) as it existed at the time of
> writing. The codebase evolves independently of this document. When assessing
> whether this document remains accurate — whether by a human reviewer or an AI
> agent — the current state of the codebase should be consulted alongside the
> upstream sources listed below to verify that described patterns have not been
> superseded.

### Kubernetes

| Source | Sections influenced |
|----|----|
| [Kubernetes API Conventions (`api-conventions.md`)](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api-conventions.md) | Foundational structure rules: spec/status separation, object anatomy, naming, serialization, conditions schema — §1.2, §3.1, §3.2, §6 |
| [Kubernetes API Changes Guide (`api_changes.md`)](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api_changes.md) | Rules for evolving APIs safely: new fields must be optional, backward-compatible defaulting, hub-and-spoke versioning — §3.2, §4.2.3 |
| [Kubernetes Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/) | Minimum support lifespans by graduation track (alpha/beta/GA), co-existence and warning header requirements — §4.2.3 |
| [Kubernetes API Concepts](https://kubernetes.io/docs/reference/using-api/api-concepts/) | Operational standards: server-side apply field ownership, pagination, dry-run — §1.2 |
| [Kubernetes CRD Validation Rules](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules) | CEL validation rule mechanics and CRD schema enforcement — §4.2 |
| [Kubernetes CRD Status Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource) | Structural enforcement of single-owner status writes — §6.6 |
| [Server-Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) | Field ownership model for multi-actor reconciliation — §1.2 |
| [Gateway API Design Guide](https://gateway-api.sigs.k8s.io/guides/api-design/) | Real-world CRD design patterns: conflict resolution, naming conventions (use plural nouns for array fields, singular verbs for action fields) — §6.7 |

### Tooling

| Source | Sections influenced |
|----|----|
| [Kubebuilder Book](https://book.kubebuilder.io/) | Practical reference for generating OpenAPI v3 schemas from Go source annotations — §4 |
| [Kubebuilder CRD Validation Markers](https://book.kubebuilder.io/reference/markers/crd-validation) | `+kubebuilder:validation:XValidation` and structural constraint markers — §4.1, §4.2 |

### KubeVirt-Specific

| Source | Sections influenced |
|----|----|
| [KubeVirt Enhancement Proposals (VEPs)](https://github.com/kubevirt/enhancements) | Process and rationale for significant API changes; every justified deviation from upstream conventions requires a VEP — §1.1, §1.2, §2.1 |
| [VEP #376: Migrate conditions to `metav1.Condition`](https://github.com/kubevirt/enhancements/pull/377) | Target standard for per-condition `observedGeneration`; gated population pattern — §4.2.3, §6.5 |
| [KubeVirt Feature Lifecycle](https://github.com/kubevirt/enhancements/blob/main/docs/feature-lifecycle.md) | Feature gate graduation criteria, alpha/beta/GA definitions, full deprecation process — §4.2.3 |
| [KubeVirt API types (`staging/src/kubevirt.io/api/core/v1/types.go`)](https://github.com/kubevirt/kubevirt/blob/v1.8.4/staging/src/kubevirt.io/api/core/v1/types.go) | Canonical source of existing VM and VMI condition types — §6.7.3 |
