#!/usr/bin/env bash

set -e

source hack/common.sh

HCO_DIR="$(readlink -f $(dirname $0)/../)"
BUILD_DIR=${HCO_DIR}/tests/build
WORK_DIR="/go/src/github.com/kubevirt/hyperconverged-cluster-operator"
BUILD_TAG="hco-test-build"

# Build the encapsulated compile and test container
(cd ${BUILD_DIR} && docker build --tag ${BUILD_TAG} .)

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
