#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

rm -rf "${TESTS_OUT_DIR}"
mkdir -p "${TESTS_OUT_DIR}/tools"

bazel build \
    --config=${HOST_ARCHITECTURE} \
    //vendor/github.com/onsi/ginkgo/v2/ginkgo:ginkgo

bazel run \
    --config=${HOST_ARCHITECTURE} \
    :build-ginkgo -- ${TESTS_OUT_DIR}/ginkgo
