#!/usr/bin/env bash

set -e

source hack/common.sh

HCO_DIR="$(readlink -f $(dirname $0)/../)"
WORK_DIR="/go/src/github.com/kubevirt/hyperconverged-cluster-operator"
REGISTRY=${REGISTRY:-docker.io/kubevirtci}
REPOSITORY=${REPOSITORY:-hco-test-build}
TAG=${TAG:-v20200427-db8c50b}
BUILD_TAG="${REGISTRY}/${REPOSITORY}:${TAG}"

# Execute the build
[ -t 1 ] && USE_TTY="-it"
docker run ${USE_TTY} \
    --rm \
    -v ${HCO_DIR}:${WORK_DIR}:rw,Z \
    -e RUN_UID=$(id -u) \
    -e RUN_GID=$(id -g) \
    -e GOCACHE=/gocache \
    -w ${WORK_DIR} \
    ${BUILD_TAG} "$1"
