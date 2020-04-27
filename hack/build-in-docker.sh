#!/usr/bin/env bash

set -e

main() {
  local HCO_DIR
  HCO_DIR="$(readlink -f $(dirname $0)/../)"
  local BUILD_DIR=${HCO_DIR}/tests/build
  local BUILD_TAG="hco-test-build"
  local REGISTRY="docker.io/kubevirtci"
  local TAG
  TAG="$(get_image_tag)"
  local TEST_BUILD_TAG="${REGISTRY}/${BUILD_TAG}:${TAG}"

  # Build the encapsulated compile and test container
  (cd "${BUILD_DIR}" && docker build --tag "${TEST_BUILD_TAG}" .)

  docker push "${TEST_BUILD_TAG}"

  echo "Successfully created and pushed new test utils image: ${TEST_BUILD_TAG}"
}

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

source hack/common.sh
main "$@"
