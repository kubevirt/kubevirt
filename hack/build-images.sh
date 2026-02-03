#!/usr/bin/env bash
#
# Build container images using native tools (Podman or Docker)
#
# This script builds KubeVirt container images from RPM lock files,
# supporting both Podman and Docker with auto-detection.
#
# Usage:
#   ./hack/build-images.sh [OPTIONS] [IMAGE_NAME...]
#
# Options:
#   --arch ARCH         Build for specific architecture (x86_64, aarch64, s390x)
#                       Default: host architecture
#   --all-arch          Build for all architectures
#   --push              Push images after building
#   --registry REG      Registry prefix (default: localhost:5000/kubevirt or DOCKER_PREFIX)
#   --tag TAG           Image tag (default: latest)
#   --dry-run           Show commands without executing
#   -h, --help          Show this help
#
# Examples:
#   ./hack/build-images.sh virt-launcher
#   ./hack/build-images.sh --arch aarch64 virt-handler
#   ./hack/build-images.sh --all-arch --push virt-launcher
#   KUBEVIRT_CRI=docker ./hack/build-images.sh virt-launcher
#

set -eo pipefail

# Timing support
START_TIME=$(date +%s.%N)

show_timing() {
    local end_time=$(date +%s.%N)
    local duration=$(echo "${end_time} - ${START_TIME}" | bc)
    echo ""
    echo "=============================================="
    echo "Total Build Time: ${duration}s"
    echo "=============================================="
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Detect container runtime (extracted from common.sh to avoid sourcing kubevirtci)
determine_cri_bin() {
    if [ "${KUBEVIRT_CRI:-}" = "podman" ]; then
        echo podman
    elif [ "${KUBEVIRT_CRI:-}" = "docker" ]; then
        echo docker
    else
        if podman ps >/dev/null 2>&1; then
            echo podman
        elif docker ps >/dev/null 2>&1; then
            echo docker
        else
            echo ""
        fi
    fi
}

KUBEVIRT_CRI="$(determine_cri_bin)"

if [ -z "${KUBEVIRT_CRI}" ]; then
    echo >&2 "no working container runtime found. Neither docker nor podman seems to work."
    exit 1
fi

# Defaults
REGISTRY="${DOCKER_PREFIX:-localhost:5000/kubevirt}"
TAG="${DOCKER_TAG:-latest}"
DRY_RUN=false
PUSH=false
BUILD_ARCHS=()
IMAGES=()

# Get host architecture in the format we use
get_host_arch() {
    local arch
    arch=$(uname -m)
    case "${arch}" in
        x86_64)  echo "x86_64" ;;
        aarch64) echo "aarch64" ;;
        s390x)   echo "s390x" ;;
        arm64)   echo "aarch64" ;;  # macOS uses arm64
        *)       echo "${arch}" ;;
    esac
}

# Convert our arch format to OCI platform format
arch_to_platform() {
    local arch=$1
    case "${arch}" in
        x86_64)  echo "linux/amd64" ;;
        aarch64) echo "linux/arm64" ;;
        s390x)   echo "linux/s390x" ;;
        *)       echo "linux/${arch}" ;;
    esac
}

# Map image name to package set
image_to_package_set() {
    local image=$1
    case "${image}" in
        virt-launcher)    echo "launcherbase" ;;
        virt-handler)     echo "handlerbase" ;;
        virt-operator)    echo "sandboxroot" ;;
        virt-api)         echo "sandboxroot" ;;
        virt-controller)  echo "sandboxroot" ;;
        virt-exportserver) echo "exportserverbase" ;;
        virt-exportproxy) echo "sandboxroot" ;;  # No RPMs, uses passwd-image base
        libguestfs-tools) echo "libguestfs-tools" ;;
        pr-helper)        echo "pr-helper" ;;
        sidecar-shim)     echo "sidecar-shim" ;;
        *)                echo "${image}" ;;
    esac
}

# Print usage
usage() {
    head -40 "$0" | grep -E "^#" | sed 's/^# *//' | tail -n +3
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --arch)
            BUILD_ARCHS+=("$2")
            shift 2
            ;;
        --all-arch)
            BUILD_ARCHS=("x86_64" "aarch64" "s390x")
            shift
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "ERROR: Unknown option: $1" >&2
            usage
            exit 1
            ;;
        *)
            IMAGES+=("$1")
            shift
            ;;
    esac
done

