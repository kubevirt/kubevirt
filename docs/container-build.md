# Building KubeVirt with Podman or Docker

## Prerequisites

- Podman 4.0+ OR Docker 20.10+
- Make
- Git

No Bazel installation required on host - the builder image includes Bazel for RPM generation.

## Quick Start

```bash
export KUBEVIRT_NO_BAZEL=true

make container-build-images
```

## What Gets Built

The build creates the following images:
- `virt-operator` 
- `virt-api` 
- `virt-controller` 
- `virt-handler` 
- `virt-launcher` (includes QEMU/libvirt)
- `virt-exportserver` 
- `virt-exportproxy`
- Plus many additional helper, test, and sidecar images

## Configuration

Environment variables:
- `KUBEVIRT_NO_BAZEL` - Set to `true` to use container builds instead of Bazel (required)
- `KUBEVIRT_CRI` - Container engine to use 
- `BUILD_ARCH` - Target architecture (amd64, arm64, s390x) or comma-separated list for multi-arch
- `DOCKER_TAG` - Image tag (default: devel)
- `DOCKER_PREFIX` - Registry prefix (default: quay.io/kubevirt)
- `BUILDER_IMAGE` - Builder image to use

## Multi-Architecture Builds

Build for multiple architectures:

```bash
BUILD_ARCH=amd64,arm64,s390x make container-build-images-multi-arch

BUILD_ARCH=amd64,arm64 ./hack/multi-arch-container.sh

BUILD_ARCH=amd64,arm64,s390x make container-push-images-multi-arch
```

The multi-arch workflow:
1. Builds each architecture with arch-specific tags (e.g., `devel-amd64`, `devel-arm64`)
2. Uses architecture-specific distroless base image digests (pinned, matching Bazel)
3. Pushes each arch-tagged image
4. Creates a multi-arch manifest combining all architectures
5. Pushes the manifest with the main tag (e.g., `devel`)


## Examples

```bash
# Build for specific architecture
export KUBEVIRT_NO_BAZEL=true
BUILD_ARCH=arm64 make container-build-images

# Build with custom tag
export KUBEVIRT_NO_BAZEL=true
DOCKER_TAG=v1.2.3 make container-build-images

# Build with custom registry
export KUBEVIRT_NO_BAZEL=true
DOCKER_PREFIX=my-registry.com/kubevirt make container-build-images
```

### Multi-Architecture Testing

Test builds for multiple architectures:

```bash
export KUBEVIRT_NO_BAZEL=true
export BUILD_ARCH=amd64,arm64

make cluster-sync
```

### Switching Between Bazel and Container Builds

```bash
# Use Bazel (default)
unset KUBEVIRT_NO_BAZEL
make cluster-sync

# Use Container builds
export KUBEVIRT_NO_BAZEL=true
make cluster-sync
```
