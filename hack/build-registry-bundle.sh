#!/bin/bash -x

IMAGE_REGISTRY="${IMAGE_REGISTRY:-docker.io}"
REGISTRY_NAMESPACE="${REGISTRY_NAMESPACE:-}"
CONTAINER_TAG="${CONTAINER_TAG:-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 8 | head -n 1)}"
CONTAINER_BUILD_CMD="${CONTAINER_BUILD_CMD:-docker}"
REGISTRY_DOCKERFILE="${REGISTRY_DOCKERFILE:-Dockerfile}"
REGISTRY_IMAGE_NAME="${REGISTRY_IMAGE_NAME:-hco-registry}"
REGISTRY_EXTRA_BULID_ARGS="${REGISTRY_EXTRA_BUILD_ARGS:-}"

if [ -z "${REGISTRY_NAMESPACE}" ]; then
    echo "Please set REGISTRY_NAMESPACE"
    echo "   REGISTRY_NAMESPACE=rthallisey ./hack/build-registry-bundle.sh"
    echo "   make bundle-registry REGISTRY_NAMESPACE=rthallisey"
    exit 1
fi

TMP_ROOT="$(dirname "${BASH_SOURCE[@]}")/.."
REPO_ROOT=$(readlink -e "${TMP_ROOT}" 2> /dev/null || perl -MCwd -e 'print Cwd::abs_path shift' "${TMP_ROOT}")

$CONTAINER_BUILD_CMD build ${REGISTRY_EXTRA_BUILD_ARGS} --no-cache -t ${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${REGISTRY_IMAGE_NAME}:${CONTAINER_TAG} -f deploy/$REGISTRY_DOCKERFILE .
$CONTAINER_BUILD_CMD push ${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${REGISTRY_IMAGE_NAME}:${CONTAINER_TAG}