# Default to host architecture if none specified
if [[ ${#BUILD_ARCHS[@]} -eq 0 ]]; then
    BUILD_ARCHS=("$(get_host_arch)")
fi

# Validate we have images to build
if [[ ${#IMAGES[@]} -eq 0 ]]; then
    echo "ERROR: No images specified" >&2
    echo "" >&2
    echo "Available images:" >&2
    echo "  virt-launcher, virt-handler, virt-operator, virt-api, virt-controller" >&2
    echo "  libguestfs-tools, pr-helper, sidecar-shim" >&2
    exit 1
fi

# Run or print command
run_cmd() {
    if [[ "${DRY_RUN}" == "true" ]]; then
        echo "[DRY-RUN] $*"
    else
        echo "[RUN] $*"
        "$@"
    fi
}

# Build image for a specific architecture
build_image() {
    local image=$1
    local arch=$2
    local package_set
    local platform
    local containerfile
    local lock_file
    local full_tag

    package_set=$(image_to_package_set "${image}")
    platform=$(arch_to_platform "${arch}")
    containerfile="${SCRIPT_DIR}/../build/${image}/Containerfile"
    lock_file="${SCRIPT_DIR}/../rpm-lockfiles/${package_set}-${arch}.lock.json"
    full_tag="${REGISTRY}/${image}:${TAG}-${arch}"

    echo ""
    echo "=============================================="
    echo "Building: ${image} (${arch})"
    echo "=============================================="
    echo "Package set:    ${package_set}"
    echo "Lock file:      ${lock_file}"
    echo "Containerfile:  ${containerfile}"
    echo "Platform:       ${platform}"
    echo "Tag:            ${full_tag}"
    echo "Runtime:        ${KUBEVIRT_CRI}"
    echo ""

    # Check lock file exists
    if [[ ! -f "${lock_file}" ]]; then
        echo "ERROR: Lock file not found: ${lock_file}" >&2
        echo "Run: ./hack/rpm-freeze-native.sh ${arch} ${package_set}" >&2
        return 1
    fi

    # Check Containerfile exists
    if [[ ! -f "${containerfile}" ]]; then
        echo "WARNING: Containerfile not found: ${containerfile}" >&2
        echo "Generating from lock file..." >&2
        mkdir -p "$(dirname "${containerfile}")"
        run_cmd "${SCRIPT_DIR}/containerfile-from-lock.sh" "${lock_file}" "${containerfile}"
    fi

    # Build the image
    run_cmd ${KUBEVIRT_CRI} build \
        --platform "${platform}" \
        --tag "${full_tag}" \
        --file "${containerfile}" \
        "${SCRIPT_DIR}/.."

    echo ""
    echo "Successfully built: ${full_tag}"
}

# Push image
push_image() {
    local image=$1
    local arch=$2
    local full_tag="${REGISTRY}/${image}:${TAG}-${arch}"

    echo ""
    echo "Pushing: ${full_tag}"
    run_cmd ${KUBEVIRT_CRI} push "${full_tag}"
}

# Create and push multi-arch manifest
create_manifest() {
    local image=$1
    local manifest_tag="${REGISTRY}/${image}:${TAG}"
    local arch_tags=()

    for arch in "${BUILD_ARCHS[@]}"; do
        arch_tags+=("${REGISTRY}/${image}:${TAG}-${arch}")
    done

    echo ""
    echo "=============================================="
    echo "Creating manifest: ${manifest_tag}"
    echo "=============================================="
    echo "Architectures: ${BUILD_ARCHS[*]}"
    echo ""

    # Remove existing manifest if any
    run_cmd ${KUBEVIRT_CRI} manifest rm "${manifest_tag}" 2>/dev/null || true

    # Create manifest
    run_cmd ${KUBEVIRT_CRI} manifest create "${manifest_tag}" "${arch_tags[@]}"

    if [[ "${PUSH}" == "true" ]]; then
        echo ""
        echo "Pushing manifest: ${manifest_tag}"
        run_cmd ${KUBEVIRT_CRI} manifest push "${manifest_tag}"
    fi
}

# =============================================================================
# Main
# =============================================================================

echo "=============================================="
echo "KubeVirt Container Image Builder"
echo "=============================================="
echo "Container runtime: ${KUBEVIRT_CRI}"
echo "Registry:          ${REGISTRY}"
echo "Tag:               ${TAG}"
echo "Architectures:     ${BUILD_ARCHS[*]}"
echo "Images:            ${IMAGES[*]}"
echo "Push:              ${PUSH}"
echo "Dry run:           ${DRY_RUN}"
echo ""

# Build each image for each architecture
for image in "${IMAGES[@]}"; do
    for arch in "${BUILD_ARCHS[@]}"; do
        build_image "${image}" "${arch}"

        if [[ "${PUSH}" == "true" ]]; then
            push_image "${image}" "${arch}"
        fi
    done

    # Create multi-arch manifest if building multiple architectures
    if [[ ${#BUILD_ARCHS[@]} -gt 1 ]]; then
        create_manifest "${image}"
    fi
done

echo ""
echo "=============================================="
echo "Build complete!"
echo "=============================================="

show_timing
