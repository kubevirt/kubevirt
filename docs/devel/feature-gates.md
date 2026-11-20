# Feature Gates

Feature gates guard new or experimental functionality behind named flags that
cluster administrators can toggle in the KubeVirt CR. Every non-trivial feature
that could affect cluster stability should be introduced behind a feature gate.

For the deprecation and removal policy, see [deprecation.md](../deprecation.md).

## Lifecycle States

Feature gates progress through a defined lifecycle. The source of truth for
state semantics is
[`pkg/virt-config/featuregate/feature-gates.go`](../../pkg/virt-config/featuregate/feature-gates.go).

| State | Default | User enables via | User disables via | When to use |
|-------|---------|------------------|-------------------|-------------|
| **Alpha** | Off | `featureGates[]` | `disabledFeatureGates[]` | New feature under experimentation |
| **Beta** | On | — | `disabledFeatureGates[]` | Feature considered stable enough for broad testing |
| **GA** | On (always) | — | Cannot be disabled | Feature is stable and permanent |
| **Deprecated** | Off | `featureGates[]` | `disabledFeatureGates[]` | Feature gate is being phased out (feature itself is always on) |
| **Discontinued** | Off | Cannot be enabled | — | Feature gate and/or feature has been removed |

These fields live in the KubeVirt CR at
`spec.configuration.developerConfiguration.featureGates` and
`spec.configuration.developerConfiguration.disabledFeatureGates`.

## Package Structure

Feature gates are organized into SIG-owned sub-packages:

```
pkg/virt-config/featuregate/
├── feature-gates.go          # Core types: State, FeatureGate, ConfigReader, IsEnabled()
├── validator.go              # Discontinued gate validation for admission webhooks
├── compute/                  # sig-compute gates
│   ├── feature_gates.go      # ComputeFeatureGates struct
│   ├── template.go           # One file per gate
│   └── ...
├── network/                  # sig-network gates
│   ├── feature_gates.go      # NetworkFeatureGates struct
│   └── ...
├── storage/                  # sig-storage gates
│   ├── feature_gates.go      # StorageFeatureGates struct
│   └── ...
└── legacy/                   # Gates not yet assigned to a SIG
    ├── feature_gates.go      # LegacyFeatureGates struct
    └── ...
```

Each SIG package contains:
- A **grouping struct** (e.g. `ComputeFeatureGates`) that embeds
  `featuregate.ConfigReader`.
- One **`.go` file per gate** with a name constant, `init()` registration, and
  an `Enabled()` method on the grouping struct.

`ClusterConfig` in `pkg/virt-config/configuration.go` embeds all four grouping
structs, so every gate's `Enabled()` method is promoted directly onto
`ClusterConfig` with no additional wiring.

## Adding a New Feature Gate

This walkthrough adds a hypothetical `MyNewFeature` gate to sig-compute.

### 1. Create the gate file

Create `pkg/virt-config/featuregate/compute/my_new_feature.go`:

```go
package compute

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: sig-compute / @your-github-handle
// Alpha: v1.10.0
//
// MyNewFeature enables the new feature.
const MyNewFeature = "MyNewFeature"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{
		Name:  MyNewFeature,
		State: featuregate.Alpha,
	})
}

// MyNewFeatureEnabled returns true when the MyNewFeature feature gate is enabled.
func (g ComputeFeatureGates) MyNewFeatureEnabled() bool {
	return featuregate.GateEnabled(MyNewFeature, g.ConfigReader)
}
```

Key points:
- The `init()` function self-registers the gate into the global registry when
  the package is imported.
- The `Owner` comment identifies the responsible SIG and individual.
- The version comment tracks when the gate entered each state.

### 2. Update BUILD.bazel

Add the new file to `pkg/virt-config/featuregate/compute/BUILD.bazel`:

```python
go_library(
    name = "go_default_library",
    srcs = [
        ...
        "my_new_feature.go",
        ...
    ],
    ...
)
```

### 3. Use the gate in product code

The gate method is available directly on `ClusterConfig`:

```go
if config.MyNewFeatureEnabled() {
    // feature-gated code path
}
```

No additional imports beyond your SIG package are needed — the method is
promoted through the embedded `ComputeFeatureGates` struct.

