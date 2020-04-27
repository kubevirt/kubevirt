#!/usr/bin/env bash

set -e

source hack/common.sh
HCO_DIR="$(readlink -f $(dirname $0)/../)"
BUILD_DIR=${HCO_DIR}/tests/build
BUILD_TAG="hco-test-build"
REGISTRY="docker.io/kubevirtci"
TAG=$(get_image_tag)
TEST_BUILD_TAG="${REGISTRY}/${BUILD_TAG}:${TAG}"


# Build the encapsulated compile and test container
(cd ${BUILD_DIR} && docker build --tag ${TEST_BUILD_TAG} .)

docker push ${TEST_BUILD_TAG}

echo "Successfully created and pushed new test utils image: ${TEST_BUILD_TAG}"

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}
