# Building KubeVirt with Podman or Docker

## Prerequisites

- Podman 4.0+ OR Docker 20.10+
- Make
- Git

No Bazel installation required on host - the builder image includes Bazel for RPM generation in base images.

## Quick Start

```bash
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
2. Uses architecture-specific distroless base image digests (pinned)
3. Pushes each arch-tagged image
4. Creates a multi-arch manifest combining all architectures
5. Pushes the manifest with the main tag (e.g., `devel`)


## Examples

```bash
# Build for specific architecture
BUILD_ARCH=arm64 make container-build-images

# Build with custom tag
DOCKER_TAG=v1.2.3 make container-build-images

# Build with custom registry
DOCKER_PREFIX=my-registry.com/kubevirt make container-build-images
```

### Multi-Architecture Testing

Test builds for multiple architectures:

```bash
export BUILD_ARCH=amd64,arm64

make cluster-sync
```

## Base Images

RPM base images (containing system libraries like libvirt, qemu, etc.) are built separately
and only need to be rebuilt when RPM dependencies change. See `hack/rpm-base-images/` for details.

```bash
# Build base images (uses Bazel internally for RPM extraction)
make rpm-base-build

# Push base images
make rpm-base-push
```