### 4. Enable the gate in tests

Use the test config helper with the exported constant:

```go
import "kubevirt.io/kubevirt/pkg/virt-config/featuregate/compute"

config.EnableFeatureGate(compute.MyNewFeature)
```

### 5. Verify registration

Run the feature gate report to confirm the gate appears:

```bash
make feature-gate-report
```

This prints a JSON summary of all active (non-GA, non-Discontinued) gates.

## Progressing a Feature Gate

### Alpha → Beta

The feature is considered stable enough for broad testing. It becomes enabled by
default — users who want to opt out must add it to `disabledFeatureGates[]`.

1. Change the state:
   ```go
   State: featuregate.Beta,
   ```
2. Add a version comment:
   ```go
   // Alpha: v1.10.0
   // Beta: v1.12.0
   ```

See [`compute/template.go`](../../pkg/virt-config/featuregate/compute/template.go)
for a real example (Alpha in v1.8.0, Beta in v1.9.0).

### Beta → GA

The feature is stable and permanent. The gate is always enabled and cannot be
disabled, even if listed in `disabledFeatureGates[]`.

1. Change the state:
   ```go
   State: featuregate.GA,
   ```
2. Add a version comment:
   ```go
   // Beta: v1.12.0
   // GA: v1.14.0
   ```
3. Clean up tests that explicitly enable/disable this gate — those calls are
   now no-ops.

See [`compute/secure_execution.go`](../../pkg/virt-config/featuregate/compute/secure_execution.go)
for a real example (Alpha v1.6.0 → Beta v1.7.0 → GA v1.9.0).

### GA → Deprecated

Use this when the feature gate *string* is no longer needed (the feature itself
stays permanently on). This warns users who still list the gate in their CR.

1. Change the state and add a message:
   ```go
   State:   featuregate.Deprecated,
   Message: "MyNewFeature has been deprecated since v1.16.0",
   ```
2. Add a version comment:
   ```go
   // GA: v1.14.0
   // Deprecated: v1.16.0
   ```
3. Send a notification to the kubevirt-dev mailing list
   (kubevirt-dev@googlegroups.com) as described in
   [deprecation.md](../deprecation.md).

See [`compute/multi_architecture.go`](../../pkg/virt-config/featuregate/compute/multi_architecture.go)
for a real example.

### Deprecated → Discontinued

The gate and optionally its associated API surface are fully removed.

1. Change the state:
   ```go
   State: featuregate.Discontinued,
   ```
2. If the feature had API fields that are being removed, add a `VmiSpecUsed`
   function so the admission webhook rejects VMI specs still referencing them:
   ```go
   func myNewFeatureApiUsed(spec *v1.VirtualMachineInstanceSpec) bool {
       // return true if the spec uses the removed API
       return false
   }

   func init() {
       featuregate.RegisterFeatureGate(featuregate.FeatureGate{
           Name:        MyNewFeature,
           State:       featuregate.Discontinued,
           Message:     "MyNewFeature has been removed since v1.18.0",
           VmiSpecUsed: myNewFeatureApiUsed,
       })
   }
   ```
3. Remove the `Enabled()` method — nothing should call it anymore.

See [`network/passt.go`](../../pkg/virt-config/featuregate/network/passt.go)
for a real example with `VmiSpecUsed` validation.

## Choosing the Right SIG Package

Place the gate in the SIG package that owns the feature area:

| Package | Scope |
|---------|-------|
| `compute/` | VM lifecycle, migration, CPU/memory, scheduling, virt-handler |
| `network/` | Networking, interfaces, bindings, network plugins |
| `storage/` | Disks, volumes, snapshots, backup/restore, CDI integration |
| `legacy/` | Cross-cutting or not yet assigned to a SIG |

If a gate in `legacy/` is later claimed by a SIG, move its file to the
appropriate package and update imports across the codebase.

## Tooling

- **`make feature-gate-report`** — Prints a JSON inventory of all active
  (non-GA, non-Discontinued) gates. See
  [`tools/feature-gate-report/README.md`](../../tools/feature-gate-report/README.md).
- **`validator.go`** — The `ValidateFeatureGates()` function in the core
  `featuregate` package enforces Discontinued gate rejection at admission time
  via `VmiSpecUsed` callbacks.
