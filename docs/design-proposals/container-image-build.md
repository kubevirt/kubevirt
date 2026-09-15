# Design Proposal: Container-Native Image Build System for KubeVirt

## Summary

This proposal introduces a Podman/Docker-based container build flow as an alternative to Bazel for building KubeVirt component images. Both flows will coexist side by side, gated by the `KUBEVIRT_NO_BAZEL` environment variable, enabling a staged migration away from Bazel.

RPM dependency management uses standalone `bazeldnf` (no Bazel invocation) to generate RPM tarballs, which are then consumed by Containerfiles to build base images.

## Motivation

Bazel presents several challenges for the KubeVirt project (ref: [#14038](https://github.com/kubevirt/kubevirt/issues/14038)):

- **Knowledge barrier** — few contributors understand Bazel internals
- **Tech debt** — pinned to an old Bazel version; deprecated `rules_docker` blocks upgrades
- **s390x** — not supported by upstream Bazel; requires cross-building or custom Bazel binaries
- **Tooling incompatibility** — `BUILD.bazel` files break standard Go tooling and dependency bots (renovatebot, dependabot)
- **Divergence from Kubernetes** — Kubernetes already dropped Bazel

## Goals

1. Provide a fully functional container build flow that can produce all KubeVirt images without Bazel
2. Maintain both flows in parallel so existing CI and release processes are not disrupted
3. Optimize the container flow (caching, build times) before eliminating Bazel entirely
4. Stage-by-stage removal of Bazel dependencies
5. Use standalone `bazeldnf` CLI for RPM dependency management (no Bazel invocation)

## Non-Goals

- Immediate removal of Bazel (this is a phased approach)

## Architecture

### Two-Path Build System

```
┌─────────────────────────────────────────────────────────┐
│                    automation/test.sh                     │
│              (sets KUBEVIRT_NO_BAZEL=true/false)          │
└───────────────────────────┬─────────────────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
    ┌───────────▼───────────┐ ┌────────▼─────────────┐
    │   Container Flow      │ │     Bazel Flow        │
    │ (KUBEVIRT_NO_BAZEL=   │ │ (KUBEVIRT_NO_BAZEL=   │
    │       true)           │ │       false)          │
    └───────────┬───────────┘ └────────┬─────────────┘
                │                       │
    ┌───────────▼───────────┐ ┌────────▼─────────────┐
    │ hack/cluster-build.sh │ │ hack/cluster-build.sh │
    │  → multi-arch-        │ │  → hack/dockerized    │
    │    container.sh       │ │  → hack/multi-arch.sh │
    │  → push-images-       │ │  → bazel-push-        │
    │    container.sh       │ │    images.sh          │
    └───────────────────────┘ └──────────────────────┘
```

### RPM Base Image Layer

RPM dependencies are decoupled from component image builds into separate base images.
Base images are built using standalone `bazeldnf rpm2tar` — no Bazel invocation required:

```
┌──────────────────────────────────────────────────────────┐
│  hack/rpm-deps.sh (standalone bazeldnf)                   │
│  → Resolves RPM dependencies, updates BUILD.bazel/WORKSPACE│
└──────────────────────────┬───────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────┐
│  hack/rpm-base-images/generate-rpm-tars.sh                │
│  → Parses rpmtree rules from BUILD.bazel                  │
│  → Downloads RPMs using URLs from WORKSPACE               │
│  → Runs bazeldnf rpm2tar with --symlinks/--capabilities   │
│  → Outputs tars to _out/rpm-tars/                         │
└──────────────────────────┬───────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────┐
│  hack/rpm-base-images/build-base-images.sh                │
│  → Runs generate-rpm-tars.sh for each rpmtree target      │
│  → Builds Containerfiles (FROM scratch + ADD tar)          │
│  → Tags as quay.io/kubevirt/<name>:bazeldnf               │
├──────────────────────────────────────────────────────────┤
│  launcherbase │ handlerbase │ exportserverbase │ pr-helper │
│  libguestfs-  │ sidecar-   │ testimage        │ libvirt-  │
│  tools        │ shim       │                  │ devel     │
└──────────────┬───────────────────────────────────────────┘
               │
               ▼ (used as FROM in component Containerfiles)
┌──────────────────────────────────────────────────────────┐
│         Component Images (Containerfiles)                 │
│  virt-operator, virt-api, virt-controller,               │
│  virt-handler, virt-launcher, etc.                       │
└──────────────────────────────────────────────────────────┘
```

**When to rebuild base images:**
- When files under `rpm/` or `WORKSPACE` change (detected via `RPM_CHANGES` in CI)
- Manually via `make rpm-base-build && make rpm-base-push`

**Multi-arch support:**
- Base images are built for `amd64`, `arm64`, `s390x`
- Component images are built for the native architecture of the CI node
- Cross-arch container-disk images are built for emulation testing

## Migration Stages

### Stage 1: Standalone bazeldnf for RPM Management (Completed)

- `hack/rpm-deps.sh` and `hack/verify-rpm-deps.sh` use standalone `bazeldnf` CLI instead of `bazel run`
- `hack/install-bazeldnf.sh` downloads and caches the standalone binary
- `rpm/BUILD.bazel` and `WORKSPACE` are still generated/consumed, but no Bazel invocation needed
- RPM tars for base images are generated via `bazeldnf rpm2tar` (not `bazel build`)

### Stage 2: Dual-Flow Coexistence (Current)

- `KUBEVIRT_NO_BAZEL=true` activates the container flow
- All existing Prow lanes continue using Bazel by default
- Container flow is tested by hardcoding `KUBEVIRT_NO_BAZEL=true` in `automation/test.sh`
- Both flows produce identical image sets and pass the same E2E tests

### Stage 3: Container Flow Optimization

- Implement build caching (Go build cache persistence, layer caching)
- Reduce image build times to be competitive with Bazel's cached builds
- Validate container flow across all CI lanes (sig-compute, sig-network, sig-operator, sig-storage)
- Move `KUBEVIRT_NO_BAZEL` to Prow job configs (remove hardcode from `test.sh`)

### Stage 4: Container Flow as Default

- Flip the default: `KUBEVIRT_NO_BAZEL` defaults to `true`
- Bazel flow kept as opt-in fallback
- All release processes use container flow

### Stage 5: Bazel Removal

- Remove `KUBEVIRT_NO_BAZEL` flag and all conditional logic
- Remove Bazel build files (`BUILD.bazel`) for image targets
- Remove Bazel sandbox bootstrap (`hack/bootstrap.sh` regeneration)
- `rpm/BUILD.bazel` and `WORKSPACE` retained only as data files consumed by standalone `bazeldnf`

## Key Implementation Details

### RPM Dependency Management (`hack/rpm-deps.sh`)

RPM dependencies are managed using standalone `bazeldnf` (installed via `hack/install-bazeldnf.sh`):

```bash
source hack/install-bazeldnf.sh

bazeldnf fetch --repofile rpm/repo-cs9.yaml
bazeldnf rpmtree --name launcherbase_x86_64_cs9 --basesystem centos-stream-release ...
bazeldnf prune ...
bazeldnf verify ...
```

### Base Image Generation (`hack/rpm-base-images/generate-rpm-tars.sh`)

Converts rpmtree rules into rootfs tars without Bazel:

```bash
source hack/rpm-base-images/generate-rpm-tars.sh

# Parses BUILD.bazel for RPM names, looks up URLs in WORKSPACE,
# downloads RPMs, runs bazeldnf rpm2tar with symlinks/capabilities
generate_rpm_tar launcherbase_x86_64_cs9
```

### Flow Selection (`hack/cluster-build.sh`)

```bash
if [ "${KUBEVIRT_NO_BAZEL}" = "true" ]; then
    # Container flow: build with Podman/Docker, push directly
    hack/multi-arch-container.sh
    hack/push-images-container.sh
else
    # Bazel flow: build and push via Bazel rules
    hack/dockerized "... hack/multi-arch.sh push-images"
fi
```

### Functest Build Selection (`Makefile`)

```makefile
ifeq ($(KUBEVIRT_NO_BAZEL),true)
build-functests: go-build-functests
else
build-functests: bazel-build-functests
endif
```

### Manifest Generation (`hack/build-manifests.sh`)

```bash
if [ "${KUBEVIRT_NO_BAZEL}" != "true" ]; then
    bazel run //:build-manifest-templator -- ${templator}
else
    (cd tools/manifest-templator/ && go_build && cp manifest-templator ${templator})
fi
```

### Bootstrap Skip (`hack/bootstrap.sh`)

When `KUBEVIRT_NO_BAZEL=true`, the Bazel sandbox regeneration is skipped entirely:

```bash
if [ "${KUBEVIRT_NO_BAZEL}" != "true" ] && [ "${KUBEVIRT_SKIP_BOOTSTRAP:-}" != "true" ]; then
    kubevirt::bootstrap::regenerate ${HOST_ARCHITECTURE}
fi
```

### Alt Tag/Prefix Handling for Operator Tests

The container flow publishes images in two additional ways for sig-operator upgrade tests:
1. Same prefix with alt tag: `registry:5000/kubevirt/virt-operator:devel_alt`
2. Alt prefix with base tag: `registry:5000/kubevirt/kv-virt-operator:devel`

This matches the Bazel flow's behavior in `hack/bazel-push-images.sh`.

## Image Inventory

All images from the Bazel flow are covered by the container flow:

### Base Images (built via `hack/rpm-base-images/build-base-images.sh`)

| Image | Containerfile | Architectures |
|-------|--------------|---------------|
| launcherbase | hack/rpm-base-images/Containerfile.launcherbase | amd64, arm64, s390x |
| handlerbase | hack/rpm-base-images/Containerfile.handlerbase | amd64, arm64, s390x |
| exportserverbase | hack/rpm-base-images/Containerfile.exportserverbase | amd64, arm64, s390x |
| libvirt-devel | hack/rpm-base-images/Containerfile.libvirt-devel | amd64, arm64, s390x |
| sidecar-shim | hack/rpm-base-images/Containerfile.sidecar-shim | amd64, arm64, s390x |
| testimage | hack/rpm-base-images/Containerfile.testimage | amd64, arm64, s390x |
| pr-helper | hack/rpm-base-images/Containerfile.pr-helper | amd64, arm64 |
| libguestfs-tools | hack/rpm-base-images/Containerfile.libguestfs-tools | amd64, s390x |

### Component Images (built via `hack/build-images-container.sh`)

| Image | Containerfile |
|-------|--------------|
| virt-operator | cmd/virt-operator/Containerfile |
| virt-api | cmd/virt-api/Containerfile |
| virt-controller | cmd/virt-controller/Containerfile |
| virt-handler | cmd/virt-handler/Containerfile |
| virt-launcher | cmd/virt-launcher/Containerfile |
| virt-exportserver | cmd/virt-exportserver/Containerfile |
| virt-exportproxy | cmd/virt-exportproxy/Containerfile |
| virt-synchronization-controller | cmd/synchronization-controller/Containerfile |
| conformance | tests/conformance/Containerfile |
| sidecar-shim | cmd/sidecars/Containerfile |
| example-hook-sidecar | cmd/sidecars/smbios/Containerfile |
| example-disk-mutation-hook-sidecar | cmd/sidecars/disk-mutation/Containerfile |
| example-cloudinit-hook-sidecar | cmd/sidecars/cloudinit/Containerfile |
| example-node-hook-plugin | cmd/example-node-hook-plugin/Containerfile |
| test-domain-hook-sidecar | cmd/plugin-sidecars/test-domain-hook/Containerfile |
| test-helpers | cmd/test-helpers/pod-mutator/Containerfile |
| network-slirp-binding | cmd/sidecars/network-slirp-binding/Containerfile |
| network-passt-binding | cmd/sidecars/network-passt-binding/Containerfile |
| network-passt-binding-cni | cmd/cniplugins/passt-binding/cmd/Containerfile |
| pr-helper | cmd/pr-helper/Containerfile |
| libguestfs-tools | cmd/libguestfs/Containerfile |
| vm-killer | images/vm-killer/Containerfile |
| disks-images-provider | images/disks-images-provider/Containerfile |
| winrmcli | images/winrmcli/Containerfile |
| Container disk images | hack/build-container-disks.sh |

## What Bazel Is Still Used For

1. **Sandbox bootstrap** — creates the build sandbox with system libraries needed for CGO compilation (skipped when `KUBEVIRT_NO_BAZEL=true`)
2. **Bazel flow builds** — still the default in production CI (until Stage 4)

Note: RPM dependency resolution (`hack/rpm-deps.sh`) now uses standalone `bazeldnf` and no longer requires Bazel.

## Testing Strategy

- All existing Prow E2E test lanes (sig-compute, sig-network, sig-operator, sig-storage) run with the container flow
- Unit tests are unaffected (they use `go test` directly)
- The container flow is validated by setting `KUBEVIRT_NO_BAZEL=true` in `automation/test.sh`

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make rpm-deps` | Resolve RPM dependencies using standalone bazeldnf |
| `make rpm-base-build` | Build RPM base images (generate tars + podman build) |
| `make rpm-base-push` | Push RPM base images to registry |
| `make container-build-images` | Build all component images using Containerfiles |
| `make container-push-images` | Push all component images to registry |
| `make verify-rpm-deps` | Verify RPM dependency checksums |

## Related Links

- [Issue #14038: State of Bazel in Kubevirt / Can we remove Bazel?](https://github.com/kubevirt/kubevirt/issues/14038)
- [PR #18286: Container build flow implementation](https://github.com/kubevirt/kubevirt/pull/18286)
