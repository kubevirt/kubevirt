#!/usr/bin/env bash
# Default hooks for the KubeVirt Bazel build, registry push, and cluster.
# Override functions through CLUSTER_SYNC_FAST_HOOKS for another environment.

cluster_sync_fast_init() {
    # shellcheck disable=SC1091
    source hack/virt-template/default.sh
    VIRT_TEMPLATE_VERSION=${VIRT_TEMPLATE_VERSION:-$virt_template_version}
    export VIRT_TEMPLATE_VERSION
}

cluster_sync_fast_cluster_ready() {
    # shellcheck disable=SC1091
    source hack/common.sh
    # Provider is selected at runtime from KUBEVIRT_PROVIDER.
    # shellcheck disable=SC1090
    source "kubevirtci/cluster-up/cluster/${KUBEVIRT_PROVIDER}/provider.sh"
    # shellcheck disable=SC1091
    source hack/config.sh

    namespace=${CLUSTER_SYNC_FAST_NAMESPACE:-${KUBEVIRT_INSTALLED_NAMESPACE:-kubevirt}}
    kubectl_cmd=${CLUSTER_SYNC_FAST_KUBECTL:-./kubevirtci/cluster-up/kubectl.sh}
    local installed_image
    installed_image=$("$kubectl_cmd" -n "$namespace" get deployment virt-operator \
        -o=jsonpath='{.spec.template.spec.containers[?(@.name=="virt-operator")].image}' 2>/dev/null || true)
    if [[ -z "$installed_image" ]]; then
        echo "No KubeVirt installation found in namespace '$namespace'. Run 'make cluster-sync' once to bootstrap the cluster, then use 'make cluster-sync-fast' for incremental updates." >&2
        return 1
    fi
    if [[ -z "$requested_docker_tag" && "$installed_image" == *:* ]]; then
        DOCKER_TAG=${installed_image##*:}
        export DOCKER_TAG
    fi
}

# Arguments: component names, build targets, digest paths.
# Output: FAST_SYNC_DIGEST=component:digest records.
cluster_sync_fast_build() {
    local components=$1 targets=$2 digest_files=$3
    hack/dockerized "KUBEVIRT_FAST_COMPONENTS='$components' KUBEVIRT_FAST_TARGETS='$targets' \
KUBEVIRT_FAST_DIGEST_FILES='$digest_files' KUBEVIRT_PROVIDER='$KUBEVIRT_PROVIDER' ./hack/cluster-sync-fast-build.sh"
}

# Arguments: image names, image tag, alternate tag.
cluster_sync_fast_push() {
    local targets=$1 tag=$2 alt_tag=$3
    hack/dockerized "PUSH_TARGETS='$targets' BUILD_ARCH='${BUILD_ARCH:-}' DOCKER_PREFIX='${DOCKER_PREFIX:-}' \
DOCKER_TAG='$tag' DOCKER_TAG_ALT='$alt_tag' \
IMAGE_PREFIX='${IMAGE_PREFIX:-$image_prefix}' IMAGE_PREFIX_ALT='${IMAGE_PREFIX_ALT:-$image_prefix_alt}' \
KUBEVIRT_PROVIDER='$KUBEVIRT_PROVIDER' ./hack/bazel-push-images.sh"
}
